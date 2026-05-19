package pixelforge_render

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
)

// runtimeScreenDims returns the width + height of the destination
// surface RenderTickAtRGBA should allocate. Prefers the runtime's
// project ScreenWidth/ScreenHeight (so a project authored at 256x224
// renders at native resolution even if the global pixelforge.Screen
// was sized differently by some earlier caller); falls back to the
// global pixelforge.Screen if the project is nil or zero-sized
// (matches the engine's init default of 320x180).
//
// The fallback ensures tests that construct a bare-bones Runtime
// without a fully-populated project still get a valid surface — the
// determinism test in particular does this.
func runtimeScreenDims(rt *capsuleruntime.Runtime) (int, int) {
	if rt != nil && rt.Project != nil &&
		rt.Project.ScreenWidth > 0 && rt.Project.ScreenHeight > 0 {
		return rt.Project.ScreenWidth, rt.Project.ScreenHeight
	}
	scr := pixelforge.Screen()
	return scr.W(), scr.H()
}

// newOffscreen allocates a fresh *ebiten.Image of the given size for
// use as the RGBA-mode render target. Wrapped in a helper so a future
// pooling/reuse strategy (if the per-test allocation cost becomes
// measurable) lands in one place.
func newOffscreen(w, h int) *ebiten.Image {
	return ebiten.NewImage(w, h)
}
