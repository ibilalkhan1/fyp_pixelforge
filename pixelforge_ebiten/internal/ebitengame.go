package internal

import (
	ebitenaudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten/internal/audio"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten/internal/input"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge"
	piaudio "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	pidebug "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_debug"
	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
)

func RunEbitenGame() *EbitenGame {
	screen := pixelforge.Screen()

	ctx := ebitenaudio.NewContext(audio.CtxSampleRate)
	theAudioBackend := audio.StartAudioBackend(ctx)
	piaudio.Backend = theAudioBackend

	game := &EbitenGame{
		piScreen:       screen,
		ebitenScreen:   ebiten.NewImage(screen.W(), screen.H()),
		drawScreenOpts: &ebiten.DrawImageOptions{},
		audioBackend:   theAudioBackend,
	}
	game.inputBackend = &input.Backend{
		Paused:     &game.paused,
		LeftOffset: &game.left,
		TopOffset:  &game.top,
		Scale:      &game.scale,
	}

	pidebug.Target().SubscribeAll(game.onPidebugEvent)

	return game
}

// get the monitor once per second and cache it.
//
// This is not a strong optimization, since Ebitengine itself
// performs syscalls on each call to ebiten's Draw.
type cachedMonitor struct {
	monitor       *ebiten.MonitorType
	lastCheckTime time.Time
}

func (c *cachedMonitor) Get() *ebiten.MonitorType {
	if time.Since(c.lastCheckTime) > time.Second {
		c.monitor = ebiten.Monitor()
		c.lastCheckTime = time.Now()
	}
	return c.monitor
}

// TODO split into multiple objects to reduce complexity
type EbitenGame struct {
	piScreen       pixelforge.Canvas
	ebitenScreen   *ebiten.Image
	drawScreenOpts *ebiten.DrawImageOptions
	cachedMonitor  cachedMonitor
	started        bool

	scale     float64
	left, top float64

	// When true, indicates that the frame was rendered by pixelforge.Draw
	// and should be displayed on the next Draw call.
	dirty bool

	// When true, indicates that the last update+draw cycle
	// exceeded the tick duration of 1/TPS (e.g., 33 ms for TPS=30).
	skipNextDraw bool

	paused bool

	windowState  windowState
	audioBackend *audio.Backend
	inputBackend *input.Backend

	ebitenFrame int // frame incremented on each Ebiten tick

	// tickCounter is the monotonic Pixelforge-tick counter passed to
	// TickHook when the U4 single-render-path seam is installed.
	// Distinct from ebitenFrame (which counts Ebiten frames at
	// ebitenTPS) and from pixelforge.Frame (which is bumped at
	// pixelforge.TPS). Bumped once per Pixelforge tick, only when
	// TickHook is non-nil and not paused, so the counter the player
	// binary sees aligns with the recorded tick numbers in
	// .trace.jsonl files (U5).
	tickCounter uint64
}

func (g *EbitenGame) Update() error {
	if ebiten.IsWindowBeingClosed() {
		piloop.Target().Publish(piloop.EventWindowClose)
		return ebiten.Termination
	}

	g.windowState.store()

	g.audioBackend.OnBeforeUpdate()

	started := time.Now()

	if !g.started {
		if pixelforge.Init != nil {
			pixelforge.Init()
		}
		piloop.Target().Publish(piloop.EventInit)
	}
	g.started = true

	if g.ebitenFrame%(ebitenTPS/pixelforge.TPS()) == 0 {
		if !g.paused {
			piloop.Target().Publish(piloop.EventFrameStart)
		}
		piloop.DebugTarget().Publish(piloop.EventFrameStart)
	}

	inputStart := time.Now()
	g.inputBackend.Update()
	inputDur := time.Since(inputStart)

	var updateDur, drawDur time.Duration

	if g.ebitenFrame%(ebitenTPS/pixelforge.TPS()) == (ebitenTPS/pixelforge.TPS())-1 {
		updateStart := time.Now()
		// Idea #6 v1 U3 — the package-level pause gate freezes
		// EventUpdate / EventLateUpdate dispatch so menu overlays
		// can hold the underlying scene still while remaining
		// responsive.
		gatePaused := piloop.IsPaused()

		// arcade-shipping U4 — if a TickHook is installed (the
		// pixelforge-player binary installs one that delegates to
		// pixelforge_render.RenderTickAtScreen), it replaces the
		// default update + draw + event sequence with the unified
		// "one render path". The hook is responsible for publishing
		// the Update/LateUpdate/Draw/LateDraw events itself; we
		// only mediate the pause gate + dirty flag bookkeeping.
		if TickHook != nil {
			if !g.paused && !gatePaused {
				TickHook(uint64(g.tickCounter))
				g.tickCounter++
			}
			piloop.DebugTarget().Publish(piloop.EventUpdate)
			piloop.DebugTarget().Publish(piloop.EventLateUpdate)
			updateDur = time.Since(updateStart)

			if !g.skipNextDraw {
				drawStart := time.Now()
				piloop.DebugTarget().Publish(piloop.EventDraw)
				piloop.DebugTarget().Publish(piloop.EventLateDraw)
				drawDur = time.Since(drawStart)
				g.dirty = true
			} else {
				g.skipNextDraw = false
			}
		} else {
			if !g.paused && !gatePaused {
				pixelforge.Update()
				piloop.Target().Publish(piloop.EventUpdate)
			}
			piloop.DebugTarget().Publish(piloop.EventUpdate)

			if !g.paused && !gatePaused {
				piloop.Target().Publish(piloop.EventLateUpdate)
			}
			piloop.DebugTarget().Publish(piloop.EventLateUpdate)
			updateDur = time.Since(updateStart)

			if !g.skipNextDraw {
				drawStart := time.Now()
				if !g.paused {
					pixelforge.Draw()
					piloop.Target().Publish(piloop.EventDraw)
				}
				piloop.DebugTarget().Publish(piloop.EventDraw)

				if !g.paused {
					piloop.Target().Publish(piloop.EventLateDraw)
				}
				piloop.DebugTarget().Publish(piloop.EventLateDraw)
				drawDur = time.Since(drawStart)

				g.dirty = true
			} else {
				g.skipNextDraw = false
			}
		}

		if time.Since(started).Seconds() > 1/float64(pixelforge.TPS()) {
			g.skipNextDraw = true // game is too slow. Try to keep up by discarding next pixelforge.Draw()
		}

		pixelforge.Time += 1.0 / float64(pixelforge.TPS())
		pixelforge.Frame++

		pixelforge.SetFramePhaseDurations(inputDur, updateDur, drawDur, time.Since(started))
	}

	g.audioBackend.OnAfterUpdate()

	g.ebitenFrame++

	return nil
}

