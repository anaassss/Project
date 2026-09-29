package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

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
		"2", "to\\ edit.txt", "abc", "4", // edit, invalid then 4 edits each
		"0",
	}, "\n") + "\n"
	var out strings.Builder
	if err := menu(strings.NewReader(input), &out, "model.json"); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{
		"Nothing learned yet",
		"learned 155 new usernames",
		"model.json now knows 155 usernames",
		"Enter a whole number above 0",
		"Saved 8 edited usernames to edited_8.txt",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("menu output missing %q:\n%s", want, got)
		}
	}

	edits := readLines(t, "edited_8.txt")
	if len(edits) != 8 {
		t.Fatalf("edited_8.txt has %d lines, want 8", len(edits))
	}
	for _, ed := range edits {
		if strings.EqualFold(ed, "ShadowFox") || strings.EqualFold(ed, "DarkWolf_7") {
			t.Errorf("edited file contains input name %q", ed)
		}
	}
}

func TestMenuSurvivesErrorsAndEOF(t *testing.T) {
	t.Chdir(t.TempDir())
	var out strings.Builder
	// A missing file is reported and the menu continues; input then ends.
	if err := menu(strings.NewReader("1\nmissing.txt\n7\n"), &out, "model.json"); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); !strings.Contains(got, "Error: open missing.txt") || !strings.Contains(got, `"7" is not an option`) {
		t.Errorf("unexpected output:\n%s", got)
	}
}

func TestWriteEditedNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	names := []string{"NightFox", "ShadowHawk"}
	var paths []string
	for range 3 {
		p, err := writeEdited(dir, names)
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, filepath.Base(p))
	}
	if want := []string{"edited_2.txt", "edited_2_2.txt", "edited_2_3.txt"}; !slices.Equal(paths, want) {
		t.Errorf("wrote %v, want %v", paths, want)
	}
	if got := readLines(t, filepath.Join(dir, "edited_2.txt")); !slices.Equal(got, names) {
		t.Errorf("edited_2.txt = %v, want %v", got, names)
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

// repoRoot is the package directory, captured before tests change directory.
var repoRoot = func() string {
	wd, _ := os.Getwd()
	return wd
}()
