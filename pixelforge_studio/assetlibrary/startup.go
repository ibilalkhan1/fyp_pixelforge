package assetlibrary

import (
	"context"
	"log"
	"os"
	"path/filepath"
)

// StudioCacheRoot returns the canonical per-host cache directory
// the studio uses for the asset library. Falls back to
// os.TempDir() when UserCacheDir is unavailable (fresh container,
// non-standard home). Tests inject t.TempDir() through NewLibrary
// directly and don't call this helper.
func StudioCacheRoot() string {
	root, err := os.UserCacheDir()
	if err != nil {
		root = os.TempDir()
	}
	return filepath.Join(root, "pixelforge")
}

// LaunchBackgroundBootstrap kicks off EnsureBootstrap in a
// goroutine bound to ctx. Returns the library so the studio's
// chrome can read installed packs as they land. Per-pack failures
// log + are dropped so a flaky network on studio startup doesn't
// block the editor; complete failure (manifest unreachable) is
// also non-fatal — the user still gets the studio.
//
// Callers that want fully-offline operation skip this helper
// (the studio's --no-asset-library flag).
func LaunchBackgroundBootstrap(ctx context.Context) *Library {
	cacheRoot := StudioCacheRoot()
	lib := NewLibrary(cacheRoot)
	go func() {
		err := EnsureBootstrap(ctx, lib, DownloadOptions{
			Progress: func(ev Event) {
				if ev.Err != nil {
					log.Printf("[assetlibrary] %s pack=%s: %v", eventKindName(ev.Kind), ev.PackID, ev.Err)
				}
			},
		})
		if err != nil {
			// Manifest fetch or schema-mismatch failure — log
			// once and move on.
			log.Printf("[assetlibrary] bootstrap failed (studio still functional): %v", err)
		}
	}()
	return lib
}

func eventKindName(k EventKind) string {
	switch k {
	case EventManifestFetched:
		return "manifest-fetch"
	case EventPackDownloaded:
		return "pack-download"
	case EventPackInstalled:
		return "pack-install"
	case EventPackSkipped:
		return "pack-skip"
	}
	return "unknown"
}
