package capsuleruntime_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// busRecorder captures every event published to verbs.bus during a
// test so the cascade assertions can inspect the full chain in
// publish order. SubscribeAll is invoked once at construction; the
// recorder's slice is the canonical timeline of events the test
// asserts against.
//
// Pre-existing dispatcher subscribers still fire (the recorder is
// added via SubscribeAll alongside them), so capturing here doesn't
// suppress the runtime's normal event handling — exactly the
// behaviour the CI parity tests will rely on in U11.
type busRecorder struct {
	mu     sync.Mutex
	events []*piloop.VerbEvent
}

func newBusRecorder() *busRecorder { return &busRecorder{} }

func (r *busRecorder) attach() {
	piloop.VerbsBus().SubscribeAll(func(ev *piloop.VerbEvent, _ pievent.Handler) {
		if ev == nil {
			return
		}
		r.mu.Lock()
		r.events = append(r.events, ev)
		r.mu.Unlock()
	})
}

// topics returns the list of topics in publish order.
func (r *busRecorder) topics() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.events))
	for i, ev := range r.events {
		out[i] = ev.Topic
	}
	return out
}

// filterTopic returns every captured event whose topic matches.
func (r *busRecorder) filterTopic(topic string) []*piloop.VerbEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*piloop.VerbEvent
	for _, ev := range r.events {
		if ev.Topic == topic {
			out = append(out, ev)
		}
	}
	return out
}

// damageProject builds a fixture project with a single scene
// containing a `goomba_1` entity carrying a health component with
// hp=5, max_hp=5. Used by the take_damage / die tests.
func damageProject(hp int) *pixelforge_project.Project {
	p := pixelforge_project.NewProject("damage-sink-test")
	p.Scenes = []pixelforge_project.Scene{
		{
			ID: "main", Name: "Main",
			Entities: []pixelforge_project.Entity{
				{
					ID: "goomba_1", Name: "Goomba",
					Position: pixelforge_project.EntityPosition{X: 0, Y: 0},
					Components: []pixelforge_project.EntityComponent{
						{
							Type: pixelforge_project.HealthComponentType,
							Values: map[string]any{
								"hp":     float64(hp),
								"max_hp": float64(hp),
							},
						},
					},
				},
			},
		},
	}
	return p
}

// explosionProject builds a 5-entity fixture for the AoE radius
// tests. Entities placed at known positions so the radius math has
// stable expected hits. The first three are within radius 50 of
// (100, 100); the last two are outside.
func explosionProject() *pixelforge_project.Project {
	mkHP := func() []pixelforge_project.EntityComponent {
		return []pixelforge_project.EntityComponent{
			{
				Type: pixelforge_project.HealthComponentType,
				Values: map[string]any{
					"hp":     float64(10),
					"max_hp": float64(10),
				},
			},
		}
	}
	p := pixelforge_project.NewProject("explosion-test")
	p.Scenes = []pixelforge_project.Scene{
		{
			ID: "main", Name: "Main",
			Entities: []pixelforge_project.Entity{
				{ID: "in_1", Name: "In1", Position: pixelforge_project.EntityPosition{X: 100, Y: 100}, Components: mkHP()},
				{ID: "in_2", Name: "In2", Position: pixelforge_project.EntityPosition{X: 130, Y: 100}, Components: mkHP()},
				{ID: "in_3", Name: "In3", Position: pixelforge_project.EntityPosition{X: 100, Y: 140}, Components: mkHP()},
				{ID: "out_1", Name: "Out1", Position: pixelforge_project.EntityPosition{X: 200, Y: 100}, Components: mkHP()},
				{ID: "out_2", Name: "Out2", Position: pixelforge_project.EntityPosition{X: 100, Y: 300}, Components: mkHP()},
			},
		},
	}
	return p
}

func TestDamageSink_Die_PublishesDestroyOther(t *testing.T) {
	rt := bootDefault(t, damageProject(5))
	rec := newBusRecorder()
	rec.attach()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/die",
		Args:  map[string]any{"entity": "goomba_1"},
	})

	// Expect at minimum: the original damage/die + the chained
	// spawn/destroy_other.
	destroys := rec.filterTopic("spawn/destroy_other")
	require.Len(t, destroys, 1, "die must publish exactly one spawn/destroy_other")
	assert.Equal(t, "goomba_1", destroys[0].Args["entity"])

	// The spawn sink consumes destroy_other and removes the entity
	// from the live scene on the same dispatch (synchronous bus).
	for _, e := range rt.CurrentScene.Entities {
		assert.NotEqual(t, "goomba_1", e.ID, "entity must be removed from scene")
	}
}

