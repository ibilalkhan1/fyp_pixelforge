package pixelforge_camera

import (
	"math"
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
)

// resetGlobalCamera clears the engine-wide camera between tests so
// state does not leak between cases. The pixelforge package is
// single-threaded by design, so tests run sequentially.
func resetGlobalCamera() {
	pixelforge.Camera.X = 0
	pixelforge.Camera.Y = 0
}

// defaultConfig returns a follower config sized for a long
// horizontal Mario-strip level (16 screens wide, 1 screen tall) with
// the documented Mario-class tuning.
func defaultConfig() FollowerConfig {
	return FollowerConfig{
		DeadZoneW:  DefaultDeadZoneW,
		DeadZoneH:  DefaultDeadZoneH,
		LookAheadX: DefaultLookAheadX,
		LookAheadY: DefaultLookAheadY,
		SmoothingX: DefaultSmoothingX,
		SmoothingY: DefaultSmoothingY,
		BoundsW:    256 * 16,
		BoundsH:    240,
		ViewportW:  256,
		ViewportH:  240,
	}
}

func TestFollower_PlayerInDeadZone_CameraStill(t *testing.T) {
	resetGlobalCamera()
	f := NewFollower(defaultConfig())
	f.SetCamera(128, 120)

	// First tick establishes the velocity baseline (no motion yet).
	f.Update(128, 120, 1.0/60.0)
	beforeX, beforeY := f.CameraX(), f.CameraY()

	// Move the player a tiny amount, well within the dead-zone
	// (DeadZoneW/2 = 38.5, DeadZoneH/2 = 48).
	f.Update(130, 121, 1.0/60.0)

	if math.Abs(f.CameraX()-beforeX) > 0.5 {
		t.Errorf("camera X moved unexpectedly: before=%.3f after=%.3f", beforeX, f.CameraX())
	}
	if math.Abs(f.CameraY()-beforeY) > 0.5 {
		t.Errorf("camera Y moved unexpectedly: before=%.3f after=%.3f", beforeY, f.CameraY())
	}
}

func TestFollower_PlayerExitsDeadZone_CameraFollows(t *testing.T) {
	resetGlobalCamera()
	f := NewFollower(defaultConfig())
	f.SetCamera(128, 120)

	// Player jumps to (200, 120). The horizontal offset from the
	// camera is +72, which exceeds DeadZoneW/2 = 38.5; the camera
	// should ease toward the player.
	const dt = 1.0 / 60.0
	for i := 0; i < 600; i++ { // 10s of game time
		f.Update(200, 120, dt)
	}

	// After convergence the player should sit on the trailing
	// dead-zone edge: camera = player - DeadZoneW/2.
	wantX := 200.0 - DefaultDeadZoneW/2
	if math.Abs(f.CameraX()-wantX) > 1.0 {
		t.Errorf("camera X did not converge: got=%.3f want=%.3f", f.CameraX(), wantX)
	}
}

func TestFollower_LookAheadVelocityBased(t *testing.T) {
	resetGlobalCamera()
	f := NewFollower(defaultConfig())
	f.SetCamera(128, 120)

	// Move the player right by 5px per tick for a while so look-
	// ahead builds up. Then reverse direction and verify the look-
	// ahead component eventually flips sign.
	const dt = 1.0 / 60.0
	playerX := 128.0
	for i := 0; i < 60; i++ {
		playerX += 5
		f.Update(playerX, 120, dt)
	}
	if f.lookAheadEasedX <= 0 {
		t.Fatalf("expected positive look-ahead while moving right, got %.3f", f.lookAheadEasedX)
	}

	// Reverse direction. The look-ahead should flip sign within
	// roughly half a second of sustained reverse motion.
	for i := 0; i < 60; i++ {
		playerX -= 5
		f.Update(playerX, 120, dt)
	}
	if f.lookAheadEasedX >= 0 {
		t.Errorf("expected negative look-ahead after reversing direction, got %.3f", f.lookAheadEasedX)
	}
}

func TestFollower_HardClampAtLevelEdges(t *testing.T) {
	resetGlobalCamera()
	cfg := defaultConfig()
	f := NewFollower(cfg)

	// Drive the player past the right edge of the level. The level
	// is 16*256 = 4096 px wide; the viewport is 256 px; max camera
	// X is 4096 - 256 = 3840.
	const dt = 1.0 / 60.0
	playerX := 0.0
	for playerX < float64(cfg.BoundsW)+200 {
		playerX += 16
		f.Update(playerX, 120, dt)
	}
	// Hold the player past the edge for long enough to fully
	// converge against the clamp.
	for i := 0; i < 600; i++ {
		f.Update(playerX, 120, dt)
	}

	maxCam := float64(cfg.BoundsW - cfg.ViewportW)
	if f.CameraX() > maxCam+0.001 {
		t.Errorf("camera X exceeded clamp: got=%.3f max=%.3f", f.CameraX(), maxCam)
	}
	if f.CameraX() < maxCam-1.0 {
		t.Errorf("camera X did not reach clamp: got=%.3f max=%.3f", f.CameraX(), maxCam)
	}

	// Also verify the negative bound: drive player back to 0 and
	// past; camera must not go negative.
	for i := 0; i < 600; i++ {
		f.Update(0, 120, dt)
	}
	if f.CameraX() < 0 {
		t.Errorf("camera X went negative: got=%.3f", f.CameraX())
	}
}

