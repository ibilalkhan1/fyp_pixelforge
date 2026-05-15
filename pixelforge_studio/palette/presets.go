package palette

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// PresetStack is the Lightroom-style preset list. Each entry is a
// `ColorTablePreset` with a toggleable on/off state. Active presets
// compose on top of the base palette in declaration order; later
// active presets win on overlapping slots.
type PresetStack struct {
	active map[int]bool // map[idx-in-Presets]bool
}

// NewPresetStack returns a fresh stack with no presets active.
func NewPresetStack() *PresetStack {
	return &PresetStack{active: map[int]bool{}}
}

// Toggle flips the active flag for preset idx.
func (s *PresetStack) Toggle(idx int) {
	s.active[idx] = !s.active[idx]
}

// IsActive reports whether preset idx is currently active.
func (s *PresetStack) IsActive(idx int) bool { return s.active[idx] }

// Add appends a new empty preset to the project and marks it active.
// Returns the new preset's index.
func (s *PresetStack) Add(p *pixelforge_project.Project, name string) int {
	preset := pixelforge_project.ColorTablePreset{
		Name:             name,
		PaletteOverrides: map[int]string{},
	}
	p.Palette.Presets = append(p.Palette.Presets, preset)
	idx := len(p.Palette.Presets) - 1
	s.active[idx] = true
	return idx
}

// Remove drops preset idx from the project.
func (s *PresetStack) Remove(p *pixelforge_project.Project, idx int) {
	if idx < 0 || idx >= len(p.Palette.Presets) {
		return
	}
	p.Palette.Presets = append(p.Palette.Presets[:idx], p.Palette.Presets[idx+1:]...)
	// Re-key active map.
	newActive := map[int]bool{}
	for k, v := range s.active {
		if k == idx {
			continue
		}
		if k > idx {
			newActive[k-1] = v
		} else {
			newActive[k] = v
		}
	}
	s.active = newActive
}

// Rename updates preset idx's name. Idempotent if idx is out of range.
func (s *PresetStack) Rename(p *pixelforge_project.Project, idx int, name string) {
	if idx < 0 || idx >= len(p.Palette.Presets) {
		return
	}
	p.Palette.Presets[idx].Name = name
}

// Compose returns a copy of p whose Palette.Base + ColorTables have the
// active presets layered on top, in declaration order. The original p
// is left untouched. Callers that want to keep editing the *underlying*
// palette can read from the returned snapshot for display while writing
// back to p directly.
func (s *PresetStack) Compose(p *pixelforge_project.Project) *pixelforge_project.Project {
	if p == nil {
		return nil
	}
	out := *p
	out.Palette = p.Palette
	// Apply presets in order. Out-of-range slot indices are silently
	// skipped; the preset row surfaces a warning marker for those.
	for idx, preset := range p.Palette.Presets {
		if !s.active[idx] {
			continue
		}
		for slot, hex := range preset.PaletteOverrides {
			if slot < 0 || slot >= pixelforge_project.MaxColors {
				continue
			}
			out.Palette.Base[slot] = hex
		}
		for _, ov := range preset.ColorTableOverrides {
			if ov.Table < 0 || ov.Table >= 4 {
				continue
			}
			if ov.Source < 0 || ov.Source >= pixelforge_project.MaxColors {
				continue
			}
			if ov.Target < 0 || ov.Target >= pixelforge_project.MaxColors {
				continue
			}
			out.Palette.ColorTables[ov.Table][ov.Source][ov.Target] = ov.Value
		}
	}
	return &out
}

// Update routes input to the preset list.
func (s *PresetStack) Update(area widgets.Rect, p *pixelforge_project.Project, e *editor.Editor) bool {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return false
	}
	mx, my := ebiten.CursorPosition()
	// New preset button at the bottom.
	newBtn := widgets.Rect{X: area.X + 4, Y: area.Y + area.H - presetRowH - 4, W: area.W - 8, H: presetRowH}
	if newBtn.Contains(mx, my) {
		idx := s.Add(p, presetName(len(p.Palette.Presets)))
		if e != nil {
			e.MarkDirty()
			e.SetStatusMessage("preset: " + p.Palette.Presets[idx].Name)
		}
		return true
	}
	// Row hit-test.
	for i := range p.Palette.Presets {
		row := s.rowRect(area, i)
		if !row.Contains(mx, my) {
			continue
		}
		// Trash icon on the right.
		trash := widgets.Rect{X: row.X + row.W - 24, Y: row.Y + 4, W: 16, H: presetRowH - 8}
		if trash.Contains(mx, my) {
			s.Remove(p, i)
			if e != nil {
				e.MarkDirty()
			}
			return true
		}
		// Otherwise toggle.
		s.Toggle(i)
		if e != nil {
			e.MarkDirty()
		}
		return true
	}
	return false
}

