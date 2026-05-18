// Package input hosts the Settings -> Input workspace. The workspace
// lets a designer edit the active Project.InputMap one intent at a
// time: per-intent rows show keyboard, gamepad, and modifier dropdowns
// that mutate the underlying InputBinding in place. Every edit calls
// editor.MarkDirty and pixelforge_input.Recompile so the live intent
// layer reflects the new map without restarting the engine (per
// docs/solutions/always-on-game-embedding.md).
//
// The workspace is registered alongside palette / capture / scripting
// from pixelforge_studio/main.go via RegisterWith(e).
//
// Key-name canonicalization. Project.DefaultInputMap (U2) uses friendly
// names like "Space", "ArrowUp", "Escape", and "GamepadA" — but
// pixelforge_key publishes raw values like " ", "Up", and "Esc", and
// pixelforge_pad publishes bare letters like "A". The U3 compiler
// matches by exact string equality, so the friendly defaults never
// fire. To keep the dropdown choices readable while still emitting
// strings the compiler will actually match, the workspace shows
// friendly labels in the combo box and stores the underlying
// pixelforge_key.Key / pixelforge_pad.Button string in the InputMap.
// keyboardOptions and gamepadOptions below own that translation; the
// reverse lookup (stored value -> displayed label) lives in
// labelForKeyValue / labelForButtonValue.
package input

import (
	"sort"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_input"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_pad"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// recompileFn is the package-level seam used to invoke the intent
// compiler after an edit. Production code points at
// pixelforge_input.Recompile; tests swap it for a counter / capture
// closure so TestWorkspace_EditCallsRecompile can verify the call
// without driving the global pievent registry.
var recompileFn = pixelforge_input.Recompile

// Workspace is the Settings -> Input panel. Implements editor.Workspace
// (Name / DisplayName / Render) so the dockspace picks it up the same
// way palette / capture / scripting do.
type Workspace struct{}

// NewWorkspace constructs a fresh input workspace. The workspace is
// stateless: every Render reads from e.Project().InputMap directly.
func NewWorkspace() *Workspace { return &Workspace{} }

// Name is the stable workspace identifier.
func (w *Workspace) Name() string { return "input" }

// DisplayName is the dock window title.
func (w *Workspace) DisplayName() string { return "Input" }

// Render emits the Input ImGui window. Each row shows one intent with
// a keyboard / gamepad / modifier dropdown; any change mutates the
// underlying InputBinding, marks the editor dirty, and calls Recompile
// so the intent layer reflects the edit on the next physical event.
//
// If the active project has no InputMap (e.g. a brand-new project
// before applyDefaults runs), Render populates it with the default
// bindings on first edit so the dropdowns have something to mutate.
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

	p := e.Project()
	if p == nil {
		imgui.TextDisabled("(no project)")
		return
	}
	if len(p.InputMap) == 0 {
		// Lazy default: a brand-new project that hasn't been through
		// applyDefaults (e.g. editor.New() in a test) gets the same
		// 9 NES-style bindings load would have applied. Surfacing
		// these here keeps the panel useful without forcing every
		// editor constructor to call applyDefaults.
		p.InputMap = pixelforge_project.DefaultInputMap()
	}

	imgui.TextDisabled("Settings / Input — edit per-intent bindings")
	imgui.Separator()

	if imgui.BeginTableV("##input-bindings", 4, imgui.TableFlagsBordersInnerH|imgui.TableFlagsSizingStretchProp, imgui.NewVec2(0, 0), 0) {
		imgui.TableSetupColumn("Intent")
		imgui.TableSetupColumn("Keyboard")
		imgui.TableSetupColumn("Gamepad")
		imgui.TableSetupColumn("Modifier")
		imgui.TableHeadersRow()

		for i := range p.InputMap {
			w.renderRow(e, p, i)
		}
		imgui.EndTable()
	}

	imgui.Separator()
	if imgui.Button("Restore defaults") {
		w.restoreDefaults(e, p)
	}
}

