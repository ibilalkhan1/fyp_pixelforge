package assetlibrary

import (
	"path/filepath"
)

// libraryRoot returns the directory packs are unpacked under,
// inside the supplied cache root. cacheRoot is typically
// os.UserCacheDir()/pixelforge but tests inject t.TempDir() so
// they don't pollute the user's actual cache.
func libraryRoot(cacheRoot string) string {
	return filepath.Join(cacheRoot, "library")
}

// PackDir returns the on-disk directory the pack with id unpacks
// to. The Library workspace + credits assembler resolve asset
// paths under this directory.
func PackDir(cacheRoot, id string) string {
	return filepath.Join(libraryRoot(cacheRoot), id)
}

// AssetPath returns the absolute path to an asset inside the
// pack's on-disk directory. Used by the workspace's "Add to
// project" copy and the credits assembler's filename → license
// lookup.
func AssetPath(cacheRoot, packID, relPath string) string {
	return filepath.Join(PackDir(cacheRoot, packID), filepath.FromSlash(relPath))
}

// userLibraryRoot returns the watched-folder dir users drop
// custom assets into (U11). Held next to the cache's curated
// library so the two share an obvious convention.
func userLibraryRoot(cacheRoot string) string {
	return filepath.Join(cacheRoot, "user-library")
}

// UserLibraryDir returns the absolute path to the user-library
// directory inside cacheRoot. The watcher (U11) reads from here.
func UserLibraryDir(cacheRoot string) string {
	return userLibraryRoot(cacheRoot)
}
