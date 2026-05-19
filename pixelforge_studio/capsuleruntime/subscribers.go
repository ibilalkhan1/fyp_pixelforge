package capsuleruntime

import (
	"log"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_menus"
	pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"
)

// Sinks groups the subsystem-side handlers verb-recipe subscribers
// dispatch into. Each field is an interface so tests can inject
// fakes that record invocations; Boot fills in production defaults
// for any field left nil (real audio for AudioSink, an in-memory
// inventory for InventoryStore, no-op-with-log stubs for the sinks
// whose engine-side implementations are deferred to later plans).
//
// The interface split keeps engine packages unaware of the verb-
// recipe topic surface — engine code never imports this package or
// the scripting catalog; everything that "binds a verb to a
// subsystem" lives here.
type Sinks struct {
	Audio     AudioSink
	Music     MusicSink
	Scene     SceneController
	Menu      MenuController
	Dialogue  DialogueController
	Inventory InventoryStore
	Damage    DamageSink
	Motion    MotionSink
	Spawn     SpawnSink
	Visual    VisualSink
	Save      SaveSink
}

// AudioSink plays a one-shot sample. Production default wraps
// pixelforge_audio.Play.
type AudioSink interface {
	Play(ch pixelforge_audio.Chan, sample *pixelforge_audio.Sample, pitch, vol float64)
}

// MusicSink is the BGM manager seam. Production default logs and
// no-ops until a streaming-audio subsystem lands.
type MusicSink interface {
	PlayMusic(name string)
	StopMusic()
}

// SceneController routes scene/change/restart/wait verbs. Default
// is a no-op-with-log stub — the runtime scene controller is
// deferred to a follow-on plan; subscribers fire correctly here so
// authored verbs are visible in logs immediately.
type SceneController interface {
	Change(sceneID string)
	Restart()
	Wait(ticks int)
}

// MenuController routes ui/open_menu and ui/close_menu. Production
// default wraps a *pixelforge_menus.MenuStack and looks up the
// menu's template via pixelforge_project.Project.Menus.
type MenuController interface {
	Open(menuName string)
	Close()
}

// DialogueController routes ui/open_dialogue and ui/close_dialogue.
// Default looks up the parsed Tree via pixelforge_dialogue.Lookup
// Script and logs — the renderer wiring is deferred.
type DialogueController interface {
	Open(scriptID string)
	Close()
}

// InventoryStore routes inventory/give_item / take_item /
// set_item_count. Production default is an in-memory map keyed by
// item id; future plans replace it with a blackboard-backed store
// once the engine inventory model lands.
type InventoryStore interface {
	Give(itemID string, count int)
	Take(itemID string, count int)
	SetCount(itemID string, count int)
}

// DamageSink routes damage/die / hurt_player / take_damage /
// explode_radius / grid_explode. Plan-009 U9 lands the concrete
// implementation for the first four in damage_sink.go; U15 adds
// GridExplode for the grid-cardinal Bomberman variant. The log*
// fallback remains as a test fixture.
type DamageSink interface {
	Die(entityID string)
	HurtPlayer(amount int)
	TakeDamage(entityID string, amount int)
	ExplodeRadius(args map[string]any)
	GridExplode(args map[string]any)
}

// MotionSink receives any motion/* verb that wasn't translated to
// a concrete Action (move_pattern, bounce, intent — teleport_to is
// handled by the move_entity Action directly). The handler routes
// by topic so a single sink covers the family.
type MotionSink interface {
	Apply(topic string, args map[string]any)
}

// SpawnSink routes spawn/entity / destroy_self / destroy_other.
type SpawnSink interface {
	Spawn(args map[string]any)
	DestroySelf(args map[string]any)
	DestroyOther(args map[string]any)
}

// VisualSink routes visual/hide / show / flash / swap_sprite.
type VisualSink interface {
	Hide(args map[string]any)
	Show(args map[string]any)
	Flash(args map[string]any)
	SwapSprite(args map[string]any)
}

