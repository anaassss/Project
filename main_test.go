package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/anaassss/Project/knowledge"
	"github.com/anaassss/Project/markov"
)

// repoRoot is the package directory, captured before tests change directory.
var repoRoot = func() string {
	wd, _ := os.Getwd()
	return wd
}()

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Fields(string(data))
}

// trainedDir changes to a temporary directory holding the test usernames as
// names.txt.
func trainedDir(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	train, err := os.ReadFile(filepath.Join(repoRoot, "testdata", "usernames.txt"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, "names.txt", string(train))
}

func runMenu(t *testing.T, answers ...string) string {
	t.Helper()
	var out strings.Builder
	if err := menu(strings.NewReader(strings.Join(answers, "\n")+"\n"), &out, settingsFile); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func mustContain(t *testing.T, got string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func mustGlob(t *testing.T, pattern string) string {
	t.Helper()
	files, _ := filepath.Glob(pattern)
	if len(files) != 1 {
		t.Fatalf("%s matched %v", pattern, files)
	}
	return files[0]
}

// editedFile returns the lines of the one edited_*.txt file.
func editedFile(t *testing.T) []string {
	t.Helper()
	return readLines(t, mustGlob(t, "edited_*.txt"))
}

func TestMenuTrainThenEditOwn(t *testing.T) {
	trainedDir(t)
	writeFile(t, "to edit.txt", "ShadowFox\nDarkWolf_7\nshadowfox\n")
	got := runMenu(t,
		"1", "names.txt", // train
		"1", "names.txt", // train again: all already known
		"2", "", "", "1", "", `to\ edit.txt`, // edit with own knowledge, defaults otherwise
		"0",
	)
	mustContain(t, got,
		"1 - Training (your own knowledge)",
		"Training [██████████████████████████████] 100%",
		"Learned 155 new usernames in ",
		"(skipped 155 already known, 0 garbage)",
		"Which knowledge?\n  1 - Own\n  2 - Claude\n  3 - Both\n",
		"Choose 1-3 [3]: ",
		"Edits per username (1-1000, or max) [10]: ",
		"Editing  [██████████████████████████████] 100%",
	)
	saved := regexp.MustCompile(`Saved (\d+) edited usernames \(3-24 characters\) to (edited_\d+\.txt) in `).FindStringSubmatch(got)
	if saved == nil {
		t.Fatalf("no edits saved:\n%s", got)
	}
	edits := readLines(t, saved[2])
	if len(edits) == 0 || len(edits) > 30 { // up to 10 for each of 3 lines
		t.Fatalf("%s has %d lines, want 1-30", saved[2], len(edits))
	}
	for _, ed := range edits {
		if strings.EqualFold(ed, "ShadowFox") || strings.EqualFold(ed, "DarkWolf_7") {
			t.Errorf("edited file contains input name %q", ed)
		}
	}
	if c, _ := loadSettings(settingsFile); c.Knowledge != ownKnowledge {
		t.Errorf("knowledge choice not remembered: %q", c.Knowledge)
	}
}

func TestMenuEditClaudeAndMax(t *testing.T) {
	t.Chdir(t.TempDir())
	writeFile(t, "mine.txt", "john.smith\nmaria.lopez92\n")
	got := runMenu(t,
		"2", "", "",
		"1",      // own: nothing trained yet, so refused
		"4", "2", // invalid, then Claude
		"abc", "max", // edits: invalid, then max
		"mine.txt",
		"0",
	)
	mustContain(t, got,
		"You haven't trained your own knowledge yet",
		"Choose 1, 2 or 3.",
		"Enter a number from 1 to 1000, or max.",
		"max makes every edit the knowledge allows",
	)
	edits := editedFile(t)
	if len(edits) < 100 {
		t.Errorf("max gave only %d edits for 2 names", len(edits))
	}
	var john, maria int
	for _, ed := range edits {
		if ed != strings.ToLower(ed) || !knowledge.Unique(ed) {
			t.Errorf("edit %q isn't lowercase and knowledge.Unique", ed)
		}
		switch {
		case strings.HasPrefix(ed, "john.") || strings.HasSuffix(ed, ".smith"):
			john++
		case strings.HasPrefix(ed, "maria.") || strings.Contains(ed, ".lopez"):
			maria++
		}
	}
	if john < 20 || maria < 20 {
		t.Errorf("edits don't cover both names: %d of john.smith, %d of maria.lopez92", john, maria)
	}
	c, _ := loadSettings(settingsFile)
	if c.Knowledge != claudeKnowledge || c.EditsPerName != 0 {
		t.Errorf("choices not remembered: knowledge %q, edits %d", c.Knowledge, c.EditsPerName)
	}

	// Next time, Enter keeps Claude and max.
	os.Remove(mustGlob(t, "edited_*.txt"))
	got = runMenu(t, "2", "", "", "", "", "mine.txt", "0")
	mustContain(t, got, "Choose 1-3 [2]: ", "or max) [max]: ")
	if n := len(editedFile(t)); n == 0 {
		t.Error("repeat with the saved choices gave no edits")
	}
}

func TestMenuBothWithoutOwnUsesClaude(t *testing.T) {
	t.Chdir(t.TempDir())
	writeFile(t, "mine.txt", "maria.garcia\n")
	got := runMenu(t, "2", "", "", "3", "5", "mine.txt", "0")
	mustContain(t, got, "No own knowledge yet, so using Claude's.", "Saved ")
	if n := len(editedFile(t)); n == 0 || n > 5 {
		t.Errorf("got %d edits, want 1-5", n)
	}
}

func TestMenuSurvivesErrorsAndEOF(t *testing.T) {
	t.Chdir(t.TempDir())
	// A missing file is reported and the menu continues; input then ends.
	got := runMenu(t, "1", "missing.txt", "9")
	mustContain(t, got, "Error: stat missing.txt", "Choose 1-6, or 0 to exit.")
}

func TestMenuInfoAndClear(t *testing.T) {
	trainedDir(t)
	writeFile(t, legacyModel, `{"version":1,"order":3,"names":["OldName"]}`)
	got := runMenu(t,
		"1", "names.txt", // train (loads the old model, saves the new one)
		"4",       // info
		"6", "no", // clear, refused
		"6", "YES", // clear, confirmed
		"4", // info: own knowledge gone, Claude's remains
		"0",
	)
	mustContain(t, got,
		"Own knowledge\n  Model file   usergen.model (",
		"Usernames    156 learned, 3-15 characters",
		"most common: wolf,",
		"Claude knowledge (built in)\n  Email-style  ",
		"(156 usernames) and cannot be undone. Claude's built-in knowledge stays.",
		"Nothing cleared.",
		"Own knowledge cleared.",
		"Own knowledge\n  Nothing learned yet. Choose 1 - Training to teach it.",
	)
	for _, f := range []string{defaultModel, legacyModel} {
		if _, err := os.Stat(f); err == nil {
			t.Errorf("%s still exists after clearing", f)
		}
	}
}

func TestMenuEditAsksLengths(t *testing.T) {
	trainedDir(t)
	writeFile(t, "mine.txt", "ShadowFox\nDarkWolf_7\nStormHawk99\nMysticPanda\n")
	got := runMenu(t,
		"1", "names.txt",
		"2",
		"2", "abc", "6", // shortest: out of range, not a number, then 6
		"5", "10", // longest: below the shortest, then 10
		"1", "", "mine.txt",
		"0",
	)
	mustContain(t, got,
		"Shortest length (3-24) [3]: ",
		"Enter a whole number from 3 to 24.",
		"Longest length (6-24) [24]: ",
		"Enter a whole number from 6 to 24.",
		"edited usernames (6-10 characters) to edited_",
	)
	for _, ed := range editedFile(t) {
		if n := len([]rune(ed)); n < 6 || n > 10 {
			t.Errorf("edit %q has %d characters, want 6-10", ed, n)
		}
	}
	if c, _ := loadSettings(settingsFile); c.EditMinLen != 6 || c.EditMaxLen != 10 {
		t.Errorf("lengths not remembered: %d-%d", c.EditMinLen, c.EditMaxLen)
	}
}

func TestMenuSettingsApply(t *testing.T) {
	trainedDir(t)
	writeFile(t, "mine.txt", "ShadowFox\nDarkWolf_7\nStormHawk99\n")
	got := runMenu(t,
		"5",             // settings
		"4", "abc", "3", // edits per username: invalid, then 3
		"9", "2", // generate longest length 2: out of range
		"8", "20", // generate shortest 20 > longest 16: refused
		"7", "7", // generate 7 usernames
		"12",       // knowledge: both → own
		"15", "42", // seed
		"16", "other.model", // own knowledge file
		"0",
		"1", "names.txt", // train into other.model
		"2", "", "", "", "", "mine.txt", // edit with the saved choices
		"3", "", // generate with the saved knowledge
		"0",
	)
	mustContain(t, got,
		"Enter a number from 1 to 1000, or max.",
		"Not changed: generate lengths must be 3-24.",
		"Not changed: generate shortest length 20 is above longest length 16.",
		"   12  Knowledge (own, claude or both)                own",
		"   13  Unique email style (no plain or decorated)     on",
		"   16  File                                           other.model",
		"Saved 7 new usernames to generated_7.txt",
	)
	if _, err := os.Stat("other.model"); err != nil {
		t.Errorf("training ignored the file setting: %v", err)
	}
	if n := len(editedFile(t)); n == 0 || n > 9 {
		t.Errorf("%d edits for 3 names at 3 per name, want 1-9", n)
	}
	if n := len(readLines(t, "generated_7.txt")); n != 7 {
		t.Errorf("generated_7.txt has %d names, want 7", n)
	}

	c, err := loadSettings(settingsFile)
	if err != nil {
		t.Fatal(err)
	}
	want := defaultSettings()
	want.EditsPerName, want.GenCount, want.Seed, want.ModelFile, want.Knowledge = 3, 7, 42, "other.model", ownKnowledge
	if c != want {
		t.Errorf("saved settings = %+v, want %+v", c, want)
	}
	mustContain(t, runMenu(t, "4", "0"), "Model file   other.model")
}

func TestMenuDryRunAndRejected(t *testing.T) {
	trainedDir(t)
	writeFile(t, "messy.txt", "ShadowFox\nasdfghjkl\nuser_123\njohn@mail.com\n")
	got := runMenu(t,
		"5", "2", "3", "0", // turn on save rejected and dry run
		"1", "messy.txt",
		"4", // info: nothing learned
		"0",
	)
	mustContain(t, got,
		"1 - Training (your own knowledge)   (dry run is on: nothing will be saved)",
		"Dry run: would learn 1 new usernames",
		"Nothing was saved.",
		"Saved 3 rejected usernames, with reasons, to rejected_3.txt",
		"Nothing learned yet",
	)
	if _, err := os.Stat(defaultModel); err == nil {
		t.Error("dry run saved a model")
	}
	data, _ := os.ReadFile("rejected_3.txt")
	if want := "asdfghjkl\tkeyboard or alphabet sequence\nuser_123\tplaceholder or auto-generated\njohn@mail.com\tdisallowed character\n"; string(data) != want {
		t.Errorf("rejected_3.txt = %q, want %q", data, want)
	}
	if tmp, _ := filepath.Glob(".rejected-*"); len(tmp) != 0 {
		t.Errorf("temporary files left behind: %v", tmp)
	}
}

func TestEditCommand(t *testing.T) {
	t.Chdir(t.TempDir())
	writeFile(t, "mine.txt", "john.smith\nSilentWolf\n")
	if err := editCommand([]string{"-knowledge", "own", "mine.txt"}); err == nil {
		t.Error("edit with own knowledge succeeded without a trained model")
	}
	if err := editCommand([]string{"-edits", "lots", "mine.txt"}); err == nil {
		t.Error("edit accepted -edits lots")
	}
	if err := editCommand([]string{"-knowledge", "claude", "-edits", "4", "-seed", "1", "mine.txt"}); err != nil {
		t.Fatal(err)
	}
	if n := len(editedFile(t)); n == 0 || n > 8 {
		t.Errorf("got %d edits for 2 names at 4 per name", n)
	}
}

func TestGenerateFrom(t *testing.T) {
	own, _ := markov.New(3)
	f, err := os.Open("testdata/usernames.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := own.LearnFrom(f, nil); err != nil {
		t.Fatal(err)
	}
	models := []*markov.Model{own, knowledge.Model()}
	opts := markov.GenerateOptions{MinLen: 4, MaxLen: 16, Temperature: 1}
	for _, unique := range []bool{false, true} {
		names, err := generateFrom(models, 21, opts, unique, 5, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(names) != 21 {
			t.Errorf("unique=%v: got %d names, want 21", unique, len(names))
		}
		seen := map[string]bool{}
		for _, n := range names {
			if seen[strings.ToLower(n)] || slices.ContainsFunc(models, func(m *markov.Model) bool { return m.Knows(n) }) {
				t.Errorf("%q is a duplicate or a learned name", n)
			}
			seen[strings.ToLower(n)] = true
			if unique && (n != strings.ToLower(n) || !knowledge.Unique(n)) {
				t.Errorf("unique name %q isn't lowercase and knowledge.Unique", n)
			}
		}
	}
}

func TestParseEdits(t *testing.T) {
	for in, want := range map[string]int{"1": 1, "10": 10, "1,000": 1000, "max": 0, "MAX": 0} {
		if got, err := parseEdits(in); err != nil || got != want {
			t.Errorf("parseEdits(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"0", "-1", "1001", "all", ""} {
		if _, err := parseEdits(in); err == nil {
			t.Errorf("parseEdits(%q) succeeded", in)
		}
	}
}

func TestPlaceNumberedNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	var got []string
	for i := range 3 {
		tmp := filepath.Join(dir, "tmp")
		writeFile(t, tmp, strings.Repeat("x", i))
		p, err := placeNumbered(tmp, dir, "edited", 2)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, filepath.Base(p))
	}
	if want := []string{"edited_2.txt", "edited_2_2.txt", "edited_2_3.txt"}; !slices.Equal(got, want) {
		t.Errorf("wrote %v, want %v", got, want)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "edited_2.txt")); len(data) != 0 {
		t.Error("edited_2.txt was overwritten")
	}
}

func TestCleanPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	for in, want := range map[string]string{
		"names.txt":         "names.txt",
		`"my names.txt"`:    "my names.txt",
		`'my names.txt'`:    "my names.txt",
		`my\ names.txt`:     "my names.txt",
		"~/lists/names.txt": filepath.Join(home, "lists", "names.txt"),
		"":                  "",
	} {
		if got := cleanPath(in); got != want {
			t.Errorf("cleanPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatCount(t *testing.T) {
	for n, want := range map[int]string{0: "0", 999: "999", 1000: "1,000", 1234567: "1,234,567", -1234: "-1,234"} {
		if got := formatCount(n); got != want {
			t.Errorf("formatCount(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestFileLabel(t *testing.T) {
	t.Chdir(t.TempDir())
	if got := fileLabel(defaultModel); got != defaultModel {
		t.Errorf("missing model labelled %q", got)
	}
	writeFile(t, legacyModel, strings.Repeat("x", 1500))
	if got, want := fileLabel(defaultModel), "usergen.json (1.5 KB, old format; converted on next training)"; got != want {
		t.Errorf("fileLabel = %q, want %q", got, want)
	}
	writeFile(t, defaultModel, strings.Repeat("x", 2_500_000))
	if got, want := fileLabel(defaultModel), "usergen.model (2.5 MB)"; got != want {
		t.Errorf("fileLabel = %q, want %q", got, want)
	}
}

func TestLoadSettings(t *testing.T) {
	t.Chdir(t.TempDir())
	if c, err := loadSettings("missing.json"); err != nil || c != defaultSettings() {
		t.Errorf("missing file: %+v, %v; want defaults", c, err)
	}
	writeFile(t, "partial.json", `{"edits_per_username": 4}`)
	if c, err := loadSettings("partial.json"); err != nil || c.EditsPerName != 4 || c.Knowledge != bothKnowledge {
		t.Errorf("partial file: %+v, %v", c, err)
	}
	for name, body := range map[string]string{
		"garbage.json":   "not json",
		"invalid.json":   `{"generate_min_length": 20, "generate_max_length": 5}`,
		"knowledge.json": `{"knowledge": "everything"}`,
	} {
		writeFile(t, name, body)
		if c, err := loadSettings(name); err == nil || c != defaultSettings() {
			t.Errorf("%s: %+v, %v; want defaults and an error", name, c, err)
		}
	}
	writeFile(t, settingsFile, "not json")
	mustContain(t, runMenu(t, "0"), "using default settings")
}
