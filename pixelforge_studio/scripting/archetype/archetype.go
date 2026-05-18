// Package archetype is the per-archetype trigger-slot template
// registry consumed by the character-sheet inspector (U8).
//
// Each Template binds an archetype name (Player, Enemy, Pickup,
// Hazard, NPC, World — the closed v1 set per the brainstorm) to an
// ordered list of TriggerSlot entries describing which gameplay
// moments the inspector exposes for that role, what verb recipe is
// pre-filled by default, and which recipes from the catalog should
// appear in the slot's dropdown.
//
// Registration follows the same package-level init() pattern as
// pixelforge_studio/scripting/catalog (catalog.go:84-120) — duplicate
// names log a warning and overwrite so test reloads and overrides
// behave identically across both registries.
package archetype

import (
	"log"
	"sort"
	"sync"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
)

// TriggerSlot describes one row of the character-sheet inspector for
// a given archetype: the slot's internal trigger name (matches one of
// the catalog.Trigger* constants), the human label rendered in the
// inspector, the verb recipe name pre-filled before the designer
// touches anything, and the list of recipe names that should appear in
// the slot's verb-pick dropdown.
//
// DefaultVerb may be empty when the spec deliberately leaves the slot
// blank for the designer to fill in (NPC's when_interacted, for
// example, waits for idea #6 to wire up open_dialogue:<name>).
// RelevantRecipes is populated at init time by walking
// catalog.AllRecipes and filtering by RelevantTriggers.
type TriggerSlot struct {
	Name            string
	DisplayName     string
	DefaultVerb     string
	RelevantRecipes []string
}

// Template binds an archetype name to its ordered TriggerSlot list.
// The inspector renders slots top-to-bottom in the declared order so
// the most-likely-edited rows surface first (per-archetype designer
// flow established in the brainstorm).
type Template struct {
	Archetype    string
	TriggerSlots []TriggerSlot
}

// Archetype name constants. The six values are the closed v1 set
// declared in the brainstorm and re-asserted as a Key Decision in the
// plan; expansion to a seventh archetype is a code change, not a
// designer choice. The strings match pixelforge_project's archetype
// constants verbatim so schema round-trip stays straightforward.
const (
	ArchetypePlayer = "Player"
	ArchetypeEnemy  = "Enemy"
	ArchetypePickup = "Pickup"
	ArchetypeHazard = "Hazard"
	ArchetypeNPC    = "NPC"
	ArchetypeWorld  = "World"
)

var (
	tmplMu    sync.RWMutex
	templates = map[string]Template{}
)

// RegisterArchetype adds (t.Archetype, t) to the template registry.
// An empty Archetype field is rejected silently to match the
// catalog.RegisterAction shape; duplicate names log a warning and
// overwrite. RegisterArchetype does not populate RelevantRecipes —
// callers register the template with the slot's RelevantRecipes
// already filled (the built-in init() block in builtin_archetypes.go
// walks catalog.AllRecipes to do this).
func RegisterArchetype(t Template) {
	if t.Archetype == "" {
		return
	}
	tmplMu.Lock()
	defer tmplMu.Unlock()
	if _, exists := templates[t.Archetype]; exists {
		log.Printf("[archetype] template %q re-registered; overwriting", t.Archetype)
	}
	templates[t.Archetype] = t
}

// LookupArchetype returns the Template registered under name and
// whether one is present. The inspector calls this for the entity's
// Archetype field; an unknown archetype on a loaded project falls
// back to ArchetypeWorld per the U2 schema sanitize rules.
func LookupArchetype(name string) (Template, bool) {
	tmplMu.RLock()
	defer tmplMu.RUnlock()
	t, ok := templates[name]
	return t, ok
}

// AllArchetypes returns every registered Template in alphabetical
// order of Archetype name. Studio dropdowns surface their own labels;
// this helper exists for the archetype-pick combo (U8) and for tests.
func AllArchetypes() []Template {
	tmplMu.RLock()
	defer tmplMu.RUnlock()
	names := make([]string, 0, len(templates))
	for k := range templates {
		names = append(names, k)
	}
	sort.Strings(names)
	out := make([]Template, 0, len(names))
	for _, n := range names {
		out = append(out, templates[n])
	}
	return out
}

// ResetForTest clears the template registry. Test-only; callers that
// exercise registration semantics restore the baseline by calling
// RegisterBuiltinArchetypes (or simply not invoking this helper).
func ResetForTest() {
	tmplMu.Lock()
	defer tmplMu.Unlock()
	templates = map[string]Template{}
}

// relevantRecipesFor returns the recipe names from catalog.AllRecipes
// whose RelevantTriggers list includes trigger. The order matches
// catalog.AllRecipes (alphabetical) so dropdowns render
// deterministically across runs.
//
// Exported via the internal builtin_archetypes.go init path; kept
// unexported because external callers should declare their slots
// statically rather than introspect the catalog at runtime.
func relevantRecipesFor(trigger string) []string {
	names := catalog.AllRecipes()
	out := make([]string, 0, len(names))
	for _, name := range names {
		r, ok := catalog.LookupRecipe(name)
		if !ok {
			continue
		}
		for _, t := range r.RelevantTriggers {
			if t == trigger {
				out = append(out, name)
				break
			}
		}
	}
	return out
}
