package editor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// stubExampleSource is the in-memory test double for the
// OpenExampleSource interface. Each test constructs one with the
// shape it needs — entries to advertise, the canned Fetch result,
// and a hit counter so tests assert Fetch was/wasn't invoked.
type stubExampleSource struct {
	mu       sync.Mutex
	entries  []OpenExampleEntry
	results  map[string]OpenExampleResult
	hits     map[string]int
	fetchErr error
}

func newStubSource(entries []OpenExampleEntry) *stubExampleSource {
	return &stubExampleSource{
		entries: entries,
		results: map[string]OpenExampleResult{},
		hits:    map[string]int{},
	}
}

func (s *stubExampleSource) Examples() []OpenExampleEntry { return s.entries }

func (s *stubExampleSource) Fetch(_ context.Context, id string) OpenExampleResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hits[id]++
	if s.fetchErr != nil {
		return OpenExampleResult{Err: s.fetchErr}
	}
	r, ok := s.results[id]
	if !ok {
		return OpenExampleResult{Err: errors.New("no canned result for " + id)}
	}
	return r
}

func (s *stubExampleSource) hitCount(id string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits[id]
}

// writeValidPforge stamps a minimal `.pforge` file at path that
// pixelforge_project.Load accepts. Returns the same path for
// convenience.
func writeValidPforge(t *testing.T, path string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	p := pixelforge_project.NewProject("example_proof")
	require.NoError(t, p.Save(path))
	return path
}

func TestOpenExampleMenu_NoSourceShowsDisabledRow(t *testing.T) {
	e := New()
	items := e.OpenExampleMenu()
	require.Len(t, items, 1)
	assert.True(t, items[0].Disabled)
	assert.Contains(t, items[0].Label, "No examples")
}

func TestOpenExampleMenu_EmptyManifestShowsLoadingState(t *testing.T) {
	e := New()
	src := newStubSource(nil)
	e.SetOpenExampleSource(src)
	items := e.OpenExampleMenu()
	require.Len(t, items, 1)
	assert.True(t, items[0].Disabled, "loading-state row must be disabled")
	assert.Contains(t, items[0].Label, "Loading")
}

func TestOpenExampleMenu_PopulatesFromSource(t *testing.T) {
	e := New()
	src := newStubSource([]OpenExampleEntry{
		{ID: "asteroids_proof", Title: "Asteroids — Reference", Label: "Asteroids (reference build)"},
		{ID: "mario_proof", Title: "Mario — Reference"},
		{ID: "bomberman_proof"},
		{ID: "donkey_kong_proof", Title: "DK"},
	})
	e.SetOpenExampleSource(src)

	items := e.OpenExampleMenu()
	require.Len(t, items, 4, "one menu row per example entry")

	// MenuLabel order: Label > Title > ID. Quick check on the
	// first row whose Label override should appear in the menu.
	assert.Equal(t, "Asteroids (reference build)", items[0].Label)
	// Each row carries a Submenu with Open + Fork.
	require.Len(t, items[0].Submenu, 2)
	assert.Equal(t, "Open", items[0].Submenu[0].Label)
	assert.Equal(t, "Fork into new project", items[0].Submenu[1].Label)
}

func TestOpenExample_CacheHitOpensProject(t *testing.T) {
	dir := t.TempDir()
	cachePath := writeValidPforge(t, filepath.Join(dir, "asteroids_proof.pforge"))

	e := New()
	src := newStubSource([]OpenExampleEntry{
		{ID: "asteroids_proof", Title: "Asteroids"},
	})
	src.results["asteroids_proof"] = OpenExampleResult{LocalPath: cachePath}
	e.SetOpenExampleSource(src)

	items := e.OpenExampleMenu()
	require.Len(t, items, 1)
	require.NotNil(t, items[0].Submenu[0].OnSelect)
	items[0].Submenu[0].OnSelect()

	assert.Equal(t, 1, src.hitCount("asteroids_proof"))
	assert.Equal(t, cachePath, e.CurrentProjectPath())
	assert.Equal(t, "example_proof", e.Project().Name)
	assert.Contains(t, e.StatusMessage(), "Opened example")
	assert.False(t, e.IsDirty())
}

func TestOpenExample_NetworkFailureSurfacesStatus(t *testing.T) {
	e := New()
	src := newStubSource([]OpenExampleEntry{{ID: "down_proof", Title: "Down"}})
	src.fetchErr = errors.New("connection refused")
	e.SetOpenExampleSource(src)

	items := e.OpenExampleMenu()
	items[0].Submenu[0].OnSelect()

	assert.Contains(t, e.StatusMessage(), "Failed to fetch")
	assert.Contains(t, e.StatusMessage(), "connection refused")
	// Menu row is rebuilt each frame so retry stays available
	// — verify the same source still advertises the entry.
	rebuilt := e.OpenExampleMenu()
	assert.Len(t, rebuilt, 1)
	assert.False(t, rebuilt[0].Disabled, "menu remains enabled for retry after failure")
}

