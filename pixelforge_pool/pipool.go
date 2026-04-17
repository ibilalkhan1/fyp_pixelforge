// Package pixelforge_pool provides an extremely simple, non-thread-safe pool
// that can be used to reduce heap memory allocations.
package pixelforge_pool

// Pool is a very simple, non-thread-safe object pool.
//
// It can only be used from a single goroutine. Since Pixelforge runs on
// a single goroutine, it can safely be used in pixelforge.Update and pixelforge.Draw.
type Pool[T any] struct {
	objects []*T // LIFO
}

// Get returns an object from the pool.
//
// If the pool is empty, it creates a new zero-value object
// and returns a pointer to it.
func (p *Pool[T]) Get() *T {
	n := len(p.objects)
	if n == 0 {
		var t T
		return &t
	}
	last := p.objects[n-1]
	p.objects = p.objects[:n-1] // decrease len, cap will remain the same
	return last
}

// Put returns an object to the pool.
func (p *Pool[T]) Put(obj *T) {
	p.objects = append(p.objects, obj)
}
