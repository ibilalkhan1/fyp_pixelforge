// Package blackboard hosts the Studio's Blackboard inspector workspace.
// The workspace surfaces live key/value state from the process-wide
// pixelforge_blackboard.Default() instance: canonical keys (score,
// lives, health, current_scene, player_x, player_y per R11) render at
// the top with typed editor widgets driven by the key's declared
// CanonicalKeySpec.Type; arbitrary designer-added keys appear below as
// a read-only flat string -> value list. Editing designer-added keys
// is intentionally deferred — those are mutated by verbs at runtime,
// not by the inspector (per the U9 Approach in the plan).
//
// Live updates. Render reads from the blackboard each frame, so any
// mutation observed since the previous frame surfaces naturally. The
// workspace additionally subscribes to the blackboard's change target
// via SubscribeAll so a future invalidate hook (or a "last-changed"
// indicator) can react to mutations without polling. The subscription
// handle is stored on the workspace so a teardown path can unsubscribe
// when the workspace is replaced.
//
// Registration. RegisterWith(e) installs the workspace on the editor;
// pixelforge_studio/main.go calls it alongside palette / capture /
// scripting / input so all five workspaces compose uniformly.
package blackboard

import (
	"fmt"
	"sort"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_blackboard"
	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// blackboardFn is the package-level seam used to resolve the active
// Blackboard instance. Production code points at
// pixelforge_blackboard.Default; tests may swap it for a hermetic
// instance so assertions about Set / Get round-trips do not touch the
// process-wide singleton other packages' tests would observe.
var blackboardFn = pixelforge_blackboard.Default

// Workspace is the Blackboard inspector panel. Implements
// editor.Workspace (Name / DisplayName / Render) so the dockspace picks
// it up the same way palette / capture / scripting / input do.
//
// State is intentionally minimal: every Render reads from the active
// Blackboard via blackboardFn directly. The subscription handle stored
// on changeSub exists only so the workspace can unsubscribe if its
// teardown path ever calls Close (none ships in v1 — workspaces live
// for the editor's lifetime).
type Workspace struct {
	changeSub pievent.Handler
	dirty     bool
}

// NewWorkspace constructs a fresh Blackboard workspace. The workspace
// does not subscribe to the active blackboard on construction —
// subscription is wired in RegisterWith so test code that builds a
// bare Workspace doesn't accidentally attach to the process-wide
// instance.
func NewWorkspace() *Workspace { return &Workspace{} }

// Name is the stable workspace identifier.
func (w *Workspace) Name() string { return "blackboard" }

// DisplayName is the dock window title.
func (w *Workspace) DisplayName() string { return "Blackboard" }

// MarkDirty is called by the subscription closure each time the
// blackboard publishes a BlackboardChange. v1 has no explicit ImGui
// invalidation API (every Begin/End cycle reads fresh values), so the
// flag exists primarily so tests + future indicator widgets can observe
// that a change occurred between renders.
func (w *Workspace) MarkDirty() { w.dirty = true }

// ConsumeDirty returns the workspace's dirty flag and clears it. The
// LiveUpdate test uses this to assert that a blackboard mutation
// triggered the subscription callback.
func (w *Workspace) ConsumeDirty() bool {
	d := w.dirty
	w.dirty = false
	return d
}

// Render emits the Blackboard ImGui window. Canonical keys render first
// with typed widgets driven by CanonicalKeys[key].Type; designer-added
// keys render below as read-only "key: value" text.
//
// When ImGui is not live (the cimgui-go backend hasn't attached, e.g.
// editor.New() in a test), Render is a no-op aside from the dirty
// flag bookkeeping — the test seam runs through Set / Get directly.
func (w *Workspace) Render(e *editor.Editor) {
	if e == nil {
		return
	}
	flags := imgui.WindowFlagsNoCollapse
	if !imgui.BeginV(w.DisplayName(), nil, flags) {
		imgui.End()
		e.SetPanelRect(w.DisplayName(), widgets.Rect{})
		return
	}
	defer imgui.End()
	e.CaptureCurrentWindowRect(w.DisplayName())

	bb := blackboardFn()
	if bb == nil {
		imgui.TextDisabled("(no blackboard)")
		return
	}
	w.dirty = false

	imgui.TextDisabled("Blackboard — live key/value state")
	imgui.Separator()

	// Canonical keys: render in stable alphabetical order so the panel
	// layout doesn't reshuffle between frames when CanonicalKeys map
	// iteration changes order.
	for _, key := range canonicalKeyOrder() {
		spec := pixelforge_blackboard.CanonicalKeys[key]
		current, _ := bb.Lookup(key)
		if current == nil {
			current = spec.Default
		}
		w.renderCanonicalRow(bb, key, spec, current)
	}

	imgui.Separator()
	imgui.TextDisabled("Designer-added keys (read-only)")

	for _, key := range bb.KeysSnapshot() {
		if _, isCanonical := pixelforge_blackboard.CanonicalKeys[key]; isCanonical {
			continue
		}
		v, _ := bb.Lookup(key)
		imgui.Text(fmt.Sprintf("%s: %s", key, formatValue(v)))
	}
}

// renderCanonicalRow emits the typed widget appropriate for spec.Type.
// Edits write back to the blackboard via bb.Set, which fires a
// BlackboardChange that the subscription closure observes (re-flipping
// w.dirty for next frame).
func (w *Workspace) renderCanonicalRow(
	bb *pixelforge_blackboard.Blackboard,
	key string,
	spec pixelforge_blackboard.CanonicalKeySpec,
	current any,
) {
	switch spec.Type {
	case pixelforge_blackboard.TypeInt:
		v := toInt32(current)
		if imgui.InputIntV(key, &v, 1, 10, 0) {
			bb.Set(key, int(v))
		}
	case pixelforge_blackboard.TypeFloat:
		v := toFloat32(current)
		if imgui.InputFloatV(key, &v, 0, 0, "%.3f", 0) {
			bb.Set(key, float64(v))
		}
	case pixelforge_blackboard.TypeString:
		v := toString(current)
		if imgui.InputTextWithHint(key, "", &v, 0, nil) {
			bb.Set(key, v)
		}
	case pixelforge_blackboard.TypeBool:
		v := toBool(current)
		if imgui.Checkbox(key, &v) {
			bb.Set(key, v)
		}
	default:
		// Unknown type — fall back to read-only text so future
		// CanonicalKeyType additions don't crash the workspace.
		imgui.Text(fmt.Sprintf("%s: %s", key, formatValue(current)))
	}
}

// SetCanonical mutates a canonical key on the active blackboard. Test
// mirror of the widget edit flow that does not depend on ImGui.
// Returns false when the key is not canonical (the inspector does not
// edit designer-added keys in v1).
func (w *Workspace) SetCanonical(key string, value any) bool {
	if _, ok := pixelforge_blackboard.CanonicalKeys[key]; !ok {
		return false
	}
	bb := blackboardFn()
	if bb == nil {
		return false
	}
	bb.Set(key, value)
	return true
}

// WidgetKindFor reports the imgui-widget classification the workspace
// renders for a canonical key. Used by tests asserting that, e.g.,
// score renders as an int field rather than a text input.
//
// Returns "" for unknown / non-canonical keys.
func WidgetKindFor(key string) string {
	spec, ok := pixelforge_blackboard.CanonicalKeys[key]
	if !ok {
		return ""
	}
	switch spec.Type {
	case pixelforge_blackboard.TypeInt:
		return "int"
	case pixelforge_blackboard.TypeFloat:
		return "float"
	case pixelforge_blackboard.TypeString:
		return "text"
	case pixelforge_blackboard.TypeBool:
		return "checkbox"
	}
	return ""
}

// RenderedDesignerKeys returns the keys currently held by the active
// blackboard that fall outside the canonical registry, in the same
// stable order Render iterates them. Used by tests asserting that
// designer-added keys appear in the read-only section.
func RenderedDesignerKeys() []string {
	bb := blackboardFn()
	if bb == nil {
		return nil
	}
	out := []string{}
	for _, key := range bb.KeysSnapshot() {
		if _, isCanonical := pixelforge_blackboard.CanonicalKeys[key]; isCanonical {
			continue
		}
		out = append(out, key)
	}
	return out
}

// RenderedDesignerLine returns the exact "key: value" string Render
// would emit for the supplied designer-added key, or "" when the key
// is canonical / absent. Lets tests assert on rendered content without
// driving the live cimgui-go backend.
func RenderedDesignerLine(key string) string {
	if _, isCanonical := pixelforge_blackboard.CanonicalKeys[key]; isCanonical {
		return ""
	}
	bb := blackboardFn()
	if bb == nil {
		return ""
	}
	v, ok := bb.Lookup(key)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s: %s", key, formatValue(v))
}

// RenderedCanonicalLine returns the "key: value" string a canonical
// key would surface (either the stored value or the spec's default
// when the key has not been written yet). Lets tests assert on
// rendered content without driving the live cimgui-go backend.
func RenderedCanonicalLine(key string) string {
	spec, ok := pixelforge_blackboard.CanonicalKeys[key]
	if !ok {
		return ""
	}
	bb := blackboardFn()
	if bb == nil {
		return ""
	}
	v, present := bb.Lookup(key)
	if !present {
		v = spec.Default
	}
	return fmt.Sprintf("%s: %s", key, formatValue(v))
}

// canonicalKeyOrder returns the canonical key names in stable
// alphabetical order. Iterating Go's map[K]V is non-deterministic, so
// rendering directly off CanonicalKeys would shuffle the panel layout
// between frames — undesirable for a state-inspection surface.
func canonicalKeyOrder() []string {
	out := make([]string, 0, len(pixelforge_blackboard.CanonicalKeys))
	for k := range pixelforge_blackboard.CanonicalKeys {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// RegisterWith installs the blackboard workspace on the editor and
// subscribes it to the active Blackboard's change target so future
// "last-changed" / invalidation surfaces can react without polling.
// Mirrors palette.RegisterWith / capture.RegisterWith /
// scripting.RegisterWith / input.RegisterWith so main.go can compose
// the five lines uniformly.
func RegisterWith(e *editor.Editor) *Workspace {
	if e == nil {
		return nil
	}
	w := NewWorkspace()
	e.RegisterWorkspace(w)
	if bb := blackboardFn(); bb != nil {
		// SubscribeAll captures w; the closure flips the dirty flag so
		// callers (tests today, an invalidate-on-change indicator
		// tomorrow) can observe that a mutation happened between
		// renders. We intentionally do not re-bind imgui state from
		// here — Render reads fresh values each frame regardless.
		w.changeSub = bb.Changes().SubscribeAll(func(_ pixelforge_blackboard.BlackboardChange, _ pievent.Handler) {
			w.MarkDirty()
		})
	}
	return w
}

// ---- value coercion helpers ------------------------------------------------
//
// Mirrors of the unexported helpers in pixelforge_studio/editor/inspector.go.
// Duplicated here so the workspace package does not depend on private editor
// internals (and so U8 is free to evolve those helpers without rippling).

func formatValue(v any) string {
	if v == nil {
		return "-"
	}
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		return fmt.Sprintf("%.2f", x)
	case float32:
		return fmt.Sprintf("%.2f", x)
	case int:
		return fmt.Sprintf("%d", x)
	case int32:
		return fmt.Sprintf("%d", x)
	case int64:
		return fmt.Sprintf("%d", x)
	default:
		return fmt.Sprintf("%v", x)
	}
}

func toInt32(v any) int32 {
	switch x := v.(type) {
	case int:
		return int32(x)
	case int32:
		return x
	case int64:
		return int32(x)
	case float64:
		return int32(x)
	case float32:
		return int32(x)
	default:
		return 0
	}
}

func toFloat32(v any) float32 {
	switch x := v.(type) {
	case float64:
		return float32(x)
	case float32:
		return x
	case int:
		return float32(x)
	case int32:
		return float32(x)
	case int64:
		return float32(x)
	default:
		return 0
	}
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", x)
	}
}

func toBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case float64:
		return x != 0
	case int:
		return x != 0
	default:
		return false
	}
}
