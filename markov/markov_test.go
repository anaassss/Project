package markov

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
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
	before := m.Patterns()
	for _, name := range []string{"ShadowWolf", "shadowwolf", "SHADOWWOLF"} {
		if added, err := m.Learn(name); err != nil || added {
			t.Errorf("Learn(%q) = %v, %v; want false, nil", name, added, err)
		}
	}
	if got := m.Stats().Names; got != uint64(len(training)) {
		t.Errorf("learned %d names, want %d", got, len(training))
	}
	if m.Patterns() != before {
		t.Error("re-learning a known name changed the model")
	}
}

func TestLearnRejectsGarbage(t *testing.T) {
	m, _ := New(3)
	for _, name := range []string{"", "has space", "tab\there", "ctrl\x02", "bad\xff", "asdfghjkl", "user_123"} {
		if _, err := m.Learn(name); !errors.Is(err, ErrGarbage) {
			t.Errorf("Learn(%q) error = %v, want ErrGarbage", name, err)
		}
	}
	if m.Stats().Names != 0 || m.Patterns() != 0 {
		t.Error("garbage was learned")
	}
}

func TestLearnFromMatchesLearn(t *testing.T) {
	// Enough names that every worker gets several batches.
	var lines []string
	for i := range 3 * batchSize * 4 {
		if name := fmt.Sprintf("%s_%d", training[i%len(training)], i/len(training)); Check(name) == nil {
			lines = append(lines, name)
		}
	}
	input := "# comment\n\n" + strings.Join(lines, "\n") + "\n  ShadowWolf_0  \nshadowwolf_0\nasdfghjkl\nuser_5\n"

	parallel, _ := New(3)
	var rejected []string
	res, err := parallel.LearnFrom(strings.NewReader(input), func(name, reason string) {
		rejected = append(rejected, name+": "+reason)
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != len(lines) || res.Known != 2 || res.RejectedTotal() != 2 {
		t.Errorf("LearnFrom = %+v, want %d added, 2 known, 2 rejected", res, len(lines))
	}
	if want := []string{"asdfghjkl: keyboard or alphabet sequence", "user_5: placeholder or auto-generated"}; !slices.Equal(rejected, want) {
		t.Errorf("rejected %v, want %v", rejected, want)
	}

	sequential, _ := New(3)
	ScanNames(strings.NewReader(input), func(name string) { sequential.Learn(name) })
	if !maps.Equal(parallel.grams, sequential.grams) || !maps.Equal(parallel.words, sequential.words) ||
		!maps.Equal(parallel.numbers, sequential.numbers) || parallel.stats != sequential.stats {
		t.Error("LearnFrom built a different model than Learn")
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

func TestComplete(t *testing.T) {
	m := trained(t, 3)
	r := rng()
	completed := 0
	for range 50 {
		got, ok := m.Complete(r, "Shadow", 12)
		if !ok {
			continue
		}
		completed++
		if !strings.HasPrefix(got, "Shadow") || len(got) <= len("Shadow") || len(got) > 12 {
			t.Errorf("Complete(Shadow, 12) = %q", got)
		}
	}
	if completed == 0 {
		t.Error("Complete never succeeded")
	}
	// Models too deep to pack contexts into integers complete the same way.
	deep := trained(t, maxPackedOrder+1)
	if deep.sampler().full != nil {
		t.Fatal("deep model unexpectedly uses packed contexts")
	}
	ok := false
	for range 50 {
		if got, done := deep.Complete(r, "Shadow", 12); done {
			ok = true
			if !strings.HasPrefix(got, "Shadow") || len(got) <= len("Shadow") {
				t.Errorf("deep Complete(Shadow) = %q", got)
			}
		}
	}
	if !ok {
		t.Error("deep Complete never succeeded")
	}
	// It must not guess past a context it has never seen.
	if got, ok := m.Complete(r, "Qzx", 20); ok {
		t.Errorf("Complete(Qzx) = %q, want failure", got)
	}
	// A full learned name must be extended or fail, never returned as is.
	for range 20 {
		if got, ok := m.Complete(r, "ShadowWolf", 30); ok && got == "ShadowWolf" {
			t.Fatal("Complete returned the prefix unchanged")
		}
	}
}

func TestFoldHash(t *testing.T) {
	if FoldHash("ShadowWolf") != FoldHash("shadowWOLF") {
		t.Error("ASCII hash is case-sensitive")
	}
	if FoldHash("ShadowWolf") != foldHashSlow("ShadowWolf") {
		t.Error("fast and slow paths disagree on ASCII")
	}
	if FoldHash("Ünïcødé") != FoldHash("üNÏCØDÉ") {
		t.Error("non-ASCII hash is case-sensitive")
	}
	if FoldHash("ShadowWolf") == FoldHash("ShadowWolf1") {
		t.Error("different names hash alike")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	m := trained(t, 3)
	m.Learn("Ünïcødé_Ωmega")
	m.Learn("Agent_007")
	path := filepath.Join(t.TempDir(), "model")
	if err := m.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Order() != m.Order() || loaded.Stats() != m.Stats() ||
		!maps.Equal(loaded.grams, m.grams) || !maps.Equal(loaded.words, m.words) ||
		!maps.Equal(loaded.numbers, m.numbers) || loaded.seen.len() != m.seen.len() {
		t.Fatal("loaded model differs from saved model")
	}
	if !loaded.Knows("ünïcødé_ωmega") || !loaded.Knows("SHADOWWOLF") {
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

func TestLoadConvertsVersion1(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.json")
	body := `{"version":1,"order":2,"names":["ShadowWolf","asdfghjkl","NightHawk"],"counts":{}}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if m.Order() != 2 || m.Stats().Names != 2 || !m.Knows("nighthawk") || m.Knows("asdfghjkl") {
		t.Errorf("converted model: order %d, %d names, knows garbage %v",
			m.Order(), m.Stats().Names, m.Knows("asdfghjkl"))
	}
}

func TestLoadRejectsBadFiles(t *testing.T) {
	dir := t.TempDir()
	if _, err := Load(filepath.Join(dir, "missing")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("loading a missing file: %v, want fs.ErrNotExist", err)
	}

	good := filepath.Join(dir, "good")
	if err := trained(t, 3).Save(good); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(good)
	for name, body := range map[string][]byte{
		"empty":      nil,
		"garbage":    []byte("not a model"),
		"truncated":  data[:len(data)/2],
		"bad json":   []byte(`{"version":1,`),
		"json v9":    []byte(`{"version":9,"order":3}`),
		"json order": []byte(`{"version":1,"order":0}`),
		"version":    append([]byte(magic), 9),
	} {
		path := filepath.Join(dir, "bad")
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Errorf("%s: Load succeeded, want error", name)
		}
	}
}

func TestHashSetFilterIsExact(t *testing.T) {
	s := newHashSet()
	r := rng()
	in := map[uint64]bool{}
	for range 50000 {
		h := r.Uint64()
		s.add(h)
		in[h] = true
	}
	for h := range in {
		if !s.has(h) {
			t.Fatalf("has(%x) = false for an added element", h)
		}
	}
	for range 50000 {
		if h := r.Uint64(); s.has(h) != in[h] {
			t.Fatalf("has(%x) disagrees with membership", h)
		}
	}
	// Adding invalidates the filter.
	h := r.Uint64()
	s.add(h)
	if !s.has(h) {
		t.Error("element added after the filter was built is missing")
	}
}

func TestLeetFragmentsAreNotWords(t *testing.T) {
	m, _ := New(3)
	for _, name := range []string{"Fr0zenBreaker", "Byt3Master", "ShadowWolf99", "Agent007Bond"} {
		m.Learn(name)
	}
	for _, frag := range []string{"fr", "zen", "byt"} {
		if _, ok := m.words[frag]; ok {
			t.Errorf("learned leet fragment %q as a word", frag)
		}
	}
	for _, word := range []string{"breaker", "master", "shadow", "wolf", "agent", "bond"} {
		if _, ok := m.words[word]; !ok {
			t.Errorf("did not learn %q", word)
		}
	}
}

func TestWordLike(t *testing.T) {
	m, _ := New(3)
	// Enough two-word names to pass minWordsToJudge.
	first := strings.Fields("john maria juan sarah mike david emma lucas nora omar ivan mei raj ana leo sam kate ben lily max owen zoe ali eva tom")
	last := strings.Fields("smith garcia baker lee jones brown silva khan chen park wood hill king ford hale ross moss nash reed lowe cole dunn fox gray hunt")
	for i, f := range first {
		m.Learn(f + "." + last[i])
	}
	if len(m.words) < minWordsToJudge {
		t.Fatalf("setup learned only %d words", len(m.words))
	}
	for word, want := range map[string]bool{
		"smith": true, "Smith": true, "juanbaker": true, "jsmith": true, "smithj": false,
		"bakerh": false, "x": true, "leepy": false, "retr": false, "smithxyz": false, "jdsmith": false,
	} {
		if got := m.WordLike(word); got != want {
			t.Errorf("WordLike(%q) = %v, want %v", word, got, want)
		}
	}
	if !m.NameWordsLike("john.smith92") || m.NameWordsLike("john.leepy") {
		t.Error("NameWordsLike judged names wrongly")
	}

	small, _ := New(3)
	small.Learn("ShadowWolf")
	if !small.WordLike("anything") {
		t.Error("a model with few words judged a word")
	}
}
