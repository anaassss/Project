package markov

import "unicode"

// SegmentKind classifies a Segment.
type SegmentKind int

const (
	Word SegmentKind = iota
	Number
	Separator
	Other
)

// Segment is one part of a username: a word, a number, or separators.
type Segment struct {
	Text string
	Kind SegmentKind
}

func kindOf(r rune) SegmentKind {
	switch {
	case unicode.IsLetter(r):
		return Word
	case unicode.IsDigit(r):
		return Number
	case r == '_' || r == '-' || r == '.':
		return Separator
	}
	return Other
}

// Split breaks a username into words, numbers and separators, splitting
// words at case changes: "xXShadowWolf_99" → x, X, Shadow, Wolf, _, 99.
func Split(name string) []Segment {
	runes := []rune(name)
	var segs []Segment
	start := 0
	for i := 1; i <= len(runes); i++ {
		if i == len(runes) || boundary(runes, i) {
			segs = append(segs, Segment{string(runes[start:i]), kindOf(runes[start])})
			start = i
		}
	}
	return segs
}

// boundary reports whether a new segment starts at runes[i].
func boundary(runes []rune, i int) bool {
	prev, cur := runes[i-1], runes[i]
	if kindOf(prev) != kindOf(cur) {
		return true
	}
	if kindOf(cur) != Word {
		return false
	}
	if unicode.IsLower(prev) && unicode.IsUpper(cur) {
		return true // camelCase
	}
	// An acronym followed by a word: "XMLParser" splits before "P".
	return unicode.IsUpper(prev) && unicode.IsUpper(cur) && i+1 < len(runes) && unicode.IsLower(runes[i+1])
}
