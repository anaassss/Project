// Package knowledge is usergen's built-in knowledge of how real usernames
// are made. Rather than a list of anyone's actual accounts, it generates
// usernames from the patterns people use, combining common first and last
// names and everyday words the way real accounts do:
//
//   - Email-style: john.smith, jsmith, smithj, j.smith, john.smith92,
//     jsmith1987, john.m.smith
//   - Normal (social and gaming): SilentWolf, itsmike, sarahbakes, mike_92,
//     TheNightOwl, xXDragonSlayerXx, NovaTV
//
// Models returns this knowledge as ready-made models, one per style, which
// usergen uses as Claude's knowledge alongside whatever the user trains.
package knowledge

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/anaassss/Project/markov"
)

// Style is a kind of username.
type Style int

const (
	Email  Style = iota // name-based, like email addresses: john.smith92
	Normal              // social media and gaming handles: SilentWolf
)

// Styles lists every style.
var Styles = []Style{Email, Normal}

func (s Style) String() string {
	if s == Email {
		return "email"
	}
	return "normal"
}

// ParseStyle reads a style name: "email", "normal" or "both".
func ParseStyle(name string) ([]Style, error) {
	switch strings.ToLower(name) {
	case "email":
		return []Style{Email}, nil
	case "normal":
		return []Style{Normal}, nil
	case "both":
		return Styles, nil
	}
	return nil, fmt.Errorf("unknown style %q: use email, normal or both", name)
}

// pattern builds one username. Patterns are weighted by roughly how common
// each shape is among real accounts.
type pattern struct {
	weight int
	build  func(r *rand.Rand) string
}

func pick(r *rand.Rand, list []string) string { return list[r.IntN(len(list))] }

func title(s string) string {
	c, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(c)) + s[size:]
}

func initial(s string) string {
	_, size := utf8.DecodeRuneInString(s)
	return s[:size]
}

// digits returns the numbers people add to usernames: mostly birth years,
// in full or short, else small or favourite numbers.
func digits(r *rand.Rand) string {
	switch x := r.IntN(100); {
	case x < 40:
		return fmt.Sprintf("%02d", (70+r.IntN(40))%100) // 70..99, 00..09
	case x < 70:
		return strconv.Itoa(1965 + r.IntN(46)) // 1965..2010
	case x < 90:
		return strconv.Itoa(1 + r.IntN(99))
	default:
		return pick(r, []string{"123", "007", "01", "11", "22", "777"})
	}
}

// separator returns what joins two parts: mostly nothing, else _ or .
func separator(r *rand.Rand) string {
	switch x := r.IntN(10); {
	case x < 6:
		return ""
	case x < 8:
		return "_"
	}
	return "."
}

var emailPatterns = []pattern{
	{22, func(r *rand.Rand) string { return pick(r, firstNames) + "." + pick(r, lastNames) }},
	{10, func(r *rand.Rand) string { return pick(r, firstNames) + pick(r, lastNames) }},
	{5, func(r *rand.Rand) string { return pick(r, firstNames) + "_" + pick(r, lastNames) }},
	{12, func(r *rand.Rand) string { return initial(pick(r, firstNames)) + pick(r, lastNames) }},
	{5, func(r *rand.Rand) string { return initial(pick(r, firstNames)) + "." + pick(r, lastNames) }},
	{4, func(r *rand.Rand) string { return pick(r, firstNames) + initial(pick(r, lastNames)) }},
	{2, func(r *rand.Rand) string { return pick(r, firstNames) + "." + initial(pick(r, lastNames)) }},
	{3, func(r *rand.Rand) string { return pick(r, lastNames) + "." + pick(r, firstNames) }},
	{3, func(r *rand.Rand) string { return pick(r, lastNames) + initial(pick(r, firstNames)) }},
	{2, func(r *rand.Rand) string {
		return pick(r, firstNames) + "." + initial(pick(r, firstNames)) + "." + pick(r, lastNames)
	}},
	{1, func(r *rand.Rand) string { return pick(r, firstNames) + "-" + pick(r, lastNames) }},
	{12, func(r *rand.Rand) string { return pick(r, firstNames) + digits(r) }},
	{10, func(r *rand.Rand) string { return pick(r, firstNames) + pick(r, lastNames) + digits(r) }},
	{6, func(r *rand.Rand) string { return pick(r, firstNames) + "." + pick(r, lastNames) + digits(r) }},
	{5, func(r *rand.Rand) string { return initial(pick(r, firstNames)) + pick(r, lastNames) + digits(r) }},
	{2, func(r *rand.Rand) string { return pick(r, firstNames) + "_" + pick(r, lastNames) + digits(r) }},
}

