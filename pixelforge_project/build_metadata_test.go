package pixelforge_project

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// build_metadata_test.go covers idea #7 v1 U2's Project.Version +
// IconSpriteName fields.

func TestProject_VersionEmptyDefaultsToISODate(t *testing.T) {
	const pre = `{"schema_version": 1, "name": "test", "scenes": []}`
	p := loadFromBytes(t, []byte(pre))
	today := time.Now().Format("2006-01-02")
	assert.Equal(t, today, p.Version,
		"empty Version defaults to today's ISO date for shipped-artifact stamping")
}

func TestProject_VersionExplicitPreserved(t *testing.T) {
	pre := `{"schema_version": 1, "name": "test", "scenes": [], "version": "1.2.3"}`
	p := loadFromBytes(t, []byte(pre))
	assert.Equal(t, "1.2.3", p.Version)
}

func TestProject_IconSpriteNamePersisted(t *testing.T) {
	p := NewProject("test")
	p.Sprites = []SpriteAsset{{Name: "hero"}}
	p.IconSpriteName = "hero"
	data, err := json.Marshal(p)
	require.NoError(t, err)
	var loaded Project
	require.NoError(t, json.Unmarshal(data, &loaded))
	loaded.applyDefaults()
	assert.Equal(t, "hero", loaded.IconSpriteName)
}

func TestProject_IconSpriteNameSanitisedWhenSpriteMissing(t *testing.T) {
	p := NewProject("test")
	p.IconSpriteName = "ghost" // no such sprite
	p.applyDefaults()
	assert.Empty(t, p.IconSpriteName,
		"dangling IconSpriteName clears so auto-pick takes over")
}

func TestProject_IconSpriteNameOmitEmpty(t *testing.T) {
	p := &Project{Name: "test", Scenes: []Scene{{ID: "m"}}}
	data, err := json.Marshal(p)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"icon_sprite_name"`,
		"empty IconSpriteName omitted from JSON")
}

func TestProject_VersionIsISODateFormat(t *testing.T) {
	p := NewProject("test")
	p.applyDefaults()
	_, err := time.Parse("2006-01-02", p.Version)
	assert.NoError(t, err, "auto-stamped Version parses as ISO date")
}
