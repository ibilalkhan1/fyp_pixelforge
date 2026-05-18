package pixelforge_camera

import (
	"math"

	"github.com/ibilalkhan1/fyp_pixelforge"
)

// Default viewport dimensions. The engine ships with a 256x240
// NES-class screen by default; FollowerConfig values left at zero
// for ViewportW / ViewportH fall back to these constants.
const (
	DefaultViewportW = 256
	DefaultViewportH = 240
)

// Default Mario-class follower tuning constants. Callers that
// construct a FollowerConfig with zero values for these fields will
// have the defaults applied by NewFollower. Designers can override
// any of them per-scene through Scene.Camera in the project schema.
const (
	DefaultDeadZoneW = 77.0 // 30% of 256
	DefaultDeadZoneH = 96.0 // 40% of 240
	DefaultLookAheadX = 32.0
	DefaultLookAheadY = 0.0
	DefaultSmoothingX = 0.20
	DefaultSmoothingY = 0.40

	// lookAheadSmoothing controls how quickly the look-ahead component
	// eases toward its velocity-based target. The relatively quick
	// constant (0.05 per the plan) lets the camera adapt to direction
	// changes within ~0.3-0.5 seconds while still damping rapid taps.
	lookAheadSmoothing = 0.05
)

// FollowerConfig describes a follower's tuning. All float64 values
// are in pixels; smoothing constants are in inverse seconds (larger
// = snappier). ViewportW / ViewportH default to DefaultViewportW /
// DefaultViewportH when zero. BoundsW / BoundsH are the total
// pixel size of the level (typically GridWidthScreens *
// ScreenWidth); when smaller than the viewport on an axis the
// camera is held at zero on that axis.
type FollowerConfig struct {
	DeadZoneW, DeadZoneH float64
	LookAheadX, LookAheadY float64
	SmoothingX, SmoothingY float64
	BoundsW, BoundsH int
	ViewportW, ViewportH int
}

// Follower is the stateful Mario-style camera follower. Construct
// one per active scene via NewFollower, then call Update each tick
// with the player's pixel position and the elapsed game-time
// seconds. Update writes the rounded camera offset to the global
// pixelforge.Camera on every call.
//
// The internal camera position (camX, camY) is kept as float64 to
// avoid jitter from int rounding; only the final write to
// pixelforge.Camera is rounded to int.
type Follower struct {
	Config FollowerConfig

	camX, camY float64

	lastPlayerX, lastPlayerY float64
	hasLastPlayer            bool

	lookAheadEasedX, lookAheadEasedY float64
}

// NewFollower builds a Follower with the given configuration. Any
// zero-valued field in cfg that has a documented default falls back
// to that default. The camera state is initialized at (0, 0); the
// first call to Update establishes the player-velocity baseline.
func NewFollower(cfg FollowerConfig) *Follower {
	if cfg.DeadZoneW == 0 {
		cfg.DeadZoneW = DefaultDeadZoneW
	}
	if cfg.DeadZoneH == 0 {
		cfg.DeadZoneH = DefaultDeadZoneH
	}
	if cfg.LookAheadX == 0 {
		cfg.LookAheadX = DefaultLookAheadX
	}
	// LookAheadY defaults to 0 (no vertical look-ahead in v1) so we
	// intentionally do NOT replace a zero here.
	if cfg.SmoothingX == 0 {
		cfg.SmoothingX = DefaultSmoothingX
	}
	if cfg.SmoothingY == 0 {
		cfg.SmoothingY = DefaultSmoothingY
	}
	if cfg.ViewportW == 0 {
		cfg.ViewportW = DefaultViewportW
	}
	if cfg.ViewportH == 0 {
		cfg.ViewportH = DefaultViewportH
	}
	return &Follower{Config: cfg}
}

// CameraX returns the internal float64 camera X position. Exposed
// primarily for tests; production callers should read pixelforge.
// Camera.X.
func (f *Follower) CameraX() float64 { return f.camX }

// CameraY returns the internal float64 camera Y position. Exposed
// primarily for tests; production callers should read pixelforge.
// Camera.Y.
func (f *Follower) CameraY() float64 { return f.camY }

// SetCamera explicitly positions the camera, bypassing smoothing.
// Useful for scene teleports / restarts where the camera should
// jump rather than ease.
func (f *Follower) SetCamera(x, y float64) {
	f.camX = x
	f.camY = y
	f.hasLastPlayer = false
	f.lookAheadEasedX = 0
	f.lookAheadEasedY = 0
	f.writeGlobal()
}

