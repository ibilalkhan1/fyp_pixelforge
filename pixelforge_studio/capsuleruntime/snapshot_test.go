package capsuleruntime_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_physics"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
)

// bootSnapshotRuntime constructs a populated runtime with one scene
// carrying entities + components + physics bodies. Tests share this
// builder so the encode/decode tests exercise the same realistic
// shape. Backend is caller-supplied so each test picks
// in-memory (cross-platform) or native (desktop only).
func bootSnapshotRuntime(t *testing.T, backend pisave.Backend) *capsuleruntime.Runtime {
	t.Helper()
	resetAll()
	pievent.ResetRegistryForTest()
	piloop.ResetVerbsBusForTest()

	p := pixelforge_project.NewProject("snap-test")
	p.PhysicsPreset = "asteroids"
	p.Scenes = []pixelforge_project.Scene{
		{
			ID:   "level1",
			Name: "Level 1",
			Entities: []pixelforge_project.Entity{
				{
					ID:       "player_1",
					Name:     "Player",
					Position: pixelforge_project.EntityPosition{X: 1.5, Y: 2.5, Z: 0},
					Components: []pixelforge_project.EntityComponent{
						{Type: "health", Values: map[string]any{"hp": float64(7), "max_hp": float64(10)}},
						{Type: "inventory", Values: map[string]any{"holder": "player_1", "item_count": float64(3)}},
					},
				},
				{
					ID:       "barrel_1",
					Name:     "Barrel",
					Position: pixelforge_project.EntityPosition{X: 40, Y: 10, Z: 0},
					Components: []pixelforge_project.EntityComponent{
						{Type: "health", Values: map[string]any{"hp": float64(2), "max_hp": float64(2)}},
					},
				},
			},
		},
	}
	rt, err := capsuleruntime.Boot(p, fakeAssets(nil), capsuleruntime.Options{
		SaveBackendOverride: backend,
	})
	require.NoError(t, err)
	return rt
}

func TestEncodeSnapshot_PopulatesV2Fields(t *testing.T) {
	rt := bootSnapshotRuntime(t, newMemBackend())
	rt.Tick = 42
	rt.Globals["lives"] = 3
	rt.Globals["score"] = 42
	rt.Globals["player_entity"] = "player_1"

	snap, err := capsuleruntime.EncodeSnapshot(rt)
	require.NoError(t, err)
	assert.Equal(t, pisave.CurrentSchemaVersion, snap.SchemaVersion)
	assert.Equal(t, uint64(42), snap.Tick)
	assert.Equal(t, "level1", snap.CurrentSceneID)
	require.Len(t, snap.Entities, 2)
	assert.Equal(t, "player_1", snap.Entities[0].ID)
	assert.Equal(t, [3]float64{1.5, 2.5, 0}, snap.Entities[0].Position)
	assert.Contains(t, snap.Entities[0].Components, "health")
	assert.Contains(t, snap.Entities[0].Components, "inventory")
	require.NotNil(t, snap.Entities[0].BodyState)
	assert.Equal(t, 1.5, snap.Entities[0].BodyState.PositionX)
	// Globals round-trip into RawMessage.
	require.Contains(t, snap.Globals, "lives")
	require.Contains(t, snap.Globals, "score")
	require.Contains(t, snap.Globals, "player_entity")
}

