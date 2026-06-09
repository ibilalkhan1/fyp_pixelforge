// Package pixelforge_metrics renders an on-screen overlay that exposes the
// engine's runtime internals: FPS/CPU/RAM, per-phase frame budget, audio
// channel state, color-table activity, event-bus traffic, and an optional
// per-pixel write density heat map.
//
// All text and bar panels are drawn at native window resolution via the
// pixelforge_ebiten native overlay hook, so they remain crisp and readable
// regardless of how aggressively the game canvas is upscaled. The heat map
// is intentionally drawn on the canvas because it represents per-pixel
// game state and must align with the game pixels underneath.
package pixelforge_metrics

import (
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten"
	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_pad"
	pistat "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_stat"
	"math"
)

// RenderMode is a bitmask selecting which overlay panels to render.
type RenderMode uint32

const (
	// RenderTextMetrics shows the FPS / TPS / CPU / RAM / FRM text lines.
	RenderTextMetrics RenderMode = 1 << iota
	// RenderInputs shows currently held keys and gamepad buttons.
	RenderInputs
	// RenderBudget shows the per-phase frame budget bar.
	RenderBudget
	// RenderAudio shows the 4-channel Paula audio mixer dashboard.
	RenderAudio
	// RenderEventBus shows the dual event bus traffic monitor.
	RenderEventBus
	// RenderColorTable shows the color-table access matrix.
	RenderColorTable
)

// RenderAll enables every panel except the heat map (which has its own toggle).
const RenderAll = RenderTextMetrics | RenderInputs | RenderBudget |
	RenderAudio | RenderEventBus | RenderColorTable

// Mode controls which panels are rendered. Defaults to RenderAll.
//
// Set this before calling Start, or change at runtime to toggle panels.
var Mode = RenderAll

// ShowHeatMap toggles the per-pixel write density overlay.
//
// When enabled, the engine populates pixelforge.HeatMapBuffer for every
// screen pixel write; the overlay renders a colored intensity overlay
// on the game canvas before the other (native) panels are drawn.
var ShowHeatMap bool

// Scale multiplies the native overlay's font size and panel paddings.
// 1 = ebitenutil's default (~6×16px). 2 doubles everything (uses GeoM).
var Scale = 2

// Visible controls whether the metrics overlay is drawn. Toggle at runtime
// with SetVisible. Defaults to on.
var visible = true

// SetVisible shows or hides the metrics overlay while keeping it
// registered as the native overlay callback.
func SetVisible(v bool) {
	visible = v
}

var (
	fps      int
	frameCnt int
	fpsTimer time.Time

	started      bool
	heatHandler  pievent.Handler
	frameHandler pievent.Handler

	prevTargetPubs uint64
	prevDebugPubs  uint64
	pubRateTimer   time.Time
	targetPubRate  float64
	debugPubRate   float64
)

// Start enables the metrics overlay.
//
// An optional RenderMode argument selects which panels to render.
// If omitted, all panels (except the heat map) are shown.
func Start(modes ...RenderMode) {
	if started {
		return
	}
	if len(modes) > 0 {
		Mode = modes[0]
	}
	started = true
	pistat.Start()
	fpsTimer = time.Now()
	pubRateTimer = time.Now()
	prevTargetPubs = piloop.Target().PublishCount()
	prevDebugPubs = piloop.DebugTarget().PublishCount()

	frameHandler = piloop.DebugTarget().Subscribe(piloop.EventFrameStart, onFrameStart)
	heatHandler = piloop.DebugTarget().Subscribe(piloop.EventLateDraw, drawCanvasOverlays)
	pixelforge_ebiten.SetNativeOverlay(drawNative)
}

// Stop disables the metrics overlay.
func Stop() {
	if !started {
		return
	}
	pixelforge_ebiten.SetNativeOverlay(nil)
	piloop.DebugTarget().Unsubscribe(heatHandler)
	piloop.DebugTarget().Unsubscribe(frameHandler)
	started = false
}

func onFrameStart(piloop.Event, pievent.Handler) {
	pixelforge.ResetFrameCounters()
	if ShowHeatMap {
		ensureHeatMapBuffer()
		pixelforge.DecayHeatMap()
	}
}

func ensureHeatMapBuffer() {
	screen := pixelforge.Screen()
	want := screen.W() * screen.H()
	if want <= 0 {
		return
	}
	if len(pixelforge.HeatMapBuffer) != want {
		pixelforge.HeatMapBuffer = make([]uint16, want)
	}
}

