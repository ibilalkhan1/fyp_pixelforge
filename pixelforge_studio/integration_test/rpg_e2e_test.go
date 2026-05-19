// rpg_e2e_test.go exercises idea #6 v1 (RPG-class systems) end-to-
// end through the public APIs of the new packages. Each test
// composes the relevant subsystems and asserts on the contract a
// designer would see — author a dialogue, bind a menu, save a
// snapshot, reload it.
package integration_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_blackboard"
	pdialogue "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_dialogue"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	pimenus "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_menus"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/dialogue"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/items"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/menus"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
)

// e2eRPGEditor returns an editor with all three RPG workspaces
// registered — what the studio's main.go composes at startup.
func e2eRPGEditor(t *testing.T) (*editor.Editor, *dialogue.Workspace, *menus.Workspace, *items.Workspace) {
	t.Helper()
	e := editor.New()
	d := dialogue.RegisterWith(e)
	m := menus.RegisterWith(e)
	i := items.RegisterWith(e)
	return e, d, m, i
}

func TestE2E_RPG_AE1_DialogueAuthoringRoundTrip(t *testing.T) {
	e, d, _, _ := e2eRPGEditor(t)
	d.NewTree("old_man_hint")
	script := `OLD MAN: Take this potion.
[[Accept -> accept]]
[[Decline -> decline]]
:: accept
OLD MAN: A wise choice.
:: decline
OLD MAN: Then begone.`
	errs := d.SetScript("old_man_hint", script)
	assert.Empty(t, errs)
	got := e.Project().Dialogues["old_man_hint"]
	assert.Contains(t, got.Script, "OLD MAN")

	// Parse the saved script and verify the renderer can walk it.
	tree, parseErrs := pdialogue.Parse(got.Script)
	require.Empty(t, parseErrs)
	r := pdialogue.NewTextBoxRenderer(tree, nil, nil)
	speaker, text := r.CurrentLine()
	assert.Equal(t, "OLD MAN", speaker)
	assert.Contains(t, text, "potion")
}

func TestE2E_RPG_AE2_MenuStackPausesUnderlyingScene(t *testing.T) {
	t.Cleanup(piloop.Resume)
	pimenus.ResetForTest()
	t.Cleanup(pimenus.ResetForTest)

	stack := pimenus.NewMenuStack()
	require.False(t, piloop.IsPaused())
	stack.Push("pause_menu", "pause", nil)
	assert.True(t, piloop.IsPaused(),
		"opening an overlay menu pauses the underlying scene (R12)")
	_, _ = stack.Pop()
	assert.False(t, piloop.IsPaused(),
		"closing the last overlay resumes the scene")
}

func TestE2E_RPG_AE3_SaveRoundTripPreservesBlackboardAndScene(t *testing.T) {
	bb := pixelforge_blackboard.New()
	bb.Set("score", 100)
	bb.Set("name", "hero")
	bb.Set("inventory", map[string]any{"items": []any{"sword", "potion"}})

	snap := pisave.Snapshot{
		SchemaVersion:  pisave.CurrentSchemaVersion,
		GameTitle:      "rpg_e2e",
		SavedAt:        time.Now(),
		Blackboard:     bb.Snapshot(),
		CurrentSceneID: "level1",
		PlayerPos:      pisave.PlayerPosition{TileX: 5, TileY: 24},
	}

	dir := t.TempDir()
	svc := pisave.NewService(pisave.NewBackendNativeAtPath(dir))
	require.NoError(t, svc.SaveToSlot(snap, pisave.Slot1Name))

	loaded, err := svc.LoadFromSlot(pisave.Slot1Name)
	require.NoError(t, err)
	assert.Equal(t, "rpg_e2e", loaded.GameTitle)
	assert.Equal(t, "level1", loaded.CurrentSceneID)
	assert.Equal(t, 5, loaded.PlayerPos.TileX)

	// Restore the blackboard from the snapshot — every key returns.
	fresh := pixelforge_blackboard.New()
	fresh.Restore(loaded.Blackboard)
	v, _ := fresh.Get("name")
	assert.Equal(t, "hero", v)
}

func TestE2E_RPG_AE4_AutosaveThrottle(t *testing.T) {
	dir := t.TempDir()
	clock := time.Now()
	clockFn := func() time.Time { return clock }
	svc := pisave.NewServiceWithThrottle(pisave.NewBackendNativeAtPath(dir), 30*time.Second, clockFn)

	wrote1, err := svc.Autosave(pisave.Snapshot{GameTitle: "t"})
	require.NoError(t, err)
	require.True(t, wrote1)

	// Advance only 5s — throttled.
	clock = clock.Add(5 * time.Second)
	wrote2, err := svc.Autosave(pisave.Snapshot{})
	require.NoError(t, err)
	assert.False(t, wrote2, "autosave throttled within window (R5)")

	// Advance past window.
	clock = clock.Add(30 * time.Second)
	wrote3, err := svc.Autosave(pisave.Snapshot{})
	require.NoError(t, err)
	assert.True(t, wrote3)
}