var normalPatterns = []pattern{
	{14, func(r *rand.Rand) string { return title(pick(r, adjectives)) + title(pick(r, nouns)) }},
	{8, func(r *rand.Rand) string { return title(pick(r, nouns)) + title(pick(r, nouns)) }},
	{5, func(r *rand.Rand) string { return pick(r, adjectives) + pick(r, nouns) }},
	{4, func(r *rand.Rand) string { return pick(r, adjectives) + pick(r, []string{"_", "."}) + pick(r, nouns) }},
	{12, func(r *rand.Rand) string { return pick(r, firstNames) + separator(r) + digits(r) }},
	{8, func(r *rand.Rand) string { return pick(r, namePrefixes) + pick(r, firstNames) }},
	{8, func(r *rand.Rand) string { return pick(r, firstNames) + separator(r) + pick(r, hobbies) }},
	{4, func(r *rand.Rand) string { return title(pick(r, firstNames)) + title(pick(r, nouns)) }},
	{4, func(r *rand.Rand) string {
		if r.IntN(2) == 0 {
			return "The" + title(pick(r, nouns))
		}
		return "The" + title(pick(r, adjectives)) + title(pick(r, nouns))
	}},
	{3, func(r *rand.Rand) string { return "xX" + title(pick(r, adjectives)) + title(pick(r, nouns)) + "Xx" }},
	{10, func(r *rand.Rand) string {
		if r.IntN(2) == 0 {
			return title(pick(r, nouns)) + title(pick(r, nouns)) + separator(r) + digits(r)
		}
		return title(pick(r, adjectives)) + title(pick(r, nouns)) + digits(r)
	}},
	{3, func(r *rand.Rand) string { return title(pick(r, nouns)) + pick(r, channelSuffixes) }},
	{3, func(r *rand.Rand) string { return pick(r, adjectives) + pick(r, firstNames) }},
	{3, func(r *rand.Rand) string { return pick(r, nouns) + "." + pick(r, nouns) }},
	// Leetspeak ("Fr0zenWolf") is left out: a model can't tell a digit
	// standing in for a letter from a number, so it would learn fragments
	// like "fr" and "zen" as words.
}

func patternsFor(s Style) []pattern {
	if s == Email {
		return emailPatterns
	}
	return normalPatterns
}

// Username returns one username in the given style. It always passes
// markov.Check.
func Username(r *rand.Rand, s Style) string {
	patterns := patternsFor(s)
	total := 0
	for _, p := range patterns {
		total += p.weight
	}
	for {
		x := r.IntN(total)
		for _, p := range patterns {
			if x -= p.weight; x < 0 {
				if name := p.build(r); markov.Check(name) == nil {
					return name
				}
				break
			}
		}
	}
}

// Write writes perStyle usernames in each style to w, one per line,
// alternating styles so a partial read still covers both. The same seed
// always gives the same usernames.
func Write(w io.Writer, styles []Style, perStyle int, seed uint64) error {
	r := rand.New(rand.NewPCG(seed, seed))
	bw := bufio.NewWriterSize(w, 1<<20)
	for range perStyle {
		for _, s := range styles {
			bw.WriteString(Username(r, s))
			bw.WriteByte('\n')
		}
	}
	return bw.Flush()
}

const (
	// PerStyle is how many usernames of each style Models learns: enough
	// to cover the name and word lists many times over.
	PerStyle = 100_000
	// Seed fixes the usernames Models learns, so its models never change.
	Seed  = 1
	order = 3
)

var (
	modelsOnce sync.Once
	models     []*markov.Model
)

// Models returns the built-in knowledge as one model per style, in the
// order of Styles. They are built on first use, in a fraction of a second,
// and shared; callers must not train them further.
func Models() []*markov.Model {
	modelsOnce.Do(func() {
		for _, s := range Styles {
			var buf bytes.Buffer
			if err := Write(&buf, []Style{s}, PerStyle, Seed); err != nil {
				panic(err) // writing to memory can't fail
			}
			m, err := markov.New(order)
			if err != nil {
				panic(err)
			}
			if _, err := m.LearnFrom(&buf, nil); err != nil {
				panic(err)
			}
			models = append(models, m)
		}
	})
	return models
}
