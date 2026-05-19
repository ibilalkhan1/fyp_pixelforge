// builtin_arcade.go owns plan-008 U8's arcade-primitive verb
// recipes. These are the missing verbs the four reference games
// (Asteroids, Bomberman, Mario, Donkey Kong) need but the v1
// catalog didn't ship — see docs/reference-games-capability-matrix.md
// for the per-game justification.
//
// All recipes are placeholder shells that publish well-known event
// topics on the "verbs.bus" target (see EventBusTarget). Concrete
// physics + entity behaviour lives in the capsule runtime's
// subscriber set (pixelforge_studio/capsuleruntime/subscribers.go,
// U2). The catalog stays decoupled from the runtime engines:
// designers get the verb dropdown without the catalog importing
// every physics package.

package catalog

// Arcade event-bus topics. Names follow the existing convention —
// subsystem-then-action, lower-snake. The capsule subscriber
// dispatches by these strings.
const (
	// Motion / physics topics.
	EventTopicMotionApplyThrust  = "motion/apply_thrust"
	EventTopicMotionRotateEntity = "motion/rotate_entity"
	EventTopicMotionScreenWrap   = "motion/screen_wrap"
	EventTopicMotionJump         = "motion/jump"
	EventTopicMotionApplyGravity = "motion/apply_gravity"
	EventTopicMotionPlaceOnGrid  = "motion/place_on_grid"
	EventTopicMotionLadderClimb  = "motion/ladder_climb"
	EventTopicMotionBarrelRoll   = "motion/barrel_roll"

	// Collision topic — solid-tile penetration resolution.
	EventTopicCollisionSolidCollide = "collision/solid_collide"

	// Damage topic — radius-AoE explosion (Bomberman bomb).
	EventTopicDamageExplodeRadius = "damage/explode_radius"

	// Flow topic — fixed-rate per-tick grouping. Authoring helper
	// for "do X every N ticks while Y is true" sequences that the
	// per-rule trigger surface can't express in one step.
	EventTopicFlowFixedTickLoop = "flow/fixed_tick_loop"
)

// Arcade recipe-name constants. Studio surfaces + tests reference
// recipes through these so typos fail at compile time.
const (
	RecipeApplyThrust   = "apply_thrust"
	RecipeRotateEntity  = "rotate_entity"
	RecipeScreenWrap    = "screen_wrap"
	RecipeJump          = "jump"
	RecipeApplyGravity  = "apply_gravity"
	RecipeSolidCollide  = "solid_collide"
	RecipePlaceOnGrid   = "place_on_grid"
	RecipeExplodeRadius = "explode_radius"
	RecipeLadderClimb   = "ladder_climb"
	RecipeBarrelRoll    = "barrel_roll"
	RecipeFixedTickLoop = "fixed_tick_loop"
)

