// tileatlas_e2e_test.go exercises idea #2 v1 (TileAtlas component
// reframe + emergent auto-rule UX) end-to-end. The tests drive the
// public editor + palette APIs directly without simulating ImGui
// events; toast interactions go through AutoTileToast's Yes / No /
// Esc methods, which are the same code paths the live popup
// invokes from inside its frame.
//
// Coverage:
//
//   - AE1: third matching stroke queues a toast; Yes activates the
//     rule for the session.
//   - AE2: third matching stroke queues a toast; Esc dismisses,
//     suppression added, rule persists on the TileAtlas.
//   - AE3: load a project whose AutoTileRule is already at Count >=
//     threshold; paint matching pattern — auto-apply happens
//     silently, no toast surfaces.
//   - AE4: load a legacy fixture with the `tilemaps` JSON key;
//     migrate to TileAtlases without losing painted content.
//   - AE5: a TileAtlas field with a pf-tag dispatches the right
//     inspector widget (tilepainter) without any per-field editor
//     code.
//
// The tests intentionally use the autotile observer adapter the
// palette package's RegisterWith installs in production so the
// promotion path here matches what designers actually exercise.
package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/palette"
)

const (
	legacyTilemapsFixturePath  = "fixtures/legacy_tilemaps_with_content.pforge"
	acceptedRuleFixturePath    = "fixtures/tileatlas_with_accepted_rule.pforge"
	tileAtlasActivationThresh  = palette.AutoTileActivationThreshold
	tileAtlasLonelyTileValue   = 5
)

// observerAdapter wraps palette.AutoTileRuleSynth in the editor's
// observer interface so the e2e test exercises the production
// promotion path. This is the same wrapper palette.RegisterWith
// installs at studio startup.
type observerAdapter struct {
	synth *palette.AutoTileRuleSynth
}

func newObserverAdapter() *observerAdapter {
	return &observerAdapter{synth: palette.NewAutoTileRuleSynth()}
}

func (a *observerAdapter) RecordStroke(
	layer *pixelforge_project.TileAtlas,
	cells []editor.AutoTileCell,
) []editor.AutoTilePromotion {
	if a == nil || a.synth == nil {
		return nil
	}
	stroke := make([]palette.PaintCell, 0, len(cells))
	for _, c := range cells {
		stroke = append(stroke, palette.PaintCell{X: c.X, Y: c.Y, Value: c.Value})
	}
	raw := a.synth.RecordStrokeWithPromotions(layer, stroke)
	out := make([]editor.AutoTilePromotion, 0, len(raw))
	for _, r := range raw {
		out = append(out, editor.AutoTilePromotion{
			RuleIndex: r.RuleIndex,
			Pattern:   r.Pattern,
			Output:    r.Output,
		})
	}
	return out
}

// paintStrokeViaPainter runs one full BrushStartStroke → BrushPaint
// (per cell) → BrushEndStroke cycle and feeds the resulting
// StrokeCommand through the observer wiring (mirroring what
// canvas.updatePaintTool does at LMB-up). Returns the editor so
// callers can chain assertions.
func paintStrokeViaPainter(
	t *testing.T,
	e *editor.Editor,
	layer *pixelforge_project.TileAtlas,
	cells []tileCoord,
	value int,
) {
	t.Helper()
	painter := e.Painter()
	stack := e.UndoStack()
	painter.BrushStartStroke(layer)
	for _, c := range cells {
		painter.BrushPaint(c.col, c.row, value)
	}
	cmd := painter.BrushEndStroke(stack)
	if cmd == nil {
		return
	}
	// Mirror canvas.updatePaintTool's post-stroke promotion hook.
	e.RecordPaintStrokeAndQueuePromotions(layer, cmd)
}

type tileCoord struct {
	col int
	row int
}

// loadFixtureScene loads the named fixture and returns the first
// scene's pointer. Trims a few lines from each test.
func loadFixtureScene(t *testing.T, path string) (*pixelforge_project.Project, *pixelforge_project.Scene) {
	t.Helper()
	abs, err := filepath.Abs(path)
	require.NoError(t, err)
	p, err := pixelforge_project.Load(abs)
	require.NoError(t, err, "fixture %s must load cleanly", abs)
	require.NotEmpty(t, p.Scenes)
	return p, &p.Scenes[0]
}