// SaveSink routes save/now / load_slot / delete_slot. Production
// default wraps the capsule's *pisave.Service.
type SaveSink interface {
	SaveNow(slot string)
	LoadSlot(slot string)
	DeleteSlot(slot string)
}

// InstallSubscribers wires the verb-recipe topic surface to the
// runtime's Sinks. Looks up the "verbs.bus" target via
// pievent.LookupTarget, asserts to Target[*VerbEvent], and
// SubscribeAlls a single dispatcher that switches on
// VerbEvent.Topic. Idempotent — calling twice on the same runtime
// installs two subscribers, but the second is dead weight; tests
// that exercise idempotency should reset the bus via
// pievent.ResetRegistryForTest between runs.
func InstallSubscribers(rt *Runtime) {
	if rt == nil {
		return
	}
	rt.Sinks.fillDefaults(rt)
	target := piloop.VerbsBus()
	target.SubscribeAll(func(ev *piloop.VerbEvent, _ pievent.Handler) {
		if ev == nil {
			return
		}
		rt.dispatch(ev)
	})
}

// dispatch routes one VerbEvent to the matching sink. Unknown
// topics no-op silently — recipes published from older catalogs
// might name topics this runtime doesn't know yet, and dropping
// them quietly matches the broader "never crash a shipped game"
// posture.
func (rt *Runtime) dispatch(ev *piloop.VerbEvent) {
	switch ev.Topic {
	case "audio/play_sound":
		handleAudioPlaySound(rt, ev.Args)
	case "audio/play_music":
		handleAudioPlayMusic(rt, ev.Args)
	case "audio/stop_music":
		rt.Sinks.Music.StopMusic()

	case "scene/change":
		handleSceneChange(rt, ev.Args)
	case "scene/restart":
		rt.Sinks.Scene.Restart()
	case "scene/wait":
		handleSceneWait(rt, ev.Args)

	case "ui/open_dialogue":
		handleDialogueOpen(rt, ev.Args)
	case "ui/close_dialogue":
		rt.Sinks.Dialogue.Close()
	case "ui/open_menu":
		handleMenuOpen(rt, ev.Args)
	case "ui/close_menu":
		rt.Sinks.Menu.Close()

	case "inventory/give_item":
		handleInventoryGive(rt, ev.Args)
	case "inventory/take_item":
		handleInventoryTake(rt, ev.Args)
	case "inventory/set_item_count":
		handleInventorySetCount(rt, ev.Args)

	case "damage/die":
		handleDamageDie(rt, ev.Args)
	case "damage/hurt_player":
		handleDamageHurtPlayer(rt, ev.Args)
	case "damage/take_damage":
		handleDamageTakeDamage(rt, ev.Args)

	case "motion/move_pattern", "motion/bounce", "motion/move_with_intent",
		"motion/apply_thrust", "motion/rotate_entity", "motion/screen_wrap",
		"motion/jump", "motion/apply_gravity", "motion/place_on_grid",
		"motion/ladder_climb", "motion/barrel_roll":
		rt.Sinks.Motion.Apply(ev.Topic, ev.Args)

	case "collision/solid_collide":
		rt.Sinks.Motion.Apply(ev.Topic, ev.Args)

	case "damage/explode_radius":
		// Radius-AoE explosion is routed through the Damage sink
		// because the resolution step damages every entity in the
		// blast — the actual entity-removal happens via the
		// existing damage path (Die/TakeDamage). The sink decodes
		// fuse_ticks + radius + damage from args.
		handleDamageExplodeRadius(rt, ev.Args)

	case "damage/grid_explode":
		// Grid-cardinal explosion is the Bomberman variant: the
		// blast walks 4 cardinal rays out from the origin tile,
		// stopping at solid tiles, damaging every entity it touches.
		// Routed through the Damage sink alongside explode_radius
		// because the resolution step also goes through the
		// damage/take_damage cascade.
		rt.Sinks.Damage.GridExplode(ev.Args)

	case "flow/fixed_tick_loop":
		// fixed_tick_loop is an authoring helper, not an effect —
		// the catalog publishes it so per-tick condition + action
		// sequences have a visible bus signature for diagnostics.
		// No sink behaviour today; future work (per plan-008 U8)
		// can grow a Flow sink if grouping needs runtime support.
		debugDrop("flow/fixed_tick_loop", "no sink yet (authoring marker)", ev.Args)

	case "spawn/entity":
		rt.Sinks.Spawn.Spawn(ev.Args)
	case "spawn/destroy_self":
		rt.Sinks.Spawn.DestroySelf(ev.Args)
	case "spawn/destroy_other":
		rt.Sinks.Spawn.DestroyOther(ev.Args)

	case "visual/hide":
		rt.Sinks.Visual.Hide(ev.Args)
	case "visual/show":
		rt.Sinks.Visual.Show(ev.Args)
	case "visual/flash":
		rt.Sinks.Visual.Flash(ev.Args)
	case "visual/swap_sprite":
		rt.Sinks.Visual.SwapSprite(ev.Args)

	case "save/now":
		handleSaveNow(rt, ev.Args)
	case "save/load_slot":
		handleSaveLoad(rt, ev.Args)
	case "save/delete_slot":
		handleSaveDelete(rt, ev.Args)
	}
}

