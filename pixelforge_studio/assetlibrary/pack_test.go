package assetlibrary_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/assetlibrary"
)

func TestPackDir_NestsUnderLibraryRoot(t *testing.T) {
	root := t.TempDir()
	got := assetlibrary.PackDir(root, "asteroids")
	assert.Equal(t, filepath.Join(root, "library", "asteroids"), got)
}

func TestAssetPath_JoinsRelativeUnderPack(t *testing.T) {
	root := t.TempDir()
	got := assetlibrary.AssetPath(root, "asteroids", "sprites/ship.png")
	assert.Equal(t, filepath.Join(root, "library", "asteroids", "sprites", "ship.png"), got)
}

func TestUserLibraryDir_SiblingOfLibrary(t *testing.T) {
	root := t.TempDir()
	user := assetlibrary.UserLibraryDir(root)
	assert.Equal(t, filepath.Join(root, "user-library"), user)
	curated := assetlibrary.PackDir(root, "x")
	assert.NotEqual(t, user, curated)
}
