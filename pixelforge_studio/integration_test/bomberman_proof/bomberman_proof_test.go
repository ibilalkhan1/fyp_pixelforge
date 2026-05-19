//go:build long

// Package bomberman_proof_test is plan-009 U16's end-to-end smoke test
// for the Bomberman reference cart. It loads the .pforge fixture,
// generates placeholder sprite assets into an in-memory FS, boots the
// capsule runtime, drives the trace through the verb-recipe bus +
// physics integrator + RenderTickAtRGBA pipeline, captures pixel hashes
// at three checkpoint ticks (0, 100, 299) plus the per-checkpoint
// verbs.bus event sequence, and asserts parity against a committed
// baseline.
//
// # Scope (U16 smoke)
//
// This unit lands the FIRST end-to-end run of the grid-bomber stack
// (Phase 3's terminal artifact) and is deliberately scoped pragmatically
// to mirror U11's + U14's smoke-level template: the trace exercises
// the BombTimer countdown + spawn/destroy_other + damage/grid_explode
// cascade and proves the load -> boot -> bomb-timer scan -> render ->
// hash pipeline ties together. The trace does NOT clear all breakable
// walls because:
//
//   - There is no per-tick input-driven bomb-placement verb yet. The
//     bomb is pre-staged in the .pforge fixture with a BombTimer
//     component already set to ticks_remaining=120 (option (a) in U16's
//     decision tree). Authoring a `spawn/bomb` or `motion/place_bomb`
//     input-driven verb that constructs an entity at runtime is a U16
//     polish iteration; the smoke value here is "place_on_grid
//     primitive + grid-explode cascade replay deterministically through
//     verbs.bus".
//
//   - The input intent layer (pixelforge_render.applyInputFrame) is
//     still a stash-only stub (same constraint U11 + U14 document).
//     The hero stays idle for the entire 300-tick trace; the bomb
//     detonates independently via the BombTimer AdvanceTick scan.
//
//   - Pixel-correctness of the rendered hero/bomb/breakable sprites is
//     gated on the renderer learning to draw Sprite components at
//     body.Position. For U16, the smoke covers the fact that frames
//     are produced (non-empty, deterministic) and that the verb-bus
//     event sequence at each checkpoint matches the recorded baseline.
//
// # Trace driver design (structural difference from U14)
//
// U14's mario_proof publishes platformer verbs (apply_gravity +
// solid_collide every tick, jump on Space) per trace frame because the
// platformer convention is per-tick verb dispatch from the input
// layer. Bomberman's bomb-detonation cascade is driven entirely by
// `rt.AdvanceTick()` — the BombTimer scan inside the runtime
// decrements the fuse + publishes spawn/destroy_other +
// damage/grid_explode when the timer hits zero. So the U16 trace loop
// publishes NO per-tick verbs (no place_on_grid, no jump, no gravity);
// it only renders the frame and advances the runtime. The cascade is
// observable purely from the BombTimer system's AdvanceTick scan.
//
// (A future U16 polish iteration that wires Space -> spawn/bomb +
// place_on_grid will reintroduce the per-tick verb publish; that's
// orthogonal to the bomb-fuse smoke.)
//
// # Deferred to a U16 polish iteration
//
//   - Full screen clear: every breakable wall destroyed via a chain
//     of placed bombs.
//   - Input-driven bomb placement (a `spawn/bomb` verb wired to a
//     key — option (b) from U16's decision tree).
//   - Hero movement (arrow-key driven place_on_grid, two cells away
//     from the bomb to "survive" the blast).
//   - Multi-bomb chain reactions (bomb explosion triggers adjacent
//     bombs' detonation early — a U17+ concern).
//   - Pixel-hash regression detector under a deliberately-introduced
//     BombTimer off-by-one (mirrors U14's AE5 t.Skip).
//
// # File layout
//
// Mirrors U11/U14: this sub-package lives at
// pixelforge_studio/integration_test/bomberman_proof/ so the Ebitengine
// TestMain wrapper (RenderTickAtRGBA needs a live graphics context for
// ebiten.NewImage + ReadPixels) is isolated from the other long-tag
// tests in the parent package. Fixtures live one level up at
// ../fixtures/bomberman_proof.{pforge,trace.jsonl,baseline.json}.
//
// # Body-size note (per U13/U14 lesson)
//
// Bomberman uses 16×16 cells. We shrink the hero + bomb bodies to 8×8
// inside their cells (matching the SMB-style "narrower collider"
// convention U14 used). Breakable bodies stay at 16×16 (matching tile
// size; they don't move and need to overlap the damage tile rectangle
// for grid_explode's per-tile damage to find them via entityAABB).
package bomberman_proof_test

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
// Run once via `go test -tags=long ./... -run TestBombermanProof
// -update-baseline` after authoring or modifying the trace; the
// generated baseline.json is then committed and gates future runs.
var updateBaseline = flag.Bool("update-baseline", false, "regenerate bomberman_proof.baseline.json from this run")

