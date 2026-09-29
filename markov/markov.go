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
	"unicode"
	"unicode/utf8"
)

const (
	formatVersion = 1

	// MaxOrder bounds how many previous characters a model can condition on.
	MaxOrder = 8

	// startRune and endRune pad each name so the model learns how names
	// begin and end. Both are control characters, which Learn rejects in
	// input, so they can never collide with a real username character.
	startRune = '\x02'
	endRune   = '\x03'
)

// ErrInvalidName is returned by Learn for input that cannot be a username.
var ErrInvalidName = errors.New("invalid username")

// Model counts, for every context of 0..order preceding characters, how often
// each next character follows it.
type Model struct {
	order  int
	counts map[string]map[rune]int
	names  []string            // every learned name, in learning order
	seen   map[string]struct{} // lowercased names, for dedupe and novelty
}

// New returns an empty model that conditions on up to order characters.
func New(order int) (*Model, error) {
	if order < 1 || order > MaxOrder {
		return nil, fmt.Errorf("order must be between 1 and %d, got %d", MaxOrder, order)
	}
	return &Model{
		order:  order,
		counts: make(map[string]map[rune]int),
		seen:   make(map[string]struct{}),
	}, nil
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

// Learn adds name to the model. It returns false without changing the model
// if the name (case-insensitively) was already learned, so re-training on the
// same list does not skew the counts.
func (m *Model) Learn(name string) (bool, error) {
	if err := validate(name); err != nil {
		return false, err
	}
	key := strings.ToLower(name)
	if _, ok := m.seen[key]; ok {
		return false, nil
	}

	runes := []rune(name)
	padded := make([]rune, 0, m.order+len(runes)+1)
	for range m.order {
		padded = append(padded, startRune)
	}
	padded = append(padded, runes...)
	padded = append(padded, endRune)

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
		}
	}

	m.seen[key] = struct{}{}
	m.names = append(m.names, name)
	return true, nil
}

func validate(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty", ErrInvalidName)
	}
	if !utf8.ValidString(name) {
		return fmt.Errorf("%w: not valid UTF-8", ErrInvalidName)
	}
	for _, r := range name {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("%w: %q contains whitespace or control characters", ErrInvalidName, name)
		}
	}
	return nil
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

// Generate returns up to n distinct usernames. It returns fewer than n when
// the model cannot produce enough distinct names satisfying opts, which
// typically means it was trained on too few names for its order.
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
		produced[key] = struct{}{}
		out = append(out, name)
	}
	return out, nil
}

// sample walks the chain once, returning false if the result is out of bounds.
func (m *Model) sample(rng *rand.Rand, opts GenerateOptions) (string, bool) {
	ctx := make([]rune, opts.Order)
	for i := range ctx {
		ctx[i] = startRune
	}
	var out []rune
	for {
		next, ok := m.next(rng, ctx, opts.Temperature)
		if !ok || next == endRune {
			break
		}
		out = append(out, next)
		if len(out) > opts.MaxLen {
			return "", false
		}
		copy(ctx, ctx[1:])
		ctx[len(ctx)-1] = next
	}
	if len(out) < opts.MinLen {
		return "", false
	}
	return string(out), true
}

// next samples the character following ctx, backing off to shorter contexts
// when the full one was never observed.
func (m *Model) next(rng *rand.Rand, ctx []rune, temperature float64) (rune, bool) {
	for k := len(ctx); k >= 0; k-- {
		row := m.counts[string(ctx[len(ctx)-k:])]
		if len(row) == 0 {
			continue
		}
		// Sort so a fixed seed always yields the same names.
		candidates := make([]rune, 0, len(row))
		for r := range row {
			candidates = append(candidates, r)
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
		}
		m.counts[ctx] = counts
	}
	for _, name := range f.Names {
		m.seen[strings.ToLower(name)] = struct{}{}
	}
	m.names = f.Names
	return m, nil
}
