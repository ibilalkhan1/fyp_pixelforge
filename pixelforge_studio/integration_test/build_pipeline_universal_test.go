// build_pipeline_universal_test.go is plan-009 U12's end-to-end
// smoke test for the universal-player + cart-append shipping path.
// It exercises the full studio-side shipping pipeline against the
// Asteroids proof fixture authored in U11:
//
//  1. Load the .pforge cart (the same one ce-asteroids_proof_test.go
//     drives the replay against).
//  2. Invoke buildpipeline.Build with TargetHost. The long builders
//     resolve a pre-built player binary (cache hit / embed extract /
//     developer go-build fallback), call codegen.EncodeCart for the
//     project, and call pixelforge_cart.Append to splice the cart
//     onto a copy of the player.
//  3. Locate the resulting host binary at <outDir>/host/<gameName>.
//  4. Spawn it as a subprocess with --smoke-tick=60. The smoke seam
//     in cmd/pixelforge-player/main.go skips Ebitengine.Run and
//     advances 60 capsule-runtime ticks instead, exiting 0 on
//     success.
//  5. Independently extract the appended cart payload via the
//     pixelforge_cart envelope helpers and assert it round-trips
//     into a Project whose structural fields match the source — the
//     "binary carries its own game" invariant.
//
// The WASM half of the smoke is compile-only: we build the same
// fixture with TargetWASM, verify the .html bundle exists + is
// non-empty + carries both base64 blobs (wasm + cart). Loading the
// HTML in a real browser via wasmbrowsertest is explicitly out of
// scope per plan-009's Scope Boundaries section.
//
// Cross-compile rejection is reused from build_pipeline_long_test.go's
// TestLong_CrossCompilePreflightStillRejects; we don't duplicate it
// here but call out the linkage in the failing-test guidance so
// future maintainers know where to look.
//
// Deferred (Phase 1 closure note): AE1's "sprite jumps in response
// to Space" assertion cannot land until U13 adds the platformer
// motion sinks (jump / apply_gravity / solid_collide). For U12 the
// AE1 surface is "binary loads + boots + ticks cleanly" — the strong
// pre-condition for the user-visible jump. The full sprite-input
// integration appears in U13's TestMarioProof.

//go:build long

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cart"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/buildpipeline"
)

// smokeTickTimeout is the upper bound on a single subprocess
// invocation of the built host binary with --smoke-tick=60. The
// smoke path skips Ebitengine.Run entirely (no window allocation)
// so the wall-clock cost is dominated by capsule-runtime Boot +
// sprite-loader work plus 60 trivial AdvanceTick calls; 10 seconds
// is generous on every supported host.
const smokeTickTimeout = 30 * time.Second

// loadAsteroidsProofProject opens the U11 .pforge fixture and
// returns the loaded *Project plus its on-disk path. The build
// pipeline only consumes the project struct; the path is returned
// for callers who want to assert the fixture exists alongside the
// loaded payload.
//
// Resolution: walk from the integration_test package directory to
// fixtures/asteroids_proof.pforge. runtime.Caller(0) keeps the
// resolution robust to the test binary's cwd.
func loadAsteroidsProofProject(t *testing.T) (*pixelforge_project.Project, string) {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller must succeed in tests")
	path := filepath.Join(filepath.Dir(here), "fixtures", "asteroids_proof.pforge")

	f, err := os.Open(path)
	require.NoError(t, err, "open asteroids_proof.pforge at %s", path)
	defer f.Close()

	project, err := pixelforge_project.LoadReader(f)
	require.NoError(t, err, "decode asteroids_proof.pforge")
	require.NotNil(t, project)
	require.Equal(t, "asteroids_proof", project.Name,
		"fixture sanity-check: expected U11's asteroids_proof name")
	return project, path
}

