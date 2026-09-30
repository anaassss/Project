package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/anaassss/Project/knowledge"
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
	fmt.Fprintf(w, "  Words        %s%s\n", formatCount(words), mostCommonList(topWords(m, topShown)))
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

// topWords returns m's n most common words, leaving out affixes like xx
// and the, which frame names but say little about them.
func topWords(m *markov.Model, n int) []string {
	words := slices.DeleteFunc(m.TopWords(4*n), markov.IsAffix)
	return words[:min(n, len(words))]
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

// Knowledge choices for Edit and Generate.
const (
	ownKnowledge    = "own"
	claudeKnowledge = "claude"
	bothKnowledge   = "both"
)

var knowledgeChoices = []string{ownKnowledge, claudeKnowledge, bothKnowledge}

var errNoOwnKnowledge = errors.New("you haven't trained your own knowledge yet")

// knowledgeFor returns the models to use for choice, given the user's own
// model (nil if nothing is trained). For both without own knowledge it
// uses Claude's alone and returns a note saying so.
func knowledgeFor(choice string, own *markov.Model) ([]*markov.Model, string, error) {
	switch choice {
	case ownKnowledge:
		if own == nil {
			return nil, "", errNoOwnKnowledge
		}
		return []*markov.Model{own}, "", nil
	case claudeKnowledge:
		return []*markov.Model{knowledge.Model()}, "", nil
	case bothKnowledge:
		if own == nil {
			return []*markov.Model{knowledge.Model()}, "No own knowledge yet, so using Claude's.", nil
		}
		return []*markov.Model{own, knowledge.Model()}, "", nil
	}
	return nil, "", fmt.Errorf("unknown knowledge %q: use own, claude or both", choice)
}

// ownOrNil loads the user's model, or returns nil if nothing is trained.
func ownOrNil(path string) (*markov.Model, error) {
	m, err := loadModel(path)
	if errors.Is(err, errNoModel) {
		return nil, nil
	}
	return m, err
}

// parseEdits reads edits per username: a number from 1 to maxEditsPerName,
// or "max" for every edit the knowledge allows, returned as 0.
func parseEdits(s string) (int, error) {
	if strings.EqualFold(s, "max") {
		return 0, nil
	}
	n, err := strconv.Atoi(strings.ReplaceAll(s, ",", ""))
	if err != nil || n < 1 || n > maxEditsPerName {
		return 0, fmt.Errorf("enter a number from 1 to %d, or max", maxEditsPerName)
	}
	return n, nil
}

func editsLabel(n int) string {
	if n == 0 {
		return "max"
	}
	return strconv.Itoa(n)
}

// describeClaude prints what Claude's built-in knowledge contains.
func describeClaude(w io.Writer) {
	m := knowledge.Model()
	words, _ := m.Vocabulary()
	fmt.Fprintf(w, "  Email-style  %s usernames, %s words%s\n",
		formatCount(int(m.Stats().Names)), formatCount(words), mostCommonList(topWords(m, 6)))
}

// generateFrom generates up to count distinct usernames, sharing the count
// between models. With unique, they are lowercase and knowledge.Unique, as
// edit.Options.Unique makes edits. progress, if non-nil, is called with the
// total found.
func generateFrom(models []*markov.Model, count int, opts markov.GenerateOptions, unique bool, seed uint64, progress func(int)) ([]string, error) {
	if unique {
		opts.Keep = knowledge.Unique
	}
	rng := rand.New(rand.NewPCG(seed, seed))
	var names []string
	seen := map[uint64]bool{}
	for i, m := range models {
		n := count / len(models)
		if i < count%len(models) {
			n++
		}
		if n == 0 {
			continue
		}
		o := opts
		o.Order = min(o.Order, m.Order())
		base := len(names)
		if progress != nil {
			o.Progress = func(found int) { progress(base + found) }
		}
		got, err := m.Generate(rng, n, o)
		if err != nil {
			return nil, err
		}
		for _, name := range got {
			h := markov.FoldHash(name)
			known := !opts.AllowKnown && slices.ContainsFunc(models, func(m *markov.Model) bool { return m.KnowsHash(h) })
			if !seen[h] && !known {
				seen[h] = true
				if unique {
					name = strings.ToLower(name)
				}
				names = append(names, name)
			}
		}
	}
	return names, nil
}
