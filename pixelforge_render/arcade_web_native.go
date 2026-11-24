//go:build !js || !wasm

package pixelforge_render

import "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_arcade"

func installArcadeWebControls(_ *pixelforge_arcade.Game) func() { return func() {} }