// extractAppendedCart is the test-side equivalent of pixelforge_cart.ReadSelf
// against an arbitrary on-disk binary. Mirrors readCartFromFile in
// builders_long_test.go but lives here too because that helper is
// inside the buildpipeline package (different import boundary).
//
// The round-trip pattern is intentional: we don't compare bytes
// against the source EncodeCart output because Encode bumps
// ModifiedAt on every call (timestamps drift between Encode
// invocations). Instead we LoadReader the extracted bytes back
// into a *Project and compare structural fields the test author
// authored — Name, Scenes count, PhysicsPreset.
func extractAppendedCart(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "read built binary at %s", path)
	require.GreaterOrEqual(t, len(data), pixelforge_cart.FooterSize,
		"built binary too small to contain a footer (got %d bytes)", len(data))

	footer := data[len(data)-pixelforge_cart.FooterSize:]
	length, _, err := pixelforge_cart.DecodeFooter(footer)
	require.NoError(t, err, "decode cart footer from %s", path)
	require.LessOrEqual(t,
		uint64(pixelforge_cart.FooterSize)+length, uint64(len(data)),
		"declared cart length %d + footer overflows file size %d", length, len(data))

	start := uint64(len(data)) - uint64(pixelforge_cart.FooterSize) - length
	end := uint64(len(data)) - uint64(pixelforge_cart.FooterSize)
	return data[start:end]
}

// runSmokeSubprocess shells out to the built binary with
// --smoke-tick=N and asserts the exit code is 0. Captures stdout +
// stderr so a failing smoke run surfaces the player's diagnostic
// in the test output rather than just a non-zero exit code. Uses
// context.WithTimeout so a hung subprocess fails the test instead
// of stalling the long-tag CI job.
//
// Returns the combined output so callers can spot-check the smoke
// log line ("pixelforge-player: smoke-tick completed N ticks").
func runSmokeSubprocess(t *testing.T, binaryPath string, ticks uint64, timeout time.Duration) string {
	t.Helper()
	require.NotEmpty(t, binaryPath)
	info, err := os.Stat(binaryPath)
	require.NoError(t, err, "binary must exist at %s", binaryPath)
	require.False(t, info.IsDir(), "subprocess target must be a file (got directory)")
	require.NotZero(t, info.Size(), "binary must be non-empty")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// flag.Parse only accepts --flag=value or --flag value at the
	// position the parser walks; cmd/pixelforge-player passes them
	// straight through flag.CommandLine.Parse on os.Args[1:]. We
	// use the formatter directly so the ticks parameter is not
	// baked into a hard-coded literal — keeps the helper callable
	// for both the Asteroids (60 ticks) and empty-project (10
	// ticks) smokes without diverging.
	cmd := exec.CommandContext(ctx, binaryPath, fmt.Sprintf("--smoke-tick=%d", ticks))

	// CombinedOutput is the right primitive for "run + capture
	// stdout AND stderr in one go" — using separate bytes.Buffer
	// objects sometimes loses the pipe attachment under
	// `go test`'s pipe redirection (observed in CI when the test
	// runner's own goroutines hold open the parent's stderr; the
	// child's writes go nowhere). CombinedOutput uses a single
	// pipe and copies eagerly, which dodges that interaction.
	combinedBytes, err := cmd.CombinedOutput()
	combined := string(combinedBytes)

	// A context-deadline exceeded surfaces as a generic ExitError;
	// disambiguate so the failure message tells the maintainer
	// whether the binary hung or exited with a non-zero code.
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("smoke subprocess timed out after %s; output:\n%s", timeout, combined)
	}
	require.NoError(t, err,
		"smoke subprocess at %s with --smoke-tick=%d must exit 0; output:\n%s",
		binaryPath, ticks, combined)
	return combined
}

// hostArtifactPath returns the absolute path the host long builder
// writes for a project, matching builders_long.go's outPath
// composition. On macOS the executable lives inside the .app
// bundle; everywhere else it's the bare binary at <outDir>/host/<game>[.exe].
func hostArtifactPath(t *testing.T, outDir string, project *pixelforge_project.Project) string {
	t.Helper()
	hostDir := filepath.Join(outDir, "host")
	gameName := project.Name
	if gameName == "" {
		gameName = "game"
	}
	switch buildpipeline.HostTarget() {
	case buildpipeline.TargetWindows:
		return filepath.Join(hostDir, gameName+".exe")
	case buildpipeline.TargetMacOS:
		return filepath.Join(hostDir, gameName+".app", "Contents", "MacOS", gameName)
	}
	return filepath.Join(hostDir, gameName)
}

