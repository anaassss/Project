// Package knowledge is usergen's built-in knowledge of how real usernames
// are made. Rather than a list of anyone's actual accounts, it generates
// usernames from the patterns people use, combining about 1,250 first names
// and 1,200 last names from many countries and several hundred words the way
// real accounts do:
//
//   - Email-style: john.smith, jsmith, smithj, j.smith, john.m.smith,
//     johnmsmith, maria.garcia-lopez, john.smith92, jsmith1987
//   - Gaming: SilentWolf, xXDragonSlayerXx, TheNightOwl, SirWaffle,
//     Vortex.exe, dark_knight, NovaTTV, ShadowHunter99
//   - Social: itsmike, sarahbakes, mike_92, lunar.dreams, _lily_, x.luna.x,
//     mike.and.jess, sarah-dev, peach.jpg
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
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/anaassss/Project/markov"
)

// Style is a kind of username.
type Style int

const (
	Email  Style = iota // name-based, like email addresses: john.smith92
	Gaming              // gamer tags: SilentWolf, xXDragonSlayerXx
	Social              // social media handles: itsmike, lunar.dreams
)

// Styles lists every style.
var Styles = []Style{Email, Gaming, Social}

func (s Style) String() string {
	switch s {
	case Email:
		return "Email-style"
	case Gaming:
		return "Gaming"
	}
	return "Social"
}

// pattern builds one username. Patterns are weighted by roughly how common
// each shape is among real accounts of the style.
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

// Shorthands for the lists.
func first(r *rand.Rand) string { return pick(r, firstNames) }
func last(r *rand.Rand) string  { return pick(r, lastNames) }
func adj(r *rand.Rand) string   { return pick(r, gamingAdjectives) }
func noun(r *rand.Rand) string  { return pick(r, gamingNouns) }
func word(r *rand.Rand) string  { return pick(r, socialWords) }

