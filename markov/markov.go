// Package markov implements a character-level Markov model that learns the
// shape of usernames (which characters tend to follow which, and which words
// and numbers they are made of) and generates new usernames from it. Models
// are saved to and loaded from disk so learning accumulates across runs, and
// training scales to millions of usernames by using every CPU core.
package markov

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
	"unicode/utf8"
)

const (
	// MaxOrder bounds how many previous characters a model can condition on.
	MaxOrder = 8

	// startRune and endRune pad each name so the model learns how names
	// begin and end. Both are control characters, which Check rejects, so
	// they can never collide with a real username character.
	startRune = '\x02'
	endRune   = '\x03'

	batchSize = 4096 // usernames handed to a worker at a time
)

// Model is a trained username model. Its methods may be called from many
// goroutines at once, except Learn and LearnFrom, which must not overlap
// with other calls.
type Model struct {
	order int
	tally
	seen *hashSet // FoldHash of every learned name

	// positionless marks models loaded from a file without word positions.
	// They stay that way even as they learn more, because positions counted
	// only for newer names would make every older word look like it never
	// starts a name.
	positionless bool

	tableMu sync.Mutex
	table   atomic.Pointer[tables] // sampling tables built from grams; nil when stale
}

// tally holds everything counted from learned names. Workers fill their own
// tallies in parallel, which are then merged.
type tally struct {
	// grams counts every context of 0..order characters together with the
	// character that followed it, keyed by the context's UTF-8 bytes then
	// the next character's.
	grams   map[string]uint32
	words   map[string]uint32 // lowercased words of 2+ letters
	leads   map[string]uint32 // how often each word was a name's first word
	numbers map[string]uint32 // digit runs
	stats   Stats
}

// Stats summarises the usernames a model has learned.
type Stats struct {
	Names    uint64 // usernames learned
	TotalLen uint64 // their combined length in characters
	MinLen   int
	MaxLen   int
}

// New returns an empty model that conditions on up to order characters.
func New(order int) (*Model, error) {
	if order < 1 || order > MaxOrder {
		return nil, fmt.Errorf("order must be between 1 and %d, got %d", MaxOrder, order)
	}
	return newModel(order), nil
}

func newModel(order int) *Model {
	return &Model{order: order, tally: newTally(), seen: newHashSet()}
}

func newTally() tally {
	return tally{
		grams:   make(map[string]uint32),
		words:   make(map[string]uint32),
		leads:   make(map[string]uint32),
		numbers: make(map[string]uint32),
	}
}

// Order reports the longest context the model was trained with.
func (m *Model) Order() int { return m.order }

// Stats summarises what the model has learned.
func (m *Model) Stats() Stats { return m.stats }

// Patterns reports how many distinct character patterns the model has seen.
func (m *Model) Patterns() int { return len(m.grams) }

// Knows reports whether name (case-insensitively) was learned from.
func (m *Model) Knows(name string) bool { return m.seen.has(FoldHash(name)) }

// KnowsHash is Knows for a name whose FoldHash is h.
func (m *Model) KnowsHash(h uint64) bool { return m.seen.has(h) }

// Learn adds name to the model. It returns a *GarbageError without changing
// the model if name fails Check, and false if the name (case-insensitively)
// was already learned, so re-training on the same list does not skew counts.
func (m *Model) Learn(name string) (bool, error) {
	if err := Check(name); err != nil {
		return false, err
	}
	if !m.seen.add(FoldHash(name)) {
		return false, nil
	}
	m.tally.add(m.order, name, nil)
	m.table.Store(nil)
	return true, nil
}

// LearnResult tallies a LearnFrom run.
type LearnResult struct {
	Added    int            // new usernames learned
	Known    int            // usernames already learned, skipped
	Rejected map[string]int // garbage usernames skipped, by reason
}

// RejectedTotal is the number of garbage usernames skipped.
func (r LearnResult) RejectedTotal() int {
	n := 0
	for _, c := range r.Rejected {
		n += c
	}
	return n
}

