// Package knowledge is usergen's built-in knowledge of how real email-style
// usernames are made: the kind people end up with once the plain ones
// (john.smith, johnkevin) are long taken. Rather than a list of anyone's
// actual accounts, it generates usernames from the patterns such names
// follow, using about 1,250 first names and 1,500 last names from many
// countries:
//
//   - nicknames and blends of names, often with a number: brandyholt,
//     dariwex2006, oskarzo, kristofer91
//   - a first name and a less common surname, joined or with a separator:
//     tobiaslindqvist, maya.thorne, ivan7okafor
//   - a first name with a 3-4 digit number or a date: rosalind4318,
//     tariq_0912
//   - a first name with initials, a short surname, another first name, or
//     a common surname: yusuf.tk, greta.hol, amir.helena, ines.carter
//
// Model returns this knowledge as a ready-made model. Unique and Plain
// tell whether a username looks like it: real, but not so plain that it
// must be taken already.
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

	"github.com/anaassss/Project/markov"
)

// pattern builds one username. Patterns are weighted by how common each
// shape is among real, still-free usernames.
type pattern struct {
	weight int
	build  func(r *rand.Rand) string
}

func pick(r *rand.Rand, list []string) string { return list[r.IntN(len(list))] }

// first is a first name, more often one common in English- and
// Spanish-speaking countries, as among the sample the patterns follow.
func first(r *rand.Rand) string {
	if r.IntN(10) < 6 {
		return pick(r, americasFirstNames)
	}
	return pick(r, firstNames)
}

// surname is a less common surname most of the time, since a common one
// makes a name more likely taken.
func surname(r *rand.Rand) string {
	if r.IntN(10) < 3 {
		return pick(r, commonLastNames)
	}
	return pick(r, rareLastNames)
}

func initial(r *rand.Rand) string { return pick(r, firstNames)[:1] }

// cut returns name's first n letters, or all of it if shorter.
func cut(name string, n int) string { return name[:min(n, len(name))] }

func isVowel(c byte) bool { return strings.IndexByte("aeiouy", c) >= 0 }

// nick makes a nickname of a first name of 4+ letters: cut short with an
// ending (brandon → brandy, valeria → valie) or with one added (tobias →
// tobiasski).
func nick(r *rand.Rand) string {
	name := first(r)
	for len(name) < 4 {
		name = first(r)
	}
	if len(name) >= 5 && r.IntN(3) > 0 || isVowel(name[len(name)-1]) {
		k := min(3+r.IntN(2), len(name)-1)
		for k > 2 && isVowel(name[k-1]) {
			k--
		}
		return name[:k] + pick(r, []string{"y", "ie", "o", "i", "ee"})
	}
	return name + pick(r, []string{"y", "o", "ski", "zo", "ie"})
}

// alone returns name, with a number (years, two and three as for number)
// unless it is long enough to be free without one, and then only mostly.
// The number is short enough to keep the name at least half letters.
func alone(r *rand.Rand, name string, years, two, three int) string {
	if len(name) >= 8 && r.IntN(10) >= 7 {
		return name
	}
	for {
		if n := number(r, years, two, three); len(n) <= len(name) {
			return name + n
		}
	}
}

// respell spells a name another way people do (kristian → krystian, carl →
// karl, sophie → sofie), or doubles its last letter (mateo → mateoo).
func respell(r *rand.Rand, name string) string {
	switch r.IntN(4) {
	case 0:
		if i := strings.IndexByte(name[1:], 'i'); i >= 0 {
			return name[:i+1] + "y" + name[i+2:]
		}
	case 1:
		if name[0] == 'c' {
			return "k" + name[1:]
		}
	case 2:
		if i := strings.Index(name, "ph"); i >= 0 {
			return name[:i] + "f" + name[i+2:]
		}
	case 3:
		if i := strings.IndexByte(name[1:], 'y'); i >= 0 {
			return name[:i+1] + "i" + name[i+2:]
		}
	}
	return name + name[len(name)-1:]
}

