package markov

import (
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"
)

const (
	fnvOffset = 14695981039346656037
	fnvPrime  = 1099511628211
)

// FoldHash returns a case-insensitive 64-bit fingerprint of s. Models store
// these instead of the usernames themselves, so it must never change.
func FoldHash(s string) uint64 {
	h := uint64(fnvOffset)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= utf8.RuneSelf {
			return foldHashSlow(s)
		}
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		h ^= uint64(c)
		h *= fnvPrime
	}
	return mix(h)
}

// foldHashSlow handles non-ASCII names; for ASCII it agrees with FoldHash.
func foldHashSlow(s string) uint64 {
	lower := strings.ToLower(s)
	h := uint64(fnvOffset)
	for i := 0; i < len(lower); i++ {
		h ^= uint64(lower[i])
		h *= fnvPrime
	}
	return mix(h)
}

// mix spreads FNV's bits (MurmurHash3's finalizer) so every bit is usable.
func mix(h uint64) uint64 {
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	return h
}

// hashSet is a set of fingerprints. add may be called from many goroutines
// at once; has and len may too, but not while add is running.
type hashSet struct {
	shards [64]setShard

	// filter answers most "is h present?" questions about absent
	// fingerprints (the common case when editing) from a bit array small
	// enough to stay in CPU cache, instead of a cache miss into the maps.
	filterMu sync.Mutex
	filter   atomic.Pointer[filter] // nil when stale
}

type setShard struct {
	mu sync.Mutex
	m  map[uint64]struct{}
}

func newHashSet() *hashSet {
	s := &hashSet{}
	for i := range s.shards {
		s.shards[i].m = make(map[uint64]struct{})
	}
	return s
}

func (s *hashSet) shard(h uint64) *setShard { return &s.shards[h>>58] }

// add inserts h, reporting false if it was already present.
func (s *hashSet) add(h uint64) bool {
	sh := s.shard(h)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if _, ok := sh.m[h]; ok {
		return false
	}
	sh.m[h] = struct{}{}
	if s.filter.Load() != nil { // don't make every worker write a shared cache line
		s.filter.Store(nil)
	}
	return true
}

// has reports whether h is present. It takes no shard lock: a Model never
// learns (the only time add runs) while other calls are in progress.
func (s *hashSet) has(h uint64) bool {
	f := s.filter.Load()
	if f == nil {
		f = s.buildFilter()
	}
	if !f.mayHave(h) {
		return false
	}
	_, ok := s.shard(h).m[h]
	return ok
}

func (s *hashSet) len() int {
	n := 0
	for i := range s.shards {
		n += len(s.shards[i].m)
	}
	return n
}

// each calls fn for every element. It must not run alongside add.
func (s *hashSet) each(fn func(uint64) error) error {
	for i := range s.shards {
		for h := range s.shards[i].m {
			if err := fn(h); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *hashSet) buildFilter() *filter {
	s.filterMu.Lock()
	defer s.filterMu.Unlock()
	if f := s.filter.Load(); f != nil {
		return f
	}
	size := uint64(1024)
	for size < 16*uint64(s.len()) {
		size *= 2
	}
	f := &filter{words: make([]uint64, size/64), mask: size/64 - 1}
	s.each(func(h uint64) error {
		f.set(h)
		return nil
	})
	s.filter.Store(f)
	return f
}

// filter is a blocked Bloom filter: each fingerprint sets two bits in one
// 64-bit word, so a lookup reads a single word. With 16 bits of space per
// element, about 2% of absent fingerprints get past it.
type filter struct {
	words []uint64
	mask  uint64 // len(words)-1
}

// bitsFor picks h's word and the two bits it sets there. It uses bits of h
// that neither the word index (low bits) nor hashSet.shard (top bits) uses.
func (f *filter) bitsFor(h uint64) (*uint64, uint64) {
	return &f.words[h&f.mask], 1<<(h>>40&63) | 1<<(h>>46&63)
}

func (f *filter) set(h uint64) {
	w, b := f.bitsFor(h)
	*w |= b
}

func (f *filter) mayHave(h uint64) bool {
	w, b := f.bitsFor(h)
	return *w&b == b
}