// Update advances the follower one tick. playerX and playerY are
// the player's pixel-space position in the scene; dt is the elapsed
// game time in seconds since the previous call. Update writes the
// final camera offset to the global pixelforge.Camera.
//
// dt <= 0 is treated as a no-op aside from re-publishing the
// existing camera position; this keeps the call site simple at
// scene-start when no real time has elapsed yet.
func (f *Follower) Update(playerX, playerY, dt float64) {
	if dt <= 0 {
		f.writeGlobal()
		return
	}

	// 1. Player velocity (used only for look-ahead direction).
	var velX, velY float64
	if f.hasLastPlayer {
		velX = (playerX - f.lastPlayerX) / dt
		velY = (playerY - f.lastPlayerY) / dt
	}
	f.lastPlayerX = playerX
	f.lastPlayerY = playerY
	f.hasLastPlayer = true

	// 2. Dead-zone correction. The camera wants to keep the player
	//    inside a rectangle of size (DeadZoneW, DeadZoneH) centered
	//    on the current camera position. If the player is inside the
	//    dead-zone, no correction is applied; if outside, the camera
	//    target is offset so the player ends up sitting on the
	//    nearest dead-zone edge after this tick converges.
	halfDZW := f.Config.DeadZoneW / 2
	halfDZH := f.Config.DeadZoneH / 2

	dx := playerX - f.camX
	dy := playerY - f.camY

	var correctionX, correctionY float64
	if dx > halfDZW {
		correctionX = dx - halfDZW
	} else if dx < -halfDZW {
		correctionX = dx + halfDZW
	}
	if dy > halfDZH {
		correctionY = dy - halfDZH
	} else if dy < -halfDZH {
		correctionY = dy + halfDZH
	}

	// 3. Velocity-based look-ahead. Ease the look-ahead component
	//    toward sign(vel) * LookAhead with its own smoothing
	//    constant so rapid direction flips do not snap the camera.
	lookAheadTargetX := signedLookAhead(velX, f.Config.LookAheadX)
	lookAheadTargetY := signedLookAhead(velY, f.Config.LookAheadY)
	// Smoothing constants in the config are documented at the "per-
	// frame at 60fps lerp" scale (e.g. 0.20 means "approach 20% of
	// the remaining gap each 60fps frame"). We convert to a
	// framerate-independent decay rate by scaling dt by 60 so the
	// same constant produces the same per-second behavior at any
	// tick rate.
	laAlpha := 1 - math.Exp(-lookAheadSmoothing*dt*60)
	f.lookAheadEasedX += (lookAheadTargetX - f.lookAheadEasedX) * laAlpha
	f.lookAheadEasedY += (lookAheadTargetY - f.lookAheadEasedY) * laAlpha

	// 4. Camera target = current camera + dead-zone correction +
	//    eased look-ahead. The correction pulls toward the player;
	//    the look-ahead biases in the velocity direction.
	targetX := f.camX + correctionX + f.lookAheadEasedX
	targetY := f.camY + correctionY + f.lookAheadEasedY

	// 5. Framerate-independent smoothing (see comment above on the
	//    dt*60 scale).
	alphaX := 1 - math.Exp(-f.Config.SmoothingX*dt*60)
	alphaY := 1 - math.Exp(-f.Config.SmoothingY*dt*60)
	f.camX += (targetX - f.camX) * alphaX
	f.camY += (targetY - f.camY) * alphaY

	// 6. Hard-clamp the final camera position to level bounds. The
	//    plan is explicit: clamp the camera, NOT the target -
	//    clamping the target breaks look-ahead near edges.
	f.camX = clampCameraAxis(f.camX, f.Config.BoundsW, f.Config.ViewportW)
	f.camY = clampCameraAxis(f.camY, f.Config.BoundsH, f.Config.ViewportH)

	// 7. Write rounded result to the engine-wide camera global.
	f.writeGlobal()
}

// writeGlobal pushes the current camX/camY (rounded) into the
// engine-wide pixelforge.Camera.
func (f *Follower) writeGlobal() {
	pixelforge.Camera.X = int(math.Round(f.camX))
	pixelforge.Camera.Y = int(math.Round(f.camY))
}

// signedLookAhead returns sign(v) * magnitude. A zero velocity
// returns zero so the look-ahead component eases back to neutral
// when the player stops.
func signedLookAhead(v, magnitude float64) float64 {
	if v > 0 {
		return magnitude
	}
	if v < 0 {
		return -magnitude
	}
	return 0
}

// clampCameraAxis bounds a camera coordinate to [0, bounds -
// viewport]. When the level is smaller than the viewport on this
// axis the camera is pinned at 0.
func clampCameraAxis(value float64, bounds, viewport int) float64 {
	maxCam := float64(bounds - viewport)
	if maxCam <= 0 {
		return 0
	}
	if value < 0 {
		return 0
	}
	if value > maxCam {
		return maxCam
	}
	return value
}