// NativeOverlay is an optional callback invoked after the scaled game frame
// is drawn to the window. Implementations should render directly to screen
// at native window resolution (e.g. via ebitenutil.DebugPrintAt or
// ebiten/text), which keeps the overlay readable even when the game canvas
// is heavily upscaled.
var NativeOverlay func(screen *ebiten.Image)

// Per-side border widths (device-independent pixels) reserved around the
// game canvas. The right border is wider so the metrics overlay has a
// dedicated column for its text panels.
var (
	MetricsBorderTop    = 60.0
	MetricsBorderBottom = 60.0
	MetricsBorderLeft   = 60.0
	MetricsBorderRight  = 300.0
)

// GameCanvasX, GameCanvasY, GameCanvasW, GameCanvasH expose where the game
// canvas was drawn (in window coordinates, before device-scale adjustment).
// The metrics overlay reads these to position text in the border area.
var (
	GameCanvasX, GameCanvasY float64
	GameCanvasW, GameCanvasH float64
)

// TickHook is the optional "single render path" seam introduced by
// arcade-shipping U4. When non-nil, EbitenGame.Update calls TickHook
// instead of running the default pixelforge.Update + pixelforge.Draw +
// event-publish sequence; the hook receives the monotonic tick
// counter and is expected to advance + render game state by exactly
// one tick (typically by delegating to
// pixelforge_render.RenderTickAtScreen against the runtime it
// captured at install time).
//
// When nil, EbitenGame retains its legacy update/draw behaviour
// unchanged — every existing caller of pixelforge_ebiten.Run that
// does not opt into the seam gets identical pixel output to before
// U4.
//
// The hook owns the per-tick draw responsibility: it must populate
// pixelforge.Screen() before returning so the subsequent
// CopyCanvasToEbitenImage call in Draw produces a non-blank frame.
// In practice the hook delegates to RenderTickAtScreen, which calls
// pixelforge.Draw internally — so the canvas ends up populated the
// same way the legacy path produced it.
var TickHook func(tick uint64)

func (g *EbitenGame) Draw(screen *ebiten.Image) {
	if g.dirty { // draw only when needed to avoid CPU load on monitors >30 Hz
		g.dirty = false

		screen.Fill(color.Black) // black border around the game canvas

		CopyCanvasToEbitenImage(pixelforge.Screen(), g.ebitenScreen)

		screen.DrawImage(g.ebitenScreen, g.drawScreenOpts)
	}

	if NativeOverlay != nil {
		NativeOverlay(screen)
	}
}

func (g *EbitenGame) LayoutF(outsideWidth, outsideHeight float64) (screenWidth, screenHeight float64) {
	piScrW, piScrH := float64(g.piScreen.W()), float64(g.piScreen.H())

	monitor := g.cachedMonitor.Get()
	deviceScaleFactor := monitor.DeviceScaleFactor()
	realWith := outsideWidth * deviceScaleFactor
	realHeight := outsideHeight * deviceScaleFactor

	// Reserve per-side border space. The right border acts as a
	// dedicated column for the metrics overlay panels.
	left := MetricsBorderLeft * deviceScaleFactor
	right := MetricsBorderRight * deviceScaleFactor
	top := MetricsBorderTop * deviceScaleFactor
	bottom := MetricsBorderBottom * deviceScaleFactor
	availW := realWith - left - right
	availH := realHeight - top - bottom

	widthRatio := availW / piScrW
	heightRatio := availH / piScrH
	scale := math.Floor(min(widthRatio, heightRatio))
	if scale < 1 {
		scale = 1
	}

	screenWidth = realWith
	screenHeight = realHeight

	g.scale = scale
	// center within the available area (inside the borders):
	g.left = left + (availW-piScrW*scale)/2.0
	g.top = top + (availH-piScrH*scale)/2.0

	// Export game canvas bounding box (window coordinates) so the
	// metrics overlay can position text in the black border area.
	GameCanvasX = g.left
	GameCanvasY = g.top
	GameCanvasW = piScrW * scale
	GameCanvasH = piScrH * scale

	g.drawScreenOpts.GeoM.Reset()
	g.drawScreenOpts.GeoM.Scale(g.scale, g.scale)
	g.drawScreenOpts.GeoM.Translate(g.left, g.top)

	return
}

// Layout method is not executed because LayoutF is
func (g *EbitenGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return
}

func (g *EbitenGame) Resize() {
	screen := pixelforge.Screen()
	g.piScreen = screen
	g.ebitenScreen = ebiten.NewImage(screen.W(), screen.H())
}

func (g *EbitenGame) onPidebugEvent(event pidebug.Event, _ pievent.Handler) {
	switch event {
	case pidebug.EventPause:
		g.paused = true
	case pidebug.EventResume:
		g.paused = false
	}
}
