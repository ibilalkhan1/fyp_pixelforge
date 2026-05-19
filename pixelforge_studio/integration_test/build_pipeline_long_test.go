// build_pipeline_long_test.go runs end-to-end builds against the
// real Go toolchain. Gated behind //go:build long so the default
// CI matrix (which doesn't ship Go) stays headless; the long-tag
// CI job runs `go test -tags=long ./pixelforge_studio/...` to
// exercise these paths.
//
// The execution-note in plan-008 U3 prescribes test-first: this
// file asserts "real executable produced" / "real .html bundle
// produced" before the corresponding builders land.

//go:build long

package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/buildpipeline"
)

// engineRepoRoot walks up from the test source file's directory
// until it finds the go.mod whose module path is the engine's, so
// the long builders can pass an explicit EnginePath into codegen.
// The test binary itself lives under /tmp and modulepath.Detect
// can't auto-discover the source from there.
func engineRepoRoot(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir := filepath.Dir(here)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// Sanity-check: this go.mod is the engine's, not some
			// nested gen module.
			contents, _ := os.ReadFile(filepath.Join(dir, "go.mod"))
			if strings.Contains(string(contents), "module github.com/ibilalkhan1/fyp_pixelforge") {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate engine go.mod walking up from %s", here)
		}
		dir = parent
	}
}

func newBuildRequest(t *testing.T, outDir string) buildpipeline.BuildRequest {
	t.Helper()
	return buildpipeline.BuildRequest{
		Project:        newMinimalProject(t),
		OutputDir:      outDir,
		EnginePath:     engineRepoRoot(t),
		EngineStrategy: buildpipeline.ModuleStrategyDevReplace,
	}
}

// drainStatusesUntilTerminal collects every BuildStatus until the
// per-target channel closes or the supplied timeout elapses. Tests
// inspect the slice for terminal phase + output path.
func drainStatusesUntilTerminal(t *testing.T, ch <-chan buildpipeline.BuildStatus, timeout time.Duration) []buildpipeline.BuildStatus {
	t.Helper()
	var out []buildpipeline.BuildStatus
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case s, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, s)
		case <-timer.C:
			t.Fatalf("build did not finish within %s; statuses so far: %+v", timeout, out)
			return out
		}
	}
}

func newMinimalProject(t *testing.T) *pixelforge_project.Project {
	t.Helper()
	p := pixelforge_project.NewProject("longbuild")
	return p
}

