package archetype_test

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/archetype"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
)

// TestArchetype_AllSixRegisteredOnInit asserts the closed v1
// archetype set is wired up by package import alone. Anything missing
// here means the builtin_archetypes.go init() block dropped a
// template.
func TestArchetype_AllSixRegisteredOnInit(t *testing.T) {
	wanted := []string{
		archetype.ArchetypePlayer,
		archetype.ArchetypeEnemy,
		archetype.ArchetypePickup,
		archetype.ArchetypeHazard,
		archetype.ArchetypeNPC,
		archetype.ArchetypeWorld,
	}
	for _, name := range wanted {
		if _, ok := archetype.LookupArchetype(name); !ok {
			t.Errorf("archetype %q not registered", name)
		}
	}

	all := archetype.AllArchetypes()
	if len(all) != len(wanted) {
		t.Errorf("AllArchetypes returned %d templates; want %d", len(all), len(wanted))
	}
}

// TestArchetype_Player_TriggerSlotsMatchSpec pins the Player slot
// list to the exact order declared in the plan's §U5. Order matters
// because the inspector renders slots top-to-bottom and the
// "most-likely-edited first" intent is encoded in that order.
func TestArchetype_Player_TriggerSlotsMatchSpec(t *testing.T) {
	tmpl, ok := archetype.LookupArchetype(archetype.ArchetypePlayer)
	if !ok {
		t.Fatalf("Player archetype not registered")
	}

	wantSlots := []string{
		catalog.TriggerWhenInputJump,
		catalog.TriggerWhenInputAttack,
		catalog.TriggerWhenInputLeft,
		catalog.TriggerWhenInputRight,
		catalog.TriggerWhenInputUp,
		catalog.TriggerWhenInputDown,
		catalog.TriggerWhenDamaged,
		catalog.TriggerWhenSceneStarts,
		catalog.TriggerEveryTick,
	}

	if len(tmpl.TriggerSlots) != len(wantSlots) {
		t.Fatalf("Player has %d slots; want %d", len(tmpl.TriggerSlots), len(wantSlots))
	}

	for i, want := range wantSlots {
		if tmpl.TriggerSlots[i].Name != want {
			t.Errorf("Player slot[%d] = %q; want %q", i, tmpl.TriggerSlots[i].Name, want)
		}
	}
}

// TestArchetype_Enemy_DefaultVerbs covers AE1 directly: an Enemy
// archetype's when_stepped_on slot is pre-filled with die so a
// freshly-dropped Goomba behaves correctly with zero designer
// configuration.
func TestArchetype_Enemy_DefaultVerbs(t *testing.T) {
	tmpl, ok := archetype.LookupArchetype(archetype.ArchetypeEnemy)
	if !ok {
		t.Fatalf("Enemy archetype not registered")
	}

	cases := map[string]string{
		catalog.TriggerWhenSteppedOn: catalog.RecipeDie,
		catalog.TriggerWhenShot:      catalog.RecipeDie,
		catalog.TriggerWhenDestroyed: catalog.RecipeGivePoints,
	}

	for trigger, wantVerb := range cases {
		slot := findSlot(tmpl, trigger)
		if slot == nil {
			t.Errorf("Enemy missing slot %q", trigger)
			continue
		}
		if slot.DefaultVerb != wantVerb {
			t.Errorf("Enemy slot %q DefaultVerb = %q; want %q", trigger, slot.DefaultVerb, wantVerb)
		}
	}
}

// TestArchetype_World_MinimalSlots locks down the minimal-template
// invariant — World is the fallback archetype assigned by U2's
// sanitize pass to pre-v1 entities, so accidentally adding richer
// slots here would change the default load-time inspector shape for
// every existing project.
func TestArchetype_World_MinimalSlots(t *testing.T) {
	tmpl, ok := archetype.LookupArchetype(archetype.ArchetypeWorld)
	if !ok {
		t.Fatalf("World archetype not registered")
	}

	want := []string{catalog.TriggerWhenSceneStarts, catalog.TriggerEveryTick}
	if len(tmpl.TriggerSlots) != len(want) {
		t.Fatalf("World has %d slots; want %d (%v)", len(tmpl.TriggerSlots), len(want), want)
	}
	for i, w := range want {
		if tmpl.TriggerSlots[i].Name != w {
			t.Errorf("World slot[%d] = %q; want %q", i, tmpl.TriggerSlots[i].Name, w)
		}
	}
}

// TestArchetype_RelevantRecipesPopulated walks every slot in every
// archetype and asserts the dropdown won't render empty. Slots whose
// DefaultVerb is intentionally empty are allowed to have no
// recipes — some triggers (NPC's when_player_nearby in particular)
// may genuinely have no v1 recipes declared relevant. Slots with a
// DefaultVerb must always have a non-empty RelevantRecipes list,
// otherwise the inspector would fail to render its dropdown option.
func TestArchetype_RelevantRecipesPopulated(t *testing.T) {
	for _, tmpl := range archetype.AllArchetypes() {
		for _, slot := range tmpl.TriggerSlots {
			if slot.DefaultVerb == "" {
				continue
			}
			if len(slot.RelevantRecipes) == 0 {
				t.Errorf("archetype %q slot %q has DefaultVerb %q but empty RelevantRecipes",
					tmpl.Archetype, slot.Name, slot.DefaultVerb)
				continue
			}
			// The DefaultVerb itself must appear in RelevantRecipes,
			// otherwise the inspector dropdown won't be able to
			// pre-select it.
			found := false
			for _, name := range slot.RelevantRecipes {
				if name == slot.DefaultVerb {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("archetype %q slot %q DefaultVerb %q not in RelevantRecipes %v",
					tmpl.Archetype, slot.Name, slot.DefaultVerb, slot.RelevantRecipes)
			}
		}
	}
}

// TestArchetype_AllDefaultVerbsResolveInCatalog catches typos in
// DefaultVerb strings at test time rather than at studio-launch
// time. Every non-empty DefaultVerb must resolve via
// catalog.LookupRecipe.
func TestArchetype_AllDefaultVerbsResolveInCatalog(t *testing.T) {
	for _, tmpl := range archetype.AllArchetypes() {
		for _, slot := range tmpl.TriggerSlots {
			if slot.DefaultVerb == "" {
				continue
			}
			if _, ok := catalog.LookupRecipe(slot.DefaultVerb); !ok {
				t.Errorf("archetype %q slot %q DefaultVerb %q not registered in catalog",
					tmpl.Archetype, slot.Name, slot.DefaultVerb)
			}
		}
	}
}

// findSlot is a small test helper used by the per-archetype slot
// assertions. Returns nil when no slot matches.
func findSlot(t archetype.Template, name string) *archetype.TriggerSlot {
	for i := range t.TriggerSlots {
		if t.TriggerSlots[i].Name == name {
			return &t.TriggerSlots[i]
		}
	}
	return nil
}