// blend runs a cut-down first name and surname together, as people do when
// the whole names are taken: dari + wex, tobias + li.
func blend(r *rand.Rand) string {
	f, l := first(r), surname(r)
	switch r.IntN(3) {
	case 0:
		return cut(f, 3+r.IntN(2)) + cut(l, 3)
	case 1:
		return f + cut(l, 2+r.IntN(2))
	}
	return cut(f, 3) + cut(l, 4+r.IntN(2))
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

// number returns a number as people add them to names. years, two and
// three are the percentages of birth years and of two- and three-digit
// numbers; the rest have four digits, half of them a date like 0912.
func number(r *rand.Rand, years, two, three int) string {
	switch x := r.IntN(100); {
	case x < years:
		return strconv.Itoa(year(r))
	case x < years+two:
		if r.IntN(3) > 0 {
			return fmt.Sprintf("%02d", year(r)%100)
		}
		return fmt.Sprintf("%02d", r.IntN(100))
	case x < years+two+three:
		if r.IntN(10) == 0 {
			return pick(r, []string{"123", "007", "777", "100", "101", "321"})
		}
		return strconv.Itoa(100 + r.IntN(900))
	case r.IntN(2) == 0:
		return date(r)
	}
	for {
		if n := strconv.Itoa(1000 + r.IntN(9000)); !isYear(n) {
			return n
		}
	}
}

// year is a birth year, most often 2000-2012.
func year(r *rand.Rand) int {
	if r.IntN(10) < 7 {
		return 2000 + r.IntN(13)
	}
	return 1975 + r.IntN(25)
}

// date is a birthday as MMDD or DDMM.
func date(r *rand.Rand) string {
	m, d := 1+r.IntN(12), 1+r.IntN(28)
	if r.IntN(2) == 0 {
		return fmt.Sprintf("%02d%02d", m, d)
	}
	return fmt.Sprintf("%02d%02d", d, m)
}

func isYear(n string) bool {
	return len(n) == 4 && (strings.HasPrefix(n, "19") || strings.HasPrefix(n, "20"))
}

// sometimes returns a number (years, two and three as for number) in pct
// percent of calls, and "" otherwise.
func sometimes(r *rand.Rand, pct, years, two, three int) string {
	if r.IntN(100) < pct {
		return number(r, years, two, three)
	}
	return ""
}

// The patterns and their weights follow a measured sample of real,
// still-free email-style usernames, keeping every shape that made up at
// least 5% of it. Shapes rarer than that, and prefixes and suffixes (its,
// lil, xX…Xx, playz, mom, .io), are left out.
var patterns = []pattern{
	// Nicknames and blends of names, mostly with a number.
	{9, func(r *rand.Rand) string { return alone(r, blend(r), 28, 23, 27) }},
	{7, func(r *rand.Rand) string { return alone(r, nick(r), 28, 23, 27) }},
	{5, func(r *rand.Rand) string { return nick(r) + pick(r, rareLastNames) + sometimes(r, 20, 28, 23, 27) }},
	{3, func(r *rand.Rand) string { return alone(r, respell(r, first(r)), 28, 23, 27) }},

	// A first name and a less common surname, run together.
	{17, func(r *rand.Rand) string {
		f := first(r)
		switch x := r.IntN(20); {
		case x < 2:
			return f + strconv.Itoa(r.IntN(10)) + pick(r, rareLastNames)
		case x < 4:
			return f + initial(r) + pick(r, rareLastNames) + sometimes(r, 30, 26, 32, 32)
		case x < 7:
			return f + first(r) + sometimes(r, 40, 26, 32, 32)
		}
		name := f + pick(r, rareLastNames)
		if r.IntN(12) == 0 {
			name = respell(r, name)
		}
		return name + sometimes(r, 40, 26, 32, 32)
	}},

	// ... or with a separator, rarely with a number.
	{17, func(r *rand.Rand) string {
		l := pick(r, rareLastNames)
		switch x := r.IntN(20); {
		case x < 3:
			l = cut(l, 3+r.IntN(3))
		case x < 5:
			if base := strings.TrimRight(cut(l, 5), "aeiouy"); len(base) >= 3 {
				l = base + pick(r, []string{"y", "ie"}) // smithy
			}
		}
		return first(r) + sep(r) + l + sometimes(r, 7, 10, 80, 5)
	}},

	// A first name and a number, mostly three or four digits, short enough
	// to keep the name at least half letters.
	{9, func(r *rand.Rand) string {
		f, s := first(r), ""
		if len(f) > 3 && r.IntN(4) == 0 {
			s = sep(r)
		}
		for {
			if n := number(r, 10, 14, 28); len(s)+len(n) <= len(f) {
				return f + s + n
			}
		}
	}},

	// A first name and initials or a short surname.
	{9, func(r *rand.Rand) string {
		var tail string
		switch x := r.IntN(10); {
		case x < 5:
			tail = initial(r) + initial(r)
		case x < 8:
			tail = cut(surname(r), 3)
		case x < 9:
			tail = initial(r) + "." + initial(r)
		default:
			tail = initial(r) + initial(r) + initial(r)
		}
		return first(r) + sep(r) + tail + sometimes(r, 15, 20, 60, 10)
	}},

	// Two first names.
	{8, func(r *rand.Rand) string { return first(r) + sep(r) + first(r) + sometimes(r, 15, 20, 60, 10) }},

	// A first name and a common surname.
	{8, func(r *rand.Rand) string {
		return first(r) + sep(r) + pick(r, commonLastNames) + sometimes(r, 30, 20, 60, 20)
	}},
}

// Username returns one username in this style. It always passes
// markov.Check and is Unique.
func Username(r *rand.Rand) string {
	total := 0
	for _, p := range patterns {
		total += p.weight
	}
	for {
		x := r.IntN(total)
		for _, p := range patterns {
			if x -= p.weight; x < 0 {
				if name := p.build(r); markov.Check(name) == nil && Unique(name) && !tripled(name) {
					return name
				}
				break
			}
		}
	}
}

// tripled reports whether s has a character three times in a row, as
// joining names sometimes gives (ell + lars).
func tripled(s string) bool {
	for i := 2; i < len(s); i++ {
		if s[i] == s[i-1] && s[i] == s[i-2] {
			return true
		}
	}
	return false
}

// Write writes n usernames to w, one per line. The same seed always gives
// the same usernames.
func Write(w io.Writer, n int, seed uint64) error {
	r := rand.New(rand.NewPCG(seed, seed))
	bw := bufio.NewWriterSize(w, 1<<20)
	for range n {
		bw.WriteString(Username(r))
		bw.WriteByte('\n')
	}
	return bw.Flush()
}

const (
	// Names is how many usernames Model learns: enough to cover the name
	// lists many times over.
	Names = 200_000
	// Seed fixes the usernames Model learns, so its model never changes.
	Seed  = 1
	order = 3
)

var (
	modelOnce sync.Once
	model     *markov.Model
)

// Model returns the built-in knowledge as a model. It is built on first
// use, in a fraction of a second, and shared; callers must not train it
// further.
func Model() *markov.Model {
	modelOnce.Do(func() {
		var buf bytes.Buffer
		if err := Write(&buf, Names, Seed); err != nil {
			panic(err) // writing to memory can't fail
		}
		m, err := markov.New(order)
		if err != nil {
			panic(err)
		}
		if _, err := m.LearnFrom(&buf, nil); err != nil {
			panic(err)
		}
		model = m
	})
	return model
}
