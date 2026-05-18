package pixelforge_project

// InputBinding maps a named intent (e.g. "input/jump") to one or more
// physical input sources: keyboard keys, a gamepad button, and an
// optional modifier key. The runtime intent layer (pixelforge_input)
// subscribes to key/pad events, looks up the matching intent in the
// active Project.InputMap, and republishes on the intent's pievent
// target. Verbs subscribe to the intent target, not to physical keys.
//
// Multiple bindings per intent are supported via the Keyboard slice and
// the optional GamepadButton: any matching source fires the intent. The
// optional Modifier (Ctrl/Shift/Alt) gates the binding so that e.g.
// Ctrl+S can be distinguished from a bare S press.
//
// The schema is additive omitempty on Project (v1 introduces InputMap)
// so pre-v1 .pforge files load cleanly with DefaultInputMap() applied
// at load time via Project.applyDefaults.
type InputBinding struct {
	// Intent is the canonical intent name (e.g. "input/jump"). One row
	// per intent; the InputMap slice acts as an ordered intent list.
	Intent string `json:"intent"`

	// Keyboard lists physical key names (matching pixelforge_key.Key
	// values, e.g. "Z", "Space", "ArrowLeft"). Multiple keys may bind
	// the same intent; any one firing republishes on the intent target.
	Keyboard []string `json:"keyboard,omitempty"`

	// GamepadButton, when non-empty, names a gamepad button (e.g.
	// "GamepadA"). Empty means no gamepad source for this intent.
	GamepadButton string `json:"gamepad_button,omitempty"`

	// Modifier, when non-empty, gates the binding on a held modifier
	// key. Valid values: "Ctrl", "Shift", "Alt". Empty means no
	// modifier required.
	Modifier string `json:"modifier,omitempty"`
}

// Canonical intent names. These mirror the IntentName constants in
// pixelforge_input but are duplicated here so pixelforge_project does
// not depend on the runtime intent package (avoids an import cycle
// between schema and runtime).
const (
	IntentJump   = "input/jump"
	IntentAttack = "input/attack"
	IntentUp     = "input/up"
	IntentDown   = "input/down"
	IntentLeft   = "input/left"
	IntentRight  = "input/right"
	IntentUse    = "input/use"
	IntentMenu   = "input/menu"
	IntentStart  = "input/start"
)

// DefaultInputMap returns the 9 NES-style intent bindings shipped as
// defaults for newly-created projects and applied at load time to any
// pre-v1 .pforge file missing the input_map key.
//
// The bindings mirror the NES era + modern keyboard conventions:
//   - Jump:    Z / Space / GamepadA
//   - Attack:  X / GamepadB
//   - Up/Down/Left/Right: arrow keys + WASD + DPad
//   - Use:     Enter / GamepadA
//   - Menu:    Esc / GamepadStart
//   - Start:   Enter / GamepadStart
//
// Returns a fresh slice on each call so callers may mutate it without
// affecting subsequent defaults.
func DefaultInputMap() []InputBinding {
	return []InputBinding{
		{Intent: IntentJump, Keyboard: []string{"Z", "Space"}, GamepadButton: "GamepadA"},
		{Intent: IntentAttack, Keyboard: []string{"X"}, GamepadButton: "GamepadB"},
		{Intent: IntentUp, Keyboard: []string{"ArrowUp", "W"}, GamepadButton: "DPadUp"},
		{Intent: IntentDown, Keyboard: []string{"ArrowDown", "S"}, GamepadButton: "DPadDown"},
		{Intent: IntentLeft, Keyboard: []string{"ArrowLeft", "A"}, GamepadButton: "DPadLeft"},
		{Intent: IntentRight, Keyboard: []string{"ArrowRight", "D"}, GamepadButton: "DPadRight"},
		{Intent: IntentUse, Keyboard: []string{"Enter"}, GamepadButton: "GamepadA"},
		{Intent: IntentMenu, Keyboard: []string{"Escape"}, GamepadButton: "GamepadStart"},
		{Intent: IntentStart, Keyboard: []string{"Enter"}, GamepadButton: "GamepadStart"},
	}
}
