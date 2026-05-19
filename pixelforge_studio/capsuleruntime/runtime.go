package capsuleruntime

import (
	"fmt"
	"io/fs"
	"log"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_physics"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"
)

// Options tunes Boot's behaviour. Zero-value Options is the
// production posture: load everything, install subscribers, and
// hand the populated Runtime back to the caller. The capsule's
// CapsuleRun feeds its CapsuleOpts.DataOverride (used by the
// studio preview to inject unsaved edits) through here unchanged.
type Options struct {
	// SkipSubscribers leaves the verb-recipe event-bus subscribers
	// un-installed. Tests that exercise loader semantics in
	// isolation flip this true to avoid touching the global
	// pievent registry.
	SkipSubscribers bool

	// SaveBackendOverride substitutes a custom Backend (typically
	// an in-memory fake) for the file-system or localStorage one
	// the production path selects via build tags. Tests use this
	// to keep capsule-runtime Boot hermetic.
	SaveBackendOverride pisave.Backend

	// SinksOverride substitutes one or more subsystem sinks with
	// test-supplied fakes. Any field left nil receives the
	// production default via Sinks.fillDefaults. Tests that
	// exercise a single handler typically set just the field they
	// care about.
	SinksOverride Sinks
}

// Runtime is the per-capsule wiring Boot returns. Exposes the
// loaded project + the save service the subscriber layer dispatches
// save/load/delete verbs through. The capsule's CapsuleRun keeps a
// reference so the engine packages can reach Runtime state through
// well-known accessors at frame time.
//
// Plan-009 U7 adds the per-runtime mutable game-state fields the
// concrete scene / visual / spawn / dialogue / music sinks mutate:
// CurrentScene names the active scene (initialised from
// Project.Scenes[0] at Boot); InitialEntities snapshots each scene's
// entity list at Boot so scene/restart can roll the live scene back
// to its authored shape; Tick is the frame counter sinks use to
// expire timed effects; VisualEffects is the per-entity flash/swap
// override table; DialogueScript is the active dialogue ID (empty
// when no dialogue is open); SceneWaitTicks is the countdown the
// scene/wait verb sets; CurrentMusic is the BGM name the music sink
// reports. None of these fields are populated by the engine itself —
// they are mutated by the sinks in response to verb-recipe events,
// and the renderer + game-loop read them per frame.
type Runtime struct {
	Project *pixelforge_project.Project
	Save    *pisave.Service
	Sinks   Sinks

	// CurrentScene is the active scene the sinks mutate. Pointer
	// so mutations land in-place; sceneSink swaps the pointer on
	// scene/change. nil only when Project has no scenes (degenerate
	// case the loader normally prevents).
	CurrentScene *pixelforge_project.Scene

	// InitialEntities snapshots each scene's authored entity list
	// at Boot — scene/restart copies the snapshot back into the
	// live scene so the restart returns to authored state, not to
	// whatever the post-mutation state happens to be. Keyed by
	// scene ID.
	InitialEntities map[string][]pixelforge_project.Entity

	// Tick is the frame counter sinks consult to expire timed
	// effects (visual/flash duration, scene/wait countdown).
	// Engine drivers advance this once per game tick.
	Tick uint64

	// VisualEffects holds per-entity transient render overrides
	// (flash color, sprite swap, hide/show). Keyed by entity ID.
	// Effects with non-zero ExpireTick clear themselves when
	// rt.Tick reaches ExpireTick.
	VisualEffects map[string]*VisualEffect

	// DialogueScript is the active dialogue script ID. Empty when
	// no dialogue is open. Mutated by dialogueSink.
	DialogueScript string

	// SceneWaitTicks is the countdown the scene/wait verb sets.
	// Engine drivers decrement this once per tick; sinks consult
	// the field to gate further dispatch during wait windows.
	SceneWaitTicks int

	// CurrentMusic is the BGM name the music sink reports as
	// currently playing. Empty when the music channel is stopped.
	// Mutated by musicSink.
	CurrentMusic string

	// Plan-009 U8 — Physics is the per-runtime physics world the
	// motion sink dispatches into. Constructed at Boot from
	// Project.PhysicsPreset; nil only when Project is nil
	// (degenerate). The world owns the PhysicsConfig the motion
	// sink consults (e.g. ScreenWrap, ScreenWidth/Height).
	Physics *pixelforge_physics.World

	// Bodies maps entity ID → physics body. The motion sink looks
	// up the body for an incoming motion/* verb and mutates its
	// Velocity/Rotation/Position in place. Populated at Boot from
	// the current scene's entity list; subsequent spawn/destroy
	// verbs update the map as entities come and go. The map is
	// never nil after Boot; missing-entity lookups return nil and
	// the sink logs a warning.
	Bodies map[string]*pixelforge_physics.Body

	// Plan-009 U9 — Globals is the cross-sink scratchpad for
	// per-runtime mutable state that isn't tied to a single entity.
	// The damage sink stores player_hp here (decremented by
	// damage/hurt_player) and reads player_entity (the ID the next
	// damage/die targets when player HP hits zero). Other sinks may
	// store their own keys here in later units; the map is created
	// at Boot so callers never have to nil-check before writing.
	// Keys are free-form strings; the convention is "subsystem_field"
	// (e.g. "player_hp", "player_entity") so collisions between
	// sinks stay visible.
	Globals map[string]any

	// Plan-009 U17 — BarrelStates tracks the per-entity rolling-
	// barrel state for motion/barrel_roll: current direction (left/
	// right) and per-tick sweep speed. Keyed by entity ID. Populated
	// lazily on the first barrel_roll verb fire for an entity; rolled
	// forward each tick by the motion sink.
	BarrelStates map[string]pixelforge_physics.BarrelState

	// Plan-009 U17 — CameraOffsetY is the vertical scroll offset
	// the renderer applies to a multi-screen-tall scene (e.g. DK's
	// 4-screen climb). Computed from the player entity's body Y and
	// clamped to [0, totalHeight-screenHeight]. Zero for single-
	// screen scenes (Scene.GridHeightScreens <= 1). The renderer
	// (pixelforge_render/rendertick.go) consults this field and the
	// engine's Camera follower reads it for tile + entity blitting.
	CameraOffsetY int
}

