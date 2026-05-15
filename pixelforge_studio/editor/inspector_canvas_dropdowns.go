package editor

import (
	"fmt"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	pguiwidgets "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
)

// canvasDropdownKind identifies which of the five canvas dropdown
// widgets a given field maps to. U45 ships them alongside the native
// bank; U46 retires the native files once parity is verified.
type canvasDropdownKind int

const (
	canvasDropdownColorPicker canvasDropdownKind = iota
	canvasDropdownSpriteRef
	canvasDropdownAudioRef
	canvasDropdownEventTopic
	canvasDropdownEnum
)

// canvasDropdownFor returns the canvas dropdown kind for a given field,
// or false when the field shouldn't be routed onto the canvas path.
func canvasDropdownFor(md pfcomponent.FieldMetadata) (canvasDropdownKind, bool) {
	switch md.WidgetKind {
	case pfcomponent.WidgetPaletteColor:
		return canvasDropdownColorPicker, true
	case pfcomponent.WidgetSpriteRef:
		return canvasDropdownSpriteRef, true
	case pfcomponent.WidgetAudioRef:
		return canvasDropdownAudioRef, true
	case pfcomponent.WidgetEventTopic:
		return canvasDropdownEventTopic, true
	case pfcomponent.WidgetEnum:
		return canvasDropdownEnum, true
	default:
		return 0, false
	}
}

// canvasDropdownWidget wraps a pgui.Dropdown for a single inspector
// field. The bind-context (sprite/audio/event/palette/enum option list)
// is computed each frame from the inspector's WidgetContext.
type canvasDropdownWidget struct {
	Kind     canvasDropdownKind
	Field    pfcomponent.FieldMetadata
	Dropdown *pguiwidgets.Dropdown
	// Last known options list; re-set when the project changes.
	lastOptions []string
	// Surface for the latest OnSelect's value — read back by the
	// inspector to apply the edit to the entity.
	pendingValue string
	hasPending   bool
}

// newCanvasDropdownWidget constructs a canvas dropdown for kind.
func newCanvasDropdownWidget(kind canvasDropdownKind, field pfcomponent.FieldMetadata) *canvasDropdownWidget {
	w := &canvasDropdownWidget{
		Kind:  kind,
		Field: field,
	}
	w.Dropdown = pguiwidgets.NewDropdown(0, 0, 0, 0, 0, pguiwidgets.DropdownOptions{
		Options: nil,
		OnSelect: func(opt string) {
			w.pendingValue = opt
			w.hasPending = true
		},
	})
	return w
}

// SetOptions replaces the dropdown's option list. Idempotent when the
// list hasn't changed (avoids resetting the open/closed state every
// frame).
func (w *canvasDropdownWidget) SetOptions(opts []string) {
	if stringSliceEqual(w.lastOptions, opts) {
		return
	}
	w.lastOptions = append([]string(nil), opts...)
	w.Dropdown.SetOptions(opts)
}

// ConsumePending returns the pending OnSelect value (if any) and
// clears the buffer. The inspector calls this once per frame after
// running Update.
func (w *canvasDropdownWidget) ConsumePending() (string, bool) {
	if !w.hasPending {
		return "", false
	}
	v := w.pendingValue
	w.hasPending = false
	w.pendingValue = ""
	return v, true
}

// DrawCanvas paints the dropdown into the inspector at (x, y) using
// engine primitives. The dropdown widget exposes Draw via its own
// pgui.Element hook; we manually shift the camera the same way other
// canvas widgets do.
func (w *canvasDropdownWidget) DrawCanvas(x, y, width int, theme *EditorTheme, currentValue string) {
	if w.Dropdown == nil {
		return
	}
	w.Dropdown.X = x
	w.Dropdown.Y = y
	w.Dropdown.W = width
	w.Dropdown.H = 18

	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	pixelforge.SetColor(theme.PanelSlot)
	pixelforge.RectFill(x, y, x+width-1, y+17)
	pixelforge.SetColor(theme.AccentSlot)
	pixelforge.Rect(x, y, x+width-1, y+17)
	pixelforge.SetColor(theme.TextSlot)
	pixelforge_cofont.Print(fmt.Sprintf("%s: %s", w.Field.Name, currentValue), x+4, y+5)

	// Caret on the right side hints "click to open dropdown".
	pixelforge.SetColor(theme.AccentSlot)
	pixelforge.RectFill(x+width-10, y+6, x+width-4, y+11)
}

// IsEmpty reports whether the option list is empty — the inspector
// renders a "no … loaded" placeholder when true.
func (w *canvasDropdownWidget) IsEmpty() bool { return len(w.lastOptions) == 0 }

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