// LearnFrom learns every username in r (see ScanNames). It checks and
// deduplicates names in input order, so the result is the same as calling
// Learn on each, and counts their patterns on all CPU cores. If onRejected is
// non-nil it is called with every garbage username and the reason.
func (m *Model) LearnFrom(r io.Reader, onRejected func(name, reason string)) (LearnResult, error) {
	workers := runtime.GOMAXPROCS(0)
	batches := make(chan []string, workers)
	tallies := make(chan tally, workers)
	for range workers {
		go func() {
			t := newTally()
			var offs []int
			for batch := range batches {
				for _, name := range batch {
					offs = t.add(m.order, name, offs)
				}
			}
			tallies <- t
		}()
	}

	res := LearnResult{Rejected: map[string]int{}}
	batch := make([]string, 0, batchSize)
	err := ScanNames(r, func(name string) {
		if err := Check(name); err != nil {
			reason := err.(*GarbageError).Reason
			res.Rejected[reason]++
			if onRejected != nil {
				onRejected(name, reason)
			}
			return
		}
		if !m.seen.add(FoldHash(name)) {
			res.Known++
			return
		}
		res.Added++
		if batch = append(batch, name); len(batch) == batchSize {
			batches <- batch
			batch = make([]string, 0, batchSize)
		}
	})
	if len(batch) > 0 {
		batches <- batch
	}
	close(batches)

	for range workers {
		t := <-tallies
		m.tally.merge(&t)
	}
	m.table.Store(nil)
	return res, err
}

// ScanNames calls fn with every username in r: one per line, trimmed, with
// blank lines and lines starting with # skipped.
func ScanNames(r io.Reader, fn func(string)) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" && line[0] != '#' {
			fn(line)
		}
	}
	return sc.Err()
}

// add counts name's patterns, words and numbers. offs is scratch space that
// add returns for reuse. Map keys are substrings of one padded string, so
// counting a pattern the tally has already seen allocates nothing.
func (t *tally) add(order int, name string, offs []int) []int {
	padded := strings.Repeat(string(rune(startRune)), order) + name + string(rune(endRune))
	offs = offs[:0]
	for i := range padded {
		offs = append(offs, i)
	}
	offs = append(offs, len(padded))
	for i := order; i < len(offs)-1; i++ {
		end := offs[i+1]
		for k := 0; k <= order; k++ {
			t.grams[padded[offs[i-k]:end]]++
		}
	}

	// Only names that mark where words start and end ("ShadowWolf",
	// "shadow_wolf") teach words; "shadowwolf" would teach the compound.
	// Letters around a digit inside a word are leetspeak ("Fr0zen"), so
	// they are fragments, not words.
	segs := Split(name)
	wordCount := 0
	for _, seg := range segs {
		if seg.Kind == Word {
			wordCount++
		}
	}
	lead := true // the next content word is the name's first
	for i, seg := range segs {
		switch {
		case seg.Kind == Word && utf8.RuneCountInString(seg.Text) == 1:
			// An initial is the name's first word (r.lindqvist), except the x
			// of an xX...Xx wrapper.
			if seg.Text != "x" && seg.Text != "X" {
				lead = false
			}
		case seg.Kind == Word && wordCount >= 2 && !leetNeighbour(segs, i):
			w := strings.ToLower(seg.Text)
			t.words[w]++
			if !IsAffix(w) {
				if lead {
					t.leads[w]++
				}
				lead = false
			}
		case seg.Kind == Number:
			t.numbers[seg.Text]++
		}
	}

	n := utf8.RuneCountInString(name)
	if t.stats.Names == 0 || n < t.stats.MinLen {
		t.stats.MinLen = n
	}
	t.stats.MaxLen = max(t.stats.MaxLen, n)
	t.stats.Names++
	t.stats.TotalLen += uint64(n)
	return offs
}