// MetricsSnapshot is a single-frame view of the engine internals
// rendered by the overlay.
type MetricsSnapshot struct {
	FPS, TPS int
	CPU      int
	MemoryMB int
	Frame    int
	Allocs   uint64
	Time     float64

	InputDur, UpdateDur, DrawDur, TotalDur time.Duration

	PixelsWritten uint64

	AudioChannels [4]AudioChannelSnapshot

	TargetSubs, DebugSubs int
	TargetPubs, DebugPubs uint64
	TargetPubRate         float64
	DebugPubRate          float64
}

// AudioChannelSnapshot holds the read-only state of one audio channel.
type AudioChannelSnapshot struct {
	Active   bool
	Pitch    float64
	Volume   float64
	Position float64
}

var inputKeys = []pixelforge_key.Key{
	pixelforge_key.Left, pixelforge_key.Right, pixelforge_key.Up, pixelforge_key.Down,
	pixelforge_key.A, pixelforge_key.B, pixelforge_key.C, pixelforge_key.D,
	pixelforge_key.W, pixelforge_key.S, pixelforge_key.D, pixelforge_key.Z,
	pixelforge_key.X, pixelforge_key.Space, pixelforge_key.Enter, pixelforge_key.Esc,
	pixelforge_key.ShiftLeft, pixelforge_key.ShiftRight,
	pixelforge_key.CtrlLeft, pixelforge_key.CtrlRight,
}

var padButtons = []pixelforge_pad.Button{
	pixelforge_pad.Left, pixelforge_pad.Right, pixelforge_pad.Top, pixelforge_pad.Bottom,
	pixelforge_pad.A, pixelforge_pad.B, pixelforge_pad.X, pixelforge_pad.Y,
}

var audioChans = [4]pixelforge_audio.Chan{
	pixelforge_audio.Chan1,
	pixelforge_audio.Chan2,
	pixelforge_audio.Chan3,
	pixelforge_audio.Chan4,
}

func collectSnapshot() MetricsSnapshot {
	frameCnt++
	if time.Since(fpsTimer) >= time.Second {
		fps = frameCnt
		frameCnt = 0
		fpsTimer = time.Now()
	}

	targetPubs := piloop.Target().PublishCount()
	debugPubs := piloop.DebugTarget().PublishCount()
	if elapsed := time.Since(pubRateTimer); elapsed >= 500*time.Millisecond {
		secs := elapsed.Seconds()
		targetPubRate = float64(targetPubs-prevTargetPubs) / secs
		debugPubRate = float64(debugPubs-prevDebugPubs) / secs
		prevTargetPubs = targetPubs
		prevDebugPubs = debugPubs
		pubRateTimer = time.Now()
	}

	input, update, draw, total := pixelforge.FramePhaseDurations()

	snap := MetricsSnapshot{
		FPS:           fps,
		TPS:           pixelforge.TPS(),
		CPU:           pistat.CPU,
		MemoryMB:      pistat.MemoryMB,
		Frame:         pixelforge.Frame,
		Allocs:        pistat.Allocs,
		Time:          pixelforge.Time,
		InputDur:      input,
		UpdateDur:     update,
		DrawDur:       draw,
		TotalDur:      total,
		PixelsWritten: pixelforge.PixelsWrittenThisFrame,
		TargetSubs:    piloop.Target().SubscriberCount(),
		DebugSubs:     piloop.DebugTarget().SubscriberCount(),
		TargetPubs:    targetPubs,
		DebugPubs:     debugPubs,
		TargetPubRate: targetPubRate,
		DebugPubRate:  debugPubRate,
	}
	for i, ch := range audioChans {
		snap.AudioChannels[i] = AudioChannelSnapshot{
			Active:   pixelforge_audio.ChannelActive(ch),
			Pitch:    pixelforge_audio.ChannelPitch(ch),
			Volume:   pixelforge_audio.ChannelVolume(ch),
			Position: pixelforge_audio.ChannelPosition(ch),
		}
	}
	return snap
}

