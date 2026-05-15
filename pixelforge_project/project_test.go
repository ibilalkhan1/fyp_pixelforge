package pixelforge_project

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Zero-value Project marshals without error and unmarshals back equal.
func TestProject_ZeroValueRoundTrip(t *testing.T) {
	original := Project{}
	original.normalizeSlices()

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var loaded Project
	require.NoError(t, json.Unmarshal(data, &loaded))
	loaded.normalizeSlices()

	assert.Equal(t, original, loaded)
}

// A populated snake-shaped project round-trips through JSON.
func TestProject_SnakeShapedRoundTrip(t *testing.T) {
	original := snakeShapedProject(t)

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var loaded Project
	require.NoError(t, json.Unmarshal(data, &loaded))
	loaded.normalizeSlices()

	assert.Equal(t, original.Name, loaded.Name)
	assert.Equal(t, original.ScreenWidth, loaded.ScreenWidth)
	assert.Equal(t, original.ScreenHeight, loaded.ScreenHeight)
	assert.Equal(t, original.TPS, loaded.TPS)
	assert.Len(t, loaded.Sprites, 1)
	assert.Len(t, loaded.Scenes, 1)
	assert.Len(t, loaded.Scenes[0].Entities, 5)
	assert.Equal(t, "fruit", loaded.Scenes[0].Entities[0].Components[0].Type)
}

// Nil slices marshal as [] not null. Loaders that don't tolerate null
// won't trip on freshly-created projects.
func TestProject_EmptySlicesNotNull(t *testing.T) {
	p := NewProject("empty")
	// Sprites/Audio/Scenes etc. default to []
	data, err := json.Marshal(p)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"sprites":[]`)
	assert.Contains(t, string(data), `"audio":[]`)
	assert.Contains(t, string(data), `"behaviors":[]`)
	assert.Contains(t, string(data), `"bindings":[]`)
	assert.NotContains(t, string(data), `"sprites":null`)
}

// Two marshals of the same Project produce byte-identical JSON. This is
// the git-merge-friendly invariant the schema promises.
func TestProject_DeterministicMarshalOrder(t *testing.T) {
	p := snakeShapedProject(t)

	first, err := json.MarshalIndent(p, "", "  ")
	require.NoError(t, err)
	second, err := json.MarshalIndent(p, "", "  ")
	require.NoError(t, err)

	assert.True(t, bytes.Equal(first, second), "marshals must be byte-identical")
}

// EntityComponent.Values with multiple keys serializes them in sorted
// order each time so diffs stay tiny.
func TestEntityComponent_ValuesSorted(t *testing.T) {
	c := EntityComponent{
		Type: "MoveLogic",
		Values: map[string]any{
			"zebra":  1,
			"apple":  2,
			"middle": 3,
		},
	}
	data, err := json.Marshal(c)
	require.NoError(t, err)
	s := string(data)
	// "apple" must appear before "middle" before "zebra"
	posApple := strings.Index(s, "apple")
	posMiddle := strings.Index(s, "middle")
	posZebra := strings.Index(s, "zebra")
	require.NotEqual(t, -1, posApple)
	require.NotEqual(t, -1, posMiddle)
	require.NotEqual(t, -1, posZebra)
	assert.Less(t, posApple, posMiddle)
	assert.Less(t, posMiddle, posZebra)
}

// Nil Values serializes as {} (not null) so the schema is always
// consumable as an object.
func TestEntityComponent_NilValuesToEmptyObject(t *testing.T) {
	c := EntityComponent{Type: "Empty"}
	data, err := json.Marshal(c)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"values":{}`)
}

// NewProject creates a valid v1 project with sensible defaults.
func TestNewProject_Defaults(t *testing.T) {
	p := NewProject("test")
	assert.Equal(t, SchemaVersion, p.SchemaVersion)
	assert.Equal(t, "test", p.Name)
	assert.Equal(t, 320, p.ScreenWidth)
	assert.Equal(t, 180, p.ScreenHeight)
	assert.Equal(t, 30, p.TPS)
	assert.NotZero(t, p.CreatedAt)
	assert.NotZero(t, p.ModifiedAt)
	assert.Len(t, p.Scenes, 1)
	assert.Equal(t, "main", p.Scenes[0].ID)
}

// DefaultPalette produces a 64-color grey ramp and identity color tables.
func TestDefaultPalette(t *testing.T) {
	pd := DefaultPalette()
	assert.Equal(t, MaxColors, len(pd.Base))
	assert.Equal(t, "#000000", pd.Base[0])
	assert.Equal(t, "#ffffff", pd.Base[MaxColors-1])
	// Identity: ColorTables[t][s][d] == s
	for t2 := 0; t2 < 4; t2++ {
		for s := 0; s < MaxColors; s++ {
			assert.Equal(t, uint8(s), pd.ColorTables[t2][s][s])
		}
	}
}

// snakeShapedProject builds a representative project for round-trip
// tests. Mirrors the snake example's layout in broad strokes.
func snakeShapedProject(t *testing.T) *Project {
	t.Helper()
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	return &Project{
		SchemaVersion: SchemaVersion,
		Name:          "snake",
		ScreenWidth:   128,
		ScreenHeight:  128,
		TPS:           30,
		CreatedAt:     now,
		ModifiedAt:    now,
		Palette:       DefaultPalette(),
		Sprites: []SpriteAsset{
			{
				Name:         "sprites",
				RelativePath: "sprites/sprites.png",
				Width:        32, Height: 8,
				FrameW: 8, FrameH: 8,
				Animations: []AnimationClip{},
			},
		},
		Audio: []AudioSample{},
		Scenes: []Scene{
			{
				ID:   "main",
				Name: "Main",
				Entities: []Entity{
					{ID: "fruit-0", Name: "Fruit", Position: EntityPosition{X: 64, Y: 64}, Components: []EntityComponent{{Type: "fruit", Values: map[string]any{}}}},
					{ID: "head-0", Name: "Head", Position: EntityPosition{X: 32, Y: 32}, Components: []EntityComponent{{Type: "snake_head", Values: map[string]any{}}}},
					{ID: "body-0", Name: "Body", Position: EntityPosition{X: 24, Y: 32}, Components: []EntityComponent{{Type: "snake_body", Values: map[string]any{}}}},
					{ID: "body-1", Name: "Body", Position: EntityPosition{X: 16, Y: 32}, Components: []EntityComponent{{Type: "snake_body", Values: map[string]any{}}}},
					{ID: "body-2", Name: "Body", Position: EntityPosition{X: 8, Y: 32}, Components: []EntityComponent{{Type: "snake_body", Values: map[string]any{}}}},
				},
			},
		},
		Behaviors:          []BehaviorGraph{},
		Bindings:           []AudioBinding{},
		EventSubscriptions: []EventSubscription{},
		ExtensionHooks:     []ExtensionHook{},
	}
}
