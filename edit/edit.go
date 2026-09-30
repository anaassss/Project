// Package edit rewrites usernames into new ones using what one or more
// markov.Models have learned. For each username it tries four kinds of edit:
//
//   - swap a word or number for one learned from other usernames
//     (ShadowWolf → MidnightWolf)
//   - keep the leading words and let the model write the rest
//     (ShadowWolf → ShadowHunter)
//   - let the model add a few characters to the end (ShadowWolf → ShadowWolf42)
//   - drop a number (MysticPanda99 → MysticPanda)
//
// With several models, each username is edited with the ones that know its
// words best. With Options.Unique, every edit also looks like a real but
// uncommon email-style username (knowledge.Unique). A model only writes from patterns it
// has actually seen, any new word it writes must be made of words it learned
// (markov.Model.WordLike), and every edit must pass markov.Check. A username
// gets fewer edits when the models know less that fits it. Run edits
// millions of usernames using every CPU core.
package edit

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/anaassss/Project/knowledge"
	"github.com/anaassss/Project/markov"
)

const (
	// vocabWords and vocabNumbers cap how many of the most common learned
	// words and numbers are swapped in.
	vocabWords   = 3000
	vocabNumbers = 100
	// maxGrowth caps how many characters longer than the original an edit
	// written by the model may be, so it can't chain fragments of learned
	// names into run-ons like "ShadowWolfPackOverflow".
	maxGrowth = 4
	// attemptsPerEdit bounds the work per username when few edits exist.
	attemptsPerEdit = 4
	// allSamples is how many rewrites and extensions each model tries per
	// username when Max is 0; swaps are then tried exhaustively.
	allSamples = 200
	// batchSize is how many usernames a worker edits at a time.
	batchSize = 1024
)

// Options controls editing.
type Options struct {
	// Max is the most edits per username. 0 means every edit the knowledge
	// allows: every learned word or number swapped into every part, plus
	// whatever the models write in allSamples tries each.
	Max        int
	AllowKnown bool // allow edits that are usernames a model learned from

	// MinLen and MaxLen bound every edit's length in characters. Zero means
	// markov.MinNameLen and markov.MaxNameLen, the garbage filter's limits.
	MinLen, MaxLen int

	// Unique makes every edit look like a real email-style username that
	// isn't taken already: lowercase, with no decorations like its, lil,
	// xX…Xx, playz or .io (taken off each username before it is edited),
	// and not plain like johnkevin (see knowledge.Unique).
	Unique bool
}

// Editor edits usernames with one or more models. It is safe for concurrent
// use.
type Editor struct {
	sources []source
	opts    Options
}

// source is one model and the vocabulary swapped in from it.
type source struct {
	m *markov.Model
	// Learned words, lowercased, most common first, each split by length:
	// [0] holds initials (1-2 letters), [1] longer words, so a word is only
	// swapped for one like it.
	words   [2][]string
	leads   [2][]string // those that can start a name: dark, john
	tails   [2][]string // those that can come later: wolf, smith, jb
	numbers []string    // learned numbers, most common first

	// byShape groups numbers by numberShape, so a number is swapped for one
	// like it: a year for a year, 4137 for another random 4-digit number.
	byShape map[int][]string
}

// numberShape is a number's length, and whether it looks like a year.
func numberShape(n string) int {
	year := len(n) == 4 && (strings.HasPrefix(n, "19") || strings.HasPrefix(n, "20"))
	if year {
		return -4
	}
	return len(n)
}

// minShapeNumbers is how many learned numbers of a shape are needed to
// swap from; with fewer, a random number of that shape is made instead.
const minShapeNumbers = 5

// numberLike returns a number shaped like num, from src's learned numbers
// when it knows enough of them.
func (src source) numberLike(rng *rand.Rand, num string) string {
	shape := numberShape(num)
	if same := src.byShape[shape]; len(same) >= minShapeNumbers {
		return same[rng.IntN(len(same))]
	}
	if shape < 0 {
		return strconv.Itoa(1975 + rng.IntN(38))
	}
	b := make([]byte, len(num))
	for {
		for i := range b {
			b[i] = byte('0' + rng.IntN(10))
		}
		if num[0] != '0' && b[0] == '0' {
			b[0] = byte('1' + rng.IntN(9))
		}
		if numberShape(string(b)) == shape { // not a year by accident
			return string(b)
		}
	}
}

