// builders_long_test.go exercises the U3 universal-player + cart-
// append pipeline. Runs only under -tags=long because it shells
// out to `go build` for the developer-fallback scenario and reads
// real pre-built player binaries via the discovery chain. The
// non-long suite stays headless (builders.go's scaffold path).

//go:build long

package buildpipeline

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cart"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// minimalCart returns a fresh project sized to U3's exercise
// shape. Callers pass the project to the builder; the resulting
// cart bytes are extracted via readCartFromFile + compared by
// round-tripping back into a Project so the per-call ModifiedAt
// drift in pixelforge_project.Encode doesn't cause a byte-level
// false positive.
func minimalCart(t *testing.T, name string) *pixelforge_project.Project {
	t.Helper()
	p := pixelforge_project.NewProject(name)
	p.ScreenWidth = 320
	p.ScreenHeight = 200
	p.TPS = 60
	return p
}

// fakePlayerBinary stages a player-binary placeholder on disk —
// just enough content to exercise pixelforge_cart.Append + ReadSelf
// round-trip without needing the actual Ebitengine player. Returns
// the path.
func fakePlayerBinary(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "pixelforge-player-fake")
	// Sufficiently long that the footer-strip math in
	// pixelforge_cart isn't tripping over an artificially small
	// "host program" boundary.
	playerBytes := make([]byte, 4096)
	for i := range playerBytes {
		playerBytes[i] = byte(i % 251)
	}
	require.NoError(t, os.WriteFile(path, playerBytes, 0o755))
	return path
}

// TestHostLongBuilder_HappyPath_AppendsCart covers the U3 host
// happy path on Linux/macOS hosts: the builder copies the player
// binary, appends the cart payload, and the result round-trips
// via pixelforge_cart.ReadSelf to the original cart bytes.
//
// On Windows the host builder triggers a relink step (.syso + go
// build) which is too expensive for the default long suite; that
// scenario is deferred to U12's smoke test.
func TestHostLongBuilder_HappyPath_AppendsCart(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows host triggers the .syso relink path; covered by U12 smoke test")
	}

	stagingDir := t.TempDir()
	playerPath := fakePlayerBinary(t, stagingDir)

	project := minimalCart(t, "happy_host")
	outputDir := filepath.Join(stagingDir, "out")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	req := BuildRequest{
		Project:          project,
		OutputDir:        outputDir,
		PlayerBinaryPath: playerPath, // skip discovery chain
	}

	var lastStatus BuildStatus
	emit := func(s BuildStatus) { lastStatus = s }
	builder := &hostLongBuilder{}
	require.NoError(t, builder.Build(context.Background(), req, emit))

	// Output exists at outputDir/host/<name>[.app on darwin]
	hostDir := filepath.Join(outputDir, "host")
	entries, err := os.ReadDir(hostDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries, "host build must produce at least one output entry")

	// On darwin the artifact is the .app bundle (a directory) and
	// the inner Mach-O carries the cart. Find the file pixelforge_cart
	// would target.
	gameName := projectGameName(project)
	verifyPath := filepath.Join(hostDir, gameName+artifactExt(HostTarget()))
	if HostTarget() == TargetMacOS {
		verifyPath = filepath.Join(hostDir, gameName+".app", "Contents", "MacOS", gameName)
	}
	info, err := os.Stat(verifyPath)
	require.NoError(t, err, "expected cart-bearing binary at %s", verifyPath)
	assert.False(t, info.IsDir(), "cart-bearing artifact must be a file (not a directory)")

	// Round-trip: read the file's appended cart payload back via
	// the same DecodeFooter wire format the runtime sees, then
	// LoadReader it to confirm the project survives the trip.
	// We compare structural project fields (Name/ScreenWidth/...)
	// instead of raw bytes because pixelforge_project.Encode bumps
	// ModifiedAt on each invocation; a byte-level compare would
	// false-positive on the timestamp drift, not on real cart
	// corruption.
	gotCart, err := readCartFromFile(verifyPath)
	require.NoError(t, err, "ReadSelf-equivalent extraction must succeed on the appended binary")
	require.NotEmpty(t, gotCart, "cart payload must be non-empty")
	loaded, err := pixelforge_project.LoadReader(bytes.NewReader(gotCart))
	require.NoError(t, err, "LoadReader must accept the appended cart bytes")
	assert.Equal(t, project.Name, loaded.Name)
	assert.Equal(t, project.ScreenWidth, loaded.ScreenWidth)
	assert.Equal(t, project.ScreenHeight, loaded.ScreenHeight)
	assert.Equal(t, project.TPS, loaded.TPS)
	assert.Equal(t, PhaseDone, lastStatus.Phase, "builder must emit PhaseDone")
}