func TestOpenExample_DirtyProjectGetsConfirmModal(t *testing.T) {
	dir := t.TempDir()
	cachePath := writeValidPforge(t, filepath.Join(dir, "asteroids_proof.pforge"))

	e := New()
	e.MarkDirty()
	src := newStubSource([]OpenExampleEntry{{ID: "asteroids_proof", Title: "Asteroids"}})
	src.results["asteroids_proof"] = OpenExampleResult{LocalPath: cachePath}
	e.SetOpenExampleSource(src)

	items := e.OpenExampleMenu()
	items[0].Submenu[0].OnSelect()

	// Confirm dialog should be visible (dirty-check guards the
	// destructive open); the actual load runs only after Confirm.
	assert.True(t, e.confirmDialog.Visible())
	assert.Equal(t, 0, src.hitCount("asteroids_proof"), "fetch deferred until confirm")
	e.confirmDialog.Confirm()
	assert.Equal(t, 1, src.hitCount("asteroids_proof"), "fetch runs after confirm")
	assert.Equal(t, cachePath, e.CurrentProjectPath())
}

func TestForkExample_CopiesAndOpensCleanProject(t *testing.T) {
	cacheDir := t.TempDir()
	cachePath := writeValidPforge(t, filepath.Join(cacheDir, "asteroids_proof.pforge"))

	projectsDir := t.TempDir()
	SetForkProjectsDirOverride(projectsDir)
	t.Cleanup(func() { SetForkProjectsDirOverride("") })

	e := New()
	src := newStubSource([]OpenExampleEntry{{ID: "asteroids_proof", Title: "Asteroids"}})
	src.results["asteroids_proof"] = OpenExampleResult{LocalPath: cachePath}
	e.SetOpenExampleSource(src)

	items := e.OpenExampleMenu()
	// Submenu[1] is the Fork action.
	items[0].Submenu[1].OnSelect()

	assert.Equal(t, 1, src.hitCount("asteroids_proof"))
	assert.False(t, e.IsDirty(), "fork opens with dirty=false per design")
	assert.NotEqual(t, cachePath, e.CurrentProjectPath(), "current path points at the fork, not the cache")

	// Fork lives under projectsDir/asteroids_proof_fork_<ts>/asteroids_proof.pforge.
	rel, err := filepath.Rel(projectsDir, e.CurrentProjectPath())
	require.NoError(t, err)
	parts := strings.Split(rel, string(filepath.Separator))
	require.Len(t, parts, 2)
	assert.True(t, strings.HasPrefix(parts[0], "asteroids_proof_fork_"), "fork dir uses example_id_fork_<ts> convention; got %q", parts[0])
	assert.Equal(t, "asteroids_proof.pforge", parts[1])

	// The forked file actually exists on disk and is loadable.
	info, err := os.Stat(e.CurrentProjectPath())
	require.NoError(t, err)
	assert.Equal(t, "asteroids_proof.pforge", info.Name())

	assert.Contains(t, e.StatusMessage(), "Forked")
}

func TestForkExample_FetchFailureSurfacesStatus(t *testing.T) {
	projectsDir := t.TempDir()
	SetForkProjectsDirOverride(projectsDir)
	t.Cleanup(func() { SetForkProjectsDirOverride("") })

	e := New()
	src := newStubSource([]OpenExampleEntry{{ID: "down_proof", Title: "Down"}})
	src.fetchErr = errors.New("network down")
	e.SetOpenExampleSource(src)

	items := e.OpenExampleMenu()
	items[0].Submenu[1].OnSelect()

	assert.Contains(t, e.StatusMessage(), "Failed to fetch")
	// No fork directory created when fetch failed.
	entries, _ := os.ReadDir(projectsDir)
	assert.Empty(t, entries, "no fork dir created on fetch failure")
}

func TestOpenExampleMenu_BuildMenuDefsIncludesSubmenu(t *testing.T) {
	// Sanity check the File menu integration — File → Open Example
	// is present and exposes the OpenExampleMenu output as its
	// Submenu. Without a source attached, the submenu shows the
	// disabled "No examples available" row.
	e := New()
	defs := e.buildMenuDefs()
	var openExample *struct{ items []interface{} }
	_ = openExample // satisfy linter — actual capture is below
	var foundLabel string
	for _, def := range defs {
		if def.Label != "File" {
			continue
		}
		for _, item := range def.Items {
			if item.Label == "Open Example" {
				foundLabel = item.Label
				require.Len(t, item.Submenu, 1, "no source attached → single placeholder row")
				assert.True(t, item.Submenu[0].Disabled)
			}
		}
	}
	require.Equal(t, "Open Example", foundLabel, "File menu must include the Open Example submenu")
}

func TestOpenExampleMenu_BuildMenuDefsLightsUpWithSource(t *testing.T) {
	e := New()
	src := newStubSource([]OpenExampleEntry{
		{ID: "asteroids_proof", Title: "Asteroids"},
		{ID: "mario_proof", Title: "Mario"},
	})
	e.SetOpenExampleSource(src)

	defs := e.buildMenuDefs()
	var submenuLen int
	for _, def := range defs {
		if def.Label != "File" {
			continue
		}
		for _, item := range def.Items {
			if item.Label == "Open Example" {
				submenuLen = len(item.Submenu)
			}
		}
	}
	assert.Equal(t, 2, submenuLen, "submenu must rebuild with two example rows once source is attached")
}