// checkpointTicks are the trace tick numbers the baseline records.
// Three points: start (tick 0 — pre-fuse), mid-trace (tick 100 —
// fuse counting down, not yet exploded), end (tick 299 — long after
// the bomb fired at tick ~119, scene state should be settled).
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
// suite (and U11/U14's *_proof_test packages). Ebitengine requires a
// live graphics context for ebiten.NewImage + ReadPixels — both used
// by RenderTickAtRGBA. Hosting m.Run() inside one Update +
// ebiten.Termination is the standard workaround.
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
// content variation. Width/height/color are read off the project's
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

// setupBombermanAssetsFS builds an in-memory fs.FS rooted such that
// the four sprite paths the .pforge declares all resolve. Matches the
// capsuleruntime loader's expected layout (assetsRoot = "assets").
func setupBombermanAssetsFS(t *testing.T) fstest.MapFS {
	t.Helper()
	bomber := generatePlaceholderPNG(t, 16, 16, color.RGBA{R: 200, G: 200, B: 80, A: 255})
	bomb := generatePlaceholderPNG(t, 16, 16, color.RGBA{R: 40, G: 40, B: 40, A: 255})
	wall := generatePlaceholderPNG(t, 16, 16, color.RGBA{R: 110, G: 110, B: 130, A: 255})
	breakable := generatePlaceholderPNG(t, 16, 16, color.RGBA{R: 170, G: 110, B: 70, A: 255})
	return fstest.MapFS{
		"assets/sprites/bomber.png":         &fstest.MapFile{Data: bomber},
		"assets/sprites/bomb.png":           &fstest.MapFile{Data: bomb},
		"assets/sprites/wall_tile.png":      &fstest.MapFile{Data: wall},
		"assets/sprites/breakable_tile.png": &fstest.MapFile{Data: breakable},
	}
}

// ----------------------------------------------------------------------------
// In-memory save backend (Boot needs SOMETHING for the save service;
// the production native backend touches the user's home directory and
// pollutes long-running CI). Pattern mirrored from U11/U14.
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

// BaselineEvent / Checkpoint / Baseline mirror U11's + U14's shapes
// byte-for-byte so the CI gate has one schema across all four games'
// baselines.
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

