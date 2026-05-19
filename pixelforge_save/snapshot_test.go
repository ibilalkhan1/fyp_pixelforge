package pixelforge_save_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"
)

func TestSnapshot_MarshalRoundTrip(t *testing.T) {
	original := pisave.Snapshot{
		SchemaVersion:  pisave.CurrentSchemaVersion,
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
	assert.Contains(t, string(data), `"schema_version": 2`)
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
	// Future version → best-effort load + warning log (default path).
	data := []byte(`{"schema_version":99,"game_title":"future"}`)
	got, err := pisave.UnmarshalSnapshot(data)
	require.NoError(t, err)
	assert.Equal(t, 99, got.SchemaVersion)
	assert.Equal(t, "future", got.GameTitle)
}

func TestSnapshot_StableIndentedJSONOutput(t *testing.T) {
	s := pisave.Snapshot{SchemaVersion: pisave.CurrentSchemaVersion, GameTitle: "test"}
	data, err := s.MarshalToJSON()
	require.NoError(t, err)
	// Two-space indent matches pixelforge_project/saver.go convention.
	assert.True(t, strings.Contains(string(data), "  \"schema_version\""))
}

// --- Plan-009 U10 v2 schema tests ---

func TestSnapshot_V2_RoundTripEntitiesGlobalsTick(t *testing.T) {
	// Build a v2 snapshot with the new entity-component reflection
	// payload + globals + tick + per-entity body state. Marshal +
	// unmarshal → assert each field survives byte-equal.
	livesRaw, err := json.Marshal(3)
	require.NoError(t, err)
	scoreRaw, err := json.Marshal(42)
	require.NoError(t, err)

	inv, err := json.Marshal(map[string]any{"holder": "player_1"})
	require.NoError(t, err)

	original := pisave.Snapshot{
		SchemaVersion: 2,
		GameTitle:     "DK",
		SavedAt:       time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC),
		Tick:          1234,
		Globals: map[string]json.RawMessage{
			"lives": livesRaw,
			"score": scoreRaw,
		},
		Entities: []pisave.EntitySnapshotV2{
			{
				ID:       "player_1",
				Position: [3]float64{1.5, 2.5, 0},
				Components: map[string]json.RawMessage{
					"inventory": inv,
				},
				BodyState: &pisave.BodyStateV2{
					PositionX: 1.5,
					PositionY: 2.5,
					VelocityX: 0,
					VelocityY: 0,
					Rotation:  0,
					Grounded:  true,
				},
			},
		},
	}
	data, err := original.MarshalToJSON()
	require.NoError(t, err)
	got, err := pisave.UnmarshalSnapshot(data)
	require.NoError(t, err)
	assert.Equal(t, uint64(1234), got.Tick)
	require.Len(t, got.Entities, 1)
	assert.Equal(t, "player_1", got.Entities[0].ID)
	assert.Equal(t, [3]float64{1.5, 2.5, 0}, got.Entities[0].Position)
	require.NotNil(t, got.Entities[0].BodyState)
	assert.Equal(t, 1.5, got.Entities[0].BodyState.PositionX)
	assert.True(t, got.Entities[0].BodyState.Grounded)
	assert.JSONEq(t, string(inv), string(got.Entities[0].Components["inventory"]))
}

func TestSnapshot_V1ForwardCompat_LoadsLegacyFields(t *testing.T) {
	// A v1 snapshot on disk (SchemaVersion=1, only legacy fields).
	// UnmarshalSnapshot must load it without error and the legacy
	// fields must populate; the new v2 fields stay empty.
	data := []byte(`{
  "schema_version": 1,
  "game_title": "v1 game",
  "saved_at": "2026-05-18T00:00:00Z",
  "blackboard": {"score": 100},
  "current_scene_id": "level1",
  "player_pos": {"tile_x": 5, "tile_y": 10},
  "scene_entities": [
    {"id": "npc1", "tile_x": 2, "tile_y": 3}
  ]
}`)
	got, err := pisave.UnmarshalSnapshot(data)
	require.NoError(t, err)
	assert.Equal(t, 1, got.SchemaVersion)
	assert.Equal(t, "level1", got.CurrentSceneID)
	assert.Equal(t, 5, got.PlayerPos.TileX)
	require.Len(t, got.SceneEntities, 1)
	assert.Empty(t, got.Entities)
	assert.Empty(t, got.Globals)
}

func TestSnapshot_UnmarshalStrict_RefusesFuture(t *testing.T) {
	data := []byte(`{"schema_version":99,"game_title":"future"}`)
	_, err := pisave.UnmarshalStrict(data)
	require.Error(t, err)
	assert.True(t, errors.Is(err, pisave.ErrFutureSchema))
}

func TestSnapshot_UnmarshalStrict_AcceptsCurrentAndOlder(t *testing.T) {
	current := pisave.Snapshot{SchemaVersion: pisave.CurrentSchemaVersion, GameTitle: "now"}
	data, err := current.MarshalToJSON()
	require.NoError(t, err)
	got, err := pisave.UnmarshalStrict(data)
	require.NoError(t, err)
	assert.Equal(t, "now", got.GameTitle)

	v1 := []byte(`{"schema_version":1,"game_title":"old","saved_at":"2026-05-18T00:00:00Z"}`)
	gotV1, err := pisave.UnmarshalStrict(v1)
	require.NoError(t, err)
	assert.Equal(t, 1, gotV1.SchemaVersion)
}

func TestSnapshot_DeterministicMarshal(t *testing.T) {
	// Build a snapshot with map-based fields (Globals + per-entity
	// Components are map[string]json.RawMessage). Marshal 100 times;
	// every output must be byte-identical. Verifies the stdlib's
	// sorted-key contract holds across the v2 fields.
	inv, _ := json.Marshal(map[string]any{"holder": "player_1", "item_count": 3})
	stats, _ := json.Marshal(map[string]any{"hp": 5, "max_hp": 10})
	livesRaw, _ := json.Marshal(3)
	scoreRaw, _ := json.Marshal(42)
	s := pisave.Snapshot{
		SchemaVersion: 2,
		GameTitle:     "deterministic",
		SavedAt:       time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC),
		Tick:          7,
		Globals: map[string]json.RawMessage{
			"score": scoreRaw,
			"lives": livesRaw,
		},
		Entities: []pisave.EntitySnapshotV2{
			{
				ID:       "player_1",
				Position: [3]float64{1, 2, 0},
				Components: map[string]json.RawMessage{
					"inventory": inv,
					"health":    stats,
				},
			},
		},
	}
	first, err := s.MarshalToJSON()
	require.NoError(t, err)
	for i := 0; i < 99; i++ {
		data, err := s.MarshalToJSON()
		require.NoError(t, err)
		assert.Equal(t, string(first), string(data), "marshal output drifted at iteration %d", i)
	}
}
