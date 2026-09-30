package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/anaassss/Project/edit"
	"github.com/anaassss/Project/markov"
)

// session is the interactive menu's state. It keeps the model in memory
// between choices so it is only loaded once.
type session struct {
	r            *bufio.Reader
	out          io.Writer
	settingsPath string
	cfg          settings
	m            *markov.Model
}

// menu runs the interactive menu until the user exits or input ends. Problems
// with one choice (a missing file, say) are reported and the menu continues.
func menu(in io.Reader, out io.Writer, settingsPath string) error {
	cfg, err := loadSettings(settingsPath)
	if err != nil {
		fmt.Fprintf(out, "Warning: %v; using default settings.\n", err)
	}
	s := &session{r: bufio.NewReader(in), out: out, settingsPath: settingsPath, cfg: cfg}
	for {
		s.printMenu()
		choice, err := s.ask("> ")
		if errors.Is(err, io.EOF) {
			fmt.Fprintln(out)
			return nil
		}
		if err != nil {
			return err
		}
		switch strings.ToLower(choice) {
		case "1":
			err = s.train()
		case "2":
			err = s.edit()
		case "3":
			err = s.generate()
		case "4":
			err = s.info()
		case "5":
			err = s.settingsScreen()
		case "6":
			err = s.clear()
		case "0", "q", "exit", "quit":
			return nil
		default:
			fmt.Fprintln(out, "Choose 1-6, or 0 to exit.")
			continue
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			fmt.Fprintln(out, "Error:", err)
		}
	}
}

func (s *session) printMenu() {
	training := "Training"
	if s.cfg.DryRun {
		training += "   (dry run is on: nothing will be saved)"
	}
	fmt.Fprintf(s.out, `
  1 - %s
  2 - Edit
  3 - Generate
  4 - Model info
  5 - Settings
  6 - Clear all knowledge
  0 - Exit
`, training)
}

// loadQuiet returns the model, loading it if needed, or nil if nothing has
// been learned yet.
func (s *session) loadQuiet() (*markov.Model, error) {
	if s.m == nil {
		m, err := loadModel(s.cfg.ModelFile)
		if errors.Is(err, errNoModel) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		s.m = m
	}
	return s.m, nil
}

// model loads the model if needed. It reports false, after telling the
// user, if nothing has been learned yet.
func (s *session) model() (bool, error) {
	m, err := s.loadQuiet()
	if err != nil {
		return false, err
	}
	if m == nil {
		fmt.Fprintln(s.out, "Nothing learned yet. Choose 1 - Training first.")
		return false, nil
	}
	return true, nil
}

func (s *session) train() error {
	path, err := s.askPath("File: ")
	if err != nil {
		return err
	}
	if s.m == nil {
		if s.m, _, err = loadOrCreate(s.cfg.ModelFile, s.cfg.Order); err != nil {
			return err
		}
	}

	save := s.cfg.ModelFile
	if s.cfg.DryRun {
		save = ""
	}
	var rejected *numberedFile
	var onRejected func(name, reason string)
	if s.cfg.SaveRejected {
		if rejected, err = newNumberedFile(".", "rejected"); err != nil {
			return err
		}
		defer rejected.discard()
		onRejected = func(name, reason string) { rejected.writeLine(name + "\t" + reason) }
	}

	summary, err := trainFiles(s.m, []string{path}, save, onRejected, s.out)
	if err != nil || s.cfg.DryRun {
		s.m = nil // reload from disk next time rather than keep unsaved learning
	}
	if err != nil {
		return err
	}
	if s.cfg.DryRun {
		summary = strings.Replace(summary, "Learned", "Dry run: would learn", 1) + ". Nothing was saved."
	}
	fmt.Fprintln(s.out, summary)
	if rejected != nil && rejected.lines > 0 {
		out, err := rejected.keep()
		if err != nil {
			return err
		}
		fmt.Fprintf(s.out, "Saved %s rejected usernames, with reasons, to %s\n", formatCount(rejected.lines), out)
	}
	return nil
}

func (s *session) edit() error {
	if ok, err := s.model(); !ok {
		return err
	}
	minLen, err := s.askInt("Shortest length", s.cfg.EditMinLen, markov.MinNameLen, markov.MaxNameLen)
	if err != nil {
		return err
	}
	maxLen, err := s.askInt("Longest length", max(s.cfg.EditMaxLen, minLen), minLen, markov.MaxNameLen)
	if err != nil {
		return err
	}
	if minLen != s.cfg.EditMinLen || maxLen != s.cfg.EditMaxLen {
		next := s.cfg
		next.EditMinLen, next.EditMaxLen = minLen, maxLen
		if err := next.save(s.settingsPath); err != nil {
			return fmt.Errorf("saving settings: %w", err)
		}
		s.cfg = next
	}
	path, err := s.askPath("File: ")
	if err != nil {
		return err
	}
	opts := edit.Options{Max: s.cfg.EditsPerName, AllowKnown: s.cfg.AllowKnown, MinLen: minLen, MaxLen: maxLen}
	summary, err := editFile(s.m, path, opts, s.cfg.Seed, ".", s.out)
	if err != nil {
		return err
	}
	fmt.Fprintln(s.out, summary)
	return nil
}

