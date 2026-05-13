package pooled

import (
	"math/bits"
	"sync"
	"unsafe"
)

// Slices pools []T values.
//
// Zero value is ready to use. It must not be copied after first use.
type Slices[T any] struct {
	// MaxCap is the maximum requested capacity served from the pool.
	// Values <= 0 are treated as 32.
	//
	// MaxCap is rounded up to the next power of two.
	MaxCap int

	// Clear controls how slices are cleared before returning to the pool.
	Clear SliceClearPolicy

	// Cleanup is called by Put before clearing and retention checks.
	// It is called for every Put call, including nil and discarded slices.
	Cleanup func([]T)

	_     noCopy
	pools [sliceMaxShift + 1]sync.Pool
}

// SliceClearPolicy controls how a slice is cleared before pooling.
type SliceClearPolicy byte

const (
	// NoClear leaves slice contents unchanged.
	NoClear SliceClearPolicy = iota

	// ClearLen clears the current slice length.
	ClearLen

	// ClearCap clears the full slice capacity.
	ClearCap
)

const (
	sliceMinShift    = 5
	sliceMinCap      = 1 << sliceMinShift
	sliceMaxDistance = 3

	// largest sane pooled shift (134M items, ~1GiB []int64, ~2GiB []string)
	sliceMaxShift = 27
)

const (
	sliceRetainMinSlack   = 64
	sliceRetainStartShift = 10 // 1 Ki
	sliceRetainStepShift  = 4  // every x16 capacity halves relative slack
	sliceRetainBaseShift  = 2  // +25%
)

var sliceRetainCaps = func() [sliceMaxShift + 1]int {
	var a [sliceMaxShift + 1]int
	for sh := range a {
		a[sh] = retainCapForShift(sh)
	}
	return a
}()

func retainCapForShift(sh int) int {
	n := 1 << sh
	k := sliceRetainBaseShift + max(sh-sliceRetainStartShift, 0)/sliceRetainStepShift

	return n + max(n>>k, sliceRetainMinSlack)
}

// Get returns a slice with len 0 and cap at least capHint.
// If capHint is larger than MaxCap, Get allocates an unpooled slice.
func (s *Slices[T]) Get(capHint int) []T {
	maxShift := min(bits.Len(uint(max(s.MaxCap, sliceMinCap)-1)), sliceMaxShift)
	maxCap := 1 << maxShift

	if capHint > maxCap {
		return make([]T, 0, capHint)
	}

	shift := min(bits.Len(uint(max(capHint, sliceMinCap)-1)), sliceMaxShift)
	search := min(shift+sliceMaxDistance, maxShift)

	for sh := shift; sh <= search; sh++ {
		if v := s.pools[sh].Get(); v != nil {
			n := 1 << sh
			r := unsafe.Slice(v.(*T), n)
			return r[:0:n]
		}
	}

	return make([]T, 0, 1<<shift)
}

// Put returns v to the pool if its capacity is retained.
//
// Slices with too small capacity, too large capacity, or too much slack for
// their bucket are discarded.
//
// If Cleanup is set, it is called before clearing and discard checks.
func (s *Slices[T]) Put(v []T) {
	if s.Cleanup != nil {
		s.Cleanup(v)
	}
	c := cap(v)
	if c < sliceMinCap {
		return
	}

	maxShift := min(bits.Len(uint(max(s.MaxCap, sliceMinCap)-1)), sliceMaxShift)
	shift := bits.Len(uint(c)) - 1 // floor(log2(cap))
	if shift > maxShift || c > sliceRetainCaps[shift] {
		return
	}

	switch s.Clear {
	case ClearLen:
		clear(v)
	case ClearCap:
		clear(v[:c])
	}

	n := 1 << shift
	s.pools[shift].Put(unsafe.SliceData(v[:n:n]))
}
