package pixelforge_save_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"
)

func TestSnapshot_MarshalRoundTrip(t *testing.T) {
	original := pisave.Snapshot{
		SchemaVersion:  1,
		GameTitle:      "My Game",
		SavedAt:        time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC),
		Blackboard:     map[string]any{"score": float64(100), "name": "hero"},
		CurrentSceneID: "level1",
		PlayerPos:      pisave.PlayerPosition{TileX: 5, TileY: 10},
		SceneEntities: []pisave.EntitySnapshot{
			{ID: "npc1", TileX: 2, TileY: 3, Values: map[string]any{"dialogue_progress": float64(1)}},
		},
	}
	data, err := original.MarshalToJSON()
	require.NoError(t, err)
	got, err := pisave.UnmarshalSnapshot(data)
	require.NoError(t, err)

	assert.Equal(t, original.GameTitle, got.GameTitle)
	assert.Equal(t, original.CurrentSceneID, got.CurrentSceneID)
	assert.Equal(t, original.PlayerPos, got.PlayerPos)
	require.Len(t, got.SceneEntities, 1)
	assert.Equal(t, "npc1", got.SceneEntities[0].ID)
	assert.Equal(t, float64(100), got.Blackboard["score"])
}

func TestSnapshot_DefaultsSchemaVersionOnMarshal(t *testing.T) {
	s := pisave.Snapshot{GameTitle: "Test"}
	data, err := s.MarshalToJSON()
	require.NoError(t, err)
	assert.Contains(t, string(data), `"schema_version": 1`)
}

func TestSnapshot_UnmarshalEmptyReturnsError(t *testing.T) {
	_, err := pisave.UnmarshalSnapshot(nil)
	require.Error(t, err)
}

func TestSnapshot_UnmarshalMalformedJSONReturnsError(t *testing.T) {
	_, err := pisave.UnmarshalSnapshot([]byte("not-json"))
	require.Error(t, err)
}

func TestSnapshot_OldSchemaVersionDefaultsTo1(t *testing.T) {
	// A snapshot lacking the version field (zero) treated as v1.
	data := []byte(`{"game_title":"old","saved_at":"2026-05-18T00:00:00Z"}`)
	got, err := pisave.UnmarshalSnapshot(data)
	require.NoError(t, err)
	assert.Equal(t, 1, got.SchemaVersion)
}

func TestSnapshot_FutureSchemaVersionLogsBestEffort(t *testing.T) {
	// Future version → best-effort load + warning log.
	data := []byte(`{"schema_version":99,"game_title":"future"}`)
	got, err := pisave.UnmarshalSnapshot(data)
	require.NoError(t, err)
	assert.Equal(t, 99, got.SchemaVersion)
	assert.Equal(t, "future", got.GameTitle)
}

func TestSnapshot_StableIndentedJSONOutput(t *testing.T) {
	s := pisave.Snapshot{SchemaVersion: 1, GameTitle: "test"}
	data, err := s.MarshalToJSON()
	require.NoError(t, err)
	// Two-space indent matches pixelforge_project/saver.go convention.
	assert.True(t, strings.Contains(string(data), "  \"schema_version\""))
}
