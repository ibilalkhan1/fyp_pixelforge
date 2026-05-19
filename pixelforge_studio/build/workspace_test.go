package build_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/build"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/buildpipeline"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
)

func newBuildWorkspace(t *testing.T) (*editor.Editor, *build.Workspace) {
	t.Helper()
	e := editor.New()
	w := build.RegisterWith(e)
	return e, w
}

func TestWorkspace_RegistersWithEditor(t *testing.T) {
	e, w := newBuildWorkspace(t)
	assert.Equal(t, "build", w.Name())
	assert.Equal(t, "Build", w.DisplayName())
	found := false
	for _, ws := range e.Workspaces() {
		if ws.Name() == "build" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestWorkspace_RecordStatusUpdatesMap(t *testing.T) {
	_, w := newBuildWorkspace(t)
	w.RecordStatus(buildpipeline.BuildStatus{
		Target: buildpipeline.TargetLinux,
		Phase:  buildpipeline.PhaseDone,
	})
	got := w.Status(buildpipeline.TargetLinux)
	assert.Equal(t, buildpipeline.PhaseDone, got.Phase)
}

// TestWorkspace_BuildHostDispatchesHostTarget asserts the Host
// button dispatches a build for exactly the host's native target
// — the click → dispatch path the two-button UI relies on.
func TestWorkspace_BuildHostDispatchesHostTarget(t *testing.T) {
	e, w := newBuildWorkspace(t)
	dir := t.TempDir()
	e.SetCurrentProjectPath(filepath.Join(dir, "test.pforge"))

	done := w.BuildHost()
	<-done
	got := w.Status(buildpipeline.HostTarget())
	assert.True(t, got.Phase == buildpipeline.PhaseDone || got.Phase == buildpipeline.PhaseFailed,
		"Host build reached a terminal phase")
}

// TestWorkspace_BuildWASMDispatchesWASMTarget asserts the WASM
// button dispatches a WASM build.
func TestWorkspace_BuildWASMDispatchesWASMTarget(t *testing.T) {
	e, w := newBuildWorkspace(t)
	dir := t.TempDir()
	e.SetCurrentProjectPath(filepath.Join(dir, "test.pforge"))

	done := w.BuildWASM()
	<-done
	got := w.Status(buildpipeline.TargetWASM)
	assert.True(t, got.Phase == buildpipeline.PhaseDone || got.Phase == buildpipeline.PhaseFailed,
		"WASM build reached a terminal phase")
}

func TestWorkspace_CancelBuildSafeOnIdle(t *testing.T) {
	_, w := newBuildWorkspace(t)
	assert.NotPanics(t, func() { w.CancelBuild() })
}

func TestWorkspace_AllStatusesEnumeratesRecorded(t *testing.T) {
	_, w := newBuildWorkspace(t)
	w.RecordStatus(buildpipeline.BuildStatus{Target: buildpipeline.TargetLinux, Phase: buildpipeline.PhaseDone})
	w.RecordStatus(buildpipeline.BuildStatus{Target: buildpipeline.TargetWASM, Phase: buildpipeline.PhaseFailed})
	got := w.AllStatuses()
	assert.Len(t, got, 2)
}

// TestCanBuildOnHost_WASMAlwaysTrue covers the U4 invariant: WASM
// always builds regardless of host because it's pure-Go js/wasm.
func TestCanBuildOnHost_WASMAlwaysTrue(t *testing.T) {
	assert.True(t, buildpipeline.CanBuildOnHost(buildpipeline.TargetWASM))
}

// TestCanBuildOnHost_SourceAlwaysTrue: Source target just copies
// generated files, so it's never blocked by toolchain availability.
func TestCanBuildOnHost_SourceAlwaysTrue(t *testing.T) {
	assert.True(t, buildpipeline.CanBuildOnHost(buildpipeline.TargetSource))
}

// TestCanBuildOnHost_HostTrueForOwnPlatform: the host's actual
// native target is always callable.
func TestCanBuildOnHost_HostTrueForOwnPlatform(t *testing.T) {
	assert.True(t, buildpipeline.CanBuildOnHost(buildpipeline.HostTarget()))
}

// TestCanBuildOnHost_RejectsCrossOS: at least one non-host native
// target must be rejected. Picking the two that aren't the host so
// the assertion holds on every platform.
func TestCanBuildOnHost_RejectsCrossOS(t *testing.T) {
	host := buildpipeline.HostTarget()
	var foreign buildpipeline.Target
	for _, candidate := range []buildpipeline.Target{
		buildpipeline.TargetWindows, buildpipeline.TargetMacOS, buildpipeline.TargetLinux,
	} {
		if candidate != host {
			foreign = candidate
			break
		}
	}
	assert.False(t, buildpipeline.CanBuildOnHost(foreign),
		"non-host native target %s should be rejected on host %s", foreign, host)
}

// TestBuild_CrossCompilePreflightEmitsFailed covers the
// programmatic API path: callers asking for a non-host native
// target receive PhaseFailed{Err: CrossCompileNotSupportedError}
// immediately, without a goroutine spinning up.
func TestBuild_CrossCompilePreflightEmitsFailed(t *testing.T) {
	host := buildpipeline.HostTarget()
	var foreign buildpipeline.Target
	for _, candidate := range []buildpipeline.Target{
		buildpipeline.TargetWindows, buildpipeline.TargetMacOS, buildpipeline.TargetLinux,
	} {
		if candidate != host {
			foreign = candidate
			break
		}
	}
	statusCh := buildpipeline.Build(nil, buildpipeline.BuildRequest{}, []buildpipeline.Target{foreign})
	var statuses []buildpipeline.BuildStatus
	for s := range statusCh {
		statuses = append(statuses, s)
	}
	require.Len(t, statuses, 1, "rejected target must emit exactly one status (PhaseFailed)")
	assert.Equal(t, buildpipeline.PhaseFailed, statuses[0].Phase)
	require.Error(t, statuses[0].Err)
	assert.True(t, errors.Is(statuses[0].Err, buildpipeline.ErrCrossCompileNotSupported),
		"err must satisfy errors.Is(err, ErrCrossCompileNotSupported); got %v", statuses[0].Err)
}

// TestCrossCompileError_MessageNamesBothOSes guards the
// diagnosability invariant — the error string must contain both
// the requested target's name and the host's actual OS.
func TestCrossCompileError_MessageNamesBothOSes(t *testing.T) {
	host := buildpipeline.HostTarget()
	var foreign buildpipeline.Target
	for _, candidate := range []buildpipeline.Target{
		buildpipeline.TargetWindows, buildpipeline.TargetMacOS, buildpipeline.TargetLinux,
	} {
		if candidate != host {
			foreign = candidate
			break
		}
	}
	statusCh := buildpipeline.Build(nil, buildpipeline.BuildRequest{}, []buildpipeline.Target{foreign})
	statuses := drain(statusCh)
	require.Len(t, statuses, 1)
	require.Error(t, statuses[0].Err)
	msg := statuses[0].Err.Error()
	assert.Contains(t, msg, foreign.String(), "error must name the requested target")
	// host's OS is one of windows/darwin/linux; assert the message
	// names it so designers see "Windows from linux" vs "macOS from
	// linux" etc.
	hostStringer := buildpipeline.HostTarget().String()
	if hostStringer == "macos" {
		hostStringer = "darwin"
	}
	assert.Contains(t, msg, hostStringer, "error must name the host's actual OS")
}

func drain(ch <-chan buildpipeline.BuildStatus) []buildpipeline.BuildStatus {
	var out []buildpipeline.BuildStatus
	for s := range ch {
		out = append(out, s)
	}
	return out
}
