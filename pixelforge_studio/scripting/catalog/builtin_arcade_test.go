package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
)

// arcadeRecipeIDs is the U8 set the matrix doc enumerates.
// Wherever a U8 row is added or removed, this slice updates in
// lockstep — the test below uses it as the source of truth for
// "every arcade recipe is registered + valid".
var arcadeRecipeIDs = []string{
	catalog.RecipeApplyThrust,
	catalog.RecipeRotateEntity,
	catalog.RecipeScreenWrap,
	catalog.RecipeJump,
	catalog.RecipeApplyGravity,
	catalog.RecipeSolidCollide,
	catalog.RecipePlaceOnGrid,
	catalog.RecipeExplodeRadius,
	catalog.RecipeLadderClimb,
	catalog.RecipeBarrelRoll,
	catalog.RecipeFixedTickLoop,
}

// arcadeTopicIDs is the matching set of event-bus topic strings
// the recipes publish. Validates that recipe → topic plumbing
// doesn't drift: every arcade recipe carries an "event" arg whose
// value is one of these topics.
var arcadeTopicIDs = map[string]string{
	catalog.RecipeApplyThrust:   catalog.EventTopicMotionApplyThrust,
	catalog.RecipeRotateEntity:  catalog.EventTopicMotionRotateEntity,
	catalog.RecipeScreenWrap:    catalog.EventTopicMotionScreenWrap,
	catalog.RecipeJump:          catalog.EventTopicMotionJump,
	catalog.RecipeApplyGravity:  catalog.EventTopicMotionApplyGravity,
	catalog.RecipeSolidCollide:  catalog.EventTopicCollisionSolidCollide,
	catalog.RecipePlaceOnGrid:   catalog.EventTopicMotionPlaceOnGrid,
	catalog.RecipeExplodeRadius: catalog.EventTopicDamageExplodeRadius,
	catalog.RecipeLadderClimb:   catalog.EventTopicMotionLadderClimb,
	catalog.RecipeBarrelRoll:    catalog.EventTopicMotionBarrelRoll,
	catalog.RecipeFixedTickLoop: catalog.EventTopicFlowFixedTickLoop,
}

func TestArcadeRecipes_AllRegistered(t *testing.T) {
	for _, id := range arcadeRecipeIDs {
		_, ok := catalog.LookupRecipe(id)
		assert.True(t, ok, "arcade recipe %q must be registered", id)
	}
}

func TestArcadeRecipes_DefaultArgsPopulate(t *testing.T) {
	for _, id := range arcadeRecipeIDs {
		recipe, ok := catalog.LookupRecipe(id)
		require.True(t, ok, "missing recipe %q", id)
		assert.NotEmpty(t, recipe.DefaultArgs,
			"arcade recipe %q must carry DefaultArgs (target + event at minimum)", id)
		assert.Equal(t, catalog.EventBusTarget, recipe.DefaultArgs["target"],
			"arcade recipe %q must target the verbs.bus", id)
	}
}

func TestArcadeRecipes_TopicMatchesEventArg(t *testing.T) {
	for recipeID, topicID := range arcadeTopicIDs {
		recipe, ok := catalog.LookupRecipe(recipeID)
		require.True(t, ok, "missing recipe %q", recipeID)
		got, ok := recipe.DefaultArgs["event"].(string)
		require.True(t, ok, "arcade recipe %q has no string 'event' arg", recipeID)
		assert.Equal(t, topicID, got,
			"arcade recipe %q publishes wrong topic", recipeID)
	}
}

func TestArcadeRecipes_RelevantTriggersDeclaredForEach(t *testing.T) {
	for _, id := range arcadeRecipeIDs {
		recipe, ok := catalog.LookupRecipe(id)
		require.True(t, ok)
		assert.NotEmpty(t, recipe.RelevantTriggers,
			"arcade recipe %q must declare RelevantTriggers so it appears in some dropdown", id)
	}
}

// TestArcadeRecipes_AsteroidsShapedSheetParses asserts the four
// recipes Asteroids needs (rotate, thrust, shoot, screen-wrap)
// all resolve to valid Effects when applied through the existing
// publish_event Action — proves the verb-binding compiler will
// accept an Asteroids-shaped event sheet end-to-end.
func TestArcadeRecipes_AsteroidsShapedSheetParses(t *testing.T) {
	requireRecipeAppliesCleanly(t,
		catalog.RecipeRotateEntity,
		catalog.RecipeApplyThrust,
		catalog.RecipeSpawnEntity, // bullet
		catalog.RecipeScreenWrap,
	)
}

// TestArcadeRecipes_MarioShapedSheetParses covers the platformer
// triad (jump + gravity + solid_collide) plus a couple cross-cutting
// existing recipes (move_with_intent, give_points).
func TestArcadeRecipes_MarioShapedSheetParses(t *testing.T) {
	requireRecipeAppliesCleanly(t,
		catalog.RecipeJump,
		catalog.RecipeApplyGravity,
		catalog.RecipeSolidCollide,
		catalog.RecipeMoveWithIntent,
		catalog.RecipeGivePoints,
	)
}

// TestArcadeRecipes_BombermanShapedSheetParses covers grid-snap +
// bomb spawn + radius explosion.
func TestArcadeRecipes_BombermanShapedSheetParses(t *testing.T) {
	requireRecipeAppliesCleanly(t,
		catalog.RecipePlaceOnGrid,
		catalog.RecipeSpawnEntity,
		catalog.RecipeExplodeRadius,
		catalog.RecipeMoveWithIntent,
	)
}

// TestArcadeRecipes_DonkeyKongShapedSheetParses covers ladders +
// barrels + the platform jump.
func TestArcadeRecipes_DonkeyKongShapedSheetParses(t *testing.T) {
	requireRecipeAppliesCleanly(t,
		catalog.RecipeLadderClimb,
		catalog.RecipeBarrelRoll,
		catalog.RecipeJump,
		catalog.RecipeApplyGravity,
	)
}

// requireRecipeAppliesCleanly applies each named recipe with no
// overrides against a minimal fake Context and asserts a non-nil
// Effect comes back without error. Catches typos in ActionKind +
// arg keys at test time.
func requireRecipeAppliesCleanly(t *testing.T, recipeNames ...string) {
	t.Helper()
	ctx := &arcadeFakeCtx{}
	for _, name := range recipeNames {
		recipe, ok := catalog.LookupRecipe(name)
		require.True(t, ok, "recipe %q must be registered", name)
		effect, err := catalog.ApplyWithContext(recipe, nil, ctx)
		require.NoError(t, err, "recipe %q failed to apply", name)
		require.NotNil(t, effect, "recipe %q applied to nil Effect", name)
		assert.NotPanics(t, func() { effect(nil) },
			"recipe %q panicked when its effect was invoked", name)
	}
}

// arcadeFakeCtx satisfies catalog.Context with the bare minimum
// the publish_event Action needs at apply-time — LookupTarget for
// the verbs.bus target. Returns nil for everything else; these
// recipes' default args don't dereference samples/entities.
type arcadeFakeCtx struct{}

func (arcadeFakeCtx) Sample(string) any                  { return nil }
func (arcadeFakeCtx) Entity(string) any                  { return nil }
func (arcadeFakeCtx) ValueRef(string) catalog.ValueRef   { return nil }
func (arcadeFakeCtx) LookupTarget(name string) any {
	// Return nil — publish_event's effect early-returns on a nil
	// target lookup, which is the behaviour we want for the
	// "applies cleanly, doesn't panic" assertion. Subscriber-side
	// dispatch is covered separately in capsuleruntime tests.
	return nil
}
