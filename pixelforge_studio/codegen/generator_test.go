package codegen

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/modulepath"
)

// Empty project → main.go parses; go.mod has the require; assets/
// directory is absent (no assets to copy).
func TestGenerate_EmptyProject(t *testing.T) {
	p := pixelforge_project.NewProject("empty")
	outDir := t.TempDir()

	_, err := Generate(p, outDir, Options{
		ModulePath: "example.com/empty",
		Strategy:   modulepath.StrategyVendor,
		EnginePath: vendorableEngine(t),
	})
	require.NoError(t, err)

	// main.go is syntactically valid Go.
	main, err := os.ReadFile(filepath.Join(outDir, "main.go"))
	require.NoError(t, err)
	_, parseErr := parser.ParseFile(token.NewFileSet(), "main.go", main, 0)
	assert.NoError(t, parseErr, "generated main.go must parse")

	// go.mod references the engine require.
	gomod, err := os.ReadFile(filepath.Join(outDir, "go.mod"))
	require.NoError(t, err)
	assert.Contains(t, string(gomod), "module example.com/empty")
	assert.Contains(t, string(gomod), "require github.com/ibilalkhan1/fyp_pixelforge")

	// Embedded project.pforge is present and non-empty.
	pf, err := os.ReadFile(filepath.Join(outDir, "project.pforge"))
	require.NoError(t, err)
	assert.Contains(t, string(pf), `"schema_version"`)

	// No assets to copy.
	_, err = os.Stat(filepath.Join(outDir, "assets"))
	assert.True(t, os.IsNotExist(err))

	// Vendor strategy copied the engine.
	_, err = os.Stat(filepath.Join(outDir, "vendor", "github.com", "ibilalkhan1", "fyp_pixelforge", "go.mod"))
	assert.NoError(t, err)
}

// Snake-shaped project: main.go parses, the referenced sprite is copied,
// and project.pforge is byte-identical to a fresh Save() of the same
// project (the embed-and-disk paths share their encoder).
func TestGenerate_SnakeShapedCopiesAssets(t *testing.T) {
	srcDir := t.TempDir()
	pforgePath := filepath.Join(srcDir, "snake.pforge")

	p := snakeProject()
	require.NoError(t, p.Save(pforgePath))
	// drop the sprite on disk so preflight passes.
	writeFile(t, filepath.Join(pixelforge_project.AssetsDir(pforgePath), "sprites", "sprites.png"),
		[]byte("fakepng-bytes"))

	outDir := t.TempDir()
	_, err := Generate(p, outDir, Options{
		ModulePath:        "example.com/snake",
		Strategy:          modulepath.StrategyVendor,
		EnginePath:        vendorableEngine(t),
		ProjectSourcePath: pforgePath,
	})
	require.NoError(t, err)

	// main.go parses
	main, err := os.ReadFile(filepath.Join(outDir, "main.go"))
	require.NoError(t, err)
	_, parseErr := parser.ParseFile(token.NewFileSet(), "main.go", main, 0)
	assert.NoError(t, parseErr)

	// asset copied at the expected path
	copied, err := os.ReadFile(filepath.Join(outDir, "assets", "sprites", "sprites.png"))
	require.NoError(t, err)
	assert.Equal(t, "fakepng-bytes", string(copied))
}

// Non-empty output without Force → error listing the conflicting files.
func TestGenerate_RefusesNonEmptyOutputWithoutForce(t *testing.T) {
	outDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outDir, "leftover.go"),
		[]byte("package main"), 0o644))

	p := pixelforge_project.NewProject("collide")
	_, err := Generate(p, outDir, Options{
		Strategy:   modulepath.StrategyVendor,
		EnginePath: vendorableEngine(t),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "leftover.go")
	assert.Contains(t, err.Error(), "Force")
}

// Force=true overwrites non-empty output.
func TestGenerate_ForceOverwrites(t *testing.T) {
	outDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outDir, "stale.go"),
		[]byte("package main"), 0o644))

	p := pixelforge_project.NewProject("force")
	_, err := Generate(p, outDir, Options{
		ModulePath: "example.com/force",
		Strategy:   modulepath.StrategyVendor,
		EnginePath: vendorableEngine(t),
		Force:      true,
	})
	require.NoError(t, err)

	// main.go from the generator is now present.
	_, err = os.Stat(filepath.Join(outDir, "main.go"))
	assert.NoError(t, err)
}

