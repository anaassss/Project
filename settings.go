package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/anaassss/Project/markov"
)

// Settings used by the interactive menu. Commands take flags instead, so
// scripts never depend on this file.

const settingsFile = "usergen.settings.json"

type settings struct {
	ModelFile    string  `json:"model_file"`
	Order        int     `json:"order"` // context length for a new model
	SaveRejected bool    `json:"save_rejected"`
	DryRun       bool    `json:"dry_run"`
	EditsPerName int     `json:"edits_per_username"`
	GenCount     int     `json:"generate_count"`
	GenMinLen    int     `json:"generate_min_length"`
	GenMaxLen    int     `json:"generate_max_length"`
	GenTemp      float64 `json:"generate_creativity"`
	GenOrder     int     `json:"generate_context"` // 0 means the model's order
	AllowKnown   bool    `json:"allow_known"`
	Seed         uint64  `json:"seed"` // 0 means random each time
}

func defaultSettings() settings {
	return settings{
		ModelFile:    defaultModel,
		Order:        defaultOrder,
		EditsPerName: defaultMaxEdits,
		GenCount:     10,
		GenMinLen:    4,
		GenMaxLen:    16,
		GenTemp:      1,
	}
}

// Limits for settings typed into the menu.
const (
	maxEditsPerName = 1000
	maxGenCount     = 10_000_000
	maxTemp         = 10.0
)

func (c settings) validate() error {
	switch {
	case c.ModelFile == "":
		return errors.New("model file must not be empty")
	case c.Order < 1 || c.Order > markov.MaxOrder:
		return fmt.Errorf("context length must be 1-%d", markov.MaxOrder)
	case c.EditsPerName < 1 || c.EditsPerName > maxEditsPerName:
		return fmt.Errorf("edits per username must be 1-%d", maxEditsPerName)
	case c.GenCount < 1 || c.GenCount > maxGenCount:
		return fmt.Errorf("how many usernames must be 1-%s", formatCount(maxGenCount))
	case c.GenMinLen < markov.MinNameLen || c.GenMaxLen > markov.MaxNameLen:
		return fmt.Errorf("lengths must be %d-%d", markov.MinNameLen, markov.MaxNameLen)
	case c.GenMinLen > c.GenMaxLen:
		return fmt.Errorf("shortest length %d is above longest length %d", c.GenMinLen, c.GenMaxLen)
	case !(c.GenTemp > 0 && c.GenTemp <= maxTemp):
		return fmt.Errorf("creativity must be above 0 and at most %g", maxTemp)
	case c.GenOrder < 0 || c.GenOrder > markov.MaxOrder:
		return fmt.Errorf("generate context length must be 0-%d", markov.MaxOrder)
	}
	return nil
}

// loadSettings reads path, returning the defaults if it does not exist.
// Settings missing from the file keep their defaults.
func loadSettings(path string) (settings, error) {
	c := defaultSettings()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return defaultSettings(), err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return defaultSettings(), fmt.Errorf("%s is not valid: %w", path, err)
	}
	if err := c.validate(); err != nil {
		return defaultSettings(), fmt.Errorf("%s is not valid: %w", path, err)
	}
	return c, nil
}