func TestEncodeDecode_RoundTripInMemoryBackend(t *testing.T) {
	// Backend-agnostic round-trip via the in-memory test backend.
	// Verifies the EncodeSnapshot → MarshalToJSON → backend.Write →
	// backend.Read → UnmarshalSnapshot → DecodeSnapshot path against
	// a backend that mimics the BackendJS contract (string blob in,
	// string blob out — no filesystem semantics). This is the
	// cross-platform (incl. WASM) round-trip proof; the native-only
	// path is exercised in snapshot_native_test.go.
	backend := newMemBackend()
	rt := bootSnapshotRuntime(t, backend)
	rt.Tick = 99
	rt.Globals["score"] = 100
	rt.Globals["lives"] = 3
	body := rt.Bodies["player_1"]
	require.NotNil(t, body)
	body.Velocity = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromFloat(1.5),
		Y: pixelforge_physics.FixedFromFloat(-0.5),
	}
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromFloat(99.25),
		Y: pixelforge_physics.FixedFromFloat(42.75),
	}
	body.Grounded = true
	pixelforge_project.SetHealthFor(&rt.CurrentScene.Entities[0], 5)

	rt.Sinks.Save.SaveNow("auto")
	require.NotEmpty(t, backend.data["auto"], "backend should hold the encoded snapshot")

	rt2 := bootSnapshotRuntime(t, backend)
	rt2.Sinks.Save.LoadSlot("auto")
	assert.Equal(t, uint64(99), rt2.Tick)
	assert.Equal(t, float64(100), rt2.Globals["score"])
	assert.Equal(t, float64(3), rt2.Globals["lives"])
	hp, _, ok := pixelforge_project.HealthFor(&rt2.CurrentScene.Entities[0])
	require.True(t, ok)
	assert.Equal(t, 5, hp)
	b2 := rt2.Bodies["player_1"]
	require.NotNil(t, b2)
	assert.Equal(t, pixelforge_physics.FixedFromFloat(99.25), b2.Position.X)
	assert.Equal(t, pixelforge_physics.FixedFromFloat(42.75), b2.Position.Y)
	assert.Equal(t, pixelforge_physics.FixedFromFloat(1.5), b2.Velocity.X)
	assert.Equal(t, pixelforge_physics.FixedFromFloat(-0.5), b2.Velocity.Y)
	assert.True(t, b2.Grounded)
}

func TestDecodeSnapshot_V1ForwardCompat(t *testing.T) {
	// A manually-constructed v1 snapshot — what the v1 sink stub
	// would have produced (Snapshot{} with SchemaVersion auto-bumped
	// to 1 by the service). Decode should not panic + should log
	// the migration warning.
	rt := bootSnapshotRuntime(t, newMemBackend())
	v1 := pisave.Snapshot{
		SchemaVersion:  1,
		GameTitle:      "v1 game",
		CurrentSceneID: "level1",
		Blackboard:     map[string]any{"score": float64(50)},
	}
	err := capsuleruntime.DecodeSnapshot(rt, v1)
	require.NoError(t, err)
	// Legacy Blackboard migrated into Globals.
	assert.Equal(t, float64(50), rt.Globals["score"])
	// Scene pointer resolved from CurrentSceneID.
	require.NotNil(t, rt.CurrentScene)
	assert.Equal(t, "level1", rt.CurrentScene.ID)
}

func TestEncodeSnapshot_EntityReferencePreservation(t *testing.T) {
	// An entity with a component carrying an entity-reference value
	// (Inventory.holder = "player_1") must round-trip byte-equal.
	rt := bootSnapshotRuntime(t, newMemBackend())
	snap, err := capsuleruntime.EncodeSnapshot(rt)
	require.NoError(t, err)
	data, err := snap.MarshalToJSON()
	require.NoError(t, err)
	got, err := pisave.UnmarshalSnapshot(data)
	require.NoError(t, err)

	rt2 := bootSnapshotRuntime(t, newMemBackend())
	require.NoError(t, capsuleruntime.DecodeSnapshot(rt2, got))

	// Find the inventory component on the restored player_1.
	var inv map[string]any
	for _, c := range rt2.CurrentScene.Entities[0].Components {
		if c.Type == "inventory" {
			inv = c.Values
			break
		}
	}
	require.NotNil(t, inv)
	assert.Equal(t, "player_1", inv["holder"])
}

func TestEncodeSnapshot_PositionByteEqual(t *testing.T) {
	rt := bootSnapshotRuntime(t, newMemBackend())
	// Position {1.5, 2.5, 0} round-trips byte-equal.
	snap, err := capsuleruntime.EncodeSnapshot(rt)
	require.NoError(t, err)
	assert.Equal(t, [3]float64{1.5, 2.5, 0}, snap.Entities[0].Position)

	data, err := snap.MarshalToJSON()
	require.NoError(t, err)
	got, err := pisave.UnmarshalSnapshot(data)
	require.NoError(t, err)
	assert.Equal(t, [3]float64{1.5, 2.5, 0}, got.Entities[0].Position)
}

