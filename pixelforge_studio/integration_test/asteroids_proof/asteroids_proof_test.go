//go:build long

// Package asteroids_proof_test is plan-009 U11's end-to-end smoke test
// for the Asteroids reference cart. It loads the .pforge fixture,
// generates placeholder sprite assets into an in-memory FS, boots the
// capsule runtime, drives the trace through the verb-recipe bus +
// physics integrator + RenderTickAtRGBA pipeline, captures pixel hashes
// at three checkpoint ticks (0, 100, 299) plus the per-checkpoint
// verbs.bus event sequence, and asserts parity against a committed
// baseline.
//
// # Scope (U11 smoke)
//
// This unit lands the FIRST end-to-end run of U1-U10 stack and is
// deliberately scoped pragmatically: the trace exercises ship rotation
// + thrust + screen-wrap through the motion sinks and proves the
// load -> boot -> physics -> render -> hash pipeline ties together. The
// trace does NOT end with all asteroids destroyed because:
//
//   - The input intent layer (pixelforge_render.applyInputFrame) is
//     still a stash-only stub: InputFrame.Keys does not yet bridge to
//     verb publishes. The test wires keys -> motion verbs manually
//     so the verb-bus pipeline gets exercised without waiting on the
//     intent-layer landing.
//
//   - Bullet-asteroid collision, asteroid-split-on-hit, ship-death-
//     on-collision are NOT YET implemented by the capsule's motion +
//     damage sinks. Wiring those into pixelforge_physics +
//     spawn/damage sinks is U11b scope (per plan-009 U11 execution
//     note "expect physics tuning iterations").
//
//   - Pixel-correctness of the rendered ship + asteroid sprites is
//     gated on the renderer learning to draw Sprite components at
//     body.Position — a U12+ concern. For U11, the smoke covers the
//     fact that frames are produced (non-empty, deterministic) and
//     that the verb-bus event sequence at each checkpoint matches
//     the recorded baseline.
//
// # Deferred to U11b follow-up
//
//   - Bullet-asteroid collision detection (collision/solid_collide
//     or a new collision sink for entity-vs-entity).
//   - Asteroid-split-on-hit (spawn 2 medium asteroids on large
//     destroyed; spawn 2 small on medium; vanish small).
//   - Ship-death-on-collision.
//   - Full 90s trace (5400 ticks) ending with all asteroids destroyed.
//   - Pixel-hash regression detector (manually-introduced physics
//     bug + assert this test fails at the right checkpoint). U11b
//     ships the bug-injection harness alongside the collision wiring.
//
// # File layout
//
//   - Sub-package (asteroids_proof/) rather than the parent
//     integration_test/ package: RenderTickAtRGBA requires a live
//     ebiten graphics context (ebiten.NewImage + ReadPixels), which
//     means TestMain must wrap test execution in ebiten.RunGame. The
//     other long-tag tests in integration_test/ (build pipeline,
//     credits) don't need a graphics context and would be needlessly
//     wrapped in RunGame's single Update tick. Keeping this test in
//     a dedicated sub-package isolates the TestMain wrapper without
//     constraining the parent package's other long-tag tests.
//
//   - Fixtures live one level up at
//     ../fixtures/asteroids_proof.pforge,
//     ../fixtures/asteroids_proof.trace.jsonl,
//     ../fixtures/asteroids_proof.baseline.json
//     per the U11 path contract in plan-009. The test resolves them
//     relative to runtime.Caller(0).
//
// # Regression detector (deferred, documented here for U11b)
//
// To verify CI catches a real physics regression, U11b's harness will:
//
//  1. Bump pixelforge_physics.AsteroidsConfig().Gravity.Y by 1 Fixed32
//     unit (or change the SinDeg256/CosDeg256 LUT entry at index 0).
//  2. Re-run this test under -tags=long.
//  3. Assert the test FAILS at the affected checkpoint (event-sequence
//     mismatch at tick 100 or 299, depending on which constant moved).
//  4. Restore the constant.
//
// AE5 (pixel-hash regression detector) is checked here as a t.Skip
// placeholder so plan-009 traceability stays intact; the active
// detector lands in U11b alongside the collision sinks that make the
// detector observable on a real bug.
package asteroids_proof_test

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
	"math"
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
// Run once via `go test -tags=long ./... -run TestAsteroidsProof
// -update-baseline` after authoring or modifying the trace; the
// generated baseline.json is then committed and gates future runs.
var updateBaseline = flag.Bool("update-baseline", false, "regenerate asteroids_proof.baseline.json from this run")

