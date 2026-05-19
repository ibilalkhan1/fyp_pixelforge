// Package widgets renders the auto-generated inspector controls.
//
// Each WidgetKind from pfcomponent has a corresponding Widget here; the
// inspector instantiates one per field of the selected entity's
// components and delegates draw + input to them.
//
// All widgets render at native window resolution (M0-M2 chrome model)
// via ebitenutil.DebugPrintAt + vector.DrawFilledRect. The Widget
// interface is intentionally narrow so M3's canvas-resident widgets
// can adopt the same surface.
package widgets

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
)

// Rect is a window-pixel rectangle the inspector hands to each widget.
type Rect struct {
	X, Y, W, H int
}

// Contains reports whether (mx, my) is inside r.
func (r Rect) Contains(mx, my int) bool {
	return mx >= r.X && mx < r.X+r.W && my >= r.Y && my < r.Y+r.H
}

// EditEvent is emitted when a widget's value changes. The inspector
// turns these into project mutations.
type EditEvent struct {
	NewValue any
}

// Context bundles the cross-widget state the inspector hands down:
// references to the project's sprite/audio catalogs so dropdowns can
// list valid choices, the active mouse position, etc.
type Context struct {
	// SpriteNames is the sorted list of sprite names in the project.
	// Used by WidgetSpriteRef.
	SpriteNames []string

	// AudioNames is the sorted list of audio sample names.
	AudioNames []string

	// EventTopics is the catalog of pievent topics the project has
	// declared. M5 populates this; empty for now.
	EventTopics []string

	// PaletteColors carries the active palette ("#RRGGBB" per slot)
	// so WidgetPaletteColor renders an accurate preview.
	PaletteColors []string

	// Intents is the sorted list of input-intent topic names (e.g.,
	// "input/jump") the project's input layer publishes. U8's
	// character-sheet inspector exposes this list so future verb
	// parameter forms that target an intent can drop a dropdown over
	// the live intent set rather than free-text. Populated by the
	// editor's buildWidgetContext from pixelforge_input.AllIntents.
	Intents []string

	// VerbRecipes is the sorted list of all registered verb-recipe
	// names from pixelforge_studio/scripting/catalog. The character-
	// sheet inspector filters this list per trigger slot at render
	// time using each TriggerSlot's RelevantRecipes; carrying the
	// global list here keeps the Context honest about what authoring
	// can reach.
	VerbRecipes []string

	// BGSubPaletteNames lists the project's 4 background sub-
	// palette names (idea #3 v1). Source for the
	// WidgetSubPalette family=bg dropdown.
	BGSubPaletteNames []string

	// SpriteSubPaletteNames lists the project's 4 sprite sub-
	// palette names. Source for the WidgetSubPalette family=sprite
	// dropdown (e.g. SpriteAsset.SubPalette).
	SpriteSubPaletteNames []string
}

// Widget is the surface every inspector control implements.
type Widget interface {
	// Field is the metadata that produced this widget. The inspector
	// uses it to read field labels, bounds, etc.
	Field() pfcomponent.FieldMetadata

	// Draw paints the widget into area. Value is the current field
	// value; the widget interprets it according to its kind.
	Draw(dst *ebiten.Image, area Rect, value any)

	// Update processes input for the frame. mx/my is the window-pixel
	// mouse position; pressed reports whether the primary mouse
	// button is currently down. Returns a non-nil event when the
	// value changed this frame.
	Update(area Rect, value any, mx, my int, pressed bool) *EditEvent
}

// New constructs the widget appropriate for a field's metadata. Falls
// back to the default field for unknown widget kinds so newer schemas
// don't crash older editors.
func New(field pfcomponent.FieldMetadata) Widget {
	switch field.WidgetKind {
	case pfcomponent.WidgetSlider:
		return &SliderWidget{F: field}
	case pfcomponent.WidgetPaletteColor:
		return &ColorPickerWidget{F: field}
	case pfcomponent.WidgetSpriteRef:
		return &SpriteRefWidget{F: field}
	case pfcomponent.WidgetAudioRef:
		return &AudioRefWidget{F: field}
	case pfcomponent.WidgetEventTopic:
		return &EventTopicWidget{F: field}
	case pfcomponent.WidgetEnum:
		return &EnumWidget{F: field}
	case pfcomponent.WidgetText:
		return &TextWidget{F: field}
	case pfcomponent.WidgetVector2:
		return &Vector2Widget{F: field}
	case pfcomponent.WidgetCheckbox:
		return &CheckboxWidget{F: field}
	case pfcomponent.WidgetIntField, pfcomponent.WidgetFloatField:
		return &NumericWidget{F: field}
	case pfcomponent.WidgetUnknown:
		return &UnknownWidget{F: field}
	default:
		return &DefaultFieldWidget{F: field}
	}
}

// Shared paint primitives. Kept in this file so individual widget
// implementations stay short.

var (
	colWidgetBg     = color.RGBA{R: 0x18, G: 0x18, B: 0x22, A: 0xff}
	colWidgetHover  = color.RGBA{R: 0x2a, G: 0x2a, B: 0x36, A: 0xff}
	colWidgetBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x40, A: 0xff}
	colWidgetFill   = color.RGBA{R: 0x46, G: 0x86, B: 0xff, A: 0xff}
	colWidgetText   = color.RGBA{R: 0xea, G: 0xea, B: 0xf0, A: 0xff}
	colWidgetDim    = color.RGBA{R: 0x90, G: 0x90, B: 0x9c, A: 0xff}
	colWarning      = color.RGBA{R: 0xff, G: 0x9d, B: 0x40, A: 0xff}
)

func fillRect(dst *ebiten.Image, r Rect, c color.RGBA) {
	vector.DrawFilledRect(dst, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), c, false)
}

func strokeRect(dst *ebiten.Image, r Rect, c color.RGBA) {
	vector.StrokeRect(dst, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), 1, c, false)
}

// printAt is a shared helper around ebitenutil.DebugPrintAt — colored
// rendering uses a scratch image (see chrome.go in the editor pkg).
func printAt(dst *ebiten.Image, s string, x, y int) {
	ebitenutil.DebugPrintAt(dst, s, x, y)
}

// isClickJustPressed wraps inpututil so widgets share one definition.
func isClickJustPressed() bool {
	return inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
}

// asFloat best-effort converts an arbitrary numeric `any` to float64.
// JSON-loaded values arrive as float64; in-code values may be int/etc.
func asFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	default:
		return 0, false
	}
}

// asString best-effort converts an `any` to string for text widgets.
func asString(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

// asBool extracts a bool.
func asBool(v any) (bool, bool) {
	b, ok := v.(bool)
	return b, ok
}
