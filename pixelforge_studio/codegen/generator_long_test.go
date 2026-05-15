//go:build long

package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/modulepath"
)

// Run with `go test -tags=long ./pixelforge_studio/codegen/...` (slow).
// This is the test that proves R8: generated games actually build on a
// fresh machine without the hardcoded /home/tux/... path.
func TestGenerate_BuildExportedBinary(t *testing.T) {
	// Resolve the real engine source the test is being run from.
	enginePath := findRepoRoot(t)
	outDir := t.TempDir()

	p := pixelforge_project.NewProject("integration")
	_, err := Generate(p, outDir, Options{
		ModulePath:   "example.com/integration",
		Strategy:     modulepath.StrategyDevReplace,
		EnginePath:   enginePath,
		RunGoModTidy: true,
	})
	require.NoError(t, err)

	cmd := exec.Command("go", "build", "-o", filepath.Join(outDir, "exported-game"), ".")
	cmd.Dir = outDir
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "go build of generated game failed:\n%s", string(output))

	_, err = os.Stat(filepath.Join(outDir, "exported-game"))
	require.NoError(t, err, "exported binary missing")
}

// findRepoRoot walks up from the test binary's working directory looking
// for the repo's go.mod. The standard pattern: cmd-line `go test`
// invokes the test from the package directory, which is several levels
// inside the repo.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for i := 0; i < 10; i++ {
		gomod := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(gomod)
		if err == nil {
			for _, line := range splitLines(string(data)) {
				if line == "module github.com/ibilalkhan1/fyp_pixelforge" {
					return dir
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not locate engine repo root")
	return ""
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, trimCR(s[start:i]))
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, trimCR(s[start:]))
	}
	return out
}

func trimCR(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\r' {
		return s[:len(s)-1]
	}
	return s
}
