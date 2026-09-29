package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/anaassss/Project/edit"
)

const menuText = `
What do you want to do?
  1 - Training   learn from a file of usernames
  2 - Edit       edit the usernames in a file using what was learned
  0 - Exit
`

func menuCommand(args []string) error {
	flags := flag.NewFlagSet("menu", flag.ExitOnError)
	modelPath := flags.String("model", defaultModel, "model file to train and edit with")
	flags.Parse(args)
	return menu(os.Stdin, os.Stdout, *modelPath)
}

// menu runs the interactive menu until the user exits or input ends. Problems
// with one choice (a missing file, say) are reported and the menu continues.
func menu(in io.Reader, out io.Writer, modelPath string) error {
	r := bufio.NewReader(in)
	fmt.Fprintf(out, "usergen: learns how usernames are built (model: %s)\n", modelPath)
	for {
		fmt.Fprint(out, menuText)
		choice, err := ask(r, out, "Choose 1, 2 or 0: ")
		if errors.Is(err, io.EOF) {
			fmt.Fprintln(out)
			return nil
		}
		if err != nil {
			return err
		}
		switch strings.ToLower(choice) {
		case "1", "training", "train":
			err = menuTrain(r, out, modelPath)
		case "2", "edit":
			err = menuEdit(r, out, modelPath)
		case "0", "exit", "quit", "q":
			return nil
		default:
			fmt.Fprintf(out, "%q is not an option.\n", choice)
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

func menuTrain(r *bufio.Reader, out io.Writer, modelPath string) error {
	path, err := askPath(r, out, "Usernames file to learn from: ")
	if err != nil {
		return err
	}
	m, _, err := loadOrCreate(modelPath, defaultOrder)
	if err != nil {
		return err
	}
	res, err := learnFiles(m, []string{path}, nil)
	if err != nil {
		return err
	}
	res.print(out, false)
	if err := m.Save(modelPath); err != nil {
		return fmt.Errorf("saving model: %w", err)
	}
	fmt.Fprintf(out, "%s now knows %d usernames\n", modelPath, len(m.Names()))
	return nil
}

func menuEdit(r *bufio.Reader, out io.Writer, modelPath string) error {
	m, err := loadModel(modelPath)
	if errors.Is(err, errNoModel) {
		fmt.Fprintln(out, "Nothing learned yet. Choose 1 - Training first.")
		return nil
	}
	if err != nil {
		return err
	}
	path, err := askPath(r, out, "Usernames file to edit: ")
	if err != nil {
		return err
	}
	maxEdits := defaultMaxEdits
	for {
		answer, err := ask(r, out, fmt.Sprintf("Most edits per username [%d]: ", defaultMaxEdits))
		if err != nil {
			return err
		}
		if answer == "" {
			break
		}
		if n, err := strconv.Atoi(answer); err == nil && n > 0 {
			maxEdits = n
			break
		}
		fmt.Fprintln(out, "Enter a whole number above 0, or press Enter for the default.")
	}
	_, err = editFile(out, m, path, edit.Options{Max: maxEdits}, 0, ".")
	return err
}

// ask prints prompt and returns the trimmed line typed. It returns io.EOF only
// when input has ended with nothing typed.
func ask(r *bufio.Reader, out io.Writer, prompt string) (string, error) {
	fmt.Fprint(out, prompt)
	line, err := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if errors.Is(err, io.EOF) && line != "" {
		fmt.Fprintln(out)
		err = nil
	}
	return line, err
}

// askPath asks for a file path, accepting forms terminals produce when a file
// is dragged in: quoted, with escaped spaces, or starting with ~/.
func askPath(r *bufio.Reader, out io.Writer, prompt string) (string, error) {
	for {
		answer, err := ask(r, out, prompt)
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