func (w *Workspace) renderRow(e *editor.Editor, p *pixelforge_project.Project, idx int) {
	b := &p.InputMap[idx]

	imgui.TableNextRow()

	imgui.TableNextColumn()
	imgui.Text(intentDisplayName(b.Intent))

	imgui.TableNextColumn()
	// v1 shows the first keyboard binding only — multi-binding edit is
	// deferred (see Scope Boundaries in the plan). The underlying
	// slice still preserves any extra keys until an edit overwrites
	// them.
	current := ""
	if len(b.Keyboard) > 0 {
		current = b.Keyboard[0]
	}
	if picked, changed := keyboardCombo("##kb-"+b.Intent, current); changed {
		if picked == "" {
			b.Keyboard = nil
		} else {
			b.Keyboard = []string{picked}
		}
		w.afterEdit(e, p)
	}

	imgui.TableNextColumn()
	if picked, changed := gamepadCombo("##pad-"+b.Intent, b.GamepadButton); changed {
		b.GamepadButton = picked
		w.afterEdit(e, p)
	}

	imgui.TableNextColumn()
	if picked, changed := modifierCombo("##mod-"+b.Intent, b.Modifier); changed {
		b.Modifier = picked
		w.afterEdit(e, p)
	}
}

// SetBindingKeyboard mutates the keyboard slot of the binding whose
// Intent matches intent and runs the post-edit hooks (MarkDirty +
// Recompile). Returns true on success. Test-friendly mirror of the
// dropdown edit flow that does not depend on ImGui.
func (w *Workspace) SetBindingKeyboard(e *editor.Editor, intent string, keyValue string) bool {
	return w.editBinding(e, intent, func(b *pixelforge_project.InputBinding) {
		if keyValue == "" {
			b.Keyboard = nil
			return
		}
		b.Keyboard = []string{keyValue}
	})
}

// SetBindingGamepad mutates the gamepad slot. Mirror of SetBindingKeyboard.
func (w *Workspace) SetBindingGamepad(e *editor.Editor, intent string, btnValue string) bool {
	return w.editBinding(e, intent, func(b *pixelforge_project.InputBinding) {
		b.GamepadButton = btnValue
	})
}

// SetBindingModifier mutates the modifier slot. Mirror of SetBindingKeyboard.
func (w *Workspace) SetBindingModifier(e *editor.Editor, intent string, modifier string) bool {
	return w.editBinding(e, intent, func(b *pixelforge_project.InputBinding) {
		b.Modifier = modifier
	})
}

// RestoreDefaults replaces the active project's InputMap with the
// shipped defaults and runs the post-edit hooks. Test mirror of the
// "Restore defaults" button.
func (w *Workspace) RestoreDefaults(e *editor.Editor) {
	if e == nil || e.Project() == nil {
		return
	}
	w.restoreDefaults(e, e.Project())
}

func (w *Workspace) editBinding(e *editor.Editor, intent string, fn func(*pixelforge_project.InputBinding)) bool {
	if e == nil || e.Project() == nil {
		return false
	}
	p := e.Project()
	if len(p.InputMap) == 0 {
		p.InputMap = pixelforge_project.DefaultInputMap()
	}
	for i := range p.InputMap {
		if p.InputMap[i].Intent != intent {
			continue
		}
		fn(&p.InputMap[i])
		w.afterEdit(e, p)
		return true
	}
	return false
}

func (w *Workspace) restoreDefaults(e *editor.Editor, p *pixelforge_project.Project) {
	p.InputMap = pixelforge_project.DefaultInputMap()
	w.afterEdit(e, p)
}

// afterEdit centralises the dirty + recompile contract. Every dropdown
// edit and the restore-defaults button route through it so the live
// intent layer stays in sync with the InputMap (per F4 / U7 plan).
func (w *Workspace) afterEdit(e *editor.Editor, p *pixelforge_project.Project) {
	e.MarkDirty()
	recompileFn(p.InputMap)
}

