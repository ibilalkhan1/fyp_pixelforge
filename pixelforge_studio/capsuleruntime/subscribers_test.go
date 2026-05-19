package capsuleruntime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
)

// fakeAudio records every Play invocation so tests can assert on
// the (channel, sample, pitch, vol) tuple the dispatcher passed
// through.
type fakeAudio struct {
	calls []audioCall
}

type audioCall struct {
	Ch     pixelforge_audio.Chan
	Sample *pixelforge_audio.Sample
	Pitch  float64
	Vol    float64
}

func (a *fakeAudio) Play(ch pixelforge_audio.Chan, s *pixelforge_audio.Sample, pitch, vol float64) {
	a.calls = append(a.calls, audioCall{Ch: ch, Sample: s, Pitch: pitch, Vol: vol})
}

type fakeMusic struct {
	played  []string
	stopped int
}

func (m *fakeMusic) PlayMusic(name string) { m.played = append(m.played, name) }
func (m *fakeMusic) StopMusic()             { m.stopped++ }

type fakeScene struct {
	changes  []string
	restarts int
	waits    []int
}

func (s *fakeScene) Change(id string)  { s.changes = append(s.changes, id) }
func (s *fakeScene) Restart()          { s.restarts++ }
func (s *fakeScene) Wait(ticks int)    { s.waits = append(s.waits, ticks) }

type fakeMenu struct {
	opens  []string
	closes int
}

func (m *fakeMenu) Open(name string) { m.opens = append(m.opens, name) }
func (m *fakeMenu) Close()           { m.closes++ }

type fakeDialogue struct {
	opens  []string
	closes int
}

func (d *fakeDialogue) Open(id string) { d.opens = append(d.opens, id) }
func (d *fakeDialogue) Close()         { d.closes++ }

type fakeInventory struct {
	gives []inventoryCall
	takes []inventoryCall
	sets  []inventoryCall
}

type inventoryCall struct {
	ID    string
	Count int
}

func (i *fakeInventory) Give(id string, count int)     { i.gives = append(i.gives, inventoryCall{id, count}) }
func (i *fakeInventory) Take(id string, count int)     { i.takes = append(i.takes, inventoryCall{id, count}) }
func (i *fakeInventory) SetCount(id string, count int) { i.sets = append(i.sets, inventoryCall{id, count}) }

type fakeSave struct {
	saved   []string
	loaded  []string
	deleted []string
}

func (s *fakeSave) SaveNow(slot string)    { s.saved = append(s.saved, slot) }
func (s *fakeSave) LoadSlot(slot string)   { s.loaded = append(s.loaded, slot) }
func (s *fakeSave) DeleteSlot(slot string) { s.deleted = append(s.deleted, slot) }

// memBackend is an in-memory pisave.Backend for the save sink
// hermetic-default test (so the boot path can run without touching
// the user-config-dir).
type memBackend struct {
	data map[string][]byte
}

func newMemBackend() *memBackend { return &memBackend{data: map[string][]byte{}} }

func (b *memBackend) Read(slot string) ([]byte, error) {
	d, ok := b.data[slot]
	if !ok {
		return nil, assert.AnError
	}
	return d, nil
}

func (b *memBackend) Write(slot string, data []byte) error { b.data[slot] = data; return nil }
func (b *memBackend) Delete(slot string) error             { delete(b.data, slot); return nil }
func (b *memBackend) List() ([]pisave.SlotMeta, error) {
	out := make([]pisave.SlotMeta, 0, len(b.data))
	for k := range b.data {
		out = append(out, pisave.SlotMeta{Name: k})
	}
	return out, nil
}

// bootWithSinks resets every registry and boots a fresh runtime
// with the supplied sinks. Tests use this so each one is
// hermetic — the verbs.bus target is recreated and re-subscribed
// per-test instead of accumulating subscribers across cases.
func bootWithSinks(t *testing.T, sinks capsuleruntime.Sinks) *capsuleruntime.Runtime {
	t.Helper()
	resetAll()
	// Re-register verbs.bus so each test starts with zero
	// subscribers on the target.
	pievent.ResetRegistryForTest()
	piloop.ResetVerbsBusForTest()

	p := pixelforge_project.NewProject("subscribe-test")
	rt, err := capsuleruntime.Boot(p, fakeAssets(nil), capsuleruntime.Options{
		SaveBackendOverride: newMemBackend(),
		SinksOverride:       sinks,
	})
	require.NoError(t, err)
	return rt
}

