package modulepath

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeEngineTree drops a minimal-but-recognisable engine layout under
// dir. Tests use it to exercise findEnginePath / Apply without
// requiring the real working tree.
func fakeEngineTree(t *testing.T, dir string, withGit bool) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module "+EngineModule+"\n\ngo 1.24\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pixelforge.go"),
		[]byte("package pixelforge\n"), 0o644))
	for _, sub := range []string{"pixelforge_event", "pixelforge_audio", "pixelforge_ebiten"} {
		path := filepath.Join(dir, sub)
		require.NoError(t, os.MkdirAll(path, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(path, "package.go"),
			[]byte("package "+sub+"\n"), 0o644))
		// drop a test file we should NOT vendor
		require.NoError(t, os.WriteFile(filepath.Join(path, "x_test.go"),
			[]byte("package "+sub+"\n"), 0o644))
	}
	if withGit {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	}
}

// A fake engine tree with .git → StrategyDevReplace.
func TestDetect_DevReplaceWhenGitCheckout(t *testing.T) {
	root := t.TempDir()
	engine := filepath.Join(root, "engine")
	fakeEngineTree(t, engine, true)

	// Pretend the editor binary lives one level deep inside the
	// engine — Detect walks up to find go.mod.
	binary := filepath.Join(engine, "pixelforge_studio", "studio")
	require.NoError(t, os.MkdirAll(filepath.Dir(binary), 0o755))
	require.NoError(t, os.WriteFile(binary, []byte{}, 0o644))

	d := Detect(binary)
	assert.Equal(t, StrategyDevReplace, d.Strategy)
	assert.Equal(t, engine, d.EnginePath)
}

// Same tree without .git → StrategyVendor.
func TestDetect_VendorWhenNotGitCheckout(t *testing.T) {
	root := t.TempDir()
	engine := filepath.Join(root, "engine")
	fakeEngineTree(t, engine, false)

	binary := filepath.Join(engine, "pixelforge_studio", "studio")
	require.NoError(t, os.MkdirAll(filepath.Dir(binary), 0o755))
	require.NoError(t, os.WriteFile(binary, []byte{}, 0o644))

	d := Detect(binary)
	assert.Equal(t, StrategyVendor, d.Strategy)
	assert.Equal(t, engine, d.EnginePath)
}

// No engine source found → StrategyVendor with empty EnginePath and
// an explanatory Reason. Apply on such a Detection must fail loudly.
func TestDetect_NoEnginePath(t *testing.T) {
	// A tempdir with no go.mod anywhere upward → Detect falls through.
	bogus := filepath.Join(t.TempDir(), "nowhere", "studio")
	d := Detect(bogus)
	assert.Equal(t, StrategyVendor, d.Strategy)
	assert.Equal(t, "", d.EnginePath)
	assert.NotEmpty(t, d.Reason)

	// Apply must reject this because it can't copy nothing.
	_, err := Apply(d, t.TempDir())
	assert.Error(t, err)
}

// Apply(StrategyVendor) copies the engine tree, skips test files and
// the studio package, and lands at vendor/<EngineModule>.
func TestApply_VendorCopiesEngineTree(t *testing.T) {
	root := t.TempDir()
	engine := filepath.Join(root, "engine")
	fakeEngineTree(t, engine, false)

	// Drop a pixelforge_studio dir we must NOT vendor.
	require.NoError(t, os.MkdirAll(filepath.Join(engine, "pixelforge_studio"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(engine, "pixelforge_studio", "main.go"),
		[]byte("package main\n"), 0o644))

	outDir := filepath.Join(root, "exported-game")
	d := Detection{Strategy: StrategyVendor, EnginePath: engine}
	dst, err := Apply(d, outDir)
	require.NoError(t, err)

	// vendor/<module>/go.mod exists
	_, err = os.Stat(filepath.Join(dst, "go.mod"))
	assert.NoError(t, err)
	// pixelforge.go (root engine file) was carried
	_, err = os.Stat(filepath.Join(dst, "pixelforge.go"))
	assert.NoError(t, err)
	// One of the engine subpackages was vendored
	_, err = os.Stat(filepath.Join(dst, "pixelforge_event", "package.go"))
	assert.NoError(t, err)
	// _test.go files were skipped
	_, err = os.Stat(filepath.Join(dst, "pixelforge_event", "x_test.go"))
	assert.True(t, os.IsNotExist(err))
	// pixelforge_studio was NOT vendored
	_, err = os.Stat(filepath.Join(dst, "pixelforge_studio"))
	assert.True(t, os.IsNotExist(err))
}

// StrategyDevReplace verifies the path is real; missing go.mod errors.
func TestApply_DevReplaceVerifiesPath(t *testing.T) {
	engine := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(engine, "go.mod"),
		[]byte("module "+EngineModule+"\n"), 0o644))

	d := Detection{Strategy: StrategyDevReplace, EnginePath: engine}
	dst, err := Apply(d, t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, dst, "dev-replace writes no files via Apply")

	// Missing go.mod → error
	missing := t.TempDir()
	_, err = Apply(Detection{Strategy: StrategyDevReplace, EnginePath: missing}, t.TempDir())
	assert.Error(t, err)
}

// StrategyPublishedVersion needs a Version to proceed.
func TestApply_PublishedVersionRequiresVersion(t *testing.T) {
	_, err := Apply(Detection{Strategy: StrategyPublishedVersion}, t.TempDir())
	assert.Error(t, err)

	dst, err := Apply(Detection{Strategy: StrategyPublishedVersion, Version: "v0.1.0"}, t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, dst, "published-version writes no files via Apply")
}

// shouldVendorDir gatekeeps which top-level engine dirs are copied.
func TestShouldVendorDir(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"pixelforge_event", true},
		{"pixelforge_audio", true},
		{"pixelforge_ebiten", true},
		{"internal", true},
		{"pixelforge_pool", true},
		{"pixelforge_studio", false},
		{"pixelforge_examples", false},
		{"pixelforge_test_helpers", false},
		{"docs", false},
		{"mmd-diagrams", false},
		{"temp", false},
	}
	for _, c := range tests {
		assert.Equal(t, c.want, shouldVendorDir(c.name), "shouldVendorDir(%q)", c.name)
	}
}