// intentDisplayName converts an "input/jump" target name to a
// human-friendly "Jump" label. Falls back to the raw intent name for
// any future-added intent not enumerated here.
func intentDisplayName(intent string) string {
	switch intent {
	case pixelforge_input.IntentJump:
		return "Jump"
	case pixelforge_input.IntentAttack:
		return "Attack"
	case pixelforge_input.IntentUp:
		return "Up"
	case pixelforge_input.IntentDown:
		return "Down"
	case pixelforge_input.IntentLeft:
		return "Left"
	case pixelforge_input.IntentRight:
		return "Right"
	case pixelforge_input.IntentUse:
		return "Use"
	case pixelforge_input.IntentMenu:
		return "Menu"
	case pixelforge_input.IntentStart:
		return "Start"
	}
	return intent
}

// keyOption pairs a friendly label shown in the dropdown with the
// underlying string the U3 compiler matches on. Stored as a flat slice
// (rather than a map) so the dropdown order is deterministic and
// readable: directionals first, then alpha, then digits and friendlies.
type keyOption struct {
	Label string
	Value string
}

// keyboardOptions is the canonical "what a designer can pick" list for
// keyboard bindings. The Label column shows in the combo; the Value
// column is what we store in InputBinding.Keyboard so the compiler's
// string-equality match against pixelforge_key.Key actually fires.
//
// Bridging defaults: DefaultInputMap uses "Space" / "ArrowUp" /
// "Escape" / etc., which DO NOT match the raw pixelforge_key values
// (" ", "Up", "Esc"). Both the friendly default literal and the raw
// key value resolve to the same Label in labelForKeyValue, so loading
// a default-shipped project shows the right label and saving rewrites
// the binding to the compiler-matching value.
var keyboardOptions = []keyOption{
	// Directional / common gameplay keys at the top — these are the
	// 95% of picks for a NES-style remap.
	{Label: "Up Arrow", Value: string(pixelforge_key.Up)},
	{Label: "Down Arrow", Value: string(pixelforge_key.Down)},
	{Label: "Left Arrow", Value: string(pixelforge_key.Left)},
	{Label: "Right Arrow", Value: string(pixelforge_key.Right)},
	{Label: "Space", Value: string(pixelforge_key.Space)},
	{Label: "Enter", Value: string(pixelforge_key.Enter)},
	{Label: "Escape", Value: string(pixelforge_key.Esc)},
	{Label: "Tab", Value: string(pixelforge_key.Tab)},
	{Label: "Backspace", Value: string(pixelforge_key.Backspace)},
}

// alphaKeyboardOptions appends A..Z and 0..9 onto keyboardOptions at
// package init so the dropdown shows the full letter / digit picker
// below the curated common keys.
func init() {
	letters := []pixelforge_key.Key{
		pixelforge_key.A, pixelforge_key.B, pixelforge_key.C, pixelforge_key.D,
		pixelforge_key.E, pixelforge_key.F, pixelforge_key.G, pixelforge_key.H,
		pixelforge_key.I, pixelforge_key.J, pixelforge_key.K, pixelforge_key.L,
		pixelforge_key.M, pixelforge_key.N, pixelforge_key.O, pixelforge_key.P,
		pixelforge_key.Q, pixelforge_key.R, pixelforge_key.S, pixelforge_key.T,
		pixelforge_key.U, pixelforge_key.V, pixelforge_key.W, pixelforge_key.X,
		pixelforge_key.Y, pixelforge_key.Z,
	}
	for _, k := range letters {
		keyboardOptions = append(keyboardOptions, keyOption{Label: string(k), Value: string(k)})
	}
	digits := []pixelforge_key.Key{
		pixelforge_key.Digit0, pixelforge_key.Digit1, pixelforge_key.Digit2,
		pixelforge_key.Digit3, pixelforge_key.Digit4, pixelforge_key.Digit5,
		pixelforge_key.Digit6, pixelforge_key.Digit7, pixelforge_key.Digit8,
		pixelforge_key.Digit9,
	}
	for _, k := range digits {
		keyboardOptions = append(keyboardOptions, keyOption{Label: string(k), Value: string(k)})
	}
	// Function keys for power-user remaps.
	fns := []pixelforge_key.Key{
		pixelforge_key.F1, pixelforge_key.F2, pixelforge_key.F3, pixelforge_key.F4,
		pixelforge_key.F5, pixelforge_key.F6, pixelforge_key.F7, pixelforge_key.F8,
		pixelforge_key.F9, pixelforge_key.F10, pixelforge_key.F11, pixelforge_key.F12,
	}
	for _, k := range fns {
		keyboardOptions = append(keyboardOptions, keyOption{Label: string(k), Value: string(k)})
	}
}

