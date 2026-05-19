package buildpipeline_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/buildpipeline"
)

func tempProject(t *testing.T) (*pixelforge_project.Project, string) {
	t.Helper()
	dir := t.TempDir()
	p := pixelforge_project.NewProject("test_game")
	return p, filepath.Join(dir, "test_game.pforge")
}

func TestPhase_String(t *testing.T) {
	cases := []struct {
		p    buildpipeline.Phase
		want string
	}{
		{buildpipeline.PhaseQueued, "queued"},
		{buildpipeline.PhaseGenerating, "generating"},
		{buildpipeline.PhaseCompiling, "compiling"},
		{buildpipeline.PhasePackaging, "packaging"},
		{buildpipeline.PhaseDone, "done"},
		{buildpipeline.PhaseFailed, "failed"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, c.p.String())
	}
}

func TestBuild_EmptyTargetsClosesChannelImmediately(t *testing.T) {
	p, path := tempProject(t)
	ch := buildpipeline.Build(context.Background(), buildpipeline.BuildRequest{
		Project: p, ProjectPath: path,
	}, nil)
	for range ch {
		t.Fatal("expected no events on empty targets")
	}
}

func TestBuild_EveryTargetEmitsQueuedAndTerminal(t *testing.T) {
	p, path := tempProject(t)
	dir := t.TempDir()
	targets := []buildpipeline.Target{
		buildpipeline.TargetLinux,
		buildpipeline.TargetWASM,
	}
	ch := buildpipeline.Build(context.Background(), buildpipeline.BuildRequest{
		Project: p, ProjectPath: path, OutputDir: dir,
	}, targets)

	queued := map[buildpipeline.Target]bool{}
	terminal := map[buildpipeline.Target]bool{}
	for s := range ch {
		switch s.Phase {
		case buildpipeline.PhaseQueued:
			queued[s.Target] = true
		case buildpipeline.PhaseDone, buildpipeline.PhaseFailed:
			terminal[s.Target] = true
		}
	}
	for _, t2 := range targets {
		assert.True(t, queued[t2], "target %s saw Queued", t2)
		assert.True(t, terminal[t2], "target %s reached a terminal phase", t2)
	}
}

func TestBuild_OutputFileWrittenAtExpectedPath(t *testing.T) {
	p, path := tempProject(t)
	dir := t.TempDir()
	ch := buildpipeline.Build(context.Background(), buildpipeline.BuildRequest{
		Project: p, ProjectPath: path, OutputDir: dir,
	}, []buildpipeline.Target{buildpipeline.TargetLinux})

	var doneStatus *buildpipeline.BuildStatus
	for s := range ch {
		if s.Phase == buildpipeline.PhaseDone {
			s := s
			doneStatus = &s
		}
	}
	require.NotNil(t, doneStatus)
	assert.Contains(t, doneStatus.OutputPath, "linux")
	assert.Contains(t, doneStatus.OutputPath, "test_game")
}

func TestBuild_ParallelTargetsRunConcurrently(t *testing.T) {
	p, path := tempProject(t)
	dir := t.TempDir()
	start := time.Now()
	ch := buildpipeline.Build(context.Background(), buildpipeline.BuildRequest{
		Project: p, ProjectPath: path, OutputDir: dir,
	}, []buildpipeline.Target{
		buildpipeline.TargetLinux,
		buildpipeline.TargetWASM,
		buildpipeline.TargetSource,
	})
	for range ch {
	}
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 5*time.Second,
		"three parallel builds complete well under the 5s budget")
}

func TestBuild_CancelledContextStopsBuilds(t *testing.T) {
	p, path := tempProject(t)
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel
	ch := buildpipeline.Build(ctx, buildpipeline.BuildRequest{
		Project: p, ProjectPath: path, OutputDir: dir,
	}, []buildpipeline.Target{buildpipeline.TargetLinux})

	var sawFailed bool
	for s := range ch {
		if s.Phase == buildpipeline.PhaseFailed {
			sawFailed = true
		}
	}
	assert.True(t, sawFailed,
		"cancelled context surfaces a Failed status on each target")
}

func TestBuild_DefaultOutputDirIsProjectRelativeExports(t *testing.T) {
	p, _ := tempProject(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "game.pforge")
	ch := buildpipeline.Build(context.Background(), buildpipeline.BuildRequest{
		Project: p, ProjectPath: path, // no OutputDir
	}, []buildpipeline.Target{buildpipeline.TargetLinux})

	var doneStatus *buildpipeline.BuildStatus
	for s := range ch {
		if s.Phase == buildpipeline.PhaseDone {
			s := s
			doneStatus = &s
		}
	}
	require.NotNil(t, doneStatus)
	assert.Contains(t, doneStatus.OutputPath, filepath.Join(dir, "exports", "linux"))
}

func TestBuild_UnregisteredTargetFailsClearly(t *testing.T) {
	buildpipeline.ResetBuildersForTest()
	t.Cleanup(func() {
		// Re-register the production builders so other tests in
		// the same process see a populated registry.
		registerProductionBuildersForTest()
	})

	p, path := tempProject(t)
	dir := t.TempDir()
	ch := buildpipeline.Build(context.Background(), buildpipeline.BuildRequest{
		Project: p, ProjectPath: path, OutputDir: dir,
	}, []buildpipeline.Target{buildpipeline.TargetLinux})

	var failedErr error
	for s := range ch {
		if s.Phase == buildpipeline.PhaseFailed {
			failedErr = s.Err
		}
	}
	require.Error(t, failedErr)
	assert.Contains(t, failedErr.Error(), "no builder registered")
}

// registerProductionBuildersForTest rebuilds the registry that
// ResetBuildersForTest cleared, mirroring the production init().
func registerProductionBuildersForTest() {
	for _, t := range buildpipeline.AllTargets {
		// Re-use the scaffold builder via the placeholder closure
		// in the production package. The test only needs the
		// registry to be non-empty for subsequent tests; a no-op
		// builder is sufficient.
		t := t
		buildpipeline.RegisterBuilder(t, &noopBuilder{target: t})
	}
}

type noopBuilder struct{ target buildpipeline.Target }

func (n *noopBuilder) Build(ctx context.Context, req buildpipeline.BuildRequest, emit func(buildpipeline.BuildStatus)) error {
	return nil
}
