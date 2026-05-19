package capsuleruntime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
)

// bootDefault wires a runtime with the production default sinks
// against a fresh in-memory save backend so each test starts from
// zero. Returns rt + the project pointer the test mutates before
// calling AdvanceTick.
func bootDefault(t *testing.T, p *pixelforge_project.Project) *capsuleruntime.Runtime {
	t.Helper()
	resetAll()
	pievent.ResetRegistryForTest()
	piloop.ResetVerbsBusForTest()
	mb := &memBackend{data: map[string][]byte{}}
	var _ pisave.Backend = mb // assert backend contract
	rt, err := capsuleruntime.Boot(p, fakeAssets(nil), capsuleruntime.Options{
		SaveBackendOverride: mb,
	})
	require.NoError(t, err)
	return rt
}

// twoScenesProject builds a fixture project with two scenes for
// scene/change scenarios.
func twoScenesProject() *pixelforge_project.Project {
	p := pixelforge_project.NewProject("scene-sink-test")
	p.Scenes = []pixelforge_project.Scene{
		{ID: "main", Name: "Main", Entities: []pixelforge_project.Entity{
			{ID: "hero", Name: "Hero", Position: pixelforge_project.EntityPosition{X: 10, Y: 20}},
		}},
		{ID: "level_2", Name: "Level 2", Entities: []pixelforge_project.Entity{
			{ID: "boss", Name: "Boss", Position: pixelforge_project.EntityPosition{X: 100, Y: 200}},
		}},
	}
	return p
}

func TestSceneSink_ChangeSwapsCurrentScene(t *testing.T) {
	rt := bootDefault(t, twoScenesProject())
	require.NotNil(t, rt.CurrentScene)
	assert.Equal(t, "main", rt.CurrentScene.ID)

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "scene/change",
		Args:  map[string]any{"name": "level_2"},
	})

	require.NotNil(t, rt.CurrentScene)
	assert.Equal(t, "level_2", rt.CurrentScene.ID)
}

func TestSceneSink_ChangeAcceptsSceneIDKey(t *testing.T) {
	rt := bootDefault(t, twoScenesProject())
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "scene/change",
		Args:  map[string]any{"scene_id": "level_2"},
	})
	assert.Equal(t, "level_2", rt.CurrentScene.ID)
}

func TestSceneSink_ChangeUnknownSceneNoMutation(t *testing.T) {
	rt := bootDefault(t, twoScenesProject())
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "scene/change",
		Args:  map[string]any{"name": "nonexistent"},
	})
	// CurrentScene must still be the initial "main" scene.
	require.NotNil(t, rt.CurrentScene)
	assert.Equal(t, "main", rt.CurrentScene.ID)
}

func TestSceneSink_RestartRollsBackEntities(t *testing.T) {
	rt := bootDefault(t, twoScenesProject())
	// Mutate the live scene: drop the hero.
	rt.CurrentScene.Entities = []pixelforge_project.Entity{}
	require.Len(t, rt.CurrentScene.Entities, 0)

	piloop.VerbsBus().Publish(&piloop.VerbEvent{Topic: "scene/restart"})

	// Restart restores the original entity list.
	require.Len(t, rt.CurrentScene.Entities, 1)
	assert.Equal(t, "hero", rt.CurrentScene.Entities[0].ID)
}

func TestSceneSink_WaitSetsCountdown(t *testing.T) {
	rt := bootDefault(t, twoScenesProject())
	// 500ms at default 30 TPS = ceil(500*30/1000) = 15 ticks.
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "scene/wait",
		Args:  map[string]any{"ms": float64(500)},
	})
	tps := rt.Project.TPS
	expected := (500*tps + 999) / 1000
	assert.Equal(t, expected, rt.SceneWaitTicks)

	// Advance tick: countdown decrements.
	rt.AdvanceTick()
	assert.Equal(t, expected-1, rt.SceneWaitTicks)
}

func TestSceneSink_WaitTicksKeyDirect(t *testing.T) {
	rt := bootDefault(t, twoScenesProject())
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "scene/wait",
		Args:  map[string]any{"ticks": float64(30)},
	})
	assert.Equal(t, 30, rt.SceneWaitTicks)
}

func TestSceneSink_ChangePublishesSceneChangedEvent(t *testing.T) {
	rt := bootDefault(t, twoScenesProject())
	var seen []string
	piloop.VerbsBus().SubscribeAll(func(ev *piloop.VerbEvent, _ pievent.Handler) {
		if ev != nil && ev.Topic == "scene/changed" {
			if id, ok := ev.Args["scene_id"].(string); ok {
				seen = append(seen, id)
			}
		}
	})
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "scene/change",
		Args:  map[string]any{"name": "level_2"},
	})
	require.Equal(t, "level_2", rt.CurrentScene.ID)
	require.Len(t, seen, 1)
	assert.Equal(t, "level_2", seen[0])
}