// VisualEffect carries one transient per-entity render override.
// Hide/Show flip the Hidden bit; Flash queues a palette override
// that expires at ExpireTick; SwapSprite names a replacement sprite
// the renderer consults instead of the entity's authored sprite.
// Zero-value VisualEffect is "no override" — the renderer renders
// the entity exactly as authored.
type VisualEffect struct {
	// Hidden gates rendering. Set by visual/hide, cleared by
	// visual/show.
	Hidden bool

	// FlashColor is the palette override the renderer blends over
	// the entity's pixels while the flash is active. Empty when
	// no flash is queued.
	FlashColor string

	// SpriteSwap names the sprite the renderer uses instead of
	// the entity's authored sprite. Empty when no swap is in
	// effect.
	SpriteSwap string

	// ExpireTick is the tick at which timed effects (flash) clear
	// themselves. Zero means the effect persists until cleared
	// explicitly (hide/show, sprite swap).
	ExpireTick uint64
}

// Boot is the canonical capsule entry point. Loads sprites + audio
// + dialogue from the embedded assets FS, populates the capsule-
// side scene + item registries, constructs the save service for the
// host platform, then installs the verb-recipe event-bus
// subscribers (U2). Returns the populated *Runtime so the caller
// (the codegen-emitted CapsuleRun) can keep a reference.
//
// On loader failure Boot returns the error without installing
// subscribers — a partially-loaded capsule wouldn't run any
// authored game correctly, so failing fast is preferable to
// shipping a half-functional process.
//
// Boot is idempotent against the registries: re-running with the
// same project + assets produces equivalent post-state because each
// Register* call replaces the prior entry. The save service is
// re-constructed on every call.
func Boot(p *pixelforge_project.Project, assets fs.FS, opts Options) (*Runtime, error) {
	if p == nil {
		return nil, fmt.Errorf("capsuleruntime: project is nil")
	}
	if err := LoadSprites(p, assets); err != nil {
		return nil, err
	}
	if err := LoadAudio(p, assets); err != nil {
		return nil, err
	}
	if err := LoadDialogue(p); err != nil {
		return nil, err
	}
	LoadScenes(p)
	LoadItems(p)

	save, err := newSaveService(p, opts.SaveBackendOverride)
	if err != nil {
		return nil, err
	}
	rt := &Runtime{
		Project:         p,
		Save:            save,
		Sinks:           opts.SinksOverride,
		VisualEffects:   map[string]*VisualEffect{},
		InitialEntities: map[string][]pixelforge_project.Entity{},
		Bodies:          map[string]*pixelforge_physics.Body{},
		Globals:         map[string]any{},
		BarrelStates:    map[string]pixelforge_physics.BarrelState{},
	}

	// Plan-009 U7: pin the first scene as CurrentScene so the
	// scene sink has a live pointer to mutate, and snapshot every
	// scene's authored entity list so scene/restart can roll back.
	if len(p.Scenes) > 0 {
		rt.CurrentScene = &p.Scenes[0]
	}
	for i := range p.Scenes {
		s := &p.Scenes[i]
		snap := make([]pixelforge_project.Entity, len(s.Entities))
		copy(snap, s.Entities)
		rt.InitialEntities[s.ID] = snap
	}

	// Plan-009 U8: construct the physics world from the project's
	// PhysicsPreset and seed Bodies with one body per entity in
	// the current scene. The body's Position is derived from the
	// entity's authored EntityPosition (X/Y) converted to Fixed32.
	// Width/Height default to 16×16 — a placeholder until the
	// renderer hands authored sprite dimensions to the runtime in
	// a later unit; the AABB doesn't affect motion math, only
	// collision queries (which Asteroids skips entirely).
	rt.Physics = pixelforge_physics.NewWorld(physicsConfigForPreset(p.PhysicsPreset))
	if rt.CurrentScene != nil {
		for i := range rt.CurrentScene.Entities {
			e := &rt.CurrentScene.Entities[i]
			body := pixelforge_physics.NewBody(
				e.ID,
				pixelforge_physics.Vec2{
					X: pixelforge_physics.FixedFromFloat(e.Position.X),
					Y: pixelforge_physics.FixedFromFloat(e.Position.Y),
				},
				pixelforge_physics.FixedFromInt(16),
				pixelforge_physics.FixedFromInt(16),
			)
			rt.Bodies[e.ID] = body
			rt.Physics.AddBody(body)
		}
	}

	if !opts.SkipSubscribers {
		InstallSubscribers(rt)
	}
	return rt, nil
}

