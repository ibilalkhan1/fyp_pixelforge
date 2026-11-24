package buildpipeline_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/buildpipeline"
)

func tempProject(t *testing.T) (*pixelforge_project.Project, string) {
	t.Helper()

	previous := map[buildpipeline.Target]buildpipeline.Builder{}
	for _, target := range buildpipeline.AllTargets {
		if b, ok := buildpipeline.LookupBuilder(target); ok {
			previous[target] = b
		}
	}
	buildpipeline.ResetBuildersForTest()
	registerTestBuilders()
	t.Cleanup(func() {
		buildpipeline.ResetBuildersForTest()
		for target, b := range previous {
			buildpipeline.RegisterBuilder(target, b)
		}
	})
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
		buildpipeline.HostTarget(),
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
	}, []buildpipeline.Target{buildpipeline.HostTarget()})

	var doneStatus *buildpipeline.BuildStatus
	for s := range ch {
		if s.Phase == buildpipeline.PhaseDone {
			s := s
			doneStatus = &s
		}
	}
	require.NotNil(t, doneStatus)
	// !long scaffold writes to <target>/<game>; long real builder
	// writes to host/<game> (the universal-player pipeline doesn't
	// stratify by GOOS at the path level — the binary itself
	// carries that information). Accept either layout so the test
	// stays meaningful under both tag sets.
	assert.True(t,
		strings.Contains(doneStatus.OutputPath, buildpipeline.HostTarget().String()) ||
			strings.Contains(doneStatus.OutputPath, "host"),
		"output path should land under linux/ or host/, got %s", doneStatus.OutputPath)
	assert.Contains(t, doneStatus.OutputPath, "test_game")
}

func TestBuild_ParallelTargetsRunConcurrently(t *testing.T) {
	p, path := tempProject(t)
	dir := t.TempDir()
	start := time.Now()
	ch := buildpipeline.Build(context.Background(), buildpipeline.BuildRequest{
		Project: p, ProjectPath: path, OutputDir: dir,
	}, []buildpipeline.Target{
		buildpipeline.HostTarget(),
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
	}, []buildpipeline.Target{buildpipeline.HostTarget()})

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
	}, []buildpipeline.Target{buildpipeline.HostTarget()})

	var doneStatus *buildpipeline.BuildStatus
	for s := range ch {
		if s.Phase == buildpipeline.PhaseDone {
			s := s
			doneStatus = &s
		}
	}
	require.NotNil(t, doneStatus)
	// Default output dir is <project>/exports/<target>; under
	// -tags=long the long builder uses "host" as the subdir
	// regardless of the host GOOS. Accept either layout.
	assert.True(t,
		strings.Contains(doneStatus.OutputPath, filepath.Join(dir, "exports", buildpipeline.HostTarget().String())) ||
			strings.Contains(doneStatus.OutputPath, filepath.Join(dir, "exports", "host")),
		"default output dir should be exports/linux or exports/host, got %s", doneStatus.OutputPath)
}

func TestBuild_UnregisteredTargetFailsClearly(t *testing.T) {
	p, path := tempProject(t)
	buildpipeline.ResetBuildersForTest()
	dir := t.TempDir()
	ch := buildpipeline.Build(context.Background(), buildpipeline.BuildRequest{
		Project: p, ProjectPath: path, OutputDir: dir,
	}, []buildpipeline.Target{buildpipeline.HostTarget()})

	var failedErr error
	for s := range ch {
		if s.Phase == buildpipeline.PhaseFailed {
			failedErr = s.Err
		}
	}
	require.Error(t, failedErr)
	assert.Contains(t, failedErr.Error(), "no builder registered")
}

// registerTestBuilders installs explicit fakes for orchestration-only tests.

func registerTestBuilders() {
	for _, t := range buildpipeline.AllTargets {
		t := t
		buildpipeline.RegisterBuilder(t, &noopBuilder{target: t})
	}
}

type noopBuilder struct{ target buildpipeline.Target }

func (n *noopBuilder) Build(ctx context.Context, req buildpipeline.BuildRequest, emit func(buildpipeline.BuildStatus)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dir := req.OutputDir
	if dir == "" {
		dir = filepath.Join(filepath.Dir(req.ProjectPath), "exports")
	}
	output := filepath.Join(dir, n.target.String(), req.Project.Name+"-orchestrator-test-artifact")
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(output, []byte("test fixture"), 0644); err != nil {
		return err
	}
	emit(buildpipeline.BuildStatus{Phase: buildpipeline.PhaseDone, OutputPath: output})
	return nil
}
