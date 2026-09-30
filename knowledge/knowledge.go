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

// The weights below follow a measured sample of real lowercase usernames:
// about 65% built from names, 20% from a word and a number, the rest from
// names with hobbies, prefixes and suffixes; half end in a number, and 55%
// have no separator, 35% a dot, 8% an underscore and 2% a hyphen.

// digits returns the numbers people add to usernames, in the proportions
// they do: mostly birth years, full or short, then random 3-4 digit
// numbers, dates, and favourites like 123 and 777.
func digits(r *rand.Rand) string {
	switch x := r.IntN(100); {
	case x < 30:
		return strconv.Itoa(year(r))
	case x < 52:
		return fmt.Sprintf("%02d", year(r)%100)
	case x < 64:
		return strconv.Itoa(100 + r.IntN(900))
	case x < 76:
		return strconv.Itoa(1000 + r.IntN(9000))
	case x < 84:
		return date(r)
	case x < 88:
		return strconv.Itoa(r.IntN(10))
	default:
		return pick(r, []string{"123", "1234", "777", "007", "100", "101", "01", "11", "22", "99", "2000", "5000"})
	}
}

// year is a birth year, most often 2000-2012.
func year(r *rand.Rand) int {
	if r.IntN(10) < 7 {
		return 2000 + r.IntN(13)
	}
	return 1975 + r.IntN(25)
}

// date is a birthday as MMDD or DDMM, like 0409.
func date(r *rand.Rand) string {
	m, d := 1+r.IntN(12), 1+r.IntN(28)
	if r.IntN(2) == 0 {
		return fmt.Sprintf("%02d%02d", m, d)
	}
	return fmt.Sprintf("%02d%02d", d, m)
}

// digitsFor returns digits short enough that a username with letters
// letters and extra other characters stays at least half letters, as the
// garbage filter requires; "" if none fit.
func digitsFor(r *rand.Rand, letters, extra int) string {
	for range 4 {
		if d := digits(r); len(d)+extra <= letters {
			return d
		}
	}
	if room := letters - extra; room > 0 {
		return strconv.Itoa(1 + r.IntN(min(99, pow10(room)-1)))
	}
	return ""
}

func pow10(n int) int {
	p := 1
	for range min(n, 9) {
		p *= 10
	}
	return p
}

// maybeDigits adds digits one time in n.
func maybeDigits(r *rand.Rand, n int) string {
	if r.IntN(n) == 0 {
		return digits(r)
	}
	return ""
}

// sep returns a separator: mostly a dot, else an underscore or hyphen.
func sep(r *rand.Rand) string {
	switch x := r.IntN(20); {
	case x < 15:
		return "."
	case x < 19:
		return "_"
	}
	return "-"
}

// separator returns what joins two parts: mostly nothing, else a separator.
func separator(r *rand.Rand) string {
	if r.IntN(10) < 6 {
		return ""
	}
	return sep(r)
}

// short cuts a surname to its first 3-5 letters, as in maria.gonz. Two
// letters would teach stubs like "ma" as words; initials cover those.
func short(r *rand.Rand, name string) string {
	if len(name) <= 4 {
		return name
	}
	return name[:3+r.IntN(min(3, len(name)-3))]
}

// initials returns one or two random initials.
func initials(r *rand.Rand) string {
	if r.IntN(3) == 0 {
		return initial(first(r)) + initial(last(r))
	}
	return initial(last(r))
}

// stretch sometimes doubles a name's last letter, as in lucasgreenn.
func stretch(r *rand.Rand, s string) string {
	if r.IntN(25) == 0 && s != "" {
		return s + s[len(s)-1:]
	}
	return s
}

var emailPatterns = []pattern{
	{12, func(r *rand.Rand) string { return first(r) + digits(r) }},
	{3, func(r *rand.Rand) string { n := first(r); return n + sep(r) + digitsFor(r, len(n), 1) }},
	{9, func(r *rand.Rand) string { return first(r) + sep(r) + last(r) + maybeDigits(r, 4) }},
	{7, func(r *rand.Rand) string { return first(r) + sep(r) + short(r, last(r)) + maybeDigits(r, 3) }},
	{6, func(r *rand.Rand) string { return first(r) + sep(r) + initials(r) + maybeDigits(r, 3) }},
	{2, func(r *rand.Rand) string { return first(r) + "." + initial(first(r)) + "." + initial(last(r)) }},
	{6, func(r *rand.Rand) string { return first(r) + sep(r) + first(r) + maybeDigits(r, 4) }},
	{7, func(r *rand.Rand) string { return stretch(r, first(r)+last(r)) + maybeDigits(r, 2) }},
	{8, func(r *rand.Rand) string { return stretch(r, first(r)+short(r, last(r))) + maybeDigits(r, 2) }},
	{5, func(r *rand.Rand) string { return first(r) + initials(r) + digits(r) }},
	{5, func(r *rand.Rand) string { return initial(first(r)) + last(r) + digits(r) }},
	{2, func(r *rand.Rand) string { return initial(first(r)) + "." + last(r) + maybeDigits(r, 2) }},
	{3, func(r *rand.Rand) string {
		n := initial(first(r)) + short(r, last(r))
		return n + digitsFor(r, len(n), 0)
	}},
	{2, func(r *rand.Rand) string { return initial(first(r)) + initial(first(r)) + last(r) + digits(r) }},
	{3, func(r *rand.Rand) string { return last(r) + digits(r) }},
	{1, func(r *rand.Rand) string { return last(r) + digits(r) + initial(first(r)) }},
	{1, func(r *rand.Rand) string { return last(r) + sep(r) + first(r) }},
}