// physicsConfigForPreset translates the project's string preset name
// into a pixelforge_physics.PhysicsConfig. Empty preset returns the
// zero-value config (no gravity, no wrap, no collision); unrecognised
// values log a warning and fall back to the zero-value. The string-
// preset indirection keeps pixelforge_project free of an import on
// pixelforge_physics (and keeps .pforge JSON designer-friendly —
// "physics_preset": "asteroids" rather than a raw Fixed32 dump).
func physicsConfigForPreset(preset string) pixelforge_physics.PhysicsConfig {
	switch preset {
	case "":
		return pixelforge_physics.PhysicsConfig{}
	case "asteroids":
		return pixelforge_physics.AsteroidsConfig()
	case "mario":
		return pixelforge_physics.MarioConfig()
	case "bomberman":
		return pixelforge_physics.BombermanConfig()
	case "dk":
		return pixelforge_physics.DKConfig()
	default:
		log.Printf("[capsuleruntime] unknown physics_preset %q — falling back to empty config", preset)
		return pixelforge_physics.PhysicsConfig{}
	}
}

// AdvanceTick is the per-tick housekeeping hook the engine driver
// calls once per game tick. It increments rt.Tick, expires timed
// visual effects whose ExpireTick has elapsed, decrements the
// scene/wait countdown, and runs the U15 bomb-timer scan that
// detonates any fuse that reached zero this tick. Cheap (constant-
// time per active effect; linear in scene-entity count for the bomb
// scan, which is small for a typical Bomberman level); safe to call
// when Runtime is nil (no-op).
//
// Effect expiry runs in-place: an entry whose flash window has
// elapsed clears its FlashColor + ExpireTick but stays in the map
// if other fields (Hidden, SpriteSwap) are still set. Fully-cleared
// entries are deleted so the map doesn't grow without bound across
// a long session.
//
// Bomb-timer scan: see advanceBombTimers in bomb_timer.go for the
// detonation cascade ordering. The scan runs AFTER the visual-effect
// expiry so a flash queued by a previous explosion has already
// cleared before the next bomb fires (avoiding flash-overwrite
// inconsistencies across replays).
func (rt *Runtime) AdvanceTick() {
	if rt == nil {
		return
	}
	rt.Tick++
	if rt.SceneWaitTicks > 0 {
		rt.SceneWaitTicks--
	}
	for id, eff := range rt.VisualEffects {
		if eff == nil {
			delete(rt.VisualEffects, id)
			continue
		}
		if eff.ExpireTick != 0 && rt.Tick >= eff.ExpireTick {
			eff.FlashColor = ""
			eff.ExpireTick = 0
		}
		if !eff.Hidden && eff.FlashColor == "" && eff.SpriteSwap == "" {
			delete(rt.VisualEffects, id)
		}
	}
	advanceBombTimers(rt)
}

// newSaveService picks the right backend for the host platform —
// production callers leave opts.SaveBackendOverride nil and the
// build-tagged backend constructor (backend_native.go on desktop,
// backend_js.go on WASM) supplies the real backend. Tests pass an
// in-memory backend through Options to stay hermetic.
//
// gameTitle derives from the project's SaveConfig (designer-set
// override) falling back to Project.Name so two different games on
// the same host don't collide in the save directory.
func newSaveService(p *pixelforge_project.Project, override pisave.Backend) (*pisave.Service, error) {
	if override != nil {
		return pisave.NewService(override), nil
	}
	title := p.SaveConfig.GameTitle
	if title == "" {
		title = p.Name
	}
	backend, err := defaultSaveBackend(title)
	if err != nil {
		return nil, fmt.Errorf("capsuleruntime: save backend: %w", err)
	}
	return pisave.NewService(backend), nil
}
