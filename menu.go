package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/anaassss/Project/edit"
	"github.com/anaassss/Project/markov"
)

const menuText = `
  1 - Training
  2 - Edit
  3 - Model info
  4 - Clear all knowledge
  0 - Exit
`

func menuCommand(args []string) error {
	flags := flag.NewFlagSet("menu", flag.ExitOnError)
	modelPath := flags.String("model", defaultModel, "model file to train and edit with")
	flags.Parse(args)
	return menu(os.Stdin, os.Stdout, *modelPath)
}

// session is the interactive menu's state. It keeps the model in memory
// between choices so it is only loaded once.
type session struct {
	r         *bufio.Reader
	out       io.Writer
	modelPath string
	m         *markov.Model
}

// menu runs the interactive menu until the user exits or input ends. Problems
// with one choice (a missing file, say) are reported and the menu continues.
func menu(in io.Reader, out io.Writer, modelPath string) error {
	s := &session{r: bufio.NewReader(in), out: out, modelPath: modelPath}
	for {
		fmt.Fprint(out, menuText)
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
			err = s.info()
		case "4":
			err = s.clear()
		case "0", "q", "exit", "quit":
			return nil
		default:
			fmt.Fprintln(out, "Choose 1, 2, 3, 4 or 0.")
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

func (s *session) train() error {
	path, err := s.askPath("File: ")
	if err != nil {
		return err
	}
	if s.m == nil {
		if s.m, _, err = loadOrCreate(s.modelPath, defaultOrder); err != nil {
			return err
		}
	}
	summary, err := trainFiles(s.m, []string{path}, s.modelPath, nil, s.out)
	if err != nil {
		s.m = nil // reload from disk next time rather than trust a partial run
		return err
	}
	fmt.Fprintln(s.out, summary)
	return nil
}

// model loads the model if needed. It reports false, after telling the
// user, if nothing has been learned yet.
func (s *session) model() (bool, error) {
	if s.m == nil {
		m, err := loadModel(s.modelPath)
		if errors.Is(err, errNoModel) {
			fmt.Fprintln(s.out, "Nothing learned yet. Choose 1 - Training first.")
			return false, nil
		}
		if err != nil {
			return false, err
		}
		s.m = m
	}
	return true, nil
}

func (s *session) edit() error {
	if ok, err := s.model(); !ok {
		return err
	}
	path, err := s.askPath("File: ")
	if err != nil {
		return err
	}
	summary, err := editFile(s.m, path, edit.Options{Max: defaultMaxEdits}, 0, ".", s.out)
	if err != nil {
		return err
	}
	fmt.Fprintln(s.out, summary)
	return nil
}

func (s *session) info() error {
	if ok, err := s.model(); !ok {
		return err
	}
	describeModel(s.out, s.modelPath, s.m)
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
	if err := clearModel(s.modelPath); err != nil {
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
