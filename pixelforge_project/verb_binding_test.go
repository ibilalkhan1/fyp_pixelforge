package pixelforge_project

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A verb binding with an unknown recipe name is preserved through a
// load/save cycle so projects authored against a newer recipe library
// don't lose data when opened in an older studio.
func TestEntity_VerbBindings_UnknownRecipePreserved(t *testing.T) {
	raw := `{
		"schema_version": 1,
		"name": "vb",
		"screen_width": 320, "screen_height": 180, "tps": 30,
		"palette": {}, "theme": {},
		"sprites": [], "audio": [],
		"scenes": [
			{
				"id": "main", "name": "Main",
				"entities": [
					{
						"id": "e1", "name": "Future Goomba",
						"position": {"x": 0, "y": 0, "z": 0},
						"components": [],
						"archetype": "Enemy",
						"verb_bindings": [
							{
								"trigger": "when_stepped_on",
								"recipe_name": "nonexistent_verb"
							}
						]
					}
				],
				"tilemaps": []
			}
		],
		"behaviors": [], "bindings": [], "event_subscriptions": [], "extension_hooks": []
	}`

	p, err := LoadReader(bytesReader([]byte(raw)))
	require.NoError(t, err)

	require.Len(t, p.Scenes, 1)
	require.Len(t, p.Scenes[0].Entities, 1)
	require.Len(t, p.Scenes[0].Entities[0].VerbBindings, 1)

	got := p.Scenes[0].Entities[0].VerbBindings[0]
	assert.Equal(t, "when_stepped_on", got.Trigger)
	assert.Equal(t, "nonexistent_verb", got.RecipeName)
}

// ArgOverrides survive a JSON round-trip with numeric, string, and
// bool values — the catalog's recipe.Apply layer can read them back.
func TestEntity_VerbBindings_ArgOverridesPreserved(t *testing.T) {
	original := Entity{
		ID:        "e1",
		Name:      "Pickup",
		Archetype: ArchetypePickup,
		VerbBindings: []VerbBinding{
			{
				Trigger:    "when_touched",
				RecipeName: "give_points",
				ArgOverrides: map[string]any{
					"amount": float64(50), // json.Unmarshal returns float64 for numbers
					"sound":  "coin",
					"silent": false,
				},
			},
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var loaded Entity
	require.NoError(t, json.Unmarshal(data, &loaded))

	require.Len(t, loaded.VerbBindings, 1)
	gotOverrides := loaded.VerbBindings[0].ArgOverrides
	assert.Equal(t, float64(50), gotOverrides["amount"])
	assert.Equal(t, "coin", gotOverrides["sound"])
	assert.Equal(t, false, gotOverrides["silent"])
}

// A VerbBinding with no ArgOverrides omits the arg_overrides key
// entirely (omitempty), keeping pre-v1-style entities small.
func TestVerbBinding_NoOverrides_OmitsKey(t *testing.T) {
	vb := VerbBinding{Trigger: "when_touched", RecipeName: "die"}
	data, err := json.Marshal(vb)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "arg_overrides")
}

// An entity with no verb bindings serialises without a verb_bindings
// key thanks to omitempty on the slice.
func TestEntity_NoVerbBindings_OmitsKey(t *testing.T) {
	e := Entity{ID: "e1", Name: "x", Position: EntityPosition{}, Components: []EntityComponent{}}
	data, err := json.Marshal(e)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "verb_bindings")
}

// Loading the editor.pforge fixture (which has no archetype or
// verb_bindings fields) succeeds and sanitises every entity to
// ArchetypeWorld with no spurious behavior change.
func TestProject_PreV1Roundtrip(t *testing.T) {
	const fixture = "../pixelforge_studio/editor/cart_assets/editor.pforge"
	p, err := Load(fixture)
	require.NoError(t, err)

	// All entities (there are none in the fixture today, but if any
	// are added they must default to World) should have a sanitised
	// archetype.
	for _, sc := range p.Scenes {
		for _, e := range sc.Entities {
			assert.NotEmpty(t, e.Archetype, "entity %q missing sanitised Archetype", e.ID)
		}
	}

	// Re-encoding must not introduce any new top-level keys beyond the
	// additive omitempty additions (input_map appears because
	// applyDefaults materialised it; archetype/verb_bindings are
	// per-entity and the fixture has no entities, so neither key
	// surfaces).
	out, err := json.Marshal(p)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, `"schema_version":1`)
	assert.Contains(t, s, `"input_map"`)
	assert.NotContains(t, s, `"archetype"`,
		"editor.pforge has zero entities; no per-entity archetype key should appear")
	assert.NotContains(t, s, `"verb_bindings"`,
		"editor.pforge has zero entities; no per-entity verb_bindings key should appear")
}