// gamepadOptions mirrors keyboardOptions for the gamepad column. The
// Value column matches the pixelforge_pad.Button string the U3 compiler
// will see in EventButton.Button.
//
// DefaultInputMap stores "GamepadA" / "DPadUp" etc. — neither of those
// strings exist as a pixelforge_pad.Button. labelForButtonValue maps
// both the friendly default literals and the raw values onto the same
// Label so the workspace's first render shows the right text, and the
// first edit rewrites the binding to a compiler-matching value.
var gamepadOptions = []keyOption{
	{Label: "A (Bottom)", Value: string(pixelforge_pad.A)},
	{Label: "B (Right)", Value: string(pixelforge_pad.B)},
	{Label: "X (Left)", Value: string(pixelforge_pad.X)},
	{Label: "Y (Top)", Value: string(pixelforge_pad.Y)},
	{Label: "D-Pad Up", Value: string(pixelforge_pad.Top)},
	{Label: "D-Pad Down", Value: string(pixelforge_pad.Bottom)},
	{Label: "D-Pad Left", Value: string(pixelforge_pad.Left)},
	{Label: "D-Pad Right", Value: string(pixelforge_pad.Right)},
}

// modifierOptions enumerates the three supported modifier choices plus
// "(none)". Order matches what the U3 compiler accepts in
// pixelforge_project.InputBinding.Modifier.
var modifierOptions = []keyOption{
	{Label: "Ctrl", Value: "Ctrl"},
	{Label: "Shift", Value: "Shift"},
	{Label: "Alt", Value: "Alt"},
}

// labelForKeyValue resolves a stored Keyboard string back to the
// friendly label, bridging the DefaultInputMap mismatch by mapping
// known friendly defaults ("Space", "ArrowUp", "Escape") onto the same
// label as the raw pixelforge_key value (" ", "Up", "Esc").
func labelForKeyValue(value string) string {
	switch value {
	case "Space":
		return "Space"
	case "ArrowUp":
		return "Up Arrow"
	case "ArrowDown":
		return "Down Arrow"
	case "ArrowLeft":
		return "Left Arrow"
	case "ArrowRight":
		return "Right Arrow"
	case "Escape":
		return "Escape"
	}
	for _, opt := range keyboardOptions {
		if opt.Value == value {
			return opt.Label
		}
	}
	return value
}

// labelForButtonValue resolves a stored GamepadButton string to its
// friendly label. Mirrors labelForKeyValue's friendly-default bridge:
// "GamepadA" / "GamepadB" / "DPadUp" etc. resolve to the same labels
// as the raw pixelforge_pad values ("A", "B", "Top").
func labelForButtonValue(value string) string {
	switch value {
	case "GamepadA":
		return "A (Bottom)"
	case "GamepadB":
		return "B (Right)"
	case "GamepadX":
		return "X (Left)"
	case "GamepadY":
		return "Y (Top)"
	case "DPadUp":
		return "D-Pad Up"
	case "DPadDown":
		return "D-Pad Down"
	case "DPadLeft":
		return "D-Pad Left"
	case "DPadRight":
		return "D-Pad Right"
	case "GamepadStart":
		return "(start — unsupported in pad v1)"
	}
	for _, opt := range gamepadOptions {
		if opt.Value == value {
			return opt.Label
		}
	}
	return value
}

