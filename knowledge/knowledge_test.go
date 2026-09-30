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
	withName := 0
	for range 1000 {
		name := Username(r, Email)
		if name != strings.ToLower(name) {
			t.Errorf("email-style %q is not lowercase", name)
		}
		// Most contain a whole name; the rest shorten one (maria.gonz,
		// dkowal2007).
		if slices.ContainsFunc(firstNames, func(f string) bool { return len(f) > 2 && strings.Contains(name, f) }) ||
			slices.ContainsFunc(lastNames, func(l string) bool { return len(l) > 2 && strings.Contains(name, l) }) {
			withName++
		}
	}
	if withName < 750 {
		t.Errorf("only %d of 1000 email-style usernames contain a whole name", withName)
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
	hasDigit := func(n string) bool { return strings.ContainsAny(n, "0123456789") }
	// Real handles are mostly lowercase; Gaming keeps a CamelCase minority.
	if n := count(Gaming, hasUpper); n < 50 || n > 250 {
		t.Errorf("%d of 1000 gaming names use capitals, want a minority", n)
	}
	for _, s := range []Style{Email, Social} {
		if n := count(s, hasUpper); n > 0 {
			t.Errorf("%d of 1000 %s names use capitals", n, s)
		}
	}
	if n := count(Gaming, hasDigit); n < 400 {
		t.Errorf("only %d of 1000 gaming names have numbers", n)
	}
	if n := count(Email, hasDigit); n < 400 || n > 800 {
		t.Errorf("%d of 1000 email-style names have numbers, want about half or more", n)
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
