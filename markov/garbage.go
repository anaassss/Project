package markov

import (
	"errors"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Scraped username lists are full of entries that would teach the model bad
// habits: emails, numeric IDs, keyboard mashes, placeholders like "[deleted]".
// Check rejects them before they are learned, and Generate applies the same
// rules to its own output.

const (
	MinNameLen = 3
	MaxNameLen = 24

	maxDigitRun     = 4 // a year fits; phone numbers and IDs don't
	maxRepeat       = 3 // "zzz" is fine, "zzzz" is not
	maxChunkRepeat  = 2 // "haha" and "Banana" are fine, "hahaha" is not
	maxConsonantRun = 6 // "FirstStrike" has 6; mashes like "zxcvbnm" have more
	minVowelless    = 6 // Latin letters needed before a lack of vowels counts
	sequenceLen     = 5 // "qwert", "abcde", "lkjhg"
	minHexIDLen     = 8 // "8f3a9c2b", the shape of hashes and UUID chunks
	minHexIDDigits  = 3 // so words like "deadbeef" survive
)

// ErrGarbage is wrapped by every error Check returns.
var ErrGarbage = errors.New("garbage username")

// GarbageError says why Check rejected a name.
type GarbageError struct {
	Reason string
}

func (e *GarbageError) Error() string { return "garbage username: " + e.Reason }
func (e *GarbageError) Unwrap() error { return ErrGarbage }

// placeholders are names that platforms show for missing, default, or system
// accounts. They are rejected alone and with a numeric suffix ("user_123").
var placeholders = map[string]bool{
	"admin": true, "administrator": true, "anon": true, "anonymous": true,
	"automoderator": true, "default": true, "deleted": true, "example": true,
	"guest": true, "member": true, "nan": true, "nil": true, "none": true,
	"null": true, "placeholder": true, "player": true, "removed": true,
	"root": true, "sample": true, "test": true, "testing": true,
	"undefined": true, "unknown": true, "user": true, "username": true,
}

// sequences are keyboard rows and the alphabet, forwards and backwards.
var sequences = func() []string {
	var seqs []string
	for _, s := range []string{"qwertyuiop", "asdfghjkl", "zxcvbnm", "abcdefghijklmnopqrstuvwxyz"} {
		r := []rune(s)
		slices.Reverse(r)
		seqs = append(seqs, s, string(r))
	}
	return seqs
}()

// Check returns a *GarbageError if name is not worth learning from.
// Usernames may contain letters in any script, digits, '_', '-' and '.'.
func Check(name string) error {
	reject := func(reason string) error { return &GarbageError{Reason: reason} }

	if !utf8.ValidString(name) {
		return reject("not valid UTF-8")
	}
	runes := []rune(name)
	if len(runes) < MinNameLen {
		return reject("too short")
	}
	if len(runes) > MaxNameLen {
		return reject("too long")
	}

	letters := 0
	for _, r := range runes {
		switch {
		case unicode.IsLetter(r):
			letters++
		case unicode.IsDigit(r), r == '_', r == '-', r == '.':
		default:
			return reject("disallowed character")
		}
	}

	lower := strings.ToLower(name)
	if placeholders[strings.TrimRight(lower, "0123456789_-.")] {
		return reject("placeholder or auto-generated")
	}
	if letters*2 < len(runes) {
		return reject("mostly digits or symbols")
	}
	if isHexID(lower) {
		return reject("looks like an ID")
	}
	for _, seq := range sequences {
		for i := 0; i+sequenceLen <= len(seq); i++ {
			if strings.Contains(lower, seq[i:i+sequenceLen]) {
				return reject("keyboard or alphabet sequence")
			}
		}
	}

	if repeats([]rune(lower)) {
		return reject("repeated pattern")
	}

	digitRun, consonantRun, latin, vowels := 0, 0, 0, 0
	for _, r := range runes {
		if unicode.IsDigit(r) {
			if digitRun++; digitRun > maxDigitRun {
				return reject("long run of digits")
			}
		} else {
			digitRun = 0
		}

		if r < utf8.RuneSelf && unicode.IsLetter(r) {
			latin++
			if strings.ContainsRune("aeiouyAEIOUY", r) {
				vowels++
				consonantRun = 0
			} else if consonantRun++; consonantRun > maxConsonantRun {
				return reject("unpronounceable")
			}
		} else {
			consonantRun = 0
		}
	}

	if latin >= minVowelless && vowels == 0 {
		return reject("unpronounceable")
	}
	return nil
}

func isHexID(s string) bool {
	if len(s) < minHexIDLen {
		return false
	}
	digits := 0
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r < 'a' || r > 'f':
			return false
		}
	}
	return digits >= minHexIDDigits
}

// repeats reports whether a chunk of 1-3 characters repeats back to back more
// often than real names do. Only whole repeats count: "anana" in "Banana" is
// "an" 2.5 times and allowed, "hahaha" is "ha" 3 times and rejected.
func repeats(runes []rune) bool {
	for size := 1; size <= 3; size++ {
		limit := maxChunkRepeat
		if size == 1 {
			limit = maxRepeat
		}
		// Count characters that match the one size positions earlier. A chunk
		// repeated k times back to back gives a run of size*(k-1) matches.
		run := 0
		for i := size; i < len(runes); i++ {
			if runes[i] != runes[i-size] {
				run = 0
				continue
			}
			if run++; run >= size*limit {
				return true
			}
		}
	}
	return false
}