// keyboardCombo emits a Combo whose items are the friendly keyboard
// labels. Returns (picked value, changed). The current value is
// resolved through labelForKeyValue so default-shipped projects
// display correctly on first render even though the stored value
// doesn't appear in keyboardOptions.
func keyboardCombo(id, current string) (string, bool) {
	return canonicalCombo(id, current, keyboardOptions, labelForKeyValue)
}

// gamepadCombo emits a Combo over gamepadOptions.
func gamepadCombo(id, current string) (string, bool) {
	return canonicalCombo(id, current, gamepadOptions, labelForButtonValue)
}

// modifierCombo emits a Combo over the three supported modifier keys
// plus a "(none)" entry that maps to "".
func modifierCombo(id, current string) (string, bool) {
	return canonicalCombo(id, current, modifierOptions, func(v string) string {
		for _, opt := range modifierOptions {
			if opt.Value == v {
				return opt.Label
			}
		}
		return v
	})
}

// canonicalCombo is the shared dropdown renderer. It always prepends a
// "(none)" item so unsetting a binding is a single click; the returned
// "changed" boolean only fires when the user picks a value different
// from current.
func canonicalCombo(id, current string, options []keyOption, labelFn func(string) string) (string, bool) {
	const noneLabel = "(none)"
	items := []string{noneLabel}
	values := []string{""}
	currentLabel := ""
	if current != "" {
		currentLabel = labelFn(current)
	}
	// Track whether the current value is already represented in the
	// canonical options list — if not (e.g. a forward-compat unknown
	// key), prepend it so the combo can render its current state.
	seenCurrent := current == ""
	for _, opt := range options {
		items = append(items, opt.Label)
		values = append(values, opt.Value)
		if opt.Value == current {
			seenCurrent = true
		}
	}
	if !seenCurrent {
		items = append([]string{noneLabel, currentLabel}, items[1:]...)
		values = append([]string{"", current}, values[1:]...)
	}

	selected := int32(0)
	for i, v := range values {
		if v == current {
			selected = int32(i)
			break
		}
	}
	if imgui.ComboStrarrV(id, &selected, items, int32(len(items)), int32(len(items))) {
		picked := values[selected]
		if picked != current {
			return picked, true
		}
	}
	return current, false
}

// KeyboardOptions exposes the canonical keyboard option list for tests
// and (future) the inspector's intent-name picker. Returns a copy so
// callers may sort or filter without affecting package state.
func KeyboardOptions() []keyOption {
	out := make([]keyOption, len(keyboardOptions))
	copy(out, keyboardOptions)
	return out
}

// GamepadOptions exposes the canonical gamepad option list.
func GamepadOptions() []keyOption {
	out := make([]keyOption, len(gamepadOptions))
	copy(out, gamepadOptions)
	return out
}

// SortedIntentNames returns the workspace-rendered intent ordering.
// Used by tests asserting that the workspace surfaces all 9 intents.
func SortedIntentNames() []string {
	intents := pixelforge_input.AllIntents()
	sort.Strings(intents)
	return intents
}

// RegisterWith installs the input workspace on the editor. Mirrors
// palette.RegisterWith / capture.RegisterWith / scripting.RegisterWith
// so pixelforge_studio/main.go can compose the three (soon four) lines
// of workspace registration uniformly.
func RegisterWith(e *editor.Editor) *Workspace {
	if e == nil {
		return nil
	}
	w := NewWorkspace()
	e.RegisterWorkspace(w)
	return w
}
