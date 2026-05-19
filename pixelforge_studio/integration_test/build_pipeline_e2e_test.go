// build_pipeline_e2e_test.go exercises idea #7 v1 end-to-end at
// the orchestrator + WASM bundler + icon-pipeline layers. Real
// `go build` invocations remain behind a future //go:build long
// gate so the default test suite stays fast + offline.
package integration_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/buildpipeline"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/codegen"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/modulepath"
)

func tempCapsuleProject(t *testing.T) (*pixelforge_project.Project, string) {
	t.Helper()
	dir := t.TempDir()
	p := pixelforge_project.NewProject("rpg_demo")
	p.Scenes = []pixelforge_project.Scene{{ID: "title", Name: "Title"}, {ID: "level_1"}}
	p.Items = []pixelforge_project.ItemDefinition{{ID: "potion", Name: "Potion"}}
	return p, filepath.Join(dir, "rpg_demo.pforge")
}

func TestE2E_BuildPipeline_AE1_OrchestratorParallelEvents(t *testing.T) {
	p, path := tempCapsuleProject(t)
	dir := t.TempDir()
	ch := buildpipeline.Build(context.Background(), buildpipeline.BuildRequest{
		Project: p, ProjectPath: path, OutputDir: dir,
	}, []buildpipeline.Target{
		buildpipeline.TargetLinux,
		buildpipeline.TargetWASM,
		buildpipeline.TargetSource,
	})
	queued := map[buildpipeline.Target]bool{}
	done := map[buildpipeline.Target]bool{}
	for s := range ch {
		switch s.Phase {
		case buildpipeline.PhaseQueued:
			queued[s.Target] = true
		case buildpipeline.PhaseDone, buildpipeline.PhaseFailed:
			done[s.Target] = true
		}
	}
	assert.Len(t, queued, 3)
	assert.Len(t, done, 3)
}

func TestE2E_BuildPipeline_AE2_CapsuleEmbedsProjectAndAssets(t *testing.T) {
	p, path := tempCapsuleProject(t)
	out := filepath.Join(t.TempDir(), "out")
	require.NoError(t, os.MkdirAll(out, 0o755))
	// Generate with explicit published-version strategy to skip
	// the vendor/dev-replace path that needs an engine source dir.
	_, err := codegen.Generate(p, out, codegen.Options{
		Force:                true,
		ProjectSourcePath:    path,
		Strategy:             modulepath.StrategyPublishedVersion,
		EngineVersion:        "v0.0.0",
	})
	require.NoError(t, err)

	mainBytes, err := os.ReadFile(filepath.Join(out, "main.go"))
	require.NoError(t, err)
	assert.Contains(t, string(mainBytes), "CapsuleRun(CapsuleDefaults())")

	capsuleBytes, err := os.ReadFile(filepath.Join(out, "capsule.go"))
	require.NoError(t, err)
	s := string(capsuleBytes)
	assert.Contains(t, s, "//go:embed project.pforge")
	assert.Contains(t, s, "//go:embed all:assets")
	assert.Contains(t, s, `const SceneIDTitle = "title"`)
	assert.Contains(t, s, `const ItemIDPotion = "potion"`)
}

func TestE2E_BuildPipeline_AE3_WASMBundlerProducesSelfContainedHTML(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "g.wasm")
	require.NoError(t, os.WriteFile(wasmPath, []byte{0x00, 0x61, 0x73, 0x6d, 1, 0, 0, 0}, 0o644))
	execPath := filepath.Join(dir, "wasm_exec.js")
	require.NoError(t, os.WriteFile(execPath, []byte("function Go() {}"), 0o644))
	outPath := filepath.Join(dir, "g.html")

	require.NoError(t, buildpipeline.BundleWASM(wasmPath, execPath, "MyGame", "AAAA", outPath, nil))
	html, _ := os.ReadFile(outPath)
	s := string(html)
	assert.Contains(t, s, "<title>MyGame</title>")
	assert.Contains(t, s, "Click to start")
	assert.NotContains(t, s, `src="http`, "no external script URLs (offline-only)")
}

func TestE2E_BuildPipeline_AE4_IconAutoPickResolvesByReferenceCount(t *testing.T) {
	p := pixelforge_project.NewProject("hero_game")
	p.Sprites = []pixelforge_project.SpriteAsset{
		{Name: "hero", FrameW: 16, FrameH: 16},
		{Name: "rare"}, {Name: "common"},
	}
	p.Items = []pixelforge_project.ItemDefinition{
		{ID: "potion", Icon: "common"},
		{ID: "elixir", Icon: "common"},
	}
	got := buildpipeline.ResolveIconSprite(p)
	require.NotNil(t, got)
	assert.Equal(t, "common", got.Name, "reference-count winner (common, 2 refs)")
}