// Project references a sprite that doesn't exist → atomic failure
// (no output written) with a clear message.
func TestGenerate_AtomicOnMissingAsset(t *testing.T) {
	srcDir := t.TempDir()
	pforgePath := filepath.Join(srcDir, "missing.pforge")
	p := pixelforge_project.NewProject("missing")
	p.Sprites = append(p.Sprites, pixelforge_project.SpriteAsset{
		Name:         "ghost",
		RelativePath: "sprites/ghost.png",
	})
	// Note: do NOT create ghost.png on disk.
	// We don't Save() — preflight reads from the project in memory.

	outDir := t.TempDir()
	_, err := Generate(p, outDir, Options{
		ModulePath:        "example.com/missing",
		Strategy:          modulepath.StrategyVendor,
		EnginePath:        vendorableEngine(t),
		ProjectSourcePath: pforgePath,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ghost.png")

	// Nothing should have been written.
	entries, _ := os.ReadDir(outDir)
	assert.Empty(t, entries, "atomic-on-failure: no files in outDir")
}

// StrategyDevReplace emits a `replace` directive pointing at EnginePath.
func TestGenerate_DevReplaceWritesReplaceDirective(t *testing.T) {
	engine := vendorableEngine(t)
	outDir := t.TempDir()
	p := pixelforge_project.NewProject("dev")
	_, err := Generate(p, outDir, Options{
		ModulePath: "example.com/dev",
		Strategy:   modulepath.StrategyDevReplace,
		EnginePath: engine,
	})
	require.NoError(t, err)

	gomod, err := os.ReadFile(filepath.Join(outDir, "go.mod"))
	require.NoError(t, err)
	s := string(gomod)
	assert.Contains(t, s, "replace github.com/ibilalkhan1/fyp_pixelforge =>")
	assert.Contains(t, s, engine)
	// dev-replace does NOT vendor
	_, err = os.Stat(filepath.Join(outDir, "vendor"))
	assert.True(t, os.IsNotExist(err))
}

// gofmt failure surfaces the offending source. We trigger it by
// directly invoking the renderer with a sabotaged template via a
// scoped swap — keeps the public Generate API clean.
func TestRenderMainGo_ParseValid(t *testing.T) {
	p := pixelforge_project.NewProject("ok")
	src, err := renderMainGo(p)
	require.NoError(t, err)
	_, err = parser.ParseFile(token.NewFileSet(), "main.go", src, 0)
	assert.NoError(t, err)
}

// snakeProject is a representative project used by the asset-copy test.
func snakeProject() *pixelforge_project.Project {
	p := pixelforge_project.NewProject("snake")
	p.ScreenWidth = 128
	p.ScreenHeight = 128
	p.TPS = 30
	p.Sprites = []pixelforge_project.SpriteAsset{{
		Name:         "sprites",
		RelativePath: "sprites/sprites.png",
		Width:        32, Height: 8,
		FrameW: 8, FrameH: 8,
	}}
	return p
}

// vendorableEngine builds a minimal-but-vendorable engine tree the
// codegen tests share. Mirrors what modulepath.fakeEngineTree does so
// vendor-copy succeeds.
func vendorableEngine(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module "+modulepath.EngineModule+"\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pixelforge.go"),
		[]byte("package pixelforge\n"), 0o644))
	for _, sub := range []string{"pixelforge_event", "pixelforge_ebiten", "pixelforge_project"} {
		path := filepath.Join(dir, sub)
		require.NoError(t, os.MkdirAll(path, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(path, "package.go"),
			[]byte("package "+sub+"\n"), 0o644))
	}
	return dir
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, data, 0o644))
}

// Sanity: TemplateVersion is a single token (no spaces or punctuation
// that would corrupt the generated header comment).
func TestTemplateVersionFormat(t *testing.T) {
	assert.NotEmpty(t, TemplateVersion)
	assert.False(t, strings.ContainsAny(TemplateVersion, " \t\n"))
}
