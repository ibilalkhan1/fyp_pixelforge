package items_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/items"
)

func newItemsEditor(t *testing.T) (*editor.Editor, *items.Workspace) {
	t.Helper()
	e := editor.New()
	w := items.RegisterWith(e)
	return e, w
}

func TestWorkspace_RegistersWithEditor(t *testing.T) {
	e, w := newItemsEditor(t)
	assert.Equal(t, "items", w.Name())
	assert.Equal(t, "Items", w.DisplayName())
	found := false
	for _, ws := range e.Workspaces() {
		if ws.Name() == "items" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestWorkspace_NewItemAppendsAndMarksDirty(t *testing.T) {
	e, w := newItemsEditor(t)
	require.True(t, w.NewItem("potion"))
	assert.Len(t, e.Project().Items, 1)
	assert.Equal(t, "potion", e.Project().Items[0].ID)
	assert.True(t, e.IsDirty())
}

func TestWorkspace_NewItemDuplicateRejects(t *testing.T) {
	_, w := newItemsEditor(t)
	w.NewItem("potion")
	assert.False(t, w.NewItem("potion"))
}

func TestWorkspace_NewItemEmptyIDRejects(t *testing.T) {
	_, w := newItemsEditor(t)
	assert.False(t, w.NewItem(""))
}

func TestWorkspace_DeleteItemRemovesEntry(t *testing.T) {
	e, w := newItemsEditor(t)
	w.NewItem("doomed")
	require.True(t, w.DeleteItem("doomed"))
	assert.Empty(t, e.Project().Items)
}

func TestWorkspace_SetItemFieldUpdatesValues(t *testing.T) {
	e, w := newItemsEditor(t)
	w.NewItem("potion")
	require.True(t, w.SetItemField("potion", "name", "Potion"))
	require.True(t, w.SetItemField("potion", "category", "consumable"))
	require.True(t, w.SetItemField("potion", "effect_verb", "restore_health(50)"))
	require.True(t, w.SetItemField("potion", "icon", "potion_sprite"))
	require.True(t, w.SetItemField("potion", "description", "Heals 50 HP"))
	it := e.Project().Items[0]
	assert.Equal(t, "Potion", it.Name)
	assert.Equal(t, "consumable", it.Category)
	assert.Equal(t, "restore_health(50)", it.EffectVerb)
	assert.Equal(t, "potion_sprite", it.Icon)
	assert.Equal(t, "Heals 50 HP", it.Description)
}

func TestWorkspace_SetItemFieldUnknownFieldRejects(t *testing.T) {
	_, w := newItemsEditor(t)
	w.NewItem("potion")
	assert.False(t, w.SetItemField("potion", "nonsense", "x"))
}

func TestWorkspace_SetItemFieldIdempotentReturnsFalse(t *testing.T) {
	_, w := newItemsEditor(t)
	w.NewItem("potion")
	w.SetItemField("potion", "name", "Potion")
	assert.False(t, w.SetItemField("potion", "name", "Potion"))
}

func TestWorkspace_SortedItemIDs(t *testing.T) {
	_, w := newItemsEditor(t)
	w.NewItem("zebra")
	w.NewItem("alpha")
	w.NewItem("mango")
	assert.Equal(t, []string{"alpha", "mango", "zebra"}, w.SortedItemIDs())
}

func TestWorkspace_AvailableSpriteNamesEnumeratesProject(t *testing.T) {
	e, w := newItemsEditor(t)
	e.Project().Sprites = []pixelforge_project.SpriteAsset{
		{Name: "potion"}, {Name: "sword"}, {Name: "shield"},
	}
	got := w.AvailableSpriteNames()
	assert.ElementsMatch(t, []string{"potion", "shield", "sword"}, got)
}
