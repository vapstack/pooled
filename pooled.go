// Package pooled provides typed sync.Pool wrappers for temporary buffers,
// slices, maps, and pointers.
package pooled

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
