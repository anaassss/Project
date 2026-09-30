package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
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

func TestMenuTrainThenEdit(t *testing.T) {
	t.Chdir(t.TempDir())
	train, err := os.ReadFile(filepath.Join(repoRoot, "examples", "usernames.txt"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, "names.txt", string(train))
	writeFile(t, "to edit.txt", "ShadowFox\nDarkWolf_7\nshadowfox\n")

	input := strings.Join([]string{
		"2",              // edit before training: refused
		"1", "names.txt", // train
		"1", "names.txt", // train again: all already known
		"2", `to\ edit.txt`, // edit
		"0",
	}, "\n") + "\n"
	var out strings.Builder
	if err := menu(strings.NewReader(input), &out, settingsFile); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{
		"Nothing learned yet",
		"Training [██████████████████████████████] 100%",
		"Learned 155 new usernames in ",
		"Learned 0 new usernames in ",
		"(skipped 155 already known, 0 garbage)",
		"Editing  [██████████████████████████████] 100%",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("menu output missing %q:\n%s", want, got)
		}
	}

	saved := regexp.MustCompile(`Saved (\d+) edited usernames to (edited_\d+\.txt) in `).FindStringSubmatch(got)
	if saved == nil {
		t.Fatalf("no edits saved:\n%s", got)
	}
	if want := "edited_" + saved[1] + ".txt"; saved[2] != want {
		t.Errorf("saved to %s, want %s", saved[2], want)
	}
	edits := readLines(t, saved[2])
	if len(edits) == 0 || len(edits) > 20 {
		t.Fatalf("%s has %d lines, want 1-20", saved[2], len(edits))
	}
	for _, ed := range edits {
		if strings.EqualFold(ed, "ShadowFox") || strings.EqualFold(ed, "DarkWolf_7") {
			t.Errorf("edited file contains input name %q", ed)
		}
	}
	if _, err := os.Stat(defaultModel); err != nil {
		t.Errorf("model not saved: %v", err)
	}
}

func TestMenuSurvivesErrorsAndEOF(t *testing.T) {
	t.Chdir(t.TempDir())
	var out strings.Builder
	// A missing file is reported and the menu continues; input then ends.
	if err := menu(strings.NewReader("1\nmissing.txt\n7\n"), &out, settingsFile); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); !strings.Contains(got, "Error: stat missing.txt") || !strings.Contains(got, "Choose 1-6, or 0 to exit.") {
		t.Errorf("unexpected output:\n%s", got)
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

func TestMenuInfoAndClear(t *testing.T) {
	t.Chdir(t.TempDir())
	train, err := os.ReadFile(filepath.Join(repoRoot, "examples", "usernames.txt"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, "names.txt", string(train))
	writeFile(t, legacyModel, `{"version":1,"order":3,"names":["OldName"]}`)

	input := strings.Join([]string{
		"1", "names.txt", // train (loads the old model, saves the new one)
		"4",       // info
		"6", "no", // clear, refused
		"6", "YES", // clear, confirmed
		"4", // info: nothing left
		"2", // edit: nothing left
		"0",
	}, "\n") + "\n"
	var out strings.Builder
	if err := menu(strings.NewReader(input), &out, settingsFile); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{
		"Model file   usergen.model (",
		"Usernames    156 learned, 3-15 characters",
		"Words        ",
		"most common: wolf,",
		"(156 usernames) and cannot be undone",
		"Nothing cleared.",
		"All knowledge cleared.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("menu output missing %q:\n%s", want, got)
		}
	}
	if n := strings.Count(got, "Nothing learned yet"); n != 2 {
		t.Errorf("after clearing, %d choices said nothing is learned, want 2:\n%s", n, got)
	}
	for _, f := range []string{defaultModel, legacyModel} {
		if _, err := os.Stat(f); err == nil {
			t.Errorf("%s still exists after clearing", f)
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

// trainedDir changes to a temporary directory holding the example usernames
// as names.txt.
func trainedDir(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	train, err := os.ReadFile(filepath.Join(repoRoot, "examples", "usernames.txt"))
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

func TestMenuSettingsApply(t *testing.T) {
	trainedDir(t)
	writeFile(t, "mine.txt", "ShadowFox\nDarkWolf_7\nStormHawk99\n")
	got := runMenu(t,
		"5",             // settings
		"4", "abc", "3", // edits per username: invalid, then 3
		"6", "20", // shortest length 20 > longest 16: refused
		"5", "7", // generate 7 usernames
		"11", "42", // seed
		"12", "other.model", // model file
		"0",
		"1", "names.txt", // train into other.model
		"2", "mine.txt", // edit
		"3", // generate
		"0",
	)
	for _, want := range []string{
		"Enter a whole number.",
		"Not changed: shortest length 20 is above longest length 16.",
		"   12  Model file                                     other.model",
		"Saved 7 new usernames to generated_7.txt",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("menu output missing %q:\n%s", want, got)
		}
	}
	if _, err := os.Stat("other.model"); err != nil {
		t.Errorf("training ignored the model file setting: %v", err)
	}
	edited, _ := filepath.Glob("edited_*.txt")
	if len(edited) != 1 {
		t.Fatalf("edited files: %v", edited)
	}
	if n := len(readLines(t, edited[0])); n == 0 || n > 9 {
		t.Errorf("%d edits for 3 names at 3 per name, want 1-9", n)
	}
	if n := len(readLines(t, "generated_7.txt")); n != 7 {
		t.Errorf("generated_7.txt has %d names, want 7", n)
	}

	// Settings persist, and a new session uses them.
	c, err := loadSettings(settingsFile)
	if err != nil {
		t.Fatal(err)
	}
	want := defaultSettings()
	want.EditsPerName, want.GenCount, want.Seed, want.ModelFile = 3, 7, 42, "other.model"
	if c != want {
		t.Errorf("saved settings = %+v, want %+v", c, want)
	}
	if got := runMenu(t, "4", "0"); !strings.Contains(got, "Model file   other.model") {
		t.Errorf("new session did not use saved model file:\n%s", got)
	}
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
	for _, want := range []string{
		"1 - Training   (dry run is on: nothing will be saved)",
		"Dry run: would learn 1 new usernames",
		"Nothing was saved.",
		"Saved 3 rejected usernames, with reasons, to rejected_3.txt",
		"Nothing learned yet",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("menu output missing %q:\n%s", want, got)
		}
	}
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

func TestLoadSettings(t *testing.T) {
	t.Chdir(t.TempDir())
	if c, err := loadSettings("missing.json"); err != nil || c != defaultSettings() {
		t.Errorf("missing file: %+v, %v; want defaults", c, err)
	}
	writeFile(t, "partial.json", `{"edits_per_username": 4}`)
	if c, err := loadSettings("partial.json"); err != nil || c.EditsPerName != 4 || c.GenCount != defaultSettings().GenCount {
		t.Errorf("partial file: %+v, %v", c, err)
	}
	for name, body := range map[string]string{
		"garbage.json": "not json",
		"invalid.json": `{"generate_min_length": 20, "generate_max_length": 5}`,
	} {
		writeFile(t, name, body)
		if c, err := loadSettings(name); err == nil || c != defaultSettings() {
			t.Errorf("%s: %+v, %v; want defaults and an error", name, c, err)
		}
	}
	var out strings.Builder
	writeFile(t, settingsFile, "not json")
	if err := menu(strings.NewReader("0\n"), &out, settingsFile); err != nil || !strings.Contains(out.String(), "using default settings") {
		t.Errorf("menu with bad settings: %v\n%s", err, out.String())
	}
}
