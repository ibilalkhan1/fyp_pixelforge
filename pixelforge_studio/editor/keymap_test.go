package editor

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
)

// DefaultKeyMap registers the studio-wide shortcuts.
func TestDefaultKeyMap_RegistersFileShortcuts(t *testing.T) {
	k := DefaultKeyMap()
	for _, action := range []string{
		"file.new", "file.open", "file.save", "file.save_as",
		"file.close", "file.export", "file.quit",
		"tool.select", "tool.place", "tool.delete", "tool.paint",
		"workspace.cycle", "workspace.scene", "workspace.palette",
	} {
		assert.NotNil(t, k.BindingsFor(action), "expected %s registered", action)
	}

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

// bindingPressed enforces an exact modifier-mask match so overlapping
// shortcuts (Ctrl+S vs. Ctrl+Shift+S) never fire simultaneously.
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
	// Plain S does NOT fire while Ctrl is held — exact-mods match.
	assert.False(t, bindingPressed(plainS, probe))
	// Missing Shift → not pressed
	assert.False(t, bindingPressed(ctrlShiftS, probe))

	// When both Ctrl and Shift are held, only Ctrl+Shift+S matches.
	held2 := map[ebiten.Key]bool{
		ebiten.KeyControlLeft: true,
		ebiten.KeyShiftLeft:   true,
		ebiten.KeyS:           true,
	}
	probe2 := func(k ebiten.Key) bool { return held2[k] }
	assert.False(t, bindingPressed(ctrlS, probe2))
	assert.True(t, bindingPressed(ctrlShiftS, probe2))
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