func (s *session) generate() error {
	if ok, err := s.model(); !ok {
		return err
	}
	c := s.cfg
	seed := orRandom(c.Seed)
	p := startProgress(s.out, "Generating", int64(c.GenCount))
	names, err := s.m.Generate(rand.New(rand.NewPCG(seed, seed)), c.GenCount, markov.GenerateOptions{
		MinLen:      c.GenMinLen,
		MaxLen:      c.GenMaxLen,
		Temperature: c.GenTemp,
		Order:       c.GenOrder,
		AllowKnown:  c.AllowKnown,
		Progress:    func(found int) { p.done.Store(int64(found)) },
	})
	elapsed := p.end(err == nil)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		fmt.Fprintln(s.out, "No new usernames found. Train on more usernames, or raise Creativity in Settings.")
		return nil
	}

	f, err := newNumberedFile(".", "generated")
	if err != nil {
		return err
	}
	defer f.discard()
	for _, name := range names {
		f.writeLine(name)
	}
	out, err := f.keep()
	if err != nil {
		return err
	}
	fmt.Fprintf(s.out, "Saved %s new usernames to %s in %s\n", formatCount(len(names)), out, formatDuration(elapsed))
	fmt.Fprintf(s.out, "  %s", strings.Join(names[:min(10, len(names))], ", "))
	if len(names) > 10 {
		fmt.Fprint(s.out, ", ...")
	}
	fmt.Fprintln(s.out)
	if len(names) < c.GenCount {
		fmt.Fprintf(s.out, "Only %s of %s found: train on more usernames, or raise Creativity or lower Context length in Settings.\n",
			formatCount(len(names)), formatCount(c.GenCount))
	}
	return nil
}

func (s *session) info() error {
	if ok, err := s.model(); !ok {
		return err
	}
	describeModel(s.out, s.cfg.ModelFile, s.m)
	return nil
}

func (s *session) clear() error {
	if ok, err := s.model(); !ok {
		return err
	}
	answer, err := s.ask(fmt.Sprintf("This erases everything learned (%s usernames) and cannot be undone. Type yes to confirm: ",
		formatCount(int(s.m.Stats().Names))))
	if err != nil {
		return err
	}
	if strings.ToLower(answer) != "yes" {
		fmt.Fprintln(s.out, "Nothing cleared.")
		return nil
	}
	if err := clearModel(s.cfg.ModelFile); err != nil {
		return err
	}
	s.m = nil
	fmt.Fprintln(s.out, "All knowledge cleared.")
	return nil
}

// ask prints prompt and returns the trimmed line typed. It returns io.EOF only
// when input has ended with nothing typed.
func (s *session) ask(prompt string) (string, error) {
	fmt.Fprint(s.out, prompt)
	line, err := s.r.ReadString('\n')
	line = strings.TrimSpace(line)
	if errors.Is(err, io.EOF) && line != "" {
		fmt.Fprintln(s.out)
		err = nil
	}
	return line, err
}

// askInt asks for a whole number from lo to hi, returning current if the
// user just presses Enter.
func (s *session) askInt(label string, current, lo, hi int) (int, error) {
	for {
		answer, err := s.ask(fmt.Sprintf("%s (%d-%d) [%d]: ", label, lo, hi, current))
		if err != nil {
			return 0, err
		}
		if answer == "" {
			return current, nil
		}
		if n, err := strconv.Atoi(answer); err == nil && n >= lo && n <= hi {
			return n, nil
		}
		fmt.Fprintf(s.out, "Enter a whole number from %d to %d.\n", lo, hi)
	}
}

// askPath asks for a file path, accepting forms terminals produce when a file
// is dragged in: quoted, with escaped spaces, or starting with ~/.
func (s *session) askPath(prompt string) (string, error) {
	for {
		answer, err := s.ask(prompt)
		if err != nil {
			return "", err
		}
		if path := cleanPath(answer); path != "" {
			return path, nil
		}
	}
}

func cleanPath(p string) string {
	if len(p) >= 2 && (p[0] == '"' || p[0] == '\'') && p[len(p)-1] == p[0] {
		p = p[1 : len(p)-1]
	} else {
		p = strings.ReplaceAll(p, `\ `, " ")
	}
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, rest)
		}
	}
	return p
}
