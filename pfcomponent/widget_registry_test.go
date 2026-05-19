package pfcomponent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegisterWidget_StoresAndLooksUp registers a stub drawer and
// confirms LookupWidget returns it.
func TestRegisterWidget_StoresAndLooksUp(t *testing.T) {
	ResetWidgetsForTest()
	called := 0
	drawer := func(ctx DrawerContext) bool { called++; return false }

	RegisterWidget("test_drawer", drawer)

	got, ok := LookupWidget("test_drawer")
	require.True(t, ok, "registered widget must be looked up")
	require.NotNil(t, got)

	got(DrawerContext{})
	assert.Equal(t, 1, called, "looked-up drawer dispatches to the registered func")
}

// TestRegisterWidget_DuplicateNamePanics: re-registering an existing
// name (without ResetWidgetsForTest in between) panics. Matches
// Register[T] discipline.
func TestRegisterWidget_DuplicateNamePanics(t *testing.T) {
	ResetWidgetsForTest()
	RegisterWidget("dupe", func(ctx DrawerContext) bool { return false })
	assert.Panics(t, func() {
		RegisterWidget("dupe", func(ctx DrawerContext) bool { return false })
	}, "second registration of same name panics")
}

// TestRegisterWidget_EmptyNamePanics: the empty string is not a valid
// name; registration panics rather than silently shadowing.
func TestRegisterWidget_EmptyNamePanics(t *testing.T) {
	ResetWidgetsForTest()
	assert.Panics(t, func() {
		RegisterWidget("", func(ctx DrawerContext) bool { return false })
	})
}

// TestRegisterWidget_NilDrawerPanics: registering a nil drawer is a
// developer mistake; panic at registration so the broken state never
// reaches the inspector.
func TestRegisterWidget_NilDrawerPanics(t *testing.T) {
	ResetWidgetsForTest()
	assert.Panics(t, func() {
		RegisterWidget("nilly", nil)
	})
}

// TestLookupWidget_MissingReturnsFalse: looking up an unregistered
// name returns ok=false so the inspector's dispatch can fall back to
// a read-only-text render path without panicking.
func TestLookupWidget_MissingReturnsFalse(t *testing.T) {
	ResetWidgetsForTest()
	_, ok := LookupWidget("not_there")
	assert.False(t, ok)
}

// TestApplyPFTag_WidgetEqualsSyntax: a field with pf:"widget=<name>"
// resolves to WidgetKind=WidgetCustom and CustomWidget="<name>".
func TestApplyPFTag_WidgetEqualsSyntax(t *testing.T) {
	var md FieldMetadata
	require.NoError(t, applyPFTag(&md, "widget=tilepainter"))
	assert.Equal(t, WidgetCustom, md.WidgetKind)
	assert.Equal(t, "tilepainter", md.CustomWidget)
}

// TestApplyPFTag_WidgetEqualsRequiresName: pf:"widget=" with no name
// is a malformed tag — return an error so Register surfaces it as a
// register-time panic with a useful message.
func TestApplyPFTag_WidgetEqualsRequiresName(t *testing.T) {
	var md FieldMetadata
	err := applyPFTag(&md, "widget=")
	require.Error(t, err)
}

// TestApplyPFTag_UnknownTagStillFallsThroughToUnknown: a non-recognised
// tag without the widget= prefix still resolves to WidgetUnknown (no
// regression on the existing unknown-tag fallback).
func TestApplyPFTag_UnknownTagStillFallsThroughToUnknown(t *testing.T) {
	var md FieldMetadata
	require.NoError(t, applyPFTag(&md, "nonsense"))
	assert.Equal(t, WidgetUnknown, md.WidgetKind)
	assert.Empty(t, md.CustomWidget)
}

// TestRegister_CarriesCustomWidgetThroughTypeMetadata: a registered
// struct with pf:"widget=<name>" on a field carries the name into
// the FieldMetadata so the inspector can dispatch it.
func TestRegister_CarriesCustomWidgetThroughTypeMetadata(t *testing.T) {
	ResetForTest()
	type sample struct {
		Hook struct{} `json:"-" pf:"widget=samplehook"`
	}
	Register[sample]("Sample")
	md, ok := Get("Sample")
	require.True(t, ok)
	require.Len(t, md.Fields, 1)
	assert.Equal(t, WidgetCustom, md.Fields[0].WidgetKind)
	assert.Equal(t, "samplehook", md.Fields[0].CustomWidget)
}