// digits returns the numbers people add to usernames: mostly birth years,
// in full or short, else small or favourite numbers.
func digits(r *rand.Rand) string {
	switch x := r.IntN(100); {
	case x < 38:
		return fmt.Sprintf("%02d", (70+r.IntN(43))%100) // 70..99, 00..12
	case x < 66:
		return strconv.Itoa(1965 + r.IntN(48)) // 1965..2012
	case x < 88:
		return strconv.Itoa(1 + r.IntN(99))
	default:
		return pick(r, []string{"123", "007", "01", "02", "11", "13", "21", "22", "23", "77", "99", "101", "321", "777", "2000", "2020", "2024"})
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
	{22, func(r *rand.Rand) string { return first(r) + "." + last(r) }},
	{10, func(r *rand.Rand) string { return first(r) + last(r) }},
	{5, func(r *rand.Rand) string { return first(r) + "_" + last(r) }},
	{12, func(r *rand.Rand) string { return initial(first(r)) + last(r) }},
	{5, func(r *rand.Rand) string { return initial(first(r)) + "." + last(r) }},
	{4, func(r *rand.Rand) string { return first(r) + initial(last(r)) }},
	{2, func(r *rand.Rand) string { return first(r) + "." + initial(last(r)) }},
	{3, func(r *rand.Rand) string { return last(r) + "." + first(r) }},
	{3, func(r *rand.Rand) string { return last(r) + initial(first(r)) }},
	{2, func(r *rand.Rand) string { return last(r) + first(r) }},
	{2, func(r *rand.Rand) string { return first(r) + "." + initial(first(r)) + "." + last(r) }},
	{2, func(r *rand.Rand) string { return first(r) + initial(first(r)) + last(r) }},
	{1, func(r *rand.Rand) string { return initial(first(r)) + "." + initial(first(r)) + "." + last(r) }},
	{1, func(r *rand.Rand) string { return first(r) + "-" + last(r) }},
	{1, func(r *rand.Rand) string { return first(r) + "." + last(r) + "-" + last(r) }},
	{12, func(r *rand.Rand) string { return first(r) + digits(r) }},
	{10, func(r *rand.Rand) string { return first(r) + last(r) + digits(r) }},
	{6, func(r *rand.Rand) string { return first(r) + "." + last(r) + digits(r) }},
	{5, func(r *rand.Rand) string { return initial(first(r)) + last(r) + digits(r) }},
	{2, func(r *rand.Rand) string { return first(r) + "_" + last(r) + digits(r) }},
	{2, func(r *rand.Rand) string { return first(r) + initial(last(r)) + digits(r) }},
}

var gamingPatterns = []pattern{
	{14, func(r *rand.Rand) string { return title(adj(r)) + title(noun(r)) }},
	{8, func(r *rand.Rand) string { return title(noun(r)) + title(noun(r)) }},
	{10, func(r *rand.Rand) string {
		if r.IntN(2) == 0 {
			return title(noun(r)) + title(noun(r)) + separator(r) + digits(r)
		}
		return title(adj(r)) + title(noun(r)) + digits(r)
	}},
	{5, func(r *rand.Rand) string {
		if r.IntN(3) == 0 {
			return "xX" + title(noun(r)) + "Xx"
		}
		return "xX" + title(adj(r)) + title(noun(r)) + "Xx"
	}},
	{5, func(r *rand.Rand) string {
		if r.IntN(2) == 0 {
			return "The" + title(noun(r))
		}
		return "The" + title(adj(r)) + title(noun(r))
	}},
	{4, func(r *rand.Rand) string { return title(noun(r)) + pick(r, channelSuffixes) }},
	{5, func(r *rand.Rand) string { return title(pick(r, gamingTitles)) + title(noun(r)) }},
	{2, func(r *rand.Rand) string { return title(pick(r, gamingTitles)) + "_" + title(adj(r)) }},
	{4, func(r *rand.Rand) string { return adj(r) + "_" + noun(r) }},
	{4, func(r *rand.Rand) string { return adj(r) + noun(r) }},
	{3, func(r *rand.Rand) string { return adj(r) + "_" + noun(r) + digits(r) }},
	{2, func(r *rand.Rand) string { return title(noun(r)) + ".exe" }},
	{2, func(r *rand.Rand) string { return "ii" + title(noun(r)) }},
	{2, func(r *rand.Rand) string { return "NotA" + title(noun(r)) }},
	{2, func(r *rand.Rand) string { return noun(r) + "." + noun(r) }},
	// Leetspeak ("Fr0zenWolf") is left out: a model can't tell a digit
	// standing in for a letter from a number, so it would learn fragments
	// like "fr" and "zen" as words.
}

var socialPatterns = []pattern{
	{10, func(r *rand.Rand) string { return pick(r, socialPrefixes) + first(r) }},
	{10, func(r *rand.Rand) string { return first(r) + separator(r) + pick(r, hobbies) }},
	{10, func(r *rand.Rand) string { return first(r) + separator(r) + digits(r) }},
	{8, func(r *rand.Rand) string { return word(r) + pick(r, []string{".", "_"}) + word(r) }},
	{5, func(r *rand.Rand) string { return word(r) + word(r) }},
	{6, func(r *rand.Rand) string {
		switch n := first(r); r.IntN(4) {
		case 0:
			return "_" + n + "_"
		case 1:
			return "x." + n + ".x"
		case 2:
			return n + ".x"
		default:
			return n + ".xo"
		}
	}},
	{3, func(r *rand.Rand) string {
		return pick(r, []string{"_", "x."}) + word(r) + pick(r, []string{"_", ".x"})
	}},
	{5, func(r *rand.Rand) string { return word(r) + first(r) }},
	{5, func(r *rand.Rand) string { return first(r) + pick(r, []string{"", ".", "_"}) + word(r) }},
	{3, func(r *rand.Rand) string { return first(r) + ".and." + first(r) }},
	{3, func(r *rand.Rand) string {
		return word(r) + pick(r, []string{".jpg", "vibes", ".vibes", "_vibes", ".png"})
	}},
	{4, func(r *rand.Rand) string { return first(r) + initial(last(r)) + digits(r) }},
	{4, func(r *rand.Rand) string {
		switch n := first(r); r.IntN(3) {
		case 0:
			return n + "-dev"
		case 1:
			return "dev." + n
		default:
			return n + "-" + pick(r, hobbies)
		}
	}},
}

func patternsFor(s Style) []pattern {
	switch s {
	case Email:
		return emailPatterns
	case Gaming:
		return gamingPatterns
	}
	return socialPatterns
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
