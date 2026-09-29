// Command usergen learns how usernames are built from example lists and
// generates new ones. What it learns is saved to a model file, and every
// training run adds to the existing model instead of starting over.
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
	"sort"
	"strings"

	"github.com/anaassss/Project/markov"
)

const defaultModel = "usergen.json"

const usage = `usergen learns how usernames are built and generates new ones.

Usage:
  usergen train    [flags] FILE...   learn from username lists (one per line, "-" for stdin)
  usergen generate [flags]           generate new usernames from the saved model
  usergen stats    [flags]           show what the saved model has learned
  usergen clean    [flags]           remove garbage usernames from the saved model

Garbage usernames (emails, IDs, keyboard mashes, placeholders like [deleted],
and so on) are never learned and never generated.

Run "usergen <command> -h" for a command's flags.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch cmd, args := os.Args[1], os.Args[2:]; cmd {
	case "train":
		err = train(args)
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

func train(args []string) error {
	flags := flag.NewFlagSet("train", flag.ExitOnError)
	modelPath := flags.String("model", defaultModel, "model file to load and save")
	order := flags.Int("order", 3, "characters of context to learn (only when creating a new model)")
	showRejected := flags.Bool("show-rejected", false, "list every garbage username skipped, with the reason")
	dryRun := flags.Bool("dry-run", false, "report what would be learned without saving the model")
	flags.Parse(args)
	if flags.NArg() == 0 {
		return errors.New("train needs at least one file of usernames (or - for stdin)")
	}
	orderSet := false
	flags.Visit(func(f *flag.Flag) { orderSet = orderSet || f.Name == "order" })

	m, created, err := loadOrCreate(*modelPath, *order)
	if err != nil {
		return err
	}
	if !created && orderSet && *order != m.Order() {
		return fmt.Errorf("%s was trained with order %d; use a different -model to train with order %d",
			*modelPath, m.Order(), *order)
	}

	var added, known int
	rejected := map[string]int{}
	for _, path := range flags.Args() {
		err := eachName(path, func(name string) {
			ok, err := m.Learn(name)
			var garbage *markov.GarbageError
			switch {
			case errors.As(err, &garbage):
				rejected[garbage.Reason]++
				if *showRejected {
					fmt.Fprintf(os.Stderr, "rejected %q: %s\n", name, garbage.Reason)
				}
			case ok:
				added++
			default:
				known++
			}
		})
		if err != nil {
			return err
		}
	}

	total := 0
	for _, n := range rejected {
		total += n
	}
	verb := "learned"
	if *dryRun {
		verb = "would learn"
	}
	fmt.Printf("%s %d new usernames (%d already known, %d garbage rejected)\n", verb, added, known, total)
	printReasons(rejected)
	if *dryRun {
		fmt.Println("dry run: model not saved")
		return nil
	}
	if err := m.Save(*modelPath); err != nil {
		return fmt.Errorf("saving model: %w", err)
	}
	fmt.Printf("%s now knows %d usernames\n", *modelPath, len(m.Names()))
	return nil
}

// printReasons lists rejection reasons, most common first.
func printReasons(counts map[string]int) {
	reasons := make([]string, 0, len(counts))
	for r := range counts {
		reasons = append(reasons, r)
	}
	sort.Slice(reasons, func(i, j int) bool {
		if counts[reasons[i]] != counts[reasons[j]] {
			return counts[reasons[i]] > counts[reasons[j]]
		}
		return reasons[i] < reasons[j]
	})
	for _, r := range reasons {
		fmt.Printf("  %6d  %s\n", counts[r], r)
	}
}

func clean(args []string) error {
	flags := flag.NewFlagSet("clean", flag.ExitOnError)
	modelPath := flags.String("model", defaultModel, "model file to clean")
	flags.Parse(args)

	m, err := markov.Load(*modelPath)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("no model at %s; run \"usergen train FILE\" first", *modelPath)
	}
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

	m, err := markov.Load(*modelPath)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("no model at %s; run \"usergen train FILE\" first", *modelPath)
	}
	if err != nil {
		return err
	}

	s := *seed
	if s == 0 {
		s = rand.Uint64()
	}
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

	m, err := markov.Load(*modelPath)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("no model at %s; run \"usergen train FILE\" first", *modelPath)
	}
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
