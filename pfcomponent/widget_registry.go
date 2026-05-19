package pfcomponent

import (
	"fmt"
	"sync"
)

// Drawer renders a custom field widget inside the inspector. The
// implementation is supplied by an upstream package (the editor) and
// registered with RegisterWidget. The inspector calls the drawer
// whenever a field's pf-tag carries widget=<name>.
//
// Returning true signals that the drawer mutated underlying state, so
// the inspector marks the project dirty.
type Drawer func(ctx DrawerContext) bool

// DrawerContext carries everything a custom drawer might need. The
// concrete editor / widgets types are passed through Extras as `any`
// so pfcomponent stays at the bottom of the import graph (no editor
// or imgui dependency creeps into the component-registry layer).
//
// Fields:
//   - Owner: the typed parent of the dispatched field. For entity-
//     component dispatch this is the component's Values map (the
//     existing inspector convention); for non-component dispatch
//     surfaces (e.g. a TileAtlas inside Scene.TileAtlases) the
//     caller passes a typed pointer such as *TileAtlas. Drawers
//     type-assert as needed.
//   - FieldName: the Go field name being rendered (FieldMetadata.Name).
//   - JSONKey: the wire-format key for the field (FieldMetadata.JSONKey).
//   - FieldValue: the current value of the dispatched field. Nil when
//     no value has been stored yet.
//   - Extras: editor-specific dependencies the drawer needs, keyed by
//     a stable string. Conventional keys are documented at the drawer
//     site (the editor package); callers may inject anything.
type DrawerContext struct {
	Owner      any
	FieldName  string
	JSONKey    string
	FieldValue any
	Extras     map[string]any
}

var (
	widgetMu       sync.RWMutex
	widgetRegistry = map[string]Drawer{}
)

// RegisterWidget stores drawer under name so the inspector can
// dispatch to it when a field tag carries widget=<name>. Panics on
// duplicate registration of name — matches Register[T]'s discipline
// so register-time mistakes surface immediately rather than as
// silent override bugs at edit time. Production registrations live
// in central init() blocks that run exactly once per process; tests
// call ResetWidgetsForTest between runs.
func RegisterWidget(name string, drawer Drawer) {
	if name == "" {
		panic("pfcomponent.RegisterWidget: name must be non-empty")
	}
	if drawer == nil {
		panic(fmt.Sprintf("pfcomponent.RegisterWidget: %q drawer is nil", name))
	}

	widgetMu.Lock()
	defer widgetMu.Unlock()

	if _, ok := widgetRegistry[name]; ok {
		panic(fmt.Sprintf(
			"pfcomponent.RegisterWidget: name %q already registered",
			name))
	}
	widgetRegistry[name] = drawer
}

// LookupWidget returns the drawer registered under name. ok is false
// when no drawer exists; the inspector treats this as a recoverable
// fallback (render the field as read-only text) rather than a panic.
func LookupWidget(name string) (Drawer, bool) {
	widgetMu.RLock()
	defer widgetMu.RUnlock()
	d, ok := widgetRegistry[name]
	return d, ok
}

// ResetWidgetsForTest clears the widget registry. Test-only; mirrors
// ResetForTest on the component registry.
func ResetWidgetsForTest() {
	widgetMu.Lock()
	defer widgetMu.Unlock()
	widgetRegistry = map[string]Drawer{}
}