func TestSubscribers_AudioPlaySoundDispatches(t *testing.T) {
	audio := &fakeAudio{}
	bootWithSinks(t, capsuleruntime.Sinks{Audio: audio})

	pixelforge_audio.RegisterSample("blast", pixelforge_audio.NewSample([]int8{0}, 8000))

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "audio/play_sound",
		Args:  map[string]any{"name": "blast", "pitch": 1.5, "volume": 0.8, "channel": float64(pixelforge_audio.Chan2)},
	})

	require.Len(t, audio.calls, 1)
	assert.Equal(t, pixelforge_audio.Chan2, audio.calls[0].Ch)
	assert.Equal(t, 1.5, audio.calls[0].Pitch)
	assert.Equal(t, 0.8, audio.calls[0].Vol)
	assert.NotNil(t, audio.calls[0].Sample)
}

func TestSubscribers_AudioPlaySoundMissingSampleNoOps(t *testing.T) {
	audio := &fakeAudio{}
	bootWithSinks(t, capsuleruntime.Sinks{Audio: audio})

	// No sample registered under this name.
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "audio/play_sound",
		Args:  map[string]any{"name": "missing"},
	})

	assert.Empty(t, audio.calls)
}

func TestSubscribers_AudioPlaySoundMissingNameNoOps(t *testing.T) {
	audio := &fakeAudio{}
	bootWithSinks(t, capsuleruntime.Sinks{Audio: audio})

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "audio/play_sound",
		Args:  map[string]any{},
	})

	assert.Empty(t, audio.calls)
}

func TestSubscribers_SceneChange(t *testing.T) {
	scene := &fakeScene{}
	bootWithSinks(t, capsuleruntime.Sinks{Scene: scene})

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "scene/change",
		Args:  map[string]any{"scene_id": "level_2"},
	})

	assert.Equal(t, []string{"level_2"}, scene.changes)
}

func TestSubscribers_SceneRestart(t *testing.T) {
	scene := &fakeScene{}
	bootWithSinks(t, capsuleruntime.Sinks{Scene: scene})

	piloop.VerbsBus().Publish(&piloop.VerbEvent{Topic: "scene/restart"})
	assert.Equal(t, 1, scene.restarts)
}

func TestSubscribers_SceneWait(t *testing.T) {
	scene := &fakeScene{}
	bootWithSinks(t, capsuleruntime.Sinks{Scene: scene})

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "scene/wait",
		Args:  map[string]any{"ticks": float64(60)},
	})
	assert.Equal(t, []int{60}, scene.waits)
}

func TestSubscribers_InventoryGive(t *testing.T) {
	inv := &fakeInventory{}
	bootWithSinks(t, capsuleruntime.Sinks{Inventory: inv})

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "inventory/give_item",
		Args:  map[string]any{"item_id": "key", "count": float64(2)},
	})

	require.Len(t, inv.gives, 1)
	assert.Equal(t, inventoryCall{ID: "key", Count: 2}, inv.gives[0])
}

func TestSubscribers_InventoryGiveDefaultsCountToOne(t *testing.T) {
	inv := &fakeInventory{}
	bootWithSinks(t, capsuleruntime.Sinks{Inventory: inv})

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "inventory/give_item",
		Args:  map[string]any{"item_id": "potion"},
	})

	require.Len(t, inv.gives, 1)
	assert.Equal(t, inventoryCall{ID: "potion", Count: 1}, inv.gives[0])
}

func TestSubscribers_InventoryTakeAndSet(t *testing.T) {
	inv := &fakeInventory{}
	bootWithSinks(t, capsuleruntime.Sinks{Inventory: inv})

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "inventory/take_item",
		Args:  map[string]any{"item_id": "potion"},
	})
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "inventory/set_item_count",
		Args:  map[string]any{"item_id": "potion", "count": float64(5)},
	})

	require.Len(t, inv.takes, 1)
	require.Len(t, inv.sets, 1)
	assert.Equal(t, 5, inv.sets[0].Count)
}

func TestSubscribers_DialogueOpen(t *testing.T) {
	dlg := &fakeDialogue{}
	bootWithSinks(t, capsuleruntime.Sinks{Dialogue: dlg})

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "ui/open_dialogue",
		Args:  map[string]any{"script_id": "intro"},
	})

	assert.Equal(t, []string{"intro"}, dlg.opens)
}

func TestSubscribers_MenuOpen(t *testing.T) {
	menu := &fakeMenu{}
	bootWithSinks(t, capsuleruntime.Sinks{Menu: menu})

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "ui/open_menu",
		Args:  map[string]any{"menu_name": "title"},
	})

	assert.Equal(t, []string{"title"}, menu.opens)
}

