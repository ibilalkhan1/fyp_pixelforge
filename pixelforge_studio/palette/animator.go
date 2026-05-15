package palette

import (
	"fmt"
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// Animator is the palette-slot animation timeline popover. M2 stores
// keyframes + easing; the runtime driver (Steps) lands with M5 — the
// animator's preview is an inline interpolation called from Draw.
type Animator struct {
	visible    bool
	slot       int
	playing    bool
	playT      float64
	clipLength float64

	// pickerOpen, when true, surfaces a small RGB picker overlay anchored
	// to the keyframe the user clicked.
	pickerOpen   bool
	editingKfIdx int
	picker       rgbPicker
}

// NewAnimator returns a hidden animator ready to OpenForSlot().
func NewAnimator() *Animator {
	return &Animator{clipLength: 2.0}
}

// Visible reports whether the timeline is open.
func (a *Animator) Visible() bool { return a.visible }

// Slot returns the slot the timeline currently targets.
func (a *Animator) Slot() int { return a.slot }

// OpenForSlot opens the animator targeting the given palette slot.
func (a *Animator) OpenForSlot(slot int) {
	if slot < 0 || slot >= pixelforge_project.MaxColors {
		return
	}
	a.slot = slot
	a.visible = true
	a.playing = false
	a.playT = 0
}

// Close hides the animator.
func (a *Animator) Close() {
	a.visible = false
	a.playing = false
	a.pickerOpen = false
}

// Animation returns the project's animation entry for the active slot,
// or nil when no animation exists yet.
func (a *Animator) Animation(p *pixelforge_project.Project) *pixelforge_project.PaletteAnimation {
	for i := range p.Palette.Animations {
		if p.Palette.Animations[i].Slot == a.slot {
			return &p.Palette.Animations[i]
		}
	}
	return nil
}

// AddKeyframe inserts (t, color) at the active slot's animation,
// creating the animation if necessary.
func (a *Animator) AddKeyframe(p *pixelforge_project.Project, t float64, hex string) {
	anim := a.Animation(p)
	if anim == nil {
		p.Palette.Animations = append(p.Palette.Animations, pixelforge_project.PaletteAnimation{
			Slot:      a.slot,
			Keyframes: []pixelforge_project.PaletteKeyframe{},
			Easing:    "linear",
			Loop:      true,
		})
		anim = &p.Palette.Animations[len(p.Palette.Animations)-1]
	}
	anim.Keyframes = append(anim.Keyframes, pixelforge_project.PaletteKeyframe{Time: t, Color: hex})
	sort.SliceStable(anim.Keyframes, func(i, j int) bool { return anim.Keyframes[i].Time < anim.Keyframes[j].Time })
}

// SetEasing updates the active animation's easing curve.
func (a *Animator) SetEasing(p *pixelforge_project.Project, easing string) {
	anim := a.Animation(p)
	if anim == nil {
		return
	}
	anim.Easing = easing
}

// PreviewAt returns the interpolated color for the active animation at
// time t. Returns the base slot color when no keyframes exist.
func (a *Animator) PreviewAt(p *pixelforge_project.Project, t float64) string {
	anim := a.Animation(p)
	if anim == nil || len(anim.Keyframes) == 0 {
		return p.Palette.Base[a.slot]
	}
	// Identify bracketing keyframes.
	kfs := anim.Keyframes
	if t <= kfs[0].Time {
		return kfs[0].Color
	}
	if t >= kfs[len(kfs)-1].Time {
		return kfs[len(kfs)-1].Color
	}
	var lo, hi pixelforge_project.PaletteKeyframe
	for i := 1; i < len(kfs); i++ {
		if kfs[i].Time >= t {
			lo = kfs[i-1]
			hi = kfs[i]
			break
		}
	}
	if hi.Time == lo.Time {
		return lo.Color
	}
	if anim.Easing == "step" {
		return lo.Color
	}
	progress := (t - lo.Time) / (hi.Time - lo.Time)
	progress = applyEasing(anim.Easing, progress)
	c1, _ := parseHexColor(lo.Color)
	c2, _ := parseHexColor(hi.Color)
	r := uint8(float64(c1.R) + progress*float64(int(c2.R)-int(c1.R)))
	g := uint8(float64(c1.G) + progress*float64(int(c2.G)-int(c1.G)))
	b := uint8(float64(c1.B) + progress*float64(int(c2.B)-int(c1.B)))
	return formatHexColor(int(r), int(g), int(b))
}

// SetClipLength resizes the timeline.
func (a *Animator) SetClipLength(seconds float64) {
	if seconds <= 0 {
		return
	}
	a.clipLength = seconds
}

// ClipLength returns the current clip length in seconds.
func (a *Animator) ClipLength() float64 { return a.clipLength }

// Update routes input to the animator UI.
func (a *Animator) Update(area widgets.Rect, p *pixelforge_project.Project, e *editor.Editor) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.Close()
		return
	}
	mx, my := ebiten.CursorPosition()
	timeline := a.timelineRect(area)
	playBtn := a.playBtnRect(area)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		switch {
		case playBtn.Contains(mx, my):
			a.playing = !a.playing
			a.playT = 0
		case timeline.Contains(mx, my):
			t := a.timeAt(timeline, mx)
			a.AddKeyframe(p, t, p.Palette.Base[a.slot])
			if e != nil {
				e.MarkDirty()
			}
		case !area.Contains(mx, my):
			a.Close()
		}
	}
	if a.playing {
		a.playT += 1.0 / 60.0
		if a.playT > a.clipLength {
			anim := a.Animation(p)
			if anim != nil && anim.Loop {
				a.playT = 0
			} else {
				a.playing = false
				a.playT = a.clipLength
			}
		}
	}
}