// checkpointTicks are the trace tick numbers the baseline records.
// Three points: start (tick 0), mid-trace (tick 100 — well into the
// thrust period), end (tick 299 — last frame).
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
// suite. Ebitengine requires a live graphics context for ebiten.NewImage
// + ReadPixels — both used by RenderTickAtRGBA. Hosting m.Run() inside
// one Update + ebiten.Termination is the standard workaround.
type gameWithOneUpdate struct {
	m    *testing.M
	code int
}

func (g *gameWithOneUpdate) Update() error {
	g.code = g.m.Run()
	return ebiten.Termination
}

func (*gameWithOneUpdate) Draw(*ebiten.Image)            {}
func (*gameWithOneUpdate) Layout(int, int) (int, int)    { return 320, 180 }

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

// setupAsteroidsAssetsFS builds an in-memory fs.FS rooted such that
// "assets/sprites/ship.png" and "assets/sprites/asteroid_large.png"
// resolve. Matches the capsuleruntime loader's expected layout
// (assetsRoot = "assets").
func setupAsteroidsAssetsFS(t *testing.T) fstest.MapFS {
	t.Helper()
	ship := generatePlaceholderPNG(t, 16, 16, color.RGBA{R: 200, G: 200, B: 255, A: 255})
	rock := generatePlaceholderPNG(t, 24, 24, color.RGBA{R: 120, G: 120, B: 120, A: 255})
	return fstest.MapFS{
		"assets/sprites/ship.png":           &fstest.MapFile{Data: ship},
		"assets/sprites/asteroid_large.png": &fstest.MapFile{Data: rock},
	}
}

// ----------------------------------------------------------------------------
// In-memory save backend (Boot needs SOMETHING for the save service;
// the production native backend touches the user's home directory and
// pollutes long-running CI). Pattern mirrored from
// capsuleruntime's memBackend.
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

// BaselineEvent is the canonicalised per-event record stored in the
// baseline.json file. Topic + sorted-arg-keys are byte-comparable
// across runs; the raw map[string]any from VerbEvent.Args goes through
// json.Marshal so reflection-driven ordering doesn't make the baseline
// hash flap between Go versions.
type BaselineEvent struct {
	Topic   string          `json:"topic"`
	ArgsRaw json.RawMessage `json:"args"`
}

// Checkpoint pairs a tick number with the frame's SHA-256 hash and
// the verbs.bus events that fired during that tick.
type Checkpoint struct {
	Tick         uint64          `json:"tick"`
	FrameSHA256  string          `json:"frame_sha256"`
	EventsAtTick []BaselineEvent `json:"events_at_tick"`
}

// Baseline is the on-disk shape baseline.json carries.
type Baseline struct {
	SchemaVersion int          `json:"schema_version"`
	Game          string       `json:"game"`
	Checkpoints   []Checkpoint `json:"checkpoints"`
}

// ----------------------------------------------------------------------------
// Test main path
// ----------------------------------------------------------------------------

// TestAsteroidsProof is U11's CI gate. Loads the fixture, replays the
// trace, captures checkpoint hashes + bus events, compares to the
// committed baseline. The `-update-baseline` flag regenerates the
// baseline file (first-run flow).
func TestAsteroidsProof(t *testing.T) {
	dir := fixturesDir(t)

	// --- load project ----------------------------------------------------
	pforgeF, err := os.Open(filepath.Join(dir, "asteroids_proof.pforge"))
	require.NoError(t, err, "open asteroids_proof.pforge")
	defer pforgeF.Close()
	project, err := pixelforge_project.LoadReader(pforgeF)
	require.NoError(t, err, "load asteroids_proof.pforge")
	require.Equal(t, "asteroids_proof", project.Name)
	require.Equal(t, "asteroids", project.PhysicsPreset)
	require.Len(t, project.Scenes, 1)
	require.Len(t, project.Scenes[0].Entities, 5, "expected 1 ship + 4 asteroid entities")

	// --- boot runtime ----------------------------------------------------
	// Reset registries so prior tests in the same process don't leak
	// subscribers / sprite caches into this Boot.
	capsuleruntime.ResetRegistriesForTest()
	pievent.ResetRegistryForTest()
	piloop.ResetVerbsBusForTest()
	pixelforge_render.ResetForTest()

	assets := setupAsteroidsAssetsFS(t)
	rt, err := capsuleruntime.Boot(project, assets, capsuleruntime.Options{
		SaveBackendOverride: newMemSaveBackend(),
	})
	require.NoError(t, err, "boot capsule runtime")
	require.NotNil(t, rt)
	require.NotNil(t, rt.Physics)
	require.NotNil(t, rt.Bodies["ship_1"])

	// --- load trace ------------------------------------------------------
	traceF, err := os.Open(filepath.Join(dir, "asteroids_proof.trace.jsonl"))
	require.NoError(t, err, "open trace")
	defer traceF.Close()
	trace, err := pixelforge_replay.LoadTrace(traceF)
	require.NoError(t, err, "decode trace")
	require.Equal(t, "asteroids_proof", trace.Meta.Game)
	require.Len(t, trace.Frames, 300, "trace should have 300 frames at 60TPS for ~5s")

	// --- snapshot starting ship position for smoke check ----------------
	shipBody := rt.Bodies["ship_1"]
	startPos := shipBody.Position

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

	dt := pixelforge_physics.FixedFromFloat(1.0 / 60.0)

	checkpointHashes := map[uint64]string{}
	var bodyMovedDuringRun bool

	for _, f := range trace.Frames {
		currentTick = f.Tick

		// Drive motion verbs from the trace's keys. Until the input
		// intent layer publishes verbs from InputFrame (U5 follow-up),
		// the test fills in the bridge explicitly.
		publishMotionVerbsForKeys(f.Keys, shipBody)

		// Step the physics integrator for the ship body so velocity
		// from apply_thrust integrates into position. Asteroids has
		// no gravity, so this is purely velocity -> position.
		pixelforge_physics.Integrate(shipBody, rt.Physics, dt)

		// Screen-wrap check (no-op when in bounds).
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/screen_wrap",
			Args:  map[string]any{"entity": "ship_1"},
		})

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

		if !bodyMovedDuringRun && shipBody.Position != startPos {
			bodyMovedDuringRun = true
		}

		rt.AdvanceTick()
	}

	// --- smoke: ship moved -----------------------------------------------
	assert.True(t, bodyMovedDuringRun, "ship body Position should have changed during the trace (thrust + integrate)")

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
	baselinePath := filepath.Join(dir, "asteroids_proof.baseline.json")
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

