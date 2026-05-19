// autotile_observer.go is the editor-side contract for the auto-tile
// rule synth. Concrete observers (palette.AutoTileRuleSynth) live
// outside this package and are wired via SetAutoTileObserver at
// studio startup; the canvas's ToolPaint dispatch (U6) and the toast
// queue (U7) read the observer through this interface so the editor
// package stays clear of an editor → palette import cycle.
package editor

import (
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// AutoTileCell mirrors palette.PaintCell at the editor's import
// boundary. Duplicated rather than imported to keep editor at the
// bottom of the package graph; conversions happen at the call site
// inside palette.RegisterWith when wiring the concrete observer.
type AutoTileCell struct {
	X     int
	Y     int
	Value int
}

// AutoTilePromotion is the editor-side view of palette.PromotedRule.
// Same shape, repeated here for the same import-cycle reason.
type AutoTilePromotion struct {
	RuleIndex int
	Pattern   [9]int
	Output    int
}

// AutoTileObserver is the contract the canvas dispatch (U6) and the
// toast queue (U7) call out to when a paint stroke commits. The
// observer's RecordStroke returns the rules that crossed the
// activation threshold this stroke; an empty return means no toast
// to surface.
type AutoTileObserver interface {
	RecordStroke(layer *pixelforge_project.TileAtlas, cells []AutoTileCell) []AutoTilePromotion
}

// SetAutoTileObserver replaces the editor's observer. Called once at
// studio startup by palette.RegisterWith (or any other observer
// implementer); nil clears the wiring so the canvas dispatch falls
// through without observing strokes.
func (e *Editor) SetAutoTileObserver(o AutoTileObserver) {
	if e == nil {
		return
	}
	e.autoTileObserver = o
}

// AutoTileObserver returns the currently-wired observer (nil-safe).
// Exposed so the toast queue (U7) can consult the same interface
// when seeding suppression from already-accepted rules.
func (e *Editor) AutoTileObserver() AutoTileObserver {
	if e == nil {
		return nil
	}
	return e.autoTileObserver
}

// RecordPaintStrokeAndQueuePromotions is the exported entry point
// the canvas dispatch and the integration tests share. Equivalent
// to recordStrokeAndQueuePromotions; promoted to public API because
// the test harness in pixelforge_studio/integration_test needs to
// drive the same post-stroke hook the canvas calls from inside
// updatePaintTool.
func (e *Editor) RecordPaintStrokeAndQueuePromotions(
	layer *pixelforge_project.TileAtlas,
	cmd *StrokeCommand,
) {
	e.recordStrokeAndQueuePromotions(layer, cmd)
}

// recordStrokeAndQueuePromotions is the canvas dispatch's hook into
// the observer + toast queue. Called from updatePaintTool after a
// stroke commits with a non-nil StrokeCommand. The function:
//
//  1. Builds an AutoTileCell list from the stroke's CellDiff entries.
//  2. Invokes the observer (no-op when unwired).
//  3. Enqueues a toast for the first promotion the observer surfaces
//     (when U7 is wired). Stale promotions from prior strokes are
//     wiped at BeginStroke time so the queue never carries cross-
//     stroke leakage.
//
// Promotions are deduped through the session-suppression map (U7) so
// "No" decisions stick for the rest of the session.
func (e *Editor) recordStrokeAndQueuePromotions(
	layer *pixelforge_project.TileAtlas,
	cmd *StrokeCommand,
) {
	if e == nil || e.autoTileObserver == nil || cmd == nil || layer == nil {
		return
	}
	if len(cmd.Diffs) == 0 {
		return
	}
	cells := make([]AutoTileCell, 0, len(cmd.Diffs))
	for _, d := range cmd.Diffs {
		cells = append(cells, AutoTileCell{X: d.Col, Y: d.Row, Value: d.NewID})
	}
	promotions := e.autoTileObserver.RecordStroke(layer, cells)
	e.handleAutoTilePromotions(promotions)
}

// handleAutoTilePromotions consumes the observer's promotions list.
// U7's toast queue implements the actual surfacing; for now the
// editor stashes the first promotion so the toast-queue tests can
// observe what the dispatch fed in.
//
// Single-promotion-per-stroke policy: when multiple promotions land
// in one stroke (uncommon but possible for complex strokes), only
// the first surfaces a toast. The rest are added to the session-
// suppression map immediately so re-promotion does not spam the
// designer.
//
// Session-suppression check: a promotion whose signature is already
// in sessionRuleSuppression (the designer picked "No" earlier this
// session) is silently dropped without queueing.
func (e *Editor) handleAutoTilePromotions(promotions []AutoTilePromotion) {
	if len(promotions) == 0 {
		return
	}
	for i, promo := range promotions {
		if e.isSuppressed(promo) {
			continue
		}
		if i == 0 || e.queuedPromotion == nil {
			p := promo
			e.queuedPromotion = &p
		}
		// Every promotion past the surfaced one is auto-suppressed
		// for the rest of the session — the toast only handles one
		// at a time and stacking would overwhelm the designer.
		if e.queuedPromotion != nil && *e.queuedPromotion != promo {
			e.suppressRule(promo)
		}
	}
}

// QueuedPromotion exposes the most-recent promotion queued by the
// canvas dispatch. The toast subsystem (U7) reads this each frame to
// decide whether to render the popup. Returns nil when no promotion
// is pending. Nil-safe on a zero-value editor.
func (e *Editor) QueuedPromotion() *AutoTilePromotion {
	if e == nil {
		return nil
	}
	return e.queuedPromotion
}

// ClearQueuedPromotion drops the queued promotion. Called by the
// toast subsystem after the designer dismisses (Yes / No / Esc /
// click-outside) so the next promotion has a clean slot.
func (e *Editor) ClearQueuedPromotion() {
	if e == nil {
		return
	}
	e.queuedPromotion = nil
}
