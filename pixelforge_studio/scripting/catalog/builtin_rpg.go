// builtin_rpg.go owns idea #6 v1 U10's RPG-class verb-recipe
// additions. The recipes are placeholder shells that publish well-
// known event-bus topics; the concrete runtime subsystems
// (pixelforge_dialogue, pixelforge_menus, pixelforge_save,
// inventory blackboard helpers) subscribe to those topics and run
// the actual behaviour. This keeps the catalog decoupled from the
// runtime engine implementations — designers get the verb dropdown
// without the catalog importing every RPG package.
//
// has_item is the sole condition-style recipe in v1 (Recipe with
// ConditionKind set). The verb-binding compiler routes it through
// the rule's pre-fire predicate slot.
package catalog

// New event-bus topics for the RPG-class verbs. Names follow the
// existing audio/inventory/scene convention: subsystem-then-action.
const (
	// Dialogue control topics.
	EventTopicUICloseDialogue = "ui/close_dialogue"
	// Menu control topics.
	EventTopicUICloseMenu = "ui/close_menu"
	// Inventory item-count override (set, not increment).
	EventTopicInventorySetItemCount = "inventory/set_item_count"
	// Save-pipeline topics.
	EventTopicSaveNow    = "save/now"
	EventTopicSaveLoad   = "save/load_slot"
	EventTopicSaveDelete = "save/delete_slot"
	// Inventory has-item predicate topic. Conditions don't publish;
	// the topic name exists so debug surfaces can talk about it.
	EventTopicInventoryHasItem = "inventory/has_item"
)

// Recipe-name constants for the RPG additions.
const (
	RecipeCloseDialogue = "close_dialogue"
	RecipeCloseMenu     = "close_menu"
	RecipeSetItemCount  = "set_item_count"
	RecipeSaveNow       = "save_now"
	RecipeLoadSlot      = "load_slot"
	RecipeDeleteSlot    = "delete_slot"
	RecipeHasItem       = "has_item"
)

// RegisterBuiltinRPGRecipes installs the RPG-class recipes onto
// the package-level verb-recipe registry. Called from init(); also
// exported so tests + the catalog reload path can re-register
// after ResetRecipesForTest.
func RegisterBuiltinRPGRecipes() {
	RegisterVerbRecipe(RecipeCloseDialogue, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicUICloseDialogue,
		},
		RelevantTriggers: []string{TriggerWhenInputUse, TriggerWhenInputMenu, TriggerEveryTick},
	})

	RegisterVerbRecipe(RecipeCloseMenu, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicUICloseMenu,
		},
		RelevantTriggers: []string{TriggerWhenInputMenu, TriggerWhenInputStart, TriggerEveryTick},
	})

	RegisterVerbRecipe(RecipeSetItemCount, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicInventorySetItemCount,
		},
		RelevantTriggers: []string{TriggerWhenSceneStarts, TriggerWhenStoryProgress(), TriggerEveryTick},
	})

	RegisterVerbRecipe(RecipeSaveNow, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicSaveNow,
			"slot":   "slot1",
		},
		RelevantTriggers: []string{TriggerWhenInteracted, TriggerWhenSteppedOn, TriggerWhenSceneStarts},
	})

	RegisterVerbRecipe(RecipeLoadSlot, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicSaveLoad,
			"slot":   "slot1",
		},
		RelevantTriggers: []string{TriggerWhenInteracted, TriggerWhenInputUse, TriggerWhenSceneStarts},
	})

	RegisterVerbRecipe(RecipeDeleteSlot, Recipe{
		ActionKind: "publish_event",
		DefaultArgs: map[string]any{
			"target": EventBusTarget,
			"event":  EventTopicSaveDelete,
			"slot":   "slot1",
		},
		RelevantTriggers: []string{TriggerWhenInteracted, TriggerWhenInputUse},
	})

	// has_item is the predicate; runtime evaluator checks the
	// blackboard's "inventory" key. ConditionKind set ⇒ this
	// recipe routes through the rule's pre-fire condition slot
	// rather than the action slot. ActionKind stays empty per the
	// mutually-exclusive contract on Recipe.
	RegisterVerbRecipe(RecipeHasItem, Recipe{
		ConditionKind: "has_item",
		DefaultArgs: map[string]any{
			"item_id": "",
		},
		RelevantTriggers: []string{TriggerEveryTick, TriggerWhenInteracted, TriggerWhenSteppedOn},
	})
}

// TriggerWhenStoryProgress is a placeholder trigger-slot name used
// where the brainstorm calls for a "narrative progression" hook
// but no existing trigger fits exactly. v1 surfaces it as a
// pseudo-trigger; v2 promotes it to a first-class slot once
// archetype templates declare it.
func TriggerWhenStoryProgress() string { return "when_story_progress" }
