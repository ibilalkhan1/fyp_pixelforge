package catalog

import (
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_blackboard"
)

// Recipe is the verb-sheet authoring shape on top of an existing
// Action Kind. A Recipe wraps an ActionKind already registered via
// RegisterAction and pre-fills its Args with sensible defaults; the
// inspector (U8) renders Recipes as named verbs in per-trigger
// dropdowns, while the verb-binding compiler (U6) materialises them
// into EventSheetRule entries at load time.
//
// Recipes never introduce new runtime behaviour — every recipe must
// resolve to an existing ActionKind via LookupAction. Where the v1
// runtime does not yet ship a concrete Action for a verb (audio
// streaming, sprite swapping, dialogue, etc.), the recipe references
// the generic "publish_event" Action so the verb publishes a
// well-known topic that downstream subsystems can subscribe to. These
// placeholder shells are flagged in this file's source so future units
// can swap them for first-class Actions without touching authoring.
type Recipe struct {
	// ActionKind is the name of an existing registered ActionBuilder
	// (see builtin_actions.go). LookupAction(ActionKind) must return a
	// non-nil builder for the recipe to apply successfully.
	ActionKind string

	// DefaultArgs is the baked-in Args map merged with any
	// per-binding overrides supplied by the designer. Per the JSON
	// convention used throughout the catalog, numeric values are
	// stored as float64 so they round-trip cleanly through
	// .pforge marshalling.
	DefaultArgs map[string]any

	// RelevantTriggers is the non-empty list of trigger-slot names
	// (e.g., "when_stepped_on", "when_input/jump") for which this
	// recipe should appear in the verb-pick dropdown. An empty list
	// would hide the recipe from every slot, so registration enforces
	// at least one entry via the test scenarios.
	RelevantTriggers []string
}

// Verb recipe name constants. Studio surfaces and tests reference
// recipes through these symbols to catch typos at compile time
// instead of at load time.
const (
	// Damage verbs.
	RecipeDie        = "die"
	RecipeHurtPlayer = "hurt_player"
	RecipeTakeDamage = "take_damage"

	// State verbs.
	RecipeGivePoints     = "give_points"
	RecipeLoseLife       = "lose_life"
	RecipeRestoreHealth  = "restore_health"
	RecipeSetFlag        = "set_flag"
	RecipeCheckFlag      = "check_flag"
	RecipeGiveItem       = "give_item"
	RecipeTakeItem       = "take_item"

	// Audio verbs.
	RecipePlaySound = "play_sound"
	RecipePlayMusic = "play_music"
	RecipeStopMusic = "stop_music"

	// Motion verbs.
	RecipeMoveWithIntent = "move_with_intent"
	RecipeMovePattern    = "move_pattern"
	RecipeBounce         = "bounce"
	RecipeTeleportTo     = "teleport_to"

	// Spawn verbs.
	RecipeSpawnEntity   = "spawn_entity"
	RecipeDestroySelf   = "destroy_self"
	RecipeDestroyOther  = "destroy_other"

	// Flow verbs.
	RecipeChangeScene  = "change_scene"
	RecipeWait         = "wait"
	RecipeRestartScene = "restart_scene"

	// Visual verbs.
	RecipeHide       = "hide"
	RecipeShow       = "show"
	RecipeFlash      = "flash"
	RecipeSwapSprite = "swap_sprite"

	// Dialogue / menu verbs (placeholder shells, fully wired by idea #6).
	RecipeOpenDialogue = "open_dialogue"
	RecipeOpenMenu     = "open_menu"
)

// Trigger-slot name constants used by RelevantTriggers wiring. The
// archetype templates (U5) and the inspector dropdown filter (U8)
// consume the same strings.
const (
	TriggerWhenSceneStarts  = "when_scene_starts"
	TriggerEveryTick        = "every_tick"
	TriggerWhenSteppedOn    = "when_stepped_on"
	TriggerWhenShot         = "when_shot"
	TriggerWhenDamaged      = "when_damaged"
	TriggerWhenDestroyed    = "when_destroyed"
	TriggerWhenTouched      = "when_touched"
	TriggerWhenInteracted   = "when_interacted"
	TriggerWhenPlayerNearby = "when_player_nearby"
	TriggerWhenInputJump    = "when_input/jump"
	TriggerWhenInputAttack  = "when_input/attack"
	TriggerWhenInputLeft    = "when_input/left"
	TriggerWhenInputRight   = "when_input/right"
	TriggerWhenInputUp      = "when_input/up"
	TriggerWhenInputDown    = "when_input/down"
	TriggerWhenInputUse     = "when_input/use"
	TriggerWhenInputMenu    = "when_input/menu"
	TriggerWhenInputStart   = "when_input/start"
)