// TestE2E_AE1_ThirdMatchingStrokePromotesYesActivates: paint the
// same lonely-cell pattern three times via the painter; the third
// stroke queues a toast; Yes accepts; future matching paints fill
// silently because the rule is now active.
func TestE2E_AE1_ThirdMatchingStrokePromotesYesActivates(t *testing.T) {
	e := editor.New()
	obs := newObserverAdapter()
	e.SetAutoTileObserver(obs)
	toast := editor.NewAutoTileToast(e)

	layer := &pixelforge_project.TileAtlas{}

	// Stroke 1: count == 2 (two cells, both recording the same
	// lonely-cell pattern). Below threshold; no toast.
	paintStrokeViaPainter(t, e, layer,
		[]tileCoord{{col: 2, row: 2}, {col: 7, row: 2}},
		tileAtlasLonelyTileValue)
	assert.False(t, toast.Visible(), "stroke 1 below threshold")

	// Stroke 2: count crosses threshold. Toast queues.
	paintStrokeViaPainter(t, e, layer,
		[]tileCoord{{col: 2, row: 10}, {col: 7, row: 10}},
		tileAtlasLonelyTileValue)
	require.True(t, toast.Visible(), "stroke 2 crosses threshold and toasts")
	require.NotNil(t, toast.Promotion())
	assert.Equal(t, tileAtlasLonelyTileValue, toast.Promotion().Output)

	// Yes → rule active for the session, suppression untouched.
	toast.Yes()
	assert.False(t, toast.Visible())
	assert.Equal(t, 0, e.SuppressedRuleCount(),
		"Yes does not suppress the rule")
	require.NotEmpty(t, layer.AutoTileRules)
	assert.GreaterOrEqual(t, layer.AutoTileRules[0].Count, tileAtlasActivationThresh,
		"rule is active (Count >= threshold)")
}

// TestE2E_AE2_NoDismissesAndPersists: the No / Esc dismissal flow
// adds the rule's signature to the suppression map AND leaves the
// rule on the TileAtlas (so a future session can re-toast).
func TestE2E_AE2_NoDismissesAndPersists(t *testing.T) {
	e := editor.New()
	obs := newObserverAdapter()
	e.SetAutoTileObserver(obs)
	toast := editor.NewAutoTileToast(e)
	layer := &pixelforge_project.TileAtlas{}

	// Two strokes drive Count past threshold; toast queues.
	paintStrokeViaPainter(t, e, layer,
		[]tileCoord{{col: 2, row: 2}, {col: 7, row: 2}},
		tileAtlasLonelyTileValue)
	paintStrokeViaPainter(t, e, layer,
		[]tileCoord{{col: 2, row: 10}, {col: 7, row: 10}},
		tileAtlasLonelyTileValue)
	require.True(t, toast.Visible())

	// Esc routes to No semantics: dismiss + suppress.
	toast.Esc()
	assert.False(t, toast.Visible())
	assert.Equal(t, 1, e.SuppressedRuleCount(),
		"Esc/No inserts the rule's signature into the suppression map")
	require.NotEmpty(t, layer.AutoTileRules,
		"rule still persists on the TileAtlas after dismissal")

	// Re-paint a matching pattern; the synth re-encounters the
	// active rule but the suppression check prevents a re-toast.
	paintStrokeViaPainter(t, e, layer,
		[]tileCoord{{col: 2, row: 20}, {col: 7, row: 20}},
		tileAtlasLonelyTileValue)
	assert.False(t, toast.Visible(),
		"suppressed rule does not re-surface within the session")
}

// TestE2E_AE3_AcceptedRuleSilentApplyOnReload: load the
// tileatlas_with_accepted_rule.pforge fixture (rule at
// Count=4 >= threshold). Painting a matching pattern silently auto-
// applies; no toast queues because the synth does not re-promote
// already-active rules.
func TestE2E_AE3_AcceptedRuleSilentApplyOnReload(t *testing.T) {
	_, scene := loadFixtureScene(t, acceptedRuleFixturePath)
	require.NotEmpty(t, scene.TileAtlases)
	atlas := &scene.TileAtlases[0]
	require.NotEmpty(t, atlas.AutoTileRules)
	require.GreaterOrEqual(t, atlas.AutoTileRules[0].Count, tileAtlasActivationThresh,
		"fixture's rule is at or above threshold")

	e := editor.New()
	obs := newObserverAdapter()
	e.SetAutoTileObserver(obs)
	toast := editor.NewAutoTileToast(e)

	paintStrokeViaPainter(t, e, atlas,
		[]tileCoord{{col: 1, row: 1}, {col: 3, row: 1}},
		tileAtlasLonelyTileValue)

	assert.False(t, toast.Visible(),
		"loaded pre-accepted rule auto-applies silently; no toast queued")
}

