package capsuleruntime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

func projectWithHero() *pixelforge_project.Project {
	p := pixelforge_project.NewProject("visual-sink-test")
	p.Scenes = []pixelforge_project.Scene{
		{ID: "main", Name: "Main", Entities: []pixelforge_project.Entity{
			{ID: "hero", Name: "Hero"},
		}},
	}
	return p
}

func TestVisualSink_HideShowToggle(t *testing.T) {
	rt := bootDefault(t, projectWithHero())

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "visual/hide",
		Args:  map[string]any{"entity": "hero"},
	})
	require.NotNil(t, rt.VisualEffects["hero"])
	assert.True(t, rt.VisualEffects["hero"].Hidden)

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "visual/show",
		Args:  map[string]any{"entity": "hero"},
	})
	// After Show with no other active overrides, the entry is removed.
	_, ok := rt.VisualEffects["hero"]
	assert.False(t, ok, "Show should clear the empty effect record")
}

func TestVisualSink_FlashExpiresAfterTicks(t *testing.T) {
	rt := bootDefault(t, projectWithHero())
	// Project default TPS = 30; 100ms at 30 TPS = ceil(100*30/1000) = 3 ticks.
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "visual/flash",
		Args:  map[string]any{"entity": "hero", "color": "#ff0000", "ms": float64(100)},
	})
	require.NotNil(t, rt.VisualEffects["hero"])
	assert.Equal(t, "#ff0000", rt.VisualEffects["hero"].FlashColor)

	tps := rt.Project.TPS
	flashTicks := (100*tps + 999) / 1000

	// Advance up to one tick before expiry: still set.
	for i := 0; i < flashTicks-1; i++ {
		rt.AdvanceTick()
	}
	require.NotNil(t, rt.VisualEffects["hero"])
	assert.Equal(t, "#ff0000", rt.VisualEffects["hero"].FlashColor,
		"flash should still be active before expiry")

	// Advance one more: the flash expires and the (empty) record is GC'd.
	rt.AdvanceTick()
	_, ok := rt.VisualEffects["hero"]
	assert.False(t, ok, "expired flash with no other overrides should be removed")
}

func TestVisualSink_FlashAt60TPS(t *testing.T) {
	p := projectWithHero()
	p.TPS = 60
	rt := bootDefault(t, p)
	// 100ms at 60 TPS = 6 ticks.
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "visual/flash",
		Args:  map[string]any{"entity": "hero", "color": "#ff0000", "ms": float64(100)},
	})
	// At tick 5: still set.
	for i := 0; i < 5; i++ {
		rt.AdvanceTick()
	}
	require.NotNil(t, rt.VisualEffects["hero"])
	assert.Equal(t, "#ff0000", rt.VisualEffects["hero"].FlashColor)
	// Advance to tick 6: expires.
	rt.AdvanceTick()
	_, ok := rt.VisualEffects["hero"]
	assert.False(t, ok)
}

func TestVisualSink_SwapSpriteSetsOverride(t *testing.T) {
	rt := bootDefault(t, projectWithHero())
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "visual/swap_sprite",
		Args:  map[string]any{"entity": "hero", "name": "hero_hurt"},
	})
	require.NotNil(t, rt.VisualEffects["hero"])
	assert.Equal(t, "hero_hurt", rt.VisualEffects["hero"].SpriteSwap)

	// Clearing by sending an empty name removes the effect.
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "visual/swap_sprite",
		Args:  map[string]any{"entity": "hero", "name": ""},
	})
	_, ok := rt.VisualEffects["hero"]
	assert.False(t, ok)
}

func TestVisualSink_MissingEntityNoCrash(t *testing.T) {
	rt := bootDefault(t, projectWithHero())
	assert.NotPanics(t, func() {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "visual/hide",
			Args:  map[string]any{},
		})
	})
	assert.Empty(t, rt.VisualEffects)
}
