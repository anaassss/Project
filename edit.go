package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/anaassss/Project/edit"
	"github.com/anaassss/Project/markov"
)

// Edit module: rewrite the usernames in a file using what the model learned.

const (
	defaultMaxEdits = 10

	// Editing reads its input twice: once to note every input username, then
	// to edit them. The second pass takes about 20 times as long per byte,
	// so the progress bar weights its bytes to match.
	readWeight = 1
	editWeight = 20
)

func editCommand(args []string) error {
	flags := flag.NewFlagSet("edit", flag.ExitOnError)
	modelPath := flags.String("model", defaultModel, "model file to edit with")
	maxEdits := flags.Int("max", defaultMaxEdits, "most edits per username; the model may find fewer")
	seed := flags.Uint64("seed", 0, "random seed for repeatable output (0 = random)")
	allowKnown := flags.Bool("allow-known", false, "allow edits that are usernames the model learned from")
	flags.Parse(args)
	if flags.NArg() != 1 {
		return errors.New("edit needs exactly one file of usernames")
	}

	m, err := loadModel(*modelPath)
	if err != nil {
		return err
	}
	opts := edit.Options{Max: *maxEdits, AllowKnown: *allowKnown}
	summary, err := editFile(m, flags.Arg(0), opts, *seed, ".", terminalOrNil(os.Stderr))
	if err != nil {
		return err
	}
	fmt.Println(summary)
	return nil
}

// editFile edits every username in path and saves the edits, one per line,
// to edited_<count>.txt in dir, drawing a progress bar on bar (nil for none).
// It returns a one-line summary.
func editFile(m *markov.Model, path string, opts edit.Options, seed uint64, dir string, bar io.Writer) (string, error) {
	e, err := edit.New(m, opts)
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}

	p := startProgress(bar, "Editing", info.Size()*(readWeight+editWeight))
	var out string
	var count int
	err = func() error {
		// Note every input so that no edit repeats one.
		seen := map[uint64]struct{}{}
		err := markov.ScanNames(p.reader(f, readWeight), func(name string) {
			seen[markov.FoldHash(name)] = struct{}{}
		})
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		if len(seen) == 0 {
			return fmt.Errorf("%s has no usernames", path)
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}

		tmp, err := os.CreateTemp(dir, ".edited-*.tmp")
		if err != nil {
			return err
		}
		defer os.Remove(tmp.Name()) // no-op once renamed
		count, err = e.Run(p.reader(f, editWeight), tmp, seen, orRandom(seed))
		if closeErr := tmp.Close(); err == nil {
			err = closeErr
		}
		if err != nil || count == 0 {
			return err
		}
		if err := os.Chmod(tmp.Name(), 0o644); err != nil {
			return err
		}
		out, err = placeEdited(tmp.Name(), dir, count)
		return err
	}()
	elapsed := p.end(err == nil)
	if err != nil {
		return "", err
	}
	return editSummary(out, count, elapsed), nil
}

func editSummary(path string, count int, elapsed time.Duration) string {
	if count == 0 {
		return "No edits found; train on more usernames like these first."
	}
	return fmt.Sprintf("Saved %s edited usernames to %s in %s", formatCount(count), path, formatDuration(elapsed))
}

// placeEdited moves tmp to edited_<count>.txt in dir. It never overwrites:
// if that file exists it uses edited_<count>_2.txt, edited_<count>_3.txt, ...
func placeEdited(tmp, dir string, count int) (string, error) {
	for i := 1; ; i++ {
		name := fmt.Sprintf("edited_%d.txt", count)
		if i > 1 {
			name = fmt.Sprintf("edited_%d_%d.txt", count, i)
		}
		path := filepath.Join(dir, name)
		_, err := os.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) {
			return path, os.Rename(tmp, path)
		}
		if err != nil {
			return "", err
		}
	}
}