// TestLong_HostBuilder_ProducesRealExecutable asserts the host
// builder produces a non-placeholder binary at the reported
// OutputPath. We don't execute it (no --smoke flag in the
// minimal capsule yet); we assert it's not the scaffold
// placeholder marker.
func TestLong_HostBuilder_ProducesRealExecutable(t *testing.T) {
	if _, err := buildpipeline.ResolveGoBinary(); err != nil {
		t.Skipf("long-tag test requires the Go toolchain on PATH: %v", err)
	}

	outDir := t.TempDir()
	req := newBuildRequest(t, outDir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	statusCh := buildpipeline.Build(ctx, req, []buildpipeline.Target{buildpipeline.HostTarget()})
	statuses := drainStatusesUntilTerminal(t, statusCh, 5*time.Minute)

	require.NotEmpty(t, statuses)
	last := statuses[len(statuses)-1]
	require.Equal(t, buildpipeline.PhaseDone, last.Phase,
		"host long build must reach PhaseDone; got %+v", statuses)
	require.NotEmpty(t, last.OutputPath, "host long build must report an OutputPath")

	info, err := os.Stat(last.OutputPath)
	require.NoError(t, err)
	// Real binary should be ≥ several KB; the scaffold placeholder
	// is ~30 bytes. 1 KB threshold separates the two cleanly.
	if buildpipeline.HostTarget() != buildpipeline.TargetMacOS {
		assert.Greater(t, info.Size(), int64(1024),
			"output file at %s is suspiciously small (%d bytes) — likely the scaffold placeholder",
			last.OutputPath, info.Size())
	} else {
		// macOS .app is a directory.
		assert.True(t, info.IsDir(), "macOS output must be a .app bundle directory")
	}
}

// TestLong_WASMBuilder_ProducesRealHTML asserts the WASM builder
// produces a self-contained .html with the embedded base64 wasm.
func TestLong_WASMBuilder_ProducesRealHTML(t *testing.T) {
	if _, err := buildpipeline.ResolveGoBinary(); err != nil {
		t.Skipf("long-tag test requires the Go toolchain on PATH: %v", err)
	}

	outDir := t.TempDir()
	req := newBuildRequest(t, outDir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	statusCh := buildpipeline.Build(ctx, req, []buildpipeline.Target{buildpipeline.TargetWASM})
	statuses := drainStatusesUntilTerminal(t, statusCh, 5*time.Minute)

	require.NotEmpty(t, statuses)
	last := statuses[len(statuses)-1]
	require.Equal(t, buildpipeline.PhaseDone, last.Phase,
		"wasm long build must reach PhaseDone; got %+v", statuses)
	require.True(t, strings.HasSuffix(last.OutputPath, ".html"),
		"wasm output must be an HTML bundle; got %s", last.OutputPath)

	content, err := os.ReadFile(last.OutputPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "wasm",
		"WASM bundle must reference the embedded wasm runtime")
	// The base64 envelope inflates the wasm ~4/3 — a real bundle is
	// at least several hundred KB. Scaffold placeholder would be
	// well under 1 KB.
	assert.Greater(t, int64(len(content)), int64(50*1024),
		"WASM HTML at %s is suspiciously small (%d bytes) — likely a placeholder",
		last.OutputPath, len(content))
}

// TestLong_CrossCompilePreflightStillRejects: even under long the
// orchestrator's preflight rejects foreign-OS native targets so
// the long builders never see them. Guards against accidental
// preflight bypass during U3 wiring.
func TestLong_CrossCompilePreflightStillRejects(t *testing.T) {
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
	outDir := t.TempDir()
	statusCh := buildpipeline.Build(context.Background(), newBuildRequest(t, outDir), []buildpipeline.Target{foreign})
	statuses := drainStatusesUntilTerminal(t, statusCh, 10*time.Second)
	require.Len(t, statuses, 1)
	assert.Equal(t, buildpipeline.PhaseFailed, statuses[0].Phase)
	require.Error(t, statuses[0].Err)
}

// TestLong_CancellationKillsBuild covers the cancellation path —
// cancelling the context mid-build must surface PhaseFailed with
// ErrBuildCancelled rather than leaving a stranded subprocess.
func TestLong_CancellationKillsBuild(t *testing.T) {
	if _, err := buildpipeline.ResolveGoBinary(); err != nil {
		t.Skipf("long-tag test requires the Go toolchain on PATH: %v", err)
	}

	outDir := t.TempDir()
	req := newBuildRequest(t, outDir)
	ctx, cancel := context.WithCancel(context.Background())
	statusCh := buildpipeline.Build(ctx, req, []buildpipeline.Target{buildpipeline.TargetWASM})
	// Cancel almost immediately — the build has barely started.
	time.AfterFunc(100*time.Millisecond, cancel)
	statuses := drainStatusesUntilTerminal(t, statusCh, 5*time.Minute)

	require.NotEmpty(t, statuses)
	last := statuses[len(statuses)-1]
	if last.Phase == buildpipeline.PhaseDone {
		// Build raced to completion before cancel fired — acceptable
		// on a very fast machine.
		t.Skipf("build completed before cancel could fire (race condition on fast machines): %+v", last)
	}
	assert.Equal(t, buildpipeline.PhaseFailed, last.Phase,
		"cancelled build must reach PhaseFailed; got %+v", statuses)
}

// TestLong_HostOutputDirLayout: host build's OutputPath sits under
// <outDir>/host/<gameName><ext>.
func TestLong_HostOutputDirLayout(t *testing.T) {
	if _, err := buildpipeline.ResolveGoBinary(); err != nil {
		t.Skipf("long-tag test requires the Go toolchain on PATH: %v", err)
	}
	outDir := t.TempDir()
	req := newBuildRequest(t, outDir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	statusCh := buildpipeline.Build(ctx, req, []buildpipeline.Target{buildpipeline.HostTarget()})
	statuses := drainStatusesUntilTerminal(t, statusCh, 5*time.Minute)
	last := statuses[len(statuses)-1]
	require.Equal(t, buildpipeline.PhaseDone, last.Phase, "got: %+v", statuses)
	assert.True(t, strings.HasPrefix(last.OutputPath, filepath.Join(outDir, "host")),
		"OutputPath %s must sit under <outDir>/host/", last.OutputPath)
}
