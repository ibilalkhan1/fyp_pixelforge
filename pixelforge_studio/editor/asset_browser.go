package editor

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// AssetBrowser is the left-panel asset list. It renders the project's
// sprites and audio samples and dispatches clicks to the editor's
// selection state.
type AssetBrowser struct {
	scrollOffset int

	// thumbnails caches decoded sprite previews keyed by relative path.
	// Invalidated when a new project is loaded.
	thumbnails map[string]*ebiten.Image
}

// NewAssetBrowser returns a fresh browser.
func NewAssetBrowser() *AssetBrowser {
	return &AssetBrowser{thumbnails: map[string]*ebiten.Image{}}
}

// InvalidateCache drops any cached thumbnails. Called by Editor.SetProject.
func (b *AssetBrowser) InvalidateCache() {
	b.thumbnails = map[string]*ebiten.Image{}
}

// Sprites returns the sprite slice the browser should render, in
// declaration order. Pure helper for tests.
func (b *AssetBrowser) Sprites(p *pixelforge_project.Project) []pixelforge_project.SpriteAsset {
	if p == nil {
		return nil
	}
	return p.Sprites
}

// Audio returns the audio slice the browser should render.
func (b *AssetBrowser) Audio(p *pixelforge_project.Project) []pixelforge_project.AudioSample {
	if p == nil {
		return nil
	}
	return p.Audio
}

// HitTestSprite returns the sprite name at the supplied window-space
// row index, or "" when the row falls outside the sprite list.
func (b *AssetBrowser) HitTestSprite(p *pixelforge_project.Project, idx int) string {
	sprites := b.Sprites(p)
	if idx < 0 || idx >= len(sprites) {
		return ""
	}
	return sprites[idx].Name
}

// HitTestAudio returns the audio sample name at the supplied index.
func (b *AssetBrowser) HitTestAudio(p *pixelforge_project.Project, idx int) string {
	audio := b.Audio(p)
	if idx < 0 || idx >= len(audio) {
		return ""
	}
	return audio[idx].Name
}

// Draw paints the browser into area. Returns nothing — selection state
// is mutated on the editor via Update.
func (b *AssetBrowser) Draw(dst *ebiten.Image, area widgets.Rect, e *Editor) {
	if area.W <= 0 || area.H <= 0 {
		return
	}
	sprites := b.Sprites(e.project)
	audio := b.Audio(e.project)

	y := area.Y
	// "Sprites" header.
	header := widgets.Rect{X: area.X, Y: y, W: area.W, H: assetHeaderH}
	vectorFill(dst, header, colAssetHeader)
	debugPrint(dst, "Sprites", area.X+8, y+2)
	y += assetHeaderH

	if len(sprites) == 0 && len(audio) == 0 {
		debugPrint(dst, "No assets — File → Import to add", area.X+8, y+8)
		return
	}

	for i, s := range sprites {
		row := widgets.Rect{X: area.X, Y: y - b.scrollOffset, W: area.W, H: assetRowH}
		if !rectIntersectsRect(row, area) {
			y += assetRowH
			continue
		}
		drawAssetSpriteRow(dst, row, &sprites[i], s.Name == e.SelectedSpriteName())
		y += assetRowH
	}

	// "Audio" header.
	hdr2 := widgets.Rect{X: area.X, Y: y - b.scrollOffset, W: area.W, H: assetHeaderH}
	vectorFill(dst, hdr2, colAssetHeader)
	debugPrint(dst, "Audio", area.X+8, hdr2.Y+2)
	y += assetHeaderH

	for i, a := range audio {
		row := widgets.Rect{X: area.X, Y: y - b.scrollOffset, W: area.W, H: assetRowH}
		drawAssetAudioRow(dst, row, &audio[i], a.Name == e.SelectedAudioName())
		y += assetRowH
	}
}

