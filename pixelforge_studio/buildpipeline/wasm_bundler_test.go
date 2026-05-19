package buildpipeline_test

import (
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

func TestBundleWASM_OutputIsSingleHTMLFile(t *testing.T) {
	dir := t.TempDir()
	wasmPath := writeFakeWasm(t, dir)
	execPath := writeFakeWasmExec(t, dir)
	outPath := filepath.Join(dir, "game.html")

	err := buildpipeline.BundleWASM(wasmPath, execPath, "TestGame", "", outPath, nil)
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
		"MyGame", "", out, nil))
	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Contains(t, string(data), "<title>MyGame</title>")
}

func TestBundleWASM_HTMLContainsWasmExecJSInline(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.html")
	require.NoError(t, buildpipeline.BundleWASM(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		"X", "", out, nil))
	data, _ := os.ReadFile(out)
	assert.Contains(t, string(data), "function Go()")
}

func TestBundleWASM_HTMLContainsBase64Wasm(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.html")
	require.NoError(t, buildpipeline.BundleWASM(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		"X", "", out, nil))
	data, _ := os.ReadFile(out)
	assert.Contains(t, string(data), "wasmBase64")
}

func TestBundleWASM_HTMLContainsFaviconWhenSupplied(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.html")
	require.NoError(t, buildpipeline.BundleWASM(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		"X", "AAAA", out, nil))
	data, _ := os.ReadFile(out)
	assert.Contains(t, string(data), "data:image/png;base64,AAAA")
}

func TestBundleWASM_HTMLContainsSplash(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.html")
	require.NoError(t, buildpipeline.BundleWASM(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		"X", "", out, nil))
	data, _ := os.ReadFile(out)
	assert.Contains(t, string(data), `id="splash"`)
	assert.Contains(t, string(data), "Click to start")
}

func TestBundleWASM_NoExternalRefs(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.html")
	require.NoError(t, buildpipeline.BundleWASM(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		"X", "", out, nil))
	data, _ := os.ReadFile(out)
	s := string(data)
	// No <script src=> with http/https URL.
	assert.False(t, strings.Contains(s, `<script src="http`),
		"WASM bundle is offline-only — no external script URLs")
	assert.False(t, strings.Contains(s, `<link href="http`),
		"WASM bundle is offline-only — no external link URLs")
}

func TestBundleWASM_EmptyWasmPathReturnsError(t *testing.T) {
	err := buildpipeline.BundleWASM("", "anything", "X", "", "out.html", nil)
	require.Error(t, err)
}

func TestBundleWASM_EmptyWasmExecJSPathReturnsError(t *testing.T) {
	err := buildpipeline.BundleWASM("game.wasm", "", "X", "", "out.html", nil)
	require.Error(t, err)
}

func TestBundleWASM_MissingWasmReturnsError(t *testing.T) {
	dir := t.TempDir()
	err := buildpipeline.BundleWASM(
		filepath.Join(dir, "no.wasm"),
		writeFakeWasmExec(t, dir),
		"X", "", filepath.Join(dir, "out.html"), nil)
	require.Error(t, err)
}

func TestBundleWASM_EmptyWasmFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	emptyWasm := writeFile(t, dir, "empty.wasm", nil)
	err := buildpipeline.BundleWASM(
		emptyWasm,
		writeFakeWasmExec(t, dir),
		"X", "", filepath.Join(dir, "out.html"), nil)
	require.Error(t, err)
}

func TestBundleWASM_GameNameEscapedForHTML(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.html")
	require.NoError(t, buildpipeline.BundleWASM(
		writeFakeWasm(t, dir), writeFakeWasmExec(t, dir),
		"<script>alert(1)</script>", "", out, nil))
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
