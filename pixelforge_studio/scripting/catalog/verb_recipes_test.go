package catalog_test

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureLog redirects the standard logger to a buffer for the
// duration of fn so tests can assert on warning output emitted by
// catalog's registration paths. The sync ensures parallel test runs
// don't interleave their captures.
var logMu sync.Mutex

func captureLog(t *testing.T, fn func()) string {
	t.Helper()
	logMu.Lock()
	defer logMu.Unlock()
	var buf bytes.Buffer
	prevWriter := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(prevWriter)
		log.SetFlags(prevFlags)
	}()
	fn()
	return buf.String()
}

func TestRegisterVerbRecipe_LookupReturnsRecipe(t *testing.T) {
	const name = "test_recipe_lookup"
	r := catalog.Recipe{
		ActionKind:       "publish_event",
		DefaultArgs:      map[string]any{"target": "loop.main", "event": "test"},
		RelevantTriggers: []string{catalog.TriggerEveryTick},
	}
	catalog.RegisterVerbRecipe(name, r)

	got, ok := catalog.LookupRecipe(name)
	require.True(t, ok, "expected LookupRecipe to find the registered recipe")
	assert.Equal(t, r.ActionKind, got.ActionKind)
	assert.Equal(t, r.DefaultArgs, got.DefaultArgs)
	assert.Equal(t, r.RelevantTriggers, got.RelevantTriggers)
}

func TestRegisterVerbRecipe_DuplicateOverwritesAndLogs(t *testing.T) {
	const name = "test_recipe_duplicate"

	first := catalog.Recipe{
		ActionKind:       "publish_event",
		DefaultArgs:      map[string]any{"target": "loop.main", "event": "first"},
		RelevantTriggers: []string{catalog.TriggerEveryTick},
	}
	second := catalog.Recipe{
		ActionKind:       "publish_event",
		DefaultArgs:      map[string]any{"target": "loop.main", "event": "second"},
		RelevantTriggers: []string{catalog.TriggerWhenSceneStarts},
	}

	catalog.RegisterVerbRecipe(name, first)
	logged := captureLog(t, func() {
		catalog.RegisterVerbRecipe(name, second)
	})

	assert.Contains(t, logged, name)
	assert.Contains(t, strings.ToLower(logged), "re-register",
		"expected duplicate registration to log a warning containing 're-register'")

	got, ok := catalog.LookupRecipe(name)
	require.True(t, ok)
	assert.Equal(t, "second", got.DefaultArgs["event"],
		"expected the later registration to win")
}

func TestRegisterVerbRecipe_EmptyNameRejected(t *testing.T) {
	// Edge case: empty names are silently dropped (mirrors
	// RegisterAction's contract in catalog.go).
	catalog.RegisterVerbRecipe("", catalog.Recipe{ActionKind: "publish_event"})
	_, ok := catalog.LookupRecipe("")
	assert.False(t, ok, "empty recipe name must not be registered")
}

func TestApplyRecipe_DefaultArgsBaked(t *testing.T) {
	// give_points wraps set_value targeting blackboard "score" with
	// a default value of 10. Apply with no overrides should
	// materialise an Effect that, when run, writes 10 to the cell
	// keyed by the blackboard path.
	recipe, ok := catalog.LookupRecipe(catalog.RecipeGivePoints)
	require.True(t, ok)

	cell := &valueCell{}
	ctx := &fakeCtx{values: map[string]catalog.ValueRef{
		"state/score": cell,
	}}

	effect, err := catalog.ApplyWithContext(recipe, nil, ctx)
	require.NoError(t, err)
	require.NotNil(t, effect)

	effect(nil)
	assert.Equal(t, float64(10), cell.v,
		"give_points default amount should be baked at apply time")
}

func TestApplyRecipe_OverridesWin(t *testing.T) {
	recipe, ok := catalog.LookupRecipe(catalog.RecipeGivePoints)
	require.True(t, ok)

	cell := &valueCell{}
	ctx := &fakeCtx{values: map[string]catalog.ValueRef{
		"state/score": cell,
	}}

	effect, err := catalog.ApplyWithContext(recipe, map[string]any{
		"value": float64(50),
	}, ctx)
	require.NoError(t, err)

	effect(nil)
	assert.Equal(t, float64(50), cell.v,
		"override should win over the baked default")
}

func TestApplyRecipe_OverrideDoesNotMutateRecipe(t *testing.T) {
	// Defensive: a previous Apply with overrides must not leak back
	// into the recipe's DefaultArgs map (would corrupt subsequent
	// Apply calls).
	recipe, ok := catalog.LookupRecipe(catalog.RecipeGivePoints)
	require.True(t, ok)

	originalValue := recipe.DefaultArgs["value"]

	cell := &valueCell{}
	ctx := &fakeCtx{values: map[string]catalog.ValueRef{
		"state/score": cell,
	}}
	_, err := catalog.ApplyWithContext(recipe, map[string]any{"value": float64(999)}, ctx)
	require.NoError(t, err)

	again, ok := catalog.LookupRecipe(catalog.RecipeGivePoints)
	require.True(t, ok)
	assert.Equal(t, originalValue, again.DefaultArgs["value"],
		"Apply must not mutate the recipe's DefaultArgs map")
}