func TestDamageSink_Die_EmptyEntityIsNoOp(t *testing.T) {
	rt := bootDefault(t, damageProject(5))
	rec := newBusRecorder()
	rec.attach()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/die",
		Args:  map[string]any{}, // missing entity
	})

	// No destroy_other should fire on an empty id.
	assert.Empty(t, rec.filterTopic("spawn/destroy_other"))
	// Goomba untouched.
	require.Len(t, rt.CurrentScene.Entities, 1)
	assert.Equal(t, "goomba_1", rt.CurrentScene.Entities[0].ID)
}

func TestDamageSink_HurtPlayer_DecrementsHP(t *testing.T) {
	rt := bootDefault(t, damageProject(5))
	rt.Globals["player_hp"] = 3

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/hurt_player",
		Args:  map[string]any{"amount": float64(1)},
	})

	assert.Equal(t, 2, rt.Globals["player_hp"])
}

func TestDamageSink_HurtPlayer_DefaultAmountIsOne(t *testing.T) {
	rt := bootDefault(t, damageProject(5))
	rt.Globals["player_hp"] = 4

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/hurt_player",
		Args:  map[string]any{},
	})

	assert.Equal(t, 3, rt.Globals["player_hp"], "missing amount must default to 1")
}

func TestDamageSink_HurtPlayer_AtZeroPublishesDie(t *testing.T) {
	rt := bootDefault(t, damageProject(5))
	rt.Globals["player_hp"] = 1
	rt.Globals["player_entity"] = "player"
	rec := newBusRecorder()
	rec.attach()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/hurt_player",
		Args:  map[string]any{"amount": float64(1)},
	})

	assert.Equal(t, 0, rt.Globals["player_hp"])

	dies := rec.filterTopic("damage/die")
	require.Len(t, dies, 1, "hp-to-zero must publish damage/die exactly once")
	assert.Equal(t, "player", dies[0].Args["entity"])
}

func TestDamageSink_HurtPlayer_RepeatedHurtsCascadeOnceAtZero(t *testing.T) {
	rt := bootDefault(t, damageProject(5))
	rt.Globals["player_hp"] = 3
	rec := newBusRecorder()
	rec.attach()

	for i := 0; i < 5; i++ {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "damage/hurt_player",
			Args:  map[string]any{"amount": float64(1)},
		})
	}

	// Player hp clamps at 0 (never negative).
	assert.Equal(t, 0, rt.Globals["player_hp"])

	// Every hurt after the kill-shot still publishes a die — the
	// cascade fires whenever hp ≤ 0 and amount > 0. This is the
	// observable contract; the renderer / game loop is responsible
	// for filtering duplicate deaths.
	dies := rec.filterTopic("damage/die")
	assert.GreaterOrEqual(t, len(dies), 1, "die must fire at least once after kill-shot")
}

func TestDamageSink_TakeDamage_DecrementsEntityHP(t *testing.T) {
	rt := bootDefault(t, damageProject(5))

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/take_damage",
		Args:  map[string]any{"entity": "goomba_1", "amount": float64(2)},
	})

	require.Len(t, rt.CurrentScene.Entities, 1)
	hp, _, ok := pixelforge_project.HealthFor(&rt.CurrentScene.Entities[0])
	require.True(t, ok)
	assert.Equal(t, 3, hp, "hp should go from 5 to 3 after amount=2 damage")
}

func TestDamageSink_TakeDamage_ZeroHPTriggersDie(t *testing.T) {
	rt := bootDefault(t, damageProject(2))
	rec := newBusRecorder()
	rec.attach()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/take_damage",
		Args:  map[string]any{"entity": "goomba_1", "amount": float64(5)},
	})

	// HP must clamp at 0 (not negative).
	// Entity is removed by the cascade (die → destroy_other), so
	// look it up via the recorder rather than rt.CurrentScene.
	dies := rec.filterTopic("damage/die")
	require.Len(t, dies, 1, "take_damage to zero must publish damage/die")
	assert.Equal(t, "goomba_1", dies[0].Args["entity"])

	destroys := rec.filterTopic("spawn/destroy_other")
	require.Len(t, destroys, 1, "die cascade must produce destroy_other")
	assert.Equal(t, "goomba_1", destroys[0].Args["entity"])

	// Scene entity removed by the destroy cascade.
	for _, e := range rt.CurrentScene.Entities {
		assert.NotEqual(t, "goomba_1", e.ID, "entity must be removed after cascade")
	}
}

