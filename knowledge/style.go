package knowledge

import (
	"strings"

	"github.com/anaassss/Project/markov"
)

// What a username that looks like this knowledge is: real, but neither
// decorated like a gamer tag or social handle nor too plain to be free.

func set(lists ...[]string) map[string]bool {
	s := map[string]bool{}
	for _, l := range lists {
		for _, w := range l {
			s[w] = true
		}
	}
	return s
}

var (
	names  = set(firstNames, commonFirstNames, commonLastNames, rareLastNames)
	common = set(commonFirstNames, commonLastNames)

	// decorations are words people put around a name rather than in it:
	// prefixes (its, lil, mr), suffixes (playz, boi, mom), wrappers (xx,
	// xo) and web endings (io, png).
	decorations = set(strings.Fields(`
		its iam im mr mrs ms miss lil little big the real just hey not my get
		official sir lord dj xx xo xd yt ttv tv hd gg lol io exe png jpg com co
		www playz plays gamer gaming games live pro boi boy girl kid dude bro guy
		man mom mum mama mommy dad daddy papa and`))

	// Decorations run into a name. Some start names too (theo, lilian), so
	// they come off only before a name of 4+ letters...
	namePrefixes = []string{"miss", "mrs", "real", "lil", "the", "mr"}
	// ... and no name starts or ends with these, so they come off whenever
	// 4+ letters follow or 3+ come before.
	joinedPrefixes = []string{"official", "its", "iam", "xx", "mr"}
	joinedSuffixes = []string{"official", "playz", "plays", "gaming", "gamer", "mommy", "daddy", "mama", "girl", "boi", "boy", "ttv", "tv", "hd", "xx", "xd", "yt"}
)

// Decoration reports whether word is one people put around a name rather
// than in it, like its, lil, xx, playz, mom or io.
func Decoration(word string) bool { return decorations[strings.ToLower(word)] }

// TrimJoined takes a decoration run into a word off it: itsmike → mike,
// noahplayz → noah, lilzara → zara. It returns word, lowercased, if there
// is none.
func TrimJoined(word string) string {
	w := strings.ToLower(word)
	trimmed := false
	for _, p := range namePrefixes {
		if rest, ok := strings.CutPrefix(w, p); ok && len(rest) >= 4 && names[rest] {
			w, trimmed = rest, true
			break
		}
	}
	for _, p := range joinedPrefixes {
		if rest, ok := strings.CutPrefix(w, p); ok && !trimmed && len(rest) >= 4 {
			w = rest
			break
		}
	}
	for _, s := range joinedSuffixes {
		if rest, ok := strings.CutSuffix(w, s); ok && len(rest) >= 3 {
			w = rest
			break
		}
	}
	return w
}

// Unique reports whether name looks like a real username that is unlikely
// to be taken already: it has no decorations (its.mike, noahplayz,
// xXmikeXx, maria.io, NovaTV) and is not Plain.
func Unique(name string) bool {
	if len(name) >= 2 && (strings.EqualFold(name[:2], "xx") || strings.EqualFold(name[len(name)-2:], "xx")) {
		return false
	}
	segs := markov.Split(name)
	for i, seg := range segs {
		if seg.Kind != markov.Word {
			continue
		}
		w := strings.ToLower(seg.Text)
		if decorations[w] || w == "x" && (i == 0 || i == len(segs)-1) || TrimJoined(w) != w {
			return false
		}
	}
	return !plain(segs)
}

// Plain reports whether name is so plain that it is almost surely taken:
// a name on its own (stefan), a very common name with a birth year or up to
// two digits (emma2008, john92), or two very common names or an initial
// and one (johnkevin, john.smith, jsmith).
func Plain(name string) bool { return plain(markov.Split(name)) }

func plain(segs []markov.Segment) bool {
	var words [2]string
	nw := 0
	num := ""
	for _, seg := range segs {
		switch seg.Kind {
		case markov.Word:
			if nw == len(words) {
				return false
			}
			words[nw] = strings.ToLower(seg.Text)
			nw++
		case markov.Number:
			if num != "" {
				return false
			}
			num = seg.Text
		}
	}
	if nw == 1 && !names[words[0]] {
		if a, b, ok := commonPair(words[0]); ok {
			words, nw = [2]string{a, b}, 2
		}
	}
	switch nw {
	case 1:
		w := words[0]
		return names[w] && (num == "" || common[w] && (len(num) <= 2 || isYear(num)))
	case 2:
		if num != "" {
			return false
		}
		a, b := words[0], words[1]
		return common[a] && common[b] || len(a) == 1 && common[b] || common[a] && len(b) == 1
	}
	return false
}

// commonPair splits w into two very common names, or an initial and one:
// johnkevin → john + kevin, jsmith → j + smith.
func commonPair(w string) (string, string, bool) {
	if len(w) > 1 && common[w[1:]] {
		return w[:1], w[1:], true
	}
	for i := 2; i <= len(w)-2; i++ {
		if common[w[:i]] && common[w[i:]] {
			return w[:i], w[i:], true
		}
	}
	return "", "", false
}
