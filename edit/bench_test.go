package edit

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/anaassss/Project/markov"
)

func BenchmarkRun(b *testing.B) {
	m, _ := markov.New(3)
	f, err := os.Open("../testdata/usernames.txt")
	if err != nil {
		b.Fatal(err)
	}
	m.LearnFrom(f, nil)
	f.Close()
	e, _ := New(m, Options{Max: 10})

	inputs := []string{"ShadowFox", "DarkWolf_7", "StormHawk99", "coolguy", "xXSniperXx", "Mystic.Panda"}
	var in strings.Builder
	for i := range 10000 {
		in.WriteString(inputs[i%len(inputs)] + "\n")
	}
	b.SetBytes(int64(in.Len()))
	b.ResetTimer()
	for range b.N {
		e.Run(strings.NewReader(in.String()), io.Discard, map[uint64]struct{}{}, 1)
	}
}
