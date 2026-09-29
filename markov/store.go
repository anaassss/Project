package markov

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"
)

// The model file is binary so that models trained on millions of usernames
// save and load quickly:
//
//	magic, then uvarints: version, order, names, total length, min, max length
//	grams:   count, then (key length, key bytes, count) each
//	words:   count, then (length, bytes, count) each
//	numbers: same as words
//	seen:    count, then 8 little-endian bytes each
//
// Version 1 models were JSON holding every learned name; Load converts them.
const (
	magic         = "USERGEN\x00"
	formatVersion = 2

	maxStringLen = 1 << 16 // longest key a valid file can contain
	maxPrealloc  = 1 << 20 // map capacity to trust from a file's counts
)

// Save writes the model to path atomically, so an interrupted save never
// leaves a corrupt model behind.
func (m *Model) Save(path string) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()

	w := &writer{w: bufio.NewWriterSize(tmp, 1<<20)}
	w.w.WriteString(magic)
	for _, v := range []uint64{formatVersion, uint64(m.order), m.stats.Names, m.stats.TotalLen,
		uint64(m.stats.MinLen), uint64(m.stats.MaxLen)} {
		w.uvarint(v)
	}
	for _, counts := range []map[string]uint32{m.grams, m.words, m.numbers} {
		w.uvarint(uint64(len(counts)))
		for k, c := range counts {
			w.uvarint(uint64(len(k)))
			w.w.WriteString(k)
			w.uvarint(uint64(c))
		}
	}
	w.uvarint(uint64(m.seen.len()))
	m.seen.each(func(h uint64) error {
		w.buf = binary.LittleEndian.AppendUint64(w.buf[:0], h)
		_, err := w.w.Write(w.buf)
		return err
	})

	if err = w.w.Flush(); err != nil {
		return err
	}
	if err = tmp.Chmod(0o644); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

type writer struct {
	w   *bufio.Writer
	buf []byte
}

func (w *writer) uvarint(v uint64) {
	w.buf = binary.AppendUvarint(w.buf[:0], v)
	w.w.Write(w.buf)
}

// Load reads a model previously written by Save.
func Load(path string) (*Model, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReaderSize(f, 1<<20)
	if first, err := r.Peek(1); err == nil && first[0] == '{' {
		m, err := loadV1(r)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		return m, nil
	}
	m, err := load(r)
	if err != nil {
		return nil, fmt.Errorf("%s: not a valid model file: %w", path, err)
	}
	return m, nil
}

func load(r *bufio.Reader) (*Model, error) {
	head := make([]byte, len(magic))
	if _, err := io.ReadFull(r, head); err != nil || string(head) != magic {
		return nil, errors.New("missing header")
	}
	var hdr [6]uint64
	for i := range hdr {
		v, err := binary.ReadUvarint(r)
		if err != nil {
			return nil, err
		}
		hdr[i] = v
	}
	if hdr[0] != formatVersion {
		return nil, fmt.Errorf("unsupported version %d", hdr[0])
	}
	if hdr[1] < 1 || hdr[1] > MaxOrder {
		return nil, fmt.Errorf("bad order %d", hdr[1])
	}
	m := newModel(int(hdr[1]))
	m.stats = Stats{Names: hdr[2], TotalLen: hdr[3], MinLen: int(hdr[4]), MaxLen: int(hdr[5])}

	for i, dst := range []*map[string]uint32{&m.grams, &m.words, &m.numbers} {
		counts, err := readCounts(r)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			for k := range counts {
				if n := utf8.RuneCountInString(k); !utf8.ValidString(k) || n < 1 || n > m.order+1 {
					return nil, fmt.Errorf("corrupt pattern %q", k)
				}
			}
		}
		*dst = counts
	}

	n, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, err
	}
	var b [8]byte
	for range n {
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return nil, err
		}
		h := binary.LittleEndian.Uint64(b[:])
		m.seen.shard(h).m[h] = struct{}{}
	}
	return m, nil
}

func readCounts(r *bufio.Reader) (map[string]uint32, error) {
	n, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]uint32, min(n, maxPrealloc))
	var buf []byte
	for range n {
		size, err := binary.ReadUvarint(r)
		if err != nil {
			return nil, err
		}
		if size > maxStringLen {
			return nil, fmt.Errorf("key of %d bytes", size)
		}
		buf = append(buf[:0], make([]byte, size)...)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		c, err := binary.ReadUvarint(r)
		if err != nil {
			return nil, err
		}
		if c == 0 || c > 1<<32-1 {
			return nil, fmt.Errorf("bad count %d", c)
		}
		counts[string(buf)] = uint32(c)
	}
	return counts, nil
}

// loadV1 converts a version 1 (JSON) model by relearning its names, which
// also drops any that the garbage filter now rejects.
func loadV1(r io.Reader) (*Model, error) {
	var f struct {
		Version int      `json:"version"`
		Order   int      `json:"order"`
		Names   []string `json:"names"`
	}
	if err := json.NewDecoder(r).Decode(&f); err != nil {
		return nil, fmt.Errorf("not a valid model file: %w", err)
	}
	if f.Version != 1 {
		return nil, fmt.Errorf("unsupported model version %d", f.Version)
	}
	m, err := New(f.Order)
	if err != nil {
		return nil, err
	}
	for _, name := range f.Names {
		m.Learn(name)
	}
	return m, nil
}