// RegisterBuiltinArcadeRecipes installs U8's arcade primitives on
// the package-level verb-recipe registry. Called from init() —
// exposed so the catalog reload path (and tests that exercise
// re-registration semantics via ResetRecipesForTest) can restore
// the baseline without re-importing the package.
func RegisterBuiltinArcadeRecipes() {
	// ---- Asteroids primitives -----------------------------------

	// apply_thrust — pushes the entity along its current heading.
	// Asteroids ship reads this every tick while the thrust key is
	// held; magnitude tuned to feel right on a 30 TPS loop.
	RegisterVerbRecipe(RecipeApplyThrust, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target":     EventBusTarget,
			"event":      EventTopicMotionApplyThrust,
			"angle_deg":  float64(0),
			"magnitude":  float64(0.2),
		},
		RelevantTriggers: concat(gameplayInputTriggers, []string{TriggerEveryTick}),
	})

	// rotate_entity — adjusts the entity's rotation by `degrees`
	// per fire. Asteroids ship binds it to left + right inputs
	// with opposite signs.
	RegisterVerbRecipe(RecipeRotateEntity, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target":  EventBusTarget,
			"event":   EventTopicMotionRotateEntity,
			"degrees": float64(5),
		},
		RelevantTriggers: concat(gameplayInputTriggers, []string{TriggerEveryTick}),
	})

	// screen_wrap — repositions the entity to the opposite screen
	// edge when it crosses out of bounds. Asteroids ship +
	// asteroids + bullets all share the verb.
	RegisterVerbRecipe(RecipeScreenWrap, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicMotionScreenWrap,
		},
		RelevantTriggers: []string{TriggerEveryTick},
	})

	// ---- Mario / Donkey Kong platformer primitives --------------

	// jump — sets the entity's vertical velocity to a negative
	// burst. Default tuned for Mario's classic 5-cell hop on the
	// 8x8 tile grid; designers override `strength` for higher
	// jumps + power-up state.
	RegisterVerbRecipe(RecipeJump, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target":   EventBusTarget,
			"event":    EventTopicMotionJump,
			"strength": float64(5),
		},
		RelevantTriggers: []string{TriggerWhenInputJump, TriggerWhenInputUse},
	})

	// apply_gravity — adds positive vertical velocity per tick.
	// Bound to TriggerEveryTick on Mario / DK player entities;
	// `strength` is tuned for the 30 TPS loop.
	RegisterVerbRecipe(RecipeApplyGravity, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target":   EventBusTarget,
			"event":    EventTopicMotionApplyGravity,
			"strength": float64(0.3),
		},
		RelevantTriggers: []string{TriggerEveryTick, TriggerWhenSceneStarts},
	})

	// solid_collide — entity-vs-tile penetration resolution
	// against the scene's solid-flagged tile-atlas layer. Resolves
	// horizontal + vertical penetration separately so an entity
	// can slide along a wall.
	RegisterVerbRecipe(RecipeSolidCollide, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicCollisionSolidCollide,
		},
		RelevantTriggers: []string{TriggerEveryTick},
	})

	// ---- Bomberman primitives -----------------------------------

	// place_on_grid — snaps the entity's position to the nearest
	// grid cell. Bomberman bomb placement reads `cell_size = 16`
	// to match the canonical NES sprite grid.
	RegisterVerbRecipe(RecipePlaceOnGrid, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target":    EventBusTarget,
			"event":     EventTopicMotionPlaceOnGrid,
			"cell_size": float64(16),
		},
		RelevantTriggers: concat(gameplayInputTriggers, []string{TriggerWhenInteracted, TriggerWhenInputUse}),
	})

	// explode_radius — after `fuse_ticks` elapse, destroy self
	// + damage every entity within `radius` for `damage` hp.
	// Bomberman bomb's full behaviour in one verb; the subscriber
	// handles the tick countdown.
	RegisterVerbRecipe(RecipeExplodeRadius, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target":     EventBusTarget,
			"event":      EventTopicDamageExplodeRadius,
			"fuse_ticks": float64(120),
			"radius":     float64(32),
			"damage":     float64(1),
		},
		RelevantTriggers: lifecycleTriggers,
	})

	// ---- Donkey Kong primitives ---------------------------------

	// ladder_climb — overrides gravity while the entity overlaps
	// a ladder-flagged tile. `speed` is the per-tick climb rate;
	// negative values descend, positive climb.
	RegisterVerbRecipe(RecipeLadderClimb, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicMotionLadderClimb,
			"speed":  float64(1.5),
		},
		RelevantTriggers: concat([]string{TriggerWhenInputUp, TriggerWhenInputDown, TriggerEveryTick}, nil),
	})

	// barrel_roll — applies per-tile slope-flag direction to the
	// entity's velocity. DK barrel reads the scene's slope flags
	// to roll downhill at `base_speed` along each girder.
	RegisterVerbRecipe(RecipeBarrelRoll, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target":     EventBusTarget,
			"event":      EventTopicMotionBarrelRoll,
			"base_speed": float64(1.0),
		},
		RelevantTriggers: []string{TriggerEveryTick, TriggerWhenSceneStarts},
	})

	// ---- Cross-game flow primitive ------------------------------

	// fixed_tick_loop — substrate for "do X every tick while Y
	// is true". `ticks_per_step` controls cadence — every Nth
	// tick the grouped action fires once. Used by all four games
	// for input-polling sequences the v1 trigger surface can't
	// express in one rule.
	RegisterVerbRecipe(RecipeFixedTickLoop, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target":          EventBusTarget,
			"event":           EventTopicFlowFixedTickLoop,
			"ticks_per_step":  float64(1),
		},
		RelevantTriggers: []string{TriggerEveryTick, TriggerWhenSceneStarts},
	})
}
