package pixelforge_project

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Loading a project that includes an entity with no archetype key
// (pre-v1 shape) sanitises the entity's Archetype to World.
func TestEntity_Archetype_DefaultsToWorld(t *testing.T) {
	raw := `{
		"schema_version": 1,
		"name": "prev1",
		"screen_width": 320, "screen_height": 180, "tps": 30,
		"palette": {}, "theme": {},
		"sprites": [], "audio": [],
		"scenes": [
			{
				"id": "main",
				"name": "Main",
				"entities": [
					{
						"id": "e1",
						"name": "Old Entity",
						"position": {"x": 0, "y": 0, "z": 0},
						"components": []
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
	assert.Equal(t, ArchetypeWorld, p.Scenes[0].Entities[0].Archetype)
}

// Each of the six v1 archetype constants survives a JSON round-trip on
// an Entity without mutation.
func TestEntity_Archetype_RoundTrip(t *testing.T) {
	for _, a := range AllArchetypes() {
		a := a
		t.Run(a, func(t *testing.T) {
			original := Entity{
				ID:        "e1",
				Name:      "test",
				Position:  EntityPosition{X: 1, Y: 2, Z: 3},
				Archetype: a,
			}
			data, err := json.Marshal(original)
			require.NoError(t, err)

			var loaded Entity
			require.NoError(t, json.Unmarshal(data, &loaded))
			assert.Equal(t, a, loaded.Archetype)
		})
	}
}

// AllArchetypes lists exactly the six v1 constants in canonical order.
// Locks the closed set so future expansions are deliberate.
func TestArchetype_AllArchetypesClosedSet(t *testing.T) {
	want := []string{
		ArchetypePlayer,
		ArchetypeEnemy,
		ArchetypePickup,
		ArchetypeHazard,
		ArchetypeNPC,
		ArchetypeWorld,
	}
	assert.Equal(t, want, AllArchetypes())
}

// DefaultArchetype is World — the minimal template — so old entities
// don't accidentally activate behavior on sanitize.
func TestArchetype_DefaultIsWorld(t *testing.T) {
	assert.Equal(t, ArchetypeWorld, DefaultArchetype)
}

// An Entity with a non-empty Archetype that's NOT one of the v1
// constants is preserved as-is (forward-compat); applyDefaults must
// not overwrite an unknown value with World.
func TestEntity_Archetype_UnknownPreserved(t *testing.T) {
	p := NewProject("forwardcompat")
	p.Scenes[0].Entities = append(p.Scenes[0].Entities, Entity{
		ID:        "future",
		Name:      "Future",
		Archetype: "Vehicle", // hypothetical future archetype
	})
	p.applyDefaults()
	assert.Equal(t, "Vehicle", p.Scenes[0].Entities[0].Archetype)
}
