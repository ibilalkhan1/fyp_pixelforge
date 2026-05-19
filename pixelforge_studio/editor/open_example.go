// Plan-009 U21 — File → Open Example menu. This file owns the
// editor-side seam for fetching reference-game `.pforge` files
// from the asset library's manifest, caching them on disk, and
// loading them into the editor.
//
// The seam is the OpenExampleSource interface (a minimal Library
// contract): the editor package never imports assetlibrary
// directly so tests can drive the menu via a stub source and
// the production wiring lives in main.go via SetAssetLibrary +
// the adapter in cmd/. Behaviour mirrors the FilePicker path —
// dirty-check, load via pixelforge_project, push onto recents,
// status-message echo.
//
// Two click handlers per example row:
//
//   - OpenExample(id) loads the cached/downloaded `.pforge`
//     directly. Designers run/edit the example in place; the
//     editor's currentProjectPath points at the cache file so
//     accidental Ctrl+S writes back to the cache (the user can
//     File → Save As to persist elsewhere).
//   - ForkExample(id, projectsDir) copies the cached `.pforge`
//     into <projectsDir>/<id>_fork_<timestamp>/ and opens the
//     copy. The forked project opens with dirty=false (the file
//     is already on disk at the fork path); designers immediately
//     edit + Ctrl+S writes to the fork. Honours the doc-review
//     "fork is a safe-auto starting point" finding.
package editor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// OpenExampleEntry is one row's worth of menu metadata. Mirrors
// the assetlibrary.Example fields the menu actually needs;
// duplicated here so the editor package doesn't import
// assetlibrary (and so tests can construct entries inline).
type OpenExampleEntry struct {
	ID          string
	Title       string
	Label       string
	Description string
	Game        string
	SizeBytes   int64
}

// MenuLabel returns the user-facing menu-row label — Label when
// set, falling back to Title, falling back to ID. Plan-009 U20
// established the Label override convention.
func (e OpenExampleEntry) MenuLabel() string {
	if e.Label != "" {
		return e.Label
	}
	if e.Title != "" {
		return e.Title
	}
	return e.ID
}

// OpenExampleResult is the outcome of FetchExample as the editor
// menu sees it — local path on success, error on failure. The
// minimal shape lets tests inject a stub source without depending
// on assetlibrary's full event vocabulary.
type OpenExampleResult struct {
	LocalPath string
	Err       error
}

// OpenExampleSource is the seam the editor's File → Open Example
// menu pulls from. Plan-009 U21's main.go wires this to an
// adapter over *assetlibrary.Library; tests inject a stub.
//
// Methods are all synchronous from the editor's perspective —
// callers wrap Fetch in a goroutine when they need non-blocking
// behaviour. Examples() must be cheap (called every frame).
type OpenExampleSource interface {
	// Examples returns the currently-known reference-game
	// entries, sorted for stable menu rendering. Returns an
	// empty slice when the manifest hasn't loaded yet — the
	// menu then renders a disabled "Loading examples…" row.
	Examples() []OpenExampleEntry

	// Fetch resolves an example id to a local .pforge path.
	// Cache-hit returns immediately; cache-miss downloads +
	// verifies + writes to disk. Errors surface to the caller
	// via OpenExampleResult.Err so the menu can render a red
	// toast.
	Fetch(ctx context.Context, exampleID string) OpenExampleResult
}

// SetOpenExampleSource installs the source the File → Open
// Example menu consults each frame. nil resets — the menu then
// renders the loading-state row. Plan-009 U21's main.go calls
// this with an *assetLibraryAdapter immediately after editor
// construction so the menu lights up the moment the manifest
// lands (or stays "Loading…" until the background bootstrap
// completes).
func (e *Editor) SetOpenExampleSource(src OpenExampleSource) {
	if e == nil {
		return
	}
	e.openExampleSource = src
}

// OpenExampleSource returns the currently-installed source, or
// nil. Exposed for diagnostics + integration testing.
func (e *Editor) OpenExampleSource() OpenExampleSource {
	if e == nil {
		return nil
	}
	return e.openExampleSource
}

