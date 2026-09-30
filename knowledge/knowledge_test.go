package knowledge

import (
	"bytes"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/anaassss/Project/markov"
)

func TestListsHaveNoDuplicates(t *testing.T) {
	for name, list := range map[string][]string{
		"firstNames": firstNames, "lastNames": lastNames, "gamingAdjectives": gamingAdjectives,
		"gamingNouns": gamingNouns, "gamingTitles": gamingTitles, "channelSuffixes": channelSuffixes,
		"socialWords": socialWords, "hobbies": hobbies, "socialPrefixes": socialPrefixes,
	} {
		seen := map[string]bool{}
		for _, w := range list {
			if seen[w] {
				t.Errorf("%s lists %q twice", name, w)
			}
			seen[w] = true
		}
	}
}

func TestUsernamesPassCheck(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 1))
	for _, s := range Styles {
		for range 30000 {
			if name := Username(r, s); markov.Check(name) != nil {
				t.Fatalf("%s username %q fails Check: %v", s, name, markov.Check(name))
			}
		}
	}
}

// TestPatternsRarelyFail makes sure no pattern mostly builds garbage that
// Username would silently retry.
func TestPatternsRarelyFail(t *testing.T) {
	r := rand.New(rand.NewPCG(4, 4))
	for _, s := range Styles {
		for i, p := range patternsFor(s) {
			failed := 0
			for range 2000 {
				if markov.Check(p.build(r)) != nil {
					failed++
				}
			}
			if failed > 100 {
				t.Errorf("%s pattern %d fails the garbage filter %d times in 2000", s, i, failed)
			}
		}
	}
}

func TestEmailStyleIsBuiltFromNames(t *testing.T) {
	r := rand.New(rand.NewPCG(2, 2))
	for range 1000 {
		name := Username(r, Email)
		if name != strings.ToLower(name) {
			t.Errorf("email-style %q is not lowercase", name)
		}
		if !slices.ContainsFunc(firstNames, func(f string) bool { return strings.Contains(name, f) }) &&
			!slices.ContainsFunc(lastNames, func(l string) bool { return strings.Contains(name, l) }) {
			t.Errorf("email-style %q contains no name", name)
		}
	}
}

func TestStylesDiffer(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 3))
	count := func(s Style, f func(string) bool) int {
		n := 0
		for range 1000 {
			if f(Username(r, s)) {
				n++
			}
		}
		return n
	}
	hasUpper := func(n string) bool { return n != strings.ToLower(n) }
	if n := count(Gaming, hasUpper); n < 700 {
		t.Errorf("only %d of 1000 gaming names use capitals", n)
	}
	if n := count(Social, hasUpper); n > 50 {
		t.Errorf("%d of 1000 social names use capitals", n)
	}
	if n := count(Gaming, func(n string) bool { return strings.ContainsAny(n, "0123456789") }); n < 100 {
		t.Errorf("only %d of 1000 gaming names have numbers", n)
	}
}

func TestWrite(t *testing.T) {
	var a, b bytes.Buffer
	if err := Write(&a, Styles, 500, 9); err != nil {
		t.Fatal(err)
	}
	Write(&b, Styles, 500, 9)
	if a.String() != b.String() {
		t.Error("Write is not deterministic for a seed")
	}
	if n := strings.Count(a.String(), "\n"); n != 500*len(Styles) {
		t.Errorf("Write wrote %d lines, want %d", n, 500*len(Styles))
	}
}

func TestModels(t *testing.T) {
	ms := Models()
	if len(ms) != len(Styles) {
		t.Fatalf("got %d models, want %d", len(ms), len(Styles))
	}
	if again := Models(); again[0] != ms[0] {
		t.Error("Models rebuilt its models")
	}
	email, gaming, social := ms[0], ms[1], ms[2]
	for _, c := range []struct {
		m    *markov.Model
		word string
		want bool
	}{
		{email, "smith", true}, {email, "okafor", true}, {email, "yamamoto", true}, {email, "panda", false},
		{gaming, "panda", true}, {gaming, "valkyrie", true}, {gaming, "garcia", false},
		{social, "lavender", true}, {social, "bakes", true}, {social, "valkyrie", false},
	} {
		if got := c.m.KnowsWord(c.word); got != c.want {
			t.Errorf("%s model knows %q = %v, want %v", Styles[slices.Index(ms, c.m)], c.word, got, c.want)
		}
	}
}