// TestBuildPipelineUniversal_HostSmoke is the headline U12 smoke
// test: Build → Host on the Asteroids proof must produce a binary
// that boots cleanly + completes 60 capsule-runtime ticks + exits 0.
//
// Covers AE1 ("Asteroids: binary loads + ticks") + AE2 ("Build →
// Host produces a runnable artifact") in spirit. The full AE1
// surface — sprite responds to Space → jump (or, for Asteroids,
// rotation + thrust input) — lands in U13 once the input-intent
// bridge between InputFrame and verb publishes ships. For U12 the
// smoke value is the build pipeline's plumbing: cart-append round-
// trips, the binary is self-contained, the player's smoke-tick
// seam works in a real subprocess.
func TestBuildPipelineUniversal_HostSmoke(t *testing.T) {
	if _, err := buildpipeline.ResolveGoBinary(); err != nil {
		t.Skipf("long-tag smoke requires the Go toolchain on PATH: %v", err)
	}
	// Linux/amd64 is the canonical CI target. macOS works locally
	// but lands the .app bundle in a different relative path; the
	// hostArtifactPath helper handles that. Windows triggers the
	// .syso relink path which is slow + flaky on CI — defer that
	// scenario to a manual smoke (TestBuildPipelineUniversal_HostSmoke
	// is opt-in via -tags=long; the Windows leg can opt in further
	// once the relink stabilises).
	if runtime.GOOS == "windows" {
		t.Skip("Windows host triggers the .syso relink path; smoke covered by manual runs")
	}

	project, _ := loadAsteroidsProofProject(t)
	outDir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	req := buildpipeline.BuildRequest{
		Project:        project,
		OutputDir:      outDir,
		EnginePath:     engineRepoRoot(t),
		EngineStrategy: buildpipeline.ModuleStrategyDevReplace,
	}
	statusCh := buildpipeline.Build(ctx, req, []buildpipeline.Target{buildpipeline.HostTarget()})
	statuses := drainStatusesUntilTerminal(t, statusCh, 5*time.Minute)

	require.NotEmpty(t, statuses, "expected at least one build status")
	terminal := statuses[len(statuses)-1]
	require.Equal(t, buildpipeline.PhaseDone, terminal.Phase,
		"host build must reach PhaseDone; got %+v", statuses)
	require.NotEmpty(t, terminal.OutputPath, "host build must report OutputPath")

	// Round-trip: re-extract the appended cart and confirm it
	// LoadReader's back into a project whose Name + PhysicsPreset
	// + Scenes count match the source. Byte-equality is the wrong
	// invariant (Encode bumps ModifiedAt); structural equality is
	// what the runtime actually consumes.
	binaryPath := hostArtifactPath(t, outDir, project)
	cartBytes := extractAppendedCart(t, binaryPath)
	require.NotEmpty(t, cartBytes, "appended cart must be non-empty")
	roundTripped, err := pixelforge_project.LoadReader(bytes.NewReader(cartBytes))
	require.NoError(t, err, "ReadSelf-equivalent cart must LoadReader cleanly")
	assert.Equal(t, project.Name, roundTripped.Name)
	assert.Equal(t, project.PhysicsPreset, roundTripped.PhysicsPreset)
	assert.Equal(t, len(project.Scenes), len(roundTripped.Scenes),
		"scenes count must round-trip through cart-append")

	// Smoke-spawn the built binary with --smoke-tick=60 and
	// assert exit 0. This is the load-bearing assertion for U12:
	// it proves the freshly-built host binary parses its own cart,
	// boots the capsule runtime, advances 60 ticks without
	// panicking, and exits cleanly.
	output := runSmokeSubprocess(t, binaryPath, 60, smokeTickTimeout)
	assert.Contains(t, output, "smoke-tick completed",
		"smoke binary must log the completion marker; got:\n%s", output)
}

