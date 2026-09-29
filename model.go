package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/anaassss/Project/markov"
)

// Model management shown in the menu: what the model knows, and erasing it.

const topShown = 10 // most common words and numbers listed by describeModel

// describeModel prints what m, saved at path, has learned.
func describeModel(w io.Writer, path string, m *markov.Model) {
	s := m.Stats()
	fmt.Fprintf(w, "  Model file   %s\n", fileLabel(path))
	fmt.Fprintf(w, "  Usernames    %s learned", formatCount(int(s.Names)))
	if s.Names > 0 {
		fmt.Fprintf(w, ", %d-%d characters (average %.1f)", s.MinLen, s.MaxLen, float64(s.TotalLen)/float64(s.Names))
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  Patterns     %s, using %d characters of context\n", formatCount(m.Patterns()), m.Order())
	words, numbers := m.Vocabulary()
	fmt.Fprintf(w, "  Words        %s%s\n", formatCount(words), mostCommonList(m.TopWords(topShown)))
	fmt.Fprintf(w, "  Numbers      %s%s\n", formatCount(numbers), mostCommonList(m.TopNumbers(topShown)))
}

// fileLabel names the file a model at path is actually read from, with its
// size: the old-format file when only that exists.
func fileLabel(path string) string {
	note := ""
	info, err := os.Stat(path)
	if err != nil && path == defaultModel {
		if info, err = os.Stat(legacyModel); err == nil {
			path, note = legacyModel, ", old format; converted on next training"
		}
	}
	if err != nil {
		return path
	}
	size := fmt.Sprintf("%.1f MB", float64(info.Size())/1e6)
	if info.Size() < 1e6 {
		size = fmt.Sprintf("%.1f KB", float64(info.Size())/1e3)
	}
	return fmt.Sprintf("%s (%s%s)", path, size, note)
}

func mostCommonList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return "; most common: " + strings.Join(items, ", ")
}

// clearModel erases everything learned by deleting the model at path. For
// the default path it also deletes the old-format model that loadModel
// would otherwise fall back to, so cleared knowledge can't reappear.
func clearModel(path string) error {
	paths := []string{path}
	if path == defaultModel {
		paths = append(paths, legacyModel)
	}
	for _, p := range paths {
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}
