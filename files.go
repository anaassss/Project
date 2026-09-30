package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// numberedFile collects lines in a temporary file, then keeps them as
// <prefix>_<lines>.txt, never overwriting an existing file.
type numberedFile struct {
	dir, prefix string
	f           *os.File
	w           *bufio.Writer
	lines       int
	kept        bool
}

func newNumberedFile(dir, prefix string) (*numberedFile, error) {
	f, err := os.CreateTemp(dir, "."+prefix+"-*.tmp")
	if err != nil {
		return nil, err
	}
	return &numberedFile{dir: dir, prefix: prefix, f: f, w: bufio.NewWriterSize(f, 1<<20)}, nil
}

func (n *numberedFile) writeLine(s string) {
	n.w.WriteString(s)
	n.w.WriteByte('\n')
	n.lines++
}

// keep finishes the file and gives it its final name, which it returns.
func (n *numberedFile) keep() (string, error) {
	if err := n.w.Flush(); err != nil {
		return "", err
	}
	if err := n.f.Chmod(0o644); err != nil {
		return "", err
	}
	if err := n.f.Close(); err != nil {
		return "", err
	}
	path, err := placeNumbered(n.f.Name(), n.dir, n.prefix, n.lines)
	n.kept = err == nil
	return path, err
}

// discard deletes the temporary file unless keep succeeded.
func (n *numberedFile) discard() {
	if !n.kept {
		n.f.Close()
		os.Remove(n.f.Name())
	}
}

// placeNumbered moves tmp to <prefix>_<count>.txt in dir. It never
// overwrites: if that file exists it uses <prefix>_<count>_2.txt, and so on.
func placeNumbered(tmp, dir, prefix string, count int) (string, error) {
	for i := 1; ; i++ {
		name := fmt.Sprintf("%s_%d.txt", prefix, count)
		if i > 1 {
			name = fmt.Sprintf("%s_%d_%d.txt", prefix, count, i)
		}
		path := filepath.Join(dir, name)
		_, err := os.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) {
			return path, os.Rename(tmp, path)
		}
		if err != nil {
			return "", err
		}
	}
}
