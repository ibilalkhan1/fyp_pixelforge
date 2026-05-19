//go:build js

package capsuleruntime

import pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"

// defaultSaveBackend returns the WASM localStorage backend bound to
// the game's per-title prefix. The native backend's filesystem APIs
// are unavailable on js/wasm, so the build-tag split keeps each
// platform compiling cleanly.
func defaultSaveBackend(gameTitle string) (pisave.Backend, error) {
	return pisave.NewBackendJS(gameTitle), nil
}
