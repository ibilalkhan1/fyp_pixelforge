package editor

import (
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

var (
	headerColor  = color.RGBA{R: 0x2a, G: 0x2a, B: 0x36, A: 0xff}
	unknownColor = color.RGBA{R: 0x55, G: 0x33, B: 0x22, A: 0xff}
)

func vectorFill(dst *ebiten.Image, r widgets.Rect, c color.RGBA) {
	vector.DrawFilledRect(dst, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), c, false)
}

func debugPrint(dst *ebiten.Image, s string, x, y int) {
	ebitenutil.DebugPrintAt(dst, s, x, y)
}

// Inspector renders the right panel: the selected entity's components,
// with one auto-generated widget per field driven by the pfcomponent
// registry.
//
// The inspector caches widget instances per (entity ID, component
// index, field index) so transient widget state (dragging flag, text
// focus, dropdown open) survives between frames.
type Inspector struct {
	cache map[inspectorKey]widgets.Widget
}

type inspectorKey struct {
	EntityID string
	CompIdx  int
	FieldIdx int
}

// NewInspector returns an empty inspector. The first Draw call lazily
// instantiates widgets as the selection changes.
func NewInspector() *Inspector {
	return &Inspector{cache: map[inspectorKey]widgets.Widget{}}
}

// Draw renders the inspector content into the right panel. area is the
// panel rectangle the chrome carved out; project supplies catalogs;
// entity is the currently-selected entity (nil → empty inspector).
//
// Returns true if any field was edited this frame so the caller can
// mark the project dirty.
func (i *Inspector) Draw(
	dst *ebiten.Image,
	area widgets.Rect,
	project *pixelforge_project.Project,
	entity *pixelforge_project.Entity,
) bool {
	if entity == nil {
		return false
	}
	ctx := buildWidgetContext(project)

	mx, my := ebiten.CursorPosition()
	pressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	_ = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) // touched so the lib picks up state

	mutated := false
	y := area.Y + 4
	for ci, comp := range entity.Components {
		md, ok := pfcomponent.Get(comp.Type)
		if !ok {
			// Unregistered components: render a single header that
			// names the unknown type. The schema preserves the
			// values so an updated editor can pick them up.
			drawUnknownComponentHeader(dst, area, comp.Type, y)
			y += 22
			continue
		}
		drawComponentHeader(dst, area, md.Name, y)
		y += 18
		for fi, f := range md.Fields {
			fieldArea := widgets.Rect{X: area.X + 8, Y: y, W: area.W - 16, H: 28}
			w := i.widget(entity.ID, ci, fi, f, ctx)
			val := comp.Values[f.JSONKey]
			w.Draw(dst, fieldArea, val)
			ev := w.Update(fieldArea, val, mx, my, pressed)
			if ev != nil {
				if comp.Values == nil {
					comp.Values = map[string]any{}
					entity.Components[ci].Values = comp.Values
				}
				comp.Values[f.JSONKey] = ev.NewValue
				mutated = true
			}
			y += 30
		}
	}
	return mutated
}

// widget returns the cached widget for (entity, component, field). The
// metadata + context are passed so the widget can pick up live palette
// colors and asset lists.
func (i *Inspector) widget(entityID string, ci, fi int, f pfcomponent.FieldMetadata, ctx widgets.Context) widgets.Widget {
	key := inspectorKey{EntityID: entityID, CompIdx: ci, FieldIdx: fi}
	w, ok := i.cache[key]
	if !ok {
		w = widgets.New(f)
		i.cache[key] = w
	}
	bindContext(w, &ctx)
	return w
}

// bindContext threads the widget context into ref-style widgets. Slider/
// numeric/etc. don't need a context.
func bindContext(w widgets.Widget, ctx *widgets.Context) {
	switch x := w.(type) {
	case *widgets.SpriteRefWidget:
		x.Ctx = ctx
	case *widgets.AudioRefWidget:
		x.Ctx = ctx
	case *widgets.EventTopicWidget:
		x.Ctx = ctx
	case *widgets.ColorPickerWidget:
		x.Ctx = ctx
	}
}

// buildWidgetContext snapshots the project's catalogs into the form
// the widget layer expects.
func buildWidgetContext(p *pixelforge_project.Project) widgets.Context {
	if p == nil {
		return widgets.Context{}
	}
	ctx := widgets.Context{}
	for _, s := range p.Sprites {
		ctx.SpriteNames = append(ctx.SpriteNames, s.Name)
	}
	for _, a := range p.Audio {
		ctx.AudioNames = append(ctx.AudioNames, a.Name)
	}
	for _, sub := range p.EventSubscriptions {
		if sub.Topic != "" {
			ctx.EventTopics = append(ctx.EventTopics, sub.Topic)
		}
	}
	sort.Strings(ctx.SpriteNames)
	sort.Strings(ctx.AudioNames)
	sort.Strings(ctx.EventTopics)
	// Palette copy.
	ctx.PaletteColors = append(ctx.PaletteColors, p.Palette.Base[:]...)
	return ctx
}

func drawComponentHeader(dst *ebiten.Image, panel widgets.Rect, label string, y int) {
	r := widgets.Rect{X: panel.X + 4, Y: y, W: panel.W - 8, H: 16}
	vectorFill(dst, r, headerColor)
	debugPrint(dst, label, r.X+4, r.Y+1)
}

func drawUnknownComponentHeader(dst *ebiten.Image, panel widgets.Rect, typeName string, y int) {
	r := widgets.Rect{X: panel.X + 4, Y: y, W: panel.W - 8, H: 18}
	vectorFill(dst, r, unknownColor)
	debugPrint(dst, "(unknown component) "+typeName, r.X+4, r.Y+1)
}
