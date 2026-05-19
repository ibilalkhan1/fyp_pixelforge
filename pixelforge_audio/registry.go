package pixelforge_audio

import (
	"sort"
	"sync"
)

// sampleRegistry is the process-wide name→*Sample lookup the
// capsule runtime populates at boot. Recipes published on
// "audio/play_sound" carry a sample name (string); the subscriber
// turns that into a *Sample via LookupSample then dispatches to
// the existing Play API.
//
// Pattern mirrors pixelforge_menus/registry.go — sync.RWMutex +
// Register/Lookup/All + ResetForTest. Unlike the menus registry,
// Register replaces existing entries rather than panicking; the
// capsule's Boot is idempotent and may re-register the same names
// across studio-preview reloads.
var (
	sampleRegistryMu sync.RWMutex
	sampleRegistry   = map[string]*Sample{}
)

// RegisterSample stores sample under name. Empty names are ignored
// so a malformed project doesn't pollute the registry with a "" key.
// Re-registration with the same name replaces the prior entry so
// the studio preview can rebuild registries on every cart reload
// without leaking stale samples.
func RegisterSample(name string, sample *Sample) {
	if name == "" {
		return
	}
	sampleRegistryMu.Lock()
	defer sampleRegistryMu.Unlock()
	sampleRegistry[name] = sample
}

// LookupSample returns the *Sample registered under name, or nil
// when the name is unknown. Callers must nil-check; the verb
// "audio/play_sound" subscriber treats a nil return as a no-op so
// recipes targeting an unloaded sample never crash a shipped game.
func LookupSample(name string) *Sample {
	sampleRegistryMu.RLock()
	defer sampleRegistryMu.RUnlock()
	return sampleRegistry[name]
}

// AllSampleNames returns every registered sample name sorted
// alphabetically. Used by the topic-catalog diagnostics overlay
// and by tests asserting registry contents after Boot.
func AllSampleNames() []string {
	sampleRegistryMu.RLock()
	defer sampleRegistryMu.RUnlock()
	out := make([]string, 0, len(sampleRegistry))
	for name := range sampleRegistry {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// ResetSampleRegistryForTest clears the sample registry. Test-only;
// production callers never need to clear (capsule processes are
// single-shot — the registry's lifetime matches the process).
func ResetSampleRegistryForTest() {
	sampleRegistryMu.Lock()
	defer sampleRegistryMu.Unlock()
	sampleRegistry = map[string]*Sample{}
}
