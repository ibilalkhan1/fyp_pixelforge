package pixelforge_entity

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// drawCall captures one invocation of DrawSpriteFn so tests can
// assert on argument shape + call order without standing up a real
// Ebitengine context.
type drawCall struct {
	Sprite pixelforge.Sprite
	DX, DY int
}

// recorder swaps DrawSpriteFn with a closure that appends to the
// returned slice. The cleanup func restores the original DrawSpriteFn
// (the production pixelforge.DrawSprite) so tests stay isolated.
func recorder(t *testing.T) *[]drawCall {
	t.Helper()
	calls := make([]drawCall, 0, 4)
	prev := DrawSpriteFn
	DrawSpriteFn = func(s pixelforge.Sprite, dx, dy int) {
		calls = append(calls, drawCall{Sprite: s, DX: dx, DY: dy})
	}
	t.Cleanup(func() { DrawSpriteFn = prev })
	return &calls
}

// captureLog redirects the standard logger's output to a buffer so a
// test can assert that warnings were emitted. Restores the prior
// output destination on cleanup.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	prev := log.Writer()
	log.SetOutput(buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return buf
}

// heroSprite returns a SpriteAsset named "hero" with sane 8x8
// dimensions for tests that need a sprite to draw.
func heroSprite() pixelforge_project.SpriteAsset {
	return pixelforge_project.SpriteAsset{
		Name:   "hero",
		Width:  8,
		Height: 8,
		FrameW: 8,
		FrameH: 8,
	}
}

// entityWithSprite is a convenience constructor for an entity whose
// only component is a Sprite component referencing the given asset
// name. Position.Z defaults to z so tests can stage ordering.
func entityWithSprite(id, spriteName string, tileX, tileY, z int) pixelforge_project.Entity {
	return pixelforge_project.Entity{
		ID:    id,
		Name:  id,
		TileX: tileX,
		TileY: tileY,
		Position: pixelforge_project.EntityPosition{
			X: float64(tileX * TileW),
			Y: float64(tileY * TileH),
			Z: z,
		},
		Components: []pixelforge_project.EntityComponent{{
			Type: SpriteComponentType,
			Values: map[string]any{
				SpriteValueKey: spriteName,
			},
		}},
	}
}

func TestRenderAll_EmptyEntities(t *testing.T) {
	calls := recorder(t)

	RenderAll(nil, []pixelforge_project.SpriteAsset{heroSprite()})
	if len(*calls) != 0 {
		t.Fatalf("expected no DrawSprite calls for nil entities, got %d", len(*calls))
	}

	RenderAll([]pixelforge_project.Entity{}, []pixelforge_project.SpriteAsset{heroSprite()})
	if len(*calls) != 0 {
		t.Fatalf("expected no DrawSprite calls for empty entities, got %d", len(*calls))
	}
}

func TestRenderAll_SingleEntityWithSprite(t *testing.T) {
	calls := recorder(t)

	ents := []pixelforge_project.Entity{
		entityWithSprite("e1", "hero", 10, 24, 0),
	}
	sprites := []pixelforge_project.SpriteAsset{heroSprite()}

	RenderAll(ents, sprites)

	if len(*calls) != 1 {
		t.Fatalf("expected 1 DrawSprite call, got %d", len(*calls))
	}
	got := (*calls)[0]
	// TileX=10, TileY=24, TileW=TileH=8 -> scene-coord (80, 192).
	if got.DX != 80 || got.DY != 192 {
		t.Errorf("expected scene-coord (80, 192), got (%d, %d)", got.DX, got.DY)
	}
	if got.Sprite.W != 8 || got.Sprite.H != 8 {
		t.Errorf("expected sprite size 8x8, got %dx%d", got.Sprite.W, got.Sprite.H)
	}
}

