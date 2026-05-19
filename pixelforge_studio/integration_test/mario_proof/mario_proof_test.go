//go:build long

// Package mario_proof_test is plan-009 U14's end-to-end smoke test for
// the Mario platformer reference cart. It loads the .pforge fixture,
// generates placeholder sprite assets into an in-memory FS, boots the
// capsule runtime, drives the trace through the verb-recipe bus +
// physics integrator + RenderTickAtRGBA pipeline, captures pixel hashes
// at three checkpoint ticks (0, 100, 299) plus the per-checkpoint
// verbs.bus event sequence, and asserts parity against a committed
// baseline.
//
// # Scope (U14 smoke)
//
// This unit lands the FIRST end-to-end run of the platformer stack
// (Phase 2's terminal artifact) and is deliberately scoped pragmatically
// to mirror U11's smoke-level template: the trace exercises gravity,
// jump impulses, and tile-AABB ground collision through the motion
// sinks and proves the load -> boot -> physics -> render -> hash
// pipeline ties together. The trace does NOT clear level 1-1 because:
//
//   - The input intent layer (pixelforge_render.applyInputFrame) is
//     still a stash-only stub: InputFrame.Keys does not yet bridge to
//     verb publishes. The test wires Space -> motion/jump manually and
//     publishes motion/apply_gravity + collision/solid_collide per
//     tick (matching the platformer convention) so the verb-bus
//     pipeline gets exercised without waiting on the intent-layer
//     landing.
//
//   - Goomba stomp / patrol / death verbs and the level-clear flag
//     entity are NOT YET implemented (no patrol motion handler, no
//     stomp damage routing). Wiring those is a U14 polish iteration —
//     the smoke value here is "platformer physics + Mario fixture
//     replays deterministically through verbs.bus + RenderTickAtRGBA."
//
//   - Pixel-correctness of the rendered hero / goomba / ground sprites
//     is gated on the renderer learning to draw Sprite components at
//     body.Position. For U14, the smoke covers the fact that frames
//     are produced (non-empty, deterministic) and that the verb-bus
//     event sequence at each checkpoint matches the recorded baseline.
//
// # Deferred to a U14 polish iteration
//
//   - Full level-clear: 3 goomba stomps + 1 pipe jump + flag touch.
//   - Goomba patrol motion verb (left/right reversal at platform edge).
//   - Stomp-as-damage routing (entity AABB-vs-AABB + damage/die).
//   - Pixel-correct sprite blit at body.Position (U12+ concern).
//   - Pixel-hash regression detector (AE5) under a deliberately-
//     introduced gravity off-by-one. The active detector lands when
//     the bullet/asteroid + goomba/stomp paths are observable on a
//     real bug.
//
// # File layout
//
// Mirrors U11's asteroids_proof layout: this sub-package lives at
// pixelforge_studio/integration_test/mario_proof/ so the Ebitengine
// TestMain wrapper (RenderTickAtRGBA needs a live graphics context for
// ebiten.NewImage + ReadPixels) is isolated from the other long-tag
// tests in the parent package. Fixtures live one level up at
// ../fixtures/mario_proof.{pforge,trace.jsonl,baseline.json}.
//
// # Body-size note (per U13's TestMotionSink platformer fixture)
//
// U13 documented that the default 16×16 body on 16×16 ground tiles
// produces a degenerate same-axis-overlap collision: the AABB sweeper's
// "pick smaller overlap axis" tiebreak picks X-resolution on a level-
// resting body, teleporting the hero sideways out of the floor instead
// of landing on it. The substrate's resolution rule has a vy/vx tie-
// break (see pixelforge_physics/tilemap.go line 137), but for a body
// equal in width to the floor tile the tiebreak still snaps the body
// sideways in some authored layouts. U14 shrinks the hero body to 8×8
// after Boot — same fix the U13 platformer tests apply — so the
// per-tile MTV picks the Y axis cleanly. Goomba bodies stay at 16×16
// (they don't participate in collision in this trace).
package mario_proof_test

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
// Run once via `go test -tags=long ./... -run TestMarioProof
// -update-baseline` after authoring or modifying the trace; the
// generated baseline.json is then committed and gates future runs.
var updateBaseline = flag.Bool("update-baseline", false, "regenerate mario_proof.baseline.json from this run")

