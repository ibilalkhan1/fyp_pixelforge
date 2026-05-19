package starterpack

import (
	"embed"
	"io/fs"
	"sync"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/assetlibrary"
)

// assetsFS is the embedded CC0 starter set. `all:` includes files
// whose names start with `.` or `_` so future authors can drop a
// `.license-manifest` next to the assets without surprises.
//
//go:embed all:assets
var assetsFS embed.FS

// StarterPackID is the canonical pack identifier used in the
// assetlibrary index. Stable across releases — projects that
// reference "starter/sprites/hero.png" continue resolving after
// updates.
const StarterPackID = "starter"

// StarterPackVersion bumps when the embedded set changes shape
// (sprite added/removed). Additions only — never remove entries
// projects might be referencing.
const StarterPackVersion = "1.0.0"

// StarterFS returns the embedded assets rooted at "assets/", so
// callers can `fs.ReadFile(StarterFS(), "sprites/hero.png")`
// without the "assets/" prefix leaking into asset paths.
func StarterFS() fs.FS {
	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		// Panic only fires if the embed directive failed at compile
		// time — caught by the test in embed_test.go.
		panic("starterpack: embedded assets/ missing: " + err.Error())
	}
	return sub
}

// starterAssets enumerates the embedded files plus their declared
// kinds. Mirrors the directory layout under assets/ — the list is
// hand-maintained so a typo in a sprite name fails the test in
// embed_test.go rather than silently shipping a broken pack.
var starterAssets = []assetlibrary.Asset{
	// 8 placeholder sprites — generic player/enemy/world tiles.
	{Path: "sprites/hero.png", Kind: "sprite", License: "CC0", Author: "Pixelforge starter pack", SourceURL: ""},
	{Path: "sprites/enemy.png", Kind: "sprite", License: "CC0", Author: "Pixelforge starter pack"},
	{Path: "sprites/coin.png", Kind: "sprite", License: "CC0", Author: "Pixelforge starter pack"},
	{Path: "sprites/platform.png", Kind: "sprite", License: "CC0", Author: "Pixelforge starter pack"},
	{Path: "sprites/wall.png", Kind: "sprite", License: "CC0", Author: "Pixelforge starter pack"},
	{Path: "sprites/door.png", Kind: "sprite", License: "CC0", Author: "Pixelforge starter pack"},
	{Path: "sprites/key.png", Kind: "sprite", License: "CC0", Author: "Pixelforge starter pack"},
	{Path: "sprites/heart.png", Kind: "sprite", License: "CC0", Author: "Pixelforge starter pack"},

	// 2 placeholder SFX — silent 16-bit PCM WAVs. The audio sink
	// (U7) accepts them; the runtime plays silence until the
	// downloaded library overrides them.
	{Path: "sfx/jump.wav", Kind: "sfx", License: "CC0", Author: "Pixelforge starter pack"},
	{Path: "sfx/hit.wav", Kind: "sfx", License: "CC0", Author: "Pixelforge starter pack"},

	// 1 placeholder BGM — per U19 the BGM decoder is stubbed, so
	// this byte-content proves the wiring; audible BGM lands with
	// the audio-v2 polish.
	{Path: "bgm/level_theme.ogg", Kind: "bgm", License: "CC0", Author: "Pixelforge starter pack"},
}

// sizeOnce + sizeCached memoise the embedded-byte-count walk so
// repeated StarterPack() calls don't re-stat the embed.FS.
var (
	sizeOnce   sync.Once
	sizeCached int64
)

// StarterPack returns the assetlibrary.Pack metadata describing
// the embedded set. The URL + SHA256 fields are intentionally
// empty — embedded packs aren't downloadable, and the library's
// SkipIfInstalled path keys off Version, not hash. Callers that
// want the raw bytes resolve them through StarterFS() instead of
// assetlibrary.AssetPath (which assumes on-disk packs).
func StarterPack() assetlibrary.Pack {
	sizeOnce.Do(func() {
		sizeCached = totalEmbeddedSize()
	})
	// Defensive copy so callers can't mutate our internal slice.
	assets := make([]assetlibrary.Asset, len(starterAssets))
	copy(assets, starterAssets)
	return assetlibrary.Pack{
		ID:        StarterPackID,
		Version:   StarterPackVersion,
		Title:     "Starter Pack",
		Game:      "", // generic — appears under every game's tab + "All".
		URL:       "",
		SHA256:    "",
		SizeBytes: sizeCached,
		Assets:    assets,
	}
}

// totalEmbeddedSize walks the embedded FS once and sums every
// regular file's byte count. Used for the Pack.SizeBytes display
// in the workspace's "where do my assets live?" status pane.
func totalEmbeddedSize() int64 {
	var total int64
	_ = fs.WalkDir(StarterFS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total
}
