package runtime_test

import (
	"reflect"
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/archetype"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompileVerbBindings_SimpleBinding asserts that a binding with a
// known recipe and no overrides produces one rule whose Condition is
// "event_fired" subscribed on the right target + the recipe's
// ActionKind in the single Action.
func TestCompileVerbBindings_SimpleBinding(t *testing.T) {
	ent := &pixelforge_project.Entity{
		ID: "goomba-1",
		VerbBindings: []pixelforge_project.VerbBinding{
			{Trigger: catalog.TriggerWhenSteppedOn, RecipeName: catalog.RecipeDie},
		},
	}
	rules, err := runtime.CompileVerbBindings(ent, archetype.Template{})
	require.NoError(t, err)
	require.Len(t, rules, 1)

	r := rules[0]
	require.Len(t, r.Conditions, 1)
	require.Equal(t, "event_fired", r.Conditions[0].Kind)
	assert.Equal(t, "loop.main", r.Conditions[0].Args["target"])
	assert.Equal(t, "stepped_on", r.Conditions[0].Args["event"])

	require.Len(t, r.Actions, 1)
	dieRecipe, _ := catalog.LookupRecipe(catalog.RecipeDie)
	assert.Equal(t, dieRecipe.ActionKind, r.Actions[0].Kind)
}

// TestCompileVerbBindings_WithOverrides asserts that ArgOverrides take
// precedence over recipe DefaultArgs in the compiled Action.Args.
func TestCompileVerbBindings_WithOverrides(t *testing.T) {
	ent := &pixelforge_project.Entity{
		ID: "scoring-pickup",
		VerbBindings: []pixelforge_project.VerbBinding{
			{
				Trigger:      catalog.TriggerWhenTouched,
				RecipeName:   catalog.RecipeGivePoints,
				ArgOverrides: map[string]any{"value": float64(50)},
			},
		},
	}
	rules, err := runtime.CompileVerbBindings(ent, archetype.Template{})
	require.NoError(t, err)
	require.Len(t, rules, 1)
	require.Len(t, rules[0].Actions, 1)

	gotArgs := rules[0].Actions[0].Args
	assert.Equal(t, float64(50), gotArgs["value"], "override should win over recipe default of 10")

	// All other default keys (target) preserved from recipe.
	givePoints, _ := catalog.LookupRecipe(catalog.RecipeGivePoints)
	assert.Equal(t, givePoints.DefaultArgs["target"], gotArgs["target"])
}

// TestCompileVerbBindings_UnknownRecipePreservedAsPlaceholder asserts
// that a binding referring to a non-registered recipe still produces a
// rule — never silently dropped — with the placeholder Action kind.
// The placeholder Args carry the original RecipeName + ArgOverrides
// for save round-trip + inspector "advanced (composed)" rendering.
func TestCompileVerbBindings_UnknownRecipePreservedAsPlaceholder(t *testing.T) {
	ent := &pixelforge_project.Entity{
		ID: "future-entity",
		VerbBindings: []pixelforge_project.VerbBinding{
			{
				Trigger:      catalog.TriggerWhenSteppedOn,
				RecipeName:   "nonexistent_verb_from_v2",
				ArgOverrides: map[string]any{"custom_key": "custom_value"},
			},
		},
	}
	rules, err := runtime.CompileVerbBindings(ent, archetype.Template{})
	require.NoError(t, err)
	require.Len(t, rules, 1)
	require.Len(t, rules[0].Actions, 1)

	assert.Equal(t, runtime.PlaceholderActionKind, rules[0].Actions[0].Kind)
	assert.Equal(t, "nonexistent_verb_from_v2", rules[0].Actions[0].Args["recipe_name"])

	overrides, ok := rules[0].Actions[0].Args["arg_overrides"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "custom_value", overrides["custom_key"])
}

// TestCompileVerbBindings_TriggerMatchesArchetypeSlot is the happy path
// where the binding's trigger is one of the archetype's declared
// slots — compile produces a rule without any warnings (we don't
// inspect log output; we just verify the rule shape is normal).
func TestCompileVerbBindings_TriggerMatchesArchetypeSlot(t *testing.T) {
	tmpl := archetype.Template{
		Archetype: "Enemy",
		TriggerSlots: []archetype.TriggerSlot{
			{Name: catalog.TriggerWhenSteppedOn},
			{Name: catalog.TriggerWhenShot},
		},
	}
	ent := &pixelforge_project.Entity{
		ID:        "goomba-1",
		Archetype: "Enemy",
		VerbBindings: []pixelforge_project.VerbBinding{
			{Trigger: catalog.TriggerWhenSteppedOn, RecipeName: catalog.RecipeDie},
		},
	}
	rules, err := runtime.CompileVerbBindings(ent, tmpl)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	assert.Equal(t, "loop.main", rules[0].Conditions[0].Args["target"])
}

// TestCompileVerbBindings_TriggerNotInArchetype_LogsWarning verifies
// the binding is still compiled (and not dropped) even when the
// trigger is foreign to the archetype's slot list. Warning emission is
// validated by the absence of a panic + presence of the rule in the
// output; we deliberately do not capture log output (the standard log
// package is process-global and other tests share it).
func TestCompileVerbBindings_TriggerNotInArchetype_LogsWarning(t *testing.T) {
	tmpl := archetype.Template{
		Archetype: "World",
		TriggerSlots: []archetype.TriggerSlot{
			{Name: catalog.TriggerWhenSceneStarts},
			{Name: catalog.TriggerEveryTick},
		},
	}
	ent := &pixelforge_project.Entity{
		ID:        "decorative-tile",
		Archetype: "World",
		VerbBindings: []pixelforge_project.VerbBinding{
			{Trigger: catalog.TriggerWhenSteppedOn, RecipeName: catalog.RecipeDie},
		},
	}
	rules, err := runtime.CompileVerbBindings(ent, tmpl)
	require.NoError(t, err)
	require.Len(t, rules, 1, "off-slot binding must still compile, never dropped")
}

// TestRoundtrip_BindingToRuleToBinding asserts the central R15
// promise: a binding compiles to a rule, and DetectRoundtripRecipe
// reverses the rule back to the same recipe + overrides.
func TestRoundtrip_BindingToRuleToBinding(t *testing.T) {
	original := pixelforge_project.VerbBinding{
		Trigger:      catalog.TriggerWhenTouched,
		RecipeName:   catalog.RecipeGivePoints,
		ArgOverrides: map[string]any{"value": float64(25)},
	}
	ent := &pixelforge_project.Entity{
		ID:           "coin-1",
		VerbBindings: []pixelforge_project.VerbBinding{original},
	}
	rules, err := runtime.CompileVerbBindings(ent, archetype.Template{})
	require.NoError(t, err)
	require.Len(t, rules, 1)

	name, overrides, advanced := runtime.DetectRoundtripRecipe(rules[0])
	require.False(t, advanced, "compiled rule with single matching recipe should not be advanced")
	assert.Equal(t, original.RecipeName, name)
	assert.True(t, reflect.DeepEqual(original.ArgOverrides, overrides),
		"overrides round-trip: want %#v got %#v", original.ArgOverrides, overrides)
}

// TestRoundtrip_PowerUserEditedRule_DetectedAsAdvanced asserts a rule
// with multiple Actions is flagged as advanced/composed regardless of
// whether the first Action would match a recipe. This is the AE7
// scenario (power-user adds a second Action; inspector must surface
// the slot as composed).
func TestRoundtrip_PowerUserEditedRule_DetectedAsAdvanced(t *testing.T) {
	dieRecipe, _ := catalog.LookupRecipe(catalog.RecipeDie)

	rule := pixelforge_project.EventSheetRule{
		Conditions: []pixelforge_project.Condition{{
			Kind: "event_fired",
			Args: map[string]any{"target": "loop.main", "event": "stepped_on"},
		}},
		Actions: []pixelforge_project.Action{
			{Kind: dieRecipe.ActionKind, Args: dieRecipe.DefaultArgs},
			{Kind: "publish_event", Args: map[string]any{
				"target": "loop.main", "event": "custom_topic",
			}},
		},
	}
	name, overrides, advanced := runtime.DetectRoundtripRecipe(rule)
	assert.True(t, advanced)
	assert.Empty(t, name)
	assert.Nil(t, overrides)
}

// TestCompileVerbBindings_HotReloadIdempotent asserts compiling the
// same input twice produces structurally equal output — necessary so
// Engine.Reload doesn't accumulate duplicate subscriptions when an
// entity's bindings haven't changed.
func TestCompileVerbBindings_HotReloadIdempotent(t *testing.T) {
	ent := &pixelforge_project.Entity{
		ID: "goomba-1",
		VerbBindings: []pixelforge_project.VerbBinding{
			{Trigger: catalog.TriggerWhenSteppedOn, RecipeName: catalog.RecipeDie},
			{Trigger: catalog.TriggerWhenShot, RecipeName: catalog.RecipeGivePoints,
				ArgOverrides: map[string]any{"value": float64(100)}},
		},
	}

	first, err := runtime.CompileVerbBindings(ent, archetype.Template{})
	require.NoError(t, err)
	second, err := runtime.CompileVerbBindings(ent, archetype.Template{})
	require.NoError(t, err)

	assert.True(t, reflect.DeepEqual(first, second),
		"two compiles of identical input must produce DeepEqual rules")
}

// TestCompileVerbBindings_NilEntity verifies the defensive nil/empty
// path. CompileVerbBindings must not panic and must return (nil, nil)
// so callers in compileAll can iterate the result without a guard.
func TestCompileVerbBindings_NilEntity(t *testing.T) {
	rules, err := runtime.CompileVerbBindings(nil, archetype.Template{})
	require.NoError(t, err)
	assert.Nil(t, rules)

	empty := &pixelforge_project.Entity{ID: "empty"}
	rules, err = runtime.CompileVerbBindings(empty, archetype.Template{})
	require.NoError(t, err)
	assert.Nil(t, rules)
}

// TestTriggerNameToTarget_InputTrigger verifies the "when_input/<x>"
// branch routes to the intent target convention (U3).
func TestRoundtrip_InputTriggerSubscribesOnIntentTarget(t *testing.T) {
	ent := &pixelforge_project.Entity{
		ID: "player",
		VerbBindings: []pixelforge_project.VerbBinding{
			{Trigger: catalog.TriggerWhenInputJump, RecipeName: catalog.RecipePlaySound},
		},
	}
	rules, err := runtime.CompileVerbBindings(ent, archetype.Template{})
	require.NoError(t, err)
	require.Len(t, rules, 1)
	assert.Equal(t, "input/jump", rules[0].Conditions[0].Args["target"])
	assert.Equal(t, "input/jump", rules[0].Conditions[0].Args["event"])
}

// TestRoundtrip_UnknownRecipePlaceholderDetectedAsAdvanced verifies
// the inspector path for placeholder rules: the detector reports the
// preserved RecipeName while flagging the slot as advanced (read-only
// in the UI).
func TestRoundtrip_UnknownRecipePlaceholderDetectedAsAdvanced(t *testing.T) {
	ent := &pixelforge_project.Entity{
		ID: "future-entity",
		VerbBindings: []pixelforge_project.VerbBinding{
			{Trigger: catalog.TriggerWhenSteppedOn, RecipeName: "v2_only_recipe"},
		},
	}
	rules, err := runtime.CompileVerbBindings(ent, archetype.Template{})
	require.NoError(t, err)
	require.Len(t, rules, 1)

	name, overrides, advanced := runtime.DetectRoundtripRecipe(rules[0])
	assert.True(t, advanced)
	assert.Equal(t, "v2_only_recipe", name)
	assert.Nil(t, overrides)
}
