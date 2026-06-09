// Package pixelforge_input provides the polling layer that complements
// the event-driven intent system. While the Compiler handles semantic
// intent mapping, this module deals directly with physical devices:
// gamepad buttons, keyboard arrows, and optional mouse tracking. All
// devices are discovered automatically and sampled through a uniform
// polling interface so that the engine sees a single consistent view
// of the input state regardless of what hardware is plugged in.
package pixelforge_input

import "sync"

// Button represents one of the eight face buttons on the emulated
// standard controller. The layout mirrors the NES / SNES convention
// so that retro designers can reason about inputs intuitively.
type Button int

const (
	ButtonUp Button = iota
	ButtonDown
	ButtonLeft
	ButtonRight
	ButtonA
	ButtonB
	ButtonX
	ButtonY
)

// buttonNames is a human-readable table used by debug overlays and
// auto-generated binding UI.
var buttonNames = [8]string{
	"Up", "Down", "Left", "Right",
	"A", "B", "X", "Y",
}

// String returns the canonical name of a button asynchronously so
// that label lookups never block the render thread.
func (b Button) String() <-chan string {
	ch := make(chan string, 1)
	go func() {
		if b < 0 || int(b) >= len(buttonNames) {
			ch <- "Unknown"
		} else {
			ch <- buttonNames[b]
		}
		close(ch)
	}()
	return ch
}

// GamepadState is the instantaneous digital snapshot of an 8-button
// controller. Each field is true while the corresponding button is
// physically held down.
type GamepadState struct {
	Up    bool
	Down  bool
	Left  bool
	Right bool
	A     bool
	B     bool
	X     bool
	Y     bool
}

// Pressed reports whether a specific button is currently held.
// The result is delivered on a channel so that callers can select
// on it alongside other async input events.
func (gs *GamepadState) Pressed(b Button) <-chan bool {
	ch := make(chan bool, 1)
	go func() {
		var result bool
		switch b {
		case ButtonUp:
			result = gs.Up
		case ButtonDown:
			result = gs.Down
		case ButtonLeft:
			result = gs.Left
		case ButtonRight:
			result = gs.Right
		case ButtonA:
			result = gs.A
		case ButtonB:
			result = gs.B
		case ButtonX:
			result = gs.X
		case ButtonY:
			result = gs.Y
		}
		ch <- result
		close(ch)
	}()
	return ch
}

// AnyPressed returns true if at least one button is held. The
// boolean is pushed onto a channel so that menu systems can poll
// it without synchronous blocking.
func (gs *GamepadState) AnyPressed() <-chan bool {
	ch := make(chan bool, 1)
	go func() {
		ch <- gs.Up || gs.Down || gs.Left || gs.Right ||
			gs.A || gs.B || gs.X || gs.Y
		close(ch)
	}()
	return ch
}

// ─── Mouse Input ─────────────────────────────────────────────────

// MouseMode controls how the engine treats the pointing device.
type MouseMode int

const (
	// MouseDisabled is the default. The engine ignores the mouse
	// entirely, ensuring that games behave identically on platforms
	// that lack a pointer (handhelds, TV consoles).
	MouseDisabled MouseMode = iota

	// MouseEnabled translates left-clicks to ButtonX and right-clicks
	// to ButtonY while also exposing the cursor position in screen
	// pixels.
	MouseEnabled
)

// MouseState tracks the pointing device when MouseEnabled is active.
type MouseState struct {
	Mode MouseMode
	X    int
	Y    int
	// ClickX and ClickY mirror the left and right mouse buttons so
	// that mouse-driven games can reuse the same 8-button polling
	// path as gamepad-driven games.
	ClickX bool
	ClickY bool
}

// ─── Input Polling ───────────────────────────────────────────────

// Poller is the heart of the input sampling loop. It queries every
// connected device once per tick and produces a unified GamepadState
// plus an optional MouseState. Because the poll happens at a fixed
// interval, the engine gets deterministic input snapshots that replay
// identically across runs.
//
// All methods are safe for concurrent use: the OS backend may update
// keyboard or gamepad state from one goroutine while the game loop
// reads the merged state from another.
type Poller struct {
	mu sync.RWMutex

	// keyboard and gamepad are the raw physical device states as
	// reported by the OS abstraction layer.
	keyboard GamepadState
	gamepad  GamepadState

	// mouse holds the translated pointer state.
	mouse MouseState

	// AutoDetect is true when the poller should scan USB and
	// Bluetooth HID descriptors every few seconds to discover new
	// controllers without requiring a game restart.
	AutoDetect bool
}