// Update routes mouse + scroll input. area is the panel rect.
func (b *AssetBrowser) Update(area widgets.Rect, e *Editor) {
	if area.W <= 0 || area.H <= 0 {
		return
	}
	mx, my := ebiten.CursorPosition()
	if !area.Contains(mx, my) {
		return
	}
	// Mouse wheel scroll.
	_, wy := ebiten.Wheel()
	if wy != 0 {
		b.scrollOffset -= int(wy * 12)
		if b.scrollOffset < 0 {
			b.scrollOffset = 0
		}
	}

	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	// Resolve which row was clicked.
	sprites := b.Sprites(e.project)
	audio := b.Audio(e.project)
	rowY := area.Y - b.scrollOffset

	// Skip "Sprites" header.
	rowY += assetHeaderH
	for i, s := range sprites {
		row := widgets.Rect{X: area.X, Y: rowY + i*assetRowH, W: area.W, H: assetRowH}
		if row.Contains(mx, my) {
			e.SetSelectedSpriteName(s.Name)
			e.SetStatusMessage("sprite: " + s.Name)
			return
		}
	}
	rowY += len(sprites) * assetRowH
	// "Audio" header.
	rowY += assetHeaderH
	for i, a := range audio {
		row := widgets.Rect{X: area.X, Y: rowY + i*assetRowH, W: area.W, H: assetRowH}
		if row.Contains(mx, my) {
			e.SetSelectedAudioName(a.Name)
			e.SetStatusMessage("audio: " + a.Name)
			return
		}
	}
}

func drawAssetSpriteRow(dst *ebiten.Image, row widgets.Rect, s *pixelforge_project.SpriteAsset, selected bool) {
	bg := colAssetRow
	if selected {
		bg = colAssetRowSelected
	}
	vectorFill(dst, row, bg)
	// Tiny placeholder thumbnail (we don't decode the PNG from disk at
	// this layer — the M2 importer or M3 dogfood pass will fill in).
	thumb := widgets.Rect{X: row.X + 4, Y: row.Y + 4, W: 36, H: assetRowH - 8}
	vectorFill(dst, thumb, color.RGBA{R: 0x44, G: 0x44, B: 0x4c, A: 0xff})
	debugPrint(dst, s.Name, row.X+44, row.Y+4)
	if s.FrameW > 0 && s.FrameH > 0 {
		dims := formatDim(s.Width, s.Height, s.FrameW, s.FrameH)
		debugPrint(dst, dims, row.X+44, row.Y+20)
	}
}

func drawAssetAudioRow(dst *ebiten.Image, row widgets.Rect, a *pixelforge_project.AudioSample, selected bool) {
	bg := colAssetRow
	if selected {
		bg = colAssetRowSelected
	}
	vectorFill(dst, row, bg)
	icon := widgets.Rect{X: row.X + 4, Y: row.Y + 10, W: assetRowH - 20, H: assetRowH - 20}
	vectorFill(dst, icon, color.RGBA{R: 0x46, G: 0x86, B: 0xff, A: 0xff})
	debugPrint(dst, a.Name, row.X+44, row.Y+4)
	if a.SampleRateHz > 0 {
		debugPrint(dst, formatRate(a.SampleRateHz), row.X+44, row.Y+20)
	}
}

func formatDim(w, h, fw, fh int) string {
	return itoa(w) + "x" + itoa(h) + " " + itoa(fw) + "x" + itoa(fh)
}

func formatRate(hz int) string {
	return itoa(hz) + " Hz"
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

// rectIntersectsRect reports whether r overlaps area.
func rectIntersectsRect(r, area widgets.Rect) bool {
	if r.X+r.W <= area.X || r.X >= area.X+area.W {
		return false
	}
	if r.Y+r.H <= area.Y || r.Y >= area.Y+area.H {
		return false
	}
	return true
}

const (
	assetRowH    = 32
	assetHeaderH = 18
)

var (
	colAssetHeader      = color.RGBA{R: 0x28, G: 0x28, B: 0x34, A: 0xff}
	colAssetRow         = color.RGBA{R: 0x1a, G: 0x1a, B: 0x22, A: 0xff}
	colAssetRowSelected = color.RGBA{R: 0x2a, G: 0x2a, B: 0x40, A: 0xff}
)

// (U9 deleted the cart-resident AssetBrowser.DrawCanvas — the asset
// browser renders through its native Draw path against the
// PanelRect("Assets") rect ImGui carves out each frame.)
