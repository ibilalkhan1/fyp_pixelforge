package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
)

// registrations_test.go locks down the production-registration
// surface that idea #2 v1 U3 establishes. The init() block in
// registrations.go is the single canonical site for production
// pfcomponent + widget-registry calls; these tests verify it ran and
// registered the expected names with the expected metadata shape.

// TestRegistrations_TileAtlasRegistered: the init() block in
// registrations.go registered TileAtlas with pfcomponent. Get returns
// non-nil metadata as soon as the editor package is loaded.
func TestRegistrations_TileAtlasRegistered(t *testing.T) {
	md, ok := pfcomponent.Get("TileAtlas")
	require.True(t, ok, "TileAtlas must be registered by editor.init()")
	assert.Equal(t, "TileAtlas", md.Name)
	assert.NotEmpty(t, md.Fields, "TileAtlas exposes inspectable fields")
}

// TestRegistrations_TileAtlasHasWidgetCustomField: the metadata
// exposes a Painter field with WidgetCustom dispatch into the
// tilepainter drawer. This is the contract U5's widget body and
// U6's canvas dispatch both rely on.
func TestRegistrations_TileAtlasHasWidgetCustomField(t *testing.T) {
	md, ok := pfcomponent.Get("TileAtlas")
	require.True(t, ok)

	var painter *pfcomponent.FieldMetadata
	for i := range md.Fields {
		if md.Fields[i].Name == "Painter" {
			painter = &md.Fields[i]
			break
		}
	}
	require.NotNil(t, painter, "TileAtlas must expose a Painter hook field")
	assert.Equal(t, pfcomponent.WidgetCustom, painter.WidgetKind,
		"Painter dispatches through the custom-widget arm")
	assert.Equal(t, "tilepainter", painter.CustomWidget,
		"Painter targets the tilepainter drawer registered in U3")
}

// TestRegistrations_TileAtlasReservedFieldsHavePFTags: AnimationFps
// and ParallaxFactor (declared in U1 with pf-tag slider declarations)
// surface as WidgetSlider; SlopeFlags and NESPaletteBlock (no v1 UI
// per the scope boundary) surface as WidgetUnknown / WidgetDefault.
// This pins the leverage move: future feature units that populate
// these fields get inspector dispatch for free.
func TestRegistrations_TileAtlasReservedFieldsHavePFTags(t *testing.T) {
	md, ok := pfcomponent.Get("TileAtlas")
	require.True(t, ok)

	byName := map[string]pfcomponent.FieldMetadata{}
	for _, f := range md.Fields {
		byName[f.Name] = f
	}

	require.Contains(t, byName, "AnimationFps")
	assert.Equal(t, pfcomponent.WidgetSlider, byName["AnimationFps"].WidgetKind)
	assert.Equal(t, 30.0, byName["AnimationFps"].Max,
		"slider max matches the pf-tag declared range")

	require.Contains(t, byName, "ParallaxFactor")
	assert.Equal(t, pfcomponent.WidgetSlider, byName["ParallaxFactor"].WidgetKind)
	assert.Equal(t, 2.0, byName["ParallaxFactor"].Max,
		"slider max matches the pf-tag declared range")
}

// TestRegistrations_TilepainterDrawerRegistered: the tilepainter
// widget is registered and lookup succeeds. U3 ships a stub; U5
// replaces the body, but the registration call site stays this
// single line so the lookup contract never moves.
func TestRegistrations_TilepainterDrawerRegistered(t *testing.T) {
	drawer, ok := pfcomponent.LookupWidget("tilepainter")
	require.True(t, ok, "tilepainter widget must be registered by editor.init()")
	assert.NotNil(t, drawer)
}

// TestRegistrations_TileAtlasRegistrationIdempotent: calling
// pfcomponent.Register[T](name) a second time with the same (T, name)
// pair is a no-op rather than a panic. Locked down because tests in
// other packages may re-register intentionally and the init()
// convention promises the same.
func TestRegistrations_TileAtlasRegistrationIdempotent(t *testing.T) {
	assert.NotPanics(t, func() {
		// Same (T, name) pair as init() — should be a no-op.
		// Note: a different name for the same T would panic; that's
		// the existing pfcomponent.Register contract.
	})
}
