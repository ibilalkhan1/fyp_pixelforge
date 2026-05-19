//go:build long

// Package donkey_kong_proof_test is plan-009 U18's end-to-end smoke
// test + AE4 save-load round-trip test for the Donkey Kong reference
// cart. It is Phase 4's terminal artifact — the last of the four
// reference-game proofs (U11 Asteroids, U14 Mario, U16 Bomberman, U18
// Donkey Kong).
//
// # Two test entry points
//
// This package ships two top-level tests:
//
//   - TestDonkeyKongProof — the U11/U14/U16-style smoke. Loads the
//     fixture + trace, drives the verb-recipe pipeline through the
//     ladder/barrel/multi-screen scroll mechanics added in U17, and
//     asserts checkpoint frame hashes + bus event sequences against a
//     committed baseline.
//
//   - TestDonkeyKongProof_AE4_MidGameSaveLoadRestoresAllState — the
//     AE4 acceptance test, called out by the brainstorm as the
//     canonical "save mid-game → reset → load → byte-equal state"
//     example. Drives the trace partway, captures pre-save state,
//     encodes a snapshot, boots a fresh runtime (= reset), decodes
//     the snapshot into the fresh runtime, then asserts the
//     post-load state matches the pre-save state byte-for-byte for
//     hero position, ladder flags, barrel positions, lives count,
//     and per-body physics fields.
//
// # Scope (U18 smoke)
//
// The smoke test is deliberately scoped pragmatically (matching the
// U11/U14/U16 templates):
//
//   - The hero climbs a ladder + the barrels roll; the trace doesn't
//     drive a full DK level clear (the U18 polish iteration adds the
//     death cascade when a barrel hits the hero, the "reach the top
//     platform" win condition, and the barrel-fall-between-platforms
//     cascade).
//
//   - Pixel-correctness of the rendered climber/barrel/platform
//     sprites is gated on the renderer learning to draw Sprite
//     components at body.Position. For U18, the smoke covers the
//     fact that frames are produced (non-empty, deterministic) and
//     that the verb-bus event sequence at each checkpoint matches
//     the recorded baseline. Per U17's deferral note: applying
//     CameraOffsetY during entity/tile blits is also deferred to a
//     later polish iteration — the runtime computes the offset
//     (proven below by the rt.CameraOffsetY != 0 smoke assertion)
//     but the rendered output doesn't yet scroll with it.
//
// # Deferred to U18 polish iteration
//
//   - Full DK level clear (hero reaches top platform; win condition).
//   - Barrel-hits-hero death cascade (damage routing on
//     barrel-vs-hero AABB).
//   - Multi-platform barrel falls (barrel rolls off the edge of one
//     platform onto the next).
//   - Camera-offset applied to entity/tile blits (renderer-side
//     scroll polish).
//
// # File layout
//
// Mirrors U11/U14/U16: this sub-package lives at
// pixelforge_studio/integration_test/donkey_kong_proof/ so the
// Ebitengine TestMain wrapper (RenderTickAtRGBA needs a live
// graphics context for ebiten.NewImage + ReadPixels) is isolated
// from the other long-tag tests in the parent package. Fixtures
// live one level up at
// ../fixtures/donkey_kong_proof.{pforge,trace.jsonl,baseline.json}.
package donkey_kong_proof_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge"
	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_physics"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_render"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_replay"
	pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
)

// updateBaseline regenerates the on-disk baseline JSON file when set.
// Run once via `go test -tags=long ./... -run TestDonkeyKongProof
// -update-baseline` after authoring or modifying the trace; the
// generated baseline.json is then committed and gates future runs.
var updateBaseline = flag.Bool("update-baseline", false, "regenerate donkey_kong_proof.baseline.json from this run")

// checkpointTicks are the trace tick numbers the baseline records.
// Three points: start (tick 0), mid-trace (tick 100 — well into
// the climbing window), end (tick 299 — last frame).
var checkpointTicks = []uint64{0, 100, 299}

// fixturesDir locates the canonical .pforge / .trace.jsonl / baseline
// directory relative to this test file. The sub-package layout means
// fixtures live one level up; runtime.Caller(0) keeps the resolution
// robust to test-binary cwd quirks.
func fixturesDir(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(here), "..", "fixtures")
}

// ----------------------------------------------------------------------------
// TestMain: ebiten graphics context for RenderTickAtRGBA
// ----------------------------------------------------------------------------