// leetNeighbour reports whether segs[i] is a fragment of a leetspeak word:
// the letters before a single look-alike digit between words ("Fr" in
// "Fr0zen", "Byt" in "Byt3Master"), or lowercase letters continuing after
// one ("zen"). A capitalised word after the digit ("Master") is a new word,
// and longer numbers ("Agent007Bond") aren't leetspeak.
func leetNeighbour(segs []Segment, i int) bool {
	leet := func(j int) bool {
		return j > 0 && j < len(segs)-1 && segs[j].Kind == Number &&
			len(segs[j].Text) == 1 && strings.Contains("013457", segs[j].Text) &&
			segs[j-1].Kind == Word && segs[j+1].Kind == Word
	}
	if leet(i + 1) {
		return true
	}
	if leet(i - 1) {
		r, _ := utf8.DecodeRuneInString(segs[i].Text)
		return !unicode.IsUpper(r)
	}
	return false
}

// merge adds o's counts to t, reusing whichever maps are larger.
func (t *tally) merge(o *tally) {
	mergeCounts(&t.grams, o.grams)
	mergeCounts(&t.words, o.words)
	mergeCounts(&t.leads, o.leads)
	mergeCounts(&t.numbers, o.numbers)
	if o.stats.Names == 0 {
		return
	}
	if t.stats.Names == 0 || o.stats.MinLen < t.stats.MinLen {
		t.stats.MinLen = o.stats.MinLen
	}
	t.stats.MaxLen = max(t.stats.MaxLen, o.stats.MaxLen)
	t.stats.Names += o.stats.Names
	t.stats.TotalLen += o.stats.TotalLen
}

func mergeCounts(dst *map[string]uint32, src map[string]uint32) {
	if len(src) > len(*dst) {
		src, *dst = *dst, src
	}
	for k, v := range src {
		(*dst)[k] += v
	}
}

const (
	// minWordsToJudge is how many words a model must know before WordLike
	// judges words; with fewer it can't tell a new word from a fragment.
	minWordsToJudge = 50
	// maxWordBytes bounds the learned words WordLike tries when splitting.
	maxWordBytes = 24
	// minPieceBytes is the shortest learned word counted as part of a
	// run-together word, so two-letter names ("ha") can't glue junk on.
	minPieceBytes = 3
)

// affixes are the wrappers and connectors usernames are built with rather
// than words that carry meaning: xX...Xx, its/the/real/my + name,
// name.and.name, peach.jpg, lopez.io, NovaTV.
var affixes = map[string]bool{
	"xx": true, "the": true, "its": true, "im": true, "iam": true, "real": true,
	"just": true, "hey": true, "not": true, "nota": true, "ii": true, "and": true,
	"xo": true, "jpg": true, "png": true, "exe": true, "tv": true, "yt": true,
	"ttv": true, "hd": true, "gg": true, "mr": true, "ms": true, "mrs": true,
	"lil": true, "official": true, "com": true, "io": true, "co": true,
	"www": true, "xd": true, "lol": true, "my": true, "get": true, "miss": true,
}

// IsAffix reports whether word (in any case) is a username affix such as
// "xX", "its" or "jpg", which frames a name rather than carrying meaning.
func IsAffix(word string) bool { return affixes[strings.ToLower(word)] }

// HasWord reports whether word (in any case) was learned as a word.
func (m *Model) HasWord(word string) bool {
	_, ok := m.words[strings.ToLower(word)]
	return ok
}

// LeadShare reports the share of a learned word's appearances in which it
// was a name's first word: near 1 for words like "dark" and "john", near 0
// for "wolf" and "smith". It reports false for unknown words, and for any
// word in models saved before positions were recorded.
func (m *Model) LeadShare(word string) (float64, bool) {
	w := strings.ToLower(word)
	n := m.words[w]
	if n == 0 || m.positionless {
		return 0, false
	}
	return float64(m.leads[w]) / float64(n), true
}

// minPositionShare is how often a word must start names, or come later in
// them, to be used in that position.
const minPositionShare = 0.2