// OpenExampleMenu builds the items the File → Open Example
// submenu renders this frame. Three behaviours:
//
//   - No source installed (e.g. --no-asset-library): one
//     disabled item, "No examples available".
//   - Source installed but manifest empty (pre-bootstrap or
//     genuinely empty manifest): one disabled item, "Loading
//     examples…". The design-lens doc-review finding asked
//     for this loading-state row so the menu doesn't look
//     broken on first launch.
//   - Source has examples: one parent item per example with
//     an "Open" + "Fork into new project" pair under it (Submenu
//     populated). The OnSelect on the parent opens directly;
//     Fork lives in the Submenu so it doesn't crowd the row.
//
// Returns a flat slice when there's a single placeholder row;
// returns a slice with Submenu-bearing items in the populated
// case. The chrome renderer handles both shapes.
func (e *Editor) OpenExampleMenu() []widgets.MenuItem {
	if e == nil || e.openExampleSource == nil {
		return []widgets.MenuItem{{
			Label:    "No examples available",
			Disabled: true,
		}}
	}
	entries := e.openExampleSource.Examples()
	if len(entries) == 0 {
		return []widgets.MenuItem{{
			Label:    "Loading examples…",
			Disabled: true,
		}}
	}
	items := make([]widgets.MenuItem, 0, len(entries))
	for _, entry := range entries {
		entry := entry
		label := entry.MenuLabel()
		items = append(items, widgets.MenuItem{
			Label: label,
			OnSelect: func() {
				e.fileMenu.OpenExampleByID(entry.ID, entry.MenuLabel())
			},
			Submenu: []widgets.MenuItem{
				{
					Label: "Open",
					OnSelect: func() {
						e.fileMenu.OpenExampleByID(entry.ID, entry.MenuLabel())
					},
				},
				{
					Label: "Fork into new project",
					OnSelect: func() {
						e.fileMenu.ForkExampleByID(entry.ID, entry.MenuLabel())
					},
				},
			},
		})
	}
	return items
}

// OpenExampleByID drives the Fetch → load pipeline for example id.
// Surfaces a "Downloading…" status before the (blocking) Fetch,
// then loads the resolved local path via the existing OpenPath
// flow so recent-projects + load-error handling stays unified.
//
// Errors surface as a red-toned status message; the menu row
// remains enabled for retry (no per-row disable state — the next
// frame's OpenExampleMenu rebuilds from the same entries).
func (m *FileMenu) OpenExampleByID(id, displayLabel string) {
	if m == nil || m.editor == nil {
		return
	}
	m.editor.PromptIfDirty("Discard changes?", "The current project has unsaved changes.", func() {
		src := m.editor.OpenExampleSource()
		if src == nil {
			m.editor.SetStatusMessage("open example: no asset library attached")
			return
		}
		if displayLabel == "" {
			displayLabel = id
		}
		m.editor.SetStatusMessage("Downloading " + displayLabel + "…")
		result := src.Fetch(context.Background(), id)
		if result.Err != nil {
			m.editor.SetStatusMessage(fmt.Sprintf("Failed to fetch %s: %v", displayLabel, result.Err))
			return
		}
		if err := m.OpenPath(result.LocalPath); err != nil {
			// OpenPath already wrote its own status message; we
			// prepend the example display label so the user can
			// see which example failed.
			m.editor.SetStatusMessage(fmt.Sprintf("Failed to open %s: %v", displayLabel, err))
			return
		}
		m.editor.SetStatusMessage("Opened example: " + displayLabel)
	})
}

// ForkExampleByID resolves the example to a local path, copies it
// into a fresh fork directory under the user's projects dir, and
// opens the copy in the editor. The fork opens with dirty=false
// (the file is already on disk at the fork path); designers
// immediately edit + Ctrl+S writes to the fork rather than the
// cache.
//
// Fork dir layout: <projectsDir>/<id>_fork_<unix-ts>/<id>.pforge.
// The timestamp suffix lets a designer fork the same example
// multiple times in one session without collision.
func (m *FileMenu) ForkExampleByID(id, displayLabel string) {
	if m == nil || m.editor == nil {
		return
	}
	if displayLabel == "" {
		displayLabel = id
	}
	m.editor.PromptIfDirty("Discard changes?", "The current project has unsaved changes.", func() {
		src := m.editor.OpenExampleSource()
		if src == nil {
			m.editor.SetStatusMessage("fork example: no asset library attached")
			return
		}
		m.editor.SetStatusMessage("Downloading " + displayLabel + " (fork)…")
		result := src.Fetch(context.Background(), id)
		if result.Err != nil {
			m.editor.SetStatusMessage(fmt.Sprintf("Failed to fetch %s: %v", displayLabel, result.Err))
			return
		}
		projectsDir, err := m.resolveProjectsDir()
		if err != nil {
			m.editor.SetStatusMessage(fmt.Sprintf("Failed to fork %s: %v", displayLabel, err))
			return
		}
		forkPath, err := copyExampleAsFork(result.LocalPath, id, projectsDir)
		if err != nil {
			m.editor.SetStatusMessage(fmt.Sprintf("Failed to fork %s: %v", displayLabel, err))
			return
		}
		if err := m.OpenPath(forkPath); err != nil {
			m.editor.SetStatusMessage(fmt.Sprintf("Failed to open fork of %s: %v", displayLabel, err))
			return
		}
		// OpenPath already cleared dirty via SetProject. Belt + braces.
		m.editor.ClearDirty()
		m.editor.SetStatusMessage("Forked " + displayLabel + " → " + filepath.Base(filepath.Dir(forkPath)))
	})
}

