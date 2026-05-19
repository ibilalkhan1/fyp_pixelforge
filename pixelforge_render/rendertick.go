package pixelforge_render

import (
	"errors"
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
)

// ErrNilRuntime is returned by RenderTickAtScreen / RenderTickAtRGBA
// when the caller passes a nil *capsuleruntime.Runtime. The runtime is
// the source of project + sinks for the draw path; rendering without
// one would be meaningless, so we fail fast instead of panicking.
var ErrNilRuntime = errors.New("pixelforge_render: runtime is nil")

// ErrNilScreen is returned by RenderTickAtScreen when the caller
// passes a nil destination surface. The RGBA entry point allocates
// its own off-screen, so this only applies to the on-screen path.
var ErrNilScreen = errors.New("pixelforge_render: destination screen is nil")

// GamepadState mirrors the subset of pixelforge_pad data that the
// recorder + replay harness need to reproduce a tick's input. Kept
// minimal at v1: a slice of held button indices and the left + right
// stick axes. Future expansion adds rumble / haptic state if a verb
// surface needs it.
type GamepadState struct {
	// Buttons holds the gamepad button indices currently held this
	// tick. Empty slice == no buttons. Indices match
	// ebiten.GamepadButton values for portability across pad models.
	Buttons []ebiten.GamepadButton

	// LeftX / LeftY / RightX / RightY are the analog stick axes in
	// the range [-1, 1]. Quantised to 1/127 by the recorder when
	// writing to .trace.jsonl so the on-wire representation is
	// stable, but in-memory they stay float64 so verbs can read
	// fractional thrust without losing precision.
	LeftX  float64
	LeftY  float64
	RightX float64
	RightY float64
}

// InputFrame is the per-tick input snapshot that drives one
// invocation of RenderTickAt. It captures the keyboard keys held
// during this tick + optional gamepad state. The replay harness
// (U5) reconstructs an InputFrame for every recorded tick; tests
// build one inline; the live player binary samples ebiten.AppendKeys
// + the gamepad backend each Update into one of these.
//
// Empty InputFrame{} (Keys: nil, Pad: nil) is valid and means
// "this tick saw no input". advanceAndRender handles it without
// panic.
//
// Keys is a slice of ebiten.Key values rather than a string set so
// the player binary doesn't pay a per-tick map[string]bool
// allocation cost. The replay harness converts to/from the
// ebiten.Key.String() representation when (de)serialising
// .trace.jsonl.
type InputFrame struct {
	Keys []ebiten.Key
	Pad  *GamepadState
}

// IsKeyDown reports whether the named key is held in this frame.
// Linear-scan over Keys is fine for v1 — the typical tick holds at
// most a handful of keys.
func (f InputFrame) IsKeyDown(k ebiten.Key) bool {
	for _, held := range f.Keys {
		if held == k {
			return true
		}
	}
	return false
}

// RenderTickAtScreen advances the runtime by one tick (applying
// `inputs` as the input state for that tick) and draws the resulting
// frame into `screen`. This is the load-bearing path used by both
// the shipped pixelforge-player binary and the studio preview;
// neither performs a ReadPixels readback per frame.
//
// Returns ErrNilRuntime or ErrNilScreen on a nil argument; otherwise
// nil. The function does not start goroutines; it does not call
// time.Now(); it does not read the filesystem. All state mutation
// happens against `rt` (currently via the package-level pixelforge
// globals — see the parity-contract note in doc.go) and against
// `screen`.
//
// Determinism: same rt + same tick + same inputs ⇒ same pixels
// written to screen, byte-for-byte. The determinism test in
// rendertick_test.go uses the RGBA variant to assert this directly;
// the screen variant gets the same guarantee transitively via the
// shared advanceAndRender core.
func RenderTickAtScreen(rt *capsuleruntime.Runtime, screen *ebiten.Image, tick uint64, inputs InputFrame) error {
	if rt == nil {
		return ErrNilRuntime
	}
	if screen == nil {
		return ErrNilScreen
	}
	advanceAndRender(rt, screen, tick, inputs)
	return nil
}

