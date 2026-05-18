package archetype

import (
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
)

// init registers the six v1 archetype templates. The list mirrors the
// enumeration in docs/plans/2026-05-18-001-feat-entity-verb-sheet-v1-plan.md §U5.
//
// Default verb choices reflect what the U4 catalog actually ships:
//
//   - The plan's nominal "jump" recipe is not registered by U4 (no
//     concrete jump Action exists in v1). The Player's
//     when_input/jump slot defaults to move_with_intent so the
//     designer sees a working verb pre-filled; jump-specific physics
//     are deferred to a future motion subsystem (see U4's
//     EventTopicMotionIntent placeholder note).
//
//   - The plan's nominal Pickup default ("give_points + destroy_self")
//     was supposed to ship as a composite pickup_default recipe in U4,
//     but U4 only registers give_points and destroy_self as separate
//     recipes. Per the U5 implementation note, we default Pickup's
//     when_touched to give_points alone — the designer adds
//     destroy_self via a follow-up rule or via the "Show advanced"
//     fallback (U8) when authoring a real game. Defaulting to
//     destroy_self alone would silently delete the entity without
//     awarding any points; defaulting to give_points alone surfaces
//     the more visible behaviour first.
//
//   - NPC's when_interacted is intentionally empty — open_dialogue is
//     a placeholder recipe in v1 and only becomes meaningful once
//     idea #6's dialogue runtime lands. Leaving the default blank
//     signals to the designer that this slot needs configuration.
//
// All non-empty DefaultVerb values are validated by
// TestArchetype_AllDefaultVerbsResolveInCatalog so typos surface at
// test time rather than at studio-launch time.
func init() {
	RegisterBuiltinArchetypes()
}