// TestWasmLongBuilder_HappyPath_InlinesBothBlobs covers the U3
// WASM happy path: codegen.EncodeCart → BundleWASM inlines both
// the WASM module and the cart base64 + the splash that wires
// window.__pixelforgeCart before go.run.
func TestWasmLongBuilder_HappyPath_InlinesBothBlobs(t *testing.T) {
	stagingDir := t.TempDir()
	// Stage a fake WASM module that ResolvePlayerBinary's explicit
	// override hands off to BundleWASM. Real WASM magic header
	// keeps the bundler happy even though the bytes don't link.
	fakeWasm := filepath.Join(stagingDir, "player.wasm")
	require.NoError(t, os.WriteFile(fakeWasm, []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0xff, 0xff}, 0o644))

	project := minimalCart(t, "happy_wasm")
	outputDir := filepath.Join(stagingDir, "out")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	req := BuildRequest{
		Project:        project,
		OutputDir:      outputDir,
		WasmBinaryPath: fakeWasm,
	}

	builder := &wasmLongBuilder{}
	var lastStatus BuildStatus
	emit := func(s BuildStatus) { lastStatus = s }
	require.NoError(t, builder.Build(context.Background(), req, emit))

	outPath := filepath.Join(outputDir, "wasm", projectGameName(project)+".html")
	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	html := string(data)
	assert.NotEmpty(t, html)

	// Non-trivial base64 in both placeholder slots (the bundler
	// fills `const wasmBase64 = "..."` and `const cartBase64 =
	// "..."`). html/template escapes the "/" characters in the
	// base64 alphabet to `\/` inside JS string literals, so the
	// regex character class allows the escaped form alongside the
	// raw form.
	wasmAssign := regexp.MustCompile(`const wasmBase64 = "([A-Za-z0-9+/=\\]{8,})"`)
	cartAssign := regexp.MustCompile(`const cartBase64 = "([A-Za-z0-9+/=\\]{8,})"`)
	assert.Regexp(t, wasmAssign, html, "wasmBase64 must carry a non-trivial base64 payload")
	assert.Regexp(t, cartAssign, html, "cartBase64 must carry a non-trivial base64 payload")

	// Splash ordering: window.__pixelforgeCart MUST come before
	// go.run(instance) or the player's first ReadSelf is empty.
	cartIdx := strings.Index(html, "window.__pixelforgeCart")
	runIdx := strings.Index(html, "go.run(")
	require.GreaterOrEqual(t, cartIdx, 0)
	require.GreaterOrEqual(t, runIdx, 0)
	assert.Less(t, cartIdx, runIdx,
		"the cart-payload assignment must precede the go.run invocation")

	assert.Equal(t, PhaseDone, lastStatus.Phase, "builder must emit PhaseDone")
}