// Positions reports whether a learned word can start a name (it does in at
// least a fifth of its uses, like "dark" or "john") and whether it can come
// later (likewise, like "wolf" or "smith"). Words the model has no position
// data for can do both.
func (m *Model) Positions(word string) (lead, follow bool) {
	share, ok := m.LeadShare(word)
	if !ok {
		return true, true
	}
	return share >= minPositionShare, share <= 1-minPositionShare
}

// WordLike reports whether word, a run of letters, is made of words the
// model learned, as KnowsWord describes. It reports true for single letters
// and affixes, and when the model knows too few words to judge.
func (m *Model) WordLike(word string) bool {
	if len(m.words) < minWordsToJudge || utf8.RuneCountInString(word) < 2 || IsAffix(word) {
		return true
	}
	return m.KnowsWord(word)
}

// KnowsWord reports whether word is made of words the model learned: one of
// them; several run together, each piece 3+ letters, not an affix, and after
// the first one that can come later in names ("juanbaker", not "stavroshua");
// or a one-letter initial before a word that can come later ("jsmith", not
// "aming"). An initial at the end ("smithj") is not allowed, because it
// would also pass fragments like "cobrah".
func (m *Model) KnowsWord(word string) bool {
	w := strings.ToLower(word)
	if m.compound(w) {
		return true
	}
	_, first := utf8.DecodeRuneInString(w)
	rest := w[first:]
	if m.positionless { // no position data: accept what compound does
		return m.compound(rest)
	}
	_, known := m.words[rest]
	_, follow := m.Positions(rest)
	return known && follow && !affixes[rest]
}

// NameWordsLike reports whether every word in name is WordLike.
func (m *Model) NameWordsLike(name string) bool {
	for _, seg := range Split(name) {
		if seg.Kind == Word && !m.WordLike(seg.Text) {
			return false
		}
	}
	return true
}

// compound reports whether w is one or more learned words run together.
func (m *Model) compound(w string) bool {
	if _, ok := m.words[w]; ok {
		return true
	}
	var ends [4*MaxNameLen + 1]bool // ends[i]: w[:i] splits into learned words
	if len(w) < 2*minPieceBytes || len(w) >= len(ends) {
		return false
	}
	ends[0] = true
	for i := minPieceBytes; i <= len(w); i++ {
		for j := max(0, i-maxWordBytes); j <= i-minPieceBytes && !ends[i]; j++ {
			if ends[j] {
				piece := w[j:i]
				_, known := m.words[piece]
				ends[i] = known && !affixes[piece]
				if ends[i] && j > 0 {
					_, ends[i] = m.Positions(piece)
				}
			}
		}
	}
	return ends[len(w)]
}

// TopWords returns up to n learned words, lowercased, most common first.
func (m *Model) TopWords(n int) []string { return mostCommon(m.words, n) }

// TopNumbers returns up to n learned numbers, most common first.
func (m *Model) TopNumbers(n int) []string { return mostCommon(m.numbers, n) }

// Vocabulary reports how many distinct words and numbers the model learned.
func (m *Model) Vocabulary() (words, numbers int) { return len(m.words), len(m.numbers) }

func mostCommon(counts map[string]uint32, n int) []string {
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
	return keys[:min(n, len(keys))]
}

// tables index what followed each context, for sampling.
type tables struct {
	// byText is keyed by a context's UTF-8 bytes, for contexts of any length.
	byText map[string]dist
	// full holds just the order-length contexts keyed by packContext, for
	// models of order up to maxPackedOrder: the lookup Complete makes for
	// every character it writes, done without hashing strings.
	full map[uint64]dist
}

// Packing 21-bit runes into a uint64 fits three of them.
const (
	runeBits       = 21
	maxPackedOrder = 3
)

// pushRune shifts r into a packed context of order runes.
func pushRune(ctx uint64, r rune, order int) uint64 {
	return (ctx<<runeBits | uint64(r)) & (1<<(runeBits*order) - 1)
}

// dist lists the characters that followed a context, sorted so endRune (the
// smallest) is first if present, each with the cumulative count up to and
// including it. It is one flat array so sampling touches little memory.
type dist []successor

