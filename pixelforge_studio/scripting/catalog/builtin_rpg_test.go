package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
)

// builtin_rpg_test.go covers idea #6 v1 U10's RPG recipe additions
// + the new Recipe.ConditionKind field.

func TestRPGRecipes_AllRegistered(t *testing.T) {
	wanted := []string{
		catalog.RecipeCloseDialogue,
		catalog.RecipeCloseMenu,
		catalog.RecipeSetItemCount,
		catalog.RecipeSaveNow,
		catalog.RecipeLoadSlot,
		catalog.RecipeDeleteSlot,
		catalog.RecipeHasItem,
	}
	for _, name := range wanted {
		_, ok := catalog.LookupRecipe(name)
		assert.True(t, ok, "recipe %q registered", name)
	}
}

func TestRPGRecipes_HasItemIsCondition(t *testing.T) {
	r, ok := catalog.LookupRecipe(catalog.RecipeHasItem)
	require.True(t, ok)
	assert.True(t, r.IsCondition())
	assert.Equal(t, "has_item", r.ConditionKind)
	assert.Empty(t, r.ActionKind,
		"condition recipes leave ActionKind empty per the mutually-exclusive contract")
}

func TestRPGRecipes_ActionRecipesPublishWellKnownTopics(t *testing.T) {
	cases := []struct {
		recipe string
		topic  string
	}{
		{catalog.RecipeCloseDialogue, catalog.EventTopicUICloseDialogue},
		{catalog.RecipeCloseMenu, catalog.EventTopicUICloseMenu},
		{catalog.RecipeSetItemCount, catalog.EventTopicInventorySetItemCount},
		{catalog.RecipeSaveNow, catalog.EventTopicSaveNow},
		{catalog.RecipeLoadSlot, catalog.EventTopicSaveLoad},
		{catalog.RecipeDeleteSlot, catalog.EventTopicSaveDelete},
	}
	for _, tc := range cases {
		r, ok := catalog.LookupRecipe(tc.recipe)
		require.True(t, ok)
		assert.Equal(t, "publish_event", r.ActionKind,
			"%s wraps publish_event as the placeholder action", tc.recipe)
		assert.Equal(t, tc.topic, r.DefaultArgs["event"],
			"%s publishes the canonical topic", tc.recipe)
	}
}

func TestRPGRecipes_SaveSlotsDefaultToSlot1(t *testing.T) {
	for _, name := range []string{catalog.RecipeSaveNow, catalog.RecipeLoadSlot, catalog.RecipeDeleteSlot} {
		r, ok := catalog.LookupRecipe(name)
		require.True(t, ok)
		assert.Equal(t, "slot1", r.DefaultArgs["slot"],
			"%s defaults to slot1; designer overrides per binding", name)
	}
}

func TestRPGRecipes_HasRelevantTriggers(t *testing.T) {
	for _, name := range []string{
		catalog.RecipeCloseDialogue, catalog.RecipeCloseMenu,
		catalog.RecipeSetItemCount, catalog.RecipeSaveNow,
		catalog.RecipeLoadSlot, catalog.RecipeDeleteSlot,
		catalog.RecipeHasItem,
	} {
		r, ok := catalog.LookupRecipe(name)
		require.True(t, ok)
		assert.NotEmpty(t, r.RelevantTriggers,
			"recipe %q must declare at least one relevant trigger so the dropdown surfaces it", name)
	}
}

func TestRecipe_IsConditionDistinguishesKinds(t *testing.T) {
	action := catalog.Recipe{ActionKind: "publish_event"}
	condition := catalog.Recipe{ConditionKind: "has_item"}
	assert.False(t, action.IsCondition())
	assert.True(t, condition.IsCondition())
}
