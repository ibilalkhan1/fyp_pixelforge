// Capture mode for the Settings → Input workspace (plan-009 U22, R20).
//
// Track B of U22 layers a "press-key-to-bind" affordance on top of the
// existing dropdown-based workspace. The CaptureMode state machine
// below owns the small but load-bearing state transitions:
//
//	Inactive ──BeginCapture(intent)──▶ Waiting{intent}
//	Waiting{intent} ──OnKey(non-modifier, non-Esc)──▶ Inactive   (binding recorded)
//	Waiting{intent} ──OnKey(Esc)──▶ Inactive                     (cancelled)
//	Waiting{intent} ──OnKey(modifier-only)──▶ Waiting{intent}    (silently ignored)
//	Waiting{intent} ──Cancel()──▶ Inactive
//	Waiting{intent} ──BeginCapture(intent)──▶ Inactive           (toggle off)
//	Waiting{intent} ──BeginCapture(other)──▶ Waiting{other}      (retarget)
//
// The state machine is engine-agnostic: it accepts ebiten.Key values
// (since that's what the workspace's input loop already drives) and
// emits the pixelforge_key-compatible string the U3 compiler will
// match against. This keeps the test seam hermetic — no need to drive
// the global pievent registry or stand up an Ebitengine window — and
// keeps the workspace itself focused on rendering, not state.
//
// Modifier-only rejection (Shift/Ctrl/Alt alone): the design-lens
// doc-review finding was that pressing Shift to set up a Shift+S combo
// should leave the workspace in capture mode rather than silently
// binding "Shift" as the intent's key. Capture stays Waiting until a
// non-modifier key arrives.
package input

import "github.com/hajimehoshi/ebiten/v2"

// CaptureState enumerates the three valid workspace-capture states.
type CaptureState int

const (
	// CaptureInactive: no capture in progress; dropdowns drive edits.
	CaptureInactive CaptureState = iota
	// CaptureWaiting: workspace is listening for the next non-modifier
	// keypress; the Intent field on CaptureMode names the binding the
	// keypress will replace.
	CaptureWaiting
)

// CaptureMode is the workspace's per-frame capture state. Lives on
// the Workspace struct (one capture-in-flight per workspace), driven
// by the workspace's Render loop + the OnKey integrations in
// workspace.go.
//
// Zero-value is the inactive state, so a freshly-constructed
// Workspace starts with no capture in flight.
type CaptureMode struct {
	state  CaptureState
	intent string
}

// State returns the current capture state. Test seam + render-time
// dispatch.
func (c *CaptureMode) State() CaptureState {
	if c == nil {
		return CaptureInactive
	}
	return c.state
}

// Intent returns the name of the intent the capture is in flight for,
// or "" when inactive.
func (c *CaptureMode) Intent() string {
	if c == nil {
		return ""
	}
	return c.intent
}

// IsActive reports whether the workspace is currently waiting for a
// keypress to land. Render uses this to swap the Capture button label
// to "Press a key… (Esc to cancel)".
func (c *CaptureMode) IsActive() bool {
	return c != nil && c.state == CaptureWaiting
}

// BeginCapture transitions to the Waiting state for the named intent.
// Two special cases the doc-review carved out:
//
//   - Same-intent toggle: calling BeginCapture(intent) while already
//     Waiting on intent transitions back to Inactive. This implements
//     the "click Capture again to cancel" affordance from the design-
//     lens finding.
//   - Different-intent retarget: calling BeginCapture(other) while
//     Waiting on intent transitions to Waiting{other}. Designers who
//     click Capture on one row, then Capture on a different row,
//     should end up listening for the second row's keypress (no need
//     to Esc-cancel first).
func (c *CaptureMode) BeginCapture(intent string) {
	if c == nil {
		return
	}
	if c.state == CaptureWaiting && c.intent == intent {
		// Same-intent toggle: cancel.
		c.state = CaptureInactive
		c.intent = ""
		return
	}
	c.state = CaptureWaiting
	c.intent = intent
}

// Cancel forces the state machine back to Inactive regardless of the
// current state. Called when the user clicks outside the workspace,
// switches workspaces, or the workspace decides capture should not
// outlive a frame (e.g., the project changed underneath it).
func (c *CaptureMode) Cancel() {
	if c == nil {
		return
	}
	c.state = CaptureInactive
	c.intent = ""
}

