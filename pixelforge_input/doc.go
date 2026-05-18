// Package pixelforge_input is the v1 input intent layer that sits
// between the physical input packages (pixelforge_key, pixelforge_pad)
// and the verb-sheet runtime (pixelforge_studio/scripting/runtime).
//
// Designers and verbs reference *intents* — semantic names like
// "input/jump" or "input/attack" — rather than physical keys. The
// concrete mapping from intent to keyboard key, gamepad button, and
// optional modifier is data, carried by Project.InputMap, and may be
// edited live from the Settings -> Input workspace (U7) without
// restarting the engine.
//
// # Design
//
// At init time, the package creates one pievent.Target[IntentEvent]
// per canonical intent (9 total: jump, attack, up, down, left, right,
// use, menu, start) and registers each one with the pievent registry
// under its intent name (e.g. "input/jump"). Verbs subscribe to the
// intent targets via the existing subscribeAny reflection in
// pixelforge_studio/scripting/runtime/compile.go — exactly the same
// pathway used for any other named pievent.Target.
//
// Wiring the physical keyboard/gamepad events to the intent targets is
// the job of the Compiler. Compile(im) walks the InputMap, subscribes
// once to "key.main" and "pad.button" via pievent.LookupTarget +
// reflection, and on each incoming Event/EventButton looks up the
// matching binding(s) and publishes an IntentEvent on each affected
// intent target. The subscription happens at compile time — there is
// no per-tick reflection cost.
//
// Recompile(im) supports the live-edit flow: it unsubscribes the prior
// bridge handlers and re-subscribes with the new map, so changes made
// in the Settings -> Input workspace take effect on the next event
// without an engine restart.
//
// # Modifier handling
//
// The InputBinding schema carries an optional Modifier ("Ctrl",
// "Shift", or "Alt"). The compiler tracks modifier keys via the
// keyboard event stream itself: ShiftLeft/ShiftRight/CtrlLeft/CtrlRight/
// AltLeft/AltRight Down events flip an internal "held" flag that's
// consulted whenever a non-modifier key event would otherwise fire an
// intent. We deviate from pixelforge_key.Duration polling because the
// key package's Duration depends on a free-running per-frame counter
// (pixelforge.Frame) that is not advanced by the synchronous Publish
// path tests use; an event-driven held-set is both simpler and easier
// to unit-test.
//
// # What this package does NOT do
//
// It is the intent layer only. It does not own the InputMap (that
// belongs to pixelforge_project) and it does not render the editing
// workspace (U7). It also has no dependency on the studio packages —
// runtime-only, importable from a shipped game binary.
package pixelforge_input