// TestBuildPipelineUniversal_WASMSmoke is the WASM compile-only
// counterpart: build the Asteroids proof targeting WASM and verify
// the resulting .html bundle is non-empty + contains both the wasm
// and cart base64 blobs. Loading the HTML in a real browser via
// wasmbrowsertest is explicitly deferred to v3 per plan-009's
// Scope Boundaries.
//
// The two base64-presence regex matches mirror the long-tag
// builders_long_test.go's pattern (escaped `/` allowance for
// html/template output), so a future bundler change that drops
// either slot surfaces here as a clear failure.
func TestBuildPipelineUniversal_WASMSmoke(t *testing.T) {
	if _, err := buildpipeline.ResolveGoBinary(); err != nil {
		t.Skipf("long-tag smoke requires the Go toolchain on PATH: %v", err)
	}

	project, _ := loadAsteroidsProofProject(t)
	outDir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	req := buildpipeline.BuildRequest{
		Project:        project,
		OutputDir:      outDir,
		EnginePath:     engineRepoRoot(t),
		EngineStrategy: buildpipeline.ModuleStrategyDevReplace,
	}
	statusCh := buildpipeline.Build(ctx, req, []buildpipeline.Target{buildpipeline.TargetWASM})
	statuses := drainStatusesUntilTerminal(t, statusCh, 5*time.Minute)

	require.NotEmpty(t, statuses)
	terminal := statuses[len(statuses)-1]
	require.Equal(t, buildpipeline.PhaseDone, terminal.Phase,
		"WASM build must reach PhaseDone; got %+v", statuses)
	require.True(t, strings.HasSuffix(terminal.OutputPath, ".html"),
		"WASM output must be an HTML bundle; got %s", terminal.OutputPath)

	htmlBytes, err := os.ReadFile(terminal.OutputPath)
	require.NoError(t, err)
	require.NotEmpty(t, htmlBytes, "WASM HTML bundle must be non-empty")
	html := string(htmlBytes)

	// Both base64 slots must carry a non-trivial payload. The
	// alphabet class allows `\/` because html/template escapes
	// forward slashes inside JS string literals.
	wasmAssign := regexp.MustCompile(`const wasmBase64 = "([A-Za-z0-9+/=\\]{8,})"`)
	cartAssign := regexp.MustCompile(`const cartBase64 = "([A-Za-z0-9+/=\\]{8,})"`)
	assert.Regexp(t, wasmAssign, html, "wasmBase64 must carry a non-trivial base64 payload")
	assert.Regexp(t, cartAssign, html, "cartBase64 must carry a non-trivial base64 payload")

	// Splash ordering: window.__pixelforgeCart MUST be set BEFORE
	// go.run executes — otherwise the player's first ReadSelf
	// returns ErrNoCart and the boot fails. This is the same
	// invariant the builders_long unit test guards; we re-check
	// here so a regression that drops the ordering surfaces in
	// the U12 smoke as well.
	cartIdx := strings.Index(html, "window.__pixelforgeCart")
	runIdx := strings.Index(html, "go.run(")
	require.GreaterOrEqual(t, cartIdx, 0, "WASM splash must assign window.__pixelforgeCart")
	require.GreaterOrEqual(t, runIdx, 0, "WASM splash must call go.run")
	assert.Less(t, cartIdx, runIdx,
		"window.__pixelforgeCart assignment must precede go.run; otherwise ReadSelf returns ErrNoCart")
}

// TestBuildPipelineUniversal_EmptyProjectSmoke covers the v2
// graceful-fallback path: a brand-new "New Project" with no scenes
// or sprites must still build + boot + tick cleanly. The smoke
// proves the universal player handles content-less carts without
// panicking — the failure mode a user who clicks Build before
// authoring anything would otherwise hit.
func TestBuildPipelineUniversal_EmptyProjectSmoke(t *testing.T) {
	if _, err := buildpipeline.ResolveGoBinary(); err != nil {
		t.Skipf("long-tag smoke requires the Go toolchain on PATH: %v", err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("Windows .syso relink not covered in CI smoke")
	}

	project := pixelforge_project.NewProject("empty_proof")
	outDir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	req := buildpipeline.BuildRequest{
		Project:        project,
		OutputDir:      outDir,
		EnginePath:     engineRepoRoot(t),
		EngineStrategy: buildpipeline.ModuleStrategyDevReplace,
	}
	statusCh := buildpipeline.Build(ctx, req, []buildpipeline.Target{buildpipeline.HostTarget()})
	statuses := drainStatusesUntilTerminal(t, statusCh, 5*time.Minute)
	require.NotEmpty(t, statuses)
	terminal := statuses[len(statuses)-1]
	require.Equal(t, buildpipeline.PhaseDone, terminal.Phase,
		"empty-project host build must reach PhaseDone; got %+v", statuses)

	binaryPath := hostArtifactPath(t, outDir, project)
	// 10 ticks is enough to prove the runtime boots + advances;
	// shorter than the Asteroids smoke because there's no scene
	// to step through.
	output := runSmokeSubprocess(t, binaryPath, 10, smokeTickTimeout)
	assert.Contains(t, output, "smoke-tick completed",
		"empty-project smoke binary must log the completion marker")
}
