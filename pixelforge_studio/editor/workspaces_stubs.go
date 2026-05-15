package editor

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// placeholderWorkspace is the shared shape behind the placeholder workspaces
// for M4 (Capture), M5 (Behavior), M6 (Audio), and M7 (Procgen). Each
// renders a centred "coming in MX" line on the editor canvas; Draw on
// the native overlay path delegates to a debugPrint at the canvas
// region's centre so the workspace remains discoverable before the
// real implementations land.
type placeholderWorkspace struct {
	name        string
	displayName string
	milestone   string
}

// Name returns the stable workspace identifier.
func (s *placeholderWorkspace) Name() string { return s.name }

// DisplayName returns the tab strip label.
func (s *placeholderWorkspace) DisplayName() string { return s.displayName }

// Draw renders the placeholder text via the native overlay path so the
// workspace renders even on builds that don't go through the cart.
func (s *placeholderWorkspace) Draw(dst *ebiten.Image, area widgets.Rect, e *Editor) {
	if area.W <= 0 || area.H <= 0 {
		return
	}
	msg := s.displayName + " — coming in " + s.milestone
	x := area.X + (area.W-len(msg)*6)/2
	y := area.Y + area.H/2 - debugLineHeight/2
	debugPrint(dst, msg, x, y)
}

// DrawCanvas renders the canvas-resident placeholder so the stub
// workspaces dogfood R1 just like the Scene and Palette workspaces.
func (s *placeholderWorkspace) DrawCanvas(rel widgets.Rect, e *Editor) {
	if rel.W <= 0 || rel.H <= 0 || e == nil {
		return
	}
	theme := DefaultEditorTheme()
	if c := e.Cart(); c != nil && c.Theme() != nil {
		theme = c.Theme()
	}
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	pixelforge.SetColor(theme.BackgroundSlot)
	pixelforge.RectFill(rel.X, rel.Y, rel.X+rel.W-1, rel.Y+rel.H-1)

	pixelforge.SetColor(theme.TextSlot)
	msg := s.displayName + " - coming in " + s.milestone
	// Approximate 4-px wide cofont glyphs.
	x := rel.X + (rel.W-len(msg)*4)/2
	y := rel.Y + rel.H/2 - pixelforge_cofont.Sheet.Height/2
	pixelforge_cofont.Print(msg, x, y)
}

// Update is intentionally a no-op — stub workspaces capture no input.
func (s *placeholderWorkspace) Update(_ *Editor) {}

// newPlaceholderWorkspace constructs a placeholder for a future milestone.
func newPlaceholderWorkspace(name, displayName, milestone string) *placeholderWorkspace {
	return &placeholderWorkspace{name: name, displayName: displayName, milestone: milestone}
}

// installStubWorkspaces registers the M4-M7 placeholders so the tab
// strip is stable from M3 onwards. The real implementations replace
// each stub in place by re-registering with the same Name.
func (e *Editor) installStubWorkspaces() {
	e.RegisterWorkspace(newPlaceholderWorkspace("behavior", "Behavior", "M5"))
	e.RegisterWorkspace(newPlaceholderWorkspace("audio", "Audio", "M6"))
	e.RegisterWorkspace(newPlaceholderWorkspace("capture", "Capture", "M4"))
	e.RegisterWorkspace(newPlaceholderWorkspace("procgen", "Procgen", "M7"))
}