func TestEncodeSnapshot_GlobalsByteEqual(t *testing.T) {
	rt := bootSnapshotRuntime(t, newMemBackend())
	rt.Globals["lives"] = 3
	rt.Globals["score"] = 42
	snap, err := capsuleruntime.EncodeSnapshot(rt)
	require.NoError(t, err)
	data, err := snap.MarshalToJSON()
	require.NoError(t, err)

	rt2 := bootSnapshotRuntime(t, newMemBackend())
	got, err := pisave.UnmarshalSnapshot(data)
	require.NoError(t, err)
	require.NoError(t, capsuleruntime.DecodeSnapshot(rt2, got))
	assert.Equal(t, float64(3), rt2.Globals["lives"])
	assert.Equal(t, float64(42), rt2.Globals["score"])
}

func TestEncodeDecode_LadderStateRestored(t *testing.T) {
	// DK ladder-climb state preserved across save → load. The body's
	// OnLadder + LadderClimbing bits are part of BodyStateV2.
	rt := bootSnapshotRuntime(t, newMemBackend())
	body := rt.Bodies["player_1"]
	require.NotNil(t, body)
	body.OnLadder = true
	body.LadderClimbing = true

	snap, err := capsuleruntime.EncodeSnapshot(rt)
	require.NoError(t, err)
	// Save → Load mediated by JSON to exercise the marshal-side
	// preservation (the in-memory test would otherwise hand the
	// same Go struct back).
	data, err := snap.MarshalToJSON()
	require.NoError(t, err)
	got, err := pisave.UnmarshalSnapshot(data)
	require.NoError(t, err)

	rt2 := bootSnapshotRuntime(t, newMemBackend())
	require.NoError(t, capsuleruntime.DecodeSnapshot(rt2, got))
	b2 := rt2.Bodies["player_1"]
	require.NotNil(t, b2)
	assert.True(t, b2.OnLadder)
	assert.True(t, b2.LadderClimbing)
}

func TestEncodeSnapshot_DeterministicAcrossRepeatedCalls(t *testing.T) {
	// Marshal the same state 100 times; every output byte-identical.
	rt := bootSnapshotRuntime(t, newMemBackend())
	rt.Tick = 7
	rt.Globals["lives"] = 3
	rt.Globals["score"] = 42
	rt.Globals["player_entity"] = "player_1"

	first, err := capsuleruntime.EncodeSnapshot(rt)
	require.NoError(t, err)
	firstBytes, err := first.MarshalToJSON()
	require.NoError(t, err)

	for i := 0; i < 99; i++ {
		snap, err := capsuleruntime.EncodeSnapshot(rt)
		require.NoError(t, err)
		got, err := snap.MarshalToJSON()
		require.NoError(t, err)
		assert.Equal(t, string(firstBytes), string(got), "encode drift at iter %d", i)
	}
}

func TestEncodeSnapshot_NilRuntimeReturnsError(t *testing.T) {
	_, err := capsuleruntime.EncodeSnapshot(nil)
	require.Error(t, err)
}

func TestDecodeSnapshot_NilRuntimeReturnsError(t *testing.T) {
	err := capsuleruntime.DecodeSnapshot(nil, pisave.Snapshot{})
	require.Error(t, err)
}

func TestEncodeDecode_MissingEntityIDRehydratesSnapshotEntity(t *testing.T) {
	// An entity ID present in the saved snapshot but ABSENT from the
	// rebuilt scene is appended to the scene with whatever state the
	// snapshot carries — matching the "load takes priority" semantics
	// (the scene may have lost entities; the save remembers them).
	// This is the entity-reference preservation contract.
	rt := bootSnapshotRuntime(t, newMemBackend())
	snap, err := capsuleruntime.EncodeSnapshot(rt)
	require.NoError(t, err)
	// Inject a third entity into the snapshot that's NOT in the
	// authored scene.
	extraValues, err := json.Marshal(map[string]any{"hp": float64(99)})
	require.NoError(t, err)
	snap.Entities = append(snap.Entities, pisave.EntitySnapshotV2{
		ID:       "spawned_runtime_only",
		Position: [3]float64{50, 50, 0},
		Components: map[string]json.RawMessage{
			"health": extraValues,
		},
	})

	rt2 := bootSnapshotRuntime(t, newMemBackend())
	require.NoError(t, capsuleruntime.DecodeSnapshot(rt2, snap))

	// Three entities in the rebuilt scene now (including the
	// snapshot-only entity).
	require.Len(t, rt2.CurrentScene.Entities, 3)
	assert.Equal(t, "spawned_runtime_only", rt2.CurrentScene.Entities[2].ID)
}