// drawCanvasOverlays draws layers that must align with game pixels
// (currently just the optional heat map).
func drawCanvasOverlays(piloop.Event, pievent.Handler) {
	if !ShowHeatMap {
		return
	}

	prevColor := pixelforge.SetColor(7)
	prevTarget := pixelforge.SetDrawTarget(pixelforge.Screen())
	prevClip := pixelforge.Clip()
	prevHeatMap := pixelforge.HeatMapBuffer
	pixelforge.HeatMapBuffer = nil
	defer func() {
		pixelforge.HeatMapBuffer = prevHeatMap
		pixelforge.SetClip(prevClip)
		pixelforge.SetDrawTarget(prevTarget)
		pixelforge.SetColor(prevColor)
	}()

	screen := pixelforge.Screen()
	drawHeatMap(prevHeatMap, screen.W(), screen.H())
}

// drawNative is invoked by pixelforge_ebiten after the game canvas has been
// scaled to the window. Everything drawn here renders at native window
// resolution, so text stays crisp.
func drawNative(screen *ebiten.Image) {
	if !started || !visible {
		return
	}

	snap := collectSnapshot()

	winW, winH := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Position text in the right column — just after the game canvas.
	gx, _, gw, _ := pixelforge_ebiten.GameCanvasBounds()
	colX := int(math.Round(gx+gw)) + 6

	const padEdge = 6
	y := padEdge

	// ---- Top-right panels ----
	if Mode&RenderTextMetrics != 0 {
		y = drawTextMetricsNative(screen, snap, colX, y, winW, winH)
	}
	if Mode&RenderInputs != 0 {
		y = drawInputsNative(screen, colX, y, winW, winH)
	}
	if Mode&RenderBudget != 0 {
		y = drawBudgetBarNative(screen, snap, colX, y, winW, winH)
	}

	// ---- Bottom-right panels ----
	// Total height needed for bottom groups.
	var bottomH int
	if Mode&RenderAudio != 0 {
		bottomH += lineH() * 4
	}
	if Mode&RenderEventBus != 0 {
		bottomH += lineH() * 3
	}
	if Mode&RenderColorTable != 0 {
		bottomH += lineH() + 4*4*Scale + 4 // title + 4 rows × 4*Scale cell
	}
	y = winH - padEdge - bottomH

	if Mode&RenderAudio != 0 {
		y = drawAudioNative(screen, snap, colX, y, winW, winH)
	}
	if Mode&RenderEventBus != 0 {
		y = drawEventBusNative(screen, snap, colX, y, winW, winH)
	}
	if Mode&RenderColorTable != 0 {
		drawColorTableNative(screen, colX, y, winW, winH)
	}
}

// printAt draws s at native window resolution.
//
// Uses ebitenutil.DebugPrintAt's built-in monospace font (no asset
// dependency), optionally upscaled by Scale via a temporary off-screen
// image so the overlay can be enlarged on hi-DPI displays without ever
// going through the pixelforge canvas.
func printAt(dst *ebiten.Image, s string, x, y int) {
	if Scale <= 1 {
		ebitenutil.DebugPrintAt(dst, s, x, y)
		return
	}
	w := len(s) * 6
	if w < 1 {
		w = 1
	}
	tmp := ebiten.NewImage(w, 16)
	ebitenutil.DebugPrintAt(tmp, s, 0, 0)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(Scale), float64(Scale))
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(tmp, op)
}

// lineH is one row of native overlay text, scaled.
func lineH() int {
	return 16 * Scale
}

// glyphW is one column of native overlay text, scaled.
func glyphW() int {
	return 6 * Scale
}

func drawTextMetricsNative(dst *ebiten.Image, snap MetricsSnapshot, x, y, winW, winH int) int {
	if y+lineH()*2 > winH {
		return y
	}
	printAt(dst, fmt.Sprintf("FPS:%-3d TPS:%-2d CPU:%5.2f%% RAM:%-6dKB",
		snap.FPS, snap.TPS, float64(snap.CPU)/100, snap.MemoryMB), x, y)
	y += lineH()
	printAt(dst, fmt.Sprintf("FRM:%-6d ALLOC:%-6d TIME:%6.1fs PIX:%-7d",
		snap.Frame, snap.Allocs, snap.Time, snap.PixelsWritten), x, y)
	y += lineH()
	return y
}

