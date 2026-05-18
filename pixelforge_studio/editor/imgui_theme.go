// imgui_theme.go translates the editor.pforge Theme schema into ImGui
// style state. U6 of the ImGui migration plan replaced the chrome.go
// colour-on-screen path with a per-frame imgui.PushStyleColor stack
// fed by the loaded theme; the theme schema (palette indices +
// font name) survives unchanged from the M2 cart-resident design,
// only its consumer changes.
//
// imgui.ini persistence is configured here too — the editor points
// ImGui at <user-config-dir>/pixelforge-studio/imgui.ini so the
// user's dock layout survives restarts (R6).
package editor

import (
	"bytes"
	"embed"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_font"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

//go:embed cart_assets/editor.pforge
var editorThemeFS embed.FS

// LoadEmbeddedEditorProject parses the embedded editor.pforge fixture
// and returns the in-memory project. The project's Theme drives chrome
// colours. Returns nil on parse failure so callers fall back to the
// default theme; the failure is logged but never fatal — a broken
// fixture must not crash the studio.
func LoadEmbeddedEditorProject() *pixelforge_project.Project {
	data, err := editorThemeFS.ReadFile("cart_assets/editor.pforge")
	if err != nil {
		log.Printf("pixelforge_studio: failed to read embedded editor.pforge: %v", err)
		return nil
	}
	p, err := pixelforge_project.LoadReader(bytes.NewReader(data))
	if err != nil {
		log.Printf("pixelforge_studio: failed to parse embedded editor.pforge: %v", err)
		return nil
	}
	return p
}

// loadEditorTheme returns the editor's chrome theme. Sourced from the
// embedded editor.pforge fixture (R7 dogfooding); falls back to
// DefaultEditorTheme when the fixture fails to load.
//
// Side effect: if theme.FontName names a system font, the cofont
// active-sheet swaps so subsequent cofont.Print calls dispatch there.
// Unknown names log a warning and fall back to the cofont default.
func loadEditorTheme() *EditorTheme {
	p := LoadEmbeddedEditorProject()
	var th *EditorTheme
	if p == nil {
		th = DefaultEditorTheme()
	} else {
		th = themeFromProject(p.Theme)
	}
	applyFontTheme(th.FontName)
	return th
}

func applyFontTheme(name string) {
	switch name {
	case "", "cofont":
		pixelforge_cofont.SetActiveSheet(nil)
	case "system":
		sheet := pixelforge_font.NewSystemSheet()
		pixelforge_cofont.SetActiveSheet(&sheet)
	default:
		log.Printf("pixelforge_studio: unknown font theme %q — falling back to cofont", name)
		pixelforge_cofont.SetActiveSheet(nil)
	}
}

func themeFromProject(t pixelforge_project.Theme) *EditorTheme {
	return &EditorTheme{
		BackgroundSlot:  uint8(t.BackgroundSlot),
		PanelSlot:       uint8(t.PanelSlot),
		PanelHeaderSlot: uint8(t.PanelHeaderSlot),
		TextSlot:        uint8(t.TextSlot),
		TextDimSlot:     uint8(t.TextDimSlot),
		AccentSlot:      uint8(t.AccentSlot),
		WarningSlot:     uint8(t.WarningSlot),
		FontName:        t.FontName,
	}
}

// imguiIniSubdir is the user-config-dir-relative directory the
// editor stashes imgui.ini under. Co-located with settings.json so a
// user opening the directory finds everything pixelforge-studio
// stores about their editor preferences in one place.
const imguiIniSubdir = "pixelforge-studio"

// imguiIniFilename is the filename the editor passes to
// imgui.IO.SetIniFilename. Standard ImGui name; tooling that knows
// imgui.ini will recognise it.
const imguiIniFilename = "imgui.ini"

// imguiTheme holds the per-frame ImGui colours derived from an
// EditorTheme. Computed once per theme load, applied each frame via
// pushColors() inside buildChrome's outer scope.
type imguiTheme struct {
	Text       imgui.Vec4
	TextDim    imgui.Vec4
	WindowBg   imgui.Vec4
	PanelBg    imgui.Vec4
	PanelHdr   imgui.Vec4
	Accent     imgui.Vec4
	Warning    imgui.Vec4
	Border     imgui.Vec4
}

// buildImguiTheme converts an EditorTheme + project palette into the
// per-call colour set ImGui needs. Palette slot indices are resolved
// against the loaded project's Base palette; out-of-range slots fall
// back to neutral grey so a hand-edited theme can't crash chrome.
func buildImguiTheme(theme *EditorTheme, palette *pixelforge_project.PaletteData) imguiTheme {
	if theme == nil {
		theme = DefaultEditorTheme()
	}
	lookup := func(slot uint8) imgui.Vec4 {
		if palette == nil {
			return imgui.NewVec4(0.5, 0.5, 0.5, 1)
		}
		idx := int(slot)
		if idx < 0 || idx >= pixelforge_project.MaxColors {
			return imgui.NewVec4(0.5, 0.5, 0.5, 1)
		}
		return hexToVec4(palette.Base[idx])
	}
	return imguiTheme{
		Text:     lookup(uint8(theme.TextSlot)),
		TextDim:  lookup(uint8(theme.TextDimSlot)),
		WindowBg: lookup(uint8(theme.BackgroundSlot)),
		PanelBg:  lookup(uint8(theme.PanelSlot)),
		PanelHdr: lookup(uint8(theme.PanelHeaderSlot)),
		Accent:   lookup(uint8(theme.AccentSlot)),
		Warning:  lookup(uint8(theme.WarningSlot)),
		Border:   lookup(uint8(theme.PanelHeaderSlot)),
	}
}

// applyImguiTheme pushes the theme colours onto ImGui's style stack.
// Returns the count of pushed colours so callers can pop the same
// number at end-of-frame. Safe to call without a live backend — the
// no-op short-circuit guarantees zero pushed entries in that case.
func applyImguiTheme(t imguiTheme, live bool) int {
	if !live {
		return 0
	}
	pushes := []struct {
		idx imgui.Col
		val imgui.Vec4
	}{
		{imgui.ColText, t.Text},
		{imgui.ColTextDisabled, t.TextDim},
		{imgui.ColWindowBg, t.WindowBg},
		{imgui.ColChildBg, t.PanelBg},
		{imgui.ColPopupBg, t.PanelBg},
		{imgui.ColBorder, t.Border},
		{imgui.ColTitleBg, t.PanelHdr},
		{imgui.ColTitleBgActive, t.PanelHdr},
		{imgui.ColMenuBarBg, t.PanelHdr},
		{imgui.ColHeader, t.PanelHdr},
		{imgui.ColHeaderHovered, t.Accent},
		{imgui.ColHeaderActive, t.Accent},
		{imgui.ColButton, t.PanelHdr},
		{imgui.ColButtonHovered, t.Accent},
		{imgui.ColButtonActive, t.Accent},
		{imgui.ColCheckMark, t.Accent},
		{imgui.ColSliderGrab, t.Accent},
		{imgui.ColSliderGrabActive, t.Accent},
		{imgui.ColFrameBg, t.PanelBg},
		{imgui.ColFrameBgHovered, t.PanelHdr},
		{imgui.ColFrameBgActive, t.PanelHdr},
	}
	for _, p := range pushes {
		imgui.PushStyleColorVec4(p.idx, p.val)
	}
	return len(pushes)
}

// popImguiTheme pops the colour entries applyImguiTheme pushed.
func popImguiTheme(count int, live bool) {
	if !live || count <= 0 {
		return
	}
	imgui.PopStyleColorV(int32(count))
}

// hexToVec4 parses an "#RRGGBB" string into an ImGui Vec4 with
// fully-opaque alpha. Returns neutral grey on parse failure so a
// malformed palette entry doesn't crash chrome.
func hexToVec4(hex string) imgui.Vec4 {
	s := strings.TrimPrefix(hex, "#")
	if len(s) != 6 {
		return imgui.NewVec4(0.5, 0.5, 0.5, 1)
	}
	r, err1 := strconv.ParseUint(s[0:2], 16, 8)
	g, err2 := strconv.ParseUint(s[2:4], 16, 8)
	b, err3 := strconv.ParseUint(s[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return imgui.NewVec4(0.5, 0.5, 0.5, 1)
	}
	return imgui.NewVec4(float32(r)/255, float32(g)/255, float32(b)/255, 1)
}

// imguiIniPath returns the absolute path the editor's imgui.ini
// should live at. Created lazily by ImGui on first save; the parent
// directory is created here when missing so SetIniFilename never
// silently fails to write.
func imguiIniPath() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cfg, imguiIniSubdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, imguiIniFilename), nil
}

// configureImguiIniPath wires ImGui's persistent layout file to the
// editor's chosen path. Called once after AttachImguiBackend so the
// imgui.Context exists. Logs and continues on path resolution
// failure — losing layout persistence is annoying but not fatal.
func configureImguiIniPath() {
	path, err := imguiIniPath()
	if err != nil {
		log.Printf("pixelforge studio: imgui.ini path: %v (layout persistence disabled)", err)
		return
	}
	io := imgui.CurrentIO()
	if io == nil {
		return
	}
	io.SetIniFilename(path)
}