// TestBombermanProof is U16's CI gate. Loads the fixture, replays the
// trace, captures checkpoint hashes + bus events, compares to the
// committed baseline. The `-update-baseline` flag regenerates the
// baseline file (first-run flow).
func TestBombermanProof(t *testing.T) {
	dir := fixturesDir(t)

	// --- load project ----------------------------------------------------
	pforgeF, err := os.Open(filepath.Join(dir, "bomberman_proof.pforge"))
	require.NoError(t, err, "open bomberman_proof.pforge")
	defer pforgeF.Close()
	project, err := pixelforge_project.LoadReader(pforgeF)
	require.NoError(t, err, "load bomberman_proof.pforge")
	require.Equal(t, "bomberman_proof", project.Name)
	require.Equal(t, "bomberman", project.PhysicsPreset)
	require.Len(t, project.Scenes, 1)
	require.Len(t, project.Scenes[0].Entities, 5, "expected 1 hero + 1 bomb + 3 breakable entities")
	require.Len(t, project.Scenes[0].TileAtlases, 1, "expected one wall TileAtlas")

	// Verify the bomb entity is pre-staged with a BombTimer component
	// (the fixture's authored countdown is what drives the cascade —
	// see package doc on the U16 design choice option (a)).
	var bombInitialTicks int
	for i := range project.Scenes[0].Entities {
		e := &project.Scenes[0].Entities[i]
		if e.ID != "bomb_1" {
			continue
		}
		bt, ok := pixelforge_project.BombTimerFor(e)
		require.True(t, ok, "bomb_1 must carry an authored BombTimer component")
		bombInitialTicks = bt.TicksRemaining
		require.Greater(t, bombInitialTicks, 0, "bomb's initial ticks_remaining must be positive")
		require.Less(t, bombInitialTicks, 300, "bomb's fuse must fire within the 300-tick trace window")
		require.Equal(t, 5, bt.GridX, "bomb authored at grid_x=5")
		require.Equal(t, 5, bt.GridY, "bomb authored at grid_y=5")
		require.Equal(t, 2, bt.Range, "bomb authored with range=2")
	}
	require.NotZero(t, bombInitialTicks, "fixture must include a BombTimer for bomb_1")

	// --- boot runtime ----------------------------------------------------
	// Reset registries so prior tests in the same process don't leak
	// subscribers / sprite caches into this Boot.
	capsuleruntime.ResetRegistriesForTest()
	pievent.ResetRegistryForTest()
	piloop.ResetVerbsBusForTest()
	pixelforge_render.ResetForTest()

	assets := setupBombermanAssetsFS(t)
	rt, err := capsuleruntime.Boot(project, assets, capsuleruntime.Options{
		SaveBackendOverride: newMemSaveBackend(),
	})
	require.NoError(t, err, "boot capsule runtime")
	require.NotNil(t, rt)
	require.NotNil(t, rt.Physics)
	require.NotNil(t, rt.Bodies["hero_1"])
	require.NotNil(t, rt.Bodies["bomb_1"])
	require.NotNil(t, rt.Bodies["breakable_1"])

	// --- shrink hero + bomb bodies to 8×8 -------------------------------
	// See package doc / U14 body-size note: the default 16×16 body on
	// a 16×16 ground tile drives a degenerate "both axes overlap
	// equally" MTV under tile-AABB collision. Bomberman uses
	// CollisionModeGridAABB (not tile-AABB; see BombermanConfig in
	// pixelforge_physics/world.go), so the MTV pathology doesn't apply
	// here directly — but keeping the body footprint distinct from the
	// tile footprint mirrors the U13/U14 convention and leaves room for
	// a future per-entity body-footprint authoring surface. Breakable
	// bodies stay at 16×16 because their AABB must overlap the damage
	// tile rectangle for entityAABB in damage_sink.go to find them via
	// publishDamageOnTile.
	heroBody := rt.Bodies["hero_1"]
	heroBody.Width = pixelforge_physics.FixedFromInt(8)
	heroBody.Height = pixelforge_physics.FixedFromInt(8)
	heroBody.Sync()
	bombBody := rt.Bodies["bomb_1"]
	bombBody.Width = pixelforge_physics.FixedFromInt(8)
	bombBody.Height = pixelforge_physics.FixedFromInt(8)
	bombBody.Sync()

	// --- load trace ------------------------------------------------------
	traceF, err := os.Open(filepath.Join(dir, "bomberman_proof.trace.jsonl"))
	require.NoError(t, err, "open trace")
	defer traceF.Close()
	trace, err := pixelforge_replay.LoadTrace(traceF)
	require.NoError(t, err, "decode trace")
	require.Equal(t, "bomberman_proof", trace.Meta.Game)
	require.Len(t, trace.Frames, 300, "trace should have 300 frames at 60TPS for ~5s")

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

	// per-tick observation subscriber: drives the smoke flags off the
	// live bus rather than relying on checkpoint-only capture. The
	// bomb-detonation cascade fires once at the timer-zero tick, well
	// before the tick=299 checkpoint, so the checkpoint sub would miss
	// it — this observer captures the cascade regardless of which tick
	// it lands on.
	var (
		sawDestroyBomb       bool
		sawGridExplode       bool
		sawVisualFlashCenter bool
		takeDamageEntities   []string
		destroyBombTick      uint64
		gridExplodeTick      uint64
	)
	observe := bus.SubscribeAll(func(ev *piloop.VerbEvent, _ pievent.Handler) {
		if ev == nil {
			return
		}
		switch ev.Topic {
		case "spawn/destroy_other":
			if entID, _ := ev.Args["entity"].(string); entID == "bomb_1" {
				sawDestroyBomb = true
				destroyBombTick = currentTick
			}
		case "damage/grid_explode":
			sawGridExplode = true
			gridExplodeTick = currentTick
		case "visual/flash":
			if entID, _ := ev.Args["entity"].(string); entID == "_blast_center" {
				sawVisualFlashCenter = true
			}
		case "damage/take_damage":
			if entID, _ := ev.Args["entity"].(string); entID != "" {
				takeDamageEntities = append(takeDamageEntities, entID)
			}
		}
	})
	defer bus.Unsubscribe(observe)

	checkpointHashes := map[uint64]string{}

	for _, f := range trace.Frames {
		currentTick = f.Tick

		// U16 structural note: NO per-tick verb publishes. The
		// BombTimer scan inside rt.AdvanceTick() owns the entire
		// detonation cascade (decrement -> publish destroy_other +
		// grid_explode when the fuse hits zero). The hero is idle
		// for the whole trace (no key bindings; the input intent
		// layer is still a stash-only stub per U11/U14).

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

		rt.AdvanceTick()
	}

	// --- smoke: bomb fired ----------------------------------------------
	// The BombTimer started at the authored ticks_remaining and
	// decremented by one per AdvanceTick. The destroy + grid_explode
	// cascade must have fired somewhere inside the 300-tick window.
	assert.True(t, sawDestroyBomb, "spawn/destroy_other for bomb_1 must fire (BombTimer hit zero during trace)")
	assert.True(t, sawGridExplode, "damage/grid_explode must fire when the bomb's BombTimer hit zero")
	assert.True(t, sawVisualFlashCenter,
		"damage/grid_explode must publish a visual/flash on _blast_center (the damage sink cascade)")

	// --- smoke: cascade ordering ----------------------------------------
	// destroy_other fires BEFORE grid_explode (the runtime publishes
	// them in that order from advanceBombTimers; see bomb_timer.go
	// pass-2 publish order). Both should land on the same tick.
	assert.Equal(t, destroyBombTick, gridExplodeTick,
		"spawn/destroy_other and damage/grid_explode should fire on the same tick (same AdvanceTick cascade)")

	// --- smoke: bomb entity removed by end of trace ---------------------
	// The spawn sink's destroy handler removes the bomb from
	// CurrentScene.Entities synchronously on the destroy_other event.
	// By the end of the trace, the bomb should no longer be present.
	bombStillPresent := false
	for _, e := range rt.CurrentScene.Entities {
		if e.ID == "bomb_1" {
			bombStillPresent = true
		}
	}
	assert.False(t, bombStillPresent, "bomb_1 must be removed from CurrentScene.Entities after detonation")

	// --- smoke: breakable entities took damage --------------------------
	// The bomb at grid (5,5) with range=2 walks 4 cardinal rays. The
	// authored breakables sit at (5,6) [south], (6,5) [east], (4,5)
	// [west] — all within range=2 and all on open tiles. Each should
	// receive a damage/take_damage event with amount=1 from the
	// damage_sink's publishDamageOnTile. With hp=1, max_hp=1 in the
	// fixture, take_damage drives hp -> 0 and the sink cascades a
	// damage/die which publishes spawn/destroy_other.
	assert.NotEmpty(t, takeDamageEntities,
		"damage/grid_explode must publish damage/take_damage for at least one breakable entity")

	// At minimum the south breakable must have been hit (it's the
	// canonical "single-tile-south" smoke target documented in the
	// package doc).
	sawBreakable1Damage := false
	for _, id := range takeDamageEntities {
		if id == "breakable_1" {
			sawBreakable1Damage = true
		}
	}
	assert.True(t, sawBreakable1Damage,
		"breakable_1 (grid south of bomb) must receive damage/take_damage from the blast")

	// --- smoke: damaged breakable's HP dropped --------------------------
	// breakable_1 was authored with hp=1; the blast applies damage=1
	// per the BombTimer publish. After take_damage drives hp to 0 the
	// damage sink cascades to damage/die -> spawn/destroy_other, so
	// the entity should be gone from the scene by end-of-trace.
	breakable1Present := false
	for _, e := range rt.CurrentScene.Entities {
		if e.ID == "breakable_1" {
			breakable1Present = true
		}
	}
	assert.False(t, breakable1Present,
		"breakable_1 must be removed from the scene after taking lethal damage from the blast")

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
	baselinePath := filepath.Join(dir, "bomberman_proof.baseline.json")
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
		t.Fatalf("baseline file missing at %s; first-time seed with `go test -tags=long ./pixelforge_studio/integration_test/bomberman_proof -run TestBombermanProof -update-baseline`", baselinePath)
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
// AE5 — pixel-hash regression detector (deferred to U16 polish)
// ----------------------------------------------------------------------------

// TestBombermanProof_AE5_RegressionDetector documents the U16 polish
// follow-up: once the input-driven bomb-placement path lands (option
// (b) from U16's decision tree), a deliberate off-by-one in
// advanceBombTimers's TicksRemaining decrement (e.g. decrement by 2
// instead of 1, or flip the fire-on-zero check to fire-on-one) will
// move the detonation tick by a measurable amount and this proof
// test's frame hashes + bus event arg payloads will diverge from
// baseline. The active harness — temporarily mutate the decrement,
// assert TestBombermanProof fails at the expected checkpoint,
// restore — lands alongside the input-driven bomb-placement plumbing
// that makes the timing bug observable on a real user-authored game.
func TestBombermanProof_AE5_RegressionDetector(t *testing.T) {
	t.Skip("AE5 deferred to U16 polish iteration — needs input-driven bomb-placement + bug-injection harness. See package doc.")
}