func TestFollower_NoSnapOnFacingFlip(t *testing.T) {
	resetGlobalCamera()
	f := NewFollower(defaultConfig())
	f.SetCamera(128, 120)

	const dt = 1.0 / 60.0
	// Establish baseline.
	f.Update(128, 120, dt)

	// Rapidly tap left-right (5 px each direction). Track the
	// maximum per-tick camera jump; it must stay small (no snap).
	playerX := 128.0
	prevCamX := f.CameraX()
	var maxJump float64
	for i := 0; i < 120; i++ {
		if i%2 == 0 {
			playerX += 5
		} else {
			playerX -= 5
		}
		f.Update(playerX, 120, dt)
		jump := math.Abs(f.CameraX() - prevCamX)
		if jump > maxJump {
			maxJump = jump
		}
		prevCamX = f.CameraX()
	}

	// A snap from facing-based look-ahead would produce per-tick
	// jumps of ~LookAheadX = 32 px. Velocity-based + eased look-
	// ahead must keep per-tick jumps well below 5 px.
	if maxJump > 5.0 {
		t.Errorf("camera snapped during facing flip: maxJump=%.3f px/tick", maxJump)
	}
}

func TestFollower_FramerateIndependent(t *testing.T) {
	resetGlobalCamera()
	// Two identical scenarios at 30fps and 120fps for 1 second of
	// simulated game time. Final camera X must agree within 1 px.
	run := func(tps int) float64 {
		resetGlobalCamera()
		f := NewFollower(defaultConfig())
		f.SetCamera(128, 120)
		dt := 1.0 / float64(tps)
		// Player walks at a constant 60 px/sec to the right.
		const speed = 60.0
		playerX := 128.0
		for i := 0; i < tps; i++ {
			playerX += speed * dt
			f.Update(playerX, 120, dt)
		}
		return f.CameraX()
	}
	cam30 := run(30)
	cam120 := run(120)
	diff := math.Abs(cam30 - cam120)
	if diff > 1.0 {
		t.Errorf("framerate dependence detected: cam30=%.3f cam120=%.3f diff=%.3f", cam30, cam120, diff)
	}
}

func TestFollower_VerticalNoMovementWhenSingleScreenTall(t *testing.T) {
	resetGlobalCamera()
	cfg := defaultConfig()
	cfg.BoundsH = 240 // single-screen-tall: BoundsH == viewport height
	f := NewFollower(cfg)

	const dt = 1.0 / 60.0
	// Move player vertically all over the place; camera Y must stay
	// pinned at 0 because the camera has no room to scroll.
	for i := 0; i < 120; i++ {
		py := 120.0 + 80.0*math.Sin(float64(i)*0.1)
		f.Update(128, py, dt)
		if f.CameraY() != 0 {
			t.Fatalf("camera Y should stay 0 on single-screen-tall level; got %.3f at tick %d", f.CameraY(), i)
		}
		if pixelforge.Camera.Y != 0 {
			t.Fatalf("global camera Y leaked nonzero on single-screen-tall level; got %d", pixelforge.Camera.Y)
		}
	}
}

func TestFollower_WritesPixelforgeCamera(t *testing.T) {
	resetGlobalCamera()
	f := NewFollower(defaultConfig())
	f.SetCamera(128, 120)

	const dt = 1.0 / 60.0
	// Several ticks of movement to advance the camera.
	playerX := 128.0
	for i := 0; i < 60; i++ {
		playerX += 4
		f.Update(playerX, 120, dt)
	}

	wantX := int(math.Round(f.CameraX()))
	wantY := int(math.Round(f.CameraY()))
	if pixelforge.Camera.X != wantX {
		t.Errorf("pixelforge.Camera.X = %d, want %d (= round(%.6f))", pixelforge.Camera.X, wantX, f.CameraX())
	}
	if pixelforge.Camera.Y != wantY {
		t.Errorf("pixelforge.Camera.Y = %d, want %d (= round(%.6f))", pixelforge.Camera.Y, wantY, f.CameraY())
	}
}

func TestNewFollower_AppliesDefaults(t *testing.T) {
	// A fully zero-valued config should pick up the documented
	// defaults (except LookAheadY, which legitimately defaults to
	// zero for the Mario-strip case).
	f := NewFollower(FollowerConfig{})
	if f.Config.DeadZoneW != DefaultDeadZoneW {
		t.Errorf("DeadZoneW default = %.3f, want %.3f", f.Config.DeadZoneW, DefaultDeadZoneW)
	}
	if f.Config.DeadZoneH != DefaultDeadZoneH {
		t.Errorf("DeadZoneH default = %.3f, want %.3f", f.Config.DeadZoneH, DefaultDeadZoneH)
	}
	if f.Config.LookAheadX != DefaultLookAheadX {
		t.Errorf("LookAheadX default = %.3f, want %.3f", f.Config.LookAheadX, DefaultLookAheadX)
	}
	if f.Config.SmoothingX != DefaultSmoothingX {
		t.Errorf("SmoothingX default = %.3f, want %.3f", f.Config.SmoothingX, DefaultSmoothingX)
	}
	if f.Config.SmoothingY != DefaultSmoothingY {
		t.Errorf("SmoothingY default = %.3f, want %.3f", f.Config.SmoothingY, DefaultSmoothingY)
	}
	if f.Config.ViewportW != DefaultViewportW {
		t.Errorf("ViewportW default = %d, want %d", f.Config.ViewportW, DefaultViewportW)
	}
	if f.Config.ViewportH != DefaultViewportH {
		t.Errorf("ViewportH default = %d, want %d", f.Config.ViewportH, DefaultViewportH)
	}
}