// RegisterBuiltinArchetypes registers the v1 archetype set. Exposed
// as a named function (rather than left anonymous inside init) so
// tests that exercise re-registration semantics via ResetForTest can
// restore the baseline without re-importing the package.
func RegisterBuiltinArchetypes() {
	// ---- Player ----------------------------------------------------
	// Player slots span every gameplay input intent plus the lifecycle
	// and per-tick rows the inspector always offers. Order matches the
	// brainstorm's "most-likely-edited first" intent — directional
	// controls cluster after the action verbs because the designer
	// almost always rebinds attack/jump before remapping left/right.
	RegisterArchetype(Template{
		Archetype: ArchetypePlayer,
		TriggerSlots: []TriggerSlot{
			{
				Name:            catalog.TriggerWhenInputJump,
				DisplayName:     "When jump pressed",
				DefaultVerb:     catalog.RecipeMoveWithIntent,
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenInputJump),
			},
			{
				Name:            catalog.TriggerWhenInputAttack,
				DisplayName:     "When attack pressed",
				DefaultVerb:     catalog.RecipePlaySound,
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenInputAttack),
			},
			{
				Name:            catalog.TriggerWhenInputLeft,
				DisplayName:     "When left pressed",
				DefaultVerb:     catalog.RecipeMoveWithIntent,
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenInputLeft),
			},
			{
				Name:            catalog.TriggerWhenInputRight,
				DisplayName:     "When right pressed",
				DefaultVerb:     catalog.RecipeMoveWithIntent,
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenInputRight),
			},
			{
				Name:            catalog.TriggerWhenInputUp,
				DisplayName:     "When up pressed",
				DefaultVerb:     "",
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenInputUp),
			},
			{
				Name:            catalog.TriggerWhenInputDown,
				DisplayName:     "When down pressed",
				DefaultVerb:     "",
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenInputDown),
			},
			{
				Name:            catalog.TriggerWhenDamaged,
				DisplayName:     "When damaged",
				DefaultVerb:     catalog.RecipeTakeDamage,
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenDamaged),
			},
			{
				Name:            catalog.TriggerWhenSceneStarts,
				DisplayName:     "When scene starts",
				DefaultVerb:     "",
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenSceneStarts),
			},
			{
				Name:            catalog.TriggerEveryTick,
				DisplayName:     "Every tick",
				DefaultVerb:     "",
				RelevantRecipes: relevantRecipesFor(catalog.TriggerEveryTick),
			},
		},
	})

	// ---- Enemy -----------------------------------------------------
	// The Enemy template fulfils AE1 directly: when_stepped_on
	// pre-fills with die so a freshly-dropped Goomba is playable
	// without designer configuration.
	RegisterArchetype(Template{
		Archetype: ArchetypeEnemy,
		TriggerSlots: []TriggerSlot{
			{
				Name:            catalog.TriggerWhenSteppedOn,
				DisplayName:     "When stepped on",
				DefaultVerb:     catalog.RecipeDie,
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenSteppedOn),
			},
			{
				Name:            catalog.TriggerWhenShot,
				DisplayName:     "When shot",
				DefaultVerb:     catalog.RecipeDie,
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenShot),
			},
			{
				Name:            catalog.TriggerWhenPlayerNearby,
				DisplayName:     "When player nearby",
				DefaultVerb:     "",
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenPlayerNearby),
			},
			{
				Name:            catalog.TriggerWhenSceneStarts,
				DisplayName:     "When scene starts",
				DefaultVerb:     "",
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenSceneStarts),
			},
			{
				Name:            catalog.TriggerEveryTick,
				DisplayName:     "Every tick",
				DefaultVerb:     "",
				RelevantRecipes: relevantRecipesFor(catalog.TriggerEveryTick),
			},
			{
				Name:            catalog.TriggerWhenDamaged,
				DisplayName:     "When damaged",
				DefaultVerb:     "",
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenDamaged),
			},
			{
				Name:            catalog.TriggerWhenDestroyed,
				DisplayName:     "When destroyed",
				DefaultVerb:     catalog.RecipeGivePoints,
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenDestroyed),
			},
		},
	})

	// ---- Pickup ----------------------------------------------------
	// when_touched defaults to give_points alone; see package-level
	// note on the deferred composite "give_points + destroy_self"
	// recipe. The designer adds destroy_self via the scripting
	// workspace if they want the coin to disappear after collection.
	RegisterArchetype(Template{
		Archetype: ArchetypePickup,
		TriggerSlots: []TriggerSlot{
			{
				Name:            catalog.TriggerWhenTouched,
				DisplayName:     "When touched",
				DefaultVerb:     catalog.RecipeGivePoints,
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenTouched),
			},
			{
				Name:            catalog.TriggerWhenSceneStarts,
				DisplayName:     "When scene starts",
				DefaultVerb:     "",
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenSceneStarts),
			},
			{
				Name:            catalog.TriggerEveryTick,
				DisplayName:     "Every tick",
				DefaultVerb:     "",
				RelevantRecipes: relevantRecipesFor(catalog.TriggerEveryTick),
			},
		},
	})

	// ---- Hazard ----------------------------------------------------
	RegisterArchetype(Template{
		Archetype: ArchetypeHazard,
		TriggerSlots: []TriggerSlot{
			{
				Name:            catalog.TriggerWhenTouched,
				DisplayName:     "When touched",
				DefaultVerb:     catalog.RecipeHurtPlayer,
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenTouched),
			},
			{
				Name:            catalog.TriggerWhenSceneStarts,
				DisplayName:     "When scene starts",
				DefaultVerb:     "",
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenSceneStarts),
			},
			{
				Name:            catalog.TriggerEveryTick,
				DisplayName:     "Every tick",
				DefaultVerb:     "",
				RelevantRecipes: relevantRecipesFor(catalog.TriggerEveryTick),
			},
		},
	})

	// ---- NPC -------------------------------------------------------
	// when_interacted defaults to empty — the designer wires up
	// open_dialogue once idea #6's dialogue runtime exists.
	RegisterArchetype(Template{
		Archetype: ArchetypeNPC,
		TriggerSlots: []TriggerSlot{
			{
				Name:            catalog.TriggerWhenInteracted,
				DisplayName:     "When interacted",
				DefaultVerb:     "",
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenInteracted),
			},
			{
				Name:            catalog.TriggerWhenPlayerNearby,
				DisplayName:     "When player nearby",
				DefaultVerb:     "",
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenPlayerNearby),
			},
			{
				Name:            catalog.TriggerWhenSceneStarts,
				DisplayName:     "When scene starts",
				DefaultVerb:     "",
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenSceneStarts),
			},
		},
	})

	// ---- World -----------------------------------------------------
	// Minimal template for decorative entities and for the default
	// archetype assigned by U2's sanitize pass to pre-v1 entities.
	RegisterArchetype(Template{
		Archetype: ArchetypeWorld,
		TriggerSlots: []TriggerSlot{
			{
				Name:            catalog.TriggerWhenSceneStarts,
				DisplayName:     "When scene starts",
				DefaultVerb:     "",
				RelevantRecipes: relevantRecipesFor(catalog.TriggerWhenSceneStarts),
			},
			{
				Name:            catalog.TriggerEveryTick,
				DisplayName:     "Every tick",
				DefaultVerb:     "",
				RelevantRecipes: relevantRecipesFor(catalog.TriggerEveryTick),
			},
		},
	})
}