func drawInputsNative(dst *ebiten.Image, x, y, winW, winH int) int {
	heldKeys := ""
	for _, k := range inputKeys {
		if pixelforge_key.Duration(k) > 0 {
			if heldKeys != "" {
				heldKeys += " "
			}
			heldKeys += string(k)
		}
	}
	heldPad := ""
	for _, b := range padButtons {
		if pixelforge_pad.Duration(b) > 0 {
			if heldPad != "" {
				heldPad += " "
			}
			heldPad += string(b)
		}
	}
	if heldKeys != "" && y+lineH() <= winH {
		printAt(dst, "KEYS: "+heldKeys, x, y)
		y += lineH()
	}
	if heldPad != "" && y+lineH() <= winH {
		printAt(dst, "PAD:  "+heldPad, x, y)
		y += lineH()
	}
	return y
}

var (
	colDim    = color.RGBA{R: 60, G: 60, B: 60, A: 200}
	colInput  = color.RGBA{R: 80, G: 220, B: 100, A: 220}  // green
	colUpdate = color.RGBA{R: 240, G: 220, B: 60, A: 220}  // yellow
	colDraw   = color.RGBA{R: 80, G: 160, B: 240, A: 220}  // blue
	colOver   = color.RGBA{R: 230, G: 60, B: 60, A: 240}   // red
	colWhite  = color.RGBA{R: 240, G: 240, B: 240, A: 230} // off-white
	colMuted  = color.RGBA{R: 160, G: 160, B: 170, A: 230}
)

func drawBudgetBarNative(dst *ebiten.Image, snap MetricsSnapshot, x, y, winW, winH int) int {
	barH := 6 * Scale
	barW := winW / 3
	if barW < 60 {
		return y
	}
	if y+barH+lineH() > winH {
		return y
	}

	tps := snap.TPS
	if tps <= 0 {
		tps = 30
	}
	budget := time.Second / time.Duration(tps)
	total := snap.InputDur + snap.UpdateDur + snap.DrawDur

	vector.DrawFilledRect(dst, float32(x), float32(y), float32(barW), float32(barH), colDim, false)

	scale := func(d time.Duration) float32 {
		if budget <= 0 {
			return 0
		}
		return float32(barW) * float32(d) / float32(budget)
	}
	cur := float32(x)
	for _, seg := range []struct {
		w float32
		c color.RGBA
	}{
		{scale(snap.InputDur), colInput},
		{scale(snap.UpdateDur), colUpdate},
		{scale(snap.DrawDur), colDraw},
	} {
		if seg.w <= 0 {
			continue
		}
		w := seg.w
		if cur+w > float32(x+barW) {
			w = float32(x+barW) - cur
		}
		if w <= 0 {
			break
		}
		vector.DrawFilledRect(dst, cur, float32(y), w, float32(barH), seg.c, false)
		cur += w
	}

	overBudget := total > budget
	label := fmt.Sprintf("BUDGET %5.2f / %5.2f ms  in:%4.2f up:%4.2f dr:%4.2f",
		float64(total.Microseconds())/1000.0,
		float64(budget.Microseconds())/1000.0,
		float64(snap.InputDur.Microseconds())/1000.0,
		float64(snap.UpdateDur.Microseconds())/1000.0,
		float64(snap.DrawDur.Microseconds())/1000.0,
	)
	if overBudget {
		label += "  [SKIP]"
	}
	printAt(dst, label, x, y+barH+2)
	return y + barH + lineH()
}

func drawAudioNative(dst *ebiten.Image, snap MetricsSnapshot, x, y, winW, winH int) int {
	if y+lineH()*4 > winH {
		return y
	}
	vuW := 80 * Scale
	for i, ch := range snap.AudioChannels {
		ly := y + i*lineH()
		label := fmt.Sprintf("CH%d", i+1)
		labelW := len(label) * glyphW()

		barX := float32(x + labelW + 4*Scale)
		barY := float32(ly + 2*Scale)
		vector.DrawFilledRect(dst, barX, barY, float32(vuW), float32(lineH()-4*Scale), colDim, false)

		if ch.Active {
			vol := ch.Volume
			if vol < 0 {
				vol = 0
			}
			if vol > 1 {
				vol = 1
			}
			fill := float32(vuW) * float32(vol)
			c := colInput
			switch {
			case vol > 0.66:
				c = colOver
			case vol > 0.33:
				c = colUpdate
			}
			vector.DrawFilledRect(dst, barX, barY, fill, float32(lineH()-4*Scale), c, false)
			info := fmt.Sprintf("p%5.2f  v%5.2f  pos%6d", ch.Pitch, ch.Volume, int(ch.Position))
			printAt(dst, info, int(barX)+vuW+4*Scale, ly)
		} else {
			printAt(dst, "[INACTIVE]", int(barX)+vuW+4*Scale, ly)
		}
		printAt(dst, label, x, ly)
	}
	return y + lineH()*4
}