func TestRenderAll_EntityWithoutSprite_Skipped(t *testing.T) {
	calls := recorder(t)
	logBuf := captureLog(t)

	ents := []pixelforge_project.Entity{
		{
			ID:    "logic-only",
			Name:  "trigger",
			TileX: 5,
			TileY: 5,
			Components: []pixelforge_project.EntityComponent{{
				Type:   "Trigger",
				Values: map[string]any{"event": "level_start"},
			}},
		},
	}

	RenderAll(ents, []pixelforge_project.SpriteAsset{heroSprite()})

	if len(*calls) != 0 {
		t.Fatalf("expected logic-only entity to be skipped, got %d DrawSprite calls", len(*calls))
	}
	if logBuf.Len() != 0 {
		t.Errorf("expected silent skip (no warning) for entity without Sprite component, got log output: %q", logBuf.String())
	}
}

func TestRenderAll_UnknownSpriteName_LogsWarning(t *testing.T) {
	calls := recorder(t)
	logBuf := captureLog(t)

	ents := []pixelforge_project.Entity{
		entityWithSprite("e-bad", "nonexistent", 1, 1, 0),
		entityWithSprite("e-good", "hero", 2, 2, 0),
	}
	sprites := []pixelforge_project.SpriteAsset{heroSprite()}

	RenderAll(ents, sprites)

	// Render continues for the good entity.
	if len(*calls) != 1 {
		t.Fatalf("expected 1 DrawSprite call (the good entity), got %d", len(*calls))
	}
	if (*calls)[0].DX != 16 || (*calls)[0].DY != 16 {
		t.Errorf("expected good entity at (16, 16), got (%d, %d)", (*calls)[0].DX, (*calls)[0].DY)
	}
	// Warning logged for the unknown sprite reference.
	out := logBuf.String()
	if !strings.Contains(out, "nonexistent") {
		t.Errorf("expected log warning mentioning %q, got: %q", "nonexistent", out)
	}
	if !strings.Contains(out, "e-bad") {
		t.Errorf("expected log warning to mention the entity id %q, got: %q", "e-bad", out)
	}
}

func TestRenderAll_RespectsCameraOffset(t *testing.T) {
	calls := recorder(t)

	// Set the engine camera; the renderer must not subtract it
	// itself (the real DrawSprite does that), so the recorded dx/dy
	// stays in scene-space.
	prevCam := pixelforge.Camera
	pixelforge.Camera = pixelforge.Position{X: 100, Y: 50}
	t.Cleanup(func() { pixelforge.Camera = prevCam })

	ents := []pixelforge_project.Entity{
		entityWithSprite("e1", "hero", 20, 10, 0),
	}
	RenderAll(ents, []pixelforge_project.SpriteAsset{heroSprite()})

	if len(*calls) != 1 {
		t.Fatalf("expected 1 DrawSprite call, got %d", len(*calls))
	}
	// Scene-coord (TileX*8, TileY*8) = (160, 80). The recorder
	// captures whatever the renderer hands to DrawSpriteFn, which
	// must be the scene-space coord — camera subtraction happens
	// inside the real DrawSprite, downstream of the renderer.
	if (*calls)[0].DX != 160 || (*calls)[0].DY != 80 {
		t.Errorf("expected scene-coord (160, 80), got (%d, %d) -- renderer must not subtract Camera itself", (*calls)[0].DX, (*calls)[0].DY)
	}
}

func TestRenderAll_ZOrderingHonored(t *testing.T) {
	calls := recorder(t)

	// Authored in arbitrary order with Z values 1, 0, 2. Rendered
	// order must be ascending Z: 0, 1, 2.
	ents := []pixelforge_project.Entity{
		entityWithSprite("z1", "hero", 1, 0, 1),
		entityWithSprite("z0", "hero", 0, 0, 0),
		entityWithSprite("z2", "hero", 2, 0, 2),
	}
	sprites := []pixelforge_project.SpriteAsset{heroSprite()}

	RenderAll(ents, sprites)

	if len(*calls) != 3 {
		t.Fatalf("expected 3 DrawSprite calls, got %d", len(*calls))
	}
	// Ascending Z -> entities with TileX 0, 1, 2 in that order.
	wantOrderTileX := []int{0, 1, 2}
	for i, want := range wantOrderTileX {
		gotTileX := (*calls)[i].DX / TileW
		if gotTileX != want {
			t.Errorf("draw #%d: expected TileX=%d (Z order), got TileX=%d (DX=%d)", i, want, gotTileX, (*calls)[i].DX)
		}
	}

	// Verify the input slice is not mutated by RenderAll.
	if ents[0].ID != "z1" || ents[1].ID != "z0" || ents[2].ID != "z2" {
		t.Errorf("input slice was mutated: got order %s, %s, %s", ents[0].ID, ents[1].ID, ents[2].ID)
	}
}

