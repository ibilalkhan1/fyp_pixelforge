package assetlibrary_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/assetlibrary"
)

const minimalManifestJSON = `{
  "schema_version": "1",
  "packs": [
    {
      "id": "asteroids-starter",
      "version": "1.0.0",
      "title": "Asteroids Starter",
      "game": "asteroids",
      "url": "https://example.invalid/asteroids-1.0.0.tar.gz",
      "sha256": "deadbeef",
      "size_bytes": 1024,
      "assets": [
        {"path": "sprites/ship.png", "kind": "sprite", "license": "CC0", "author": "Kenney", "source_url": "https://kenney.nl/"},
        {"path": "audio/blast.wav",  "kind": "sfx",    "license": "CC-BY-4.0", "author": "freesound user X", "source_url": "https://freesound.org/x"}
      ]
    },
    {
      "id": "shared",
      "version": "0.1.0",
      "title": "Shared assets",
      "url": "https://example.invalid/shared-0.1.0.tar.gz",
      "sha256": "abc",
      "assets": []
    }
  ]
}`

func TestParseManifest_HappyPath(t *testing.T) {
	m, err := assetlibrary.ParseManifest([]byte(minimalManifestJSON))
	require.NoError(t, err)
	assert.Equal(t, "1", m.SchemaVersion)
	require.Len(t, m.Packs, 2)
	assert.Equal(t, "asteroids-starter", m.Packs[0].ID)
	assert.Equal(t, "asteroids", m.Packs[0].Game)
	require.Len(t, m.Packs[0].Assets, 2)
	assert.Equal(t, "CC-BY-4.0", m.Packs[0].Assets[1].License)
}

func TestParseManifest_UnsupportedSchemaErrors(t *testing.T) {
	m, err := assetlibrary.ParseManifest([]byte(`{"schema_version": "99"}`))
	assert.Nil(t, m)
	require.Error(t, err)
	assert.True(t, errors.Is(err, assetlibrary.ErrUnsupportedSchema),
		"schema mismatch must surface ErrUnsupportedSchema; got %v", err)
}

func TestParseManifest_EmptySchemaVersionDefaults(t *testing.T) {
	m, err := assetlibrary.ParseManifest([]byte(`{"packs": []}`))
	require.NoError(t, err)
	assert.Equal(t, assetlibrary.CurrentSchemaVersion, m.SchemaVersion)
}

func TestParseManifest_DropsAssetsMissingPathOrKind(t *testing.T) {
	const bad = `{
		"schema_version": "1",
		"packs": [{"id": "p", "version": "1", "title": "t", "url": "u", "sha256": "h",
			"assets": [
				{"path": "ok.png", "kind": "sprite", "license": "CC0"},
				{"path": "", "kind": "sprite", "license": "CC0"},
				{"path": "ok2.png", "kind": "", "license": "CC0"}
			]}]}`
	m, err := assetlibrary.ParseManifest([]byte(bad))
	require.NoError(t, err)
	require.Len(t, m.Packs[0].Assets, 1, "malformed assets must be filtered out")
}

func TestParseManifest_InvalidJSONErrors(t *testing.T) {
	_, err := assetlibrary.ParseManifest([]byte(`{not json`))
	require.Error(t, err)
}

func TestManifest_LookupPack(t *testing.T) {
	m, err := assetlibrary.ParseManifest([]byte(minimalManifestJSON))
	require.NoError(t, err)
	pack := m.LookupPack("asteroids-starter")
	require.NotNil(t, pack)
	assert.Equal(t, "asteroids", pack.Game)
	assert.Nil(t, m.LookupPack("missing"))
}

// Plan-009 U20: manifest gains an additive `Examples` field
// pointing at reference `.pforge` projects the studio's File →
// Open Example menu can fetch.

const manifestWithExamplesJSON = `{
  "schema_version": "1",
  "packs": [],
  "examples": [
    {
      "id": "asteroids_proof",
      "version": "1.0.0",
      "title": "Asteroids — Reference",
      "game": "asteroids",
      "url": "https://example.invalid/asteroids_proof-1.0.0.pforge",
      "sha256": "feedface",
      "size_bytes": 2048,
      "label": "Asteroids (reference build)",
      "description": "Thrust, fire, screen-wrap demo."
    },
    {
      "id": "mario_proof",
      "version": "1.0.0",
      "title": "Mario — Reference",
      "game": "mario",
      "url": "https://example.invalid/mario_proof-1.0.0.pforge",
      "sha256": "cafebabe"
    }
  ]
}`

func TestParseManifest_WithExamplesPopulatesField(t *testing.T) {
	m, err := assetlibrary.ParseManifest([]byte(manifestWithExamplesJSON))
	require.NoError(t, err)
	require.Len(t, m.Examples, 2, "both well-formed examples must survive sanitisation")
	assert.Equal(t, "asteroids_proof", m.Examples[0].ID)
	assert.Equal(t, "Asteroids (reference build)", m.Examples[0].Label)
	assert.Equal(t, int64(2048), m.Examples[0].SizeBytes)
	assert.Equal(t, "mario_proof", m.Examples[1].ID)
}

func TestParseManifest_WithoutExamplesYieldsEmptySlice(t *testing.T) {
	// Forward-compat: pre-U20 manifests don't carry the field.
	m, err := assetlibrary.ParseManifest([]byte(minimalManifestJSON))
	require.NoError(t, err)
	assert.NotNil(t, m.Examples, "Examples must default to non-nil even when absent in JSON")
	assert.Empty(t, m.Examples)
}

func TestParseManifest_DropsExamplesMissingRequiredFields(t *testing.T) {
	const bad = `{
		"schema_version": "1",
		"packs": [],
		"examples": [
			{"id": "ok", "version": "1", "url": "https://example.invalid/ok.pforge", "sha256": "h"},
			{"id": "", "version": "1", "url": "https://example.invalid/no-id.pforge", "sha256": "h"},
			{"id": "no-url", "version": "1", "url": "", "sha256": "h"}
		]
	}`
	m, err := assetlibrary.ParseManifest([]byte(bad))
	require.NoError(t, err)
	require.Len(t, m.Examples, 1, "examples missing id or url must be filtered out")
	assert.Equal(t, "ok", m.Examples[0].ID)
}

func TestManifest_LookupExample(t *testing.T) {
	m, err := assetlibrary.ParseManifest([]byte(manifestWithExamplesJSON))
	require.NoError(t, err)

	ex := m.LookupExample("asteroids_proof")
	require.NotNil(t, ex)
	assert.Equal(t, "asteroids", ex.Game)
	assert.Equal(t, "Thrust, fire, screen-wrap demo.", ex.Description)

	assert.Nil(t, m.LookupExample("missing"))

	var nilManifest *assetlibrary.Manifest
	assert.Nil(t, nilManifest.LookupExample("x"), "nil-receiver lookup is safe")
}