var (
	recipeMu sync.RWMutex
	recipes  = map[string]Recipe{}
)

// RegisterVerbRecipe adds (name, recipe) to the verb-recipe registry.
// Duplicate names log a warning and overwrite, mirroring
// RegisterAction (catalog.go:110-120) so test reloads and overrides
// behave identically across both registries.
//
// RegisterVerbRecipe does NOT validate that recipe.ActionKind is
// already registered — registration order across init() blocks is not
// guaranteed across packages, so the resolve-to-Action check runs at
// Apply time (or at TestVerbRecipes_AllBuiltinRecipesValid time when
// the catalog's own builtins are exercised together).
func RegisterVerbRecipe(name string, recipe Recipe) {
	if name == "" {
		return
	}
	recipeMu.Lock()
	defer recipeMu.Unlock()
	if _, exists := recipes[name]; exists {
		log.Printf("[catalog] verb recipe %q re-registered; overwriting", name)
	}
	recipes[name] = recipe
}

// LookupRecipe returns the Recipe registered under name and whether
// one is present.
func LookupRecipe(name string) (Recipe, bool) {
	recipeMu.RLock()
	defer recipeMu.RUnlock()
	r, ok := recipes[name]
	return r, ok
}

// AllRecipes returns every registered recipe name in alphabetical
// order. Studio dropdowns sort their own lists by display name; this
// helper exists for tests + diagnostics.
func AllRecipes() []string {
	recipeMu.RLock()
	defer recipeMu.RUnlock()
	out := make([]string, 0, len(recipes))
	for k := range recipes {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ResetRecipesForTest clears the verb-recipe registry. Test-only;
// callers that exercise registration semantics must re-run init by
// calling RegisterBuiltinRecipes (or simply not invoke this helper).
func ResetRecipesForTest() {
	recipeMu.Lock()
	defer recipeMu.Unlock()
	recipes = map[string]Recipe{}
}

// Apply materialises recipe into an Effect by merging recipe defaults
// with per-binding overrides and invoking the underlying
// ActionBuilder. Overrides win on key collision.
//
// Apply returns an error when recipe.ActionKind is not registered,
// when the merged Args fail the builder's validation, or when the
// builder itself reports an error. The verb-binding compiler (U6)
// surfaces these errors by marking the slot "advanced (composed)".
func Apply(recipe Recipe, overrides map[string]any) (Effect, error) {
	builder := LookupAction(recipe.ActionKind)
	if builder == nil {
		return nil, fmt.Errorf("verb recipe references unknown action kind %q", recipe.ActionKind)
	}
	merged := mergeArgs(recipe.DefaultArgs, overrides)
	return builder(merged, nil)
}

// ApplyWithContext is the Context-aware sibling of Apply. The
// verb-binding compiler (U6) uses this form so the resulting Effect
// can dereference scene-resident assets at fire time.
func ApplyWithContext(recipe Recipe, overrides map[string]any, ctx Context) (Effect, error) {
	builder := LookupAction(recipe.ActionKind)
	if builder == nil {
		return nil, fmt.Errorf("verb recipe references unknown action kind %q", recipe.ActionKind)
	}
	merged := mergeArgs(recipe.DefaultArgs, overrides)
	return builder(merged, ctx)
}

// mergeArgs returns a fresh map carrying every entry from defaults
// overlaid with overrides. Neither input is mutated; callers may pass
// nil for either side.
func mergeArgs(defaults, overrides map[string]any) map[string]any {
	out := make(map[string]any, len(defaults)+len(overrides))
	for k, v := range defaults {
		out[k] = v
	}
	for k, v := range overrides {
		out[k] = v
	}
	return out
}

// blackboardPath constructs the canonical ValueRef path for a
// blackboard key. The runtime Context (pixelforge_studio/scripting/
// runtime/context.go) interprets paths beginning with "state/" as
// blackboard lookups; recipes that mutate blackboard state spell
// their target with this helper to keep the convention discoverable.
func blackboardPath(key string) string {
	return "state/" + key
}

// Event topic names published by placeholder recipes. Downstream
// subsystems (audio streaming, sprite swapping, dialogue, menus)
// subscribe to these topics in their own init() blocks. The strings
// are public via these constants so the consumers can reference the
// same symbol the producers use.
const (
	EventTopicAudioPlaySound = "audio/play_sound"
	EventTopicAudioPlayMusic = "audio/play_music"
	EventTopicAudioStopMusic = "audio/stop_music"

	EventTopicMotionMovePattern = "motion/move_pattern"
	EventTopicMotionBounce      = "motion/bounce"
	EventTopicMotionTeleport    = "motion/teleport_to"
	EventTopicMotionIntent      = "motion/move_with_intent"

	EventTopicSpawnEntity   = "spawn/entity"
	EventTopicDestroySelf   = "spawn/destroy_self"
	EventTopicDestroyOther  = "spawn/destroy_other"

	EventTopicSceneChange  = "scene/change"
	EventTopicSceneRestart = "scene/restart"
	EventTopicSceneWait    = "scene/wait"

	EventTopicVisualHide       = "visual/hide"
	EventTopicVisualShow       = "visual/show"
	EventTopicVisualFlash      = "visual/flash"
	EventTopicVisualSwapSprite = "visual/swap_sprite"

	EventTopicInventoryGiveItem = "inventory/give_item"
	EventTopicInventoryTakeItem = "inventory/take_item"

	EventTopicDamageDie        = "damage/die"
	EventTopicDamageHurtPlayer = "damage/hurt_player"
	EventTopicDamageTakeDamage = "damage/take_damage"

	EventTopicUIOpenDialogue = "ui/open_dialogue"
	EventTopicUIOpenMenu     = "ui/open_menu"

	// EventBusTarget is the well-known pievent target name on which
	// placeholder recipes publish. Subsystems looking to bind a
	// concrete behaviour to a verb subscribe here and filter by event
	// topic string.
	EventBusTarget = "loop.main"
)

// gameplayInputTriggers groups the input intent triggers that
// represent active player intent — most verbs that are meaningful
// "the player presses a button" responses appear under these slots.
// Defined once so the per-recipe RelevantTriggers slices stay
// readable.
var gameplayInputTriggers = []string{
	TriggerWhenInputJump,
	TriggerWhenInputAttack,
	TriggerWhenInputUse,
	TriggerWhenInputLeft,
	TriggerWhenInputRight,
	TriggerWhenInputUp,
	TriggerWhenInputDown,
}

// collisionTriggers groups slots where two entities interact —
// damage, pickup, and hazard verbs cluster here.
var collisionTriggers = []string{
	TriggerWhenSteppedOn,
	TriggerWhenShot,
	TriggerWhenTouched,
	TriggerWhenDamaged,
}

// lifecycleTriggers groups one-shot per-entity slots — spawn,
// destroy, scene boundaries.
var lifecycleTriggers = []string{
	TriggerWhenSceneStarts,
	TriggerWhenDestroyed,
}

// concat returns a fresh slice carrying every element from each input.
// Used to compose RelevantTriggers from the shared groupings above.
func concat(slices ...[]string) []string {
	n := 0
	for _, s := range slices {
		n += len(s)
	}
	out := make([]string, 0, n)
	for _, s := range slices {
		out = append(out, s...)
	}
	return out
}

func init() {
	RegisterBuiltinRecipes()
}

// RegisterBuiltinRecipes registers the v1 verb recipe set. Exposed as
// a named function (rather than left anonymous inside init) so tests
// that exercise re-registration semantics via ResetRecipesForTest can
// restore the baseline without re-importing the package.
//
// The list below mirrors the v1 enumeration in
// docs/plans/2026-05-18-001-feat-entity-verb-sheet-v1-plan.md §U4.
// Recipes flagged as "PLACEHOLDER" wrap publish_event because no
// concrete Action for the verb exists in the v1 runtime; the topic
// strings under EventTopic* are the contract downstream subsystems
// implement.
func RegisterBuiltinRecipes() {
	// ---- Damage ----------------------------------------------------
	// die — PLACEHOLDER: emits "damage/die" so the spawn subsystem
	// (idea #1) can remove the entity. No concrete "remove entity"
	// Action exists in v1.
	RegisterVerbRecipe(RecipeDie, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicDamageDie,
		},
		RelevantTriggers: concat(collisionTriggers, []string{TriggerWhenDestroyed}),
	})

	// hurt_player — PLACEHOLDER: emits "damage/hurt_player" carrying
	// the default damage amount via the event topic. Idea #4's audio
	// + idea #5's blackboard subscribers translate this into a
	// blackboard health decrement.
	RegisterVerbRecipe(RecipeHurtPlayer, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicDamageHurtPlayer,
			"amount": float64(1),
		},
		RelevantTriggers: concat(collisionTriggers, []string{TriggerEveryTick}),
	})

	// take_damage — wraps set_value against blackboard "health".
	// DefaultArgs target the canonical health key with a sentinel
	// "value" that the runtime interprets as a delta when paired
	// with set_value's ValueRef. v1 keeps this as a simple
	// overwrite-to-default; recipes that need delta math go through
	// publish_event (hurt_player) until the runtime ships a delta
	// Action.
	RegisterVerbRecipe(RecipeTakeDamage, Recipe{
		ActionKind: "set_value",
		DefaultArgs: map[string]any{
			"target": blackboardPath(pixelforge_blackboard.KeyHealth),
			"value":  float64(0),
		},
		RelevantTriggers: concat(collisionTriggers, []string{TriggerWhenDestroyed}),
	})

	// ---- State -----------------------------------------------------
	// give_points — wraps set_value against blackboard "score" with a
	// default of 10. The verb-binding compiler (U6) is expected to
	// upgrade this to a delta-add at runtime; for v1 the recipe shape
	// is the seam.
	RegisterVerbRecipe(RecipeGivePoints, Recipe{
		ActionKind: "set_value",
		DefaultArgs: map[string]any{
			"target": blackboardPath(pixelforge_blackboard.KeyScore),
			"value":  float64(10),
		},
		RelevantTriggers: concat(collisionTriggers, lifecycleTriggers, []string{TriggerWhenTouched}),
	})

	// lose_life — wraps set_value against blackboard "lives".
	RegisterVerbRecipe(RecipeLoseLife, Recipe{
		ActionKind: "set_value",
		DefaultArgs: map[string]any{
			"target": blackboardPath(pixelforge_blackboard.KeyLives),
			"value":  float64(0),
		},
		RelevantTriggers: concat(collisionTriggers, []string{TriggerWhenDestroyed}),
	})

	// restore_health — wraps set_value against blackboard "health"
	// with the canonical default value.
	RegisterVerbRecipe(RecipeRestoreHealth, Recipe{
		ActionKind: "set_value",
		DefaultArgs: map[string]any{
			"target": blackboardPath(pixelforge_blackboard.KeyHealth),
			"value":  float64(3),
		},
		RelevantTriggers: []string{TriggerWhenTouched, TriggerWhenInteracted, TriggerWhenSceneStarts},
	})

	// set_flag — wraps set_value against an arbitrary blackboard
	// flag key (designer supplies the key + value via overrides).
	RegisterVerbRecipe(RecipeSetFlag, Recipe{
		ActionKind: "set_value",
		DefaultArgs: map[string]any{
			"target": blackboardPath("flag"),
			"value":  true,
		},
		RelevantTriggers: concat(collisionTriggers, lifecycleTriggers, gameplayInputTriggers, []string{TriggerWhenInteracted}),
	})

	// check_flag — PLACEHOLDER: no Action-side "check" verb exists in
	// v1; the verb merely publishes a "flag/check" probe topic so
	// other rules can react. The real check happens via Condition
	// kind "value_eq" in the rule's Conditions list (compiled by U6).
	RegisterVerbRecipe(RecipeCheckFlag, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  "flag/check",
		},
		RelevantTriggers: []string{TriggerEveryTick, TriggerWhenSceneStarts, TriggerWhenInteracted},
	})

	// give_item — PLACEHOLDER: emits "inventory/give_item" so the
	// inventory subsystem (idea #6) can update its own state.
	RegisterVerbRecipe(RecipeGiveItem, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicInventoryGiveItem,
		},
		RelevantTriggers: []string{TriggerWhenTouched, TriggerWhenInteracted, TriggerWhenSceneStarts},
	})

	// take_item — PLACEHOLDER: emits "inventory/take_item".
	RegisterVerbRecipe(RecipeTakeItem, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicInventoryTakeItem,
		},
		RelevantTriggers: []string{TriggerWhenInteracted, TriggerWhenTouched},
	})

	// ---- Audio -----------------------------------------------------
	// play_sound — wraps play_sample directly (a concrete Action
	// exists). Default name is empty so the designer must pick a
	// sample in the inspector parameter form.
	RegisterVerbRecipe(RecipePlaySound, Recipe{
		ActionKind: "play_sample",
		DefaultArgs: map[string]any{
			"name":    "",
			"pitch":   float64(1.0),
			"volume":  float64(1.0),
			"channel": float64(1),
		},
		RelevantTriggers: concat(collisionTriggers, lifecycleTriggers, gameplayInputTriggers, []string{TriggerWhenTouched, TriggerWhenInteracted, TriggerEveryTick}),
	})

	// play_music — PLACEHOLDER: streaming music is idea #4 scope;
	// v1 emits "audio/play_music" so the eventual streaming module
	// can subscribe.
	RegisterVerbRecipe(RecipePlayMusic, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicAudioPlayMusic,
		},
		RelevantTriggers: []string{TriggerWhenSceneStarts, TriggerWhenInteracted},
	})

	// stop_music — PLACEHOLDER counterpart to play_music.
	RegisterVerbRecipe(RecipeStopMusic, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicAudioStopMusic,
		},
		RelevantTriggers: []string{TriggerWhenSceneStarts, TriggerWhenInteracted, TriggerWhenDestroyed},
	})

	// ---- Motion ----------------------------------------------------
	// move_with_intent — PLACEHOLDER: the runtime's move_entity
	// Action moves by a fixed delta; intent-driven movement requires
	// the U3 input layer + per-tick polling, so v1 emits a topic for
	// the dedicated motion subsystem to consume.
	RegisterVerbRecipe(RecipeMoveWithIntent, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicMotionIntent,
		},
		RelevantTriggers: concat(gameplayInputTriggers, []string{TriggerEveryTick}),
	})

	// move_pattern — PLACEHOLDER: pattern motion (sine wave, circle,
	// path-follow) is deferred to a future motion library; v1
	// announces the verb so authoring is unblocked.
	RegisterVerbRecipe(RecipeMovePattern, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target":  EventBusTarget,
			"event":   EventTopicMotionMovePattern,
			"pattern": "sine",
		},
		RelevantTriggers: []string{TriggerEveryTick, TriggerWhenSceneStarts},
	})

	// bounce — PLACEHOLDER: physics bounce response is deferred.
	RegisterVerbRecipe(RecipeBounce, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicMotionBounce,
		},
		RelevantTriggers: concat(collisionTriggers, []string{TriggerEveryTick}),
	})

	// teleport_to — wraps move_entity with a default zero offset; the
	// designer typically overrides dx / dy in the inspector. The
	// "entity" arg is left empty so the verb-binding compiler can
	// stamp in the host entity ID at compile time.
	RegisterVerbRecipe(RecipeTeleportTo, Recipe{
		ActionKind: "move_entity",
		DefaultArgs: map[string]any{
			"entity": "",
			"dx":     float64(0),
			"dy":     float64(0),
		},
		RelevantTriggers: concat(gameplayInputTriggers, lifecycleTriggers, []string{TriggerWhenInteracted, TriggerWhenTouched}),
	})

	// ---- Spawn -----------------------------------------------------
	// spawn_entity — PLACEHOLDER: entity spawning is idea #1's
	// ScreenRoom scope; v1 emits the topic.
	RegisterVerbRecipe(RecipeSpawnEntity, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicSpawnEntity,
		},
		RelevantTriggers: concat(lifecycleTriggers, []string{TriggerEveryTick, TriggerWhenInteracted}),
	})

	// destroy_self — PLACEHOLDER counterpart to spawn_entity for
	// the host entity.
	RegisterVerbRecipe(RecipeDestroySelf, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicDestroySelf,
		},
		RelevantTriggers: concat(collisionTriggers, []string{TriggerWhenDestroyed, TriggerWhenTouched}),
	})

	// destroy_other — PLACEHOLDER: removes the colliding entity.
	RegisterVerbRecipe(RecipeDestroyOther, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicDestroyOther,
		},
		RelevantTriggers: concat(collisionTriggers, []string{TriggerWhenTouched}),
	})

	// ---- Flow ------------------------------------------------------
	// change_scene — PLACEHOLDER: scene transitions are not yet a
	// first-class Action; v1 announces via the bus and also writes
	// the canonical "current_scene" blackboard key in a follow-on
	// rule that U6 generates from the recipe.
	RegisterVerbRecipe(RecipeChangeScene, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicSceneChange,
		},
		RelevantTriggers: []string{TriggerWhenInteracted, TriggerWhenTouched, TriggerWhenSceneStarts, TriggerWhenDestroyed},
	})

	// wait — PLACEHOLDER: a Step "Wait" exists for the step-lane
	// authoring path, but the verb-sheet path has no Action
	// equivalent. v1 emits a "scene/wait" topic so a future
	// scheduling subsystem can implement it; the recipe default
	// carries a duration in ticks.
	RegisterVerbRecipe(RecipeWait, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicSceneWait,
			"ticks":  float64(60),
		},
		RelevantTriggers: concat(lifecycleTriggers, []string{TriggerEveryTick, TriggerWhenInteracted}),
	})

	// restart_scene — PLACEHOLDER counterpart to change_scene.
	RegisterVerbRecipe(RecipeRestartScene, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicSceneRestart,
		},
		RelevantTriggers: []string{TriggerWhenDestroyed, TriggerWhenInteracted, TriggerWhenSceneStarts},
	})

	// ---- Visual ----------------------------------------------------
	// hide — PLACEHOLDER: visibility toggling is deferred.
	RegisterVerbRecipe(RecipeHide, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicVisualHide,
		},
		RelevantTriggers: concat(collisionTriggers, lifecycleTriggers, []string{TriggerWhenInteracted, TriggerWhenTouched}),
	})

	// show — PLACEHOLDER counterpart to hide.
	RegisterVerbRecipe(RecipeShow, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicVisualShow,
		},
		RelevantTriggers: concat(lifecycleTriggers, []string{TriggerWhenInteracted, TriggerWhenTouched, TriggerEveryTick}),
	})

	// flash — PLACEHOLDER: invuln-flash visual effect.
	RegisterVerbRecipe(RecipeFlash, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target":   EventBusTarget,
			"event":    EventTopicVisualFlash,
			"duration": float64(30),
		},
		RelevantTriggers: concat(collisionTriggers, []string{TriggerWhenDamaged, TriggerWhenSceneStarts}),
	})

	// swap_sprite — PLACEHOLDER: animated sprite swap is deferred.
	RegisterVerbRecipe(RecipeSwapSprite, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicVisualSwapSprite,
		},
		RelevantTriggers: concat(gameplayInputTriggers, collisionTriggers, lifecycleTriggers, []string{TriggerEveryTick, TriggerWhenInteracted, TriggerWhenTouched}),
	})

	// ---- Dialogue / Menu (placeholder shells, idea #6) ------------
	// open_dialogue — PLACEHOLDER: emits "ui/open_dialogue" for the
	// idea #6 dialogue runtime to consume.
	RegisterVerbRecipe(RecipeOpenDialogue, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicUIOpenDialogue,
		},
		RelevantTriggers: []string{TriggerWhenInteracted, TriggerWhenTouched, TriggerWhenPlayerNearby, TriggerWhenSceneStarts},
	})

	// open_menu — PLACEHOLDER counterpart to open_dialogue.
	RegisterVerbRecipe(RecipeOpenMenu, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicUIOpenMenu,
		},
		RelevantTriggers: []string{TriggerWhenInputMenu, TriggerWhenInputStart, TriggerWhenInteracted, TriggerWhenSceneStarts},
	})
}
