package editor

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
)

// import_diff_modal_test.go covers idea #3 v1 U4's modal logic
// without driving cimgui-go. Each test exercises one state
// transition: Refresh into Showing, Accept/Reject dismissal, the
// Re-quantize sub-state, and the warning-banner threshold.

func newModalWithPending(t *testing.T, deltaE float64, autoPicked bool) (*Editor, *ImportDiffModal, *stubImportRunner) {
	t.Helper()
	e := New()
	stub := &stubImportRunner{}
	e.ImportHandler().SetRunner(stub)
	stub.nextResult = &PNGImportResult{
		SpriteName: "hero", RegisteredIdx: 0,
		Diff: &PNGImportDiff{
			OriginalImage:    image.NewRGBA(image.Rect(0, 0, 4, 4)),
			ChosenSubPalette: "sprite_0",
			AutoPicked:       autoPicked,
			MeanDeltaE:       deltaE,
		},
	}
	_, _ = e.ImportHandler().Import("/tmp/hero.png")
	return e, e.ImportDiffModal(), stub
}

// TestDiffModal_HiddenWhenNoPendingImport: a modal with no pending
// import reports Hidden.
func TestDiffModal_HiddenWhenNoPendingImport(t *testing.T) {
	e := New()
	m := e.ImportDiffModal()
	assert.False(t, m.Visible())
	assert.Equal(t, ImportDiffHidden, m.State())
}

// TestDiffModal_RefreshTransitionsHiddenToShowing: when a pending
// import arrives, Refresh flips the state to Showing.
func TestDiffModal_RefreshTransitionsHiddenToShowing(t *testing.T) {
	_, m, _ := newModalWithPending(t, 3.0, true)
	m.Refresh()
	assert.True(t, m.Visible())
	assert.Equal(t, ImportDiffShowing, m.State())
}

// TestDiffModal_WarningBannerAboveThreshold: HasWarning reports
// true for diffs whose MeanDeltaE exceeds the threshold (10.0).
func TestDiffModal_WarningBannerAboveThreshold(t *testing.T) {
	_, m, _ := newModalWithPending(t, 15.0, true)
	assert.True(t, m.HasWarning())
}

// TestDiffModal_NoBannerBelowThreshold: MeanDeltaE under threshold
// suppresses the banner.
func TestDiffModal_NoBannerBelowThreshold(t *testing.T) {
	_, m, _ := newModalWithPending(t, 5.0, true)
	assert.False(t, m.HasWarning())
}

// TestDiffModal_LabelForChosenSubPaletteAutoPicked: caption notes
// the auto-pick provenance.
func TestDiffModal_LabelForChosenSubPaletteAutoPicked(t *testing.T) {
	_, m, _ := newModalWithPending(t, 0.0, true)
	assert.Equal(t, "Quantized against sprite_0 (auto-picked)",
		m.LabelForChosenSubPalette())
}

// TestDiffModal_LabelForChosenSubPaletteManuallySelected: caption
// notes manual choice when AutoPicked is false (post-Re-quantize).
func TestDiffModal_LabelForChosenSubPaletteManuallySelected(t *testing.T) {
	_, m, _ := newModalWithPending(t, 0.0, false)
	assert.Equal(t, "Quantized against sprite_0 (manually selected)",
		m.LabelForChosenSubPalette())
}

// TestDiffModal_AcceptDismisses: Accept clears the pending import +
// flips the modal Hidden.
func TestDiffModal_AcceptDismisses(t *testing.T) {
	e, m, _ := newModalWithPending(t, 0.0, true)
	m.Accept()
	assert.False(t, m.Visible())
	assert.Equal(t, ImportDiffHidden, m.State())
	assert.True(t, e.IsDirty(), "Accept marks dirty")
}

// TestDiffModal_RejectDismissesAndRollsBack: Reject clears pending
// + invokes the runner's rollback hook.
func TestDiffModal_RejectDismissesAndRollsBack(t *testing.T) {
	_, m, stub := newModalWithPending(t, 0.0, true)
	m.Reject()
	assert.False(t, m.Visible())
	assert.Equal(t, []int{0}, stub.rollbacks,
		"Reject dispatches rollback through the runner")
}

// TestDiffModal_OpenRequantizeTransitions: OpenRequantize moves to
// the sub-state with the current chosen sub-palette pre-selected.
func TestDiffModal_OpenRequantizeTransitions(t *testing.T) {
	_, m, _ := newModalWithPending(t, 0.0, true)
	m.OpenRequantize()
	assert.Equal(t, ImportDiffRequantize, m.State())
	assert.Equal(t, "sprite_0", m.RequantizeTarget())
}

