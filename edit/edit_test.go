package edit

import (
	"bytes"
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

func model(t *testing.T) *markov.Model {
	t.Helper()
	m, _ := markov.New(3)
	for _, name := range training {
		if _, err := m.Learn(name); err != nil {
			t.Fatal(err)
		}
	}
	return m
}

func editor(t *testing.T, opts Options) *Editor {
	t.Helper()
	e, err := New(model(t), opts)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func rng() *rand.Rand { return rand.New(rand.NewPCG(1, 2)) }

func TestEditProducesGoodDistinctEdits(t *testing.T) {
	e := editor(t, Options{Max: 10})
	for _, name := range []string{"ShadowFox", "DarkWolf_7", "xXSniperXx", "StormHawk99"} {
		edits := e.Edit(rng(), name, nil)
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
			case e.m.Knows(ed):
				t.Errorf("Edit(%q) returned learned name %q", name, ed)
			case markov.Check(ed) != nil:
				t.Errorf("Edit(%q) returned garbage %q", name, ed)
			}
			seen[key] = true
		}
	}
}

func TestEditAppends(t *testing.T) {
	e := editor(t, Options{Max: 3})
	dst := e.Edit(rng(), "ShadowFox", []string{"keep"})
	if dst[0] != "keep" || len(dst) < 2 || len(dst) > 4 {
		t.Errorf("Edit appended wrongly: %v", dst)
	}
}

func TestEditUsesLearnedParts(t *testing.T) {
	e := editor(t, Options{Max: 40})
	edits := e.Edit(rng(), "ShadowFox", nil)
	swapped := slices.ContainsFunc(edits, func(ed string) bool {
		rest, ok := strings.CutPrefix(ed, "Shadow")
		return ok && slices.Contains(e.words, strings.ToLower(rest))
	})
	if !swapped {
		t.Errorf("Edit(ShadowFox) = %v, want a learned word swapped in", edits)
	}
	if edits := e.Edit(rng(), "StormHawk99", nil); !slices.Contains(edits, "StormHawk") {
		t.Errorf("Edit(StormHawk99) = %v, want the number dropped", edits)
	}
}

func TestEditAllowKnown(t *testing.T) {
	strict := editor(t, Options{Max: 40})
	loose := editor(t, Options{Max: 40, AllowKnown: true})
	knownIn := func(e *Editor) bool {
		return slices.ContainsFunc(e.Edit(rng(), "ShadowFox", nil), e.m.Knows)
	}
	if knownIn(strict) {
		t.Error("learned names returned without AllowKnown")
	}
	if !knownIn(loose) {
		t.Error("no learned names returned with AllowKnown")
	}
}

func TestEditDeclinesWhatItDoesNotKnow(t *testing.T) {
	e := editor(t, Options{Max: 10})
	if edits := e.Edit(rng(), "Влад", nil); len(edits) != 0 {
		t.Errorf("Edit(Влад) = %v, want none from a model that has seen no Cyrillic", edits)
	}
}

func TestRun(t *testing.T) {
	e := editor(t, Options{Max: 5})
	var in strings.Builder
	in.WriteString("# comment\n\n")
	inputs := []string{"ShadowFox", "DarkWolf_7", "StormHawk99", "NightFox", "shadowfox"}
	for i := range 3000 { // several batches
		in.WriteString(inputs[i%len(inputs)] + "\n")
	}

	run := func() (string, int) {
		seen := map[uint64]struct{}{}
		for _, name := range inputs {
			seen[markov.FoldHash(name)] = struct{}{}
		}
		var out bytes.Buffer
		n, err := e.Run(strings.NewReader(in.String()), &out, seen, 7)
		if err != nil {
			t.Fatal(err)
		}
		return out.String(), n
	}
	out, n := run()
	lines := strings.Fields(out)
	if n == 0 || n != len(lines) {
		t.Fatalf("Run wrote %d lines, reported %d", len(lines), n)
	}
	seen := map[string]bool{}
	for _, l := range lines {
		key := strings.ToLower(l)
		if seen[key] {
			t.Errorf("Run wrote %q twice", l)
		}
		if slices.ContainsFunc(inputs, func(in string) bool { return strings.EqualFold(in, l) }) {
			t.Errorf("Run wrote input name %q", l)
		}
		seen[key] = true
	}
	if again, _ := run(); again != out {
		t.Error("Run is not deterministic for a seed")
	}
}

func TestNewRejectsBadInput(t *testing.T) {
	empty, _ := markov.New(3)
	if _, err := New(empty, Options{Max: 5}); err == nil {
		t.Error("New accepted an empty model")
	}
	if _, err := New(model(t), Options{Max: 0}); err == nil {
		t.Error("New accepted Max 0")
	}
}

func TestJoinMatchesCase(t *testing.T) {
	for _, c := range []struct{ name, want string }{
		{"ShadowWolf", "ShadowHawk"},
		{"shadow_wolf", "shadow_hawk"},
		{"SHADOW_WOLF", "SHADOW_HAWK"},
	} {
		segs := markov.Split(c.name)
		i := len(segs) - 1
		if got := join(segs, i, "hawk", styleOf(segs[i].Text)); got != c.want {
			t.Errorf("swapping hawk into %q = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestDrop(t *testing.T) {
	for name, want := range map[string]string{
		"Dark_Wolf_7": "Dark_7",
		"DarkWolf99":  "Dark99",
		"Wolf_Dark":   "Dark",
		"Dark_Wolf":   "Dark",
	} {
		segs := markov.Split(name)
		i := slices.IndexFunc(segs, func(s markov.Segment) bool { return s.Text == "Wolf" })
		if got := drop(segs, i); got != want {
			t.Errorf("drop(%q, Wolf) = %q, want %q", name, got, want)
		}
	}
}