type successor struct {
	r   rune
	cum uint32
}

// sampler returns the sampling tables, building them on first use after
// learning.
func (m *Model) sampler() *tables {
	if t := m.table.Load(); t != nil {
		return t
	}
	m.tableMu.Lock()
	defer m.tableMu.Unlock()
	if t := m.table.Load(); t != nil {
		return t
	}

	type entry struct {
		r rune
		c uint32
	}
	rows := make(map[string][]entry)
	for key, c := range m.grams {
		r, size := utf8.DecodeLastRuneInString(key)
		ctx := key[:len(key)-size]
		rows[ctx] = append(rows[ctx], entry{r, c})
	}
	t := &tables{byText: make(map[string]dist, len(rows))}
	if m.order <= maxPackedOrder {
		t.full = make(map[uint64]dist)
	}
	for ctx, entries := range rows {
		sort.Slice(entries, func(i, j int) bool { return entries[i].r < entries[j].r })
		d := make(dist, len(entries))
		var sum uint32
		for i, e := range entries {
			sum += e.c
			d[i] = successor{e.r, sum}
		}
		t.byText[ctx] = d
		if t.full != nil && utf8.RuneCountInString(ctx) == m.order {
			var key uint64
			for _, r := range ctx {
				key = pushRune(key, r, m.order)
			}
			t.full[key] = d
		}
	}
	m.table.Store(t)
	return t
}

// pick samples a character in proportion to its count, or returns -1 if
// allowEnd is false and ending the name is the only option.
func (d dist) pick(rng *rand.Rand, allowEnd bool) rune {
	total := d[len(d)-1].cum
	var lo uint32
	if !allowEnd && d[0].r == endRune {
		if lo = d[0].cum; lo == total {
			return -1
		}
	}
	x := lo + rng.Uint32N(total-lo)
	// Find the first cumulative count above x.
	i, j := 0, len(d)-1
	for i < j {
		if h := int(uint(i+j) >> 1); d[h].cum > x {
			j = h
		} else {
			i = h + 1
		}
	}
	return d[i].r
}

// pickTemp samples with counts raised to the power 1/temperature.
func (d dist) pickTemp(rng *rand.Rand, temperature float64) rune {
	if temperature == 1 {
		return d.pick(rng, true)
	}
	weights := make([]float64, len(d))
	var total float64
	var prev uint32
	for i, s := range d {
		weights[i] = math.Pow(float64(s.cum-prev), 1/temperature)
		total += weights[i]
		prev = s.cum
	}
	x := rng.Float64() * total
	for i, w := range weights {
		if x -= w; x < 0 {
			return d[i].r
		}
	}
	return d[len(d)-1].r
}

// ctxStart returns where the last k characters of buf begin.
func ctxStart(buf []byte, k int) int {
	i := len(buf)
	for ; k > 0 && i > 0; k-- {
		_, size := utf8.DecodeLastRune(buf[:i])
		i -= size
	}
	return i
}

// Complete keeps prefix and lets the model write the rest of the name, adding
// at least one character. It only continues from patterns the model has
// actually seen with its full context, never guessing from shorter ones, and
// returns false if it would have to, or if the result would be longer than
// maxLen characters.
func (m *Model) Complete(rng *rand.Rand, prefix string, maxLen int) (string, bool) {
	t := m.sampler()
	buf := make([]byte, 0, m.order+len(prefix)+4*8)
	for range m.order {
		buf = append(buf, startRune)
	}
	buf = append(buf, prefix...)
	var packed uint64
	if t.full != nil {
		for i := 0; i < m.order; i++ {
			packed = pushRune(packed, startRune, m.order)
		}
		for _, r := range prefix {
			packed = pushRune(packed, r, m.order)
		}
	}
	n := utf8.RuneCountInString(prefix)
	for added := 0; ; added++ {
		var d dist
		if t.full != nil {
			d = t.full[packed]
		} else {
			d = t.byText[string(buf[ctxStart(buf, m.order):])]
		}
		if d == nil {
			return "", false
		}
		r := d.pick(rng, added > 0)
		if r < 0 {
			return "", false
		}
		if r == endRune {
			return string(buf[m.order:]), true
		}
		if n++; n > maxLen {
			return "", false
		}
		buf = utf8.AppendRune(buf, r)
		packed = pushRune(packed, r, m.order)
	}
}

