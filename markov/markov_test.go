package markov

import (
	"errors"
	"io/fs"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

var training = []string{
	"ShadowWolf", "ShadowHunter", "DarkKnight", "DarkPhoenix", "NightWolf",
	"NightHawk", "IronFist", "IronWolf", "StormRider", "StormBreaker",
	"FrostByte", "FrostFang", "PixelNinja", "PixelKnight", "CyberHawk",
	"CyberPunk", "SilentStorm", "SilentBlade", "BlazeRunner", "LunarFox",
}

func trained(t *testing.T, order int) *Model {
	t.Helper()
	m, err := New(order)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range training {
		if _, err := m.Learn(name); err != nil {
			t.Fatalf("Learn(%q): %v", name, err)
		}
	}
	return m
}

func rng() *rand.Rand { return rand.New(rand.NewPCG(1, 2)) }

func TestLearnDeduplicatesCaseInsensitively(t *testing.T) {
	m := trained(t, 2)
	before := m.Contexts()
	for _, name := range []string{"ShadowWolf", "shadowwolf", "SHADOWWOLF"} {
		if added, err := m.Learn(name); err != nil || added {
			t.Errorf("Learn(%q) = %v, %v; want false, nil", name, added, err)
		}
	}
	if got := len(m.Names()); got != len(training) {
		t.Errorf("learned %d names, want %d", got, len(training))
	}
	if m.Contexts() != before {
		t.Error("re-learning a known name changed the model")
	}
}

func TestLearnRejectsInvalid(t *testing.T) {
	m, _ := New(3)
	for _, name := range []string{"", "has space", "tab\there", "ctrl\x02", "bad\xff", "asdfghjkl", "user_123"} {
		if _, err := m.Learn(name); !errors.Is(err, ErrGarbage) {
			t.Errorf("Learn(%q) error = %v, want ErrGarbage", name, err)
		}
	}
	if len(m.Names()) != 0 || m.Contexts() != 0 {
		t.Error("garbage was learned")
	}
}

func TestGenerateRespectsOptions(t *testing.T) {
	m := trained(t, 2)
	opts := GenerateOptions{MinLen: 5, MaxLen: 12, Temperature: 1}
	names, err := m.Generate(rng(), 20, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("generated no names")
	}
	seen := map[string]bool{}
	for _, name := range names {
		if n := len([]rune(name)); n < opts.MinLen || n > opts.MaxLen {
			t.Errorf("%q has length %d, want %d-%d", name, n, opts.MinLen, opts.MaxLen)
		}
		if m.Knows(name) {
			t.Errorf("%q is a training name", name)
		}
		if seen[name] {
			t.Errorf("%q generated twice", name)
		}
		seen[name] = true
	}
}

func TestGenerateIsDeterministicForSeed(t *testing.T) {
	m := trained(t, 3)
	opts := GenerateOptions{MinLen: 3, MaxLen: 20, Temperature: 1.2, Order: 2}
	a, _ := m.Generate(rng(), 10, opts)
	b, _ := m.Generate(rng(), 10, opts)
	if !slices.Equal(a, b) {
		t.Errorf("same seed gave different names:\n%v\n%v", a, b)
	}
}

func TestGenerateRejectsBadOptions(t *testing.T) {
	m := trained(t, 3)
	for _, opts := range []GenerateOptions{
		{MinLen: 0, MaxLen: 10, Temperature: 1},
		{MinLen: 8, MaxLen: 4, Temperature: 1},
		{MinLen: MaxNameLen + 1, MaxLen: MaxNameLen + 5, Temperature: 1},
		{MinLen: 3, MaxLen: 10, Temperature: 0},
		{MinLen: 3, MaxLen: 10, Temperature: 1, Order: 4},
	} {
		if _, err := m.Generate(rng(), 5, opts); err == nil {
			t.Errorf("Generate(%+v) succeeded, want error", opts)
		}
	}
	empty, _ := New(3)
	if _, err := empty.Generate(rng(), 5, GenerateOptions{MinLen: 1, MaxLen: 5, Temperature: 1}); err == nil {
		t.Error("empty model generated names")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	m := trained(t, 3)
	m.Learn("Ünïcødé_Ωmega")
	path := filepath.Join(t.TempDir(), "model.json")
	if err := m.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Order() != m.Order() || loaded.Contexts() != m.Contexts() ||
		!slices.Equal(loaded.Names(), m.Names()) {
		t.Fatal("loaded model differs from saved model")
	}
	if !loaded.Knows("ünïcødé_ωmega") {
		t.Error("loaded model forgot a learned name")
	}

	opts := GenerateOptions{MinLen: 3, MaxLen: 20, Temperature: 1}
	want, _ := m.Generate(rng(), 10, opts)
	got, _ := loaded.Generate(rng(), 10, opts)
	if !slices.Equal(got, want) {
		t.Errorf("loaded model generates differently:\n%v\n%v", got, want)
	}

	// Learning continues from where the saved model left off.
	if added, _ := loaded.Learn("BrandNewName"); !added {
		t.Error("loaded model could not learn a new name")
	}
	if added, _ := loaded.Learn("ShadowWolf"); added {
		t.Error("loaded model re-learned a saved name")
	}
}

func TestLoadRejectsBadFiles(t *testing.T) {
	dir := t.TempDir()
	if _, err := Load(filepath.Join(dir, "missing.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("loading a missing file: %v, want fs.ErrNotExist", err)
	}
	for name, body := range map[string]string{
		"garbage":     `not json`,
		"version":     `{"version":99,"order":3}`,
		"order":       `{"version":1,"order":0}`,
		"long ctx":    `{"version":1,"order":1,"counts":{"ab":{"c":1}}}`,
		"bad count":   `{"version":1,"order":2,"counts":{"a":{"b":0}}}`,
		"multi-char":  `{"version":1,"order":2,"counts":{"a":{"bc":1}}}`,
		"empty rune":  `{"version":1,"order":2,"counts":{"a":{"":1}}}`,
		"trailing ok": `{"version":1,"order":2,"counts":{"a":{"b":1}}} `,
	} {
		path := filepath.Join(dir, "model.json")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Load(path)
		if wantOK := name == "trailing ok"; (err == nil) != wantOK {
			t.Errorf("%s: Load error = %v, want ok=%v", name, err, wantOK)
		}
	}
}

func TestScoreRanksPlausibleAboveGarbage(t *testing.T) {
	m := trained(t, 3)
	if good, bad := m.Score("ShadowHawk"), m.Score("Qzxwvk"); good <= bad {
		t.Errorf("Score(ShadowHawk) = %.2f, not above Score(Qzxwvk) = %.2f", good, bad)
	}
}

func TestHeldOutScores(t *testing.T) {
	m := trained(t, 3)
	held := m.HeldOutScores()
	if len(held) != len(training) {
		t.Fatalf("got %d held-out scores, want %d", len(held), len(training))
	}
	for i, name := range m.Names() {
		if in := m.Score(name); held[i] >= in {
			t.Errorf("%s: held-out score %.2f not below in-sample %.2f", name, held[i], in)
		}
	}

	// Holding out a lone name leaves nothing learned: every character backs off
	// through all 3 context lengths (order 2, 1, 0) and is then unseen.
	one, _ := New(2)
	one.Learn("Solo")
	if got, want := one.HeldOutScores()[0], 3*backoffPenalty+unseenPenalty; math.Abs(got-want) > 1e-9 {
		t.Errorf("lone held-out score = %v, want %v", got, want)
	}
}

func TestComplete(t *testing.T) {
	m := trained(t, 3)
	r := rng()
	for range 50 {
		got, ok := m.Complete(r, "Shadow", 12)
		if !ok {
			continue
		}
		if !strings.HasPrefix(got, "Shadow") || len([]rune(got)) <= len("Shadow") || len([]rune(got)) > 12 {
			t.Errorf("Complete(Shadow, 12) = %q", got)
		}
	}
	// A full learned name must still be extended, not ended immediately.
	if got, ok := m.Complete(r, "ShadowWolf", 30); ok && got == "ShadowWolf" {
		t.Error("Complete returned the prefix unchanged")
	}
}
