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

// source is something to learn usernames from.
type source struct {
	name string
	size int64 // bytes, or -1 if unknown
	open func() (io.ReadCloser, error)
}

// fileSources returns a source for each path ("-" is stdin).
func fileSources(paths []string) ([]source, error) {
	var srcs []source
	for _, path := range paths {
		if path == "-" {
			srcs = append(srcs, source{"stdin", -1, func() (io.ReadCloser, error) { return io.NopCloser(os.Stdin), nil }})
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		srcs = append(srcs, source{path, info.Size(), func() (io.ReadCloser, error) { return os.Open(path) }})
	}
	return srcs, nil
}

// trainFiles teaches m every username in paths ("-" is stdin); see train.
func trainFiles(m *markov.Model, paths []string, save string, onRejected func(name, reason string), bar io.Writer) (string, error) {
	srcs, err := fileSources(paths)
	if err != nil {
		return "", err
	}
	return trainSources(m, srcs, save, onRejected, bar)
}

// trainSources teaches m every username in srcs, drawing a progress bar on
// bar (nil for none), then saves m to save unless it is "". It returns a
// one-line summary.
func trainSources(m *markov.Model, srcs []source, save string, onRejected func(name, reason string), bar io.Writer) (string, error) {
	var total int64
	for _, src := range srcs {
		if src.size < 0 {
			total = -1
			break
		}
		total += src.size
	}

	p := startProgress(bar, "Training", total)
	res := markov.LearnResult{Rejected: map[string]int{}}
	err := func() error {
		for _, src := range srcs {
			r, err := src.open()
			if err != nil {
				return err
			}
			got, err := m.LearnFrom(p.reader(r, 1), onRejected)
			r.Close()
			res.Added += got.Added
			res.Known += got.Known
			for reason, n := range got.Rejected {
				res.Rejected[reason] += n
			}
			if err != nil {
				return fmt.Errorf("reading %s: %w", src.name, err)
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