// RenderTickAtRGBA advances the runtime by one tick (applying
// `inputs`) and returns the resulting frame as a fresh *image.RGBA.
// Used by the CI replay harness and by any test that wants to hash
// or compare frame bytes.
//
// Mechanically: allocates an off-screen *ebiten.Image sized to the
// project's screen dimensions, runs advanceAndRender into it, then
// performs ebiten.Image.ReadPixels into a freshly-allocated RGBA
// pixel buffer. The readback is the dominant cost — do not call
// this on hot paths.
//
// Returns ErrNilRuntime on a nil runtime; otherwise a non-nil
// *image.RGBA + nil error. The returned RGBA's Pix slice is owned
// by the caller — modifying it does not affect the next call.
func RenderTickAtRGBA(rt *capsuleruntime.Runtime, tick uint64, inputs InputFrame) (*image.RGBA, error) {
	if rt == nil {
		return nil, ErrNilRuntime
	}
	w, h := runtimeScreenDims(rt)
	off := newOffscreen(w, h)
	advanceAndRender(rt, off, tick, inputs)

	pix := make([]byte, w*h*4)
	off.ReadPixels(pix)

	img := &image.RGBA{
		Pix:    pix,
		Stride: w * 4,
		Rect:   image.Rect(0, 0, w, h),
	}
	return img, nil
}

// advanceAndRender is the single shared per-tick simulation + draw
// implementation. Both public entry points funnel here; the only
// differences between them are the target surface and (for the RGBA
// path) the post-draw readback. This is the literal "one render
// path" the arcade-shipping plan promises.
//
// Pipeline:
//  1. Bind the runtime's tick counter to the pixelforge global Frame
//     so engine code that branches on Frame sees the same value
//     regardless of whether the caller is player, studio, or replay.
//  2. Apply InputFrame to the input layer — for v1 this is a no-op
//     placeholder; U5's replay harness and the player's Update will
//     each wire this through pixelforge_key / pixelforge_pad
//     synthetic events once the input intent layer exposes a
//     deterministic "set held state" API. The seam is in place so
//     the wiring can land without an API churn.
//  3. Publish the FrameStart / Update / LateUpdate / Draw /
//     LateDraw event sequence to the existing pixelforge_loop
//     target — verb-recipe subscribers (installed by capsuleruntime
//     Boot) listen here and mutate the runtime state.
//  4. Copy the resulting pixelforge.Canvas (which the engine's Draw
//     stack wrote into) onto the target *ebiten.Image via the
//     existing CopyCanvasToEbitenImage palette-blit.
//
// The pixelforge globals (Frame, Time, Screen, Palette,
// PaletteMapping) are the engine's source of truth today. Moving
// them onto the Runtime struct is a larger refactor (the entire
// pixelforge_* surface assumes globals) and is explicitly out of
// scope for U4 — the parity guarantee is "deterministic given the
// same starting state", which holds because (a) advanceAndRender is
// single-threaded, (b) every state-mutating call inside it is
// driven from the same arguments every time.
func advanceAndRender(rt *capsuleruntime.Runtime, screen *ebiten.Image, tick uint64, inputs InputFrame) {
	// Step 1: bind tick counter. The engine reads pixelforge.Frame
	// in many places; matching it to the caller's `tick` lets the
	// replay harness reproduce frame-counter-dependent logic.
	pixelforge.Frame = int(tick)
	if tps := pixelforge.TPS(); tps > 0 {
		pixelforge.Time = float64(tick) / float64(tps)
	}

	// Step 2: apply inputs. v1 wires the InputFrame into the
	// existing intent layer via a deterministic synthetic-event
	// path. The actual key.Publish / pad.Publish bridge lands in
	// U5 when the replay harness needs it end-to-end; today we
	// stash the frame on a package-level pointer the harness can
	// consult.
	applyInputFrame(inputs)

	// Step 2b (U17): update the runtime's camera offset for multi-
	// screen-tall scenes. For single-screen scenes this is a no-op
	// (rt.CameraOffsetY stays at zero); for DK-class climbs the
	// offset follows the hero's body so the climber stays mid-
	// screen and the camera clamps at top + bottom. Computed here
	// (before the draw step) so the engine's draw code reads a
	// stable, deterministic offset for this tick.
	updateCameraOffset(rt)

	// Step 3: drive the engine update + draw event sequence. The
	// capsule subscribers run here. Note: we deliberately bypass
	// the pause + skipNextDraw heuristics in pixelforge_ebiten's
	// EbitenGame.Update — those are wall-clock-driven and would
	// break determinism. The replay path always runs the full
	// sequence every tick.
	if !engineInitialised {
		if pixelforge.Init != nil {
			pixelforge.Init()
		}
		piloop.Target().Publish(piloop.EventInit)
		engineInitialised = true
	}

	piloop.Target().Publish(piloop.EventFrameStart)
	pixelforge.Update()
	piloop.Target().Publish(piloop.EventUpdate)
	piloop.Target().Publish(piloop.EventLateUpdate)
	pixelforge.Draw()
	piloop.Target().Publish(piloop.EventDraw)
	piloop.Target().Publish(piloop.EventLateDraw)

	// Step 4: blit the engine's paletted canvas onto the
	// destination Ebiten image. This is the same palette-aware
	// copy the legacy EbitenGame.Draw performs; routing both the
	// player and studio through this single call is the parity
	// guarantee.
	pixelforge_ebiten.CopyCanvasToEbitenImage(pixelforge.Screen(), screen)

	_ = rt // rt is the parity contract subject; mutation happens
	// indirectly via the capsule subscribers installed during Boot.
}

