// Command usergen learns how usernames are built from example lists and uses
// that knowledge to edit usernames you give it into new ones. What it learns
// is saved to a model file, and every training run adds to the existing model
// instead of starting over.
//
// Run it without arguments for an interactive menu with its two modules,
// Training and Edit, or use the subcommands below from scripts.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"os"
	"strings"

	"github.com/anaassss/Project/markov"
)

const (
	defaultModel = "usergen.json"
	defaultOrder = 3
)

const usage = `usergen learns how usernames are built and edits usernames into new ones.

Usage:
  usergen                            interactive menu: 1 - Training, 2 - Edit
  usergen train    [flags] FILE...   learn from username lists (one per line, "-" for stdin)
  usergen edit     [flags] FILE      edit each username in FILE; saves edited_<count>.txt
  usergen generate [flags]           generate new usernames from the saved model
  usergen stats    [flags]           show what the saved model has learned
  usergen clean    [flags]           remove garbage usernames from the saved model

Garbage usernames (emails, IDs, keyboard mashes, placeholders like [deleted],
and so on) are never learned and never generated.

Run "usergen <command> -h" for a command's flags.
`

func main() {
	if len(os.Args) < 2 {
		if err := menu(os.Stdin, os.Stdout, defaultModel); err != nil {
			fmt.Fprintln(os.Stderr, "usergen:", err)
			os.Exit(1)
		}
		return
	}
	var err error
	switch cmd, args := os.Args[1], os.Args[2:]; cmd {
	case "menu":
		err = menuCommand(args)
	case "train":
		err = train(args)
	case "edit":
		err = editCommand(args)
	case "generate", "gen":
		err = generate(args)
	case "stats":
		err = stats(args)
	case "clean":
		err = clean(args)
	case "-h", "-help", "--help", "help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "usergen:", err)
		os.Exit(1)
	}
}

func clean(args []string) error {
	flags := flag.NewFlagSet("clean", flag.ExitOnError)
	modelPath := flags.String("model", defaultModel, "model file to clean")
	flags.Parse(args)

	m, err := loadModel(*modelPath)
	if err != nil {
		return err
	}
	dropped := m.Clean()
	if len(dropped) == 0 {
		fmt.Printf("%s has no garbage usernames\n", *modelPath)
		return nil
	}
	for _, name := range dropped {
		fmt.Fprintf(os.Stderr, "removed %q: %s\n", name, reason(name))
	}
	if err := m.Save(*modelPath); err != nil {
		return fmt.Errorf("saving model: %w", err)
	}
	fmt.Printf("removed %d garbage usernames; %s now knows %d usernames\n", len(dropped), *modelPath, len(m.Names()))
	return nil
}

func reason(name string) string {
	var garbage *markov.GarbageError
	if errors.As(markov.Check(name), &garbage) {
		return garbage.Reason
	}
	return "garbage"
}

// errNoModel reports that nothing has been learned yet.
var errNoModel = errors.New("nothing learned yet")

// loadModel loads the model at path, returning an error wrapping errNoModel
// if it does not exist.
func loadModel(path string) (*markov.Model, error) {
	m, err := markov.Load(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: no model at %s; train on a file of usernames first", errNoModel, path)
	}
	return m, err
}

func loadOrCreate(path string, order int) (*markov.Model, bool, error) {
	m, err := markov.Load(path)
	if err == nil {
		return m, false, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, false, err
	}
	m, err = markov.New(order)
	return m, true, err
}

// eachName calls fn for every non-blank, non-comment line in path.
func eachName(path string, fn func(string)) error {
	var r io.Reader = os.Stdin
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fn(line)
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	return nil
}

func generate(args []string) error {
	flags := flag.NewFlagSet("generate", flag.ExitOnError)
	modelPath := flags.String("model", defaultModel, "model file to generate from")
	n := flags.Int("n", 10, "number of usernames to generate")
	minLen := flags.Int("min", 4, "minimum username length")
	maxLen := flags.Int("max", 16, "maximum username length")
	temp := flags.Float64("temp", 1.0, "creativity: below 1 is safer, above 1 is wilder")
	order := flags.Int("order", 0, "characters of context to use, up to the model's order; lower is wilder (0 = model's order)")
	seed := flags.Uint64("seed", 0, "random seed for repeatable output (0 = random)")
	allowKnown := flags.Bool("allow-known", false, "allow usernames that appear in the training data")
	flags.Parse(args)

	m, err := loadModel(*modelPath)
	if err != nil {
		return err
	}

	names, err := m.Generate(newRand(*seed), *n, markov.GenerateOptions{
		MinLen:      *minLen,
		MaxLen:      *maxLen,
		Temperature: *temp,
		Order:       *order,
		AllowKnown:  *allowKnown,
	})
	if err != nil {
		return err
	}
	for _, name := range names {
		fmt.Println(name)
	}
	if len(names) < *n {
		fmt.Fprintf(os.Stderr, "usergen: only found %d new usernames; train on more names, lower -order, or raise -temp\n", len(names))
	}
	return nil
}

func stats(args []string) error {
	flags := flag.NewFlagSet("stats", flag.ExitOnError)
	modelPath := flags.String("model", defaultModel, "model file to inspect")
	flags.Parse(args)

	m, err := loadModel(*modelPath)
	if err != nil {
		return err
	}

	names := m.Names()
	total, shortest, longest := 0, 0, 0
	for i, name := range names {
		l := len([]rune(name))
		total += l
		if i == 0 || l < shortest {
			shortest = l
		}
		if l > longest {
			longest = l
		}
	}
	fmt.Printf("model:     %s\n", *modelPath)
	fmt.Printf("order:     %d characters of context\n", m.Order())
	fmt.Printf("usernames: %d learned\n", len(names))
	fmt.Printf("contexts:  %d distinct patterns\n", m.Contexts())
	if len(names) > 0 {
		fmt.Printf("length:    %d-%d characters, average %.1f\n", shortest, longest, float64(total)/float64(len(names)))
	}
	return nil
}

// newRand returns a generator seeded with seed, or randomly if seed is 0.
func newRand(seed uint64) *rand.Rand {
	if seed == 0 {
		seed = rand.Uint64()
	}
	return rand.New(rand.NewPCG(seed, seed))
}