// ---- handlers ----------------------------------------------------

// handleAudioPlaySound resolves the named sample via the audio
// registry and dispatches to the AudioSink. Missing name, missing
// sample, or wrong-type args no-op without panic.
func handleAudioPlaySound(rt *Runtime, args map[string]any) {
	name, ok := argString(args, "name")
	if !ok || name == "" {
		debugDrop("audio/play_sound", "missing name", args)
		return
	}
	sample := pixelforge_audio.LookupSample(name)
	if sample == nil {
		debugDrop("audio/play_sound", "unknown sample", args)
		return
	}
	pitch := argFloatOr(args, "pitch", 1.0)
	vol := argFloatOr(args, "volume", 1.0)
	chRaw := argFloatOr(args, "channel", float64(pixelforge_audio.Chan1))
	rt.Sinks.Audio.Play(pixelforge_audio.Chan(chRaw), sample, pitch, vol)
}

func handleAudioPlayMusic(rt *Runtime, args map[string]any) {
	name, _ := argString(args, "name")
	rt.Sinks.Music.PlayMusic(name)
}

func handleSceneChange(rt *Runtime, args map[string]any) {
	id, ok := argString(args, "scene_id")
	if !ok {
		// Plan-009 U7 accepts the shorthand "name" key — verb
		// recipes that pass scene names rather than IDs land
		// without a recipe-author migration.
		id, ok = argString(args, "name")
	}
	if !ok {
		debugDrop("scene/change", "missing scene_id", args)
		return
	}
	rt.Sinks.Scene.Change(id)
}

func handleSceneWait(rt *Runtime, args map[string]any) {
	// Accept "ticks" (engine-native) or "ms" (recipe-natural). When
	// "ms" is the only key, convert via Project.TPS so the recipe
	// fires identically across 30-TPS and 60-TPS projects.
	if raw, ok := args["ms"]; ok && raw != nil {
		ms := int(argFloatOr(args, "ms", 0))
		tps := 60
		if rt.Project != nil && rt.Project.TPS > 0 {
			tps = rt.Project.TPS
		}
		ticks := (ms*tps + 999) / 1000
		rt.Sinks.Scene.Wait(ticks)
		return
	}
	ticks := int(argFloatOr(args, "ticks", 0))
	rt.Sinks.Scene.Wait(ticks)
}

func handleDialogueOpen(rt *Runtime, args map[string]any) {
	id, ok := argString(args, "script_id")
	if !ok {
		// Recipes pre-dating the "script_id" key spell it "id"; the
		// fallback keeps older sheets working without forcing a
		// migration pass.
		id, ok = argString(args, "id")
	}
	if !ok || id == "" {
		debugDrop("ui/open_dialogue", "missing script_id", args)
		return
	}
	rt.Sinks.Dialogue.Open(id)
}

