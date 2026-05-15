package pixelforge_event

import (
	"log"
	"sort"
	"sync"
)

// Inspectable is the non-generic subset of Target[T] that the
// scripting topic catalog (M5) iterates uniformly. Target[T] already
// implements both methods, so registering a target is just a type
// assertion at the call site.
type Inspectable interface {
	SubscriberCount() int
	PublishCount() uint64
}

// TargetInfo is a snapshot of an Inspectable's metadata, returned by
// EnumerateTargets so callers don't need to retain the underlying
// target reference.
type TargetInfo struct {
	Name        string
	Subscribers int
	Publishes   uint64
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Inspectable{}
)

// RegisterTarget adds (name, target) to the process-wide registry the
// M5 topic catalog enumerates. Duplicate names log a warning and
// overwrite the prior registration (engines under test sometimes
// import packages twice).
func RegisterTarget(name string, target Inspectable) {
	if name == "" || target == nil {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[name]; exists {
		log.Printf("[pievent] target %q re-registered; overwriting prior entry", name)
	}
	registry[name] = target
}

// LookupTarget returns the registered target with the given name, or
// nil if no target is registered under that name.
func LookupTarget(name string) Inspectable {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[name]
}

// EnumerateTargets returns a snapshot of every registered target's
// metadata, sorted alphabetically by name so callers can display a
// stable list. Returns an empty slice (never nil) when the registry
// is empty.
func EnumerateTargets() []TargetInfo {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]TargetInfo, 0, len(registry))
	for name, t := range registry {
		out = append(out, TargetInfo{
			Name:        name,
			Subscribers: t.SubscriberCount(),
			Publishes:   t.PublishCount(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ResetRegistryForTest empties the registry. Test-only; production
// code never deregisters.
func ResetRegistryForTest() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = map[string]Inspectable{}
}
