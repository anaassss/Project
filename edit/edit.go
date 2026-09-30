// Package edit rewrites usernames into new ones using what a markov.Model has
// learned. For each username it tries four kinds of edit:
//
//   - swap a word or number for one learned from other usernames
//     (ShadowWolf → MidnightWolf)
//   - keep the leading words and let the model write the rest
//     (ShadowWolf → ShadowHunter)
//   - let the model add a few characters to the end (ShadowWolf → ShadowWolf42)
//   - drop a number (MysticPanda99 → MysticPanda)
//
// The model only writes from patterns it has actually seen, any new word it
// writes must be made of words it learned (markov.Model.WordLike), and every
// edit must pass markov.Check. A username gets fewer edits when the model knows
// less that fits it. Run edits millions of usernames using every CPU core.
package edit

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"runtime"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/anaassss/Project/markov"
)

const (
	// vocabWords and vocabNumbers cap how many of the most common learned
	// words and numbers are swapped in.
	vocabWords   = 1000
	vocabNumbers = 100
	// maxGrowth caps how many characters longer than the original an edit
	// written by the model may be, so it can't chain fragments of learned
	// names into run-ons like "ShadowWolfPackOverflow".
	maxGrowth = 4
	// attemptsPerEdit bounds the work per username when few edits exist.
	attemptsPerEdit = 4
	// batchSize is how many usernames a worker edits at a time.
	batchSize = 1024
)

// Options controls editing.
type Options struct {
	Max        int  // most edits per username
	AllowKnown bool // allow edits that are usernames the model learned from

	// MinLen and MaxLen bound every edit's length in characters. Zero means
	// markov.MinNameLen and markov.MaxNameLen, the garbage filter's limits.
	MinLen, MaxLen int
}

// Editor edits usernames with one model. It is safe for concurrent use.
type Editor struct {
	m       *markov.Model
	opts    Options
	words   []string // learned words, lowercased, most common first
	numbers []string // learned numbers, most common first
}

// New prepares an Editor. It returns an error if the model has learned
// nothing or opts.Max is not positive.
func New(m *markov.Model, opts Options) (*Editor, error) {
	if m.Stats().Names == 0 {
		return nil, errors.New("model has not learned any usernames yet")
	}
	if opts.Max < 1 {
		return nil, errors.New("edits per username must be at least 1")
	}
	if opts.MinLen == 0 {
		opts.MinLen = markov.MinNameLen
	}
	if opts.MaxLen == 0 {
		opts.MaxLen = markov.MaxNameLen
	}
	if opts.MinLen < markov.MinNameLen || opts.MaxLen > markov.MaxNameLen || opts.MinLen > opts.MaxLen {
		return nil, fmt.Errorf("lengths must be %d-%d with the shortest no longer than the longest, got %d-%d",
			markov.MinNameLen, markov.MaxNameLen, opts.MinLen, opts.MaxLen)
	}
	return &Editor{
		m:       m,
		opts:    opts,
		words:   m.TopWords(vocabWords),
		numbers: m.TopNumbers(vocabNumbers),
	}, nil
}

// Edit appends up to Max distinct edits of name, each MinLen-MaxLen
// characters long, to dst and returns it.
func (e *Editor) Edit(rng *rand.Rand, name string, dst []string) []string {
	start := len(dst)
	// FoldHash identifies a candidate for every duplicate check; Check, the
	// costliest test, runs last.
	hashes := make([]uint64, 1, e.opts.Max+1)
	hashes[0] = markov.FoldHash(name)
	add := func(cand string) {
		if n := utf8.RuneCountInString(cand); n < e.opts.MinLen || n > e.opts.MaxLen {
			return
		}
		h := markov.FoldHash(cand)
		if slices.Contains(hashes, h) || (!e.opts.AllowKnown && e.m.KnowsHash(h)) || markov.Check(cand) != nil {
			return
		}
		hashes = append(hashes, h)
		dst = append(dst, cand)
	}

	segs := markov.Split(name)
	var swappable []int // words of 2+ letters, and numbers
	styles := make([]caseStyle, len(segs))
	parts := 0
	for i, seg := range segs {
		switch {
		case seg.Kind == markov.Number:
			parts++
			swappable = append(swappable, i)
		case seg.Kind == markov.Word:
			parts++
			if utf8.RuneCountInString(seg.Text) >= 2 {
				swappable = append(swappable, i)
				styles[i] = styleOf(seg.Text)
			}
		}
	}
	// Swapping or dropping the only part would replace the whole name.
	if parts < 2 {
		swappable = nil
	}
	for _, i := range swappable {
		if segs[i].Kind == markov.Number {
			add(drop(segs, i))
		}
	}

	// Rewrite only from whole-segment boundaries: cutting inside a word
	// leaves fragments the model can't finish sensibly ("BlueSk" → "BlueSker").
	runes := []rune(name)
	// The model may write past maxGrowth only as far as MinLen requires.
	maxLen := min(max(len(runes)+maxGrowth, e.opts.MinLen), e.opts.MaxLen)
	var cuts []int
	kept := 0
	for _, seg := range segs[:max(len(segs)-1, 0)] {
		kept += utf8.RuneCountInString(seg.Text)
		if kept >= max(3, (len(runes)+1)/2) {
			cuts = append(cuts, kept)
		}
	}

	// Words the model writes must be words it learned; the name's own
	// words are kept as they are.
	own := map[string]bool{}
	for _, seg := range segs {
		if seg.Kind == markov.Word {
			own[strings.ToLower(seg.Text)] = true
		}
	}
	addWritten := func(cand string) {
		for _, seg := range markov.Split(cand) {
			if seg.Kind == markov.Word && !own[strings.ToLower(seg.Text)] && !e.m.WordLike(seg.Text) {
				return
			}
		}
		add(cand)
	}

	for try := 0; len(dst)-start < e.opts.Max && try < e.opts.Max*attemptsPerEdit; try++ {
		switch try % 4 {
		case 0, 2: // swap
			if len(swappable) == 0 {
				continue
			}
			i := swappable[rng.IntN(len(swappable))]
			vocab := e.words
			if segs[i].Kind == markov.Number {
				vocab = e.numbers
			}
			if len(vocab) > 0 {
				if v := vocab[rng.IntN(len(vocab))]; !strings.EqualFold(v, segs[i].Text) {
					add(join(segs, i, v, styles[i]))
				}
			}
		case 1: // rewrite the ending
			if len(cuts) == 0 {
				continue
			}
			cut := cuts[rng.IntN(len(cuts))]
			if cand, ok := e.m.Complete(rng, string(runes[:cut]), maxLen); ok {
				addWritten(cand)
			}
		case 3: // extend
			if cand, ok := e.m.Complete(rng, name, maxLen); ok {
				addWritten(cand)
			}
		}
	}
	return dst
}

