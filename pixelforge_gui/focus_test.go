package pixelforge_gui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
)

func TestFocusManager_Empty(t *testing.T) {
	m := pgui.NewFocusManager()
	assert.Nil(t, m.Focused())
	m.Tab(true)
	assert.Nil(t, m.Focused(), "Tab on an empty manager must not panic")
}

func TestFocusManager_RegisterAndFocus(t *testing.T) {
	m := pgui.NewFocusManager()
	a := pgui.New()
	b := pgui.New()
	m.Register(a)
	m.Register(b)

	m.Focus(a)
	assert.Same(t, a, m.Focused())

	m.Focus(b)
	assert.Same(t, b, m.Focused())
}

func TestFocusManager_TabWrapsForward(t *testing.T) {
	m := pgui.NewFocusManager()
	a, b, c := pgui.New(), pgui.New(), pgui.New()
	m.Register(a)
	m.Register(b)
	m.Register(c)

	m.Focus(a)
	m.Tab(true)
	assert.Same(t, b, m.Focused())
	m.Tab(true)
	assert.Same(t, c, m.Focused())
	m.Tab(true)
	assert.Same(t, a, m.Focused(), "Tab past the last element wraps to the first")
}

func TestFocusManager_TabBackwardWraps(t *testing.T) {
	m := pgui.NewFocusManager()
	a, b := pgui.New(), pgui.New()
	m.Register(a)
	m.Register(b)

	m.Focus(a)
	m.Tab(false)
	assert.Same(t, b, m.Focused(), "Shift+Tab on the first element wraps to the last")
}

func TestFocusManager_TabFromNilStartsAtFirst(t *testing.T) {
	m := pgui.NewFocusManager()
	a, b := pgui.New(), pgui.New()
	m.Register(a)
	m.Register(b)
	m.Tab(true)
	assert.Same(t, a, m.Focused())
}

func TestFocusManager_Unregister_ClearsFocus(t *testing.T) {
	m := pgui.NewFocusManager()
	a := pgui.New()
	m.Register(a)
	m.Focus(a)
	m.Unregister(a)
	assert.Nil(t, m.Focused())
}

func TestFocusManager_DoubleRegister_NoOp(t *testing.T) {
	m := pgui.NewFocusManager()
	a := pgui.New()
	m.Register(a)
	m.Register(a)
	assert.Len(t, m.Order(), 1)
}

func TestFocusManager_FocusUnregisteredElement_NoOp(t *testing.T) {
	m := pgui.NewFocusManager()
	a := pgui.New()
	stranger := pgui.New()
	m.Register(a)
	m.Focus(a)
	m.Focus(stranger)
	assert.Same(t, a, m.Focused(), "focusing an unregistered element must not move focus")
}

func TestFocusManager_Blur(t *testing.T) {
	m := pgui.NewFocusManager()
	a := pgui.New()
	m.Register(a)
	m.Focus(a)
	m.Blur()
	assert.Nil(t, m.Focused())
}