func TestSubscribers_SaveNow(t *testing.T) {
	save := &fakeSave{}
	bootWithSinks(t, capsuleruntime.Sinks{Save: save})

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "save/now",
		Args:  map[string]any{"slot_id": "slot1"},
	})

	assert.Equal(t, []string{"slot1"}, save.saved)
}

func TestSubscribers_SaveLoadAndDelete(t *testing.T) {
	save := &fakeSave{}
	bootWithSinks(t, capsuleruntime.Sinks{Save: save})

	piloop.VerbsBus().Publish(&piloop.VerbEvent{Topic: "save/load_slot", Args: map[string]any{"slot": "slot2"}})
	piloop.VerbsBus().Publish(&piloop.VerbEvent{Topic: "save/delete_slot", Args: map[string]any{"slot": "slot3"}})

	assert.Equal(t, []string{"slot2"}, save.loaded)
	assert.Equal(t, []string{"slot3"}, save.deleted)
}

func TestSubscribers_WrongTypeArgsNoCrash(t *testing.T) {
	audio := &fakeAudio{}
	inv := &fakeInventory{}
	bootWithSinks(t, capsuleruntime.Sinks{Audio: audio, Inventory: inv})

	pixelforge_audio.RegisterSample("blast", pixelforge_audio.NewSample([]int8{0}, 8000))

	// pitch should be float but is a string here.
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "audio/play_sound",
		Args:  map[string]any{"name": "blast", "pitch": "fast"},
	})
	// count should be int/float but is a string.
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "inventory/give_item",
		Args:  map[string]any{"item_id": "key", "count": "five"},
	})

	// Wrong-type fields fall back to defaults; the call still
	// dispatches without panic.
	require.Len(t, audio.calls, 1)
	assert.Equal(t, 1.0, audio.calls[0].Pitch)
	require.Len(t, inv.gives, 1)
	assert.Equal(t, 1, inv.gives[0].Count) // default
}

func TestSubscribers_NilEventNoOps(t *testing.T) {
	bootWithSinks(t, capsuleruntime.Sinks{})
	assert.NotPanics(t, func() { piloop.VerbsBus().Publish(nil) })
}

// TestSubscribers_ArcadeMotionTopicsRouteThroughMotionSink covers
// plan-008 U8's expansion: every new arcade motion / collision
// recipe lands on the existing Motion sink with its topic and
// args. Tests one representative per family rather than 11 cases.
func TestSubscribers_ArcadeMotionTopicsRouteThroughMotionSink(t *testing.T) {
	cases := []struct {
		topic string
		args  map[string]any
	}{
		{"motion/apply_thrust", map[string]any{"angle_deg": 90.0, "magnitude": 0.2}},
		{"motion/rotate_entity", map[string]any{"degrees": 5.0}},
		{"motion/screen_wrap", nil},
		{"motion/jump", map[string]any{"strength": 5.0}},
		{"motion/apply_gravity", map[string]any{"strength": 0.3}},
		{"motion/place_on_grid", map[string]any{"cell_size": 16.0}},
		{"motion/ladder_climb", map[string]any{"speed": 1.5}},
		{"motion/barrel_roll", map[string]any{"base_speed": 1.0}},
		{"collision/solid_collide", nil},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.topic, func(t *testing.T) {
			motion := &recordingMotion{}
			bootWithSinks(t, capsuleruntime.Sinks{Motion: motion})

			piloop.VerbsBus().Publish(&piloop.VerbEvent{Topic: tc.topic, Args: tc.args})

			require.Len(t, motion.calls, 1, "expected one Motion.Apply call for %s", tc.topic)
			assert.Equal(t, tc.topic, motion.calls[0].Topic,
				"Motion sink received wrong topic for %s", tc.topic)
		})
	}
}

// TestSubscribers_ArcadeExplodeRoutesThroughDamageSink — the
// Bomberman explosion verb publishes "damage/explode_radius"; the
// subscriber routes it through the Damage sink's dedicated
// ExplodeRadius method (plan-009 U9 promoted this from the prior
// HurtPlayer fallback to a first-class sink method).
func TestSubscribers_ArcadeExplodeRoutesThroughDamageSink(t *testing.T) {
	damage := &recordingDamage{}
	bootWithSinks(t, capsuleruntime.Sinks{Damage: damage})

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/explode_radius",
		Args:  map[string]any{"fuse_ticks": 120.0, "radius": 32.0, "damage": 3.0},
	})

	require.Len(t, damage.explodes, 1)
	assert.Equal(t, 3.0, damage.explodes[0]["damage"], "explode_radius args must flow into ExplodeRadius")
	assert.Equal(t, 32.0, damage.explodes[0]["radius"])
}