// TestHostLongBuilder_CrossCompileRejected preserves plan-008's
// preflight invariant: foreign-OS native targets reach a
// rejectingBuilder via the orchestrator + emit
// CrossCompileNotSupportedError.
func TestHostLongBuilder_CrossCompileRejected(t *testing.T) {
	// Pick a foreign target relative to the host. On Linux: Windows;
	// on darwin: Windows; etc. (anything that isn't HostTarget()).
	foreign := TargetWindows
	if HostTarget() == TargetWindows {
		foreign = TargetLinux
	}

	builder, ok := LookupBuilder(foreign)
	require.True(t, ok, "rejectingBuilder must be registered for %s", foreign)

	emit := func(BuildStatus) {}
	err := builder.Build(context.Background(), BuildRequest{}, emit)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCrossCompileNotSupported),
		"foreign target must return ErrCrossCompileNotSupported, got %v", err)
}

// TestWasmLongBuilder_CrossCompileAllowed covers the WASM
// invariant: GOOS=js/GOARCH=wasm is by definition a cross-compile
// and is permitted from every host OS. Without this carve-out the
// preflight would refuse Linux→WASM, Windows→WASM, etc.
func TestWasmLongBuilder_CrossCompileAllowed(t *testing.T) {
	assert.True(t, CanBuildOnHost(TargetWASM),
		"WASM must be buildable from every host OS")
}

// TestResolvePlayerBinary_ExplicitOverride covers discovery
// step 1: an explicit PlayerBinaryPath short-circuits the cache +
// embed + go-build chain.
func TestResolvePlayerBinary_ExplicitOverride(t *testing.T) {
	dir := t.TempDir()
	playerPath := fakePlayerBinary(t, dir)
	req := BuildRequest{PlayerBinaryPath: playerPath}

	resolved, err := resolvePlayerBinary(context.Background(), req, HostTarget())
	require.NoError(t, err)
	defer resolved.Cleanup()
	assert.Equal(t, playerPath, resolved.Path)
}

// TestResolvePlayerBinary_ExplicitOverride_MissingFile rejects an
// override that points at a non-existent file rather than
// silently falling through to the embed/build chain — a typo in
// the override should fail loudly so the user fixes it.
func TestResolvePlayerBinary_ExplicitOverride_MissingFile(t *testing.T) {
	req := BuildRequest{PlayerBinaryPath: "/no/such/path/pixelforge-player"}
	_, err := resolvePlayerBinary(context.Background(), req, HostTarget())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "explicit player path")
}

// TestResolvePlayerBinary_CacheHit covers discovery step 2: a
// valid cache entry with matching SHA-256 sidecar short-circuits
// the embed extraction.
func TestResolvePlayerBinary_CacheHit(t *testing.T) {
	// Redirect the cache directory under t.TempDir() via
	// XDG_CACHE_HOME so other test runs + the real user cache
	// don't poison this test.
	cacheRoot := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheRoot)
	if runtime.GOOS == "darwin" {
		// On darwin os.UserCacheDir reads HOME/Library/Caches.
		t.Setenv("HOME", cacheRoot)
	}

	target := HostTarget()
	dir, err := playerCacheDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(dir, 0o755))

	payload := []byte("FAKE_PLAYER_BYTES_FOR_CACHE_HIT_TEST")
	path := filepath.Join(dir, playerCacheFilename(target))
	require.NoError(t, os.WriteFile(path, payload, 0o755))
	require.NoError(t, os.WriteFile(path+".sha256", []byte(sha256HexOf(payload)+"\n"), 0o644))

	req := BuildRequest{}
	resolved, err := resolvePlayerBinary(context.Background(), req, target)
	require.NoError(t, err)
	defer resolved.Cleanup()
	assert.Equal(t, path, resolved.Path, "valid cache entry must be returned verbatim")
}