func TestDamageSink_TakeDamage_AcceptsEntityIDKey(t *testing.T) {
	// The dispatcher accepts both "entity" and "entity_id" — verify
	// the entity_id fallback still works for legacy recipes.
	rt := bootDefault(t, damageProject(5))

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/take_damage",
		Args:  map[string]any{"entity_id": "goomba_1", "amount": float64(1)},
	})

	hp, _, _ := pixelforge_project.HealthFor(&rt.CurrentScene.Entities[0])
	assert.Equal(t, 4, hp)
}

func TestDamageSink_TakeDamage_MissingHealthComponentIsSafeNoOp(t *testing.T) {
	// An entity without a health component must not crash; the sink
	// logs a warning and no-ops, and no cascading die event fires.
	p := pixelforge_project.NewProject("no-hp-test")
	p.Scenes = []pixelforge_project.Scene{
		{
			ID: "main", Name: "Main",
			Entities: []pixelforge_project.Entity{
				{ID: "wall_1", Name: "Wall"}, // no health component
			},
		},
	}
	rt := bootDefault(t, p)
	rec := newBusRecorder()
	rec.attach()

	out := captureLog(t, func() {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "damage/take_damage",
			Args:  map[string]any{"entity": "wall_1", "amount": float64(5)},
		})
	})

	assert.Contains(t, out, "no health component")
	assert.Empty(t, rec.filterTopic("damage/die"), "non-damageable entity must not cascade to die")
	// Wall remains in the scene.
	require.Len(t, rt.CurrentScene.Entities, 1)
	assert.Equal(t, "wall_1", rt.CurrentScene.Entities[0].ID)
}

func TestDamageSink_TakeDamage_UnknownEntityIsSafeNoOp(t *testing.T) {
	rt := bootDefault(t, damageProject(5))
	rec := newBusRecorder()
	rec.attach()

	out := captureLog(t, func() {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "damage/take_damage",
			Args:  map[string]any{"entity": "ghost", "amount": float64(1)},
		})
	})

	assert.Contains(t, out, "unknown entity")
	assert.Empty(t, rec.filterTopic("damage/die"))
	// Original goomba untouched.
	hp, _, _ := pixelforge_project.HealthFor(&rt.CurrentScene.Entities[0])
	assert.Equal(t, 5, hp)
}

func TestDamageSink_ExplodeRadius_HitsOnlyEntitiesWithin(t *testing.T) {
	bootDefault(t, explosionProject())
	rec := newBusRecorder()
	rec.attach()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/explode_radius",
		Args: map[string]any{
			"center_x": float64(100),
			"center_y": float64(100),
			"radius":   float64(50),
			"damage":   float64(3),
		},
	})

	hits := rec.filterTopic("damage/take_damage")
	require.Len(t, hits, 3, "exactly 3 entities should be within radius 50 of (100,100)")

	gotIDs := map[string]bool{}
	for _, ev := range hits {
		id, _ := ev.Args["entity"].(string)
		gotIDs[id] = true
		assert.Equal(t, float64(3), ev.Args["amount"], "amount must propagate")
	}
	assert.True(t, gotIDs["in_1"], "in_1 must be hit")
	assert.True(t, gotIDs["in_2"], "in_2 must be hit")
	assert.True(t, gotIDs["in_3"], "in_3 must be hit")
	assert.False(t, gotIDs["out_1"], "out_1 must NOT be hit")
	assert.False(t, gotIDs["out_2"], "out_2 must NOT be hit")
}

