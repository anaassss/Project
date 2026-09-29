// Package edit rewrites usernames into new ones using what a markov.Model has
// learned. For each username it tries four kinds of edit:
//
//   - swap a word or number for one learned from other usernames
//     (ShadowWolf → MidnightWolf)
//   - keep the leading words and let the model write the rest
//     (ShadowWolf → ShadowHunter)
//   - let the model add to the end (ShadowWolf → ShadowWolf42)
//   - drop a number (MysticPanda99 → MysticPanda)
//
// The model then judges every candidate. Only candidates that pass
// markov.Check and score at least as well as a typical learned username would
// if the model had never seen it are kept, so how many edits a username gets
// depends on how much the model knows that fits it. More training data raises
// that bar.
package edit

import (
	"errors"
	"math/rand/v2"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/anaassss/Project/markov"
)

const (
	// vocabSize caps how many of the most common learned words (and numbers)
	// are tried in place of each part of a username.
	vocabSize = 300
	// completions is how many endings the model writes per kept prefix.
	completions = 6
	// maxGrowth caps how many characters longer than the original an edit
	// written by the model may be, so it can't chain fragments of learned
	// names into run-ons like "ShadowWolfPackOverflow".
	maxGrowth = 4
	// floorPercentile sets the quality bar: an edit must score at least as
	// well as this fraction of learned usernames would if they were new.
	floorPercentile = 0.5
)

// Options controls editing.
type Options struct {
	Max        int  // most edits returned per username
	AllowKnown bool // allow edits that are usernames the model learned from
}

// Editor edits usernames with one model. Create it with New.
type Editor struct {
	m       *markov.Model
	rng     *rand.Rand
	opts    Options
	words   []string // learned words, lowercased, most common first
	numbers []string // learned numbers, most common first
	floor   float64  // lowest markov.Model.Score an edit may have
}

// New prepares an Editor. It returns an error if the model has learned
// nothing or opts.Max is not positive.
func New(m *markov.Model, rng *rand.Rand, opts Options) (*Editor, error) {
	names := m.Names()
	if len(names) == 0 {
		return nil, errors.New("model has not learned any usernames yet")
	}
	if opts.Max < 1 {
		return nil, errors.New("edits per username must be at least 1")
	}

	wordCounts, numberCounts := map[string]int{}, map[string]int{}
	for _, name := range names {
		for _, seg := range split(name) {
			switch {
			case seg.kind == word && utf8.RuneCountInString(seg.text) >= 2:
				wordCounts[strings.ToLower(seg.text)]++
			case seg.kind == number:
				numberCounts[seg.text]++
			}
		}
	}
	scores := m.HeldOutScores()
	sort.Float64s(scores)

	return &Editor{
		m:       m,
		rng:     rng,
		opts:    opts,
		words:   mostCommon(wordCounts, vocabSize),
		numbers: mostCommon(numberCounts, vocabSize),
		floor:   scores[int(float64(len(scores)-1)*floorPercentile)],
	}, nil
}

// strategy identifies which kind of edit produced a candidate.
type strategy int

const (
	swapPart strategy = iota
	rewriteEnding
	extend
	dropNumber
	numStrategies
)

type candidate struct {
	name  string
	score float64
}