// TestResolvePlayerBinary_CacheChecksumMismatch covers the
// poisoned-cache protection: a binary whose SHA-256 doesn't match
// its sidecar must be invalidated.
func TestResolvePlayerBinary_CacheChecksumMismatch(t *testing.T) {
	cacheRoot := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheRoot)
	if runtime.GOOS == "darwin" {
		t.Setenv("HOME", cacheRoot)
	}

	target := HostTarget()
	dir, err := playerCacheDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(dir, 0o755))

	path := filepath.Join(dir, playerCacheFilename(target))
	require.NoError(t, os.WriteFile(path, []byte("REAL_BYTES"), 0o755))
	// Sidecar carries the WRONG hash — simulates a poisoned cache.
	require.NoError(t, os.WriteFile(path+".sha256", []byte("0000000000000000000000000000000000000000000000000000000000000000\n"), 0o644))

	// lookupPlayerCache must return false; the entry's binary + sidecar
	// should be removed as a side effect.
	gotPath, ok := lookupPlayerCache(target)
	assert.False(t, ok, "checksum mismatch must invalidate the cache")
	assert.Empty(t, gotPath)

	_, statErr := os.Stat(path)
	assert.Error(t, statErr, "invalidated binary must be removed")
}

// TestResolvePlayerBinary_EmbedExtract covers discovery step 3
// via the playerbins test-fake fixture. Uses the fixture's
// "test/fake" GOOS/GOARCH pair by mocking targetToGOOS/GOARCH —
// since the fixture lives at bins/test-fake/pixelforge-player, we
// need a Target whose GOOS/GOARCH report "test"/"fake". This unit
// has no such target, so the embed-extract scenario is exercised
// indirectly via the playerbins package's own tests; here we
// confirm the chain calls into playerbins by checking the error
// surface when no embedded binary is available for the real
// target (and no cache + no explicit override).
//
// The full happy-path embed extraction lands when `make playerbins`
// has populated the real per-target bins/<goos>-<goarch>/
// pixelforge-player entries; see plan-009 U3 verification notes.
func TestResolvePlayerBinary_EmbedFallsThroughToBuild(t *testing.T) {
	// Isolate cache dir so a stray previous run doesn't satisfy
	// the lookup before we reach the embed step.
	cacheRoot := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheRoot)
	if runtime.GOOS == "darwin" {
		t.Setenv("HOME", cacheRoot)
	}

	// The dev checkout's playerbins/bins/ is empty (only .gitkeep
	// + the test-fake fixture) for HostTarget()'s GOOS/GOARCH, so
	// the discovery chain MUST reach the developer go-build
	// fallback. That fallback is slow (compiles the player); we
	// just confirm the function returns a non-empty path without
	// failing.
	if testing.Short() {
		t.Skip("developer go-build fallback is slow; skip under -short")
	}

	req := BuildRequest{}
	resolved, err := resolvePlayerBinary(context.Background(), req, HostTarget())
	require.NoError(t, err, "developer fallback must succeed on a dev checkout with go on PATH")
	defer resolved.Cleanup()
	assert.NotEmpty(t, resolved.Path)
	info, statErr := os.Stat(resolved.Path)
	require.NoError(t, statErr)
	assert.False(t, info.IsDir())
}

// TestHostLongBuilder_EmptyProject covers the corner where the
// user clicks Build against a fresh "New Project" with no scenes
// or sprites yet — EncodeCart still produces valid bytes and the
// pipeline runs end to end.
func TestHostLongBuilder_EmptyProject(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows relink path is U12's territory")
	}

	stagingDir := t.TempDir()
	playerPath := fakePlayerBinary(t, stagingDir)

	project := pixelforge_project.NewProject("empty_proof")
	outputDir := filepath.Join(stagingDir, "out")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	req := BuildRequest{
		Project:          project,
		OutputDir:        outputDir,
		PlayerBinaryPath: playerPath,
	}
	require.NoError(t, (&hostLongBuilder{}).Build(context.Background(), req, func(BuildStatus) {}))

	hostDir := filepath.Join(outputDir, "host")
	entries, err := os.ReadDir(hostDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries, "empty project must still produce an artifact")
}