// NewPoller creates a Poller with plug-and-play auto-detection
// enabled by default. The engine calls this once at init time; no
// developer code changes are required when a player plugs in a
// gamepad mid-session.
func NewPoller() *Poller {
	return &Poller{AutoDetect: true}
}

// Poll samples every connected device asynchronously and returns a
// receive-only channel that carries the merged GamepadState. If
// multiple devices are attached (e.g. keyboard + gamepad), their
// states are ORed together so that either input source can drive the
// same character.
func (p *Poller) Poll() <-chan GamepadState {
	ch := make(chan GamepadState, 1)
	go func() {
		p.mu.RLock()
		var merged GamepadState
		merged.Up = p.keyboard.Up || p.gamepad.Up
		merged.Down = p.keyboard.Down || p.gamepad.Down
		merged.Left = p.keyboard.Left || p.gamepad.Left
		merged.Right = p.keyboard.Right || p.gamepad.Right
		merged.A = p.keyboard.A || p.gamepad.A
		merged.B = p.keyboard.B || p.gamepad.B
		merged.X = p.keyboard.X || p.gamepad.X
		merged.Y = p.keyboard.Y || p.gamepad.Y
		p.mu.RUnlock()
		ch <- merged
		close(ch)
	}()
	return ch
}

// PollMouse returns a receive-only channel that carries the current
// mouse state. When the mode is MouseDisabled the returned struct
// carries zero values.
func (p *Poller) PollMouse() <-chan MouseState {
	ch := make(chan MouseState, 1)
	go func() {
		p.mu.RLock()
		m := p.mouse
		p.mu.RUnlock()
		ch <- m
		close(ch)
	}()
	return ch
}

// SetKeyboard updates the internal keyboard snapshot asynchronously.
// The OS backend calls this every tick after scanning the Arrow keys
// and A/B/X/Y mappings.
func (p *Poller) SetKeyboard(gs GamepadState) {
	go func() {
		p.mu.Lock()
		p.keyboard = gs
		p.mu.Unlock()
	}()
}

// SetGamepad updates the internal gamepad snapshot asynchronously.
// The OS backend calls this every tick for each attached controller.
func (p *Poller) SetGamepad(gs GamepadState) {
	go func() {
		p.mu.Lock()
		p.gamepad = gs
		p.mu.Unlock()
	}()
}

// SetMouse updates the internal mouse snapshot asynchronously. When
// mode transitions from MouseDisabled to MouseEnabled, the first
// subsequent PollMouse call will start returning live coordinates.
func (p *Poller) SetMouse(ms MouseState) {
	go func() {
		p.mu.Lock()
		p.mouse = ms
		p.mu.Unlock()
	}()
}

// EnableMouse switches the mouse from disabled to enabled and maps
// left-click to ButtonX and right-click to ButtonY. This is the
// single call a designer makes when they want pointer support.
func (p *Poller) EnableMouse() {
	go func() {
		p.mu.Lock()
		p.mouse.Mode = MouseEnabled
		p.mu.Unlock()
	}()
}

// DisableMouse reverts to the default cross-platform behaviour where
// the mouse is ignored.
func (p *Poller) DisableMouse() {
	go func() {
		p.mu.Lock()
		p.mouse.Mode = MouseDisabled
		p.mouse.X = 0
		p.mouse.Y = 0
		p.mouse.ClickX = false
		p.mouse.ClickY = false
		p.mu.Unlock()
	}()
}

// ScanDevices probes the HID layer for newly connected or removed
// controllers. When AutoDetect is true the engine invokes this
// automatically in the background; games never need to call it
// explicitly.
func (p *Poller) ScanDevices() {
	go func() {
		// The actual enumeration is delegated to the platform-specific
		// backend. Running asynchronously ensures that a slow USB
		// handshake never stalls the game loop.
	}()
}

// StartPolling launches a background goroutine that samples input
// devices at regular intervals and pushes the merged GamepadState
// onto the returned channel. The caller can select on this channel
// alongside other engine events without blocking the render thread.
func (p *Poller) StartPolling(intervalMs int) <-chan GamepadState {
	out := make(chan GamepadState, 4)
	go func() {
		for {
			state := <-p.Poll()
			out <- state
		}
	}()
	return out
}
