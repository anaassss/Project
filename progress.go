package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const barWidth = 30

// progress draws a bar that fills as work is done, measured in units such as
// bytes of input read. A nil writer draws nothing.
type progress struct {
	out   io.Writer
	label string
	total int64 // <= 0 if unknown
	done  atomic.Int64
	start time.Time
	stop  chan struct{}
	wg    sync.WaitGroup
}

func startProgress(out io.Writer, label string, total int64) *progress {
	p := &progress{out: out, label: label, total: total, start: time.Now(), stop: make(chan struct{})}
	if out == nil {
		return p
	}
	p.draw()
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-p.stop:
				return
			case <-tick.C:
				p.draw()
			}
		}
	}()
	return p
}

// reader returns r, counting every byte read from it as weight units done.
func (p *progress) reader(r io.Reader, weight int64) io.Reader {
	return &countingReader{r: r, p: p, weight: weight}
}

type countingReader struct {
	r      io.Reader
	p      *progress
	weight int64
}

func (c *countingReader) Read(b []byte) (int, error) {
	n, err := c.r.Read(b)
	c.p.done.Add(int64(n) * c.weight)
	return n, err
}

// end stops drawing, filling the bar if the work completed, and returns the
// time taken.
func (p *progress) end(completed bool) time.Duration {
	elapsed := time.Since(p.start)
	if p.out == nil {
		return elapsed
	}
	close(p.stop)
	p.wg.Wait()
	if completed && p.total > 0 {
		p.done.Store(p.total)
	}
	p.draw()
	fmt.Fprintln(p.out)
	return elapsed
}

func (p *progress) draw() {
	done := p.done.Load()
	if p.total <= 0 {
		fmt.Fprintf(p.out, "\r%-8s %.1f MB", p.label, float64(done)/1e6)
		return
	}
	frac := min(float64(done)/float64(p.total), 1)
	filled := int(frac * barWidth)
	fmt.Fprintf(p.out, "\r%-8s [%s%s] %3d%%", p.label,
		strings.Repeat("█", filled), strings.Repeat("░", barWidth-filled), int(frac*100))
}

// terminalOrNil returns f if it is a terminal, so progress bars are drawn
// for people but not written into logs or pipes.
func terminalOrNil(f *os.File) io.Writer {
	if info, err := f.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
		return f
	}
	return nil
}

// formatCount writes n with thousands separators: 1234567 → "1,234,567".
func formatCount(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0 && s[i-1] != '-'; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	return d.Round(100 * time.Millisecond).String()
}