func handleMenuOpen(rt *Runtime, args map[string]any) {
	name, ok := argString(args, "menu_name")
	if !ok {
		name, ok = argString(args, "name")
	}
	if !ok || name == "" {
		debugDrop("ui/open_menu", "missing menu_name", args)
		return
	}
	rt.Sinks.Menu.Open(name)
}

func handleInventoryGive(rt *Runtime, args map[string]any) {
	id, ok := argString(args, "item_id")
	if !ok {
		debugDrop("inventory/give_item", "missing item_id", args)
		return
	}
	count := int(argFloatOr(args, "count", 1))
	rt.Sinks.Inventory.Give(id, count)
}

func handleInventoryTake(rt *Runtime, args map[string]any) {
	id, ok := argString(args, "item_id")
	if !ok {
		debugDrop("inventory/take_item", "missing item_id", args)
		return
	}
	count := int(argFloatOr(args, "count", 1))
	rt.Sinks.Inventory.Take(id, count)
}

func handleInventorySetCount(rt *Runtime, args map[string]any) {
	id, ok := argString(args, "item_id")
	if !ok {
		debugDrop("inventory/set_item_count", "missing item_id", args)
		return
	}
	count := int(argFloatOr(args, "count", 0))
	rt.Sinks.Inventory.SetCount(id, count)
}

func handleDamageDie(rt *Runtime, args map[string]any) {
	id, ok := argString(args, "entity_id")
	if !ok {
		id, _ = argString(args, "entity")
	}
	rt.Sinks.Damage.Die(id)
}

func handleDamageHurtPlayer(rt *Runtime, args map[string]any) {
	amount := int(argFloatOr(args, "amount", 1))
	rt.Sinks.Damage.HurtPlayer(amount)
}

func handleDamageTakeDamage(rt *Runtime, args map[string]any) {
	id, ok := argString(args, "entity_id")
	if !ok {
		id, _ = argString(args, "entity")
	}
	amount := int(argFloatOr(args, "amount", 1))
	rt.Sinks.Damage.TakeDamage(id, amount)
}

// handleDamageExplodeRadius dispatches the point-blast explosion
// verb to the damage sink, which iterates within-radius entities,
// fires chained damage/take_damage events on each, and emits a
// visual/flash + audio/play_sound for the blast itself. Plan-009
// U9 lands the concrete handler in damage_sink.go.
func handleDamageExplodeRadius(rt *Runtime, args map[string]any) {
	rt.Sinks.Damage.ExplodeRadius(args)
}

func handleSaveNow(rt *Runtime, args map[string]any) {
	slot, ok := argString(args, "slot_id")
	if !ok {
		slot, _ = argString(args, "slot")
	}
	rt.Sinks.Save.SaveNow(slot)
}

func handleSaveLoad(rt *Runtime, args map[string]any) {
	slot, ok := argString(args, "slot_id")
	if !ok {
		slot, _ = argString(args, "slot")
	}
	rt.Sinks.Save.LoadSlot(slot)
}

func handleSaveDelete(rt *Runtime, args map[string]any) {
	slot, ok := argString(args, "slot_id")
	if !ok {
		slot, _ = argString(args, "slot")
	}
	rt.Sinks.Save.DeleteSlot(slot)
}

// ---- arg-decoding helpers ---------------------------------------

// argString returns (s, true) when args[key] is a non-empty string,
// (s, true) for any string (incl. empty) so callers can distinguish
// missing-key from empty-value, or ("", false) for missing/wrong-type.
func argString(args map[string]any, key string) (string, bool) {
	if args == nil {
		return "", false
	}
	raw, ok := args[key]
	if !ok {
		return "", false
	}
	s, ok := raw.(string)
	if !ok {
		return "", false
	}
	return s, true
}

// argFloatOr returns args[key] as float64 falling back to def for
// missing keys or wrong-type values. JSON unmarshals numeric verb
// args as float64 universally; ints and uint8 (audio channel)
// also coerce through here for tests that build payloads
// programmatically.
func argFloatOr(args map[string]any, key string, def float64) float64 {
	if args == nil {
		return def
	}
	raw, ok := args[key]
	if !ok {
		return def
	}
	switch v := raw.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	}
	return def
}

