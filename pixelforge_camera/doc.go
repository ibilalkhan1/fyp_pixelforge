// Package pixelforge_camera provides a Mario-style smooth-follow
// camera primitive that drives the engine-wide pixelforge.Camera
// offset each tick.
//
// A Follower is constructed once per active scene with a
// FollowerConfig describing the dead-zone size, velocity look-ahead,
// per-axis smoothing constants, and the level bounds. On every game
// tick the caller invokes Update(playerX, playerY, dt) and the
// follower computes a target offset, eases the camera toward it with
// framerate-independent smoothing, hard-clamps the result to the
// level bounds, and writes the rounded value to the global
// pixelforge.Camera variable.
//
// # Design
//
// The follower implements the canonical Mark Brown / GMTK and Cave
// Story camera pattern:
//
//   - A rectangular dead-zone centered on the camera. When the player
//     moves inside it the camera does not move; when the player
//     leaves it the camera moves so the player stays at the dead-zone
//     edge.
//   - A velocity-based look-ahead (NOT facing-based) so the camera
//     leads in the player's direction of motion. The look-ahead
//     amount eases toward its target so rapid taps left/right do not
//     produce a snap.
//   - Framerate-independent smoothing of the form
//     camera += (target - camera) * (1 - exp(-k * dt))
//     so 30fps, 60fps, and 120fps produce visually identical motion.
//   - Hard-clamp at level bounds applied to the final camera
//     position, not the target. Clamping the target would cancel
//     look-ahead near the edges (the camera would briefly snap back
//     into the level).
//
// # Viewport assumption
//
// The follower clamps the camera to the range
// [0, BoundsW - ViewportW] (and similarly for Y). ViewportW and
// ViewportH default to 256 and 240 (the engine's standard NES-class
// screen size) when left at zero in FollowerConfig; callers may
// override them for non-standard screen sizes. When the level is
// smaller than the viewport on an axis the camera is held at 0 on
// that axis.
//
// # Defaults
//
// The recommended Mario-class defaults are:
//
//   - DeadZoneW   = 77  (30% of 256)
//   - DeadZoneH   = 96  (40% of 240)
//   - LookAheadX  = 32
//   - LookAheadY  = 0
//   - SmoothingX  = 0.20
//   - SmoothingY  = 0.40
//
// These match the values documented in Scene.Camera (see
// pixelforge_project) and the camera follower research in the
// feat-screenroom-mario-strip-v1 plan.
//
// # Thread-safety
//
// Like the rest of pixelforge, this package is not thread-safe.
// Update must be called from the same goroutine that drives the
// engine game loop.
package pixelforge_camera