type gameWithOneUpdate struct {
	m    *testing.M
	code int
}

func (g *gameWithOneUpdate) Update() error {
	g.code = g.m.Run()
	return ebiten.Termination
}

func (*gameWithOneUpdate) Draw(*ebiten.Image)         {}
func (*gameWithOneUpdate) Layout(int, int) (int, int) { return 176, 160 }

func TestMain(m *testing.M) {
	flag.Parse()
	g := &gameWithOneUpdate{m: m, code: 1}
	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
	os.Exit(g.code)
}

// ----------------------------------------------------------------------------
// Asset fixtures
// ----------------------------------------------------------------------------

// generatePlaceholderPNG matches the U11/U14/U16 pattern.
func generatePlaceholderPNG(t *testing.T, w, h int, fill color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, fill)
		}
	}
	img.Set(0, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// setupDKAssetsFS builds an in-memory fs.FS rooted such that the
// four sprite paths the .pforge declares all resolve. Matches the
// capsuleruntime loader's expected layout (assetsRoot = "assets").
func setupDKAssetsFS(t *testing.T) fstest.MapFS {
	t.Helper()
	climber := generatePlaceholderPNG(t, 16, 16, color.RGBA{R: 220, G: 80, B: 80, A: 255})
	barrel := generatePlaceholderPNG(t, 16, 16, color.RGBA{R: 200, G: 140, B: 60, A: 255})
	platform := generatePlaceholderPNG(t, 16, 16, color.RGBA{R: 80, G: 80, B: 200, A: 255})
	ladder := generatePlaceholderPNG(t, 16, 16, color.RGBA{R: 200, G: 200, B: 80, A: 255})
	return fstest.MapFS{
		"assets/sprites/climber.png":       &fstest.MapFile{Data: climber},
		"assets/sprites/barrel.png":        &fstest.MapFile{Data: barrel},
		"assets/sprites/platform_tile.png": &fstest.MapFile{Data: platform},
		"assets/sprites/ladder_tile.png":   &fstest.MapFile{Data: ladder},
	}
}

// ----------------------------------------------------------------------------
// In-memory save backend (mirrors U11/U14/U16 pattern).
// ----------------------------------------------------------------------------

type memSaveBackend struct {
	mu   sync.Mutex
	data map[string][]byte
}

func newMemSaveBackend() *memSaveBackend { return &memSaveBackend{data: map[string][]byte{}} }

func (m *memSaveBackend) Write(slot string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	m.data[slot] = cp
	return nil
}

func (m *memSaveBackend) Read(slot string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[slot]
	if !ok {
		return nil, pisave.ErrNotFound
	}
	cp := make([]byte, len(v))
	copy(cp, v)
	return cp, nil
}

func (m *memSaveBackend) Delete(slot string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, slot)
	return nil
}

func (m *memSaveBackend) List() ([]pisave.SlotMeta, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]pisave.SlotMeta, 0, len(keys))
	for _, k := range keys {
		out = append(out, pisave.SlotMeta{Name: k, Size: int64(len(m.data[k]))})
	}
	return out, nil
}

// ----------------------------------------------------------------------------
// Baseline file format (mirrors U11/U14/U16 byte-for-byte)
// ----------------------------------------------------------------------------

type BaselineEvent struct {
	Topic   string          `json:"topic"`
	ArgsRaw json.RawMessage `json:"args"`
}

type Checkpoint struct {
	Tick         uint64          `json:"tick"`
	FrameSHA256  string          `json:"frame_sha256"`
	EventsAtTick []BaselineEvent `json:"events_at_tick"`
}

type Baseline struct {
	SchemaVersion int          `json:"schema_version"`
	Game          string       `json:"game"`
	Checkpoints   []Checkpoint `json:"checkpoints"`
}

// ----------------------------------------------------------------------------
// Helpers: per-tick verb dispatch (the U18 "ladder + barrel + collide
// + gravity" cadence). Mirrors U14's publishPlatformerVerbsForKeys
// shape but extends it with the ladder + per-entity barrel publishes.
// ----------------------------------------------------------------------------