// TestDiffModal_CloseRequantizeReturnsToShowing: cancelling the
// sub-state pops back to the diff popup without closing.
func TestDiffModal_CloseRequantizeReturnsToShowing(t *testing.T) {
	_, m, _ := newModalWithPending(t, 0.0, true)
	m.OpenRequantize()
	m.CloseRequantize()
	assert.Equal(t, ImportDiffShowing, m.State())
}

// TestDiffModal_ConfirmRequantizeReplacesPending: ConfirmRequantize
// invokes the runner's Requantize + lands a new pending result with
// the picked target.
func TestDiffModal_ConfirmRequantizeReplacesPending(t *testing.T) {
	_, m, stub := newModalWithPending(t, 0.0, true)
	stub.nextResult = &PNGImportResult{
		SpriteName: "hero", RegisteredIdx: 0,
		Diff: &PNGImportDiff{
			OriginalImage:    image.NewRGBA(image.Rect(0, 0, 4, 4)),
			ChosenSubPalette: "sprite_2",
		},
	}
	m.OpenRequantize()
	m.SetRequantizeTarget("sprite_2")
	err := m.ConfirmRequantize()
	require.NoError(t, err)
	assert.Equal(t, ImportDiffShowing, m.State())
	assert.Equal(t, []string{"sprite_2"}, stub.requantizes)
	assert.Equal(t, "sprite_2", m.PendingDiff().ChosenSubPalette)
}

// TestDiffModal_ConfirmRequantizeNoTargetErrors: ConfirmRequantize
// without a selected target returns a sentinel error rather than
// proceeding silently.
func TestDiffModal_ConfirmRequantizeNoTargetErrors(t *testing.T) {
	_, m, _ := newModalWithPending(t, 0.0, true)
	m.OpenRequantize()
	m.SetRequantizeTarget("")
	err := m.ConfirmRequantize()
	require.Error(t, err)
}

// TestApplyPFTag_SubPaletteFamily: pf:"subpalette,family=sprite"
// resolves to WidgetSubPalette with Options[0]="sprite".
func TestApplyPFTag_SubPaletteFamily(t *testing.T) {
	withRegistry(t)
	type s struct {
		SubPal string `json:"sub_palette" pf:"subpalette,family=sprite"`
	}
	pfcomponent.Register[s]("SubPaletteSample")
	md, ok := pfcomponent.Get("SubPaletteSample")
	require.True(t, ok)
	require.Len(t, md.Fields, 1)
	assert.Equal(t, pfcomponent.WidgetSubPalette, md.Fields[0].WidgetKind)
	require.NotEmpty(t, md.Fields[0].Options)
	assert.Equal(t, "sprite", md.Fields[0].Options[0])
}

// TestApplyPFTag_SubPaletteFamilyBG: family=bg parses to the bg
// option.
func TestApplyPFTag_SubPaletteFamilyBG(t *testing.T) {
	withRegistry(t)
	type s struct {
		SubPal string `json:"sub_palette" pf:"subpalette,family=bg"`
	}
	pfcomponent.Register[s]("BGSample")
	md, _ := pfcomponent.Get("BGSample")
	assert.Equal(t, []string{"bg"}, md.Fields[0].Options)
}

// TestApplyPFTag_SubPaletteDefaultsSpriteFamily: missing family
// token defaults to "sprite".
func TestApplyPFTag_SubPaletteDefaultsSpriteFamily(t *testing.T) {
	withRegistry(t)
	type s struct {
		SubPal string `json:"sub_palette" pf:"subpalette"`
	}
	pfcomponent.Register[s]("DefaultFamilySample")
	md, _ := pfcomponent.Get("DefaultFamilySample")
	assert.Equal(t, []string{"sprite"}, md.Fields[0].Options)
}

// TestBuildWidgetContext_PopulatesSubPaletteNames: the context the
// inspector hands to renderField surfaces the project's 4 BG + 4
// Sprite sub-palette names, populated by applyDefaults.
func TestBuildWidgetContext_PopulatesSubPaletteNames(t *testing.T) {
	e := New()
	e.project.Palette.ApplyDefaults()
	ctx := buildWidgetContext(e.project)
	assert.Len(t, ctx.BGSubPaletteNames, 4)
	assert.Contains(t, ctx.BGSubPaletteNames, "bg_0")
	assert.Len(t, ctx.SpriteSubPaletteNames, 4)
	assert.Contains(t, ctx.SpriteSubPaletteNames, "sprite_0")
}