// TestWasmLongBuilder_EmitsSizeReport covers the plan-009 U23
// invariant: a successful WASM build emits PhaseDone with a
// populated WASMSizeReport AND writes the gzip sibling alongside
// the .html.
func TestWasmLongBuilder_EmitsSizeReport(t *testing.T) {
	stagingDir := t.TempDir()
	fakeWasm := filepath.Join(stagingDir, "player.wasm")
	require.NoError(t, os.WriteFile(fakeWasm, []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0xff, 0xff}, 0o644))

	project := minimalCart(t, "size_report")
	outputDir := filepath.Join(stagingDir, "out")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	req := BuildRequest{
		Project:        project,
		OutputDir:      outputDir,
		WasmBinaryPath: fakeWasm,
	}

	var lastStatus BuildStatus
	emit := func(s BuildStatus) {
		if s.Phase == PhaseDone {
			lastStatus = s
		}
	}
	require.NoError(t, (&wasmLongBuilder{}).Build(context.Background(), req, emit))

	require.Equal(t, PhaseDone, lastStatus.Phase)
	require.NotNil(t, lastStatus.SizeReport,
		"WASM PhaseDone must carry a populated SizeReport")
	assert.Greater(t, lastStatus.SizeReport.RawBytes, int64(0))
	assert.Greater(t, lastStatus.SizeReport.GzipBytes, int64(0))

	gzPath := filepath.Join(outputDir, "wasm", projectGameName(project)+".html.gz")
	gzInfo, err := os.Stat(gzPath)
	require.NoError(t, err, "WASM build must produce <name>.html.gz alongside <name>.html")
	assert.Equal(t, lastStatus.SizeReport.GzipBytes, gzInfo.Size(),
		"SizeReport.GzipBytes must match the stat'd .gz file size")
}

// TestWasmLongBuilder_RejectsOversizeWithoutForce covers the
// 30MB hard-limit gate. We can't realistically pad a wasm to
// 30MB in the test (the test would balloon to ~40MB in flight),
// so we exercise the gate via a direct call to BundleWASMWithSize
// + threshold check using a small fixture but with synthetic
// thresholds — that's covered in wasm_size_report_test.go.
// Here we cover the orchestrator pathway by asserting the error
// type satisfies errors.Is(ErrWASMTooLarge).
func TestWasmLongBuilder_WASMTooLargeErrorSatisfiesSentinel(t *testing.T) {
	err := &WASMTooLargeError{Report: WASMSizeReport{
		RawBytes:         35 * 1024 * 1024,
		GzipBytes:        12 * 1024 * 1024,
		WarnThresholdMB:  WASMWarnThresholdMB,
		ErrorThresholdMB: WASMErrorThresholdMB,
	}}
	assert.True(t, errors.Is(err, ErrWASMTooLarge),
		"WASMTooLargeError must satisfy errors.Is(ErrWASMTooLarge)")
	// Error message must name the threshold + the size suffix.
	msg := err.Error()
	assert.Contains(t, msg, "30MB")
	assert.Contains(t, msg, "ForceLargeWASM")
}

// readCartFromFile is the test-side equivalent of
// pixelforge_cart.ReadSelf: open a binary on disk, walk back to
// the footer, decode + verify, and return the cart payload bytes.
// Uses only the public pixelforge_cart helpers so we're testing
// the same wire format the runtime sees.
func readCartFromFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < pixelforge_cart.FooterSize {
		return nil, errors.New("file shorter than footer")
	}
	footer := data[len(data)-pixelforge_cart.FooterSize:]
	length, _, err := pixelforge_cart.DecodeFooter(footer)
	if err != nil {
		return nil, err
	}
	if uint64(len(data)) < uint64(pixelforge_cart.FooterSize)+length {
		return nil, errors.New("file shorter than footer+payload")
	}
	start := uint64(len(data)) - uint64(pixelforge_cart.FooterSize) - length
	end := uint64(len(data)) - uint64(pixelforge_cart.FooterSize)
	return data[start:end], nil
}