func debugDrop(topic, reason string, args map[string]any) {
	log.Printf("[capsuleruntime] dropped %s (%s): %v", topic, reason, args)
}

// ---- default sink implementations -------------------------------

// fillDefaults populates any Sinks field left nil with a production
// default. Called by InstallSubscribers; tests that inject one or
// two sinks keep the rest as defaults.
func (s *Sinks) fillDefaults(rt *Runtime) {
	if s.Audio == nil {
		s.Audio = realAudioSink{}
	}
	if s.Music == nil {
		s.Music = newMusicSink(rt)
	}
	if s.Scene == nil {
		s.Scene = newSceneSink(rt)
	}
	if s.Menu == nil {
		s.Menu = newMenuController(rt)
	}
	if s.Dialogue == nil {
		s.Dialogue = newDialogueSink(rt)
	}
	if s.Inventory == nil {
		s.Inventory = newInventoryStore()
	}
	if s.Damage == nil {
		s.Damage = newDamageSink(rt)
	}
	if s.Motion == nil {
		s.Motion = newMotionSink(rt)
	}
	if s.Spawn == nil {
		s.Spawn = newSpawnSink(rt)
	}
	if s.Visual == nil {
		s.Visual = newVisualSink(rt)
	}
	if s.Save == nil {
		s.Save = saveServiceSink{rt: rt, svc: rt.Save}
	}
}

// realAudioSink dispatches to pixelforge_audio.Play directly.
type realAudioSink struct{}

func (realAudioSink) Play(ch pixelforge_audio.Chan, sample *pixelforge_audio.Sample, pitch, vol float64) {
	pixelforge_audio.Play(ch, sample, pitch, vol)
}

// logMusicSink, logSceneController, logDamageSink, logMotionSink,
// logSpawnSink, logVisualSink are silent-no-op fallback types kept
// as test fixtures. Plan-009 U7 flipped the production defaults in
// fillDefaults to concrete sinks (sceneSink, visualSink, spawnSink,
// dialogueSink, musicSink); the log* types remain here so tests
// that want a pure no-op fallback (e.g. "the dispatcher routed the
// event, we don't care what the sink did") can still inject them.
// They are no longer the default for any field — fillDefaults wires
// concrete sinks for every nil entry.
//
// Damage + Motion log sinks remain the production defaults until
// U8 + U9 land — both units explicitly replace these types with
// concrete implementations.

type logMusicSink struct{}

func (logMusicSink) PlayMusic(string) {}
func (logMusicSink) StopMusic()       {}

type logSceneController struct{}

func (logSceneController) Change(string) {}
func (logSceneController) Restart()      {}
func (logSceneController) Wait(int)      {}

type logDamageSink struct{}

func (logDamageSink) Die(string)                   {}
func (logDamageSink) HurtPlayer(int)               {}
func (logDamageSink) TakeDamage(string, int)       {}
func (logDamageSink) ExplodeRadius(map[string]any) {}
func (logDamageSink) GridExplode(map[string]any)   {}

type logMotionSink struct{}

func (logMotionSink) Apply(string, map[string]any) {}

type logSpawnSink struct{}

func (logSpawnSink) Spawn(map[string]any)        {}
func (logSpawnSink) DestroySelf(map[string]any)  {}
func (logSpawnSink) DestroyOther(map[string]any) {}

type logVisualSink struct{}

func (logVisualSink) Hide(map[string]any)       {}
func (logVisualSink) Show(map[string]any)       {}
func (logVisualSink) Flash(map[string]any)      {}
func (logVisualSink) SwapSprite(map[string]any) {}

// menuController wraps a MenuStack and resolves the menu name to
// its template via the loaded project's Menus map. Production
// path.
type menuController struct {
	rt    *Runtime
	stack *pixelforge_menus.MenuStack
}

func newMenuController(rt *Runtime) *menuController {
	return &menuController{rt: rt, stack: pixelforge_menus.NewMenuStack()}
}

