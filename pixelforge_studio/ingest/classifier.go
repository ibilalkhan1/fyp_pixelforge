// Package ingest is plan-008 U11's custom-asset routing layer.
// Two complementary entry points let users add their own (or
// internet-sourced) assets to the library:
//
//   - Watcher polls a sibling user-library directory under
//     os.UserCacheDir()/pixelforge/user-library/ via fsnotify;
//     files appearing there are debounced + classified + routed
//     through the dispatcher.
//   - DragDrop reads ebiten.AppendDroppedFiles every frame and
//     funnels drops through the same dispatcher.
//
// Both paths converge on Dispatcher.Ingest(path) — the imperative
// API tests assert against directly without standing up Ebitengine
// input mocks or filesystem watchers.
package ingest

import (
	"path/filepath"
	"strings"
)

// AssetKind names the family the classifier routes an asset into.
// Matches the assetlibrary.Asset.Kind taxonomy so the curated
// library + custom ingest share one vocabulary.
type AssetKind int

const (
	// KindUnknown is the fallback for extensions the classifier
	// doesn't recognise. Dispatcher.Ingest no-ops on this kind.
	KindUnknown AssetKind = iota

	// KindSprite is PNG. Routes through SpriteRunner.
	KindSprite

	// KindSFX is WAV. Routes through SFXRunner.
	KindSFX

	// KindBGM is OGG or MP3 (longer-form looping audio). Routes
	// through BGMRunner.
	KindBGM
)

// String returns the lowercase kind name for diagnostics.
func (k AssetKind) String() string {
	switch k {
	case KindSprite:
		return "sprite"
	case KindSFX:
		return "sfx"
	case KindBGM:
		return "bgm"
	}
	return "unknown"
}

// Classify reads the file extension (case-insensitive) and
// returns the matching kind. Unrecognised extensions return
// KindUnknown so the dispatcher's Ingest no-ops silently —
// dropping a random .txt onto the studio shouldn't crash or toast.
func Classify(path string) AssetKind {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return KindSprite
	case ".wav":
		return KindSFX
	case ".ogg", ".mp3":
		return KindBGM
	}
	return KindUnknown
}

// IsAssetExtension reports whether the extension belongs to any
// recognised asset family. Useful for the watcher's first-pass
// filter so we don't queue every .txt change in the user-library
// directory for debouncing.
func IsAssetExtension(path string) bool {
	return Classify(path) != KindUnknown
}