// OnKey processes a single keypress. Returns:
//   - bound == true: a binding was captured; newBinding carries the
//     pixelforge_key-compatible string the workspace should store on
//     the intent's InputBinding.Keyboard slot. The state machine
//     transitions to Inactive.
//   - bound == false, newBinding == "": the key was rejected (Esc
//     cancel, modifier-only, or capture wasn't active). Capture stays
//     Waiting for modifier-only; transitions to Inactive for Esc.
//
// The intent the keypress applied to is also returned via Intent()
// before the transition completes — callers that need it should read
// Intent() before the OnKey call clears it.
func (c *CaptureMode) OnKey(key ebiten.Key) (bound bool, newBinding string) {
	if c == nil || c.state != CaptureWaiting {
		return false, ""
	}
	// Esc: cancel.
	if key == ebiten.KeyEscape {
		c.state = CaptureInactive
		c.intent = ""
		return false, ""
	}
	// Modifier-only: ignore silently. Capture stays Waiting.
	if isModifierOnlyKey(key) {
		return false, ""
	}
	// Regular key: capture, transition to Inactive.
	value := captureKeyToBindingValue(key)
	if value == "" {
		// Unmapped key — treat as a no-op rather than recording an
		// empty binding (which would silently clear the intent).
		// Capture stays Waiting so the designer can try again.
		return false, ""
	}
	c.state = CaptureInactive
	c.intent = ""
	return true, value
}

// isModifierOnlyKey reports whether key is one of Shift/Ctrl/Alt (left
// or right). The doc-review finding asked for these to be rejected so
// designers can hold Shift before pressing the modified key without
// accidentally binding the modifier itself.
func isModifierOnlyKey(key ebiten.Key) bool {
	switch key {
	case ebiten.KeyShiftLeft, ebiten.KeyShiftRight,
		ebiten.KeyControlLeft, ebiten.KeyControlRight,
		ebiten.KeyAltLeft, ebiten.KeyAltRight,
		ebiten.KeyMetaLeft, ebiten.KeyMetaRight:
		return true
	}
	return false
}

// captureKeyToBindingValue maps an ebiten.Key to the pixelforge_key
// string the U3 input compiler matches against. Mirrors the keyMap in
// pixelforge_ebiten/internal/input/keyboard.go but covers only the
// keys the designer is likely to bind (alphanumerics, arrows, space,
// enter, tab, function keys). Unmapped keys return "".
func captureKeyToBindingValue(key ebiten.Key) string {
	switch key {
	// Letters A..Z.
	case ebiten.KeyA:
		return "A"
	case ebiten.KeyB:
		return "B"
	case ebiten.KeyC:
		return "C"
	case ebiten.KeyD:
		return "D"
	case ebiten.KeyE:
		return "E"
	case ebiten.KeyF:
		return "F"
	case ebiten.KeyG:
		return "G"
	case ebiten.KeyH:
		return "H"
	case ebiten.KeyI:
		return "I"
	case ebiten.KeyJ:
		return "J"
	case ebiten.KeyK:
		return "K"
	case ebiten.KeyL:
		return "L"
	case ebiten.KeyM:
		return "M"
	case ebiten.KeyN:
		return "N"
	case ebiten.KeyO:
		return "O"
	case ebiten.KeyP:
		return "P"
	case ebiten.KeyQ:
		return "Q"
	case ebiten.KeyR:
		return "R"
	case ebiten.KeyS:
		return "S"
	case ebiten.KeyT:
		return "T"
	case ebiten.KeyU:
		return "U"
	case ebiten.KeyV:
		return "V"
	case ebiten.KeyW:
		return "W"
	case ebiten.KeyX:
		return "X"
	case ebiten.KeyY:
		return "Y"
	case ebiten.KeyZ:
		return "Z"

	// Digits 0..9.
	case ebiten.KeyDigit0:
		return "0"
	case ebiten.KeyDigit1:
		return "1"
	case ebiten.KeyDigit2:
		return "2"
	case ebiten.KeyDigit3:
		return "3"
	case ebiten.KeyDigit4:
		return "4"
	case ebiten.KeyDigit5:
		return "5"
	case ebiten.KeyDigit6:
		return "6"
	case ebiten.KeyDigit7:
		return "7"
	case ebiten.KeyDigit8:
		return "8"
	case ebiten.KeyDigit9:
		return "9"

	// Arrows, common gameplay keys.
	case ebiten.KeyArrowUp:
		return "Up"
	case ebiten.KeyArrowDown:
		return "Down"
	case ebiten.KeyArrowLeft:
		return "Left"
	case ebiten.KeyArrowRight:
		return "Right"
	case ebiten.KeySpace:
		return " "
	case ebiten.KeyEnter:
		return "Enter"
	case ebiten.KeyTab:
		return "Tab"
	case ebiten.KeyBackspace:
		return "Backspace"

	// Function keys.
	case ebiten.KeyF1:
		return "F1"
	case ebiten.KeyF2:
		return "F2"
	case ebiten.KeyF3:
		return "F3"
	case ebiten.KeyF4:
		return "F4"
	case ebiten.KeyF5:
		return "F5"
	case ebiten.KeyF6:
		return "F6"
	case ebiten.KeyF7:
		return "F7"
	case ebiten.KeyF8:
		return "F8"
	case ebiten.KeyF9:
		return "F9"
	case ebiten.KeyF10:
		return "F10"
	case ebiten.KeyF11:
		return "F11"
	case ebiten.KeyF12:
		return "F12"
	}
	return ""
}
