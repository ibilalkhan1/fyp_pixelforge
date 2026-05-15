package editor

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
)

// DefaultKeyMap registers the four file shortcuts and nothing else.
func TestDefaultKeyMap_RegistersFileShortcuts(t *testing.T) {
	k := DefaultKeyMap()
	assert.ElementsMatch(t,
		[]string{"file.new", "file.open", "file.save", "file.close"},
		k.Actions())

	saveBindings := k.BindingsFor("file.save")
	assert.Equal(t, []Binding{{Mods: ModCtrl, Key: ebiten.KeyS}}, saveBindings)
}

// Re-registering the same (action, binding) pair is a no-op.
func TestKeyMap_RegisterDeduplicates(t *testing.T) {
	k := NewKeyMap()
	b := Binding{Mods: ModCtrl, Key: ebiten.KeyS}
	k.Register("file.save", b)
	k.Register("file.save", b)
	assert.Len(t, k.BindingsFor("file.save"), 1)
}

// Multiple distinct bindings for one action are kept.
func TestKeyMap_MultipleBindings(t *testing.T) {
	k := NewKeyMap()
	k.Register("file.save", Binding{Mods: ModCtrl, Key: ebiten.KeyS})
	k.Register("file.save", Binding{Mods: ModCtrl, Key: ebiten.KeyEnter})
	assert.Len(t, k.BindingsFor("file.save"), 2)
}

// IsPressed for an unregistered action returns false (does not panic).
func TestKeyMap_IsPressedUnregisteredAction(t *testing.T) {
	k := NewKeyMap()
	assert.False(t, k.IsPressed("does.not.exist"))
}

// bindingPressed honours the modifier mask via the injected key probe.
func TestBindingPressed_HonoursModifiers(t *testing.T) {
	// stub: only Ctrl+S is held
	held := map[ebiten.Key]bool{
		ebiten.KeyControlLeft: true,
		ebiten.KeyS:           true,
	}
	probe := func(k ebiten.Key) bool { return held[k] }

	ctrlS := Binding{Mods: ModCtrl, Key: ebiten.KeyS}
	plainS := Binding{Mods: ModNone, Key: ebiten.KeyS}
	ctrlShiftS := Binding{Mods: ModCtrl | ModShift, Key: ebiten.KeyS}

	assert.True(t, bindingPressed(ctrlS, probe))
	// Modless binding still fires when Ctrl is *also* down — common UX
	// since we don't claim modifier-exclusivity.
	assert.True(t, bindingPressed(plainS, probe))
	// Missing Shift → not pressed
	assert.False(t, bindingPressed(ctrlShiftS, probe))
}

// Unregistering removes an action; IsPressed reflects the absence.
func TestKeyMap_Unregister(t *testing.T) {
	k := DefaultKeyMap()
	assert.True(t, k.Unregister("file.save"))
	assert.Nil(t, k.BindingsFor("file.save"))
	// Second unregister returns false
	assert.False(t, k.Unregister("file.save"))
}

// describeBinding renders modifiers in the canonical order.
func TestDescribeBinding(t *testing.T) {
	assert.Equal(t, "Ctrl+S", describeBinding(Binding{Mods: ModCtrl, Key: ebiten.KeyS}))
	assert.Equal(t, "Ctrl+Shift+S", describeBinding(Binding{Mods: ModCtrl | ModShift, Key: ebiten.KeyS}))
	assert.Equal(t, "Alt+F4", describeBinding(Binding{Mods: ModAlt, Key: ebiten.KeyF4}))
}