// Draw paints the preset list and the "New Preset" button.
func (s *PresetStack) Draw(dst *ebiten.Image, area widgets.Rect, p *pixelforge_project.Project) {
	vector.DrawFilledRect(dst, float32(area.X), float32(area.Y), float32(area.W), float32(area.H), colPresetsBg, false)
	for i, preset := range p.Palette.Presets {
		row := s.rowRect(area, i)
		bg := colPresetRow
		if s.IsActive(i) {
			bg = colPresetRowActive
		}
		vector.DrawFilledRect(dst, float32(row.X), float32(row.Y), float32(row.W), float32(row.H), bg, false)

		box := widgets.Rect{X: row.X + 6, Y: row.Y + (row.H-12)/2, W: 12, H: 12}
		vector.StrokeRect(dst, float32(box.X), float32(box.Y), float32(box.W), float32(box.H), 1, colPresetCheckBox, false)
		if s.IsActive(i) {
			vector.DrawFilledRect(dst, float32(box.X+2), float32(box.Y+2), float32(box.W-4), float32(box.H-4), colPresetCheckMark, false)
		}
		ebitenutilPrint(dst, preset.Name, row.X+24, row.Y+8)

		// Warning for out-of-range overrides.
		if s.presetHasWarning(preset) {
			ebitenutilPrint(dst, "!", row.X+row.W-40, row.Y+8)
		}

		trash := widgets.Rect{X: row.X + row.W - 24, Y: row.Y + 4, W: 16, H: presetRowH - 8}
		vector.DrawFilledRect(dst, float32(trash.X), float32(trash.Y), float32(trash.W), float32(trash.H), colPresetTrash, false)
		ebitenutilPrint(dst, "X", trash.X+5, trash.Y+5)
	}
	newBtn := widgets.Rect{X: area.X + 4, Y: area.Y + area.H - presetRowH - 4, W: area.W - 8, H: presetRowH}
	vector.DrawFilledRect(dst, float32(newBtn.X), float32(newBtn.Y), float32(newBtn.W), float32(newBtn.H), colPresetAddBtn, false)
	ebitenutilPrint(dst, "+ New Preset", newBtn.X+8, newBtn.Y+8)
}

func (s *PresetStack) rowRect(area widgets.Rect, i int) widgets.Rect {
	return widgets.Rect{X: area.X + 4, Y: area.Y + 4 + i*presetRowH, W: area.W - 8, H: presetRowH - 2}
}

// presetHasWarning reports whether the preset references an
// out-of-range slot or cell. Used to surface the "!" marker.
func (s *PresetStack) presetHasWarning(p pixelforge_project.ColorTablePreset) bool {
	for slot := range p.PaletteOverrides {
		if slot < 0 || slot >= pixelforge_project.MaxColors {
			return true
		}
	}
	for _, ov := range p.ColorTableOverrides {
		if ov.Table < 0 || ov.Table >= 4 {
			return true
		}
		if ov.Source < 0 || ov.Source >= pixelforge_project.MaxColors {
			return true
		}
		if ov.Target < 0 || ov.Target >= pixelforge_project.MaxColors {
			return true
		}
	}
	return false
}

func presetName(n int) string {
	return "Preset " + itoa(n+1)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

const presetRowH = 30

var (
	colPresetsBg       = color.RGBA{R: 0x18, G: 0x18, B: 0x22, A: 0xff}
	colPresetRow       = color.RGBA{R: 0x1f, G: 0x1f, B: 0x29, A: 0xff}
	colPresetRowActive = color.RGBA{R: 0x2a, G: 0x2a, B: 0x40, A: 0xff}
	colPresetCheckBox  = color.RGBA{R: 0x88, G: 0x88, B: 0x95, A: 0xff}
	colPresetCheckMark = color.RGBA{R: 0x46, G: 0x86, B: 0xff, A: 0xff}
	colPresetTrash     = color.RGBA{R: 0x55, G: 0x33, B: 0x22, A: 0xff}
	colPresetAddBtn    = color.RGBA{R: 0x2a, G: 0x44, B: 0x66, A: 0xff}
)
