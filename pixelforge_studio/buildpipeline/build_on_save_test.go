package buildpipeline_test

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/buildpipeline"
)

func TestHostTarget_ReturnsCurrentPlatform(t *testing.T) {
	got := buildpipeline.HostTarget()
	switch runtime.GOOS {
	case "windows":
		assert.Equal(t, buildpipeline.TargetWindows, got)
	case "darwin":
		assert.Equal(t, buildpipeline.TargetMacOS, got)
	case "linux":
		assert.Equal(t, buildpipeline.TargetLinux, got)
	}
}

func TestBuildOnSaveDaemon_OnSaveSchedulesTrigger(t *testing.T) {
	var triggered atomic.Int32
	d := buildpipeline.NewBuildOnSaveDaemon(
		func() bool { return true },
		func() { triggered.Add(1) },
	)
	d.SetDebounce(20 * time.Millisecond)
	d.OnSave()
	time.Sleep(60 * time.Millisecond)
	assert.Equal(t, int32(1), triggered.Load())
}

func TestBuildOnSaveDaemon_RapidSavesCoalesceIntoOneBuild(t *testing.T) {
	var triggered atomic.Int32
	d := buildpipeline.NewBuildOnSaveDaemon(
		func() bool { return true },
		func() { triggered.Add(1) },
	)
	d.SetDebounce(30 * time.Millisecond)
	for i := 0; i < 5; i++ {
		d.OnSave()
		time.Sleep(5 * time.Millisecond)
	}
	// Wait for the final debounce window to elapse.
	time.Sleep(70 * time.Millisecond)
	assert.Equal(t, int32(1), triggered.Load(),
		"5 rapid saves coalesce into 1 build")
}

func TestBuildOnSaveDaemon_DisabledCheckSuppressesBuild(t *testing.T) {
	var triggered atomic.Int32
	d := buildpipeline.NewBuildOnSaveDaemon(
		func() bool { return false }, // disabled
		func() { triggered.Add(1) },
	)
	d.SetDebounce(10 * time.Millisecond)
	d.OnSave()
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, int32(0), triggered.Load())
}

func TestBuildOnSaveDaemon_TriggerForcesImmediateFire(t *testing.T) {
	var triggered atomic.Int32
	d := buildpipeline.NewBuildOnSaveDaemon(
		func() bool { return true },
		func() { triggered.Add(1) },
	)
	d.SetDebounce(1 * time.Hour) // ridiculous, then forced.
	d.OnSave()
	d.Trigger()
	assert.Equal(t, int32(1), triggered.Load())
}

func TestBuildOnSaveDaemon_UpdateStatusAndLatestStatus(t *testing.T) {
	d := buildpipeline.NewBuildOnSaveDaemon(nil, nil)
	_, seen := d.LatestStatus()
	assert.False(t, seen)

	d.UpdateStatus(buildpipeline.BuildStatus{
		Target: buildpipeline.TargetLinux,
		Phase:  buildpipeline.PhaseDone,
	})
	got, seen := d.LatestStatus()
	require.True(t, seen)
	assert.Equal(t, buildpipeline.PhaseDone, got.Phase)
	assert.Equal(t, buildpipeline.TargetLinux, got.Target)
}

func TestBuildOnSaveDaemon_NilSafeOnAllMethods(t *testing.T) {
	var d *buildpipeline.BuildOnSaveDaemon
	assert.NotPanics(t, func() {
		d.OnSave()
		d.Trigger()
		d.UpdateStatus(buildpipeline.BuildStatus{})
		_, _ = d.LatestStatus()
		_ = d.HasPendingTimer()
		d.SetDebounce(time.Second)
	})
}

func TestBuildOnSaveDaemon_HasPendingTimerAfterSave(t *testing.T) {
	d := buildpipeline.NewBuildOnSaveDaemon(
		func() bool { return true },
		func() {},
	)
	d.SetDebounce(1 * time.Hour)
	d.OnSave()
	assert.True(t, d.HasPendingTimer())
	d.Trigger()
	assert.False(t, d.HasPendingTimer(),
		"Trigger fires + clears pending")
}

func TestBuildOnSaveDaemon_DisabledDuringPendingCancelsTimer(t *testing.T) {
	allowed := atomic.Bool{}
	allowed.Store(true)
	var triggered atomic.Int32
	d := buildpipeline.NewBuildOnSaveDaemon(
		func() bool { return allowed.Load() },
		func() { triggered.Add(1) },
	)
	d.SetDebounce(50 * time.Millisecond)

	d.OnSave()
	// Toggle off and re-OnSave — the second OnSave's allowed-check
	// fails and the daemon clears the pending timer.
	allowed.Store(false)
	d.OnSave()
	time.Sleep(80 * time.Millisecond)
	assert.Equal(t, int32(0), triggered.Load(),
		"disabling mid-pending suppresses the queued build")
}
