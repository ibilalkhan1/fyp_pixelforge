package capsuleruntime

import (
	"sort"
	"sync"
)

// CreditEntry is one attribution line the auto-injected Credits
// screen renders. Plan-008 U10 assembles the slice at codegen time
// (only CC-BY-licensed assets are included — CC0 carries no
// attribution duty) and embeds a literal in the generated
// capsule.go; the capsule's init calls RegisterCredits to publish
// it to this registry.
//
// The engine's menu system reads via LookupCredits and conditional-
// renders a "Credits" entry on the title screen when the slice is
// non-empty; the WASM bundler's HTML reads the same data through a
// "View Credits" splash button.
type CreditEntry struct {
	// Name is the project-scoped asset identifier (sprite name,
	// audio sample name). Rendered as the row's left column.
	Name string

	// License is the SPDX-style license identifier (CC0, CC-BY-4.0,
	// CC-BY-SA-4.0, etc.). Rendered next to Name so designers can
	// audit at a glance.
	License string

	// Author is the asset's creator. Required for CC-BY rows;
	// optional for CC0 (which is omitted from credits anyway).
	Author string

	// SourceURL is the original asset URL the manifest declared.
	// Rendered as a clickable link on platforms that support one
	// (WASM "View Credits" overlay), or as plain text otherwise.
	SourceURL string
}

// creditsRegistry is the process-wide credits store the generated
// capsule populates at init. Empty when the project uses only CC0
// assets — the menu system reads this to decide whether to render
// a "Credits" entry at all.
var (
	creditsRegistryMu sync.RWMutex
	creditsRegistry   = []CreditEntry{}
)

// RegisterCredits replaces the credits slice with entries. Called
// from generated capsule init code via the codegen-emitted
// capsuleCredits literal; tests reset between runs via
// ResetCreditsForTest.
func RegisterCredits(entries []CreditEntry) {
	creditsRegistryMu.Lock()
	defer creditsRegistryMu.Unlock()
	// Defensive copy so the caller can't mutate the registry
	// out-of-band.
	out := make([]CreditEntry, len(entries))
	copy(out, entries)
	// Stable sort by Name so the order is deterministic regardless
	// of how the generator walked the project (Go map iteration is
	// unordered for items + bindings).
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	creditsRegistry = out
}

// LookupCredits returns a copy of the registered credits slice.
// Empty when no credits have been registered (CC0-only project,
// or codegen hasn't run yet in a test context).
func LookupCredits() []CreditEntry {
	creditsRegistryMu.RLock()
	defer creditsRegistryMu.RUnlock()
	out := make([]CreditEntry, len(creditsRegistry))
	copy(out, creditsRegistry)
	return out
}

// HasCredits reports whether the registry has any entries. Used
// by the menu system to decide whether to render a Credits entry
// on the title screen.
func HasCredits() bool {
	creditsRegistryMu.RLock()
	defer creditsRegistryMu.RUnlock()
	return len(creditsRegistry) > 0
}

// ResetCreditsForTest clears the credits registry. Test-only.
func ResetCreditsForTest() {
	creditsRegistryMu.Lock()
	defer creditsRegistryMu.Unlock()
	creditsRegistry = nil
}
