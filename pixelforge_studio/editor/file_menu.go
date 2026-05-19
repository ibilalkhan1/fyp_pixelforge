package editor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/codegen"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/templates"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// FileMenu carries the editor-level File-menu actions. Each method
// drives the editor's underlying state through the same paths the
// keyboard shortcuts use; the menu_bar widget calls these on click.
type FileMenu struct {
	editor *Editor
}

// NewFileMenu binds a menu instance to its editor.
func NewFileMenu(e *Editor) *FileMenu { return &FileMenu{editor: e} }

// New replaces the project with a freshly-minted blank one. Confirms
// with the user when the current project is dirty.
func (m *FileMenu) New() {
	m.editor.PromptIfDirty("Discard changes?", "The current project has unsaved changes.", func() {
		m.editor.SetProject(pixelforge_project.NewProject("untitled"))
		m.editor.SetCurrentProjectPath("")
		m.editor.SetStatusMessage("new project")
	})
}

// NewFromTemplate replaces the project with the named genre-starter
// template (plan-009 U22, R10). Unknown template names surface as a
// red-toned status message and leave the current project alone. Dirty-
// state prompt mirrors the File → New / File → Open path so designers
// never lose work to an accidental menu click.
//
// The new project loads with dirty=false: the template's content is
// the designer's starting baseline, not a "modification" relative to
// some other project (matches the U21 doc-review consistency on the
// forked / opened project flow). Designers immediately edit and the
// first mutation flips the dirty bit.
func (m *FileMenu) NewFromTemplate(templateName string) {
	m.editor.PromptIfDirty("Discard changes?", "The current project has unsaved changes.", func() {
		ctor := templates.Lookup(templateName)
		if ctor == nil {
			m.editor.SetStatusMessage("new from template: unknown template " + templateName)
			return
		}
		p := ctor()
		m.editor.SetProject(p)
		m.editor.SetCurrentProjectPath("")
		m.editor.ClearDirty()
		m.editor.SetStatusMessage("new " + templateName + " project")
	})
}

// Open shows the file picker filtered to .pforge files.
func (m *FileMenu) Open() {
	m.editor.PromptIfDirty("Discard changes?", "The current project has unsaved changes.", func() {
		m.editor.filePicker.Open(widgets.FilePickerOptions{
			StartPath:  m.openStartPath(),
			Mode:       widgets.PickOpen,
			Extensions: []string{".pforge"},
			Title:      "Open Project",
			OnConfirm:  func(path string) { _ = m.OpenPath(path) },
		})
	})
}

// OpenPath loads `path` into the editor. Public so tests and the
// recent-projects menu can drive it without going through the picker.
func (m *FileMenu) OpenPath(path string) error {
	p, err := pixelforge_project.Load(path)
	if err != nil {
		m.editor.SetStatusMessage("open failed: " + err.Error())
		return err
	}
	m.editor.SetProject(p)
	m.editor.SetCurrentProjectPath(path)
	m.editor.settings.PushRecentProject(path)
	m.editor.settings.MarkDirty()
	m.editor.SetStatusMessage("opened " + filepath.Base(path))
	return nil
}

// Save writes the project to its current path. Routes to Save As when
// the project has no source path yet.
func (m *FileMenu) Save() error {
	if m.editor.currentProjectPath == "" {
		m.SaveAs()
		return nil
	}
	return m.SaveTo(m.editor.currentProjectPath)
}

// SaveTo writes the project to path and updates the current path.
func (m *FileMenu) SaveTo(path string) error {
	if err := m.editor.project.Save(path); err != nil {
		m.editor.SetStatusMessage("save failed: " + err.Error())
		return err
	}
	m.editor.SetCurrentProjectPath(path)
	m.editor.ClearDirty()
	m.editor.settings.PushRecentProject(path)
	m.editor.settings.MarkDirty()
	m.editor.SetStatusMessage("saved " + filepath.Base(path))
	return nil
}

// SaveAs shows the picker in save mode.
func (m *FileMenu) SaveAs() {
	m.editor.filePicker.Open(widgets.FilePickerOptions{
		StartPath:  m.saveStartPath(),
		Mode:       widgets.PickSave,
		Extensions: []string{".pforge"},
		Title:      "Save Project As",
		OnConfirm: func(path string) {
			_ = m.SaveTo(path)
		},
	})
}

// Export runs the codegen pipeline into a user-chosen directory.
func (m *FileMenu) Export() {
	m.editor.filePicker.Open(widgets.FilePickerOptions{
		StartPath: m.saveStartPath(),
		Mode:      widgets.PickDir,
		Title:     "Export Game",
		OnConfirm: func(path string) { _ = m.ExportTo(path) },
	})
}

// ExportTo writes the project to outDir via codegen.Generate.
func (m *FileMenu) ExportTo(outDir string) error {
	_, err := codegen.Generate(m.editor.project, outDir, codegen.Options{
		Force:             true,
		RunGoModTidy:      true,
		ProjectSourcePath: m.editor.currentProjectPath,
	})
	if err != nil {
		m.editor.SetStatusMessage("export failed: " + err.Error())
		return err
	}
	m.editor.SetStatusMessage("exported to " + outDir)
	return nil
}