func TestRenderAll_MultiEntityScene(t *testing.T) {
	calls := recorder(t)
	logBuf := captureLog(t)

	// 10 entities of mixed shape:
	//  - 4 with valid Sprite components (zs: 5, 2, 8, 0)
	//  - 3 without any Sprite component (logic-only)
	//  - 2 with a Sprite component but referencing an unknown sprite
	//  - 1 with a Sprite component whose Values map is missing the key
	ents := []pixelforge_project.Entity{
		entityWithSprite("draw-z5", "hero", 5, 0, 5),
		{ID: "logic-1", Components: []pixelforge_project.EntityComponent{{Type: "Trigger"}}},
		entityWithSprite("draw-z2", "hero", 2, 0, 2),
		entityWithSprite("draw-bad-1", "ghost", 9, 0, 9),
		{ID: "logic-2"}, // No components at all.
		entityWithSprite("draw-z8", "hero", 8, 0, 8),
		{
			ID: "draw-missing-key",
			Components: []pixelforge_project.EntityComponent{{
				Type:   SpriteComponentType,
				Values: map[string]any{"other": "fruit"},
			}},
		},
		entityWithSprite("draw-bad-2", "phantom", 11, 0, 11),
		entityWithSprite("draw-z0", "hero", 0, 0, 0),
		{ID: "logic-3", Components: []pixelforge_project.EntityComponent{{Type: "Behavior"}}},
	}
	sprites := []pixelforge_project.SpriteAsset{heroSprite()}

	RenderAll(ents, sprites)

	// Only the 4 with valid sprite assets draw.
	if len(*calls) != 4 {
		t.Fatalf("expected 4 DrawSprite calls (only entities with a resolvable sprite), got %d", len(*calls))
	}
	// Ascending Z -> TileX in order 0, 2, 5, 8.
	wantOrderTileX := []int{0, 2, 5, 8}
	for i, want := range wantOrderTileX {
		gotTileX := (*calls)[i].DX / TileW
		if gotTileX != want {
			t.Errorf("draw #%d: expected TileX=%d, got TileX=%d", i, want, gotTileX)
		}
	}
	// Each of the two unknown-sprite entities produced a warning;
	// the missing-key entity is a schema violation handled
	// defensively (silent skip per spriteComponentName's contract).
	out := logBuf.String()
	if !strings.Contains(out, "ghost") || !strings.Contains(out, "phantom") {
		t.Errorf("expected warnings for ghost and phantom, got: %q", out)
	}
}

func TestRenderAll_StableOrderForTiedZ(t *testing.T) {
	// Stability check: entities at the same Z render in their
	// authored order. This is the contract that lets a designer
	// stack identical-Z sprites and predict the result.
	calls := recorder(t)

	ents := []pixelforge_project.Entity{
		entityWithSprite("first", "hero", 1, 0, 0),
		entityWithSprite("second", "hero", 2, 0, 0),
		entityWithSprite("third", "hero", 3, 0, 0),
	}
	RenderAll(ents, []pixelforge_project.SpriteAsset{heroSprite()})

	if len(*calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(*calls))
	}
	wantTileX := []int{1, 2, 3}
	for i, want := range wantTileX {
		if (*calls)[i].DX/TileW != want {
			t.Errorf("tied-Z order broken at index %d: expected TileX=%d, got TileX=%d", i, want, (*calls)[i].DX/TileW)
		}
	}
}
