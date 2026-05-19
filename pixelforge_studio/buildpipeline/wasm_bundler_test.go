package buildpipeline_test

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/buildpipeline"
)

func writeFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, content, 0o644))
	return path
}

func writeFakeWasm(t *testing.T, dir string) string {
	// Minimal WASM magic header + a few bytes of payload so it's
	// non-empty and base64-encodes deterministically.
	bytes := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	return writeFile(t, dir, "game.wasm", bytes)
}

func writeFakeWasmExec(t *testing.T, dir string) string {
	js := []byte(`// fake wasm_exec.js for tests
function Go() { /* mock runtime */ }
`)
	return writeFile(t, dir, "wasm_exec.js", js)
}

// fakeCart is the canonical "cart payload" the wasm_bundler tests
// feed into BundleWASM. Non-empty so base64 encoding produces a
// distinguishable string we can assert against.
func fakeCart() []byte {
	return []byte(`{"schema_version":1,"name":"BundleWASMTest"}`)
}

func TestBundleWASM_OutputIsSingleHTMLFile(t *testing.T) {
	dir := t.TempDir()
	wasmPath := writeFakeWasm(t, dir)
	execPath := writeFakeWasmExec(t, dir)
	outPath := filepath.Join(dir, "game.html")

	err := buildpipeline.BundleWASM(wasmPath, execPath, fakeCart(), "TestGame", "", outPath, nil)
	require.NoError(t, err)
	info, err := os.Stat(outPath)
	require.NoError(t, err)
	assert.False(t, info.IsDir())
}

func TestBundleWASM_HTMLContainsGameName(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.html")
	require.NoError(t, buildpipeline.BundleWASM(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		fakeCart(), "MyGame", "", out, nil))
	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Contains(t, string(data), "<title>MyGame</title>")
}

func TestBundleWASM_HTMLContainsWasmExecJSInline(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.html")
	require.NoError(t, buildpipeline.BundleWASM(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		fakeCart(), "X", "", out, nil))
	data, _ := os.ReadFile(out)
	assert.Contains(t, string(data), "function Go()")
}

func TestBundleWASM_HTMLContainsBase64Wasm(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.html")
	require.NoError(t, buildpipeline.BundleWASM(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		fakeCart(), "X", "", out, nil))
	data, _ := os.ReadFile(out)
	assert.Contains(t, string(data), "wasmBase64")
}

// TestBundleWASM_HTMLContainsBase64Cart covers the U3 inline-cart
// invariant: the cart payload rides as a base64 literal next to
// the WASM so the splash handler can wire window.__pixelforgeCart
// before booting the runtime.
func TestBundleWASM_HTMLContainsBase64Cart(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.html")
	cart := fakeCart()
	require.NoError(t, buildpipeline.BundleWASM(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		cart, "X", "", out, nil))
	data, _ := os.ReadFile(out)
	s := string(data)
	assert.Contains(t, s, "cartBase64", "cart base64 literal must appear in the inlined script")
	expectedB64 := base64.StdEncoding.EncodeToString(cart)
	assert.Contains(t, s, expectedB64,
		"the actual base64-encoded cart bytes must appear verbatim")
}

// TestBundleWASM_SplashAssignsCartBeforeGoRun guards the load-
// bearing ordering: window.__pixelforgeCart must land BEFORE
// go.run(instance) or the WASM player's first ReadSelf returns
// undefined.
func TestBundleWASM_SplashAssignsCartBeforeGoRun(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.html")
	require.NoError(t, buildpipeline.BundleWASM(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		fakeCart(), "X", "", out, nil))
	data, _ := os.ReadFile(out)
	s := string(data)
	cartIdx := strings.Index(s, "window.__pixelforgeCart")
	runIdx := strings.Index(s, "go.run(")
	require.NotEqual(t, -1, cartIdx, "expected window.__pixelforgeCart assignment in template")
	require.NotEqual(t, -1, runIdx, "expected go.run invocation in template")
	assert.Less(t, cartIdx, runIdx,
		"window.__pixelforgeCart must be assigned before go.run(instance)")
}

