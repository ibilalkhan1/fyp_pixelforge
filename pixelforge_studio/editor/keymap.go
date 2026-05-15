package editor

import (
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Modifier is a bitmask of held modifier keys required for a shortcut.
// Editor shortcuts are platform-portable; we never bake macOS-specific
// Cmd handling into the action layer.
type Modifier uint8

const (
	ModNone  Modifier = 0
	ModCtrl  Modifier = 1 << 0
	ModShift Modifier = 1 << 1
	ModAlt   Modifier = 1 << 2
)

// Binding is a (modifier-set, key) pair representing one keyboard
// shortcut. Multiple bindings can fire the same action; we treat the
// list as a set and dedupe on Register.
type Binding struct {
	Mods Modifier
	Key  ebiten.Key
}

// KeyMap maps human-readable action names ("file.save") to one or more
// keyboard bindings. Bindings are queried via IsPressed (held this
// frame) and JustPressed (transitioned this frame).
type KeyMap struct {
	bindings map[string][]Binding
}

// NewKeyMap returns an empty KeyMap. Most callers want DefaultKeyMap,
// which pre-registers the studio-wide shortcuts.
func NewKeyMap() *KeyMap {
	return &KeyMap{bindings: map[string][]Binding{}}
}

// DefaultKeyMap pre-registers the file-menu shortcuts (M0) and leaves
// tool shortcuts for M2 to layer on.
func DefaultKeyMap() *KeyMap {
	k := NewKeyMap()
	k.Register("file.new", Binding{Mods: ModCtrl, Key: ebiten.KeyN})
	k.Register("file.open", Binding{Mods: ModCtrl, Key: ebiten.KeyO})
	k.Register("file.save", Binding{Mods: ModCtrl, Key: ebiten.KeyS})
	k.Register("file.close", Binding{Mods: ModCtrl, Key: ebiten.KeyW})
	return k
}

// Register adds a binding to an action. Re-registering the same
// (action, binding) pair is a no-op; that lets callers safely set
// defaults in init paths without worrying about duplicates.
func (k *KeyMap) Register(action string, b Binding) {
	existing := k.bindings[action]
	for _, e := range existing {
		if e == b {
			return
		}
	}
	k.bindings[action] = append(existing, b)
}

// Unregister removes all bindings for an action. Returns true if any
// bindings existed.
func (k *KeyMap) Unregister(action string) bool {
	if _, ok := k.bindings[action]; !ok {
		return false
	}
	delete(k.bindings, action)
	return true
}

// Bindings returns the (sorted) action names registered. Used by the
// settings UI and tests; the slice is freshly allocated and safe to
// retain or mutate.
func (k *KeyMap) Actions() []string {
	out := make([]string, 0, len(k.bindings))
	for a := range k.bindings {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

// BindingsFor returns the bindings for action. Returns nil if the
// action is unregistered.
func (k *KeyMap) BindingsFor(action string) []Binding {
	src := k.bindings[action]
	if len(src) == 0 {
		return nil
	}
	out := make([]Binding, len(src))
	copy(out, src)
	return out
}

// IsPressed reports whether any binding for action is held this frame.
// Unregistered actions always return false (never panics).
func (k *KeyMap) IsPressed(action string) bool {
	bindings := k.bindings[action]
	for _, b := range bindings {
		if bindingPressed(b, ebiten.IsKeyPressed) {
			return true
		}
	}
	return false
}

// JustPressed reports whether any binding for action transitioned from
// released to pressed this frame. Most editor shortcuts care about the
// edge, not the held state, so this is the more common query.
func (k *KeyMap) JustPressed(action string) bool {
	bindings := k.bindings[action]
	for _, b := range bindings {
		if bindingJustPressed(b) {
			return true
		}
	}
	return false
}

// bindingPressed is a small helper so tests can stub the key query.
func bindingPressed(b Binding, isPressed func(ebiten.Key) bool) bool {
	if !modsPressed(b.Mods, isPressed) {
		return false
	}
	return isPressed(b.Key)
}

// bindingJustPressed requires the modifier-set to be held this frame
// AND the key to have just transitioned. This is how Ctrl+S is normally
// expressed — the user holds Ctrl, then taps S.
func bindingJustPressed(b Binding) bool {
	if !modsPressed(b.Mods, ebiten.IsKeyPressed) {
		return false
	}
	return inpututil.IsKeyJustPressed(b.Key)
}

func modsPressed(m Modifier, isPressed func(ebiten.Key) bool) bool {
	if m&ModCtrl != 0 && !(isPressed(ebiten.KeyControlLeft) || isPressed(ebiten.KeyControlRight)) {
		return false
	}
	if m&ModShift != 0 && !(isPressed(ebiten.KeyShiftLeft) || isPressed(ebiten.KeyShiftRight)) {
		return false
	}
	if m&ModAlt != 0 && !(isPressed(ebiten.KeyAltLeft) || isPressed(ebiten.KeyAltRight)) {
		return false
	}
	return true
}

// describeBinding renders a binding as a human-readable string
// ("Ctrl+S"). Exposed for the settings UI in later milestones.
func describeBinding(b Binding) string {
	parts := []string{}
	if b.Mods&ModCtrl != 0 {
		parts = append(parts, "Ctrl")
	}
	if b.Mods&ModShift != 0 {
		parts = append(parts, "Shift")
	}
	if b.Mods&ModAlt != 0 {
		parts = append(parts, "Alt")
	}
	parts = append(parts, b.Key.String())
	return strings.Join(parts, "+")
}