func TestE2E_BuildPipeline_AE5_IconResultIncludesFavicon(t *testing.T) {
	p := pixelforge_project.NewProject("test")
	p.Sprites = []pixelforge_project.SpriteAsset{{Name: "hero", FrameW: 16, FrameH: 16}}
	res, err := buildpipeline.GenerateIconResult(p)
	require.NoError(t, err)
	assert.NotEmpty(t, res.FaviconBase64)
	assert.NotNil(t, res.Sprite)
}

func TestE2E_BuildPipeline_AE6_BuildOnSaveDebouncesCoalescedSaves(t *testing.T) {
	fired := make(chan struct{}, 10)
	d := buildpipeline.NewBuildOnSaveDaemon(
		func() bool { return true },
		func() { fired <- struct{}{} },
	)
	d.SetDebounce(30 * time.Millisecond)
	for i := 0; i < 5; i++ {
		d.OnSave()
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(60 * time.Millisecond)
	close(fired)
	n := 0
	for range fired {
		n++
	}
	assert.Equal(t, 1, n, "5 rapid saves coalesce into a single build trigger")
}

func TestE2E_BuildPipeline_AE7_SourceTargetProducesOutputFile(t *testing.T) {
	if isLongBuildTag() {
		t.Skip("Source target under //go:build long uses real codegen which needs an explicit EnginePath; long-tag equivalent is exercised via host/wasm long tests.")
	}
	p, path := tempCapsuleProject(t)
	dir := t.TempDir()
	ch := buildpipeline.Build(context.Background(), buildpipeline.BuildRequest{
		Project: p, ProjectPath: path, OutputDir: dir,
	}, []buildpipeline.Target{buildpipeline.TargetSource})
	var doneStatus *buildpipeline.BuildStatus
	for s := range ch {
		if s.Phase == buildpipeline.PhaseDone {
			s := s
			doneStatus = &s
		}
	}
	require.NotNil(t, doneStatus)
	assert.Contains(t, doneStatus.OutputPath, "source")
	_, err := os.Stat(doneStatus.OutputPath)
	require.NoError(t, err)
}

func TestE2E_BuildPipeline_AE8_ToolchainResolvesGoBinary(t *testing.T) {
	got, err := buildpipeline.ResolveGoBinary()
	require.NoError(t, err)
	assert.NotEmpty(t, got, "PATH-installed go discovered")
}

func TestE2E_BuildPipeline_F1_ProjectVersionAutoStamped(t *testing.T) {
	p := pixelforge_project.NewProject("v_demo")
	// Save then reload to trigger applyDefaults.
	data, err := pixelforge_project.Encode(p)
	require.NoError(t, err)
	loaded, err := pixelforge_project.LoadReader(bytes.NewReader(data))
	require.NoError(t, err)
	assert.NotEmpty(t, loaded.Version, "Version auto-stamped on load")
	assert.Equal(t, time.Now().Format("2006-01-02"), loaded.Version)
}

func TestE2E_BuildPipeline_F2_IconStubsProduceRealBytesAfterU5(t *testing.T) {
	// Plan-008 U5 replaces the per-platform stubs with real
	// rasterised .ico / .icns output from docs/logo.svg. The
	// legacy IsIconUnsupported sentinel survives one release for
	// source compat but is never returned by the active code path.
	got, err := buildpipeline.GenerateWindowsIcoStub(nil)
	require.NoError(t, err)
	assert.NotEmpty(t, got)
	assert.False(t, buildpipeline.IsIconUnsupported(err))
}

func TestE2E_BuildPipeline_F3_TargetGOOSMatrix(t *testing.T) {
	cases := []struct {
		target buildpipeline.Target
		os     string
	}{
		{buildpipeline.TargetWindows, "windows"},
		{buildpipeline.TargetMacOS, "darwin"},
		{buildpipeline.TargetLinux, "linux"},
		{buildpipeline.TargetWASM, "js"},
	}
	for _, c := range cases {
		assert.Equal(t, c.os, c.target.GOOS(), c.target.DisplayName())
	}
}

func TestE2E_BuildPipeline_F4_CancellationStopsAllInFlight(t *testing.T) {
	p, path := tempCapsuleProject(t)
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel
	ch := buildpipeline.Build(ctx, buildpipeline.BuildRequest{
		Project: p, ProjectPath: path, OutputDir: dir,
	}, []buildpipeline.Target{buildpipeline.TargetLinux, buildpipeline.TargetWASM})
	terminal := 0
	for s := range ch {
		if s.Phase == buildpipeline.PhaseDone || s.Phase == buildpipeline.PhaseFailed {
			terminal++
		}
	}
	assert.Equal(t, 2, terminal, "each target reaches a terminal phase even on cancel")
}

