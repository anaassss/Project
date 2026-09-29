package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/anaassss/Project/edit"
	"github.com/anaassss/Project/markov"
)

// Edit module: rewrite the usernames in a file using what the model learned.

const defaultMaxEdits = 10

func editCommand(args []string) error {
	flags := flag.NewFlagSet("edit", flag.ExitOnError)
	modelPath := flags.String("model", defaultModel, "model file to edit with")
	maxEdits := flags.Int("max", defaultMaxEdits, "most edits per username; the model may find fewer")
	seed := flags.Uint64("seed", 0, "random seed for repeatable output (0 = random)")
	allowKnown := flags.Bool("allow-known", false, "allow edits that are usernames the model learned from")
	flags.Parse(args)
	if flags.NArg() != 1 {
		return errors.New("edit needs exactly one file of usernames (or - for stdin)")
	}

	m, err := loadModel(*modelPath)
	if err != nil {
		return err
	}
	opts := edit.Options{Max: *maxEdits, AllowKnown: *allowKnown}
	_, err = editFile(os.Stdout, m, flags.Arg(0), opts, *seed, ".")
	return err
}

// editFile edits every username in path, printing each one's edits to w, and
// saves all edits, one per line, to edited_<count>.txt in dir. It returns the
// file written, or "" if the model found no edits.
func editFile(w io.Writer, m *markov.Model, path string, opts edit.Options, seed uint64, dir string) (string, error) {
	var inputs []string
	isInput := map[string]bool{}
	err := eachName(path, func(name string) {
		if key := strings.ToLower(name); !isInput[key] {
			isInput[key] = true
			inputs = append(inputs, name)
		}
	})
	if err != nil {
		return "", err
	}
	if len(inputs) == 0 {
		return "", fmt.Errorf("%s has no usernames", path)
	}

	e, err := edit.New(m, newRand(seed), opts)
	if err != nil {
		return "", err
	}

	width := 0
	for _, name := range inputs {
		width = max(width, utf8.RuneCountInString(name))
	}
	width = min(width, markov.MaxNameLen)

	var all []string
	written := map[string]bool{}
	for _, name := range inputs {
		var edits []string
		for _, ed := range e.Edit(name) {
			// Skip edits another input already produced, or that are inputs.
			if key := strings.ToLower(ed); !written[key] && !isInput[key] {
				written[key] = true
				edits = append(edits, ed)
			}
		}
		all = append(all, edits...)

		summary := strings.Join(edits, ", ")
		if len(edits) == 0 {
			summary = "(nothing the model learned fits this name)"
		}
		fmt.Fprintf(w, "  %-*s %3d  %s\n", width, name, len(edits), summary)
	}

	if len(all) == 0 {
		fmt.Fprintln(w, "No edits found; train on more usernames like these first.")
		return "", nil
	}
	out, err := writeEdited(dir, all)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(w, "Saved %d edited usernames to %s\n", len(all), out)
	return out, nil
}

// writeEdited saves names to edited_<count>.txt in dir. It never overwrites:
// if that file exists it uses edited_<count>_2.txt, edited_<count>_3.txt, ...
func writeEdited(dir string, names []string) (string, error) {
	for i := 1; ; i++ {
		base := fmt.Sprintf("edited_%d.txt", len(names))
		if i > 1 {
			base = fmt.Sprintf("edited_%d_%d.txt", len(names), i)
		}
		path := filepath.Join(dir, base)
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		bw := bufio.NewWriter(f)
		for _, name := range names {
			bw.WriteString(name + "\n")
		}
		if err := bw.Flush(); err != nil {
			f.Close()
			return "", err
		}
		return path, f.Close()
	}
}
