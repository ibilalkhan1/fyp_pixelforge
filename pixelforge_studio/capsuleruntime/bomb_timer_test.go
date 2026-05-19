package capsuleruntime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// bombTimerProject builds a Bomberman-preset fixture with a single
// "bomb_a" entity carrying a BombTimer component. Tests tune the
// initial ticks_remaining + grid coords per scenario before calling
// AdvanceTick.
func bombTimerProject(initialTicks, rng, gx, gy int) *pixelforge_project.Project {
	p := pixelforge_project.NewProject("bomb-timer-test")
	p.PhysicsPreset = "bomberman"
	p.Scenes = []pixelforge_project.Scene{
		{
			ID: "main", Name: "Main",
			Entities: []pixelforge_project.Entity{
				{
					ID:   "bomb_a",
					Name: "BombA",
					Position: pixelforge_project.EntityPosition{
						X: float64(gx * 16),
						Y: float64(gy * 16),
					},
					Components: []pixelforge_project.EntityComponent{
						{
							Type: pixelforge_project.BombTimerComponentType,
							Values: map[string]any{
								"ticks_remaining": float64(initialTicks),
								"range":           float64(rng),
								"grid_x":          float64(gx),
								"grid_y":          float64(gy),
							},
						},
					},
				},
			},
		},
	}
	return p
}

func TestBombTimer_DecrementsEachTick(t *testing.T) {
	rt := bootDefault(t, bombTimerProject(5, 2, 1, 1))

	for i := 0; i < 3; i++ {
		rt.AdvanceTick()
	}

	// 5 → 4 → 3 → 2: should still be alive on the scene.
	require.Len(t, rt.CurrentScene.Entities, 1)
	bt, ok := pixelforge_project.BombTimerFor(&rt.CurrentScene.Entities[0])
	require.True(t, ok)
	assert.Equal(t, 2, bt.TicksRemaining, "ticks_remaining should be 5 - 3 = 2 after 3 AdvanceTick calls")
}

func TestBombTimer_FiresAtZero(t *testing.T) {
	rt := bootDefault(t, bombTimerProject(3, 2, 5, 5))
	rec := newBusRecorder()
	rec.attach()

	// Two ticks shouldn't fire yet (3 → 2 → 1).
	rt.AdvanceTick()
	rt.AdvanceTick()
	assert.Empty(t, rec.filterTopic("damage/grid_explode"), "no fire expected before timer reaches zero")
	assert.Empty(t, rec.filterTopic("spawn/destroy_other"), "no destroy expected before timer reaches zero")

	// Third tick should fire (1 → 0).
	rt.AdvanceTick()

	dies := rec.filterTopic("spawn/destroy_other")
	require.Len(t, dies, 1, "bomb must be destroyed exactly once at timer expiry")
	assert.Equal(t, "bomb_a", dies[0].Args["entity"])

	explodes := rec.filterTopic("damage/grid_explode")
	require.Len(t, explodes, 1, "grid_explode must fire exactly once at timer expiry")
	// Origin coords must come from the BombTimer component, not from
	// the live body position — the timer carries the canonical
	// origin so a recipe that moved the bomb without updating the
	// timer still detonates from the original cell.
	origin, ok := explodes[0].Args["origin"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 5, origin["gx"])
	assert.Equal(t, 5, origin["gy"])
	assert.Equal(t, float64(2), explodes[0].Args["range"])
}

