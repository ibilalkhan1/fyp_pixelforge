// credits_e2e_test.go covers plan-008 U10 end-to-end at the
// codegen + WASM bundler boundary. The credits dataset assembled
// by assetlibrary.AssembleCredits round-trips through the
// generator's capsule.go literal AND through the WASM bundler's
// HTML credits div + "View Credits" splash button.
//
// Default-tag friendly: doesn't invoke the real Go toolchain.

package integration_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/assetlibrary"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/buildpipeline"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/codegen"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/modulepath"
)

// repoRoot walks up from this test source file to the engine's
// go.mod. The codegen tests pin StrategyDevReplace + this path so
// modulepath.Apply doesn't fail trying to copy the full engine
// tree into vendor/.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir := filepath.Dir(here)
	for i := 0; i < 8; i++ {
		gomod := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(gomod); err == nil {
			if strings.Contains(string(data), "module "+modulepath.EngineModule) {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not find engine go.mod starting at %s", here)
	return ""
}

func devReplaceOpts(t *testing.T, credits []capsuleruntime.CreditEntry) codegen.Options {
	return codegen.Options{
		Force:      true,
		Strategy:   modulepath.StrategyDevReplace,
		EnginePath: repoRoot(t),
		Credits:    credits,
	}
}

// libWithCCBYAudio returns a library carrying one CC-BY audio
// asset and one CC0 sprite asset. Credits should include the
// audio entry and exclude the sprite.
func libWithCCBYAudio(t *testing.T) *assetlibrary.Library {
	t.Helper()
	lib := assetlibrary.NewLibrary(t.TempDir())
	lib.MarkInstalled(assetlibrary.Pack{
		ID: "mix",
		Assets: []assetlibrary.Asset{
			{Path: "audio/blast.wav", Kind: "sfx", License: "CC-BY-4.0",
				Author: "freesound user X", SourceURL: "https://freesound.org/x"},
			{Path: "sprites/ship.png", Kind: "sprite", License: "CC0", Author: "Kenney"},
		},
	})
	return lib
}

func projectWithMixedAssets() *pixelforge_project.Project {
	p := pixelforge_project.NewProject("credits_test")
	p.Sprites = []pixelforge_project.SpriteAsset{
		{Name: "ship", RelativePath: "sprites/ship.png", Width: 16, Height: 16},
	}
	p.Audio = []pixelforge_project.AudioSample{
		{Name: "blast", RelativePath: "audio/blast.wav"},
	}
	return p
}

// TestE2E_Credits_CCBYAppearsInGeneratedCapsule covers the
// codegen-side embedding: AssembleCredits → codegen.Options.Credits
// → capsuleCredits literal in capsule.go.
func TestE2E_Credits_CCBYAppearsInGeneratedCapsule(t *testing.T) {
	lib := libWithCCBYAudio(t)
	p := projectWithMixedAssets()
	credits := assetlibrary.AssembleCredits(p, lib)
	require.Len(t, credits, 1, "exactly one CC-BY asset should land in credits")

	outDir := t.TempDir()
	_, err := codegen.Generate(p, outDir, devReplaceOpts(t, credits))
	require.NoError(t, err)

	capsuleSrc, err := os.ReadFile(filepath.Join(outDir, "capsule.go"))
	require.NoError(t, err)
	src := string(capsuleSrc)

	assert.Contains(t, src, "capsuleruntime.CreditEntry",
		"generated capsule.go must reference the CreditEntry type")
	assert.Contains(t, src, `Name: "blast"`,
		"CC-BY audio sample must appear in the credits literal")
	assert.Contains(t, src, `License: "CC-BY-4.0"`)
	assert.Contains(t, src, `Author: "freesound user X"`)
	assert.Contains(t, src, "capsuleruntime.RegisterCredits(capsuleCredits)",
		"generated capsule must register credits at init")
	assert.NotContains(t, src, `Name: "ship"`,
		"CC0 sprite must not appear in the credits literal")
}

// TestE2E_Credits_CC0OnlyProjectEmitsEmptyCredits covers the
// quiet path: a CC0-only project produces an empty credits
// literal so the menu auto-suppresses the Credits entry.
func TestE2E_Credits_CC0OnlyProjectEmitsEmptyCredits(t *testing.T) {
	lib := assetlibrary.NewLibrary(t.TempDir())
	lib.MarkInstalled(assetlibrary.Pack{
		ID: "p",
		Assets: []assetlibrary.Asset{
			{Path: "sprites/ship.png", Kind: "sprite", License: "CC0", Author: "K"},
		},
	})
	p := pixelforge_project.NewProject("cc0_only")
	p.Sprites = []pixelforge_project.SpriteAsset{
		{Name: "ship", RelativePath: "sprites/ship.png", Width: 16, Height: 16},
	}
	credits := assetlibrary.AssembleCredits(p, lib)
	assert.Empty(t, credits)

	outDir := t.TempDir()
	_, err := codegen.Generate(p, outDir, devReplaceOpts(t, credits))
	require.NoError(t, err)

	capsuleSrc, err := os.ReadFile(filepath.Join(outDir, "capsule.go"))
	require.NoError(t, err)
	src := string(capsuleSrc)
	// The CC-BY token only appears in comments for an empty-credits
	// project — no literal CreditEntry with License: "CC-BY-..." should
	// be present.
	assert.NotContains(t, src, `License: "CC-BY`,
		"empty credits literal must not include any CC-BY license entry")
	// Empty slice literal still renders; the registry stays empty
	// and HasCredits returns false. The capsule registers the empty
	// slice at init.
	assert.Contains(t, src, "capsuleruntime.RegisterCredits(capsuleCredits)")
}

// TestE2E_Credits_WASMBundleShowsCreditsButton covers the WASM
// side: BundleWASM with non-empty credits renders the "View
// Credits" splash button + the credits div populated with entries.
func TestE2E_Credits_WASMBundleShowsCreditsButton(t *testing.T) {
	dir := t.TempDir()
	wasmPath := writeFakeWasm(t, dir)
	execPath := writeFakeWasmExec(t, dir)
	outPath := filepath.Join(dir, "game.html")

	credits := []capsuleruntime.CreditEntry{
		{Name: "blast", License: "CC-BY-4.0", Author: "freesound user X", SourceURL: "https://freesound.org/x"},
	}
	require.NoError(t, buildpipeline.BundleWASM(wasmPath, execPath, "Game", "", outPath, credits))

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	html := string(data)
	assert.Contains(t, html, `id="credits-open"`, "splash must carry the credits-open button")
	assert.Contains(t, html, `id="credits"`, "HTML must include the credits pane")
	assert.Contains(t, html, "blast", "credits entry name must render")
	assert.Contains(t, html, "CC-BY-4.0", "credits entry license must render")
	assert.Contains(t, html, "freesound user X", "credits entry author must render")
	assert.Contains(t, html, "https://freesound.org/x", "credits entry source URL must render as a link")
}

// TestE2E_Credits_WASMBundleHidesButtonWhenEmpty covers the
// quiet path on the WASM side.
func TestE2E_Credits_WASMBundleHidesButtonWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	wasmPath := writeFakeWasm(t, dir)
	execPath := writeFakeWasmExec(t, dir)
	outPath := filepath.Join(dir, "game.html")

	require.NoError(t, buildpipeline.BundleWASM(wasmPath, execPath, "Game", "", outPath, nil))

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	html := string(data)
	assert.NotContains(t, html, `id="credits-open"`,
		"empty credits suppresses the splash button entirely")
	assert.NotContains(t, html, `id="credits"`,
		"empty credits suppresses the credits div entirely")
}

// TestE2E_Credits_GeneratedCapsuleParses asserts the generated
// capsule.go is syntactically valid Go (gofmt-able). Catches
// template typos in the credits literal early.
func TestE2E_Credits_GeneratedCapsuleParses(t *testing.T) {
	credits := []capsuleruntime.CreditEntry{
		{Name: "with-quote\"x", License: "CC-BY-4.0", Author: "needs \\ escape", SourceURL: "https://x"},
	}
	p := pixelforge_project.NewProject("escape_test")
	outDir := t.TempDir()
	_, err := codegen.Generate(p, outDir, devReplaceOpts(t, credits))
	require.NoError(t, err, "tricky credit string contents must round-trip cleanly through Go template + gofmt")

	// Confirm the generated source contains the expected escapes.
	capsuleSrc, err := os.ReadFile(filepath.Join(outDir, "capsule.go"))
	require.NoError(t, err)
	// Both characters survive — %q-formatted strings escape correctly.
	assert.True(t, bytes.Contains(capsuleSrc, []byte(`"with-quote\"x"`)) ||
		bytes.Contains(capsuleSrc, []byte(`\"x`)),
		"quotes inside credit names must escape: %s",
		string(extractCapsuleCreditsBlock(capsuleSrc)))
}

func extractCapsuleCreditsBlock(src []byte) []byte {
	const marker = "capsuleCredits"
	idx := bytes.Index(src, []byte(marker))
	if idx < 0 {
		return nil
	}
	end := bytes.Index(src[idx:], []byte("\n}"))
	if end < 0 {
		return src[idx:]
	}
	return src[idx : idx+end+2]
}

// writeFakeWasm / writeFakeWasmExec mirror the helpers in
// wasm_bundler_test.go but live here so credits_e2e_test.go is
// self-contained.
func writeFakeWasm(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "game.wasm")
	require.NoError(t, os.WriteFile(p, []byte("\x00asm"), 0o644))
	return p
}

func writeFakeWasmExec(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "wasm_exec.js")
	require.NoError(t, os.WriteFile(p, []byte("function Go() {}"), 0o644))
	return p
}

// ensure imports used.
var _ = context.Background
var _ = strings.Contains