// publishMotionVerbsForKeys converts a trace-tick's held-keys snapshot
// into motion-verb publishes on verbs.bus. This is the manual bridge
// until applyInputFrame in pixelforge_render bridges keys -> verbs (a
// U5 follow-up — see package doc comment).
//
// Key mappings (Asteroids-classic):
//   - ArrowLeft  -> motion/rotate_entity delta=-3 (deg/tick)
//   - ArrowRight -> motion/rotate_entity delta=+3
//   - Space      -> motion/apply_thrust direction=<ship Rotation> force=0.5
func publishMotionVerbsForKeys(keys []ebiten.Key, ship *pixelforge_physics.Body) {
	bus := piloop.VerbsBus()
	for _, k := range keys {
		switch k {
		case ebiten.KeyArrowLeft:
			bus.Publish(&piloop.VerbEvent{
				Topic: "motion/rotate_entity",
				Args:  map[string]any{"entity": "ship_1", "delta": float64(-3)},
			})
		case ebiten.KeyArrowRight:
			bus.Publish(&piloop.VerbEvent{
				Topic: "motion/rotate_entity",
				Args:  map[string]any{"entity": "ship_1", "delta": float64(3)},
			})
		case ebiten.KeySpace:
			// Use the ship's CURRENT rotation (degrees) as the thrust
			// direction. Fold into [0, 360) so the verb's degree256
			// mapping stays stable when rotation drifts negative
			// across many rotate-left presses.
			dir := ship.Rotation.Float()
			dir = math.Mod(dir, 360)
			if dir < 0 {
				dir += 360
			}
			bus.Publish(&piloop.VerbEvent{
				Topic: "motion/apply_thrust",
				Args:  map[string]any{"entity": "ship_1", "direction": dir, "force": float64(0.5)},
			})
		}
	}
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
		t.Fatalf("baseline file missing at %s; first-time seed with `go test -tags=long ./pixelforge_studio/integration_test/asteroids_proof -run TestAsteroidsProof -update-baseline`", baselinePath)
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
// AE5 — pixel-hash regression detector (deferred to U11b)
// ----------------------------------------------------------------------------

// TestAsteroidsProof_AE5_RegressionDetector documents the U11b follow-up:
// once the bullet/asteroid collision + split logic lands, a deliberate
// off-by-one in pixelforge_physics.SinDeg256 (or in screenwrap.wrapAxis)
// will move the ship's tick-100 / tick-299 position by a measurable
// amount and this proof test's frame hashes + bus event arg payloads
// will diverge from baseline. The active harness — temporarily mutate
// a Fixed32 constant, assert TestAsteroidsProof fails at the expected
// checkpoint, restore — lands in U11b alongside the collision sinks
// that make the bug observable.
func TestAsteroidsProof_AE5_RegressionDetector(t *testing.T) {
	t.Skip("AE5 deferred to U11b — needs collision sinks + bug-injection harness. See package doc.")
}
