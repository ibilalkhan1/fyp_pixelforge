package scripting

import (
	"fmt"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/runtime"
)

// Debugger is the canvas-resident debugger sub-pane (U13).
//
// Wraps the engine's breakpoint and pause API for the UI, plus
// surfaces "Step / Continue / Stop" controls and the M4-recorder-
// backed time-travel scrub.
type Debugger struct {
	engine *runtime.Engine

	// localBreakpoints tracks paths in the debugger UI before bind()
	// hands them to the engine. Once bound, this map mirrors the
	// engine's; ToggleBreakpoint sets both.
	localBreakpoints map[string]bool

	selectedName string
	scrubFrame   int

	// lastEvent is the most recent event observed via the engine's
	// hook. Used to display the live "at: ..." indicator.
	lastEvent runtime.DebugEvent
}

func newDebugger() *Debugger {
	return &Debugger{
		localBreakpoints: map[string]bool{},
		scrubFrame:       -1,
	}
}

func (d *Debugger) bind(eng *runtime.Engine) {
	d.engine = eng
	if eng == nil {
		return
	}
	// Forward any locally-toggled breakpoints into the engine.
	for path, on := range d.localBreakpoints {
		eng.SetBreakpoint(path, on)
	}
	eng.SetDebugHook(func(ev runtime.DebugEvent) {
		d.lastEvent = ev
	})
}

// ToggleBreakpoint flips the breakpoint at path.
func (d *Debugger) ToggleBreakpoint(path string) {
	if d == nil {
		return
	}
	on := !d.localBreakpoints[path]
	if on {
		d.localBreakpoints[path] = true
	} else {
		delete(d.localBreakpoints, path)
	}
	if d.engine != nil {
		d.engine.SetBreakpoint(path, on)
	}
}

// BreakpointSet reports whether a breakpoint is set at path.
func (d *Debugger) BreakpointSet(path string) bool {
	if d == nil {
		return false
	}
	return d.localBreakpoints[path]
}

// Paused reports whether the engine is currently paused.
func (d *Debugger) Paused() (bool, string) {
	if d == nil || d.engine == nil {
		return false, ""
	}
	paused, ev := d.engine.Paused()
	return paused, ev.Path()
}

// Step advances one step / rule then re-pauses.
func (d *Debugger) Step() {
	if d == nil || d.engine == nil {
		return
	}
	d.engine.Step()
}

// Continue resumes the engine until the next breakpoint.
func (d *Debugger) Continue() {
	if d == nil || d.engine == nil {
		return
	}
	d.engine.Continue()
}

// Stop tears down the runtime engine.
func (d *Debugger) Stop() {
	if d == nil || d.engine == nil {
		return
	}
	d.engine.Stop()
}

// Update routes input.
func (d *Debugger) Update(e *editor.Editor, _ *runtime.Engine) {
	if d == nil || e == nil {
		return
	}
}

// DrawCanvas paints the debugger.
func (d *Debugger) DrawCanvas(rel widgets.Rect, _ *editor.Editor, theme *editor.EditorTheme) {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	pixelforge.SetColor(theme.PanelSlot)
	pixelforge.RectFill(rel.X, rel.Y, rel.X+rel.W-1, rel.Y+rel.H-1)

	pixelforge.SetColor(theme.TextSlot)
	pixelforge_cofont.Print("DEBUGGER", rel.X+4, rel.Y+2)

	statusY := rel.Y + 14
	paused, pausedAt := d.Paused()
	if paused {
		pixelforge.SetColor(theme.AccentSlot)
		pixelforge_cofont.Print(fmt.Sprintf("PAUSED at %s", pausedAt), rel.X+4, statusY)
	} else {
		pixelforge.SetColor(theme.TextDimSlot)
		pixelforge_cofont.Print("(running)", rel.X+4, statusY)
	}

	// List breakpoints.
	pixelforge.SetColor(theme.TextSlot)
	pixelforge_cofont.Print(fmt.Sprintf("breakpoints (%d):", len(d.localBreakpoints)), rel.X+4, statusY+12)
	y := statusY + 24
	for path := range d.localBreakpoints {
		pixelforge.SetColor(theme.TextDimSlot)
		pixelforge_cofont.Print("• "+path, rel.X+8, y)
		y += 9
		if y > rel.Y+rel.H-20 {
			break
		}
	}

	// Footer: instructions.
	pixelforge.SetColor(theme.TextDimSlot)
	pixelforge_cofont.Print("[Alt+S] Step  [Alt+R] Continue  [Alt+Esc] Stop", rel.X+4, rel.Y+rel.H-12)
}