// resolveProjectsDir returns the directory new forks land under.
// Honours the same Documents/pixelforge convention the file
// picker's openStartPath uses. Tests inject via forkProjectsDirOverride.
func (m *FileMenu) resolveProjectsDir() (string, error) {
	if forkProjectsDirOverride != "" {
		return forkProjectsDirOverride, nil
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, "Documents", "pixelforge")
		if err := os.MkdirAll(candidate, 0o755); err == nil {
			return candidate, nil
		}
		// Fallback: home itself.
		return home, nil
	}
	return os.Getwd()
}

// forkProjectsDirOverride is the test hook for forks. Set via
// SetForkProjectsDirOverride; cleared with empty string. Held in
// a sync.Mutex so the test cleanup path doesn't race a parallel
// run.
var (
	forkProjectsDirOverrideMu sync.Mutex
	forkProjectsDirOverride   string
)

// SetForkProjectsDirOverride pins the directory ForkExampleByID
// writes into. Empty string clears the override. Tests use this
// to keep fork writes inside t.TempDir().
func SetForkProjectsDirOverride(dir string) {
	forkProjectsDirOverrideMu.Lock()
	defer forkProjectsDirOverrideMu.Unlock()
	forkProjectsDirOverride = dir
}

// copyExampleAsFork copies srcPath into <projectsDir>/<id>_fork_<ts>/<id>.pforge.
// The new directory is created fresh; existing files at the
// destination are not overwritten (the timestamp suffix makes
// collisions effectively impossible).
//
// If the example has a sibling `*-assets/` directory on disk, it
// is copied alongside so the forked project's asset-validation
// pass succeeds. For the v2 reference games the cached `.pforge`
// is self-contained (no sibling assets) — the copy is a no-op
// in that case, kept for forward-compat.
func copyExampleAsFork(srcPath, id, projectsDir string) (string, error) {
	ts := time.Now().Unix()
	forkDir := filepath.Join(projectsDir, fmt.Sprintf("%s_fork_%d", id, ts))
	if err := os.MkdirAll(forkDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir fork dir: %w", err)
	}
	dstPath := filepath.Join(forkDir, id+".pforge")
	if err := copyFile(srcPath, dstPath); err != nil {
		return "", fmt.Errorf("copy pforge: %w", err)
	}

	// Best-effort: copy the sibling assets dir if one exists.
	// The cached examples in the v2 manifest don't ship sibling
	// assets, but the contract leaves room for them.
	srcAssets := pixelforge_project.AssetsDir(srcPath)
	if info, err := os.Stat(srcAssets); err == nil && info.IsDir() {
		dstAssets := pixelforge_project.AssetsDir(dstPath)
		if err := copyDirRecursive(srcAssets, dstAssets); err != nil {
			return "", fmt.Errorf("copy assets: %w", err)
		}
	}

	return dstPath, nil
}

// copyFile reads src and writes its bytes to dst. The destination
// directory must already exist. Errors are wrapped with the src/
// dst paths for diagnosis.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// copyDirRecursive copies the tree rooted at src into dst.
// Regular files + directories are honoured; symlinks + devices
// are skipped to mirror the assetlibrary unpacker's defensive
// posture.
func copyDirRecursive(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcChild := filepath.Join(src, entry.Name())
		dstChild := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDirRecursive(srcChild, dstChild); err != nil {
				return err
			}
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeType != 0 {
			// Skip symlinks / devices / sockets.
			continue
		}
		if err := copyFile(srcChild, dstChild); err != nil {
			return err
		}
	}
	return nil
}

// errNoExampleSource is the sentinel surfaced when a click
// arrives before main.go has wired the asset library. Currently
// only used in the menu's status message path; exported so
// integration tests can errors.Is-detect it if needed.
var errNoExampleSource = errors.New("editor: no OpenExampleSource attached")