// Draw paints the timeline popover.
func (a *Animator) Draw(dst *ebiten.Image, area widgets.Rect, p *pixelforge_project.Project) {
	vector.DrawFilledRect(dst, float32(area.X), float32(area.Y), float32(area.W), float32(area.H), colAnimatorBg, false)
	vector.StrokeRect(dst, float32(area.X), float32(area.Y), float32(area.W), float32(area.H), 1, colAnimatorBorder, false)
	ebitenutilPrint(dst, fmt.Sprintf("Animate slot %d", a.slot), area.X+8, area.Y+6)

	timeline := a.timelineRect(area)
	vector.DrawFilledRect(dst, float32(timeline.X), float32(timeline.Y), float32(timeline.W), float32(timeline.H), colAnimatorTimeline, false)
	// Tick marks every 0.5s.
	for t := 0.0; t <= a.clipLength; t += 0.5 {
		x := timeline.X + int(float64(timeline.W)*(t/a.clipLength))
		vector.DrawFilledRect(dst, float32(x), float32(timeline.Y), 1, float32(timeline.H), colAnimatorTick, false)
	}
	anim := a.Animation(p)
	if anim != nil {
		for _, kf := range anim.Keyframes {
			x := timeline.X + int(float64(timeline.W)*(kf.Time/a.clipLength))
			c, _ := parseHexColor(kf.Color)
			vector.DrawFilledRect(dst, float32(x-3), float32(timeline.Y+timeline.H/2-3), 6, 6, c, false)
		}
	}
	if a.playing {
		x := timeline.X + int(float64(timeline.W)*(a.playT/a.clipLength))
		vector.DrawFilledRect(dst, float32(x), float32(timeline.Y), 1, float32(timeline.H), colAnimatorPlayhead, false)
	}

	play := a.playBtnRect(area)
	vector.DrawFilledRect(dst, float32(play.X), float32(play.Y), float32(play.W), float32(play.H), colAnimatorPlay, false)
	label := "▶"
	if a.playing {
		label = "■"
	}
	ebitenutilPrint(dst, label, play.X+12, play.Y+4)
}

func (a *Animator) timelineRect(area widgets.Rect) widgets.Rect {
	return widgets.Rect{X: area.X + 16, Y: area.Y + 60, W: area.W - 60, H: 28}
}

func (a *Animator) playBtnRect(area widgets.Rect) widgets.Rect {
	return widgets.Rect{X: area.X + area.W - 36, Y: area.Y + 60, W: 28, H: 28}
}

func (a *Animator) timeAt(timeline widgets.Rect, mx int) float64 {
	if timeline.W <= 0 {
		return 0
	}
	t := float64(mx-timeline.X) / float64(timeline.W) * a.clipLength
	if t < 0 {
		t = 0
	}
	if t > a.clipLength {
		t = a.clipLength
	}
	return t
}

// applyEasing maps a linear progress 0..1 through the named easing
// curve. Unknown names fall back to linear.
func applyEasing(easing string, x float64) float64 {
	switch easing {
	case "ease_in":
		return x * x
	case "ease_out":
		return 1 - (1-x)*(1-x)
	case "ease_in_out":
		if x < 0.5 {
			return 2 * x * x
		}
		return 1 - 2*(1-x)*(1-x)
	case "step":
		return 0
	default:
		return x
	}
}

var (
	colAnimatorBg       = color.RGBA{R: 0x18, G: 0x18, B: 0x22, A: 0xff}
	colAnimatorBorder   = color.RGBA{R: 0x44, G: 0x44, B: 0x50, A: 0xff}
	colAnimatorTimeline = color.RGBA{R: 0x10, G: 0x10, B: 0x18, A: 0xff}
	colAnimatorTick     = color.RGBA{R: 0x44, G: 0x44, B: 0x4c, A: 0xff}
	colAnimatorPlayhead = color.RGBA{R: 0x46, G: 0x86, B: 0xff, A: 0xff}
	colAnimatorPlay     = color.RGBA{R: 0x2a, G: 0x2a, B: 0x40, A: 0xff}
)
