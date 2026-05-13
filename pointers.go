package pooled

import (
	"sync"
)

// Pointers pools *T values.
//
// Zero value is ready to use. It must not be copied after first use.
type Pointers[T any] struct {
	New     func() *T // New called by Get to allocate a new value.
	Init    func(*T)  // Init called by Get before returning a value.
	Cleanup func(*T)  // Cleanup called by Put before Clear.
	Clear   bool      // Clear zeroes values before returning to the pool.

	_    noCopy
	pool sync.Pool
}

// Get returns a pooled pointer or allocates a new one.
func (p *Pointers[T]) Get() *T {
	if pv := p.pool.Get(); pv != nil {
		v := pv.(*T)
		if p.Init != nil {
			p.Init(v)
		}
		return v
	}
	var v *T
	if p.New != nil {
		v = p.New()
	} else {
		v = new(T)
	}
	if p.Init != nil {
		p.Init(v)
	}
	return v
}

// Put returns v to the pool.
// If Cleanup is set, it is called before Clear.
func (p *Pointers[T]) Put(v *T) {
	if v != nil {
		if p.Cleanup != nil {
			p.Cleanup(v)
		}
		if p.Clear {
			var zero T
			*v = zero
		}
		p.pool.Put(v)
	}
}