var gamingPatterns = []pattern{
	{25, func(r *rand.Rand) string {
		w := noun(r)
		if r.IntN(3) == 0 {
			w = adj(r)
		}
		return w + strconv.Itoa(100+r.IntN(9900))
	}},
	{18, func(r *rand.Rand) string {
		if r.IntN(2) == 0 {
			return adj(r) + noun(r) + maybeDigits(r, 2)
		}
		return noun(r) + noun(r) + maybeDigits(r, 2)
	}},
	{12, func(r *rand.Rand) string { return stretch(r, noun(r)+pick(r, gamingSuffixes)) + maybeDigits(r, 2) }},
	{8, func(r *rand.Rand) string { return pick(r, gamingPrefixes) + noun(r) + maybeDigits(r, 2) }},
	{4, func(r *rand.Rand) string { return noun(r) + sep(r) + noun(r) + maybeDigits(r, 4) }},
	{3, func(r *rand.Rand) string { return adj(r) + sep(r) + noun(r) + maybeDigits(r, 3) }},
	{4, func(r *rand.Rand) string { n := noun(r); return n + sep(r) + digitsFor(r, len(n), 1) }},
	{2, func(r *rand.Rand) string { return noun(r) + "." + pick(r, []string{"exe", "png", "io"}) }},
	// A minority in CamelCase, so CamelCase names have knowledge to draw on.
	{6, func(r *rand.Rand) string { return title(adj(r)) + title(noun(r)) + maybeDigits(r, 3) }},
	{3, func(r *rand.Rand) string { return title(noun(r)) + title(noun(r)) + maybeDigits(r, 3) }},
	{1, func(r *rand.Rand) string { return "xX" + title(adj(r)) + title(noun(r)) + "Xx" }},
	{1, func(r *rand.Rand) string { return "The" + title(noun(r)) }},
	{1, func(r *rand.Rand) string { return title(pick(r, gamingTitles)) + title(noun(r)) }},
	{1, func(r *rand.Rand) string { return title(noun(r)) + pick(r, channelSuffixes) }},
	// Leetspeak ("Fr0zenWolf") is left out: a model can't tell a digit
	// standing in for a letter from a number, so it would learn fragments
	// like "fr" and "zen" as words.
}

var socialPatterns = []pattern{
	{14, func(r *rand.Rand) string { return first(r) + sep(r) + pick(r, hobbies) + maybeDigits(r, 5) }},
	{4, func(r *rand.Rand) string { return pick(r, hobbies) + sep(r) + first(r) }},
	{10, func(r *rand.Rand) string {
		return pick(r, socialPrefixes) + pick(r, []string{"", ".", "_"}) + first(r) + maybeDigits(r, 3)
	}},
	{4, func(r *rand.Rand) string {
		return first(r) + "s" + pick(r, []string{"mama", "mommy", "mom", "dad", "daddy"}) + fmt.Sprintf("%02d", year(r)%100)
	}},
	{4, func(r *rand.Rand) string {
		if r.IntN(2) == 0 {
			return first(r) + sep(r) + pick(r, relationWords)
		}
		return pick(r, relationWords) + sep(r) + first(r)
	}},
	{8, func(r *rand.Rand) string { return stretch(r, first(r)+pick(r, nameSuffixes)) + maybeDigits(r, 2) }},
	{8, func(r *rand.Rand) string { return first(r) + sep(r) + word(r) + maybeDigits(r, 4) }},
	{4, func(r *rand.Rand) string { return word(r) + sep(r) + first(r) }},
	{6, func(r *rand.Rand) string {
		w := word(r)
		if r.IntN(2) == 0 {
			w = noun(r) // oliviamango, sampanda
		}
		return first(r) + w + maybeDigits(r, 2)
	}},
	{5, func(r *rand.Rand) string { return first(r) + "." + pick(r, webSuffixes) }},
	{5, func(r *rand.Rand) string {
		n := first(r) + maybeDigits(r, 2)
		return n + pick(r, []string{"xx", "xd", "x"})
	}},
	{5, func(r *rand.Rand) string {
		if r.IntN(2) == 0 {
			return first(r) + sep(r) + pick(r, funWords)
		}
		return pick(r, funWords) + sep(r) + first(r)
	}},
	{3, func(r *rand.Rand) string { return word(r) + pick(r, []string{".", "_"}) + word(r) }},
	{2, func(r *rand.Rand) string { return first(r) + pick(r, []string{"and", ".and."}) + first(r) }},
	{2, func(r *rand.Rand) string {
		return pick(r, []string{"ilike", "iheart", "ilove"}) + pick(r, []string{"pizza", "tacos", "milk", "cats", "dogs", "cookies", "pasta", "sushi", "music", "games"}) + maybeDigits(r, 2)
	}},
	{1, func(r *rand.Rand) string { return "_" + first(r) + "_" }},
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
