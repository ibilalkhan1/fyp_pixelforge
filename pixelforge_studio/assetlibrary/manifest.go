// Package assetlibrary is plan-008 U9–U12's curated asset library.
// The substrate ships here: manifest parsing, first-launch
// downloader with SHA-256 verification + atomic writes, an
// in-memory index of installed packs, a Library workspace browser
// (U12), and a credits assembler (U10).
//
// The actual asset packs themselves are content not code — they
// live in a separate GitHub Release that the downloader fetches
// on first launch. See docs/plans/2026-05-18-008-feat-arcade-
// shipping-v1-plan.md for the content workstream split.
//
// Schema discipline (per docs/solutions/editor-pforge-schema-shape.md):
// manifest fields are additive forever; future schema versions
// load with best-effort sanitisation, never error on unknown
// fields, and surface a clear ErrUnsupportedSchema when the major
// version exceeds what this build understands.
package assetlibrary

import (
	"encoding/json"
	"errors"
	"fmt"
)

// CurrentSchemaVersion is the manifest schema this build was
// written against. Bump only on backwards-incompatible changes;
// additive fields don't need a bump.
const CurrentSchemaVersion = "1"

// Manifest is the root of the JSON document hosted at a GitHub
// Release URL (overridable via PIXELFORGE_ASSET_LIBRARY_URL).
// SchemaVersion gates major-format changes; Packs enumerates
// every downloadable asset bundle.
type Manifest struct {
	SchemaVersion string `json:"schema_version"`
	Packs         []Pack `json:"packs"`
}

// Pack is one downloadable asset bundle (one game's worth of
// sprites/sfx/bgm). The downloader fetches Pack.URL, verifies
// against Pack.SHA256, unpacks to <cacheDir>/library/<ID>/, and
// records the result in the in-memory library index.
type Pack struct {
	// ID is the pack's stable identifier; doubles as the on-disk
	// directory name. lower_snake_case for filesystem safety.
	ID string `json:"id"`

	// Version follows semver; bumping it triggers a re-download.
	Version string `json:"version"`

	// Title is the human-readable pack name (rendered in the
	// Library workspace's pack header).
	Title string `json:"title"`

	// Game is one of "asteroids" / "bomberman" / "mario" /
	// "donkey_kong" / "" for game-neutral packs. Drives the
	// Library workspace's per-game tab filter.
	Game string `json:"game,omitempty"`

	// URL is the HTTPS download location for the pack archive.
	// .tar.gz today; the downloader's archive handler dispatches
	// on extension so future formats slot in additively.
	URL string `json:"url"`

	// SHA256 is the hex-encoded SHA-256 of the pack archive bytes.
	// Downloader verifies before unpacking; mismatch returns
	// ErrChecksumMismatch.
	SHA256 string `json:"sha256"`

	// SizeBytes is the archive's expected byte count. Used for
	// progress UI; not authoritative — verification is by hash.
	SizeBytes int64 `json:"size_bytes,omitempty"`

	// Assets enumerates the individual files inside the pack with
	// their license + author + source attribution. Powers the
	// credits screen (U10) and the per-asset license badge (U12).
	Assets []Asset `json:"assets"`
}

// Asset is one file inside a Pack — a sprite PNG, SFX WAV, or BGM
// loop. Path is relative to the pack's unpacked root.
type Asset struct {
	Path      string `json:"path"`
	Kind      string `json:"kind"` // "sprite" | "sfx" | "bgm"
	License   string `json:"license"`
	Author    string `json:"author,omitempty"`
	SourceURL string `json:"source_url,omitempty"`
}

// ErrUnsupportedSchema is returned by ParseManifest when the
// manifest's SchemaVersion is newer than CurrentSchemaVersion.
// Older versions load via best-effort sanitisation.
var ErrUnsupportedSchema = errors.New("assetlibrary: unsupported manifest schema_version")

// ParseManifest decodes raw JSON manifest bytes into a Manifest.
// Returns ErrUnsupportedSchema for forward-incompatible versions
// (major version > CurrentSchemaVersion). Unknown additive fields
// inside packs/assets are silently dropped per the schema's
// "additive forever" contract.
func ParseManifest(raw []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("assetlibrary: parse manifest: %w", err)
	}
	if m.SchemaVersion != "" && m.SchemaVersion != CurrentSchemaVersion {
		return nil, fmt.Errorf("%w: manifest %q vs this build %q",
			ErrUnsupportedSchema, m.SchemaVersion, CurrentSchemaVersion)
	}
	if m.SchemaVersion == "" {
		// Empty defaults to current; pre-v1 manifests don't exist
		// but the loader is forgiving.
		m.SchemaVersion = CurrentSchemaVersion
	}
	for i := range m.Packs {
		sanitizePack(&m.Packs[i])
	}
	return &m, nil
}

// sanitizePack drops empty assets and normalises optional fields
// so downstream callers don't need to nil-check every entry.
func sanitizePack(p *Pack) {
	if p.Assets == nil {
		p.Assets = []Asset{}
	}
	// Drop assets missing required fields (Path + Kind) — they
	// couldn't be located or routed anyway.
	filtered := p.Assets[:0]
	for _, a := range p.Assets {
		if a.Path == "" || a.Kind == "" {
			continue
		}
		filtered = append(filtered, a)
	}
	p.Assets = filtered
}

// LookupPack returns the pack with the given ID, or nil when no
// pack matches. Used by the Library workspace's per-game tab
// filter and by the credits assembler.
func (m *Manifest) LookupPack(id string) *Pack {
	if m == nil {
		return nil
	}
	for i := range m.Packs {
		if m.Packs[i].ID == id {
			return &m.Packs[i]
		}
	}
	return nil
}