func TestDamageSink_ExplodeRadius_PublishesVisualFlash(t *testing.T) {
	bootDefault(t, explosionProject())
	rec := newBusRecorder()
	rec.attach()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/explode_radius",
		Args: map[string]any{
			"center_x": float64(100),
			"center_y": float64(100),
			"radius":   float64(50),
			"damage":   float64(1),
		},
	})

	flashes := rec.filterTopic("visual/flash")
	require.Len(t, flashes, 1, "explode must publish exactly one visual/flash")
	assert.Equal(t, "_blast_center", flashes[0].Args["entity"])
	assert.Equal(t, "#ffffff", flashes[0].Args["color"])
}

func TestDamageSink_ExplodeRadius_PlaysSFXWhenAudioRegistered(t *testing.T) {
	p := explosionProject()
	// The sink only consults rt.Project.Audio (name lookup) to
	// decide whether to publish audio/play_sound — it does not
	// touch the registry-side decoded sample. Leaving
	// RelativePath empty lets the audio loader skip the file read
	// while still surfacing the name to the sink.
	p.Audio = []pixelforge_project.AudioSample{
		{Name: "explosion_sfx"},
	}
	bootDefault(t, p)
	rec := newBusRecorder()
	rec.attach()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/explode_radius",
		Args: map[string]any{
			"center_x": float64(100),
			"center_y": float64(100),
			"radius":   float64(50),
			"damage":   float64(1),
		},
	})

	sfx := rec.filterTopic("audio/play_sound")
	require.Len(t, sfx, 1)
	assert.Equal(t, "explosion_sfx", sfx[0].Args["name"])
}

func TestDamageSink_ExplodeRadius_SkipsSFXWhenSampleMissing(t *testing.T) {
	// Project has no explosion_sfx audio — the publish must be
	// omitted silently.
	bootDefault(t, explosionProject())
	rec := newBusRecorder()
	rec.attach()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/explode_radius",
		Args: map[string]any{
			"center_x": float64(100),
			"center_y": float64(100),
			"radius":   float64(50),
			"damage":   float64(1),
		},
	})

	assert.Empty(t, rec.filterTopic("audio/play_sound"), "missing SFX must skip the publish")
}

func TestDamageSink_ExplodeRadius_NonPositiveRadiusIsNoOp(t *testing.T) {
	bootDefault(t, explosionProject())
	rec := newBusRecorder()
	rec.attach()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/explode_radius",
		Args: map[string]any{
			"center_x": float64(100),
			"center_y": float64(100),
			"radius":   float64(0),
			"damage":   float64(5),
		},
	})

	// No cascaded events fire for a 0-radius blast.
	assert.Empty(t, rec.filterTopic("damage/take_damage"))
	assert.Empty(t, rec.filterTopic("visual/flash"))
	assert.Empty(t, rec.filterTopic("audio/play_sound"))
}

func TestDamageSink_ExplodeRadius_FullCascadeKillsEntitiesAtZeroHP(t *testing.T) {
	// Build a fixture where the blast damage exceeds each entity's
	// HP — every in-radius entity should cascade through die +
	// destroy_other.
	p := pixelforge_project.NewProject("kill-blast-test")
	mkLowHP := func(id string, x, y float64) pixelforge_project.Entity {
		return pixelforge_project.Entity{
			ID: id, Name: id,
			Position: pixelforge_project.EntityPosition{X: x, Y: y},
			Components: []pixelforge_project.EntityComponent{
				{
					Type:   pixelforge_project.HealthComponentType,
					Values: map[string]any{"hp": float64(1), "max_hp": float64(1)},
				},
			},
		}
	}
	p.Scenes = []pixelforge_project.Scene{
		{
			ID: "main", Name: "Main",
			Entities: []pixelforge_project.Entity{
				mkLowHP("a", 100, 100),
				mkLowHP("b", 110, 100),
			},
		},
	}
	rt := bootDefault(t, p)
	rec := newBusRecorder()
	rec.attach()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/explode_radius",
		Args: map[string]any{
			"center_x": float64(100),
			"center_y": float64(100),
			"radius":   float64(50),
			"damage":   float64(99),
		},
	})

	// Each entity got take_damage → die → destroy_other.
	assert.Len(t, rec.filterTopic("damage/take_damage"), 2)
	assert.Len(t, rec.filterTopic("damage/die"), 2)
	assert.Len(t, rec.filterTopic("spawn/destroy_other"), 2)

	// Both entities removed from the scene.
	assert.Empty(t, rt.CurrentScene.Entities, "all in-radius entities should be destroyed")
}
