package assetlibrary

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// CanonicalGameTabs is the ordered list of per-game tabs the
// workspace renders. Custom is the U11 user-library tab; All is
// the cross-pack view.
var CanonicalGameTabs = []string{
	"all", "asteroids", "bomberman", "mario", "donkey_kong", "custom",
}

// TabDisplayName returns the user-facing label for a tab id.
func TabDisplayName(id string) string {
	switch id {
	case "all":
		return "All"
	case "asteroids":
		return "Asteroids"
	case "bomberman":
		return "Bomberman"
	case "mario":
		return "Mario"
	case "donkey_kong":
		return "Donkey Kong"
	case "custom":
		return "Custom"
	}
	return id
}

// EditorBinding is the minimal contract the workspace needs from
// the studio editor — read the current project, set the dirty
// flag after adding an asset. Defined here so the assetlibrary
// package doesn't import pixelforge_studio/editor (avoids a
// cycle when U12 wires both).
type EditorBinding interface {
	Project() *pixelforge_project.Project
	CurrentProjectPath() string
	MarkDirty()
}

// Workspace browses installed packs + user-library assets. State
// methods (ActiveTab, SetActiveTab, AssetsForActiveTab,
// AddToProject) are imgui-free so tests exercise them without
// standing up an imgui frame. The Render method (in a build-tagged
// sibling once UI lands) wraps these calls in imgui chrome.
type Workspace struct {
	editor EditorBinding
	lib    *Library

	mu        sync.Mutex
	activeTab string
}

// NewWorkspace binds a workspace to editor + lib. Active tab
// defaults to "all".
func NewWorkspace(editor EditorBinding, lib *Library) *Workspace {
	return &Workspace{editor: editor, lib: lib, activeTab: "all"}
}

// Name + DisplayName satisfy the editor.Workspace contract.
func (w *Workspace) Name() string        { return "library" }
func (w *Workspace) DisplayName() string { return "Library" }

// ActiveTab returns the currently-selected tab id.
func (w *Workspace) ActiveTab() string {
	if w == nil {
		return ""
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.activeTab
}

// SetActiveTab switches the focused tab. Unknown tab ids are
// ignored so a stale UI state can't push the workspace into an
// invalid view.
func (w *Workspace) SetActiveTab(id string) {
	if w == nil {
		return
	}
	for _, valid := range CanonicalGameTabs {
		if id == valid {
			w.mu.Lock()
			w.activeTab = id
			w.mu.Unlock()
			return
		}
	}
}

// PacksForActiveTab returns the packs the active tab should
// render. Empty when the active tab has no installed packs (or
// the tab is "custom" — Custom assets come from the user-library
// dir, not curated packs).
func (w *Workspace) PacksForActiveTab() []Pack {
	if w == nil || w.lib == nil {
		return nil
	}
	tab := w.ActiveTab()
	return w.lib.PacksForGame(tab)
}

// CustomAssets returns the user-library files Custom tab
// displays. Reads the user-library directory's contents — each
// asset-extension file becomes one Asset entry with License
// "unknown" + Author "user". The watcher (U11) populates the
// directory; the workspace reads its current state.
func (w *Workspace) CustomAssets() []Asset {
	if w == nil || w.lib == nil {
		return nil
	}
	dir := UserLibraryDir(w.lib.CacheRoot())
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Asset
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		kind := classifyAssetExt(name)
		if kind == "" {
			continue
		}
		out = append(out, Asset{
			Path:    name,
			Kind:    kind,
			License: "unknown",
			Author:  "user",
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// AddToProject copies a pack asset into the bound project's
// assets dir and creates the matching SpriteAsset or AudioSample
// record. Idempotent — re-adding the same name no-ops without
// duplicating the record.
//
// Returns ErrNoEditor when no editor binding is attached (tests
// + diagnostic surfaces that hold a workspace without a project).
// Returns ErrUnknownAsset when (packID, relPath) doesn't resolve.
func (w *Workspace) AddToProject(packID, relPath string) error {
	if w == nil {
		return ErrNoEditor
	}
	if w.editor == nil {
		return ErrNoEditor
	}
	asset := w.lib.LookupAsset(packID, relPath)
	if asset == nil {
		return fmt.Errorf("%w: pack=%q path=%q", ErrUnknownAsset, packID, relPath)
	}
	p := w.editor.Project()
	if p == nil {
		return ErrNoEditor
	}
	projectPath := w.editor.CurrentProjectPath()
	if projectPath == "" {
		return ErrNoProjectPath
	}
	assetsDir := pixelforge_project.AssetsDir(projectPath)
	srcPath := AssetPath(w.lib.CacheRoot(), packID, relPath)

	// Destination path under the project's *-assets/ dir mirrors
	// the source's "kind/basename" layout (sprites/, audio/, etc.).
	subdir := assetSubdir(asset.Kind)
	dstRel := filepath.Join(subdir, filepath.Base(relPath))
	dstAbs := filepath.Join(assetsDir, dstRel)
	if err := copyAssetFile(srcPath, dstAbs); err != nil {
		return fmt.Errorf("library: copy asset: %w", err)
	}

	switch asset.Kind {
	case "sprite":
		name := nameWithoutExt(filepath.Base(relPath))
		if hasSprite(p, name) {
			return nil
		}
		p.Sprites = append(p.Sprites, pixelforge_project.SpriteAsset{
			Name:         name,
			RelativePath: filepath.ToSlash(dstRel),
		})
	case "sfx", "bgm":
		name := nameWithoutExt(filepath.Base(relPath))
		if hasAudio(p, name) {
			return nil
		}
		channel := "sfx"
		if asset.Kind == "bgm" {
			channel = "bgm"
		}
		p.Audio = append(p.Audio, pixelforge_project.AudioSample{
			Name:                     name,
			RelativePath:             filepath.ToSlash(dstRel),
			SuggestedChannelPriority: channel,
		})
	default:
		return fmt.Errorf("library: unknown asset kind %q", asset.Kind)
	}
	w.editor.MarkDirty()
	return nil
}

// ErrNoEditor is returned when AddToProject is called on a
// workspace without a bound editor + project.
var ErrNoEditor = errors.New("library: no editor binding")

// ErrNoProjectPath is returned when AddToProject is called and
// the editor has no current project path (asset copy needs a
// destination directory).
var ErrNoProjectPath = errors.New("library: editor has no current project path")

// ErrUnknownAsset is returned when (packID, relPath) doesn't
// resolve to a registered asset.
var ErrUnknownAsset = errors.New("library: unknown asset")

// ---- internals --------------------------------------------------

// classifyAssetExt mirrors the ingest package's classifier
// without adding a circular import. Kept tiny on purpose; the
// kind strings match what manifests use.
func classifyAssetExt(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".png", ".PNG":
		return "sprite"
	case ".wav", ".WAV":
		return "sfx"
	case ".ogg", ".OGG", ".mp3", ".MP3":
		return "bgm"
	}
	return ""
}

func assetSubdir(kind string) string {
	switch kind {
	case "sprite":
		return "sprites"
	case "sfx", "bgm":
		return "audio"
	}
	return ""
}

func nameWithoutExt(base string) string {
	ext := filepath.Ext(base)
	if ext == "" {
		return base
	}
	return base[:len(base)-len(ext)]
}

func hasSprite(p *pixelforge_project.Project, name string) bool {
	for _, s := range p.Sprites {
		if s.Name == name {
			return true
		}
	}
	return false
}

func hasAudio(p *pixelforge_project.Project, name string) bool {
	for _, a := range p.Audio {
		if a.Name == name {
			return true
		}
	}
	return false
}

func copyAssetFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