// GenerateOptions controls username generation.
type GenerateOptions struct {
	MinLen int // minimum length in characters
	MaxLen int // maximum length in characters

	// Temperature reshapes the learned probabilities: 1 samples them as
	// learned, below 1 favours the most common patterns, above 1 flattens
	// them toward rarer ones.
	Temperature float64

	// Order limits how many previous characters are used, from 1 to the
	// model's order. Lower values give wilder names. 0 means the model's order.
	Order int

	// AllowKnown permits returning names the model was trained on.
	AllowKnown bool

	// Progress, if non-nil, is called with the number of names found so far
	// each time one is found.
	Progress func(found int)
}

// Generate returns up to n distinct usernames that pass Check and whose
// words are WordLike. It returns
// fewer than n when the model cannot produce enough distinct names satisfying
// opts, which typically means it was trained on too few names for its order.
func (m *Model) Generate(rng *rand.Rand, n int, opts GenerateOptions) ([]string, error) {
	if m.stats.Names == 0 {
		return nil, errors.New("model has not learned any usernames yet")
	}
	if opts.Order == 0 {
		opts.Order = m.order
	}
	switch {
	case n < 1:
		return nil, fmt.Errorf("count must be at least 1, got %d", n)
	case opts.MinLen < 1:
		return nil, fmt.Errorf("minimum length must be at least 1, got %d", opts.MinLen)
	case opts.MinLen > MaxNameLen:
		return nil, fmt.Errorf("minimum length %d exceeds the %d-character username limit", opts.MinLen, MaxNameLen)
	case opts.MaxLen < opts.MinLen:
		return nil, fmt.Errorf("maximum length %d is below minimum length %d", opts.MaxLen, opts.MinLen)
	case opts.Temperature <= 0:
		return nil, fmt.Errorf("temperature must be positive, got %g", opts.Temperature)
	case opts.Order < 1 || opts.Order > m.order:
		return nil, fmt.Errorf("order must be between 1 and the model's order %d, got %d", m.order, opts.Order)
	}

	out := make([]string, 0, n)
	produced := make(map[uint64]struct{}, n)
	for attempts := 0; len(out) < n && attempts < n*500; attempts++ {
		name, ok := m.sample(rng, opts)
		if !ok {
			continue
		}
		h := FoldHash(name)
		if _, dup := produced[h]; dup {
			continue
		}
		if (!opts.AllowKnown && m.seen.has(h)) || Check(name) != nil || !m.NameWordsLike(name) {
			continue
		}
		produced[h] = struct{}{}
		out = append(out, name)
		if opts.Progress != nil {
			opts.Progress(len(out))
		}
	}
	return out, nil
}

// sample walks the chain once from the start, backing off to shorter
// contexts when needed, and returns false if the result is out of bounds.
func (m *Model) sample(rng *rand.Rand, opts GenerateOptions) (string, bool) {
	t := m.sampler()
	buf := make([]byte, 0, m.order+opts.MaxLen+4)
	for range m.order {
		buf = append(buf, startRune)
	}
	n := 0
	for {
		r := rune(-1)
		for k := opts.Order; k >= 0 && r < 0; k-- {
			if d := t.byText[string(buf[ctxStart(buf, k):])]; d != nil {
				r = d.pickTemp(rng, opts.Temperature)
			}
		}
		if r < 0 || r == endRune {
			break
		}
		if n++; n > opts.MaxLen {
			return "", false
		}
		buf = utf8.AppendRune(buf, r)
	}
	if n < opts.MinLen {
		return "", false
	}
	return string(buf[m.order:]), true
}
