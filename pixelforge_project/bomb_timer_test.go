package pixelforge_project_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

func TestBombTimerFor_ReturnsAllFields(t *testing.T) {
	e := &pixelforge_project.Entity{
		ID: "bomb_1",
		Components: []pixelforge_project.EntityComponent{
			{
				Type: pixelforge_project.BombTimerComponentType,
				Values: map[string]any{
					"ticks_remaining": float64(120),
					"range":           float64(3),
					"grid_x":          float64(5),
					"grid_y":          float64(7),
				},
			},
		},
	}

	bt, ok := pixelforge_project.BombTimerFor(e)
	require.True(t, ok)
	assert.Equal(t, 120, bt.TicksRemaining)
	assert.Equal(t, 3, bt.Range)
	assert.Equal(t, 5, bt.GridX)
	assert.Equal(t, 7, bt.GridY)
}

func TestBombTimerFor_MissingComponentReportsNotOK(t *testing.T) {
	e := &pixelforge_project.Entity{ID: "wall"}

	_, ok := pixelforge_project.BombTimerFor(e)
	assert.False(t, ok)
}

func TestBombTimerFor_NilEntityIsSafe(t *testing.T) {
	_, ok := pixelforge_project.BombTimerFor(nil)
	assert.False(t, ok)
}

func TestBombTimerFor_AcceptsIntStorage(t *testing.T) {
	// Programmatic tests sometimes pass int rather than the JSON-
	// normal float64. The helper coerces both via readIntValue.
	e := &pixelforge_project.Entity{
		Components: []pixelforge_project.EntityComponent{
			{
				Type: pixelforge_project.BombTimerComponentType,
				Values: map[string]any{
					"ticks_remaining": 30,
					"range":           2,
					"grid_x":          1,
					"grid_y":          1,
				},
			},
		},
	}

	bt, ok := pixelforge_project.BombTimerFor(e)
	require.True(t, ok)
	assert.Equal(t, 30, bt.TicksRemaining)
	assert.Equal(t, 2, bt.Range)
}

func TestSetBombTimerFor_WritesNewValues(t *testing.T) {
	e := &pixelforge_project.Entity{
		Components: []pixelforge_project.EntityComponent{
			{
				Type: pixelforge_project.BombTimerComponentType,
				Values: map[string]any{
					"ticks_remaining": float64(120),
					"range":           float64(3),
					"grid_x":          float64(5),
					"grid_y":          float64(7),
				},
			},
		},
	}

	ok := pixelforge_project.SetBombTimerFor(e, pixelforge_project.BombTimer{
		TicksRemaining: 30,
		Range:          3,
		GridX:          5,
		GridY:          7,
	})
	require.True(t, ok)

	bt, _ := pixelforge_project.BombTimerFor(e)
	assert.Equal(t, 30, bt.TicksRemaining)
	assert.Equal(t, 3, bt.Range)
}

func TestSetBombTimerFor_ClampsNegativeTicksToZero(t *testing.T) {
	e := &pixelforge_project.Entity{
		Components: []pixelforge_project.EntityComponent{
			{
				Type:   pixelforge_project.BombTimerComponentType,
				Values: map[string]any{"ticks_remaining": float64(5)},
			},
		},
	}

	ok := pixelforge_project.SetBombTimerFor(e, pixelforge_project.BombTimer{TicksRemaining: -3, Range: 2})
	require.True(t, ok)

	bt, _ := pixelforge_project.BombTimerFor(e)
	assert.Equal(t, 0, bt.TicksRemaining)
}

func TestSetBombTimerFor_MissingComponentReturnsFalse(t *testing.T) {
	e := &pixelforge_project.Entity{ID: "wall"}

	ok := pixelforge_project.SetBombTimerFor(e, pixelforge_project.BombTimer{TicksRemaining: 30})
	assert.False(t, ok, "non-bomb entity should report no-write")
}

func TestSetBombTimerFor_NilEntityIsSafe(t *testing.T) {
	ok := pixelforge_project.SetBombTimerFor(nil, pixelforge_project.BombTimer{TicksRemaining: 30})
	assert.False(t, ok)
}

func TestSetBombTimerFor_InitialisesNilValuesMap(t *testing.T) {
	e := &pixelforge_project.Entity{
		Components: []pixelforge_project.EntityComponent{
			{Type: pixelforge_project.BombTimerComponentType, Values: nil},
		},
	}

	ok := pixelforge_project.SetBombTimerFor(e, pixelforge_project.BombTimer{
		TicksRemaining: 30,
		Range:          3,
		GridX:          2,
		GridY:          2,
	})
	require.True(t, ok)

	bt, _ := pixelforge_project.BombTimerFor(e)
	assert.Equal(t, 30, bt.TicksRemaining)
	assert.Equal(t, 2, bt.GridX)
}

func TestRemoveBombTimerFor_StripsComponent(t *testing.T) {
	e := &pixelforge_project.Entity{
		Components: []pixelforge_project.EntityComponent{
			{Type: "tag", Values: map[string]any{"flag": "x"}},
			{Type: pixelforge_project.BombTimerComponentType, Values: map[string]any{"ticks_remaining": float64(30)}},
		},
	}

	ok := pixelforge_project.RemoveBombTimerFor(e)
	require.True(t, ok)

	_, hasBT := pixelforge_project.BombTimerFor(e)
	assert.False(t, hasBT, "BombTimer must be gone after Remove")
	require.Len(t, e.Components, 1)
	assert.Equal(t, "tag", e.Components[0].Type, "non-bomb components must survive")
}

func TestRemoveBombTimerFor_MissingComponentReturnsFalse(t *testing.T) {
	e := &pixelforge_project.Entity{ID: "wall"}

	ok := pixelforge_project.RemoveBombTimerFor(e)
	assert.False(t, ok)
}

func TestRemoveBombTimerFor_NilEntityIsSafe(t *testing.T) {
	ok := pixelforge_project.RemoveBombTimerFor(nil)
	assert.False(t, ok)
}
