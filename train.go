package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

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

	var onRejected func(name, reason string)
	if *showRejected {
		onRejected = func(name, reason string) { fmt.Fprintf(os.Stderr, "rejected %q: %s\n", name, reason) }
	}
	save := *modelPath
	if *dryRun {
		save = ""
	}
	summary, err := trainFiles(m, flags.Args(), save, onRejected, terminalOrNil(os.Stderr))
	if err != nil {
		return err
	}
	if *dryRun {
		summary = "Dry run, nothing saved. " + summary
	}
	fmt.Println(summary)
	return nil
}

// trainFiles teaches m every username in paths ("-" is stdin), drawing a
// progress bar on bar (nil for none), then saves m to save unless it is "".
// It returns a one-line summary.
func trainFiles(m *markov.Model, paths []string, save string, onRejected func(name, reason string), bar io.Writer) (string, error) {
	var total int64
	for _, path := range paths {
		if path == "-" {
			total = -1 // unknown
			break
		}
		info, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		total += info.Size()
	}

	p := startProgress(bar, "Training", total)
	res := markov.LearnResult{Rejected: map[string]int{}}
	err := func() error {
		for _, path := range paths {
			r := io.Reader(os.Stdin)
			if path != "-" {
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()
				r = f
			}
			got, err := m.LearnFrom(p.reader(r, 1), onRejected)
			res.Added += got.Added
			res.Known += got.Known
			for reason, n := range got.Rejected {
				res.Rejected[reason] += n
			}
			if err != nil {
				return fmt.Errorf("reading %s: %w", path, err)
			}
		}
		if save != "" {
			if err := m.Save(save); err != nil {
				return fmt.Errorf("saving model: %w", err)
			}
		}
		return nil
	}()
	elapsed := p.end(err == nil)
	if err != nil {
		return "", err
	}
	return trainSummary(res, elapsed), nil
}

func trainSummary(res markov.LearnResult, elapsed time.Duration) string {
	s := fmt.Sprintf("Learned %s new usernames in %s", formatCount(res.Added), formatDuration(elapsed))
	if skipped := res.Known + res.RejectedTotal(); skipped > 0 {
		s += fmt.Sprintf(" (skipped %s already known, %s garbage)", formatCount(res.Known), formatCount(res.RejectedTotal()))
	}
	return s
}