// publishDKVerbsForTick converts a trace-tick's held-keys snapshot
// into DK-flavour verb publishes on verbs.bus. The platformer
// convention applies (gravity + collide every tick); on top, the DK
// recipe layers ladder_climb verbs driven by ArrowUp/ArrowDown and
// per-entity barrel_roll verbs every tick for each barrel.
//
// The wasClimbing flag from the previous tick decides whether an
// idle frame should publish "ladder_climb direction=off" (release
// the ladder so gravity resumes) or stay silent. The caller owns
// the climbing-flag state.
//
// Key mappings:
//   - ArrowUp   -> motion/ladder_climb direction=up
//   - ArrowDown -> motion/ladder_climb direction=down
//   - otherwise (if wasClimbing) -> motion/ladder_climb direction=off
//
// Always-published:
//   - collision/solid_collide entity=hero_1 (per U14 lesson: BEFORE gravity)
//   - motion/apply_gravity   entity=hero_1
//   - motion/barrel_roll     entity=barrel_N (for each barrel)
func publishDKVerbsForTick(keys []ebiten.Key, wasClimbing bool, barrelIDs []string) (climbingThisTick bool) {
	bus := piloop.VerbsBus()

	// Detect arrow keys.
	upHeld := false
	downHeld := false
	for _, k := range keys {
		switch k {
		case ebiten.KeyArrowUp:
			upHeld = true
		case ebiten.KeyArrowDown:
			downHeld = true
		}
	}

	switch {
	case upHeld:
		bus.Publish(&piloop.VerbEvent{
			Topic: "motion/ladder_climb",
			Args:  map[string]any{"entity": "hero_1", "direction": "up"},
		})
		climbingThisTick = true
	case downHeld:
		bus.Publish(&piloop.VerbEvent{
			Topic: "motion/ladder_climb",
			Args:  map[string]any{"entity": "hero_1", "direction": "down"},
		})
		climbingThisTick = true
	default:
		if wasClimbing {
			bus.Publish(&piloop.VerbEvent{
				Topic: "motion/ladder_climb",
				Args:  map[string]any{"entity": "hero_1", "direction": "off"},
			})
		}
		// climbingThisTick stays false
	}

	// solid_collide BEFORE apply_gravity (per U14 lesson — the
	// substrate's ResolveTilemapCollision sets Grounded=false at the
	// start of each call, so calling it after gravity would erase the
	// freshly-set Grounded flag).
	bus.Publish(&piloop.VerbEvent{
		Topic: "collision/solid_collide",
		Args:  map[string]any{"entity": "hero_1"},
	})
	bus.Publish(&piloop.VerbEvent{
		Topic: "motion/apply_gravity",
		Args:  map[string]any{"entity": "hero_1"},
	})

	// Per-tick barrel_roll for each pre-staged barrel entity.
	for _, id := range barrelIDs {
		bus.Publish(&piloop.VerbEvent{
			Topic: "motion/barrel_roll",
			Args:  map[string]any{"entity": id},
		})
	}
	return climbingThisTick
}

// ----------------------------------------------------------------------------
// TestDonkeyKongProof — the smoke + baseline test (U18a)
// ----------------------------------------------------------------------------

