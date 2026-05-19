//go:build !js

package capsuleruntime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_physics"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"
)

// Plan-009 U10 — native-only round-trip tests. The native backend
// uses os filesystem APIs and is build-tagged off under js/wasm; the
// tests here exercise the on-disk save path (the snapshot lives as a
// JSON file under the test's t.TempDir()). The WASM path is verified
// by the cross-platform tests in snapshot_test.go (in-memory backend
// + the GOOS=js GOARCH=wasm go vet gate); real WASM round-trip in
// the browser is deferred to U12+ harness work.

func TestEncodeDecode_RoundTripNativeBackend(t *testing.T) {
	// Boot a runtime, populate state, SaveNow via the real native
	// backend (writing into t.TempDir()), boot a fresh runtime
	// against the same project, LoadSlot, assert state restored
	// byte-equal (incl. Fixed32 body state).
	dir := t.TempDir()
	backend := pisave.NewBackendNativeAtPath(dir)
	rt := bootSnapshotRuntime(t, backend)
	rt.Tick = 1234
	rt.Globals["lives"] = 3
	rt.Globals["score"] = 42
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

	rt.Sinks.Save.SaveNow("slot1")

	// Fresh runtime against the same project + backend dir.
	rt2 := bootSnapshotRuntime(t, pisave.NewBackendNativeAtPath(dir))
	rt2.Sinks.Save.LoadSlot("slot1")

	assert.Equal(t, uint64(1234), rt2.Tick)
	assert.Equal(t, float64(3), rt2.Globals["lives"])
	assert.Equal(t, float64(42), rt2.Globals["score"])
	require.Len(t, rt2.CurrentScene.Entities, 2)
	assert.Equal(t, "player_1", rt2.CurrentScene.Entities[0].ID)
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

func TestEncodeDecode_AE4_MidGameSaveLoadRestoresAllState(t *testing.T) {
	// AE4: mid-game save → reset → load → state restored. The full
	// surface: tick, globals, per-entity components, per-entity
	// body state, scene pointer. Uses the real native backend so the
	// save path lands on disk and the reset-then-load flow mirrors
	// what the studio's "Save / Load" buttons do today.
	dir := t.TempDir()
	rt := bootSnapshotRuntime(t, pisave.NewBackendNativeAtPath(dir))
	rt.Tick = 540
	rt.Globals["lives"] = 2
	rt.Globals["score"] = 1500
	rt.Globals["player_entity"] = "player_1"
	pixelforge_project.SetHealthFor(&rt.CurrentScene.Entities[0], 4)
	pixelforge_project.SetHealthFor(&rt.CurrentScene.Entities[1], 1)
	body := rt.Bodies["player_1"]
	body.Position.X = pixelforge_physics.FixedFromFloat(120.5)
	body.Position.Y = pixelforge_physics.FixedFromFloat(80.25)
	body.Velocity.X = pixelforge_physics.FixedFromFloat(2.5)
	body.OnLadder = true
	body.LadderClimbing = true

	rt.Sinks.Save.SaveNow("midgame")

	// "Reset" = boot a fresh runtime (the studio's reset-then-load
	// flow). Use the same project + the same on-disk backend.
	rt2 := bootSnapshotRuntime(t, pisave.NewBackendNativeAtPath(dir))
	// Pre-load: ensure the fresh runtime has authored state (HP 7
	// from the bootSnapshotRuntime builder), so we can prove load
	// overwrote it.
	hp, _, ok := pixelforge_project.HealthFor(&rt2.CurrentScene.Entities[0])
	require.True(t, ok)
	require.Equal(t, 7, hp)

	rt2.Sinks.Save.LoadSlot("midgame")

	assert.Equal(t, uint64(540), rt2.Tick)
	assert.Equal(t, float64(2), rt2.Globals["lives"])
	assert.Equal(t, float64(1500), rt2.Globals["score"])
	assert.Equal(t, "player_1", rt2.Globals["player_entity"])

	hp, _, ok = pixelforge_project.HealthFor(&rt2.CurrentScene.Entities[0])
	require.True(t, ok)
	assert.Equal(t, 4, hp)
	hp, _, ok = pixelforge_project.HealthFor(&rt2.CurrentScene.Entities[1])
	require.True(t, ok)
	assert.Equal(t, 1, hp)

	b2 := rt2.Bodies["player_1"]
	require.NotNil(t, b2)
	assert.Equal(t, pixelforge_physics.FixedFromFloat(120.5), b2.Position.X)
	assert.Equal(t, pixelforge_physics.FixedFromFloat(80.25), b2.Position.Y)
	assert.Equal(t, pixelforge_physics.FixedFromFloat(2.5), b2.Velocity.X)
	assert.True(t, b2.OnLadder)
	assert.True(t, b2.LadderClimbing)
}