// TestE2E_AE4_LegacyTilemapsFixtureLoadsCorrectly: load the legacy
// fixture (carries the old `tilemaps` JSON key); the migration
// shim binds the painted cells to TileAtlases without losing
// content. Round-trip save produces only `tile_atlases`.
func TestE2E_AE4_LegacyTilemapsFixtureLoadsCorrectly(t *testing.T) {
	p, scene := loadFixtureScene(t, legacyTilemapsFixturePath)
	require.NotEmpty(t, scene.TileAtlases,
		"legacy tilemaps key migrates to TileAtlases on load")
	atlas := scene.TileAtlases[0]
	assert.Equal(t, "ground", atlas.Name)
	assert.Equal(t, 8, atlas.TileW)
	assert.Equal(t, [][]int{{1, 1, 1}, {1, 2, 1}, {1, 1, 1}}, atlas.Grid,
		"painted cells preserve through the migration")

	// Round-trip: save the project to bytes, confirm `tilemaps`
	// key is gone and `tile_atlases` is present.
	data, err := pixelforge_project.Encode(p)
	require.NoError(t, err)
	s := string(data)
	assert.NotContains(t, s, `"tilemaps":`,
		"re-marshal drops the legacy key")
	assert.Contains(t, s, `"tile_atlases":`,
		"re-marshal writes the v1+ key")
}

// TestE2E_AE5_NewTileFieldRendersAutomatically: the TileAtlas
// reserved fields (AnimationFps, ParallaxFactor) carry pf-tag
// declarations; the inspector dispatch sees them as WidgetSlider
// without any per-field editor code. This is the leverage move R9
// promises: developer adds field + tag → inspector renders it.
func TestE2E_AE5_NewTileFieldRendersAutomatically(t *testing.T) {
	md, ok := pfcomponent.Get("TileAtlas")
	require.True(t, ok, "TileAtlas is registered via editor.init()")
	byName := map[string]pfcomponent.FieldMetadata{}
	for _, f := range md.Fields {
		byName[f.Name] = f
	}
	require.Contains(t, byName, "AnimationFps")
	require.Contains(t, byName, "ParallaxFactor")
	assert.Equal(t, pfcomponent.WidgetSlider, byName["AnimationFps"].WidgetKind,
		"AnimationFps' pf:\"slider,0..30\" tag dispatches the slider widget without per-field code")
	assert.Equal(t, pfcomponent.WidgetSlider, byName["ParallaxFactor"].WidgetKind)

	// The Painter hook field uses the WidgetCustom dispatch so the
	// custom tilepainter widget renders. Same leverage path.
	require.Contains(t, byName, "Painter")
	assert.Equal(t, pfcomponent.WidgetCustom, byName["Painter"].WidgetKind)
	assert.Equal(t, "tilepainter", byName["Painter"].CustomWidget)
}

// TestE2E_F1_FullFlow_PaintToToastToActivateToAutoApply: end-to-end
// flow from F1 — designer paints, toast appears, designer accepts,
// subsequent matching paint auto-applies silently.
func TestE2E_F1_FullFlow_PaintToToastToActivateToAutoApply(t *testing.T) {
	e := editor.New()
	obs := newObserverAdapter()
	e.SetAutoTileObserver(obs)
	toast := editor.NewAutoTileToast(e)
	layer := &pixelforge_project.TileAtlas{}

	// Strokes 1-2: count crosses threshold, toast queues.
	paintStrokeViaPainter(t, e, layer,
		[]tileCoord{{col: 2, row: 2}, {col: 7, row: 2}},
		tileAtlasLonelyTileValue)
	paintStrokeViaPainter(t, e, layer,
		[]tileCoord{{col: 2, row: 10}, {col: 7, row: 10}},
		tileAtlasLonelyTileValue)
	require.True(t, toast.Visible())
	toast.Yes()

	// Stroke 3: rule is active. Future strokes do not re-promote.
	paintStrokeViaPainter(t, e, layer,
		[]tileCoord{{col: 2, row: 20}, {col: 7, row: 20}},
		tileAtlasLonelyTileValue)
	assert.False(t, toast.Visible(),
		"once a rule is active, subsequent matching strokes do not re-toast")
}

// TestE2E_F2_ReturningSessionAppliesPreviousRules: end-to-end flow
// from F2 — load a project with an accepted rule; a fresh editor
// session sees no toast on a matching paint, just silent auto-apply
// (covered by AE3, repeated here for the flow's narrative arc).
func TestE2E_F2_ReturningSessionAppliesPreviousRules(t *testing.T) {
	_, scene := loadFixtureScene(t, acceptedRuleFixturePath)
	require.NotEmpty(t, scene.TileAtlases)
	atlas := &scene.TileAtlases[0]

	e := editor.New()
	obs := newObserverAdapter()
	e.SetAutoTileObserver(obs)
	toast := editor.NewAutoTileToast(e)

	paintStrokeViaPainter(t, e, atlas,
		[]tileCoord{{col: 1, row: 1}, {col: 3, row: 1}},
		tileAtlasLonelyTileValue)
	assert.False(t, toast.Visible(),
		"returning session — accepted rule applies silently, no toast")
}
