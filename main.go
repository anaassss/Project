// Command usergen learns how usernames are built from example lists and uses
// that knowledge to edit usernames you give it into new ones. What it learns
// is saved to a model file, and every training run adds to the existing model
// instead of starting over. Both scale to millions of usernames.
//
// Run it without arguments for an interactive menu with its two modules,
// Training and Edit, or use the subcommands below from scripts.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"

	"github.com/anaassss/Project/markov"
)

const (
	defaultModel = "usergen.model"
	legacyModel  = "usergen.json" // the default before models became binary
	defaultOrder = 3
)

const usage = `usergen learns how usernames are built and edits usernames into new ones.

Usage:
  usergen                            interactive menu with every feature and its settings
  usergen train    [flags] FILE...   learn from username lists (one per line, "-" for stdin)
  usergen edit     [flags] FILE      edit each username in FILE; saves edited_<count>.txt
  usergen generate [flags]           generate new usernames from the saved model
  usergen stats    [flags]           show what the saved model has learned

Garbage usernames (emails, IDs, keyboard mashes, placeholders like [deleted],
and so on) are never learned and never generated.

Run "usergen <command> -h" for a command's flags.
`

func main() {
	if len(os.Args) < 2 {
		if err := menu(os.Stdin, os.Stdout, settingsFile); err != nil {
			fmt.Fprintln(os.Stderr, "usergen:", err)
			os.Exit(1)
		}
		return
	}
	var err error
	switch cmd, args := os.Args[1], os.Args[2:]; cmd {
	case "train":
		err = train(args)
	case "edit":
		err = editCommand(args)
	case "generate", "gen":
		err = generate(args)
	case "stats":
		err = stats(args)
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

// errNoModel reports that nothing has been learned yet.
var errNoModel = errors.New("nothing learned yet")

// loadModel loads the model at path, returning an error wrapping errNoModel
// if it does not exist. A model saved under the old default name is used if
// the default one does not exist yet.
func loadModel(path string) (*markov.Model, error) {
	m, err := markov.Load(path)
	if errors.Is(err, fs.ErrNotExist) && path == defaultModel {
		m, err = markov.Load(legacyModel)
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: no model at %s; train on a file of usernames first", errNoModel, path)
	}
	return m, err
}

// loadOrCreate loads the model at path, or returns a new one of the given
// order and true if none exists.
func loadOrCreate(path string, order int) (*markov.Model, bool, error) {
	m, err := loadModel(path)
	if errors.Is(err, errNoModel) {
		m, err = markov.New(order)
		return m, true, err
	}
	return m, false, err
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
	s := orRandom(*seed)
	names, err := m.Generate(rand.New(rand.NewPCG(s, s)), *n, markov.GenerateOptions{
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
	describeModel(os.Stdout, *modelPath, m)
	return nil
}

// orRandom returns seed, or a random one if seed is 0.
func orRandom(seed uint64) uint64 {
	if seed == 0 {
		return rand.Uint64()
	}
	return seed
}
