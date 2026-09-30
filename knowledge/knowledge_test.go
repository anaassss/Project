package knowledge

import (
	"bytes"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/anaassss/Project/markov"
)

func TestListsHaveNoDuplicates(t *testing.T) {
	for name, list := range map[string][]string{
		"firstNames": firstNames, "commonFirstNames": commonFirstNames,
		"commonLastNames": commonLastNames, "rareLastNames": rareLastNames,
	} {
		seen := map[string]bool{}
		for _, w := range list {
			if seen[w] {
				t.Errorf("%s lists %q twice", name, w)
			}
			seen[w] = true
		}
	}
	for _, w := range rareLastNames {
		if slices.Contains(commonLastNames, w) {
			t.Errorf("%q is both a common and a rare last name", w)
		}
	}
}

func TestUsernamesLookLikeTheKnowledge(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 1))
	for range 30000 {
		name := Username(r)
		if err := markov.Check(name); err != nil {
			t.Fatalf("username %q fails Check: %v", name, err)
		}
		if name != strings.ToLower(name) {
			t.Fatalf("username %q is not lowercase", name)
		}
		if !Unique(name) {
			t.Fatalf("username %q is not Unique", name)
		}
	}
}

// TestPatternsRarelyFail makes sure no pattern mostly builds names that
// Username would silently retry: garbage, or names that aren't Unique,
// which a pattern of common names (thomas.hughes) builds on purpose now
// and then.
func TestPatternsRarelyFail(t *testing.T) {
	r := rand.New(rand.NewPCG(4, 4))
	for i, p := range patterns {
		garbage, plain := 0, 0
		for range 2000 {
			name := p.build(r)
			switch {
			case markov.Check(name) != nil:
				garbage++
			case !Unique(name):
				plain++
			}
		}
		if garbage > 100 || plain > 500 {
			t.Errorf("pattern %d builds %d garbage and %d plain usernames in 2000", i, garbage, plain)
		}
	}
}

// TestShape checks the proportions the patterns are weighted to give.
func TestShape(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 3))
	const n = 10000
	var digits, joined, dotted, withName int
	for range n {
		name := Username(r)
		if strings.ContainsFunc(name, unicode.IsDigit) {
			digits++
		}
		if !strings.ContainsAny(name, "._-") {
			joined++
		}
		if strings.Contains(name, ".") {
			dotted++
		}
		if slices.ContainsFunc(firstNames, func(f string) bool { return len(f) > 3 && strings.Contains(name, f) }) {
			withName++
		}
	}
	for _, c := range []struct {
		what     string
		got      int
		min, max int // percent
	}{
		{"with a number", digits, 30, 55},
		{"without a separator", joined, 45, 65},
		{"with a dot", dotted, 25, 45},
		{"with a whole first name", withName, 60, 95},
	} {
		if pct := 100 * c.got / n; pct < c.min || pct > c.max {
			t.Errorf("%d%% of usernames are %s, want %d-%d%%", pct, c.what, c.min, c.max)
		}
	}
}

func TestWriteIsRepeatable(t *testing.T) {
	var a, b bytes.Buffer
	if err := Write(&a, 500, 9); err != nil {
		t.Fatal(err)
	}
	Write(&b, 500, 9)
	if a.String() != b.String() {
		t.Error("Write gave different usernames for the same seed")
	}
	if n := strings.Count(a.String(), "\n"); n != 500 {
		t.Errorf("Write wrote %d lines, want 500", n)
	}
}

func TestModel(t *testing.T) {
	m := Model()
	if m != Model() {
		t.Error("Model built twice")
	}
	if got := m.Stats().Names; got < Names*9/10 {
		t.Errorf("model learned %d usernames, want about %d", got, Names)
	}
	for word, want := range map[string]bool{"lindqvist": true, "okafor": true, "yamamoto": true, "panda": false, "gamer": false} {
		if got := m.KnowsWord(word); got != want {
			t.Errorf("model knows %q = %v, want %v", word, got, want)
		}
	}
}

func TestPlain(t *testing.T) {
	for name, want := range map[string]bool{
		"stefan": true, "emma2008": true, "john92": true, "johnkevin": true,
		"john.smith": true, "jsmith": true, "j.smith": true, "Emma_Jones": true,
		"jason4471": false, "john.smith4821": false, "stefan.hollis": false,
		"maya.thorne": false, "kevinlindqvist": false, "brandyholt": false,
		"caleb.tw": false, "emma.jones.uk": false, "jb2004": false,
	} {
		if got := Plain(name); got != want {
			t.Errorf("Plain(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestUnique(t *testing.T) {
	for name, want := range map[string]bool{
		"itsmike": false, "its.mike": false, "noahplayz": false, "xXShadowXx": false,
		"maria.io": false, "lil_zara": false, "lilzara": false, "x.luna.x": false,
		"mike.and.jess": false, "noah_gamer": false, "emmagirl": false, "johnkevin": false,
		"maya.thorne": true, "brandyholt": true, "theodore": true, "lilian4410": true,
		"itsuki.mori": true, "tariq_0912": true, "NovaTV": false, "pizzamantv": false,
	} {
		if got := Unique(name); got != want {
			t.Errorf("Unique(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestTrimJoined(t *testing.T) {
	for word, want := range map[string]string{
		"itsmike": "mike", "noahplayz": "noah", "lilzara": "zara", "mrsmith": "smith",
		"xxshadow": "shadow", "ItsMike": "mike", "lilian": "lilian", "theo": "theo", "maya": "maya",
		"mrsjones": "jones", "mrbiscuit": "biscuit", "leoboy": "leo", "coleman": "coleman",
	} {
		if got := TrimJoined(word); got != want {
			t.Errorf("TrimJoined(%q) = %q, want %q", word, got, want)
		}
	}
}