// checkpointTicks are the trace tick numbers the baseline records.
// Three points: start (tick 0), mid-trace (tick 100 — well past the
// first jump), end (tick 299 — last frame).
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

// gameWithOneUpdate mirrors the pattern in pixelforge_replay's test
// suite (and U11's asteroids_proof_test). Ebitengine requires a live
// graphics context for ebiten.NewImage + ReadPixels — both used by
// RenderTickAtRGBA. Hosting m.Run() inside one Update + ebiten.Termination
// is the standard workaround.
type gameWithOneUpdate struct {
	m    *testing.M
	code int
}

func (g *gameWithOneUpdate) Update() error {
	g.code = g.m.Run()
	return ebiten.Termination
}

func (*gameWithOneUpdate) Draw(*ebiten.Image)         {}
func (*gameWithOneUpdate) Layout(int, int) (int, int) { return 320, 180 }

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

// generatePlaceholderPNG renders a tiny solid-color RGBA PNG with a
// distinctive corner pixel so two different sprites have non-zero
// content variation. width/height/color are read off the project's
// SpriteAsset declarations so what the loader expects matches what
// we generate.
func generatePlaceholderPNG(t *testing.T, w, h int, fill color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, fill)
		}
	}
	// Distinctive corner pixel — useful for visual diff if a future
	// pass adds a sprite-onto-screen renderer.
	img.Set(0, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// setupMarioAssetsFS builds an in-memory fs.FS rooted such that the
// three sprite paths the .pforge declares all resolve. Matches the
// capsuleruntime loader's expected layout (assetsRoot = "assets").
func setupMarioAssetsFS(t *testing.T) fstest.MapFS {
	t.Helper()
	hero := generatePlaceholderPNG(t, 16, 16, color.RGBA{R: 200, G: 80, B: 60, A: 255})
	goomba := generatePlaceholderPNG(t, 16, 16, color.RGBA{R: 120, G: 70, B: 40, A: 255})
	ground := generatePlaceholderPNG(t, 16, 16, color.RGBA{R: 90, G: 60, B: 30, A: 255})
	return fstest.MapFS{
		"assets/sprites/hero.png":        &fstest.MapFile{Data: hero},
		"assets/sprites/goomba.png":      &fstest.MapFile{Data: goomba},
		"assets/sprites/ground_tile.png": &fstest.MapFile{Data: ground},
	}
}

// ----------------------------------------------------------------------------
// In-memory save backend (Boot needs SOMETHING for the save service;
// the production native backend touches the user's home directory and
// pollutes long-running CI). Pattern mirrored from U11.
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
// Baseline file format
// ----------------------------------------------------------------------------

// BaselineEvent / Checkpoint / Baseline mirror U11's shapes byte-for-
// byte so the CI gate has one schema across all four games' baselines.
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
// Test main path
// ----------------------------------------------------------------------------

// TestMarioProof is U14's CI gate. Loads the fixture, replays the
// trace, captures checkpoint hashes + bus events, compares to the
// committed baseline. The `-update-baseline` flag regenerates the
// baseline file (first-run flow).
func TestMarioProof(t *testing.T) {
	dir := fixturesDir(t)

	// --- load project ----------------------------------------------------
	pforgeF, err := os.Open(filepath.Join(dir, "mario_proof.pforge"))
	require.NoError(t, err, "open mario_proof.pforge")
	defer pforgeF.Close()
	project, err := pixelforge_project.LoadReader(pforgeF)
	require.NoError(t, err, "load mario_proof.pforge")
	require.Equal(t, "mario_proof", project.Name)
	require.Equal(t, "mario", project.PhysicsPreset)
	require.Len(t, project.Scenes, 1)
	require.Len(t, project.Scenes[0].Entities, 3, "expected 1 hero + 2 goomba entities")
	require.Len(t, project.Scenes[0].TileAtlases, 1, "expected one ground TileAtlas")

	// --- boot runtime ----------------------------------------------------
	// Reset registries so prior tests in the same process don't leak
	// subscribers / sprite caches into this Boot.
	capsuleruntime.ResetRegistriesForTest()
	pievent.ResetRegistryForTest()
	piloop.ResetVerbsBusForTest()
	pixelforge_render.ResetForTest()

	assets := setupMarioAssetsFS(t)
	rt, err := capsuleruntime.Boot(project, assets, capsuleruntime.Options{
		SaveBackendOverride: newMemSaveBackend(),
	})
	require.NoError(t, err, "boot capsule runtime")
	require.NotNil(t, rt)
	require.NotNil(t, rt.Physics)
	require.NotNil(t, rt.Bodies["hero_1"])
	require.NotNil(t, rt.Bodies["goomba_1"])

	// --- shrink hero body to 8×8 ----------------------------------------
	// See package doc / U14 body-size note: the default 16×16 body on
	// a 16×16 ground tile drives a degenerate "both axes overlap
	// equally" MTV. 8×8 mirrors the canonical SMB hero collider
	// footprint (narrower than the 16-px ground tile) and matches the
	// substrate's tilemap_test fixtures + U13's platformer test
	// fixtures. A future polish iteration lifts this into Project so
	// the per-entity body footprint is authored, not test-stamped.
	heroBody := rt.Bodies["hero_1"]
	heroBody.Width = pixelforge_physics.FixedFromInt(8)
	heroBody.Height = pixelforge_physics.FixedFromInt(8)
	heroBody.Sync()

	// --- load trace ------------------------------------------------------
	traceF, err := os.Open(filepath.Join(dir, "mario_proof.trace.jsonl"))
	require.NoError(t, err, "open trace")
	defer traceF.Close()
	trace, err := pixelforge_replay.LoadTrace(traceF)
	require.NoError(t, err, "decode trace")
	require.Equal(t, "mario_proof", trace.Meta.Game)
	require.Len(t, trace.Frames, 300, "trace should have 300 frames at 60TPS for ~5s")

	// --- snapshot starting hero position for smoke check ----------------
	startPos := heroBody.Position

	// --- run replay with verb-bus capture --------------------------------
	checkpointSet := map[uint64]bool{}
	for _, c := range checkpointTicks {
		checkpointSet[c] = true
	}

	// per-tick event buffer + global tick counter (closures consult this
	// rather than the loop variable so the subscriber sees the right
	// tick number).
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

	checkpointHashes := map[uint64]string{}
	var bodyMovedDuringRun bool
	var sawJumpEvent bool
	var sawGravityEvent bool
	var sawSolidCollideEvent bool
	var heroLeftGround bool
	var heroMinY = heroBody.Position.Y.Int()

	// per-tick observation subscriber: drives the three flags above off
	// the live bus rather than relying on checkpoint-only capture
	// (jump/gravity may fire on every tick but we still want to assert
	// they fired regardless of which tick the assertion runs at).
	observe := bus.SubscribeAll(func(ev *piloop.VerbEvent, _ pievent.Handler) {
		if ev == nil {
			return
		}
		switch ev.Topic {
		case "motion/jump":
			sawJumpEvent = true
		case "motion/apply_gravity":
			sawGravityEvent = true
		case "collision/solid_collide":
			sawSolidCollideEvent = true
		}
	})
	defer bus.Unsubscribe(observe)

	for _, f := range trace.Frames {
		currentTick = f.Tick

		// Drive platformer verbs from the trace's keys. The platformer
		// convention is "publish apply_gravity + solid_collide every
		// tick; publish jump on Space-down". Until the input intent
		// layer publishes verbs from InputFrame (U5 follow-up), the
		// test fills in the bridge explicitly.
		publishPlatformerVerbsForKeys(f.Keys)

		// Render the frame via the shared RenderTickAt seam.
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

		if !bodyMovedDuringRun && heroBody.Position != startPos {
			bodyMovedDuringRun = true
		}
		if !heroBody.Grounded {
			heroLeftGround = true
		}
		if y := heroBody.Position.Y.Int(); y < heroMinY {
			heroMinY = y
		}

		rt.AdvanceTick()
	}

	// --- smoke: hero moved (gravity + collision actually advanced) ------
	assert.True(t, bodyMovedDuringRun,
		"hero body Position should have changed during the trace (gravity + tile-AABB resolution)")

	// --- smoke: hero ended grounded on the floor -----------------------
	// Final position must be at the floor (body top = tile-top - body
	// height = (9*16) - 8 = 136) AND Grounded must be true; the last
	// ~50 ticks of the trace are idle so gravity has had time to land
	// the body even from the peak of the last jump.
	assert.True(t, heroBody.Grounded,
		"hero should be Grounded by end of trace (last 50 ticks are idle, gravity + collision should land him)")
	assert.Equal(t, 136, heroBody.Position.Y.Int(),
		"hero should rest with bottom edge (y=144) on the floor tile-top — body height 8 + body top y=136")

	// --- smoke: the three platformer verbs fired during the trace ------
	assert.True(t, sawJumpEvent, "motion/jump should have fired during the trace (Space tick 60-69 / 150-159 / 240-249)")
	assert.True(t, sawGravityEvent, "motion/apply_gravity should have fired every tick")
	assert.True(t, sawSolidCollideEvent, "collision/solid_collide should have fired every tick")

	// --- smoke: hero actually got airborne (jump impulse > gravity) ----
	assert.True(t, heroLeftGround,
		"hero should have left the ground during the trace (Space-triggered jumps push vy negative; gravity restores)")
	// hero's resting top-edge is y=136; a real jump pushes him strictly
	// above that. Allow some slack because Fixed32 + integer-truncation
	// in tilemap resolution sometimes leaves the apex one pixel below
	// the theoretical peak — the load-bearing assertion is "min Y was
	// less than rest Y", proving the jump impulse actually integrated.
	assert.Less(t, heroMinY, 136,
		"hero's minimum Y should be strictly less than the floor-resting Y (proof the jump impulse moved him airborne)")

	// --- build the live baseline -----------------------------------------
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

	// --- baseline update path --------------------------------------------
	baselinePath := filepath.Join(dir, "mario_proof.baseline.json")
	if *updateBaseline {
		writeBaseline(t, baselinePath, &live)
		t.Logf("wrote baseline to %s", baselinePath)
		return
	}

	// --- compare to committed baseline -----------------------------------
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

// publishPlatformerVerbsForKeys converts a trace-tick's held-keys
// snapshot into platformer-verb publishes on verbs.bus, plus the per-
// tick gravity + solid_collide events the platformer convention
// expects to fire every frame. This is the manual bridge until
// applyInputFrame in pixelforge_render bridges keys -> verbs (a U5
// follow-up — see package doc comment).
//
// Key mappings (Mario-classic, U14 minimal subset):
//   - Space -> motion/jump strength=30 (the substrate gates on
//     Body.Grounded so spamming the verb mid-air is a no-op; a
//     repeat-press while still airborne does nothing).
//
// Always-published:
//   - motion/apply_gravity entity=hero_1 (every tick — the integrator
//     accumulates Gravity * dt and then ResolveTilemapCollision snaps
//     a grounded body back to tile-top + zeroes vy).
//   - collision/solid_collide entity=hero_1 (every tick — paranoid
//     redundant resolution path; the apply_gravity handler also
//     resolves but the explicit verb is what U13 ships as the
//     recipe-author escape hatch).
func publishPlatformerVerbsForKeys(keys []ebiten.Key) {
	bus := piloop.VerbsBus()
	for _, k := range keys {
		if k == ebiten.KeySpace {
			// strength=30 in Fixed32 px/tick is an initial upward
			// velocity ~2x one tick's worth of gravity
			// (980/60 ≈ 16.33 per tick). The hero leaves the ground
			// for a few ticks and lands within ~4 ticks — short
			// enough that the body's fall trajectory stays within
			// the 11-row tilemap and the floor-row resolution
			// catches the landing. Larger strengths fall through
			// the floor because the AABB sweeper only catches
			// position-level overlaps, not swept trajectories;
			// continuous-collision is a U14 polish iteration.
			// The substrate gates on Body.Grounded so a Space-held
			// trace tick only triggers a fresh jump on the first
			// frame after landing.
			bus.Publish(&piloop.VerbEvent{
				Topic: "motion/jump",
				Args:  map[string]any{"entity": "hero_1", "strength": float64(30)},
			})
		}
	}
	// Order matters: solid_collide first (no-op while airborne — it
	// resolves any pre-existing overlap and may clear Grounded), then
	// apply_gravity (the integrator-then-resolver step that owns the
	// canonical landing path + Grounded flag). Reversed order
	// (gravity-then-solid_collide) would have solid_collide reset
	// Grounded on a body that has just landed with bottom edge flush
	// on tile-top but no overlap — ResolveTilemapCollision sets
	// Grounded=false at the start of every call and only re-sets it
	// when a Y-upward resolution fires.
	bus.Publish(&piloop.VerbEvent{
		Topic: "collision/solid_collide",
		Args:  map[string]any{"entity": "hero_1"},
	})
	bus.Publish(&piloop.VerbEvent{
		Topic: "motion/apply_gravity",
		Args:  map[string]any{"entity": "hero_1"},
	})
}

// writeBaseline serialises live to baselinePath in stable JSON form
// (indented, key-sorted by encoding/json). The on-disk file is what
// CI re-reads on the next run.
func writeBaseline(t *testing.T, baselinePath string, live *Baseline) {
	t.Helper()
	f, err := os.Create(baselinePath)
	require.NoError(t, err)
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	require.NoError(t, enc.Encode(live))
}

// readBaseline loads the committed baseline.json. Missing-file is
// surfaced as a t.Fatalf with a hint about -update-baseline so a
// first-time runner knows how to seed.
func readBaseline(t *testing.T, baselinePath string) *Baseline {
	t.Helper()
	f, err := os.Open(baselinePath)
	if os.IsNotExist(err) {
		t.Fatalf("baseline file missing at %s; first-time seed with `go test -tags=long ./pixelforge_studio/integration_test/mario_proof -run TestMarioProof -update-baseline`", baselinePath)
	}
	require.NoError(t, err)
	defer f.Close()
	raw, err := io.ReadAll(f)
	require.NoError(t, err)
	var b Baseline
	require.NoError(t, json.Unmarshal(raw, &b))
	return &b
}

// ----------------------------------------------------------------------------
// AE5 — pixel-hash regression detector (deferred to U14 polish)
// ----------------------------------------------------------------------------

// TestMarioProof_AE5_RegressionDetector documents the U14 polish
// follow-up: once the goomba-stomp + flag-touch + level-clear logic
// lands, a deliberate off-by-one in pixelforge_physics's gravity
// integrator (e.g. bump MarioConfig().Gravity.Y by 1 unit, or change
// the Fixed32 shift by 1) will move the hero's tick-100 / tick-299
// position by a measurable amount and this proof test's frame hashes
// + bus event arg payloads will diverge from baseline. The active
// harness — temporarily mutate a Fixed32 constant, assert
// TestMarioProof fails at the expected checkpoint, restore — lands
// alongside the level-clear physics that make the bug observable.
func TestMarioProof_AE5_RegressionDetector(t *testing.T) {
	t.Skip("AE5 deferred to U14 polish iteration — needs goomba-stomp + flag-touch + bug-injection harness. See package doc.")
}