// save writes the settings to path atomically.
func (c settings) save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op once renamed
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// option is one line of the settings screen.
type option struct {
	group string // printed as a heading before the first option in the group
	label string
	show  func(*settings) string
	// set parses a typed value into the settings. Options without set are
	// on/off switches that flip when chosen.
	set  func(*settings, string) error
	flip func(*settings)
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func intOption(group, label string, field func(*settings) *int) option {
	return option{
		group: group,
		label: label,
		show:  func(c *settings) string { return strconv.Itoa(*field(c)) },
		set: func(c *settings, v string) error {
			n, err := strconv.Atoi(strings.ReplaceAll(v, ",", ""))
			if err != nil {
				return errors.New("enter a whole number")
			}
			*field(c) = n
			return nil
		},
	}
}

func boolOption(group, label string, field func(*settings) *bool) option {
	return option{
		group: group,
		label: label,
		show:  func(c *settings) string { return onOff(*field(c)) },
		flip:  func(c *settings) { *field(c) = !*field(c) },
	}
}

var options = []option{
	intOption("Training", "Context length for a new model (1-8)", func(c *settings) *int { return &c.Order }),
	boolOption("Training", "Save rejected usernames to a file", func(c *settings) *bool { return &c.SaveRejected }),
	boolOption("Training", "Dry run: learn nothing, just report", func(c *settings) *bool { return &c.DryRun }),
	intOption("Edit", "Edits per username", func(c *settings) *int { return &c.EditsPerName }),
	intOption("Generate", "How many usernames", func(c *settings) *int { return &c.GenCount }),
	intOption("Generate", "Shortest length", func(c *settings) *int { return &c.GenMinLen }),
	intOption("Generate", "Longest length", func(c *settings) *int { return &c.GenMaxLen }),
	{
		group: "Generate",
		label: "Creativity (1 = as learned, higher = wilder)",
		show:  func(c *settings) string { return strconv.FormatFloat(c.GenTemp, 'g', -1, 64) },
		set: func(c *settings, v string) error {
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return errors.New("enter a number such as 0.7 or 1.5")
			}
			c.GenTemp = f
			return nil
		},
	},
	intOption("Generate", "Context length (0 = the model's)", func(c *settings) *int { return &c.GenOrder }),
	boolOption("Edit and Generate", "Allow usernames the model learned from", func(c *settings) *bool { return &c.AllowKnown }),
	{
		group: "Edit and Generate",
		label: "Seed (0 = random each time)",
		show:  func(c *settings) string { return strconv.FormatUint(c.Seed, 10) },
		set: func(c *settings, v string) error {
			n, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return errors.New("enter a whole number, 0 or more")
			}
			c.Seed = n
			return nil
		},
	},
	{
		group: "Model",
		label: "Model file",
		show:  func(c *settings) string { return c.ModelFile },
		set: func(c *settings, v string) error {
			c.ModelFile = cleanPath(v)
			return nil
		},
	},
}

// settingsScreen lets the user view and change settings until they go back.
func (s *session) settingsScreen() error {
	for {
		fmt.Fprintf(s.out, "\nSettings (saved in %s)\n", s.settingsPath)
		group := ""
		for i, o := range options {
			if o.group != group {
				group = o.group
				fmt.Fprintf(s.out, "  %s\n", group)
			}
			fmt.Fprintf(s.out, "   %2d  %-46s %s\n", i+1, o.label, o.show(&s.cfg))
		}
		fmt.Fprintf(s.out, "    0  Back\n")

		answer, err := s.ask("Change which? ")
		if err != nil {
			return err
		}
		if answer == "0" || answer == "" {
			return nil
		}
		i, err := strconv.Atoi(answer)
		if err != nil || i < 1 || i > len(options) {
			fmt.Fprintf(s.out, "Choose 1-%d, or 0 to go back.\n", len(options))
			continue
		}
		if err := s.change(options[i-1]); err != nil {
			return err
		}
	}
}

// change updates one setting, asking for its new value unless it is an
// on/off switch, and saves the settings if the result is valid.
func (s *session) change(o option) error {
	next := s.cfg
	if o.flip != nil {
		o.flip(&next)
	} else {
		for {
			answer, err := s.ask(fmt.Sprintf("%s [%s]: ", o.label, o.show(&s.cfg)))
			if err != nil {
				return err
			}
			if answer == "" {
				return nil // keep the current value
			}
			if err := o.set(&next, answer); err != nil {
				fmt.Fprintln(s.out, capitalize(err.Error())+".")
				continue
			}
			break
		}
	}
	if err := next.validate(); err != nil {
		fmt.Fprintln(s.out, "Not changed: "+err.Error()+".")
		return nil
	}
	if err := next.save(s.settingsPath); err != nil {
		return fmt.Errorf("saving settings: %w", err)
	}

	if next.ModelFile != s.cfg.ModelFile {
		s.m = nil // switch models
	}
	orderChanged := next.Order != s.cfg.Order
	s.cfg = next
	if orderChanged {
		if m, err := s.loadQuiet(); err == nil && m != nil && m.Order() != s.cfg.Order {
			fmt.Fprintf(s.out, "Note: the current model uses a context length of %d. The new length applies after Clear all knowledge.\n", m.Order())
		}
	}
	return nil
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
