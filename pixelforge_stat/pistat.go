// Package pistat provides information about current resource usage,
// such as CPU and RAM.
//
// This package is not supported in browsers.
package pixelforge_stat

import (
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_stat/internal"
	"time"

	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
)

// Data refreshed every 500 ms.
var (
	CPU int // Percentage of CPU time used by the game and Go runtime.
	// 100 means full usage of one core or cumulative usage across multiple cores.
	// Values may exceed 100 when multiple cores are used.

	MemoryMB int // Amount of memory used in megabytes (including unused allocations
	// not yet collected by the GC).
)

// Data refreshed every frame.
var (
	Allocs uint64 // Number of objects allocated during the last frame.
)

var lastStatGatherTime time.Time

var started = false

var handler pievent.Handler

// Start begins monitoring resource usage.
func Start() {
	if started {
		return
	}
	started = true
	handler = piloop.DebugTarget().Subscribe(piloop.EventUpdate, monitorResourceUsage)
}

// Stop stops monitoring resource usage.
func Stop() {
	if !started {
		return
	}
	piloop.DebugTarget().Unsubscribe(handler)
	started = false
}

func monitorResourceUsage(piloop.Event, pievent.Handler) {
	now := time.Now()
	// update once every 0.5s:
	if now.After(lastStatGatherTime.Add(500 * time.Millisecond)) {
		CPU = internal.GetCPU()
		MemoryMB = internal.GetMemoryMB()

		lastStatGatherTime = now
	}
	// update each frame
	Allocs = internal.GetAllocatedObjectsCount()
}
