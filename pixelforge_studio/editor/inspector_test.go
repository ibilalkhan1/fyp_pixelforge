package editor

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// The component registry is a singleton; serialize tests that mutate it.
var registryMu sync.Mutex

func withRegistry(t *testing.T) {
	t.Helper()
	registryMu.Lock()
	t.Cleanup(func() {
		pfcomponent.ResetForTest()
		registryMu.Unlock()
	})
	pfcomponent.ResetForTest()
}

// buildWidgetContext snapshots the project's catalogs in alpha order.
func TestBuildWidgetContext(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Sprites = []pixelforge_project.SpriteAsset{
		{Name: "zebra"}, {Name: "alpha"},
	}
	p.Audio = []pixelforge_project.AudioSample{
		{Name: "delta"}, {Name: "bravo"},
	}
	p.EventSubscriptions = []pixelforge_project.EventSubscription{
		{Topic: "game/Hit"},
		{Topic: "game/Death"},
	}
	ctx := buildWidgetContext(p)
	assert.Equal(t, []string{"alpha", "zebra"}, ctx.SpriteNames)
	assert.Equal(t, []string{"bravo", "delta"}, ctx.AudioNames)
	assert.Equal(t, []string{"game/Death", "game/Hit"}, ctx.EventTopics)
	assert.Len(t, ctx.PaletteColors, pixelforge_project.MaxColors)
}

// buildWidgetContext is safe on nil project.
func TestBuildWidgetContext_NilProject(t *testing.T) {
	ctx := buildWidgetContext(nil)
	assert.Empty(t, ctx.SpriteNames)
	assert.Empty(t, ctx.AudioNames)
}

// Selecting an entity surfaces its components for the inspector to
// render. The Editor's selectedEntity() helper locates the entity by ID.
func TestEditor_SelectedEntityResolution(t *testing.T) {
	e := New()
	p := pixelforge_project.NewProject("t")
	p.Scenes[0].Entities = []pixelforge_project.Entity{
		{ID: "player", Name: "Player"},
		{ID: "enemy", Name: "Enemy"},
	}
	e.SetProject(p)

	e.SelectEntity("player")
	got := e.selectedEntity()
	require.NotNil(t, got)
	assert.Equal(t, "player", got.ID)

	// Unknown selection → nil
	e.SelectEntity("ghost")
	assert.Nil(t, e.selectedEntity())

	// Empty → nil
	e.SelectEntity("")
	assert.Nil(t, e.selectedEntity())
}

// Inspector caches widgets per (entity, component, field) so dragging
// state survives between frames.
func TestInspector_WidgetCache(t *testing.T) {
	withRegistry(t)
	type Player struct {
		Speed float64 `pf:"slider,0..10"`
	}
	pfcomponent.Register[Player]("Player")

	insp := NewInspector()
	field, _ := pfcomponent.Get("Player")
	require.Len(t, field.Fields, 1)

	w1 := insp.widget("ent-1", 0, 0, field.Fields[0], buildWidgetContext(nil))
	w2 := insp.widget("ent-1", 0, 0, field.Fields[0], buildWidgetContext(nil))
	assert.Same(t, w1, w2, "same coordinates → same widget instance (state persistence)")

	w3 := insp.widget("ent-2", 0, 0, field.Fields[0], buildWidgetContext(nil))
	assert.NotSame(t, w1, w3, "different entity → different widget")
}

// Dirty flag flips on edit; MarkDirty is idempotent.
func TestEditor_DirtyFlag(t *testing.T) {
	e := New()
	assert.False(t, e.IsDirty())
	e.MarkDirty()
	assert.True(t, e.IsDirty())
	e.MarkDirty()
	assert.True(t, e.IsDirty())

	// Setting a fresh project resets dirty.
	e.SetProject(pixelforge_project.NewProject("fresh"))
	assert.False(t, e.IsDirty())
}

// Unregistered components don't crash the inspector — they render
// an "unknown" header. We verify the lookup helper returns ok=false.
func TestInspector_UnknownComponentSurvives(t *testing.T) {
	withRegistry(t)
	_, ok := pfcomponent.Get("NotRegistered")
	assert.False(t, ok)
	// Inspector.Draw on such a component would render the unknown
	// header; we cannot exercise the *ebiten.Image path here, but the
	// lookup branch is what protects us.
}