func TestApplyRecipe_UnknownActionKind(t *testing.T) {
	r := catalog.Recipe{
		ActionKind:       "definitely_not_a_registered_action",
		DefaultArgs:      map[string]any{},
		RelevantTriggers: []string{catalog.TriggerEveryTick},
	}
	_, err := catalog.Apply(r, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "definitely_not_a_registered_action")
}

func TestVerbRecipes_AllBuiltinRecipesValid(t *testing.T) {
	// Every registered recipe must:
	//   1) reference an ActionKind registered via RegisterAction;
	//   2) produce a non-nil Effect when applied with no overrides.
	// This catches typos in ActionKind references at test time.
	names := catalog.AllRecipes()
	require.NotEmpty(t, names, "expected the v1 builtin recipes to be registered at init")

	ctx := &fakeCtx{
		samples:  map[string]any{},
		entities: map[string]any{},
		values:   map[string]catalog.ValueRef{},
		targets:  map[string]any{},
	}

	for _, name := range names {
		name := name
		t.Run(name, func(t *testing.T) {
			recipe, ok := catalog.LookupRecipe(name)
			require.True(t, ok)

			// Idea #6 v1 U10 introduces condition-style recipes
			// (ConditionKind set, ActionKind empty). Skip the
			// action-side validation for those — the predicate
			// dispatch path tests cover them separately.
			if recipe.IsCondition() {
				require.Empty(t, recipe.ActionKind,
					"condition recipe %q must leave ActionKind empty (mutually exclusive)", name)
				return
			}

			require.NotEmpty(t, recipe.ActionKind, "recipe %q has empty ActionKind", name)
			builder := catalog.LookupAction(recipe.ActionKind)
			require.NotNil(t, builder, "recipe %q references unregistered action %q", name, recipe.ActionKind)

			effect, err := catalog.ApplyWithContext(recipe, nil, ctx)
			require.NoError(t, err, "recipe %q failed to apply with default args", name)
			require.NotNil(t, effect, "recipe %q applied to a nil Effect", name)
			// Invoking the effect with nil payload is the runtime
			// fast-path; recipes must tolerate it without panicking.
			assert.NotPanics(t, func() { effect(nil) },
				"recipe %q panicked when its effect was invoked", name)
		})
	}
}

func TestVerbRecipes_RelevantTriggersDeclaredForEach(t *testing.T) {
	names := catalog.AllRecipes()
	require.NotEmpty(t, names)
	for _, name := range names {
		recipe, ok := catalog.LookupRecipe(name)
		require.True(t, ok)
		assert.NotEmpty(t, recipe.RelevantTriggers,
			"recipe %q must declare at least one RelevantTrigger so it appears in some dropdown",
			name)
	}
}

func TestVerbRecipes_ExpectedBuiltinsRegistered(t *testing.T) {
	// Spot-check that the v1 enumerated recipes are all present.
	// Failure here means a recipe was removed or renamed; update the
	// expectation deliberately.
	expected := []string{
		catalog.RecipeDie, catalog.RecipeHurtPlayer, catalog.RecipeTakeDamage,
		catalog.RecipeGivePoints, catalog.RecipeLoseLife, catalog.RecipeRestoreHealth,
		catalog.RecipeSetFlag, catalog.RecipeCheckFlag,
		catalog.RecipeGiveItem, catalog.RecipeTakeItem,
		catalog.RecipePlaySound, catalog.RecipePlayMusic, catalog.RecipeStopMusic,
		catalog.RecipeMoveWithIntent, catalog.RecipeMovePattern, catalog.RecipeBounce, catalog.RecipeTeleportTo,
		catalog.RecipeSpawnEntity, catalog.RecipeDestroySelf, catalog.RecipeDestroyOther,
		catalog.RecipeChangeScene, catalog.RecipeWait, catalog.RecipeRestartScene,
		catalog.RecipeHide, catalog.RecipeShow, catalog.RecipeFlash, catalog.RecipeSwapSprite,
		catalog.RecipeOpenDialogue, catalog.RecipeOpenMenu,
	}
	for _, name := range expected {
		_, ok := catalog.LookupRecipe(name)
		assert.True(t, ok, "expected builtin recipe %q to be registered", name)
	}

	// 28 named verbs in the v1 plan; AllRecipes should at least
	// surface that count (test-side registrations may add more).
	assert.GreaterOrEqual(t, len(catalog.AllRecipes()), len(expected),
		"AllRecipes should include every v1 builtin")
}