func drawEventBusNative(dst *ebiten.Image, snap MetricsSnapshot, x, y, winW, winH int) int {
	if y+lineH()*3 > winH {
		return y
	}
	printAt(dst, "EVENT BUS", x, y)
	y += lineH()
	printAt(dst, fmt.Sprintf("game  subs:%-4d pubs:%-9d %6.0f/s",
		snap.TargetSubs, snap.TargetPubs, snap.TargetPubRate), x, y)
	y += lineH()
	printAt(dst, fmt.Sprintf("debug subs:%-4d pubs:%-9d %6.0f/s",
		snap.DebugSubs, snap.DebugPubs, snap.DebugPubRate), x, y)
	y += lineH()
	return y
}

// drawColorTableNative paints a 4-row × 64-column heat map of color-table
// access frequency, with each cell rendered as a small native rectangle.
func drawColorTableNative(dst *ebiten.Image, x, y, winW, winH int) int {
	const cols = pixelforge.MaxColors
	const rows = 4
	cell := 4 * Scale
	gridW := cols * cell
	gridH := rows * cell
	if y+lineH()+gridH+4 > winH {
		return y
	}
	printAt(dst, "COLOR TABLES (4 x 64)", x, y)
	gridY := y + lineH()

	if x+gridW > winW {
		return y
	}

	var maxAccess uint64
	var sums [rows][cols]uint64
	for t := 0; t < rows; t++ {
		for d := 0; d < cols; d++ {
			var sum uint64
			for tg := 0; tg < cols; tg++ {
				sum += pixelforge.ColorTableAccesses[t][d][tg]
			}
			sums[t][d] = sum
			if sum > maxAccess {
				maxAccess = sum
			}
		}
	}
	for t := 0; t < rows; t++ {
		for d := 0; d < cols; d++ {
			c := intensityRGBA(sums[t][d], maxAccess)
			vector.DrawFilledRect(dst,
				float32(x+d*cell), float32(gridY+t*cell),
				float32(cell), float32(cell), c, false)
		}
	}
	return gridY + gridH + 4
}

// drawHeatMap renders the per-pixel write density buffer onto the game
// canvas. This intentionally stays on the canvas (not native) because the
// data is per-game-pixel and must align with the game underneath.
func drawHeatMap(buf []uint16, screenW, screenH int) {
	if len(buf) != screenW*screenH {
		return
	}
	var max uint16
	for _, v := range buf {
		if v > max {
			max = v
		}
	}
	if max == 0 {
		return
	}
	for y := 0; y < screenH; y++ {
		row := y * screenW
		for x := 0; x < screenW; x++ {
			v := buf[row+x]
			if v == 0 {
				continue
			}
			pixelforge.SetColor(canvasIntensityColor(uint64(v), uint64(max)))
			pixelforge.SetPixel(x, y)
		}
	}
}

// intensityRGBA maps v/max to an RGBA gradient (dark blue → cyan → yellow → red).
// Used for native overlay drawing.
func intensityRGBA(v, max uint64) color.RGBA {
	if max == 0 || v == 0 {
		return color.RGBA{R: 20, G: 20, B: 40, A: 200}
	}
	r := float64(v) / float64(max)
	switch {
	case r >= 0.85:
		return color.RGBA{R: 240, G: 240, B: 240, A: 230}
	case r >= 0.6:
		return color.RGBA{R: 230, G: 60, B: 60, A: 230}
	case r >= 0.35:
		return color.RGBA{R: 240, G: 160, B: 40, A: 230}
	case r >= 0.15:
		return color.RGBA{R: 240, G: 220, B: 40, A: 230}
	default:
		return color.RGBA{R: 60, G: 120, B: 220, A: 230}
	}
}

// canvasIntensityColor returns the engine palette index used by the
// canvas-level heat map overlay.
func canvasIntensityColor(v, max uint64) pixelforge.Color {
	if max == 0 || v == 0 {
		return 1
	}
	ratio := float64(v) / float64(max)
	switch {
	case ratio >= 0.85:
		return 7
	case ratio >= 0.6:
		return 8
	case ratio >= 0.35:
		return 9
	case ratio >= 0.15:
		return 10
	default:
		return 12
	}
}