func (c *menuController) Open(menuName string) {
	if c.rt == nil || c.rt.Project == nil {
		return
	}
	cfg, ok := c.rt.Project.Menus[menuName]
	if !ok {
		log.Printf("[capsuleruntime] open_menu %q: unknown menu", menuName)
		return
	}
	c.stack.Push(menuName, cfg.Template, cfg.Parameters)
}

func (c *menuController) Close() {
	if c == nil || c.stack == nil {
		return
	}
	c.stack.Pop()
}

// dialogueController is the legacy log-only controller plan-008
// shipped. Plan-009 U7 replaced the default with dialogueSink (see
// dialogue_sink.go); dialogueController stays in the package as a
// silent-no-op fallback type for tests that want to assert
// dispatch-only behaviour without involving the concrete sink.
type dialogueController struct {
	current string
}

func newDialogueController() *dialogueController { return &dialogueController{} }

func (d *dialogueController) Open(id string) { d.current = id }
func (d *dialogueController) Close()         { d.current = "" }

// inventoryStore is the in-memory map default. Production-grade
// inventory will swap this for a blackboard-backed store; the
// interface keeps the seam stable across that swap.
type inventoryStore struct {
	counts map[string]int
}

func newInventoryStore() *inventoryStore { return &inventoryStore{counts: map[string]int{}} }

func (s *inventoryStore) Give(id string, count int) {
	if id == "" {
		return
	}
	if count <= 0 {
		count = 1
	}
	s.counts[id] += count
}

func (s *inventoryStore) Take(id string, count int) {
	if id == "" {
		return
	}
	if count <= 0 {
		count = 1
	}
	s.counts[id] -= count
	if s.counts[id] < 0 {
		s.counts[id] = 0
	}
}

func (s *inventoryStore) SetCount(id string, count int) {
	if id == "" {
		return
	}
	if count < 0 {
		count = 0
	}
	s.counts[id] = count
}

// Count returns the stored count for id (zero when absent). Exposed
// for diagnostics + tests that inspect inventory state without
// reaching through the Sinks field.
func (s *inventoryStore) Count(id string) int { return s.counts[id] }

// saveServiceSink adapts *pisave.Service to the SaveSink interface.
// Errors from the service are logged — the save verbs don't have a
// blocking error path in the recipe surface, so failure surfaces
// in logs and gameplay continues.
//
// Plan-009 U10 swaps the literal pisave.Snapshot{} the v1 stub wrote
// for a real EncodeSnapshot pass over the runtime — the save service
// now persists the live scene's entities + components + physics body
// state + globals + tick. LoadSlot mirrors the path: read the slot,
// then DecodeSnapshot back into rt.
type saveServiceSink struct {
	rt  *Runtime
	svc *pisave.Service
}

func (s saveServiceSink) SaveNow(slot string) {
	if s.svc == nil || slot == "" {
		return
	}
	snapshot, err := EncodeSnapshot(s.rt)
	if err != nil {
		log.Printf("[capsuleruntime] save/now %q encode failed: %v", slot, err)
		return
	}
	if err := s.svc.SaveToSlot(snapshot, slot); err != nil {
		log.Printf("[capsuleruntime] save/now %q failed: %v", slot, err)
	}
}

func (s saveServiceSink) LoadSlot(slot string) {
	if s.svc == nil || slot == "" {
		return
	}
	snapshot, err := s.svc.LoadFromSlot(slot)
	if err != nil {
		log.Printf("[capsuleruntime] save/load_slot %q failed: %v", slot, err)
		return
	}
	if err := DecodeSnapshot(s.rt, snapshot); err != nil {
		log.Printf("[capsuleruntime] save/load_slot %q decode failed: %v", slot, err)
	}
}

func (s saveServiceSink) DeleteSlot(slot string) {
	if s.svc == nil || slot == "" {
		return
	}
	if err := s.svc.DeleteSlot(slot); err != nil {
		log.Printf("[capsuleruntime] save/delete_slot %q failed: %v", slot, err)
	}
}
