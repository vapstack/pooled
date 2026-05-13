package pooled

import "sync"

// Maps pools map[K]V values.
//
// Zero value is ready to use. It must not be copied after first use.
type Maps[K comparable, V any] struct {
	NewCap  int           // NewCap sets initial capacity used when allocating a new map.
	MaxLen  int           // MaxLen limits maximum length retained in the pool
	Cleanup func(map[K]V) // Cleanup is called by Put before clear and MaxLen checks

	_    noCopy
	pool sync.Pool
}

// Get returns a pooled map or allocates a new one.
func (p *Maps[K, V]) Get() map[K]V {
	if pv := p.pool.Get(); pv != nil {
		return pv.(map[K]V)
	}
	return make(map[K]V, max(p.NewCap, 32))
}

// Put clears m and returns it to the pool unless it exceeds MaxLen.
// If Cleanup is set, it is called before clearing and discard checks.
func (p *Maps[K, V]) Put(m map[K]V) {
	if m != nil {
		l := len(m)
		if p.Cleanup != nil {
			p.Cleanup(m)
		}
		if p.MaxLen > 0 && l > p.MaxLen {
			return
		}
		clear(m)
		p.pool.Put(m)
	}
}
