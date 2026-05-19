package capsuleruntime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

func projectWithGoombaPrefab() *pixelforge_project.Project {
	p := pixelforge_project.NewProject("spawn-sink-test")
	p.Scenes = []pixelforge_project.Scene{
		{ID: "main", Name: "Main", Entities: []pixelforge_project.Entity{}},
	}
	p.Prefabs = []pixelforge_project.Prefab{
		{
			Name: "goomba",
			Components: []pixelforge_project.EntityComponent{
				{Type: "hp", Values: map[string]any{"current": float64(1), "max": float64(1)}},
			},
		},
	}
	return p
}

func TestSpawnSink_SpawnAppendsEntity(t *testing.T) {
	rt := bootDefault(t, projectWithGoombaPrefab())
	require.NotNil(t, rt.CurrentScene)
	require.Len(t, rt.CurrentScene.Entities, 0)

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "spawn/entity",
		Args:  map[string]any{"prefab": "goomba", "x": float64(64), "y": float64(128)},
	})

	require.Len(t, rt.CurrentScene.Entities, 1)
	spawned := rt.CurrentScene.Entities[0]
	assert.Equal(t, "goomba", spawned.Name)
	assert.Equal(t, 64.0, spawned.Position.X)
	assert.Equal(t, 128.0, spawned.Position.Y)
	require.Len(t, spawned.Components, 1)
	assert.Equal(t, "hp", spawned.Components[0].Type)
	// Component Values were deep-copied so mutating one shouldn't
	// affect the prefab.
	spawned.Components[0].Values["current"] = float64(0)
	assert.Equal(t, float64(1), rt.Project.Prefabs[0].Components[0].Values["current"])
}

func TestSpawnSink_SpawnGeneratesUniqueIDs(t *testing.T) {
	rt := bootDefault(t, projectWithGoombaPrefab())
	for i := 0; i < 3; i++ {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "spawn/entity",
			Args:  map[string]any{"prefab": "goomba"},
		})
	}
	require.Len(t, rt.CurrentScene.Entities, 3)
	ids := map[string]bool{}
	for _, e := range rt.CurrentScene.Entities {
		ids[e.ID] = true
	}
	assert.Len(t, ids, 3, "spawn-id collisions: %v", rt.CurrentScene.Entities)
}

func TestSpawnSink_SpawnUnknownPrefabNoOp(t *testing.T) {
	rt := bootDefault(t, projectWithGoombaPrefab())
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "spawn/entity",
		Args:  map[string]any{"prefab": "ghost"},
	})
	assert.Len(t, rt.CurrentScene.Entities, 0)
}

func TestSpawnSink_DestroySelfRemovesEntity(t *testing.T) {
	p := projectWithGoombaPrefab()
	p.Scenes[0].Entities = []pixelforge_project.Entity{
		{ID: "goomba_3", Name: "goomba"},
		{ID: "goomba_4", Name: "goomba"},
	}
	rt := bootDefault(t, p)

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "spawn/destroy_self",
		Args:  map[string]any{"entity": "goomba_3"},
	})

	require.Len(t, rt.CurrentScene.Entities, 1)
	assert.Equal(t, "goomba_4", rt.CurrentScene.Entities[0].ID)
}

func TestSpawnSink_DestroyOtherRemovesEntity(t *testing.T) {
	p := projectWithGoombaPrefab()
	p.Scenes[0].Entities = []pixelforge_project.Entity{
		{ID: "a"}, {ID: "b"}, {ID: "c"},
	}
	rt := bootDefault(t, p)

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "spawn/destroy_other",
		Args:  map[string]any{"entity_id": "b"},
	})

	require.Len(t, rt.CurrentScene.Entities, 2)
	assert.Equal(t, "a", rt.CurrentScene.Entities[0].ID)
	assert.Equal(t, "c", rt.CurrentScene.Entities[1].ID)
}

func TestSpawnSink_DestroyAlsoClearsVisualEffects(t *testing.T) {
	p := projectWithGoombaPrefab()
	p.Scenes[0].Entities = []pixelforge_project.Entity{{ID: "goomba_1"}}
	rt := bootDefault(t, p)

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "visual/hide",
		Args:  map[string]any{"entity": "goomba_1"},
	})
	require.NotNil(t, rt.VisualEffects["goomba_1"])

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "spawn/destroy_self",
		Args:  map[string]any{"entity": "goomba_1"},
	})

	_, ok := rt.VisualEffects["goomba_1"]
	assert.False(t, ok, "destroyed entity's visual effects should be cleared")
}
