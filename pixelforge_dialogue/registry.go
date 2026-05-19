package pixelforge_dialogue

import (
	"sort"
	"sync"
)

// scriptRegistry is the process-wide name→*Tree lookup the capsule
// runtime populates at boot. The verb-recipe "ui/open_dialogue"
// publishes {script_id: name}; the subscriber resolves the name to
// a *Tree via LookupScript and hands it to the renderer.
//
// Pattern mirrors pixelforge_menus/registry.go — sync.RWMutex +
// Register/Lookup/All + ResetForTest. Register replaces existing
// entries so studio-preview reloads can rebuild the registry
// idempotently.
var (
	scriptRegistryMu sync.RWMutex
	scriptRegistry   = map[string]*Tree{}
)

// RegisterScript stores tree under name. Empty names are ignored so
// a malformed project doesn't pollute the registry with a "" key.
// Re-registration replaces the prior entry; the capsule's Boot is
// idempotent across studio-preview reloads.
func RegisterScript(name string, tree *Tree) {
	if name == "" {
		return
	}
	scriptRegistryMu.Lock()
	defer scriptRegistryMu.Unlock()
	scriptRegistry[name] = tree
}

// LookupScript returns the *Tree registered under name, or nil when
// the name is unknown. The "ui/open_dialogue" subscriber treats a
// nil return as a no-op so recipes targeting an unloaded dialogue
// never crash a shipped game.
func LookupScript(name string) *Tree {
	scriptRegistryMu.RLock()
	defer scriptRegistryMu.RUnlock()
	return scriptRegistry[name]
}

// AllScriptNames returns every registered script name sorted
// alphabetically. Used by diagnostics and tests asserting registry
// contents after Boot.
func AllScriptNames() []string {
	scriptRegistryMu.RLock()
	defer scriptRegistryMu.RUnlock()
	out := make([]string, 0, len(scriptRegistry))
	for name := range scriptRegistry {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// ResetScriptRegistryForTest clears the script registry. Test-only.
func ResetScriptRegistryForTest() {
	scriptRegistryMu.Lock()
	defer scriptRegistryMu.Unlock()
	scriptRegistry = map[string]*Tree{}
}