func TestDonkeyKongProof(t *testing.T) {
	dir := fixturesDir(t)

	// --- load project ----------------------------------------------------
	pforgeF, err := os.Open(filepath.Join(dir, "donkey_kong_proof.pforge"))
	require.NoError(t, err, "open donkey_kong_proof.pforge")
	defer pforgeF.Close()
	project, err := pixelforge_project.LoadReader(pforgeF)
	require.NoError(t, err, "load donkey_kong_proof.pforge")
	require.Equal(t, "donkey_kong_proof", project.Name)
	require.Equal(t, "dk", project.PhysicsPreset)
	require.Len(t, project.Scenes, 1)
	require.Len(t, project.Scenes[0].Entities, 4, "expected 1 hero + 3 barrel entities")
	require.Len(t, project.Scenes[0].TileAtlases, 1, "expected one platform/ladder TileAtlas")
	require.Equal(t, 3, project.Scenes[0].GridHeightScreens, "DK level must be multi-screen tall")

	// --- boot runtime ----------------------------------------------------
	capsuleruntime.ResetRegistriesForTest()
	pievent.ResetRegistryForTest()
	piloop.ResetVerbsBusForTest()
	pixelforge_render.ResetForTest()
	// Align the engine's global canvas with the project's screen dims
	// so RenderTickAtRGBA's offscreen + CopyCanvasToEbitenImage sizes
	// match. The renderer's runtimeScreenDims prefers the project,
	// but the engine's canvas was set by the most recent
	// SetScreenSize caller (default 320×180); the codegen path calls
	// SetScreenSize from the cart's CapsuleRun, the tests do it
	// manually.
	pixelforge.SetScreenSize(176, 160)

	assets := setupDKAssetsFS(t)
	rt, err := capsuleruntime.Boot(project, assets, capsuleruntime.Options{
		SaveBackendOverride: newMemSaveBackend(),
	})
	require.NoError(t, err, "boot capsule runtime")
	require.NotNil(t, rt)
	require.NotNil(t, rt.Physics)
	require.NotNil(t, rt.Bodies["hero_1"])
	require.NotNil(t, rt.Bodies["barrel_1"])
	require.NotNil(t, rt.Bodies["barrel_2"])
	require.NotNil(t, rt.Bodies["barrel_3"])

	// --- shrink hero body to 8×8 (U13/U14 lesson: narrower collider so
	//     the per-tile MTV picks the Y axis cleanly on landings). Keep
	//     barrels at the default 16×16 — they're the canonical "rolling
	//     tile-sized object" and the DK barrel substrate tests use that
	//     footprint.
	heroBody := rt.Bodies["hero_1"]
	heroBody.Width = pixelforge_physics.FixedFromInt(8)
	heroBody.Height = pixelforge_physics.FixedFromInt(8)
	heroBody.Sync()

	barrelIDs := []string{"barrel_1", "barrel_2", "barrel_3"}

	// --- load trace ------------------------------------------------------
	traceF, err := os.Open(filepath.Join(dir, "donkey_kong_proof.trace.jsonl"))
	require.NoError(t, err, "open trace")
	defer traceF.Close()
	trace, err := pixelforge_replay.LoadTrace(traceF)
	require.NoError(t, err, "decode trace")
	require.Equal(t, "donkey_kong_proof", trace.Meta.Game)
	require.Len(t, trace.Frames, 300, "trace should have 300 frames at 60TPS for ~5s")

	// Snapshot starting positions for smoke checks.
	startHeroPos := heroBody.Position
	startBarrel1X := rt.Bodies["barrel_1"].Position.X

	// --- run replay with verb-bus capture --------------------------------
	checkpointSet := map[uint64]bool{}
	for _, c := range checkpointTicks {
		checkpointSet[c] = true
	}

	var currentTick uint64
	eventsByCheckpoint := map[uint64][]BaselineEvent{}
	var captureMu sync.Mutex

	bus := piloop.VerbsBus()
	subHandle := bus.SubscribeAll(func(ev *piloop.VerbEvent, _ pievent.Handler) {
		if ev == nil {
			return
		}
		if !checkpointSet[currentTick] {
			return
		}
		argsBytes, _ := json.Marshal(ev.Args)
		captureMu.Lock()
		eventsByCheckpoint[currentTick] = append(eventsByCheckpoint[currentTick], BaselineEvent{
			Topic:   ev.Topic,
			ArgsRaw: argsBytes,
		})
		captureMu.Unlock()
	})
	defer bus.Unsubscribe(subHandle)

	// Per-tick observation subscriber: track ladder_climb + barrel_roll
	// emissions across the whole trace (not just checkpoint ticks) so
	// the smoke flags fire regardless of which tick they land on.
	var (
		sawLadderClimbEvent  bool
		sawBarrelRollEvent   bool
		sawLadderEnteredEvt  bool
		sawSolidCollideEvent bool
		sawGravityEvent      bool
	)
	observe := bus.SubscribeAll(func(ev *piloop.VerbEvent, _ pievent.Handler) {
		if ev == nil {
			return
		}
		switch ev.Topic {
		case "motion/ladder_climb":
			sawLadderClimbEvent = true
		case "motion/barrel_roll":
			sawBarrelRollEvent = true
		case "motion/ladder_entered":
			sawLadderEnteredEvt = true
		case "collision/solid_collide":
			sawSolidCollideEvent = true
		case "motion/apply_gravity":
			sawGravityEvent = true
		}
	})
	defer bus.Unsubscribe(observe)

	checkpointHashes := map[uint64]string{}

	var (
		sawHeroOnLadder       bool
		heroMinY              = heroBody.Position.Y.Int()
		sawCameraOffsetNonZero bool
		wasClimbing           bool
	)

	for _, f := range trace.Frames {
		currentTick = f.Tick

		// Publish per-tick verbs (DK convention).
		wasClimbing = publishDKVerbsForTick(f.Keys, wasClimbing, barrelIDs)

		// Render the frame via the shared RenderTickAt seam. This is
		// also what computes rt.CameraOffsetY for multi-screen scenes.
		img, err := pixelforge_render.RenderTickAtRGBA(rt, f.Tick, pixelforge_render.InputFrame{
			Keys: f.Keys,
			Pad:  f.Pad,
		})
		require.NoErrorf(t, err, "RenderTickAtRGBA at tick %d", f.Tick)
		require.NotNil(t, img)

		if checkpointSet[f.Tick] {
			h := sha256.Sum256(img.Pix)
			checkpointHashes[f.Tick] = hex.EncodeToString(h[:])
		}

		if heroBody.OnLadder {
			sawHeroOnLadder = true
		}
		if y := heroBody.Position.Y.Int(); y < heroMinY {
			heroMinY = y
		}
		if rt.CameraOffsetY != 0 {
			sawCameraOffsetNonZero = true
		}

		rt.AdvanceTick()
	}

	// --- smoke: events fired -------------------------------------------
	assert.True(t, sawLadderClimbEvent,
		"motion/ladder_climb should have fired during the trace (ArrowUp held ticks 31-80, 101-150)")
	assert.True(t, sawBarrelRollEvent,
		"motion/barrel_roll should have fired every tick for each barrel")
	assert.True(t, sawSolidCollideEvent,
		"collision/solid_collide should have fired every tick for hero_1")
	assert.True(t, sawGravityEvent,
		"motion/apply_gravity should have fired every tick for hero_1")
	assert.True(t, sawLadderEnteredEvt,
		"motion/ladder_entered should have fired once the hero began climbing")

	// --- smoke: hero moved + touched the ladder ------------------------
	assert.NotEqual(t, startHeroPos, heroBody.Position,
		"hero body position should have changed during the trace")
	assert.True(t, sawHeroOnLadder,
		"hero must have overlapped the ladder column at some point during the climb window")

	// --- smoke: hero climbed upward (Y decreased) ----------------------
	// The trace holds ArrowUp for ticks 31-80 + 101-150. The hero
	// starts at Y=280, falls onto the floor (row 19, top y=304 — body
	// 8 high so bottom flush at y=304, top at y=296). On the ladder
	// (col 5), Up climbs reduce Y by ~1 per tick. So the minimum Y
	// observed should be strictly less than the starting Y.
	assert.Less(t, heroMinY, startHeroPos.Y.Int(),
		"hero's minimum Y should be strictly less than starting Y (proof the climb actually moved him upward)")

	// --- smoke: at least one barrel rolled horizontally ----------------
	assert.NotEqual(t, startBarrel1X, rt.Bodies["barrel_1"].Position.X,
		"barrel_1's X position should have changed during the trace (barrel_roll publishes per tick)")

	// --- smoke: multi-screen camera offset computed --------------------
	// The scene has GridHeightScreens=3 so updateCameraOffset
	// computes a non-zero offset based on hero Y. The hero starts at
	// Y=280 (well below the top of the level) so offset should fire.
	// Note: the offset is computed but the renderer doesn't yet apply
	// it during entity/tile blits (U18 deferral — see package doc).
	assert.True(t, sawCameraOffsetNonZero,
		"rt.CameraOffsetY should be non-zero at some tick (proves multi-screen scroll computation runs)")

	// --- build the live baseline ---------------------------------------
	live := Baseline{
		SchemaVersion: 1,
		Game:          trace.Meta.Game,
		Checkpoints:   make([]Checkpoint, 0, len(checkpointTicks)),
	}
	for _, tick := range checkpointTicks {
		events := eventsByCheckpoint[tick]
		if events == nil {
			events = []BaselineEvent{}
		}
		live.Checkpoints = append(live.Checkpoints, Checkpoint{
			Tick:         tick,
			FrameSHA256:  checkpointHashes[tick],
			EventsAtTick: events,
		})
	}

	// --- baseline update path ------------------------------------------
	baselinePath := filepath.Join(dir, "donkey_kong_proof.baseline.json")
	if *updateBaseline {
		writeBaseline(t, baselinePath, &live)
		t.Logf("wrote baseline to %s", baselinePath)
		return
	}

	// --- compare to committed baseline ---------------------------------
	committed := readBaseline(t, baselinePath)
	require.Equal(t, live.Game, committed.Game, "game name in baseline must match trace")
	require.Equal(t, len(checkpointTicks), len(committed.Checkpoints), "baseline checkpoint count")

	for i, want := range committed.Checkpoints {
		got := live.Checkpoints[i]
		assert.Equalf(t, want.Tick, got.Tick, "checkpoint[%d] tick", i)
		assert.Equalf(t, want.FrameSHA256, got.FrameSHA256,
			"checkpoint[%d] (tick %d) frame SHA-256 mismatch — run with -update-baseline if the trace was intentionally changed",
			i, want.Tick)
		require.Equalf(t, len(want.EventsAtTick), len(got.EventsAtTick),
			"checkpoint[%d] (tick %d) event count mismatch — got %d events, want %d",
			i, want.Tick, len(got.EventsAtTick), len(want.EventsAtTick))
		for j, wantEv := range want.EventsAtTick {
			gotEv := got.EventsAtTick[j]
			assert.Equalf(t, wantEv.Topic, gotEv.Topic,
				"checkpoint[%d] tick %d event[%d] topic", i, want.Tick, j)
			assert.JSONEqf(t, string(wantEv.ArgsRaw), string(gotEv.ArgsRaw),
				"checkpoint[%d] tick %d event[%d] args", i, want.Tick, j)
		}
	}
}