func TestBundleWASM_HTMLContainsFaviconWhenSupplied(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.html")
	require.NoError(t, buildpipeline.BundleWASM(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		fakeCart(), "X", "AAAA", out, nil))
	data, _ := os.ReadFile(out)
	assert.Contains(t, string(data), "data:image/png;base64,AAAA")
}

func TestBundleWASM_HTMLContainsSplash(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.html")
	require.NoError(t, buildpipeline.BundleWASM(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		fakeCart(), "X", "", out, nil))
	data, _ := os.ReadFile(out)
	assert.Contains(t, string(data), `id="splash"`)
	assert.Contains(t, string(data), "Click to start")
}

func TestBundleWASM_NoExternalRefs(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.html")
	require.NoError(t, buildpipeline.BundleWASM(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		fakeCart(), "X", "", out, nil))
	data, _ := os.ReadFile(out)
	s := string(data)
	// No <script src=> with http/https URL.
	assert.False(t, strings.Contains(s, `<script src="http`),
		"WASM bundle is offline-only — no external script URLs")
	assert.False(t, strings.Contains(s, `<link href="http`),
		"WASM bundle is offline-only — no external link URLs")
}

func TestBundleWASM_EmptyWasmPathReturnsError(t *testing.T) {
	err := buildpipeline.BundleWASM("", "anything", fakeCart(), "X", "", "out.html", nil)
	require.Error(t, err)
}

func TestBundleWASM_EmptyWasmExecJSPathReturnsError(t *testing.T) {
	err := buildpipeline.BundleWASM("game.wasm", "", fakeCart(), "X", "", "out.html", nil)
	require.Error(t, err)
}

func TestBundleWASM_MissingWasmReturnsError(t *testing.T) {
	dir := t.TempDir()
	err := buildpipeline.BundleWASM(
		filepath.Join(dir, "no.wasm"),
		writeFakeWasmExec(t, dir),
		fakeCart(), "X", "", filepath.Join(dir, "out.html"), nil)
	require.Error(t, err)
}

func TestBundleWASM_EmptyWasmFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	emptyWasm := writeFile(t, dir, "empty.wasm", nil)
	err := buildpipeline.BundleWASM(
		emptyWasm,
		writeFakeWasmExec(t, dir),
		fakeCart(), "X", "", filepath.Join(dir, "out.html"), nil)
	require.Error(t, err)
}

func TestBundleWASM_GameNameEscapedForHTML(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.html")
	require.NoError(t, buildpipeline.BundleWASM(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		fakeCart(), "<script>alert(1)</script>", "", out, nil))
	data, _ := os.ReadFile(out)
	s := string(data)
	// html/template auto-escapes the title element value.
	assert.NotContains(t, s, "<script>alert(1)</script>",
		"game name with HTML chars must escape before reaching the title element")
}

func TestEstimateWASMHTMLSize_BasicInflation(t *testing.T) {
	// 3 MB wasm → ~4 MB HTML + ~35KB overhead.
	got := buildpipeline.EstimateWASMHTMLSize(3 * 1024 * 1024)
	assert.Greater(t, got, int64(3*1024*1024))
	assert.Less(t, got, int64(5*1024*1024))
}

// TestBundleWASMWithSize_ReturnsRawHTMLSize covers the plan-009
// U23 contract: BundleWASMWithSize reports the on-disk HTML byte
// count in WASMSizeReport.RawBytes.
func TestBundleWASMWithSize_ReturnsRawHTMLSize(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "g.html")
	report, err := buildpipeline.BundleWASMWithSize(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		fakeCart(), "X", "", out, nil,
		buildpipeline.BundleWASMOptions{},
	)
	require.NoError(t, err)

	info, err := os.Stat(out)
	require.NoError(t, err)
	assert.Equal(t, info.Size(), report.RawBytes,
		"report.RawBytes must equal the stat'd output file size")
	assert.Equal(t, buildpipeline.WASMWarnThresholdMB, report.WarnThresholdMB)
	assert.Equal(t, buildpipeline.WASMErrorThresholdMB, report.ErrorThresholdMB)
}