// engineInitialised tracks whether pixelforge.Init + EventInit have
// been published. The Ebitengame originally guarded this with a
// `started` field; advanceAndRender preserves the same once-only
// semantics so behaviour matches between the player binary and the
// replay harness on the first tick.
//
// Package-level scope is deliberate: the pixelforge globals
// (pixelforge.Init, pixelforge.Frame, etc.) are themselves
// package-level, and engineInitialised is the matching once-flag
// for those globals. Tests that need a fresh init reset this via
// ResetForTest.
var engineInitialised bool

// activeInputFrame is the most recently applied InputFrame.
// applyInputFrame writes it; future bridges to the synthetic-event
// path (U5) read it. Kept here so the seam is wired without the
// downstream consumers landing yet.
var activeInputFrame InputFrame

// applyInputFrame stages the InputFrame for the current tick. v1
// stashes it on a package-level pointer so the U5 replay bridge can
// consult it; v2 (when pixelforge_input exposes a deterministic
// "set held state" API) will publish synthetic key/pad events here.
func applyInputFrame(f InputFrame) {
	activeInputFrame = f
}

// ActiveInputFrame returns the most recently applied InputFrame.
// Exposed for the U5 replay-harness recorder and for tests that
// want to assert the seam observed the inputs it was given.
func ActiveInputFrame() InputFrame {
	return activeInputFrame
}

// ResetForTest restores the package-level once-flag + active input
// to their zero state. Tests that exercise the EventInit-published-
// exactly-once contract call this between subtests. Production code
// does not call it.
func ResetForTest() {
	engineInitialised = false
	activeInputFrame = InputFrame{}
}

// UpdateCameraOffsetForTest exposes the U17 multi-screen camera
// offset computation to test code without forcing a full Ebitengine
// graphics context. Production code in advanceAndRender calls
// updateCameraOffset directly each tick; tests that just want to
// assert "scene + hero Y produces expected camera offset" can call
// this helper.
func UpdateCameraOffsetForTest(rt *capsuleruntime.Runtime) {
	updateCameraOffset(rt)
}

// updateCameraOffset is the U17 multi-screen vertical scroll seam.
// For a single-screen scene (GridHeightScreens <= 1) the offset
// stays at zero — no scroll, existing behaviour preserved. For a
// taller scene the camera follows the player entity's body Y so the
// hero stays mid-screen vertically, with the offset clamped at top
// and bottom edges so the camera never reveals "off the world"
// pixels.
//
// Hero identification: the first entity in the current scene whose
// Archetype is "Player" wins. Scenes with no player entity get zero
// offset (and the camera shows the top of the level — defaulting
// the offset rather than crashing matches the runtime's broader
// "never break a shipped game" posture).
//
// Determinism: same rt state + same body positions ⇒ same offset.
// All math is integer (positions are converted from Fixed32 via the
// substrate's deterministic .Int() truncation).
func updateCameraOffset(rt *capsuleruntime.Runtime) {
	if rt == nil || rt.CurrentScene == nil {
		return
	}
	screensTall := rt.CurrentScene.GridHeightScreens
	if screensTall <= 1 {
		rt.CameraOffsetY = 0
		return
	}
	_, screenH := runtimeScreenDims(rt)
	totalH := screensTall * screenH
	// Locate the player body.
	var heroY int
	heroFound := false
	for i := range rt.CurrentScene.Entities {
		e := &rt.CurrentScene.Entities[i]
		if e.Archetype != "Player" {
			continue
		}
		body, ok := rt.Bodies[e.ID]
		if !ok || body == nil {
			continue
		}
		heroY = body.Position.Y.Int()
		heroFound = true
		break
	}
	if !heroFound {
		rt.CameraOffsetY = 0
		return
	}
	// Center the camera on the hero: offset = heroY - screenH/2.
	offset := heroY - screenH/2
	// Clamp to [0, totalH - screenH].
	maxOffset := totalH - screenH
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	rt.CameraOffsetY = offset
}