// Run edits every username in in (see markov.ScanNames) and writes the edits
// to out, one per line, using all CPU cores. An edit is written only if its
// markov.FoldHash is not already in seen, and is then added to it; start seen
// with the input names' hashes to keep edits distinct from them. Output
// follows input order and, for a given seed, is the same on every run. Run
// returns the number of edits written.
func (e *Editor) Run(in io.Reader, out io.Writer, seen map[uint64]struct{}, seed uint64) (int, error) {
	type job struct {
		seq   uint64
		names []string
		edits []string
		done  chan struct{}
	}
	workers := runtime.GOMAXPROCS(0)
	work := make(chan *job, workers)
	ordered := make(chan *job, 2*workers)

	for range workers {
		go func() {
			for j := range work {
				rng := rand.New(rand.NewPCG(seed, j.seq))
				for _, name := range j.names {
					j.edits = e.Edit(rng, name, j.edits)
				}
				close(j.done)
			}
		}()
	}

	var scanErr error
	go func() {
		defer close(work)
		defer close(ordered)
		var seq uint64
		names := make([]string, 0, batchSize)
		send := func() {
			j := &job{seq: seq, names: names, done: make(chan struct{})}
			ordered <- j
			work <- j
			seq++
			names = make([]string, 0, batchSize)
		}
		scanErr = markov.ScanNames(in, func(name string) {
			if names = append(names, name); len(names) == batchSize {
				send()
			}
		})
		if len(names) > 0 {
			send()
		}
	}()

	w := bufio.NewWriterSize(out, 1<<20)
	written := 0
	var writeErr error
	for j := range ordered {
		<-j.done
		for _, ed := range j.edits {
			h := markov.FoldHash(ed)
			if _, dup := seen[h]; dup || writeErr != nil {
				continue
			}
			seen[h] = struct{}{}
			w.WriteString(ed)
			if writeErr = w.WriteByte('\n'); writeErr == nil {
				written++
			}
		}
	}
	if err := w.Flush(); writeErr == nil {
		writeErr = err
	}
	if scanErr != nil {
		return written, scanErr
	}
	return written, writeErr
}

// join rebuilds segs with segs[i] replaced by w styled as style, in one
// allocation.
func join(segs []markov.Segment, i int, w string, style caseStyle) string {
	size := len(w) + utf8.UTFMax
	for _, seg := range segs {
		size += len(seg.Text)
	}
	var b strings.Builder
	b.Grow(size)
	for j, seg := range segs {
		if j != i {
			b.WriteString(seg.Text)
			continue
		}
		switch style {
		case lowerCase:
			b.WriteString(w)
		case upperCase:
			for _, r := range w {
				b.WriteRune(unicode.ToUpper(r))
			}
		case titleCase:
			r, size := utf8.DecodeRuneInString(w)
			b.WriteRune(unicode.ToUpper(r))
			b.WriteString(w[size:])
		}
	}
	return b.String()
}

// drop removes segs[i] and a separator next to it, so "Dark_Wolf_7" loses
// "Wolf_" rather than leaving "Dark__7".
func drop(segs []markov.Segment, i int) string {
	sep := -1
	switch {
	case i+1 < len(segs) && segs[i+1].Kind == markov.Separator:
		sep = i + 1
	case i > 0 && segs[i-1].Kind == markov.Separator:
		sep = i - 1
	}
	var b strings.Builder
	for j, seg := range segs {
		if j != i && j != sep {
			b.WriteString(seg.Text)
		}
	}
	return b.String()
}

// caseStyle is how a word is capitalised, so a swapped-in word (learned in
// lowercase) can match the word it replaces.
type caseStyle int

const (
	lowerCase caseStyle = iota // "wolf"; also numbers, which have no case
	upperCase                  // "WOLF"
	titleCase                  // "Wolf", and anything else
)

func styleOf(word string) caseStyle {
	hasUpper, hasLower := false, false
	for _, r := range word {
		hasUpper = hasUpper || unicode.IsUpper(r)
		hasLower = hasLower || unicode.IsLower(r)
	}
	switch {
	case !hasUpper:
		return lowerCase
	case !hasLower:
		return upperCase
	}
	return titleCase
}