// recordingMotion captures every Apply call so the arcade-topic
// routing test can assert on the topic + args the dispatcher
// passes through.
type recordingMotion struct {
	calls []motionCall
}

type motionCall struct {
	Topic string
	Args  map[string]any
}

func (m *recordingMotion) Apply(topic string, args map[string]any) {
	m.calls = append(m.calls, motionCall{Topic: topic, Args: args})
}

// recordingDamage captures HurtPlayer / Die / TakeDamage /
// ExplodeRadius / GridExplode so the routing tests can assert routing.
type recordingDamage struct {
	deaths       []string
	hurts        []int
	taken        []damageCall
	explodes     []map[string]any
	gridExplodes []map[string]any
}

type damageCall struct {
	ID     string
	Amount int
}

func (d *recordingDamage) Die(id string)                 { d.deaths = append(d.deaths, id) }
func (d *recordingDamage) HurtPlayer(amount int)         { d.hurts = append(d.hurts, amount) }
func (d *recordingDamage) TakeDamage(id string, amt int) { d.taken = append(d.taken, damageCall{id, amt}) }
func (d *recordingDamage) ExplodeRadius(args map[string]any) {
	d.explodes = append(d.explodes, args)
}
func (d *recordingDamage) GridExplode(args map[string]any) {
	d.gridExplodes = append(d.gridExplodes, args)
}

func TestVerbsBus_TargetRegistered(t *testing.T) {
	// Re-register so a clean target is in the registry before
	// asserting.
	pievent.ResetRegistryForTest()
	piloop.ResetVerbsBusForTest()
	inspect := pievent.LookupTarget(piloop.VerbsBusTargetName)
	require.NotNil(t, inspect, "verbs.bus must be registered in pievent registry")
}

func TestVerbsBus_IsTypedTargetVerbEvent(t *testing.T) {
	pievent.ResetRegistryForTest()
	piloop.ResetVerbsBusForTest()
	raw := pievent.LookupTarget(piloop.VerbsBusTargetName)
	require.NotNil(t, raw)
	_, ok := raw.(pievent.Target[*piloop.VerbEvent])
	assert.True(t, ok, "verbs.bus should be pievent.Target[*VerbEvent]")
}

func TestLoopMainTargetUnchanged(t *testing.T) {
	// loop.main must remain Target[Event] (typed enum) after the
	// verbs.bus split — no regression for existing engine
	// consumers.
	pievent.ResetRegistryForTest()
	piloop.ResetVerbsBusForTest()
	// Re-register loop.main from piloop's init shape.
	pievent.RegisterTarget("loop.main", piloop.Target())
	raw := pievent.LookupTarget("loop.main")
	require.NotNil(t, raw)
	_, isStringTarget := raw.(pievent.Target[string])
	assert.False(t, isStringTarget, "loop.main must not be a string Target")
}

func TestPublishEventActionRoutesThroughVerbsBus(t *testing.T) {
	// End-to-end: the catalog's publish_event Action publishes a
	// *VerbEvent to verbs.bus, and our installed subscriber routes
	// it to the right sink. This covers the EventBusTarget change
	// + the buildPublishEvent path together.
	dlg := &fakeDialogue{}
	bootWithSinks(t, capsuleruntime.Sinks{Dialogue: dlg})

	// Build the open_dialogue recipe and apply it via the catalog
	// — same code path the verb-binding compiler uses at scene
	// load.
	recipe, ok := catalog.LookupRecipe(catalog.RecipeOpenDialogue)
	require.True(t, ok, "open_dialogue recipe must be registered")
	effect, err := catalog.ApplyWithContext(recipe, map[string]any{"script_id": "intro"}, fakeCatalogContext{})
	require.NoError(t, err)
	require.NotNil(t, effect)
	effect(nil)

	// The published topic carries the recipe's default event
	// "ui/open_dialogue" (recipe defaults don't include script_id,
	// but the override map we passed does); the subscriber decodes
	// script_id and dispatches Open.
	assert.Equal(t, []string{"intro"}, dlg.opens)
}

// fakeCatalogContext satisfies catalog.Context with the bare
// minimum the publish_event Action needs — LookupTarget — and
// returns nil for everything else.
type fakeCatalogContext struct{}

func (fakeCatalogContext) Sample(string) any            { return nil }
func (fakeCatalogContext) Entity(string) any            { return nil }
func (fakeCatalogContext) ValueRef(string) catalog.ValueRef { return nil }
func (fakeCatalogContext) LookupTarget(name string) any { return pievent.LookupTarget(name) }
