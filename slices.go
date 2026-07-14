package pooled

import (
	"math/bits"
	"sync"
	"unsafe"
)

type sliceItem struct {
	data unsafe.Pointer
	cap  int
}

// slicePointer stores a power-of-two-capacity slice without a metadata wrapper.
type slicePointer unsafe.Pointer

// sliceItemPool is shared by every Slices and FlatSlices instantiation.
var sliceItemPool = sync.Pool{
	New: func() any { return new(sliceItem) },
}

// Slices pools []T values.
//
// Zero value is ready to use. It must not be copied after first use.
type Slices[T any] struct {
	// MaxCap is the maximum slice capacity served from the pool.
	// Values <= 32 are treated as 32.
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

// FlatSlices pools []T values in a single pool, regardless of capacity.
//
// Zero value is ready to use. It must not be copied after first use.
type FlatSlices[T any] struct {
	// NewCap is the capacity used when allocating a new slice.
	// Values <= 0 allocate a slice with default capacity.
	NewCap int

	// MaxCap is the maximum slice capacity served from the pool.
	// Values <= 32 are treated as 32.
	MaxCap int

	// Clear controls how slices are cleared before returning to the pool.
	Clear SliceClearPolicy

	// Cleanup is called by Put before clearing and retention checks.
	// It is called for every Put call, including nil and discarded slices.
	Cleanup func([]T)

	_    noCopy
	pool sync.Pool
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

// Get returns a slice with len 0 and cap at least capHint.
// If capHint is larger than MaxCap, Get allocates without consulting the pool.
func (s *Slices[T]) Get(capHint int) []T {
	maxShift := min(bits.Len(uint(max(s.MaxCap, sliceMinCap)-1)), sliceMaxShift)
	maxCap := 1 << maxShift

	if capHint > maxCap {
		return make([]T, 0, capHint)
	}

	shift := min(bits.Len(uint(max(capHint, sliceMinCap)-1)), sliceMaxShift)
	search := min(shift+sliceMaxDistance, maxShift)

	for sh := shift; sh <= search; sh++ {
		if raw := s.pools[sh].Get(); raw != nil {
			switch v := raw.(type) {
			case slicePointer:
				n := 1 << sh
				r := unsafe.Slice((*T)(v), n)
				return r[:0:n]

			case *sliceItem:
				c := v.cap
				r := unsafe.Slice((*T)(v.data), c)

				v.data = nil
				v.cap = 0
				sliceItemPool.Put(v)

				return r[:0:c]

			default:
				panic("pooled: unexpected slice pool value type")
			}
		}
	}

	return make([]T, 0, 1<<shift)
}

// Put returns v to the pool.
//
// Put chooses the bucket from cap(v) at call time. The slice does not need to
// have the same capacity it had when it was returned by Get.
//
// Slices with too small capacity or capacity above MaxCap are discarded.
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
	maxCap := 1 << maxShift
	if c > maxCap {
		return
	}

	shift := bits.Len(uint(c)) - 1 // floor(log2(cap))

	switch s.Clear {
	case ClearLen:
		clear(v)
	case ClearCap:
		clear(v[:c])
	}

	if c == 1<<shift {
		s.pools[shift].Put(slicePointer(unsafe.SliceData(v)))
		return
	}

	item := sliceItemPool.Get().(*sliceItem)
	item.data = unsafe.Pointer(unsafe.SliceData(v))
	item.cap = c
	s.pools[shift].Put(item)
}

// Get returns a slice with len 0.
// It allocates a new slice with capacity NewCap when the pool is empty.
func (s *FlatSlices[T]) Get() []T {
	if raw := s.pool.Get(); raw != nil {
		item, ok := raw.(*sliceItem)
		if !ok {
			panic("pooled: unexpected flat slice pool value type")
		}

		c := item.cap
		r := unsafe.Slice((*T)(item.data), c)

		item.data = nil
		item.cap = 0
		sliceItemPool.Put(item)

		return r[:0:c]
	}
	if s.NewCap > 0 {
		return make([]T, 0, s.NewCap)
	}
	return make([]T, 0)
}

// Put returns v to the pool.
//
// Slices with zero capacity or capacity above MaxCap are discarded.
//
// If Cleanup is set, it is called before clearing and discard checks.
func (s *FlatSlices[T]) Put(v []T) {
	if s.Cleanup != nil {
		s.Cleanup(v)
	}

	c := cap(v)
	if c == 0 || c > max(s.MaxCap, sliceMinCap) {
		return
	}

	switch s.Clear {
	case ClearLen:
		clear(v)
	case ClearCap:
		clear(v[:c])
	}

	item := sliceItemPool.Get().(*sliceItem)
	item.data = unsafe.Pointer(unsafe.SliceData(v))
	item.cap = c
	s.pool.Put(item)
}
