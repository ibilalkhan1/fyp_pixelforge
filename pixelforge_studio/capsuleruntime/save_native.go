//go:build !js

package capsuleruntime

import pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"

// defaultSaveBackend returns the desktop file-system backend bound
// to the game's per-title directory. The build-tag split keeps
// syscall/js out of the desktop build and the file APIs out of the
// WASM build.
func defaultSaveBackend(gameTitle string) (pisave.Backend, error) {
	return pisave.NewBackendNative(gameTitle)
}