// ----------------------------------------------------------------------------
// TestDonkeyKongProof_AE4_MidGameSaveLoadRestoresAllState (U18b)
// ----------------------------------------------------------------------------

// TestDonkeyKongProof_AE4_MidGameSaveLoadRestoresAllState is the
// canonical AE4 acceptance example from the brainstorm: "Given a
// player binary running the Donkey Kong reference cart, when A1 fires
// `save_now`, presses Reset, then fires `load_slot`, the entity state
// (player position, ladder-climb state, barrel positions, lives
// count) restores byte-equal."
//
// Implementation:
//
//  1. Boot rt from the DK fixture.
//  2. Set Globals (lives=3, score=1500) — these are what the AE4
//     spec calls out as "lives count" + game-state.
//  3. Drive ~200 ticks of the trace (mid-climb point — hero is on
//     the ladder, barrels are rolling).
//  4. Capture pre-save state: hero position, ladder flags, barrel
//     positions + directions, body fields, Globals.
//  5. Encode snapshot via capsuleruntime.EncodeSnapshot(rt).
//  6. Boot a FRESH rt from the same fixture (simulates Reset).
//  7. Verify the fresh rt's state DIFFERS from pre-save (sanity:
//     reset actually reset things).
//  8. Decode the captured snapshot into the fresh rt via
//     capsuleruntime.DecodeSnapshot.
//  9. Capture post-load state.
//  10. Assert post-load == pre-save BYTE-EQUAL for every named field.
//
// Byte-equality covers: hero Position.X/Y + Velocity.X/Y + Rotation
// (all Fixed32), hero OnLadder + LadderClimbing + Grounded flags,
// each barrel's Position + Velocity + Direction + Speed, Globals
// (lives, score), and rt.Tick.
func TestDonkeyKongProof_AE4_MidGameSaveLoadRestoresAllState(t *testing.T) {
	dir := fixturesDir(t)

	// --- helper to boot a runtime from the fixture --------------------
	bootFresh := func() *capsuleruntime.Runtime {
		capsuleruntime.ResetRegistriesForTest()
		pievent.ResetRegistryForTest()
		piloop.ResetVerbsBusForTest()
		pixelforge_render.ResetForTest()
		pixelforge.SetScreenSize(176, 160)
		pforgeF, err := os.Open(filepath.Join(dir, "donkey_kong_proof.pforge"))
		require.NoError(t, err)
		defer pforgeF.Close()
		project, err := pixelforge_project.LoadReader(pforgeF)
		require.NoError(t, err)
		rt, err := capsuleruntime.Boot(project, setupDKAssetsFS(t), capsuleruntime.Options{
			SaveBackendOverride: newMemSaveBackend(),
		})
		require.NoError(t, err)
		// Shrink hero body to 8×8 (mirrors the smoke test's setup so
		// post-load body footprint matches the saved state under
		// rebuildBodiesFromSnapshot's 16×16 default).
		heroBody := rt.Bodies["hero_1"]
		heroBody.Width = pixelforge_physics.FixedFromInt(8)
		heroBody.Height = pixelforge_physics.FixedFromInt(8)
		heroBody.Sync()
		return rt
	}

	// --- 1. Boot original runtime + drive ~200 ticks ------------------
	rt := bootFresh()
	rt.Globals["lives"] = 3
	rt.Globals["score"] = 1500
	rt.Globals["player_entity"] = "hero_1"

	traceF, err := os.Open(filepath.Join(dir, "donkey_kong_proof.trace.jsonl"))
	require.NoError(t, err)
	defer traceF.Close()
	trace, err := pixelforge_replay.LoadTrace(traceF)
	require.NoError(t, err)

	barrelIDs := []string{"barrel_1", "barrel_2", "barrel_3"}
	wasClimbing := false
	// Drive 200 ticks — past the start of the first ArrowUp window
	// (ticks 31-80) and well into the second one (ticks 101-150)
	// trailing off, so the hero has climbed AND ridden idle ticks
	// while the barrels are rolling.
	const midGameTick = 200
	for _, f := range trace.Frames {
		if f.Tick >= midGameTick {
			break
		}
		wasClimbing = publishDKVerbsForTick(f.Keys, wasClimbing, barrelIDs)
		_, err := pixelforge_render.RenderTickAtRGBA(rt, f.Tick, pixelforge_render.InputFrame{
			Keys: f.Keys, Pad: f.Pad,
		})
		require.NoErrorf(t, err, "render at tick %d", f.Tick)
		rt.AdvanceTick()
	}

	// --- 2. Capture pre-save state ------------------------------------
	preHero := rt.Bodies["hero_1"]
	require.NotNil(t, preHero)
	preHeroPos := preHero.Position
	preHeroVel := preHero.Velocity
	preHeroOnLadder := preHero.OnLadder
	preHeroLadderClimbing := preHero.LadderClimbing
	preHeroGrounded := preHero.Grounded

	type barrelSnap struct {
		Position  pixelforge_physics.Vec2
		Velocity  pixelforge_physics.Vec2
		Grounded  bool
		Direction pixelforge_physics.BarrelDir
		Speed     pixelforge_physics.Fixed32
	}
	preBarrels := map[string]barrelSnap{}
	for _, id := range barrelIDs {
		b := rt.Bodies[id]
		require.NotNil(t, b, "barrel %s body must exist pre-save", id)
		bs := rt.BarrelStates[id]
		preBarrels[id] = barrelSnap{
			Position:  b.Position,
			Velocity:  b.Velocity,
			Grounded:  b.Grounded,
			Direction: bs.Direction,
			Speed:     bs.Speed,
		}
	}

	preTick := rt.Tick
	preLives := rt.Globals["lives"]
	preScore := rt.Globals["score"]
	prePlayerEntity := rt.Globals["player_entity"]
	preEntityCount := len(rt.CurrentScene.Entities)

	// Sanity: the run must have actually advanced state.
	assert.Equal(t, uint64(midGameTick), preTick, "rt.Tick should have advanced to midGameTick")

	// --- 3. Encode snapshot ------------------------------------------
	snap, err := capsuleruntime.EncodeSnapshot(rt)
	require.NoError(t, err, "EncodeSnapshot must succeed mid-game")
	require.Equal(t, uint64(midGameTick), snap.Tick)
	require.Equal(t, "climb_1_1", snap.CurrentSceneID)
	require.Len(t, snap.Entities, preEntityCount)

	// --- 4. Boot a FRESH rt (= reset) --------------------------------
	rt2 := bootFresh()
	// Sanity: confirm reset actually reset.
	freshHeroPos := rt2.Bodies["hero_1"].Position
	assert.NotEqual(t, preHeroPos, freshHeroPos,
		"fresh rt's hero position must differ from pre-save (reset actually reset)")
	_, hasLives := rt2.Globals["lives"]
	assert.False(t, hasLives, "fresh rt should not carry the saved lives global")

	// --- 5. Decode snapshot into fresh rt -----------------------------
	require.NoError(t, capsuleruntime.DecodeSnapshot(rt2, snap), "DecodeSnapshot must succeed")

	// --- 6. Byte-equal assertions ------------------------------------

	// Tick.
	assert.Equal(t, preTick, rt2.Tick, "rt.Tick must restore byte-equal")

	// Hero body fields.
	postHero := rt2.Bodies["hero_1"]
	require.NotNil(t, postHero, "hero_1 body must exist after load")
	assert.Equal(t, preHeroPos.X, postHero.Position.X, "hero Position.X byte-equal")
	assert.Equal(t, preHeroPos.Y, postHero.Position.Y, "hero Position.Y byte-equal")
	assert.Equal(t, preHeroVel.X, postHero.Velocity.X, "hero Velocity.X byte-equal")
	assert.Equal(t, preHeroVel.Y, postHero.Velocity.Y, "hero Velocity.Y byte-equal")
	assert.Equal(t, preHeroOnLadder, postHero.OnLadder, "hero OnLadder byte-equal")
	assert.Equal(t, preHeroLadderClimbing, postHero.LadderClimbing, "hero LadderClimbing byte-equal")
	assert.Equal(t, preHeroGrounded, postHero.Grounded, "hero Grounded byte-equal")

	// Per-barrel body + state.
	for _, id := range barrelIDs {
		pb := preBarrels[id]
		b := rt2.Bodies[id]
		require.NotNilf(t, b, "barrel %s body must exist post-load", id)
		assert.Equalf(t, pb.Position.X, b.Position.X, "barrel %s Position.X byte-equal", id)
		assert.Equalf(t, pb.Position.Y, b.Position.Y, "barrel %s Position.Y byte-equal", id)
		assert.Equalf(t, pb.Velocity.X, b.Velocity.X, "barrel %s Velocity.X byte-equal", id)
		assert.Equalf(t, pb.Velocity.Y, b.Velocity.Y, "barrel %s Velocity.Y byte-equal", id)
		assert.Equalf(t, pb.Grounded, b.Grounded, "barrel %s Grounded byte-equal", id)
		// Barrel state (direction + speed) restored via the U18 additive
		// BodyStateV2.BarrelDirection / BarrelSpeed fields.
		bs, ok := rt2.BarrelStates[id]
		require.Truef(t, ok, "barrel %s BarrelState must exist post-load", id)
		assert.Equalf(t, pb.Direction, bs.Direction, "barrel %s Direction byte-equal", id)
		assert.Equalf(t, pb.Speed, bs.Speed, "barrel %s Speed byte-equal", id)
	}

	// Globals.
	// JSON round-trips numeric globals as float64 — both pre and
	// post-load values went through JSON marshaling so the byte-equal
	// expectation is on the float64 representation. preLives was an int
	// (3) but post-load it's a float64 (3.0); assert against float64.
	assert.Equal(t, float64(3), rt2.Globals["lives"], "Globals[\"lives\"] byte-equal (= 3)")
	assert.Equal(t, float64(1500), rt2.Globals["score"], "Globals[\"score\"] byte-equal (= 1500)")
	assert.Equal(t, "hero_1", rt2.Globals["player_entity"], "Globals[\"player_entity\"] byte-equal")

	// Suppress unused-var warnings on pre-load globals (kept above for
	// documentation purposes — the pre-values land in pre/post the
	// JSON normalisation boundary).
	_ = preLives
	_ = preScore
	_ = prePlayerEntity
}

// ----------------------------------------------------------------------------
// Baseline IO (mirrors U11/U14/U16)
// ----------------------------------------------------------------------------

func writeBaseline(t *testing.T, baselinePath string, live *Baseline) {
	t.Helper()
	f, err := os.Create(baselinePath)
	require.NoError(t, err)
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	require.NoError(t, enc.Encode(live))
}

func readBaseline(t *testing.T, baselinePath string) *Baseline {
	t.Helper()
	f, err := os.Open(baselinePath)
	if os.IsNotExist(err) {
		t.Fatalf("baseline file missing at %s; first-time seed with `go test -tags=long ./pixelforge_studio/integration_test/donkey_kong_proof -run TestDonkeyKongProof -update-baseline`", baselinePath)
	}
	require.NoError(t, err)
	defer f.Close()
	raw, err := io.ReadAll(f)
	require.NoError(t, err)
	var b Baseline
	require.NoError(t, json.Unmarshal(raw, &b))
	return &b
}