// TestBundleWASMWithSize_NoGzipByDefault confirms the default
// opts shape doesn't write a gzip sibling — only callers that
// opt in pay the cost.
func TestBundleWASMWithSize_NoGzipByDefault(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "g.html")
	report, err := buildpipeline.BundleWASMWithSize(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		fakeCart(), "X", "", out, nil,
		buildpipeline.BundleWASMOptions{},
	)
	require.NoError(t, err)

	_, statErr := os.Stat(out + ".gz")
	assert.True(t, os.IsNotExist(statErr),
		"WriteGzipSibling=false must NOT emit a .gz file")
	assert.Equal(t, int64(0), report.GzipBytes,
		"GzipBytes must be zero when gzip sibling is off")
}

// TestBundleWASMWithSize_GzipSiblingExists covers the gzip
// invariant: when WriteGzipSibling is true, `<outPath>.gz` lands
// on disk AND its byte count is non-zero AND smaller than the
// uncompressed HTML.
func TestBundleWASMWithSize_GzipSiblingExists(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "g.html")
	report, err := buildpipeline.BundleWASMWithSize(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		fakeCart(), "X", "", out, nil,
		buildpipeline.BundleWASMOptions{WriteGzipSibling: true},
	)
	require.NoError(t, err)

	info, err := os.Stat(out + ".gz")
	require.NoError(t, err, "WriteGzipSibling=true must emit a .gz file")
	assert.Greater(t, info.Size(), int64(0))
	assert.Equal(t, info.Size(), report.GzipBytes,
		"report.GzipBytes must equal the stat'd .gz size")

	// For a non-trivial HTML payload the gzip output is smaller
	// than the raw. Our fake wasm is tiny, but the template +
	// inline JS still compresses; assert non-strict equality
	// against the bound (raw size >= gzip size).
	assert.LessOrEqual(t, report.GzipBytes, report.RawBytes,
		"gzipped output must be no larger than raw HTML")
}

// TestBundleWASMWithSize_GzipRoundTripsToOriginal confirms the
// .gz file decompresses byte-for-byte back to the .html. Catches
// gzip writer bugs (forgot to Close, used wrong level, etc.).
func TestBundleWASMWithSize_GzipRoundTripsToOriginal(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "g.html")
	_, err := buildpipeline.BundleWASMWithSize(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		fakeCart(), "X", "", out, nil,
		buildpipeline.BundleWASMOptions{WriteGzipSibling: true},
	)
	require.NoError(t, err)

	rawHTML, err := os.ReadFile(out)
	require.NoError(t, err)
	gzBytes, err := os.ReadFile(out + ".gz")
	require.NoError(t, err)

	gzReader, err := gzip.NewReader(bytes.NewReader(gzBytes))
	require.NoError(t, err)
	defer gzReader.Close()
	decoded, err := io.ReadAll(gzReader)
	require.NoError(t, err)

	assert.Equal(t, rawHTML, decoded,
		"gunzip of <out>.gz must reproduce <out> byte-for-byte")
}

// TestBundleWASM_BackwardCompatSignatureUnchanged guards the old
// public signature — callers that didn't opt into size reporting
// (the e2e integration tests in pixelforge_studio/integration_test/)
// still compile + work.
func TestBundleWASM_BackwardCompatSignatureUnchanged(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "g.html")
	err := buildpipeline.BundleWASM(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		fakeCart(), "X", "", out, nil)
	require.NoError(t, err)

	// No gzip sibling for the legacy path.
	_, statErr := os.Stat(out + ".gz")
	assert.True(t, os.IsNotExist(statErr))
}