// ImportWAV is the File → Import WAV handler (idea #4 v1 U8).
// Opens the file picker filtered to .wav; on confirm, dispatches
// into the editor's AudioImportHandler which calls
// audiolib.ImportFromFile via the registered runner.
func (m *FileMenu) ImportWAV() {
	m.editor.filePicker.Open(widgets.FilePickerOptions{
		StartPath:  m.openStartPath(),
		Mode:       widgets.PickOpen,
		Extensions: []string{".wav"},
		Title:      "Import WAV",
		OnConfirm: func(path string) {
			if h := m.editor.AudioImportHandler(); h != nil {
				_, _ = h.Import(path)
			}
		},
	})
}

// ImportPNG is the File → Import PNG handler (idea #3 v1 U3). Opens
// the file picker filtered to .png; on confirm, dispatches into
// the editor's ImportHandler which runs palette.ImportWithDiff and
// queues the U4 diff modal for designer review.
func (m *FileMenu) ImportPNG() {
	m.editor.filePicker.Open(widgets.FilePickerOptions{
		StartPath:  m.openStartPath(),
		Mode:       widgets.PickOpen,
		Extensions: []string{".png"},
		Title:      "Import PNG",
		OnConfirm: func(path string) {
			if h := m.editor.ImportHandler(); h != nil {
				_, _ = h.Import(path)
			}
		},
	})
}

// Quit asks for confirmation when dirty, then terminates the editor.
func (m *FileMenu) Quit() {
	m.editor.PromptIfDirty("Quit without saving?", "The current project has unsaved changes.", func() {
		m.editor.Quit()
	})
}

// openStartPath returns the best starting directory for File → Open.
// Recent projects sit first, then the user's documents dir, then home.
func (m *FileMenu) openStartPath() string {
	if len(m.editor.settings.RecentProjects) > 0 {
		return filepath.Dir(m.editor.settings.RecentProjects[0])
	}
	return userDocumentsDir()
}

// saveStartPath mirrors openStartPath but also handles the current
// project path when one is set.
func (m *FileMenu) saveStartPath() string {
	if m.editor.currentProjectPath != "" {
		return filepath.Dir(m.editor.currentProjectPath)
	}
	return m.openStartPath()
}

// userDocumentsDir returns ~/Documents/pixelforge if it exists, else the
// user's home directory, else the working directory.
func userDocumentsDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, "Documents", "pixelforge")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		return home
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return string(filepath.Separator)
}

// NewFromTemplateMenu returns the items the File → New submenu renders
// (plan-009 U22). One item per genre-starter template in
// templates.AllNames() order; each item's OnSelect invokes
// NewFromTemplate with the matching name. Rebuilt every frame so a
// template registry change shows up without menu-bar invalidation.
func (e *Editor) NewFromTemplateMenu() []widgets.MenuItem {
	if e == nil {
		return nil
	}
	names := templates.AllNames()
	items := make([]widgets.MenuItem, 0, len(names))
	for _, name := range names {
		name := name
		items = append(items, widgets.MenuItem{
			Label: name,
			OnSelect: func() {
				e.fileMenu.NewFromTemplate(name)
			},
		})
	}
	return items
}

