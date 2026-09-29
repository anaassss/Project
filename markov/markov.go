// Package markov implements a character-level Markov model that learns the
// shape of usernames (which characters tend to follow which) and generates
// new usernames from what it has learned. Models can be saved to and loaded
// from disk so learning accumulates across runs.
package markov

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	formatVersion = 1

	// MaxOrder bounds how many previous characters a model can condition on.
	MaxOrder = 8

	// startRune and endRune pad each name so the model learns how names
	// begin and end. Both are control characters, which Check rejects, so
	// they can never collide with a real username character.
	startRune = '\x02'
	endRune   = '\x03'

	// backoffPenalty is the log-weight Score applies each time it falls back
	// to a shorter context ("stupid backoff", weight 0.4).
	backoffPenalty = -0.916290731874155
	// unseenPenalty is the log-weight Score gives a character the model has
	// never seen at all.
	unseenPenalty = -13.815510557964274 // log(1e-6)
)

// Model counts, for every context of 0..order preceding characters, how often
// each next character follows it.
type Model struct {
	order  int
	counts map[string]map[rune]int
	totals map[string]int      // sum of each counts row, for Score
	names  []string            // every learned name, in learning order
	seen   map[string]struct{} // lowercased names, for dedupe and novelty
}

// New returns an empty model that conditions on up to order characters.
func New(order int) (*Model, error) {
	if order < 1 || order > MaxOrder {
		return nil, fmt.Errorf("order must be between 1 and %d, got %d", MaxOrder, order)
	}
	return newModel(order), nil
}

func newModel(order int) *Model {
	return &Model{
		order:  order,
		counts: make(map[string]map[rune]int),
		totals: make(map[string]int),
		seen:   make(map[string]struct{}),
	}
}

// Order reports the longest context the model was trained with.
func (m *Model) Order() int { return m.order }

// Names returns the usernames the model has learned from.
func (m *Model) Names() []string { return m.names }

// Contexts reports how many distinct contexts the model has observed.
func (m *Model) Contexts() int { return len(m.counts) }

// Knows reports whether name (case-insensitively) was learned from.
func (m *Model) Knows(name string) bool {
	_, ok := m.seen[strings.ToLower(name)]
	return ok
}

// Learn adds name to the model. It returns a *GarbageError without changing
// the model if name fails Check, and false if the name (case-insensitively)
// was already learned, so re-training on the same list does not skew counts.
func (m *Model) Learn(name string) (bool, error) {
	if err := Check(name); err != nil {
		return false, err
	}
	if m.Knows(name) {
		return false, nil
	}
	m.learn(name)
	return true, nil
}

// learn counts name's transitions without checking it.
func (m *Model) learn(name string) {
	padded := m.pad(name)
	for i := m.order; i < len(padded); i++ {
		next := padded[i]
		for k := 0; k <= m.order; k++ {
			ctx := string(padded[i-k : i])
			row := m.counts[ctx]
			if row == nil {
				row = make(map[rune]int)
				m.counts[ctx] = row
			}
			row[next]++
			m.totals[ctx]++
		}
	}

	m.seen[strings.ToLower(name)] = struct{}{}
	m.names = append(m.names, name)
}

// pad surrounds name with start and end markers so the first and last
// characters have contexts too.
func (m *Model) pad(name string) []rune {
	runes := []rune(name)
	padded := make([]rune, 0, m.order+len(runes)+1)
	for range m.order {
		padded = append(padded, startRune)
	}
	padded = append(padded, runes...)
	return append(padded, endRune)
}

// Score rates how well name fits what the model has learned: the average log
// probability of each character (and of the name ending there) given the
// characters before it. When the full context never preceded a character it
// backs off to shorter contexts with a penalty. Higher is more typical.
func (m *Model) Score(name string) float64 {
	return m.score(name, nil)
}

// HeldOutScores returns, for each learned name, the Score it would get had
// the model never learned it. Learned names score unrealistically well on
// their own patterns; these scores show what a genuinely new but typical
// username scores.
func (m *Model) HeldOutScores() []float64 {
	scores := make([]float64, len(m.names))
	for i, name := range m.names {
		own := newModel(m.order)
		own.learn(name)
		scores[i] = m.score(name, own)
	}
	return scores
}

// score implements Score, ignoring the counts in exclude if it is non-nil.
func (m *Model) score(name string, exclude *Model) float64 {
	padded := m.pad(name)
	var total float64
	for i := m.order; i < len(padded); i++ {
		total += m.logProb(padded[i-m.order:i], padded[i], exclude)
	}
	return total / float64(len(padded)-m.order)
}

func (m *Model) logProb(ctx []rune, next rune, exclude *Model) float64 {
	var penalty float64
	for k := len(ctx); k >= 0; k-- {
		key := string(ctx[len(ctx)-k:])
		c, total := m.counts[key][next], m.totals[key]
		if exclude != nil {
			c -= exclude.counts[key][next]
			total -= exclude.totals[key]
		}
		if c > 0 {
			return penalty + math.Log(float64(c)/float64(total))
		}
		penalty += backoffPenalty
	}
	return penalty + unseenPenalty
}

// Clean rebuilds the model from only the learned names that pass Check,
// removing anything learned before the current garbage rules, and returns
// the names it dropped.
func (m *Model) Clean() []string {
	clean := newModel(m.order)
	var dropped []string
	for _, name := range m.names {
		if Check(name) != nil {
			dropped = append(dropped, name)
			continue
		}
		clean.learn(name)
	}
	*m = *clean
	return dropped
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
}