// Edit returns up to Max edits of name, alternating between kinds of edit
// and best-scoring first within each kind. It returns none when nothing the
// model knows fits the name well enough.
func (e *Editor) Edit(name string) []string {
	var pools [numStrategies][]candidate
	tried := map[string]bool{strings.ToLower(name): true}
	consider := func(s strategy, cand string) {
		key := strings.ToLower(cand)
		if tried[key] {
			return
		}
		tried[key] = true
		if markov.Check(cand) != nil || (!e.opts.AllowKnown && e.m.Knows(cand)) {
			return
		}
		if score := e.m.Score(cand); score >= e.floor {
			pools[s] = append(pools[s], candidate{cand, score})
		}
	}

	segs := split(name)
	var parts []int // indexes of the words and numbers in segs
	for i, seg := range segs {
		if seg.kind == word || seg.kind == number {
			parts = append(parts, i)
		}
	}

	// Swapping or dropping the only part would replace the whole name.
	if len(parts) >= 2 {
		for _, i := range parts {
			vocab, old := e.numbers, segs[i].text
			if segs[i].kind == word {
				if utf8.RuneCountInString(old) < 2 {
					continue
				}
				vocab = e.words
			} else {
				consider(dropNumber, drop(segs, i))
			}
			for _, v := range vocab {
				if !strings.EqualFold(v, old) {
					consider(swapPart, join(segs, i, matchCase(v, old)))
				}
			}
		}
	}

	// Rewrite from whole-segment boundaries only: cutting inside a word
	// leaves fragments the model can't finish sensibly ("BlueSk" → "BlueSker").
	runes := []rune(name)
	maxLen := min(len(runes)+maxGrowth, markov.MaxNameLen)
	kept := 0
	for _, seg := range segs[:len(segs)-1] {
		kept += utf8.RuneCountInString(seg.text)
		if kept < max(3, (len(runes)+1)/2) {
			continue
		}
		for range completions {
			if cand, ok := e.m.Complete(e.rng, string(runes[:kept]), maxLen); ok {
				consider(rewriteEnding, cand)
			}
		}
	}
	for range 2 * completions {
		if cand, ok := e.m.Complete(e.rng, name, maxLen); ok {
			consider(extend, cand)
		}
	}

	for _, pool := range pools {
		sort.SliceStable(pool, func(a, b int) bool { return pool[a].score > pool[b].score })
	}
	var out []string
	for round := 0; len(out) < e.opts.Max; round++ {
		added := false
		for _, pool := range pools {
			if round < len(pool) && len(out) < e.opts.Max {
				out = append(out, pool[round].name)
				added = true
			}
		}
		if !added {
			break
		}
	}
	return out
}

type kind int

const (
	word kind = iota
	number
	separator
	other
)

type segment struct {
	text string
	kind kind
}

func kindOf(r rune) kind {
	switch {
	case unicode.IsLetter(r):
		return word
	case unicode.IsDigit(r):
		return number
	case r == '_' || r == '-' || r == '.':
		return separator
	}
	return other
}

// split breaks a username into words, numbers and separators, splitting
// words at case changes: "xXShadowWolf_99" → x, X, Shadow, Wolf, _, 99.
func split(name string) []segment {
	runes := []rune(name)
	var segs []segment
	start := 0
	for i := 1; i <= len(runes); i++ {
		if i == len(runes) || boundary(runes, i) {
			segs = append(segs, segment{string(runes[start:i]), kindOf(runes[start])})
			start = i
		}
	}
	return segs
}

// boundary reports whether a new segment starts at runes[i].
func boundary(runes []rune, i int) bool {
	prev, cur := runes[i-1], runes[i]
	if kindOf(prev) != kindOf(cur) {
		return true
	}
	if kindOf(cur) != word {
		return false
	}
	if unicode.IsLower(prev) && unicode.IsUpper(cur) {
		return true // camelCase
	}
	// An acronym followed by a word: "XMLParser" splits before "P".
	return unicode.IsUpper(prev) && unicode.IsUpper(cur) && i+1 < len(runes) && unicode.IsLower(runes[i+1])
}

func join(segs []segment, i int, replacement string) string {
	var b strings.Builder
	for j, seg := range segs {
		if j == i {
			b.WriteString(replacement)
		} else {
			b.WriteString(seg.text)
		}
	}
	return b.String()
}

// drop removes segs[i] and a separator next to it, so "Dark_Wolf_7" loses
// "Wolf_" rather than leaving "Dark__7".
func drop(segs []segment, i int) string {
	skip := map[int]bool{i: true}
	switch {
	case i+1 < len(segs) && segs[i+1].kind == separator:
		skip[i+1] = true
	case i > 0 && segs[i-1].kind == separator:
		skip[i-1] = true
	}
	var b strings.Builder
	for j, seg := range segs {
		if !skip[j] {
			b.WriteString(seg.text)
		}
	}
	return b.String()
}

// matchCase styles the lowercase word w like the word it replaces.
func matchCase(w, like string) string {
	switch {
	case like == strings.ToLower(like):
		return w
	case like == strings.ToUpper(like) && utf8.RuneCountInString(like) > 1:
		return strings.ToUpper(w)
	}
	r, size := utf8.DecodeRuneInString(w)
	return string(unicode.ToUpper(r)) + w[size:]
}

// mostCommon returns up to n keys of counts, most frequent first.
func mostCommon(counts map[string]int, n int) []string {
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] != counts[keys[j]] {
			return counts[keys[i]] > counts[keys[j]]
		}
		return keys[i] < keys[j]
	})
	if len(keys) > n {
		keys = keys[:n]
	}
	return keys
}