// buildMenuDefs assembles the four top-level menus the menu bar renders.
// Kept on the editor so the OnSelect callbacks close over `e` directly.
func (e *Editor) buildMenuDefs() []widgets.MenuDef {
	return []widgets.MenuDef{
		{
			Label: "File",
			Items: []widgets.MenuItem{
				{
					Label:    "New",
					Shortcut: "Ctrl+N",
					OnSelect: e.fileMenu.New,
					// Plan-009 U22 — File → New → <genre>. The
					// submenu carries the four genre-starter
					// templates. The parent's OnSelect still
					// fires the legacy blank-project New so the
					// Ctrl+N shortcut keeps its meaning.
					Submenu: e.NewFromTemplateMenu(),
				},
				{Label: "Open...", Shortcut: "Ctrl+O", OnSelect: e.fileMenu.Open},
				{
					// Plan-009 U21 — File → Open Example. The
					// submenu is rebuilt every frame from the
					// asset library's current manifest, so a
					// background-bootstrap completion lights up
					// the rows without needing menu invalidation.
					Label:   "Open Example",
					Submenu: e.OpenExampleMenu(),
				},
				{Separator: true},
				{Label: "Save", Shortcut: "Ctrl+S", OnSelect: func() { _ = e.fileMenu.Save() }},
				{Label: "Save As...", Shortcut: "Ctrl+Shift+S", OnSelect: e.fileMenu.SaveAs},
				{Separator: true},
				{Label: "Import PNG...", OnSelect: e.fileMenu.ImportPNG},
				{Label: "Import WAV...", OnSelect: e.fileMenu.ImportWAV},
				{Separator: true},
				{Label: "Export...", Shortcut: "Ctrl+E", OnSelect: e.fileMenu.Export},
				{Separator: true},
				{Label: "Quit", Shortcut: "Ctrl+Q", OnSelect: e.fileMenu.Quit},
			},
		},
		{
			Label: "Edit",
			Items: []widgets.MenuItem{
				{Label: "Undo (M3)", Disabled: true},
				{Label: "Redo (M3)", Disabled: true},
			},
		},
		{
			Label: "View",
			Items: []widgets.MenuItem{
				{Label: "Scene", Shortcut: "Ctrl+1", OnSelect: func() { e.SetActiveWorkspaceByName("scene") }},
				{Label: "Palette", Shortcut: "Ctrl+2", OnSelect: func() { e.SetActiveWorkspaceByName("palette") }},
				{Label: "Audio", Shortcut: "Ctrl+4", OnSelect: func() { e.SetActiveWorkspaceByName("audio") }},
				{Label: "Dialogue", Shortcut: "Ctrl+5", OnSelect: func() { e.SetActiveWorkspaceByName("dialogue") }},
				{Label: "Items", Shortcut: "Ctrl+6", OnSelect: func() { e.SetActiveWorkspaceByName("items") }},
				{Label: "Menus", Shortcut: "Ctrl+7", OnSelect: func() { e.SetActiveWorkspaceByName("menus") }},
				{Label: "Build", Shortcut: "Ctrl+8", OnSelect: func() { e.SetActiveWorkspaceByName("build") }},
				{Separator: true},
				{
					Label:    "Build on save",
					Checked:  !e.settings.BuildOnSaveDisabled,
					OnSelect: e.ToggleBuildOnSave,
				},
				{Separator: true},
				{
					Label:    "Show 8-sprite-per-scanline overlay",
					Checked:  e.ScanlineOverlayEnabled(),
					OnSelect: e.ToggleScanlineOverlay,
				},
				{
					Label:    "Show 2x2 BG palette-block overlay",
					Checked:  e.PaletteBlockOverlayEnabled(),
					OnSelect: e.TogglePaletteBlockOverlay,
				},
			},
		},
		{
			Label: "Help",
			Items: []widgets.MenuItem{
				{Label: fmt.Sprintf("Pixelforge Studio v%s", Version), Disabled: true},
			},
		},
	}
}

// handleShortcuts routes keymap shortcuts to FileMenu actions.
func (e *Editor) handleShortcuts() {
	if e.keymap.JustPressed("file.new") {
		e.fileMenu.New()
	}
	if e.keymap.JustPressed("file.open") {
		e.fileMenu.Open()
	}
	if e.keymap.JustPressed("file.save_as") {
		e.fileMenu.SaveAs()
	}
	if e.keymap.JustPressed("file.save") {
		_ = e.fileMenu.Save()
	}
	if e.keymap.JustPressed("file.export") {
		e.fileMenu.Export()
	}
	if e.keymap.JustPressed("file.quit") {
		e.fileMenu.Quit()
	}

	// Tool shortcuts.
	if e.keymap.JustPressed("tool.select") {
		e.SetTool(ToolSelect)
	}
	if e.keymap.JustPressed("tool.place") {
		e.SetTool(ToolPlace)
	}
	if e.keymap.JustPressed("tool.delete") {
		e.SetTool(ToolDelete)
	}
	if e.keymap.JustPressed("tool.paint") {
		e.SetTool(ToolPaint)
	}

	// Idea #1 v1 — per-stroke undo/redo for the tile painter. The
	// stack lives on the editor so both the canvas dispatch (which
	// pushes commands) and the shortcut handler (which pops them)
	// see the same instance. MarkDirty fires unconditionally on a
	// successful pop because reverting/replaying is itself a project
	// mutation.
	if e.keymap.JustPressed("edit.undo") {
		if stack := e.UndoStack(); stack != nil {
			if cmd := stack.Undo(); cmd != nil {
				e.MarkDirty()
			}
		}
	}
	if e.keymap.JustPressed("edit.redo") {
		if stack := e.UndoStack(); stack != nil {
			if cmd := stack.Redo(); cmd != nil {
				e.MarkDirty()
			}
		}
	}

	// Workspace shortcuts.
	if e.keymap.JustPressed("workspace.cycle") {
		e.CycleWorkspace()
	}
	if e.keymap.JustPressed("workspace.scene") {
		e.SetActiveWorkspaceByName("scene")
	}
	if e.keymap.JustPressed("workspace.palette") {
		e.SetActiveWorkspaceByName("palette")
	}
	if e.keymap.JustPressed("workspace.behavior") {
		e.SetActiveWorkspaceByName("behavior")
	}
	if e.keymap.JustPressed("workspace.audio") {
		e.SetActiveWorkspaceByName("audio")
	}
	if e.keymap.JustPressed("workspace.capture") {
		e.SetActiveWorkspaceByName("capture")
	}
	if e.keymap.JustPressed("workspace.procgen") {
		e.SetActiveWorkspaceByName("procgen")
	}

	// M3 chrome visibility toggle. Modal precedence is enforced inside
	// handleEscape; e.handleShortcuts already runs only when no modal
	// owns input, so the keymap dispatch is safe here.
	if e.keymap.JustPressed("chrome.toggle") {
		e.handleEscape()
	}
}
