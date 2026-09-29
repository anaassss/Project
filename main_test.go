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
	if err := menu(strings.NewReader(input), &out, "model"); err != nil {
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
	if _, err := os.Stat("model"); err != nil {
		t.Errorf("model not saved: %v", err)
	}
}

func TestMenuSurvivesErrorsAndEOF(t *testing.T) {
	t.Chdir(t.TempDir())
	var out strings.Builder
	// A missing file is reported and the menu continues; input then ends.
	if err := menu(strings.NewReader("1\nmissing.txt\n7\n"), &out, "model"); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); !strings.Contains(got, "Error: stat missing.txt") || !strings.Contains(got, "Choose 1, 2, 3, 4 or 0.") {
		t.Errorf("unexpected output:\n%s", got)
	}
}

func TestPlaceEditedNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	var got []string
	for i := range 3 {
		tmp := filepath.Join(dir, "tmp")
		writeFile(t, tmp, strings.Repeat("x", i))
		p, err := placeEdited(tmp, dir, 2)
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
		"3",       // info
		"4", "no", // clear, refused
		"4", "YES", // clear, confirmed
		"3", // info: nothing left
		"2", // edit: nothing left
		"0",
	}, "\n") + "\n"
	var out strings.Builder
	if err := menu(strings.NewReader(input), &out, defaultModel); err != nil {
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
