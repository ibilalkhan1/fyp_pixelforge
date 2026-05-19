package pixelforge_menus

import (
	"fmt"
	"sort"
	"sync"
)

// Kind names the runtime behaviour class of a menu template.
// Overlay menus pause the underlying scene; full-screen menus
// don't (the underlying scene is invisible behind them anyway).
type Kind string

const (
	// KindOverlay menus pause the scene while open (Pause /
	// Inventory / Status). The MenuStack pushes piloop.Pause on
	// open and Resumes on close.
	KindOverlay Kind = "overlay"

	// KindFullScreen menus replace the scene view entirely
	// (Title / Game-Over / High-Score / Save-Game / Load-Game /
	// Stage-Select). No pause coordination needed.
	KindFullScreen Kind = "full_screen"
)

// Template carries the per-menu-kind contract. SelectionCount
// returns how many options the active menu instance has (used by
// Nav to bound the cursor); Apply runs the option's effect when
// the player presses A.
//
// Parameters live in MenuConfig.Parameters; the template's
// SelectionCount + Apply read from there so a single template
// instance composes with arbitrary per-menu data.
type Template struct {
	// Name is the registered identifier MenuConfig.Template refs.
	Name string

	// Kind drives the pause-coordination behaviour.
	Kind Kind

	// SelectionCount returns how many selectable options the
	// active menu has, given its parameter map. Nav uses this to
	// clamp the cursor.
	SelectionCount func(params map[string]any) int

	// Apply is invoked when the player picks option idx. The
	// implementation typically dispatches a verb-recipe via the
	// supplied dispatch callback (passed through from the host).
	Apply func(idx int, params map[string]any) (verbName string, verbArgs map[string]any)
}

// CanonicalTemplateNames lists the nine NES-canonical templates
// the brainstorm prescribes. Exposed so tests can iterate the
// expected set without re-reading the docs.
var CanonicalTemplateNames = []string{
	"title", "game_over", "high_score", "pause",
	"save_game", "load_game", "inventory", "status", "stage_select",
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Template{}
)

// RegisterTemplate stores a template under its Name. Panics on
// duplicate registration (mirrors pfcomponent.RegisterWidget's
// register-time discipline). Production registrations happen in
// init(); tests use ResetForTest to clear between runs.
func RegisterTemplate(t Template) {
	if t.Name == "" {
		panic("pixelforge_menus.RegisterTemplate: template name must be non-empty")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, ok := registry[t.Name]; ok {
		panic(fmt.Sprintf("pixelforge_menus.RegisterTemplate: %q already registered", t.Name))
	}
	registry[t.Name] = t
}

// LookupTemplate returns the registered template + true; false
// when the name is unknown. Unknown menu types in the project
// fall back to a "(unknown template)" placeholder in the studio.
func LookupTemplate(name string) (Template, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	t, ok := registry[name]
	return t, ok
}

// AllTemplates returns every registered template name sorted
// alphabetically. The menus workspace's template picker uses this
// to populate its dropdown.
func AllTemplates() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]string, 0, len(registry))
	for name := range registry {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// ResetForTest clears the registry — test-only.
func ResetForTest() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = map[string]Template{}
	// Re-register the canonical templates so subsequent tests see
	// a populated registry without re-running init().
	registerCanonical()
}

// registerCanonical installs the nine NES-canonical templates.
// Called from init() at package load and from ResetForTest after
// the registry is cleared.
func registerCanonical() {
	for _, name := range CanonicalTemplateNames {
		// Capture local for closure stability.
		nm := name
		registry[nm] = Template{
			Name:           nm,
			Kind:           templateKindFor(nm),
			SelectionCount: defaultSelectionCount(nm),
			Apply:          defaultApply(nm),
		}
	}
}

func templateKindFor(name string) Kind {
	switch name {
	case "pause", "inventory", "status":
		return KindOverlay
	default:
		return KindFullScreen
	}
}

// defaultSelectionCount returns per-template SelectionCount logic.
// inventory walks the parameter map's "items" key; save/load walks
// a "slots" key; the rest carry static counts.
func defaultSelectionCount(name string) func(params map[string]any) int {
	switch name {
	case "title":
		// Title screen: Start / Continue / Options (3 default).
		return func(p map[string]any) int { return intOrDefault(p, "option_count", 3) }
	case "game_over":
		return func(p map[string]any) int { return intOrDefault(p, "option_count", 2) }
	case "high_score":
		return func(p map[string]any) int { return intOrDefault(p, "row_count", 10) }
	case "pause":
		return func(p map[string]any) int { return intOrDefault(p, "option_count", 3) }
	case "save_game", "load_game":
		// Three slots + autosave (load only).
		return func(p map[string]any) int {
			if name == "load_game" {
				return 4
			}
			return 3
		}
	case "inventory":
		return func(p map[string]any) int {
			if items, ok := p["items"].([]any); ok {
				return len(items)
			}
			return 0
		}
	case "status":
		return func(p map[string]any) int { return 1 } // status pages are usually single-screen
	case "stage_select":
		return func(p map[string]any) int { return intOrDefault(p, "stage_count", 8) }
	}
	return func(p map[string]any) int { return 1 }
}

// defaultApply returns a per-template Apply that resolves the
// picked option into a verb-name / args pair. The host engine
// dispatches the verb through its catalog. For v1 the Apply
// returns ("", nil) for menus whose effects are template-defined
// (the host wires their behaviours separately).
func defaultApply(name string) func(idx int, params map[string]any) (string, map[string]any) {
	switch name {
	case "title":
		return func(idx int, p map[string]any) (string, map[string]any) {
			// Standard Title menu: 0=start, 1=continue, 2=options.
			switch idx {
			case 0:
				return "scene_change", map[string]any{"target": "level1"}
			case 1:
				return "load_slot", map[string]any{"slot": "autosave"}
			}
			return "", nil
		}
	case "save_game":
		return func(idx int, p map[string]any) (string, map[string]any) {
			return "save_now", map[string]any{"slot": fmt.Sprintf("slot%d", idx+1)}
		}
	case "load_game":
		return func(idx int, p map[string]any) (string, map[string]any) {
			if idx == 3 {
				return "load_slot", map[string]any{"slot": "autosave"}
			}
			return "load_slot", map[string]any{"slot": fmt.Sprintf("slot%d", idx+1)}
		}
	}
	return func(idx int, p map[string]any) (string, map[string]any) { return "", nil }
}

func intOrDefault(p map[string]any, key string, def int) int {
	if p == nil {
		return def
	}
	v, ok := p[key]
	if !ok {
		return def
	}
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	}
	return def
}

func init() {
	registryMu.Lock()
	registerCanonical()
	registryMu.Unlock()
}
