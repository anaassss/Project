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
		"firstNames": firstNames, "lastNames": lastNames, "adjectives": adjectives,
		"nouns": nouns, "hobbies": hobbies, "namePrefixes": namePrefixes,
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
		for range 20000 {
			if name := Username(r, s); markov.Check(name) != nil {
				t.Fatalf("%s username %q fails Check: %v", s, name, markov.Check(name))
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
		if slices.ContainsFunc(lastNames, func(l string) bool { return strings.Contains(name, l) }) ||
			slices.ContainsFunc(firstNames, func(f string) bool { return strings.Contains(name, f) }) {
			withName++
		}
	}
	if withName < 990 {
		t.Errorf("only %d of 1000 email-style usernames contain a name", withName)
	}
}

func TestNormalStyleVaries(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 3))
	var camel, lower, digits int
	for range 1000 {
		name := Username(r, Normal)
		switch {
		case name == strings.ToLower(name):
			lower++
		default:
			camel++
		}
		if strings.ContainsAny(name, "0123456789") {
			digits++
		}
	}
	if camel < 300 || lower < 300 || digits < 150 {
		t.Errorf("normal style too uniform: %d CamelCase, %d lowercase, %d with digits", camel, lower, digits)
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
	if n := strings.Count(a.String(), "\n"); n != 1000 {
		t.Errorf("Write wrote %d lines, want 1000", n)
	}
}

func TestParseStyle(t *testing.T) {
	for in, want := range map[string][]Style{"email": {Email}, "Normal": {Normal}, "both": Styles} {
		if got, err := ParseStyle(in); err != nil || !slices.Equal(got, want) {
			t.Errorf("ParseStyle(%q) = %v, %v", in, got, err)
		}
	}
	if _, err := ParseStyle("gamer"); err == nil {
		t.Error("ParseStyle accepted an unknown style")
	}
}
