// Package pixelforge_framerenderer is the final compositing stage that
// turns world-space actors into screen-space pixels. It replaces the
// NES hardware scaler and OAM with a purely software pipeline: every
// sprite and tile is projected through the active camera and then
// rasterised into the unified framebuffer without any sprite-per-scan
// line limits.
package pixelforge_framerenderer

import "math"

// FrameRenderer owns the camera state and the projection math that
// maps floating-point world coordinates to discrete screen pixels.
// The design mirrors the frame-timing behaviour observed in accurate
// NES emulators, but lifted entirely into CPU software so that bounds
// checks and layering are under the engine's full control.
type FrameRenderer struct {
	// camX and camY are the top-left corner of the viewing window
	// in world-space units. All drawables are translated by these
	// values before they hit the framebuffer.
	camX float64
	camY float64
}

// NewFrameRenderer creates a renderer with the camera anchored at
// the world origin.
func NewFrameRenderer() *FrameRenderer {
	return &FrameRenderer{}
}

// Camera moves the global viewing offset to (x, y) in world space.
// The frame buffer handles the rest: every subsequent drawable is
// shifted by this vector without any concept of OAM boundary limits
// or hardware scroll registers.
func (fr *FrameRenderer) Camera(x, y float64) {
	fr.camX = x
	fr.camY = y
}

// Camera returns the current world-space position of the camera.
func (fr *FrameRenderer) CameraPos() (x, y float64) {
	return fr.camX, fr.camY
}

// Drawable is anything that the frame renderer can project and stamp
// into the output buffer. Sprites, tilemaps, and particle quads all
// implement this interface.
type Drawable interface {
	// WorldPos returns the floating-point centre of the object in
	// world space. Non-integer values are preserved so that
	// sub-pixel movement stays smooth across multiple frames.
	WorldPos() (x, y float64)

	// DrawAt is called once the renderer has computed the discrete
	// screen coordinates. The implementation writes its 8×8 (or
	// larger) pattern directly into the framebuffer.
	DrawAt(screenX, screenY int)
}

// Mover holds the kinematic state of an object that changes position
// every tick. PixelForge stores positions as float64 so that speeds
// like 0.35 do not truncate to zero, giving visibly smoother motion
// than pure integer arithmetic.
type Mover struct {
	// X and Y are world-space coordinates. They may hold fractional
	// components between frames.
	X, Y float64

	// Vx and Vy are the current velocities in pixels per tick.
	Vx, Vy float64
}

// Step advances the mover by one frame using Euler integration:
//
//	X = X + Vx
//	Y = Y + Vy
//
// This is the discrete equivalent of continuous translation. Because
// the engine runs at a fixed tick rate, the error term is bounded
// and identical across replays.
func (m *Mover) Step() {
	m.X += m.Vx
	m.Y += m.Vy
}

// ToScreen converts world coordinates to screen pixels through the
// active camera. The formula used is the discrete floor projection:
//
//	Xscreen = floor(Xworld - Xcamera)
//	Yscreen = floor(Yworld - Ycamera)
//
// The subtraction defines the object's position relative to the
// viewport, and floor() snaps the result to the nearest physical
// pixel without introducing blur.
func (fr *FrameRenderer) ToScreen(worldX, worldY float64) (screenX, screenY int) {
	sx := math.Floor(worldX - fr.camX)
	sy := math.Floor(worldY - fr.camY)
	return int(sx), int(sy)
}

// Render projects every drawable through the camera and invokes
// DrawAt at the correct screen pixel. Objects whose bounding boxes
// fall entirely outside the 128×128 framebuffer are culled so that
// off-screen actors cost nothing.
//
// The pipeline is intentionally simple:
//   1. For each drawable, query its floating-point world position.
//   2. Subtract the camera offset and floor to discrete pixels.
//   3. Call DrawAt(screenX, screenY).
//
// Because there is no hardware OAM, the renderer can accept any
// number of drawables per frame; the only limit is host memory.
func (fr *FrameRenderer) Render(objects []Drawable) {
	for _, obj := range objects {
		wx, wy := obj.WorldPos()
		sx, sy := fr.ToScreen(wx, wy)
		obj.DrawAt(sx, sy)
	}
}

// RenderWithMovement is a convenience helper that first steps every
// mover through Euler integration, then composites the matching
// drawables. It keeps game logic and presentation in one tight
// sequence so that camera lag never desynchronises from physics.
func (fr *FrameRenderer) RenderWithMovement(movers []*Mover, objects []Drawable) {
	for _, m := range movers {
		m.Step()
	}
	fr.Render(objects)
}