// Generate returns up to n distinct usernames that pass Check. It returns
// fewer than n when the model cannot produce enough distinct names satisfying
// opts, which typically means it was trained on too few names for its order.
func (m *Model) Generate(rng *rand.Rand, n int, opts GenerateOptions) ([]string, error) {
	if len(m.names) == 0 {
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
	produced := make(map[string]struct{}, n)
	for attempts := 0; len(out) < n && attempts < n*500; attempts++ {
		name, ok := m.sample(rng, opts)
		if !ok {
			continue
		}
		key := strings.ToLower(name)
		if _, dup := produced[key]; dup {
			continue
		}
		if !opts.AllowKnown && m.Knows(name) {
			continue
		}
		if Check(name) != nil {
			continue
		}
		produced[key] = struct{}{}
		out = append(out, name)
	}
	return out, nil
}

// sample walks the chain once, returning false if the result is out of bounds.
func (m *Model) sample(rng *rand.Rand, opts GenerateOptions) (string, bool) {
	out, ok := m.walk(rng, nil, 0, opts.Order, opts.Temperature, opts.MaxLen)
	if !ok || len(out) < opts.MinLen {
		return "", false
	}
	return string(out), true
}

// Complete keeps prefix and lets the model write the rest of the name, adding
// at least one character. It returns false if the result would be longer than
// maxLen characters.
func (m *Model) Complete(rng *rand.Rand, prefix string, maxLen int) (string, bool) {
	out, ok := m.walk(rng, []rune(prefix), 1, m.order, 1, maxLen)
	return string(out), ok
}

// walk extends prefix one sampled character at a time until the model ends
// the name, not allowing an end before minNew characters have been added.
func (m *Model) walk(rng *rand.Rand, prefix []rune, minNew, order int, temperature float64, maxLen int) ([]rune, bool) {
	ctx := make([]rune, order)
	for i := range ctx {
		ctx[i] = startRune
	}
	push := func(r rune) {
		copy(ctx, ctx[1:])
		ctx[len(ctx)-1] = r
	}
	for _, r := range prefix {
		push(r)
	}
	out := append([]rune(nil), prefix...)
	for {
		next, ok := m.next(rng, ctx, temperature, len(out)-len(prefix) >= minNew)
		if !ok || next == endRune {
			break
		}
		out = append(out, next)
		if len(out) > maxLen {
			return nil, false
		}
		push(next)
	}
	return out, true
}

// next samples the character following ctx, backing off to shorter contexts
// when the full one was never observed (or, if !allowEnd, was only ever
// followed by the end of a name).
func (m *Model) next(rng *rand.Rand, ctx []rune, temperature float64, allowEnd bool) (rune, bool) {
	for k := len(ctx); k >= 0; k-- {
		row := m.counts[string(ctx[len(ctx)-k:])]
		// Sort so a fixed seed always yields the same names.
		candidates := make([]rune, 0, len(row))
		for r := range row {
			if allowEnd || r != endRune {
				candidates = append(candidates, r)
			}
		}
		if len(candidates) == 0 {
			continue
		}
		sort.Slice(candidates, func(i, j int) bool { return candidates[i] < candidates[j] })

		weights := make([]float64, len(candidates))
		var total float64
		for i, r := range candidates {
			weights[i] = math.Pow(float64(row[r]), 1/temperature)
			total += weights[i]
		}
		pick := rng.Float64() * total
		for i, w := range weights {
			pick -= w
			if pick < 0 {
				return candidates[i], true
			}
		}
		return candidates[len(candidates)-1], true
	}
	return 0, false
}

// file is the on-disk representation of a Model.
type file struct {
	Version int                       `json:"version"`
	Order   int                       `json:"order"`
	Names   []string                  `json:"names"`
	Counts  map[string]map[string]int `json:"counts"`
}

// Save writes the model to path atomically, so an interrupted save never
// leaves a corrupt model behind.
func (m *Model) Save(path string) (err error) {
	f := file{
		Version: formatVersion,
		Order:   m.order,
		Names:   m.names,
		Counts:  make(map[string]map[string]int, len(m.counts)),
	}
	for ctx, row := range m.counts {
		out := make(map[string]int, len(row))
		for r, c := range row {
			out[string(r)] = c
		}
		f.Counts[ctx] = out
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()

	w := bufio.NewWriter(tmp)
	if err = json.NewEncoder(w).Encode(f); err != nil {
		return err
	}
	if err = w.Flush(); err != nil {
		return err
	}
	if err = tmp.Chmod(0o644); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Load reads a model previously written by Save.
func Load(path string) (*Model, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("%s: not a valid model file: %w", path, err)
	}
	if f.Version != formatVersion {
		return nil, fmt.Errorf("%s: unsupported model version %d (want %d)", path, f.Version, formatVersion)
	}
	m, err := New(f.Order)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for ctx, row := range f.Counts {
		if utf8.RuneCountInString(ctx) > m.order {
			return nil, fmt.Errorf("%s: context %q is longer than order %d", path, ctx, m.order)
		}
		counts := make(map[rune]int, len(row))
		for s, c := range row {
			r, size := utf8.DecodeRuneInString(s)
			if size == 0 || size != len(s) || c < 1 {
				return nil, fmt.Errorf("%s: corrupt count %q=%d in context %q", path, s, c, ctx)
			}
			counts[r] = c
			m.totals[ctx] += c
		}
		m.counts[ctx] = counts
	}
	for _, name := range f.Names {
		m.seen[strings.ToLower(name)] = struct{}{}
	}
	m.names = f.Names
	return m, nil
}