func newSource(m *markov.Model) source {
	src := source{m: m, numbers: m.TopNumbers(vocabNumbers), byShape: map[int][]string{}}
	for _, n := range src.numbers {
		src.byShape[numberShape(n)] = append(src.byShape[numberShape(n)], n)
	}
	for _, w := range m.TopWords(vocabWords) {
		if markov.IsAffix(w) {
			continue // affixes frame names; they aren't swapped in
		}
		c := class(w)
		src.words[c] = append(src.words[c], w)
		lead, follow := m.Positions(w)
		if lead {
			src.leads[c] = append(src.leads[c], w)
		}
		if follow {
			src.tails[c] = append(src.tails[c], w)
		}
	}
	return src
}

// class is 0 for initials (words of 1-2 letters) and 1 for longer words.
func class(w string) int {
	if utf8.RuneCountInString(w) <= 2 {
		return 0
	}
	return 1
}

// New prepares an Editor from the models that have learned something. It
// returns an error if none have, or if opts is invalid.
func New(models []*markov.Model, opts Options) (*Editor, error) {
	var sources []source
	for _, m := range models {
		if m != nil && m.Stats().Names > 0 {
			sources = append(sources, newSource(m))
		}
	}
	if len(sources) == 0 {
		return nil, errors.New("nothing has been learned yet")
	}
	if opts.Max < 0 {
		return nil, errors.New("edits per username must be 0 (every edit) or more")
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
	return &Editor{sources: sources, opts: opts}, nil
}

// fitting returns the sources that know the most of the words in segs, or
// all of them if none knows any.
func (e *Editor) fitting(segs []markov.Segment) []source {
	if len(e.sources) == 1 {
		return e.sources
	}
	scores := make([]int, len(e.sources))
	best := 0
	for i, src := range e.sources {
		for _, seg := range segs {
			if seg.Kind == markov.Word && utf8.RuneCountInString(seg.Text) >= 2 && src.m.KnowsWord(seg.Text) {
				scores[i]++
			}
		}
		best = max(best, scores[i])
	}
	if best == 0 {
		return e.sources
	}
	var out []source
	for i, src := range e.sources {
		if scores[i] == best {
			out = append(out, src)
		}
	}
	return out
}

// splitJoined splits lowercase words that run a learned word together with
// more ("darkwolf" → "dark" + "wolf", "lucasbrenn" → "lucas" + "brenn"), so
// each part can be swapped on its own and edits keep the name's joined
// shape. The first part must be a learned word that starts names. The rest
// must be a learned word that comes later in names, with the whole 7+
// letters, or be 4+ letters after a first part of 4+. That keeps names the
// knowledge doesn't know, like "warren" or "egemen", from being cut into
// fragments ("ege" + "men"). A leading affix comes off before a learned
// word of 4+ letters ("itsmike" → "its" + "mike"), so the name can be
// swapped and the affix kept. Words learned whole, affixes, and non-ASCII
// words are left alone.
func (e *Editor) splitJoined(segs []markov.Segment) []markov.Segment {
	knows := func(w string) bool {
		return slices.ContainsFunc(e.sources, func(s source) bool { return s.m.HasWord(w) })
	}
	// can reports whether a source knows w in the first place (lead) or a
	// later one.
	can := func(w string, lead bool) bool {
		return slices.ContainsFunc(e.sources, func(s source) bool {
			l, f := s.m.Positions(w)
			return s.m.HasWord(w) && (lead && l || !lead && f)
		})
	}
	var out []markov.Segment
	for _, seg := range segs {
		w := seg.Text
		if seg.Kind != markov.Word || len(w) < 6 || w != strings.ToLower(w) || !isASCII(w) || knows(w) {
			out = append(out, seg)
			continue
		}
		cut := 0
		for _, p := range leadingAffixes {
			if rest, ok := strings.CutPrefix(w, p); ok && len(rest) >= 4 && knows(rest) {
				cut = len(p)
				break
			}
		}
		for n := len(w) - 3; n >= 3 && cut == 0; n-- {
			p, rest := w[:n], w[n:]
			if markov.IsAffix(p) || !can(p, true) {
				continue
			}
			if len(w) >= 7 && (max(len(p), len(rest)) >= 4 && can(rest, false) || len(p) >= 4 && len(rest) >= 4) {
				cut = n
				break
			}
		}
		if cut == 0 {
			out = append(out, seg)
			continue
		}
		out = append(out, markov.Segment{Text: w[:cut], Kind: markov.Word}, markov.Segment{Text: w[cut:], Kind: markov.Word})
	}
	return out
}

// leadingAffixes are the affixes that start joined names, longest first
// where one starts another ("mrs" before "mr").
var leadingAffixes = []string{"official", "real", "just", "miss", "mrs", "its", "the", "iam", "hey", "not", "lil", "get", "mr", "ms", "im", "my"}

// undecorate takes decorations (knowledge.Decoration, knowledge.TrimJoined)
// and x wrappers off segs, lowercases them, and tidies the separators left
// behind: its.Mike_99 → mike_99, xXShadowXx → shadow, noahplayz → noah.
func undecorate(segs []markov.Segment) []markov.Segment {
	isX := func(seg markov.Segment) bool {
		return seg.Kind == markov.Separator || seg.Kind == markov.Word && strings.EqualFold(seg.Text, "x")
	}
	start, end := 0, len(segs)
	for start < end && isX(segs[start]) {
		start++
	}
	for end > start && isX(segs[end-1]) {
		end--
	}
	var out []markov.Segment
	for _, seg := range segs[start:end] {
		switch seg.Kind {
		case markov.Word:
			if seg.Text = knowledge.TrimJoined(seg.Text); knowledge.Decoration(seg.Text) {
				continue
			}
		case markov.Separator:
			if len(out) == 0 || out[len(out)-1].Kind == markov.Separator {
				continue
			}
		}
		out = append(out, seg)
	}
	if len(out) > 0 && out[len(out)-1].Kind == markov.Separator {
		out = out[:len(out)-1]
	}
	return out
}

func isASCII(s string) bool {
	for i := range len(s) {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// hashSet holds the fingerprints of one username's edits: a slice while
// small, a map once there are many (when Max is 0).
type hashSet struct {
	list []uint64
	m    map[uint64]struct{}
}

func (s *hashSet) has(h uint64) bool {
	if s.m != nil {
		_, ok := s.m[h]
		return ok
	}
	return slices.Contains(s.list, h)
}

func (s *hashSet) add(h uint64) {
	if s.m != nil {
		s.m[h] = struct{}{}
		return
	}
	if s.list = append(s.list, h); len(s.list) > 32 {
		s.m = make(map[uint64]struct{}, 2*len(s.list))
		for _, x := range s.list {
			s.m[x] = struct{}{}
		}
	}
}

// Edit appends distinct edits of name, each MinLen-MaxLen characters long,
// to dst and returns it: up to Max of them, or every one the knowledge
// allows if Max is 0.
func (e *Editor) Edit(rng *rand.Rand, name string, dst []string) []string {
	start := len(dst)
	// FoldHash identifies a candidate for every duplicate check; Check, the
	// costliest test, runs last.
	var seen hashSet
	seen.add(markov.FoldHash(name))
	add := func(cand string) {
		if n := utf8.RuneCountInString(cand); n < e.opts.MinLen || n > e.opts.MaxLen {
			return
		}
		h := markov.FoldHash(cand)
		if seen.has(h) || (!e.opts.AllowKnown && slices.ContainsFunc(e.sources, func(s source) bool { return s.m.KnowsHash(h) })) {
			return
		}
		if e.opts.Unique {
			// Before lowercasing, which would hide the TV in NovaTV.
			if !knowledge.Unique(cand) {
				return
			}
			cand = strings.ToLower(cand)
		}
		if markov.Check(cand) != nil {
			return
		}
		seen.add(h)
		dst = append(dst, cand)
	}

	segs := e.splitJoined(markov.Split(name))
	if e.opts.Unique {
		// Edit the name without its decorations, which is itself an edit.
		segs = undecorate(segs)
		var b strings.Builder
		for _, seg := range segs {
			b.WriteString(seg.Text)
		}
		if b.String() != name {
			name = b.String()
			add(name)
		}
		if name == "" {
			return dst
		}
	}
	// Swappable parts are numbers and learned words of 2+ letters other than
	// affixes (xX, The, its). Affixes are kept, and so are words no model
	// learned, like nicknames (jusgo in jusgo2004), which are what make a
	// name its own. The first word is the lead: it is swapped for words that
	// usually start names, later ones for words that usually come later, so
	// john.smith doesn't become john.priya.
	var swappable []int
	styles := make([]caseStyle, len(segs))
	lead := -1
	parts := 0
	for i, seg := range segs {
		switch {
		case seg.Kind == markov.Number:
			parts++
			swappable = append(swappable, i)
		case seg.Kind == markov.Word:
			parts++
			if utf8.RuneCountInString(seg.Text) >= 2 && !markov.IsAffix(seg.Text) {
				if lead < 0 {
					lead = i
				}
				if slices.ContainsFunc(e.sources, func(s source) bool { return s.m.HasWord(seg.Text) }) {
					swappable = append(swappable, i)
					styles[i] = styleOf(seg.Text)
				}
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

	// Words a model writes must be words it learned; the name's own words
	// are kept as they are.
	var own []string
	for _, seg := range segs {
		if seg.Kind == markov.Word {
			own = append(own, seg.Text)
		}
	}
	write := func(src source, prefix string) {
		cand, ok := src.m.Complete(rng, prefix, maxLen)
		if !ok {
			return
		}
		// Prefixes end at a word boundary, so if only digits and separators
		// were written, every word is one of the name's own.
		if strings.IndexFunc(cand[len(prefix):], unicode.IsLetter) >= 0 {
			for w := range markov.Words(cand) {
				if !slices.ContainsFunc(own, func(o string) bool { return strings.EqualFold(o, w) }) && !src.m.WordLike(w) {
					return
				}
			}
		}
		add(cand)
	}
	vocabFor := func(src source, i int) []string {
		switch {
		case segs[i].Kind == markov.Number:
			if same := src.byShape[numberShape(segs[i].Text)]; len(same) >= minShapeNumbers {
				return same
			}
			return src.numbers
		}
		c := class(segs[i].Text)
		switch {
		case i == lead && len(src.leads[c]) > 0:
			return src.leads[c]
		case i != lead && len(src.tails[c]) > 0:
			return src.tails[c]
		}
		return src.words[c]
	}

	sources := e.fitting(segs)
	if e.opts.Max == 0 {
		for _, src := range sources {
			for _, i := range swappable {
				for _, v := range vocabFor(src, i) {
					if !strings.EqualFold(v, segs[i].Text) {
						add(join(segs, i, v, styles[i]))
					}
				}
			}
			for range allSamples {
				if len(cuts) > 0 {
					write(src, string(runes[:cuts[rng.IntN(len(cuts))]]))
				}
				write(src, name)
			}
		}
		return dst
	}

	tries := e.opts.Max * attemptsPerEdit * len(sources)
	for try := 0; len(dst)-start < e.opts.Max && try < tries; try++ {
		src := sources[try%len(sources)]
		switch try / len(sources) % 4 {
		case 0, 2: // swap
			if len(swappable) == 0 {
				continue
			}
			i := swappable[rng.IntN(len(swappable))]
			if segs[i].Kind == markov.Number {
				if v := src.numberLike(rng, segs[i].Text); v != segs[i].Text {
					add(join(segs, i, v, styles[i]))
				}
			} else if vocab := vocabFor(src, i); len(vocab) > 0 {
				if v := vocab[rng.IntN(len(vocab))]; !strings.EqualFold(v, segs[i].Text) {
					add(join(segs, i, v, styles[i]))
				}
			}
		case 1: // rewrite the ending
			if len(cuts) > 0 {
				write(src, string(runes[:cuts[rng.IntN(len(cuts))]]))
			}
		case 3: // extend
			write(src, name)
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
