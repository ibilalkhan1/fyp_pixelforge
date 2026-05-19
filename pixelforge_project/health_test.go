package pixelforge_project_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

func TestHealthFor_ReturnsHPAndMaxHP(t *testing.T) {
	e := &pixelforge_project.Entity{
		ID: "goomba_1",
		Components: []pixelforge_project.EntityComponent{
			{
				Type:   pixelforge_project.HealthComponentType,
				Values: map[string]any{"hp": float64(5), "max_hp": float64(10)},
			},
		},
	}

	hp, maxHP, ok := pixelforge_project.HealthFor(e)
	assert.True(t, ok)
	assert.Equal(t, 5, hp)
	assert.Equal(t, 10, maxHP)
}

func TestHealthFor_MissingComponentReportsNotOK(t *testing.T) {
	e := &pixelforge_project.Entity{
		ID: "wall",
	}

	hp, maxHP, ok := pixelforge_project.HealthFor(e)
	assert.False(t, ok)
	assert.Equal(t, 0, hp)
	assert.Equal(t, 0, maxHP)
}

func TestHealthFor_NilEntityIsSafe(t *testing.T) {
	hp, maxHP, ok := pixelforge_project.HealthFor(nil)
	assert.False(t, ok)
	assert.Equal(t, 0, hp)
	assert.Equal(t, 0, maxHP)
}

func TestHealthFor_AcceptsIntStorage(t *testing.T) {
	// Programmatic tests sometimes pass int rather than the JSON-
	// normal float64. The helper coerces both.
	e := &pixelforge_project.Entity{
		Components: []pixelforge_project.EntityComponent{
			{
				Type:   pixelforge_project.HealthComponentType,
				Values: map[string]any{"hp": 7, "max_hp": 7},
			},
		},
	}

	hp, maxHP, ok := pixelforge_project.HealthFor(e)
	assert.True(t, ok)
	assert.Equal(t, 7, hp)
	assert.Equal(t, 7, maxHP)
}

func TestSetHealthFor_WritesNewHP(t *testing.T) {
	e := &pixelforge_project.Entity{
		Components: []pixelforge_project.EntityComponent{
			{
				Type:   pixelforge_project.HealthComponentType,
				Values: map[string]any{"hp": float64(5), "max_hp": float64(10)},
			},
		},
	}

	ok := pixelforge_project.SetHealthFor(e, 3)
	assert.True(t, ok)

	hp, maxHP, _ := pixelforge_project.HealthFor(e)
	assert.Equal(t, 3, hp)
	assert.Equal(t, 10, maxHP, "max_hp must be preserved")
}

func TestSetHealthFor_ClampsNegativeToZero(t *testing.T) {
	e := &pixelforge_project.Entity{
		Components: []pixelforge_project.EntityComponent{
			{
				Type:   pixelforge_project.HealthComponentType,
				Values: map[string]any{"hp": float64(5), "max_hp": float64(10)},
			},
		},
	}

	ok := pixelforge_project.SetHealthFor(e, -3)
	assert.True(t, ok)

	hp, _, _ := pixelforge_project.HealthFor(e)
	assert.Equal(t, 0, hp, "negative hp must clamp to 0")
}

func TestSetHealthFor_MissingComponentReturnsFalse(t *testing.T) {
	e := &pixelforge_project.Entity{ID: "wall"}

	ok := pixelforge_project.SetHealthFor(e, 3)
	assert.False(t, ok, "non-damageable entity should report no-write")
}

func TestSetHealthFor_NilEntityIsSafe(t *testing.T) {
	ok := pixelforge_project.SetHealthFor(nil, 5)
	assert.False(t, ok)
}

func TestSetHealthFor_InitialisesNilValuesMap(t *testing.T) {
	e := &pixelforge_project.Entity{
		Components: []pixelforge_project.EntityComponent{
			{Type: pixelforge_project.HealthComponentType, Values: nil},
		},
	}

	ok := pixelforge_project.SetHealthFor(e, 4)
	assert.True(t, ok)

	hp, _, _ := pixelforge_project.HealthFor(e)
	assert.Equal(t, 4, hp)
}