func TestBombTimer_DestroysBombBeforeExplosion(t *testing.T) {
	rt := bootDefault(t, bombTimerProject(1, 2, 3, 3))
	rec := newBusRecorder()
	rec.attach()

	rt.AdvanceTick()

	// The destroy_other event must arrive before grid_explode so a
	// blast that lands on the origin cell doesn't see its own
	// authoring entity. Find each topic's index and compare.
	topics := rec.topics()
	destroyIdx := -1
	explodeIdx := -1
	for i, top := range topics {
		if top == "spawn/destroy_other" && destroyIdx == -1 {
			destroyIdx = i
		}
		if top == "damage/grid_explode" && explodeIdx == -1 {
			explodeIdx = i
		}
	}
	require.NotEqual(t, -1, destroyIdx, "spawn/destroy_other must be published")
	require.NotEqual(t, -1, explodeIdx, "damage/grid_explode must be published")
	assert.Less(t, destroyIdx, explodeIdx, "destroy_other must precede grid_explode")

	// The scene's bomb entity must be gone by the time the explode
	// processes (the spawn sink removes the entity synchronously on
	// destroy_other).
	for _, e := range rt.CurrentScene.Entities {
		assert.NotEqual(t, "bomb_a", e.ID, "bomb must be removed from scene after detonation")
	}
}

func TestBombTimer_CancelledByRemoveComponent(t *testing.T) {
	rt := bootDefault(t, bombTimerProject(5, 2, 1, 1))
	rec := newBusRecorder()
	rec.attach()

	// Tick once (5 → 4), then defuse by stripping the BombTimer
	// component from the entity. The next ticks should NOT fire.
	rt.AdvanceTick()
	require.Len(t, rt.CurrentScene.Entities, 1)
	ok := pixelforge_project.RemoveBombTimerFor(&rt.CurrentScene.Entities[0])
	require.True(t, ok)

	for i := 0; i < 10; i++ {
		rt.AdvanceTick()
	}

	assert.Empty(t, rec.filterTopic("damage/grid_explode"), "defused bomb must not explode")
	assert.Empty(t, rec.filterTopic("spawn/destroy_other"), "defused bomb must not be destroyed by the timer system")
	// Entity itself stays on the scene (defuse, not removal).
	require.Len(t, rt.CurrentScene.Entities, 1)
	assert.Equal(t, "bomb_a", rt.CurrentScene.Entities[0].ID)
}

func TestBombTimer_CancelledByEntityRemoval(t *testing.T) {
	rt := bootDefault(t, bombTimerProject(5, 2, 1, 1))
	rec := newBusRecorder()
	rec.attach()

	// Destroy the bomb outright (via the spawn sink). The next
	// AdvanceTick scans should find no bomb entity → no fire.
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "spawn/destroy_other",
		Args:  map[string]any{"entity": "bomb_a"},
	})
	require.Empty(t, rt.CurrentScene.Entities, "destroy_other should remove the bomb")

	// Reset the recorder so we ignore the manual destroy we just
	// fired; we only care about post-cancel events.
	preCount := len(rec.filterTopic("damage/grid_explode"))

	for i := 0; i < 10; i++ {
		rt.AdvanceTick()
	}

	postCount := len(rec.filterTopic("damage/grid_explode"))
	assert.Equal(t, preCount, postCount, "manually-destroyed bomb must not fire grid_explode")
}

func TestBombTimer_AdvanceTickIsDeterministic(t *testing.T) {
	// Same fixture, same tick count, same publish sequence — twice.
	// Equality on the topic list (rec1 vs rec2) is the determinism
	// guarantee plan-009 requires for replay.
	const ticks = 5

	rtA := bootDefault(t, bombTimerProject(3, 2, 4, 4))
	recA := newBusRecorder()
	recA.attach()
	for i := 0; i < ticks; i++ {
		rtA.AdvanceTick()
	}
	topicsA := recA.topics()

	rtB := bootDefault(t, bombTimerProject(3, 2, 4, 4))
	recB := newBusRecorder()
	recB.attach()
	for i := 0; i < ticks; i++ {
		rtB.AdvanceTick()
	}
	topicsB := recB.topics()

	assert.Equal(t, topicsA, topicsB, "two identical runs must publish identical topic sequences")
}