func TestE2E_RPG_AE5_InventoryViaBlackboard(t *testing.T) {
	bb := pixelforge_blackboard.New()
	bb.Set("inventory", []any{"sword", "potion"})

	// has_item recipe-style check: blackboard inventory contains target.
	inv, _ := bb.Get("inventory")
	items := inv.([]any)
	hasSword := false
	for _, it := range items {
		if it == "sword" {
			hasSword = true
		}
	}
	assert.True(t, hasSword)
}

func TestE2E_RPG_AE6_MenuTemplateCanonicalsAvailable(t *testing.T) {
	pimenus.ResetForTest()
	t.Cleanup(pimenus.ResetForTest)
	got := pimenus.AllTemplates()
	for _, want := range pimenus.CanonicalTemplateNames {
		assert.Contains(t, got, want, "canonical template %s available", want)
	}
}

func TestE2E_RPG_AE7_PreV1ProjectLoadsWithoutSurprises(t *testing.T) {
	// editor.pforge in the fixtures dir is the canonical pre-RPG project.
	abs, err := filepath.Abs(editorFixturePath)
	require.NoError(t, err)
	p, err := pixelforge_project.Load(abs)
	require.NoError(t, err, "pre-v1 project loads cleanly with new schema additions")
	assert.NotNil(t, p.Dialogues)
	assert.NotNil(t, p.Menus)
	assert.NotNil(t, p.Items)
	assert.True(t, p.SaveConfig.AutosaveEnabled)
}

func TestE2E_RPG_AE8_RecipesRegisteredAndDiscoverable(t *testing.T) {
	wanted := []string{
		catalog.RecipeCloseDialogue, catalog.RecipeCloseMenu,
		catalog.RecipeSetItemCount, catalog.RecipeSaveNow,
		catalog.RecipeLoadSlot, catalog.RecipeDeleteSlot,
		catalog.RecipeHasItem,
	}
	allRecipes := catalog.AllRecipes()
	for _, want := range wanted {
		assert.Contains(t, allRecipes, want)
	}
}

func TestE2E_RPG_F1_NPCDialogueFullFlow(t *testing.T) {
	e, d, _, _ := e2eRPGEditor(t)
	d.NewTree("npc")
	d.SetScript("npc", "NPC: Hello {state.name}\n[[Continue -> end]]\n:: end\nNPC: Bye")
	tree, _ := pdialogue.Parse(e.Project().Dialogues["npc"].Script)

	bb := pixelforge_blackboard.New()
	bb.Set("name", "Alice")
	r := pdialogue.NewTextBoxRenderer(tree, func(k string) (any, bool) {
		return bb.Get(k)
	}, nil)
	_, text := r.CurrentLine()
	assert.Equal(t, "Hello Alice", text)
	r.Advance()
	require.NotEmpty(t, r.CurrentChoices())
	r.PickChoice(0)
	_, text = r.CurrentLine()
	assert.Equal(t, "Bye", text)
}

func TestE2E_RPG_F2_InventoryMenuFullFlow(t *testing.T) {
	pimenus.ResetForTest()
	t.Cleanup(pimenus.ResetForTest)
	stack := pimenus.NewMenuStack()
	stack.Push("inv", "inventory", map[string]any{
		"items": []any{"potion", "sword", "shield"},
	})
	// 3 items; cursor cycles.
	assert.Equal(t, 1, stack.OnDpadDown())
	assert.Equal(t, 2, stack.OnDpadDown())
	assert.Equal(t, 0, stack.OnDpadDown(), "wraps")
}

func TestE2E_RPG_F3_SavePointFlow(t *testing.T) {
	dir := t.TempDir()
	svc := pisave.NewService(pisave.NewBackendNativeAtPath(dir))
	snap := pisave.Snapshot{GameTitle: "save_point_demo", CurrentSceneID: "village"}
	require.NoError(t, svc.SaveToSlot(snap, pisave.Slot1Name))

	loaded, err := svc.LoadFromSlot(pisave.Slot1Name)
	require.NoError(t, err)
	assert.Equal(t, "village", loaded.CurrentSceneID)

	require.NoError(t, svc.DeleteSlot(pisave.Slot1Name))
	_, err = svc.LoadFromSlot(pisave.Slot1Name)
	require.Error(t, err, "deleted slot no longer loads")
}

func TestE2E_RPG_F4_TitleScreenMenuFullFlow(t *testing.T) {
	pimenus.ResetForTest()
	t.Cleanup(pimenus.ResetForTest)
	stack := pimenus.NewMenuStack()
	stack.Push("title", "title", nil)
	// Cursor at 0 = Start; OnUse dispatches scene_change.
	verb, args, ok := stack.OnUse()
	require.True(t, ok)
	assert.Equal(t, "scene_change", verb)
	assert.Equal(t, "level1", args["target"])
}

func TestE2E_RPG_ItemsRoundTripViaWorkspace(t *testing.T) {
	e, _, _, i := e2eRPGEditor(t)
	require.True(t, i.NewItem("potion"))
	require.True(t, i.SetItemField("potion", "name", "Potion"))
	require.True(t, i.SetItemField("potion", "category", "consumable"))
	require.Len(t, e.Project().Items, 1)
	got := e.Project().Items[0]
	assert.Equal(t, "Potion", got.Name)
	assert.Equal(t, "consumable", got.Category)
}
