package capture

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	pirand "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRecorderForRegression(t *testing.T) (*Recorder, string, *pixelforge_project.Project) {
	t.Helper()
	pixelforge.SetScreenSize(16, 16)
	pirand.Seed(42)
	rec := New(8)
	rec.initialSeed = pirand.CurrentSeed()
	for i := 0; i < 5; i++ {
		pixelforge.SetColor(pixelforge.Color(i + 1))
		pixelforge.RectFill(0, 0, 15, 15)
		rec.RecordInput("key", "down:32")
		rec.RecordEvent("loop", "tick")
		rec.SaveFrame()
	}
	pforge, project := makeProjectWithSprite(t)
	return rec, pforge, project
}

func TestPromoteFrameToRegression_WritesLayout(t *testing.T) {
	rec, pforge, project := setupRecorderForRegression(t)
	root := t.TempDir()
	testDir, err := PromoteFrameToRegression(rec, 3, project, pforge, root, "snake-baseline")
	require.NoError(t, err)

	for _, want := range []string{"golden.png", "input.log", "events.log", "project.pforge", "seed.txt"} {
		_, err := os.Stat(filepath.Join(testDir, want))
		assert.NoError(t, err, "%s should exist", want)
	}
}

func TestPromoteFrameToRegression_OverwritesByName(t *testing.T) {
	rec, pforge, project := setupRecorderForRegression(t)
	root := t.TempDir()
	d1, err := PromoteFrameToRegression(rec, 3, project, pforge, root, "baseline")
	require.NoError(t, err)
	d2, err := PromoteFrameToRegression(rec, 4, project, pforge, root, "baseline")
	require.NoError(t, err)
	assert.Equal(t, d1, d2)
}

func TestPromoteFrameToRegression_OutOfRangeError(t *testing.T) {
	rec, pforge, project := setupRecorderForRegression(t)
	root := t.TempDir()
	_, err := PromoteFrameToRegression(rec, 99, project, pforge, root, "x")
	require.Error(t, err)
}

func TestPromoteFrameToRegression_SeedRoundTrip(t *testing.T) {
	rec, pforge, project := setupRecorderForRegression(t)
	root := t.TempDir()
	testDir, err := PromoteFrameToRegression(rec, 2, project, pforge, root, "x")
	require.NoError(t, err)
	seedBytes, err := os.ReadFile(filepath.Join(testDir, "seed.txt"))
	require.NoError(t, err)
	v, err := strconv.ParseUint(strings.TrimSpace(string(seedBytes)), 10, 64)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), v)
}

func TestPromoteFrameToRegression_InputLogContents(t *testing.T) {
	rec, pforge, project := setupRecorderForRegression(t)
	root := t.TempDir()
	testDir, err := PromoteFrameToRegression(rec, 4, project, pforge, root, "x")
	require.NoError(t, err)
	entries, err := readLogEntries(filepath.Join(testDir, "input.log"))
	require.NoError(t, err)
	// 5 frames × 1 input each.
	assert.Len(t, entries, 5)
	assert.Equal(t, "key", entries[0].Target)
}

func TestReplayRegression_HappyPath(t *testing.T) {
	rec, pforge, project := setupRecorderForRegression(t)
	root := t.TempDir()
	testDir, err := PromoteFrameToRegression(rec, 3, project, pforge, root, "x")
	require.NoError(t, err)
	// Apply frame 3 to screen so the pixel compare matches.
	ApplyFrameToScreen(rec, 3)
	result, err := ReplayRegression(testDir)
	require.NoError(t, err)
	assert.True(t, result.Passed, "detail: %s", result.Detail)
}

func TestReplayRegression_PixelDiffOnFailure(t *testing.T) {
	rec, pforge, project := setupRecorderForRegression(t)
	root := t.TempDir()
	testDir, err := PromoteFrameToRegression(rec, 3, project, pforge, root, "x")
	require.NoError(t, err)
	// Apply a *different* frame so screen mismatches the golden.
	ApplyFrameToScreen(rec, 0)
	result, err := ReplayRegression(testDir)
	require.NoError(t, err)
	assert.False(t, result.Passed)
	_, err = os.Stat(filepath.Join(testDir, "diff.png"))
	assert.NoError(t, err)
}

func TestReplayRegression_MissingDirErrors(t *testing.T) {
	_, err := ReplayRegression(filepath.Join(t.TempDir(), "does-not-exist"))
	assert.Error(t, err)
}

func TestReplayRegression_SeedRestored(t *testing.T) {
	rec, pforge, project := setupRecorderForRegression(t)
	root := t.TempDir()
	pirand.Seed(42)
	testDir, err := PromoteFrameToRegression(rec, 1, project, pforge, root, "x")
	require.NoError(t, err)
	// Knock pirand into a different seed.
	pirand.Seed(999)
	require.NotEqual(t, uint64(42), pirand.CurrentSeed())
	// Apply so the replay pixel check passes.
	ApplyFrameToScreen(rec, 1)
	_, err = ReplayRegression(testDir)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), pirand.CurrentSeed())
}