func TestBombTimer_PlaceOnGridIntegration(t *testing.T) {
	// place_on_grid before the timer fires; verify the blast emerges
	// from the timer's stored grid position (not from the body's
	// pixel position, which can be different).
	rt := bootDefault(t, bombTimerProject(2, 3, 5, 5))
	rec := newBusRecorder()
	rec.attach()

	// First, place the bomb at (5, 5) via the verb. This is the
	// canonical Bomberman flow: bomb spawns with a BombTimer, then
	// place_on_grid snaps its body to the cell origin so the
	// renderer draws it pixel-perfect.
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/place_on_grid",
		Args:  map[string]any{"entity": "bomb_a", "grid_x": float64(5), "grid_y": float64(5)},
	})

	// Now advance until the timer fires.
	rt.AdvanceTick()
	rt.AdvanceTick()

	explodes := rec.filterTopic("damage/grid_explode")
	require.Len(t, explodes, 1)
	origin := explodes[0].Args["origin"].(map[string]any)
	assert.Equal(t, 5, origin["gx"])
	assert.Equal(t, 5, origin["gy"])
}

func TestBombTimer_NoTimerNoEffect(t *testing.T) {
	// A bomb entity that has no BombTimer component must never fire.
	p := pixelforge_project.NewProject("no-timer-test")
	p.PhysicsPreset = "bomberman"
	p.Scenes = []pixelforge_project.Scene{
		{
			ID: "main", Name: "Main",
			Entities: []pixelforge_project.Entity{
				{ID: "decoration", Name: "Decoration"},
			},
		},
	}
	rt := bootDefault(t, p)
	rec := newBusRecorder()
	rec.attach()

	for i := 0; i < 50; i++ {
		rt.AdvanceTick()
	}

	assert.Empty(t, rec.filterTopic("damage/grid_explode"))
	assert.Empty(t, rec.filterTopic("spawn/destroy_other"))
}

func TestBombTimer_MultipleBombsFireIndependently(t *testing.T) {
	// Two bombs with different fuse lengths; verify each fires on
	// its own schedule + each carries its own origin.
	p := pixelforge_project.NewProject("multi-bomb-test")
	p.PhysicsPreset = "bomberman"
	p.Scenes = []pixelforge_project.Scene{
		{
			ID: "main", Name: "Main",
			Entities: []pixelforge_project.Entity{
				{
					ID:   "bomb_short",
					Name: "BombShort",
					Components: []pixelforge_project.EntityComponent{
						{
							Type: pixelforge_project.BombTimerComponentType,
							Values: map[string]any{
								"ticks_remaining": float64(2),
								"range":           float64(2),
								"grid_x":          float64(2),
								"grid_y":          float64(2),
							},
						},
					},
				},
				{
					ID:   "bomb_long",
					Name: "BombLong",
					Components: []pixelforge_project.EntityComponent{
						{
							Type: pixelforge_project.BombTimerComponentType,
							Values: map[string]any{
								"ticks_remaining": float64(5),
								"range":           float64(3),
								"grid_x":          float64(7),
								"grid_y":          float64(7),
							},
						},
					},
				},
			},
		},
	}
	rt := bootDefault(t, p)
	rec := newBusRecorder()
	rec.attach()

	// After 2 ticks, only bomb_short should have fired.
	rt.AdvanceTick()
	rt.AdvanceTick()
	explodes := rec.filterTopic("damage/grid_explode")
	require.Len(t, explodes, 1, "only the short-fuse bomb should have fired after 2 ticks")
	originShort := explodes[0].Args["origin"].(map[string]any)
	assert.Equal(t, 2, originShort["gx"])
	assert.Equal(t, 2, originShort["gy"])

	// After 3 more ticks (5 total), bomb_long should also have fired.
	rt.AdvanceTick()
	rt.AdvanceTick()
	rt.AdvanceTick()
	explodes = rec.filterTopic("damage/grid_explode")
	require.Len(t, explodes, 2, "both bombs should have fired by tick 5")
	originLong := explodes[1].Args["origin"].(map[string]any)
	assert.Equal(t, 7, originLong["gx"])
	assert.Equal(t, 7, originLong["gy"])
}
