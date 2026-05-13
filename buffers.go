package pooled

import (
	"bytes"
	"sync"
)

// Buffers pools *bytes.Buffer.
//
// Zero value is ready to use. It must not be copied after first use.
type Buffers struct {
	MinCap int // minimum capacity for newly allocated buffers
	MaxCap int // maximum capacity retained in the pool

	_    noCopy
	pool sync.Pool
}

// Get returns a reset pooled buffer or allocates a new one.
func (p *Buffers) Get() *bytes.Buffer {
	if v := p.pool.Get(); v != nil {
		return v.(*bytes.Buffer)
	}
	v := new(bytes.Buffer)
	if p.MinCap > 0 {
		v.Grow(p.MinCap)
	}
	return v
}

// Put resets b and returns it to the pool unless it exceeds MaxCap.
func (p *Buffers) Put(b *bytes.Buffer) {
	if b != nil {
		if p.MaxCap > 0 && b.Cap() > p.MaxCap {
			return
		}
		b.Reset()
		p.pool.Put(b)
	}
}
