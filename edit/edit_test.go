package edit

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/anaassss/Project/markov"
)

var training = []string{
	"ShadowWolf", "ShadowHunter", "DarkKnight", "DarkPhoenix", "NightWolf",
	"NightHawk", "IronFist", "IronWolf42", "StormRider", "StormBreaker",
	"FrostByte", "FrostFang_", "PixelNinja", "PixelKnight", "CyberHawk",
	"CyberPunk2077", "SilentStorm", "SilentBlade", "BlazeRunner", "LunarFox",
	"MoonWalker", "GhostRider", "xXDragonXx", "EagleEye_21", "RedFox",
}

func editor(t *testing.T, opts Options) (*Editor, *markov.Model) {
	t.Helper()
	m, _ := markov.New(3)
	for _, name := range training {
		if _, err := m.Learn(name); err != nil {
			t.Fatal(err)
		}
	}
	e, err := New(m, rand.New(rand.NewPCG(1, 2)), opts)
	if err != nil {
		t.Fatal(err)
	}
	return e, m
}

func TestEditProducesGoodDistinctEdits(t *testing.T) {
	e, m := editor(t, Options{Max: 10})
	for _, name := range []string{"ShadowFox", "DarkWolf_7", "xXSniperXx", "StormHawk99"} {
		edits := e.Edit(name)
		if len(edits) == 0 {
			t.Errorf("Edit(%q) returned nothing", name)
		}
		if len(edits) > 10 {
			t.Errorf("Edit(%q) returned %d edits, want at most 10", name, len(edits))
		}
		seen := map[string]bool{}
		for _, ed := range edits {
			key := strings.ToLower(ed)
			switch {
			case key == strings.ToLower(name):
				t.Errorf("Edit(%q) returned the name itself", name)
			case seen[key]:
				t.Errorf("Edit(%q) returned %q twice", name, ed)
			case m.Knows(ed):
				t.Errorf("Edit(%q) returned learned name %q", name, ed)
			case markov.Check(ed) != nil:
				t.Errorf("Edit(%q) returned garbage %q", name, ed)
			case m.Score(ed) < e.floor:
				t.Errorf("Edit(%q) returned %q scoring below the floor", name, ed)
			}
			seen[key] = true
		}
	}
}

func TestEditUsesLearnedParts(t *testing.T) {
	e, _ := editor(t, Options{Max: 50})
	edits := e.Edit("ShadowFox")
	for _, want := range []string{"ShadowHawk", "NightFox"} {
		if !slices.Contains(edits, want) {
			t.Errorf("Edit(ShadowFox) = %v, want it to include %q", edits, want)
		}
	}
	if edits := e.Edit("StormHawk99"); !slices.Contains(edits, "StormHawk") {
		t.Errorf("Edit(StormHawk99) = %v, want the number dropped", edits)
	}
}

func TestEditAllowKnown(t *testing.T) {
	e, _ := editor(t, Options{Max: 50, AllowKnown: true})
	if edits := e.Edit("ShadowFox"); !slices.Contains(edits, "ShadowWolf") {
		t.Errorf("Edit(ShadowFox) with AllowKnown = %v, want it to include ShadowWolf", edits)
	}
}

func TestEditDeclinesWhatItDoesNotKnow(t *testing.T) {
	e, _ := editor(t, Options{Max: 10})
	if edits := e.Edit("Влад"); len(edits) != 0 {
		t.Errorf("Edit(Влад) = %v, want none from a model that has seen no Cyrillic", edits)
	}
}

func TestEditIsDeterministicForSeed(t *testing.T) {
	a, _ := editor(t, Options{Max: 10})
	b, _ := editor(t, Options{Max: 10})
	if x, y := a.Edit("DarkWolf_7"), b.Edit("DarkWolf_7"); !slices.Equal(x, y) {
		t.Errorf("same seed gave different edits:\n%v\n%v", x, y)
	}
}

func TestNewRejectsBadInput(t *testing.T) {
	empty, _ := markov.New(3)
	if _, err := New(empty, rand.New(rand.NewPCG(1, 2)), Options{Max: 5}); err == nil {
		t.Error("New accepted an empty model")
	}
	m, _ := markov.New(3)
	m.Learn("ShadowWolf")
	if _, err := New(m, rand.New(rand.NewPCG(1, 2)), Options{Max: 0}); err == nil {
		t.Error("New accepted Max 0")
	}
}

func TestSplit(t *testing.T) {
	for name, want := range map[string]string{
		"ShadowWolf":    "Shadow|Wolf",
		"xXSniperXx":    "x|X|Sniper|Xx",
		"Dark_Wolf_99":  "Dark|_|Wolf|_|99",
		"XMLParser":     "XML|Parser",
		"coolguy":       "coolguy",
		"CyberPunk2077": "Cyber|Punk|2077",
		"john.doe":      "john|.|doe",
	} {
		var got []string
		for _, seg := range split(name) {
			got = append(got, seg.text)
		}
		if strings.Join(got, "|") != want {
			t.Errorf("split(%q) = %s, want %s", name, strings.Join(got, "|"), want)
		}
	}
}

func TestMatchCase(t *testing.T) {
	for _, c := range []struct{ w, like, want string }{
		{"hawk", "Wolf", "Hawk"},
		{"hawk", "wolf", "hawk"},
		{"hawk", "WOLF", "HAWK"},
		{"hawk", "X", "Hawk"},
	} {
		if got := matchCase(c.w, c.like); got != c.want {
			t.Errorf("matchCase(%q, %q) = %q, want %q", c.w, c.like, got, c.want)
		}
	}
}

func TestDrop(t *testing.T) {
	for name, want := range map[string]string{
		"Dark_Wolf_7": "Dark_7", // index 2 is "Wolf"
		"DarkWolf99":  "Dark99",
	} {
		segs := split(name)
		i := slices.IndexFunc(segs, func(s segment) bool { return s.text == "Wolf" })
		if got := drop(segs, i); got != want {
			t.Errorf("drop(%q, Wolf) = %q, want %q", name, got, want)
		}
	}
}
