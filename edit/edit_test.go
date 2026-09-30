package edit

import (
	"bytes"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

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
	e, err := New([]*markov.Model{model(t)}, opts)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// knows reports whether any of e's models learned name.
func (e *Editor) knows(name string) bool {
	return slices.ContainsFunc(e.sources, func(s source) bool { return s.m.Knows(name) })
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
			case e.knows(ed):
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
		return ok && slices.Contains(e.sources[0].words, strings.ToLower(rest))
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
		return slices.ContainsFunc(e.Edit(rng(), "ShadowFox", nil), e.knows)
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
	if _, err := New([]*markov.Model{empty, nil}, Options{Max: 5}); err == nil {
		t.Error("New accepted models that know nothing")
	}
	if _, err := New([]*markov.Model{model(t)}, Options{Max: -1}); err == nil {
		t.Error("New accepted Max -1")
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

func TestEditRespectsLengthBounds(t *testing.T) {
	for _, c := range []struct {
		name     string
		min, max int
	}{
		{"ShadowFox", 6, 9},
		{"DarkWolf_7", 11, 14},
		{"StormHawk99", 3, 9},
		{"Shadow_Fox", 13, 20}, // longer than the name itself
	} {
		e := editor(t, Options{Max: 30, MinLen: c.min, MaxLen: c.max})
		edits := e.Edit(rng(), c.name, nil)
		if len(edits) == 0 {
			t.Errorf("Edit(%q) within %d-%d returned nothing", c.name, c.min, c.max)
		}
		for _, ed := range edits {
			if n := utf8.RuneCountInString(ed); n < c.min || n > c.max {
				t.Errorf("Edit(%q) within %d-%d returned %q (%d characters)", c.name, c.min, c.max, ed, n)
			}
		}
	}
}

func TestNewRejectsBadLengths(t *testing.T) {
	for _, o := range []Options{
		{Max: 5, MinLen: 2, MaxLen: 10},
		{Max: 5, MinLen: 5, MaxLen: 25},
		{Max: 5, MinLen: 10, MaxLen: 6},
	} {
		if _, err := New([]*markov.Model{model(t)}, o); err == nil {
			t.Errorf("New accepted lengths %d-%d", o.MinLen, o.MaxLen)
		}
	}
}

func TestEditDeclinesUnreachableLengths(t *testing.T) {
	// Nothing learned can make IronFox 12+ characters: the longest learned
	// word is 7 letters and Fox has only ever ended a name.
	e := editor(t, Options{Max: 10, MinLen: 12, MaxLen: 20})
	if edits := e.Edit(rng(), "IronFox", nil); len(edits) != 0 {
		t.Errorf("Edit(IronFox) within 12-20 = %v, want none", edits)
	}
}

func TestEditPicksFittingKnowledge(t *testing.T) {
	em, _ := markov.New(3)
	gm, _ := markov.New(3)
	first := strings.Fields("john maria juan sarah mike david emma lucas nora omar ivan mei raj ana leo sam kate ben lily max owen zoe ali eva tom")
	last := strings.Fields("smith garcia baker lee jones brown silva khan chen park wood hill king ford hale ross moss nash reed lowe cole dunn fox gray hunt")
	adj := strings.Fields("silent dark happy lazy swift brave wild noble cosmic lunar solar frosty sneaky mighty tiny fuzzy misty royal rapid epic")
	noun := strings.Fields("wolf panda tiger dragon raven hawk falcon eagle bear lion shark viper ninja knight wizard ranger hunter pilot rider storm")
	for i := range first {
		em.Learn(first[i] + "." + last[i])
		em.Learn(first[(i+3)%len(first)] + "_" + last[(i+7)%len(last)])
	}
	for i := range adj {
		for j := range 3 {
			gm.Learn(strings.ToUpper(adj[i][:1]) + adj[i][1:] + strings.ToUpper(noun[(i+j)%len(noun)][:1]) + noun[(i+j)%len(noun)][1:])
		}
	}
	e, err := New([]*markov.Model{em, gm}, Options{Max: 30})
	if err != nil {
		t.Fatal(err)
	}
	hasWordFrom := func(name string, list []string) bool {
		return slices.ContainsFunc(markov.Split(name), func(seg markov.Segment) bool {
			return slices.Contains(list, strings.ToLower(seg.Text))
		})
	}
	for _, ed := range e.Edit(rng(), "john.smith", nil) {
		if hasWordFrom(ed, adj) || hasWordFrom(ed, noun) {
			t.Errorf("email-style name got a gaming word: %q", ed)
		}
	}
	for _, ed := range e.Edit(rng(), "SilentWolf", nil) {
		if hasWordFrom(ed, first) || hasWordFrom(ed, last) {
			t.Errorf("gaming name got an email word: %q", ed)
		}
	}
}

func TestEditEverything(t *testing.T) {
	e := editor(t, Options{Max: 0})
	all := e.Edit(rng(), "ShadowFox", nil)
	some := editor(t, Options{Max: 10}).Edit(rng(), "ShadowFox", nil)
	if len(all) <= len(some) {
		t.Errorf("Max 0 gave %d edits, no more than Max 10's %d", len(all), len(some))
	}
	seen := map[string]bool{}
	for _, ed := range all {
		key := strings.ToLower(ed)
		if seen[key] {
			t.Errorf("Max 0 returned %q twice", ed)
		}
		seen[key] = true
	}
	// Every word that usually starts a name is swapped into the first part,
	// and every word that usually comes later into the second.
	src := e.sources[0]
	var want []string
	for _, w := range src.leads {
		want = append(want, strings.ToUpper(w[:1])+w[1:]+"Fox")
	}
	for _, w := range src.tails {
		want = append(want, "Shadow"+strings.ToUpper(w[:1])+w[1:])
	}
	for _, cand := range want {
		if !seen[strings.ToLower(cand)] && markov.Check(cand) == nil && !e.knows(cand) && !strings.EqualFold(cand, "ShadowFox") {
			t.Errorf("Max 0 missed the swap %q", cand)
		}
	}
	for _, junk := range []string{"FoxFox", "ShadowShadow", "WolfFox"} {
		if seen[strings.ToLower(junk)] {
			t.Errorf("Max 0 swapped a word into a position it never takes: %q", junk)
		}
	}
}

func TestEditKeepsAffixes(t *testing.T) {
	e := editor(t, Options{Max: 0})
	for _, ed := range e.Edit(rng(), "xXSniperXx", nil) {
		if !strings.HasPrefix(ed, "xX") {
			t.Errorf("Edit(xXSniperXx) lost its wrapper: %q", ed)
		}
	}
	for _, ed := range e.Edit(rng(), "ShadowFox", nil) {
		if slices.ContainsFunc(markov.Split(ed), func(s markov.Segment) bool { return markov.IsAffix(s.Text) }) {
			t.Errorf("Edit(ShadowFox) swapped in an affix: %q", ed)
		}
	}
}

func TestEditSplitsJoinedNames(t *testing.T) {
	e := editor(t, Options{Max: 0})
	segs := e.splitJoined(markov.Split("shadowbyte2004"))
	var got []string
	for _, s := range segs {
		got = append(got, s.Text)
	}
	if strings.Join(got, "|") != "shadow|byte|2004" {
		t.Errorf("splitJoined(shadowbyte2004) = %v", got)
	}
	for name, want := range map[string]string{"darkwolf": "dark|wolf", "itsshadow": "its|shadow", "mrsmoon": "mrs|moon", "mrhawk": "mr|hawk"} {
		var got []string
		for _, s := range e.splitJoined(markov.Split(name)) {
			got = append(got, s.Text)
		}
		if strings.Join(got, "|") != want {
			t.Errorf("splitJoined(%s) = %v, want %s", name, got, want)
		}
	}
	// Names the model doesn't know aren't cut into fragments: "wolf" never
	// starts names, "red" is too short to be followed by an unknown rest,
	// and "redder" is too short to split at all.
	for _, name := range []string{"shadow", "ab12", "ShadowByte", "zzqqvv", "wolfgang", "reddington", "redder", "official", "itsred"} {
		if segs := e.splitJoined(markov.Split(name)); len(segs) != len(markov.Split(name)) {
			t.Errorf("splitJoined split %q into %v", name, segs)
		}
	}
	// Edits of a joined name swap one part and stay joined.
	for _, ed := range e.Edit(rng(), "shadowbyte2004", nil) {
		if strings.ContainsAny(ed, "._-") || ed != strings.ToLower(ed) {
			t.Errorf("edit %q of shadowbyte2004 isn't joined lowercase", ed)
		}
	}
	// A leading affix is kept while the name after it is swapped.
	swapped := false
	for _, ed := range e.Edit(rng(), "itsshadow", nil) {
		if !strings.HasPrefix(ed, "its") {
			t.Errorf("edit %q of itsshadow lost its prefix", ed)
		}
		swapped = swapped || !strings.Contains(ed, "shadow")
	}
	if !swapped {
		t.Error("no edit of itsshadow swapped the name after its")
	}
}

func TestNumberSwapsKeepShape(t *testing.T) {
	e := editor(t, Options{Max: 40})
	src := e.sources[0]
	r := rng()
	for _, num := range []string{"4137", "2004", "975", "0108", "7"} {
		for range 20 {
			got := src.numberLike(r, num)
			if len(got) != len(num) || numberShape(got) != numberShape(num) {
				t.Errorf("numberLike(%q) = %q, not the same shape", num, got)
			}
			if num[0] != '0' && got[0] == '0' {
				t.Errorf("numberLike(%q) = %q starts with 0", num, got)
			}
		}
	}
}
