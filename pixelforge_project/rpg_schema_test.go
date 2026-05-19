package pixelforge_project

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rpg_schema_test.go covers idea #6 v1 U4: schema additions
// (Dialogues, Menus, Items, SaveConfig) round-trip cleanly and
// pre-v1 projects load without surprises (AE7).

func TestRPGSchema_PreV1ProjectLoadsWithEmptyMaps(t *testing.T) {
	const pre = `{"schema_version": 1, "name": "pre-v1", "scenes": []}`
	p := loadFromBytes(t, []byte(pre))
	assert.NotNil(t, p.Dialogues, "Dialogues map initialised on load")
	assert.NotNil(t, p.Menus)
	assert.NotNil(t, p.Items)
	assert.Empty(t, p.Dialogues)
	assert.Empty(t, p.Menus)
	assert.Empty(t, p.Items)
}

func TestRPGSchema_SaveConfigDefaultsAutosaveEnabled(t *testing.T) {
	const pre = `{"schema_version": 1, "name": "pre-v1", "scenes": []}`
	p := loadFromBytes(t, []byte(pre))
	assert.True(t, p.SaveConfig.AutosaveEnabled,
		"new projects default to autosave-enabled")
	assert.True(t, p.SaveConfig.Set,
		"Set sentinel records 'defaults applied' for omitempty round-trip")
}

func TestRPGSchema_DialoguesRoundTrip(t *testing.T) {
	p := NewProject("test")
	p.applyDefaults()
	p.Dialogues["old_man_hint"] = DialogueScript{
		Name:   "old_man_hint",
		Script: "OLD MAN: Welcome, traveller.\n:: accept\nOLD MAN: Take this.",
	}

	data, err := json.Marshal(p)
	require.NoError(t, err)
	var loaded Project
	require.NoError(t, json.Unmarshal(data, &loaded))
	loaded.applyDefaults()
	require.Contains(t, loaded.Dialogues, "old_man_hint")
	assert.Contains(t, loaded.Dialogues["old_man_hint"].Script, "OLD MAN: Welcome")
}

func TestRPGSchema_MenusRoundTrip(t *testing.T) {
	p := NewProject("test")
	p.applyDefaults()
	p.Menus["title_screen"] = MenuConfig{
		Name:     "title_screen",
		Template: "title",
		Parameters: map[string]any{
			"game_name": "Adventure",
			"subtitle":  "A Hero's Tale",
		},
	}

	data, err := json.Marshal(p)
	require.NoError(t, err)
	var loaded Project
	require.NoError(t, json.Unmarshal(data, &loaded))
	loaded.applyDefaults()
	got := loaded.Menus["title_screen"]
	assert.Equal(t, "title", got.Template)
	assert.Equal(t, "Adventure", got.Parameters["game_name"])
}

func TestRPGSchema_ItemsRoundTrip(t *testing.T) {
	p := NewProject("test")
	p.applyDefaults()
	p.Items = append(p.Items,
		ItemDefinition{ID: "potion", Name: "Potion", Icon: "potion_sprite", Description: "Restores HP", EffectVerb: "restore_health(50)", Category: "potion"},
		ItemDefinition{ID: "sword", Name: "Sword", Category: "weapon"},
	)

	data, err := json.Marshal(p)
	require.NoError(t, err)
	var loaded Project
	require.NoError(t, json.Unmarshal(data, &loaded))
	loaded.applyDefaults()
	require.Len(t, loaded.Items, 2)
	assert.Equal(t, "Potion", loaded.Items[0].Name)
	assert.Equal(t, "weapon", loaded.Items[1].Category)
}

func TestRPGSchema_SaveConfigDisabledPersists(t *testing.T) {
	p := NewProject("test")
	p.applyDefaults()
	p.SaveConfig.AutosaveEnabled = false
	p.SaveConfig.GameTitle = "Custom Title"

	data, err := json.Marshal(p)
	require.NoError(t, err)
	var loaded Project
	require.NoError(t, json.Unmarshal(data, &loaded))
	loaded.applyDefaults()
	assert.False(t, loaded.SaveConfig.AutosaveEnabled,
		"explicit-off autosave survives reload via Set sentinel")
	assert.Equal(t, "Custom Title", loaded.SaveConfig.GameTitle)
}

func TestRPGSchema_OmitEmptyWhenAllEmpty(t *testing.T) {
	p := &Project{Scenes: []Scene{{ID: "main"}}}
	data, err := json.Marshal(p)
	require.NoError(t, err)
	s := string(data)
	assert.NotContains(t, s, `"dialogues"`,
		"empty Dialogues map omitted from JSON")
	assert.NotContains(t, s, `"menus"`)
	assert.NotContains(t, s, `"items"`)
}

func TestSaveConfig_DefaultExposed(t *testing.T) {
	d := DefaultSaveConfig()
	assert.True(t, d.AutosaveEnabled)
	assert.True(t, d.Set)
}
