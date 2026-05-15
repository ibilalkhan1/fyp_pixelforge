package editor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Happy path: save then load returns equal struct.
func TestSettings_SaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")

	original := DefaultSettings()
	original.path = path
	original.WindowWidth = 1600
	original.WindowHeight = 900
	original.Theme = "high-contrast"
	original.PushRecentProject("/tmp/a.pforge")
	original.PushRecentProject("/tmp/b.pforge")

	require.NoError(t, original.Save())

	loaded := LoadSettingsAt(path)
	assert.Equal(t, 1600, loaded.WindowWidth)
	assert.Equal(t, 900, loaded.WindowHeight)
	assert.Equal(t, "high-contrast", loaded.Theme)
	assert.Equal(t, []string{"/tmp/b.pforge", "/tmp/a.pforge"}, loaded.RecentProjects)
}

// PushRecentProject prepends and dedupes; cap clamps to MaxRecentProjects.
func TestSettings_RecentProjectsDedupeAndCap(t *testing.T) {
	s := DefaultSettings()

	s.PushRecentProject("/tmp/a.pforge")
	assert.Equal(t, []string{"/tmp/a.pforge"}, s.RecentProjects)

	// Pushing 12 distinct paths leaves length capped at 10
	for i := 0; i < 12; i++ {
		s.PushRecentProject(filepath.Join("/tmp", "p"+string(rune('A'+i))+".pforge"))
	}
	assert.Len(t, s.RecentProjects, MaxRecentProjects)
	// Most recent path is first
	assert.Equal(t, "/tmp/pL.pforge", s.RecentProjects[0])

	// Re-pushing an existing path moves it to front, no duplicate
	s.PushRecentProject("/tmp/pC.pforge")
	assert.Equal(t, "/tmp/pC.pforge", s.RecentProjects[0])
	count := 0
	for _, p := range s.RecentProjects {
		if p == "/tmp/pC.pforge" {
			count++
		}
	}
	assert.Equal(t, 1, count, "no duplicate after re-push")
}

// Missing file returns defaults without error.
func TestSettings_LoadMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.json")
	s := LoadSettingsAt(path)
	assert.Equal(t, defaultWindowWidth, s.WindowWidth)
	assert.Equal(t, defaultWindowHeight, s.WindowHeight)
	assert.Equal(t, "dark", s.Theme)
	assert.Empty(t, s.RecentProjects)
}

// Malformed JSON returns defaults; does not crash.
func TestSettings_LoadMalformedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	require.NoError(t, os.WriteFile(path, []byte("{not valid"), 0o644))

	s := LoadSettingsAt(path)
	assert.Equal(t, defaultWindowWidth, s.WindowWidth)
	assert.Equal(t, defaultWindowHeight, s.WindowHeight)
}

// sanitize repairs zero / out-of-range fields loaded from old files.
func TestSettings_SanitizeRepairsBadValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
        "window_width": 0,
        "window_height": -1,
        "theme": "",
        "recent_projects": null
    }`), 0o644))

	s := LoadSettingsAt(path)
	assert.Equal(t, defaultWindowWidth, s.WindowWidth)
	assert.Equal(t, defaultWindowHeight, s.WindowHeight)
	assert.Equal(t, "dark", s.Theme)
	assert.NotNil(t, s.RecentProjects)
	assert.Empty(t, s.RecentProjects)
}

// Empty string is rejected by PushRecentProject (no entry added).
func TestSettings_PushRecentEmptyIsNoop(t *testing.T) {
	s := DefaultSettings()
	assert.False(t, s.PushRecentProject(""))
	assert.Empty(t, s.RecentProjects)
}
