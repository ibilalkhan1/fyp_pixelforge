package pixelforge_physics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBody_BasicFields(t *testing.T) {
	b := NewBody("hero", Vec2{FixedFromInt(10), FixedFromInt(20)},
		FixedFromInt(16), FixedFromInt(16))
	require.NotNil(t, b)
	assert.Equal(t, "hero", b.ID)
	assert.Equal(t, FixedFromInt(10), b.Position.X)
	assert.Equal(t, FixedFromInt(20), b.Position.Y)
	assert.Equal(t, FixedFromInt(16), b.Width)
	assert.NotNil(t, b.Shape(), "shape must be attached")
}

func TestBody_Sync_UpdatesShape(t *testing.T) {
	b := NewBody("x", Vec2{}, FixedFromInt(8), FixedFromInt(8))
	b.Position = Vec2{FixedFromInt(42), FixedFromInt(7)}
	b.Sync()
	pos := b.Shape().Position()
	assert.InDelta(t, 42.0, pos.X, 1e-9)
	assert.InDelta(t, 7.0, pos.Y, 1e-9)
}

func TestBody_Tags(t *testing.T) {
	b := NewBody("p", Vec2{}, FixedFromInt(1), FixedFromInt(1))
	b.Tags = TagSolid | TagPlatform
	assert.True(t, b.HasTag(TagSolid))
	assert.True(t, b.HasTag(TagPlatform))
	assert.False(t, b.HasTag(TagOneWay))
}
