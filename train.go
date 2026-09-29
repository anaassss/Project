package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/anaassss/Project/markov"
)

// Training module: teach the model from files of usernames.

func train(args []string) error {
	flags := flag.NewFlagSet("train", flag.ExitOnError)
	modelPath := flags.String("model", defaultModel, "model file to load and save")
	order := flags.Int("order", defaultOrder, "characters of context to learn (only when creating a new model)")
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

	var rejectedOut io.Writer
	if *showRejected {
		rejectedOut = os.Stderr
	}
	res, err := learnFiles(m, flags.Args(), rejectedOut)
	if err != nil {
		return err
	}
	res.print(os.Stdout, *dryRun)
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

// trainResult tallies one training run.
type trainResult struct {
	added, known int
	rejected     map[string]int // garbage reason → count
}

// learnFiles teaches m every username in paths. If rejectedOut is non-nil,
// each garbage username is listed there with the reason.
func learnFiles(m *markov.Model, paths []string, rejectedOut io.Writer) (trainResult, error) {
	res := trainResult{rejected: map[string]int{}}
	for _, path := range paths {
		err := eachName(path, func(name string) {
			ok, err := m.Learn(name)
			var garbage *markov.GarbageError
			switch {
			case errors.As(err, &garbage):
				res.rejected[garbage.Reason]++
				if rejectedOut != nil {
					fmt.Fprintf(rejectedOut, "rejected %q: %s\n", name, garbage.Reason)
				}
			case ok:
				res.added++
			default:
				res.known++
			}
		})
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

// print summarises the run, listing rejection reasons most common first.
func (r trainResult) print(w io.Writer, dryRun bool) {
	reasons := make([]string, 0, len(r.rejected))
	total := 0
	for reason, n := range r.rejected {
		reasons = append(reasons, reason)
		total += n
	}
	sort.Slice(reasons, func(i, j int) bool {
		if r.rejected[reasons[i]] != r.rejected[reasons[j]] {
			return r.rejected[reasons[i]] > r.rejected[reasons[j]]
		}
		return reasons[i] < reasons[j]
	})

	verb := "learned"
	if dryRun {
		verb = "would learn"
	}
	fmt.Fprintf(w, "%s %d new usernames (%d already known, %d garbage rejected)\n", verb, r.added, r.known, total)
	for _, reason := range reasons {
		fmt.Fprintf(w, "  %6d  %s\n", r.rejected[reason], reason)
	}
}
