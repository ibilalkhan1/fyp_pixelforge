// autotile_toast.go renders the auto-rule promotion toast that idea
// #2 v1 U7 introduces. The toast appears near the brush cursor the
// moment the synth's promotion threshold crosses; Yes accepts the
// rule for the session, No / Esc / click-outside dismiss it and
// insert its signature into the session-suppression map so it does
// not re-surface for the rest of the session.
//
// Per docs/solutions/focus-manager-design.md the toast registers as
// the top modal while visible so Esc dismisses the toast first
// rather than toggling chrome visibility. Per docs/solutions/
// always-on-game-embedding.md the toast does not pause the always-
// on game preview — ImGui popups by default don't block game
// updates.
//
// Tests drive Yes / No / Esc directly via the exported methods on
// AutoTileToast so the popup rendering (which requires a live ImGui
// frame) is not part of the test surface.
package editor

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"
)

// AutoTileToast is the session-scoped toast UX. It reads the
// editor's QueuedPromotion each frame and surfaces a popup when one
// is pending. The toast owns no promotion state of its own; the
// promotion lives on the editor (queuedPromotion) so the canvas
// dispatch and the toast see the same value.
type AutoTileToast struct {
	editor *Editor
}

// NewAutoTileToast returns a toast bound to the editor. Nil-safe.
func NewAutoTileToast(e *Editor) *AutoTileToast {
	return &AutoTileToast{editor: e}
}

// Visible reports whether a promotion is currently waiting for the
// designer's decision. The render path uses this to gate imgui calls.
func (t *AutoTileToast) Visible() bool {
	if t == nil || t.editor == nil {
		return false
	}
	return t.editor.QueuedPromotion() != nil
}

// Promotion returns the currently-queued promotion (nil when the
// toast is hidden). Exposed for tests.
func (t *AutoTileToast) Promotion() *AutoTilePromotion {
	if t == nil || t.editor == nil {
		return nil
	}
	return t.editor.QueuedPromotion()
}

// Yes accepts the queued promotion. The rule is already active on
// the TileAtlas (Count crossed the threshold during the stroke that
// surfaced this toast); accepting just dismisses the prompt. The
// rule's signature is NOT added to the suppression map so subsequent
// editor sessions can still toast it again if the user wants a
// second chance.
func (t *AutoTileToast) Yes() {
	if t == nil || t.editor == nil {
		return
	}
	t.editor.ClearQueuedPromotion()
}

// No dismisses the toast AND inserts the rule's signature into the
// session-suppression map so re-promotion this session stays silent.
// The underlying rule remains on the TileAtlas — designers who
// reload the project get a fresh chance to accept it. Esc and
// click-outside are semantically equivalent to No.
func (t *AutoTileToast) No() {
	if t == nil || t.editor == nil {
		return
	}
	if promo := t.editor.QueuedPromotion(); promo != nil {
		t.editor.suppressRule(*promo)
	}
	t.editor.ClearQueuedPromotion()
}

// Esc routes to No semantics. Exposed as its own method so the
// FocusManager hook + tests can express intent clearly.
func (t *AutoTileToast) Esc() { t.No() }

// ClickOutside routes to No semantics. Exposed for tests.
func (t *AutoTileToast) ClickOutside() { t.No() }

// Render emits the toast popup inside the current ImGui frame.
// Skipped when no promotion is queued. The popup appears near the
// brush cursor (positioned via SetNextWindowPos at the mouse
// position the caller passes in). Keyboard events: Enter = Yes,
// Esc = No; the popup's button clicks are the primary path.
//
// Returns true when the toast dismissed this frame so the caller
// can short-circuit redundant per-frame work.
func (t *AutoTileToast) Render(cursorX, cursorY float32) bool {
	if !t.Visible() {
		return false
	}
	promo := t.Promotion()
	popupID := "AutoRuleToast"
	imgui.SetNextWindowPosV(imgui.Vec2{X: cursorX, Y: cursorY}, imgui.CondAppearing, imgui.Vec2{})
	imgui.OpenPopupStr(popupID)
	if !imgui.BeginPopup(popupID) {
		return false
	}
	defer imgui.EndPopup()

	imgui.TextUnformatted("Auto-apply this pattern?")
	imgui.TextDisabled(fmt.Sprintf("→ tile %d", promo.Output))
	imgui.Separator()

	if imgui.Button("Yes") {
		t.Yes()
		return true
	}
	imgui.SameLine()
	if imgui.Button("No") {
		t.No()
		return true
	}
	// Esc is routed via the editor's keymap handler; the popup also
	// observes ImGui's own popup-close events (click-outside, Esc
	// inside the popup) which the FocusManager hook below treats as
	// No.
	if imgui.IsKeyPressedBoolV(imgui.KeyEscape, false) {
		t.Esc()
		return true
	}
	if imgui.IsKeyPressedBoolV(imgui.KeyEnter, false) {
		t.Yes()
		return true
	}
	return false
}
