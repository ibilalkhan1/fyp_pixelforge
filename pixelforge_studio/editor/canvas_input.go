// canvas_input.go owns the U5 scene-as-texture preview. The Scene
// workspace's Render() builds an imgui.Image whose texture is the
// off-screen render of the editor's canvas; this file holds the
// ebiten.Game shim cimgui-go's CreateTextureFromGame asks for, plus
// the input-mapping helpers SceneWorkspace.Update uses to route
// mouse coords through the captured image rect into the canvas tool
// dispatch.
//
// Idea #1 v1 (ScreenRoom Mario-strip) extends sceneGame to also drive
// the live preview's world simulation: each tick the Player entity's
// position feeds a cached pixelforge_camera.Follower whose Update
// writes to the engine-wide pixelforge.Camera; each Draw, the bound
// tilemap layers and entity sprites are rendered through the same
// engine primitives (pixelforge_tilemap.Render, pixelforge_entity.
// RenderAll) that the shipped runtime will call. The structural
// guarantee: what the designer sees in the editor preview matches
// what ships, because both paths share the same engine renderers.
package editor

import (
	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_camera"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_entity"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_tilemap"
)

// scenePreviewW / scenePreviewH size the off-screen image the
// backend renders the scene into. Chosen to match the Pixelforge
// default canvas resolution so the texture's native pixels map 1:1 to
// project-space pixels when the image is shown at native size — the
// docked panel scales the image up/down via ImGui's bilinear filter.
const (
	scenePreviewW = 1280
	scenePreviewH = 720
)

// previewTickDt is the fixed dt the preview hands the camera
// follower per repaint. cimgui-go's CreateTextureFromGame does not
// expose the real frame delta to the embedded ebiten.Game; rather
// than thread a clock through the texture pipeline (a larger
// refactor than U5's brief), the preview commits to a fixed 1/60s
// step. The Follower's smoothing constants are framerate-
// independent (see pixelforge_camera.Follower.Update's comment on
// the dt*60 scaling), so the fixed-step choice does not bias the
// camera's settling distance — only the time-to-settle granularity,
// which is invisible at preview speeds.
const previewTickDt = 1.0 / 60.0

// SheetResolver looks up a decoded *pixelforge.Canvas for a sprite
// asset by name. The editor does not currently maintain a decoded
// canvas catalog for runtime use — SpriteAsset only carries the PNG's
// RelativePath. Rather than build a decoder pipeline in U5 (out of
// scope per the plan), the preview installs a no-op resolver by
// default; tests inject a fake that returns a synthetic Canvas so
// the tilemap-render integration can be observed. When (and if) a
// decoder lands, this hook is the single seam to wire it through.
//
// A nil resolver, or a resolver that returns (nil, false), causes
// the tilemap renderer to be skipped silently for that layer.
type SheetResolver func(sprite *pixelforge_project.SpriteAsset) (*pixelforge.Canvas, bool)

// sceneGame is the ebiten.Game shim the cimgui-go backend calls each
// frame to repaint the scene texture. It defers entirely to the
// editor's canvas drawing path — tool overlays, entity markers, and
// selection outlines all surface inside the image. The shim never
// touches the screen directly; cimgui-go composites it into the ImGui
// texture for the Scene window.
//
// v1 of idea #1 adds the world-preview fields:
//   - follower / followerSceneID together cache a single Follower
//     per active scene. The cache survives across frames so the
//     camera's lookahead + smoothing accumulators persist; when the
//     active scene changes the Follower is rebuilt with that scene's
//     Camera config. A map-per-scene is overkill in v1 (Editor only
//     has one active scene at a time per editor.activeScene), so a
//     single slot keyed by the scene's ID is enough.
//   - sheetResolver is the indirection point for decoding tilemap
//     sprite-sheets to a *pixelforge.Canvas. Default no-op; tests
//     inject a fake.
//   - tilemapRenderFn / entityRenderFn are the indirection points
//     for the engine renderers. Production wires them to
//     pixelforge_tilemap.Render / pixelforge_entity.RenderAll; tests
//     swap them for spies that record calls without needing a live
//     engine draw target.
type sceneGame struct {
	editor *Editor

	follower         *pixelforge_camera.Follower
	followerSceneID  string
	followerSceneCfg pixelforge_project.SceneCameraConfig
	followerBoundsW  int
	followerBoundsH  int

	sheetResolver SheetResolver

	tilemapRenderFn func(layer *pixelforge_project.TilemapLayer, sheet *pixelforge.Canvas)
	entityRenderFn  func(entities []pixelforge_project.Entity, sprites []pixelforge_project.SpriteAsset)
}

func newSceneGame(e *Editor) *sceneGame {
	return &sceneGame{
		editor:          e,
		tilemapRenderFn: pixelforge_tilemap.Render,
		entityRenderFn:  pixelforge_entity.RenderAll,
	}
}

// Update advances the scene-side simulation tick. v1 of idea #1
// extends the previous no-op shim with one responsibility: drive the
// camera follower from the active Player entity. The editor's own
// Update still owns tool dispatch; this hook owns the world-
// simulation side of the preview.
//
// When no Player is found, the camera state is left untouched (the
// global pixelforge.Camera stays at its last value, which is (0, 0)
// for a fresh editor). That keeps the preview deterministic for
// scenes that haven't been tagged with a Player yet.
func (g *sceneGame) Update() error {
	g.updateCamera(previewTickDt)
	return nil
}

// updateCamera is the testable seam for the U5 camera integration.
// Exposed as a method (rather than inlined into Update) so tests can
// drive it with controlled dt values + observe pixelforge.Camera
// without standing up the ebiten.Game loop.
func (g *sceneGame) updateCamera(dt float64) {
	if g == nil || g.editor == nil {
		return
	}
	scene := g.editor.activeScene()
	if scene == nil {
		return
	}
	player := findPlayerEntity(scene)
	if player == nil {
		// No Player tagged — leave pixelforge.Camera as-is. This is
		// the documented "no Player → camera still" contract.
		return
	}

	// ensureFollower may teleport the Player to Scene.SpawnTile on
	// the first frame of a scene (U7's spawn-flow integration), so
	// read playerX / playerY AFTER it runs to pick up the teleport
	// in the same tick. Otherwise the first frame would feed the
	// follower the pre-teleport position and trigger a one-frame
	// camera lurch.
	g.ensureFollower(scene, g.editor.project)
	if g.follower == nil {
		return
	}

	playerX := float64(player.TileX * pixelforge_entity.TileW)
	playerY := float64(player.TileY * pixelforge_entity.TileH)
	g.follower.Update(playerX, playerY, dt)
}

// ensureFollower lazily constructs (or rebuilds) the cached
// Follower so the active scene's Camera config + grid bounds are
// always reflected. The Follower is cached across frames so its
// lookahead easing and smoothing accumulators persist; a change in
// scene ID, camera config, or grid bounds invalidates the cache and
// rebuilds.
//
// Idea #1 v1 (U7) treats a scene-ID change as the preview-restart
// signal: when the cached scene ID does not match the active scene
// (first preview frame OR designer switched scenes OR external
// "Restart Preview" forced a rebuild by clearing the cache), the
// Player is teleported to Scene.SpawnTile via TeleportPlayerToSpawn.
// SpawnTile = (0, 0) is the documented "no override" sentinel — the
// helper leaves the Player at its authored position in that case so
// designers can preview without setting a spawn first.
func (g *sceneGame) ensureFollower(scene *pixelforge_project.Scene, project *pixelforge_project.Project) {
	boundsW, boundsH := scenePixelBounds(scene, project)

	if g.follower != nil &&
		g.followerSceneID == scene.ID &&
		g.followerSceneCfg == scene.Camera &&
		g.followerBoundsW == boundsW &&
		g.followerBoundsH == boundsH {
		return
	}

	sceneChanged := g.followerSceneID != scene.ID

	cfg := pixelforge_camera.FollowerConfig{
		DeadZoneW:  scene.Camera.DeadZoneW,
		DeadZoneH:  scene.Camera.DeadZoneH,
		LookAheadX: scene.Camera.LookAheadX,
		SmoothingX: scene.Camera.SmoothingX,
		SmoothingY: scene.Camera.SmoothingY,
		BoundsW:    boundsW,
		BoundsH:    boundsH,
	}
	g.follower = pixelforge_camera.NewFollower(cfg)
	g.followerSceneID = scene.ID
	g.followerSceneCfg = scene.Camera
	g.followerBoundsW = boundsW
	g.followerBoundsH = boundsH

	if sceneChanged {
		TeleportPlayerToSpawn(scene)
	}
}

// TeleportPlayerToSpawn moves the Player entity (the entity with
// Archetype == ArchetypePlayer) in scene to Scene.SpawnTile when
// SpawnTile is non-default. The default zero-value SpawnCell{0, 0} is
// the "no override" sentinel — in that case the Player is left at its
// authored TileX / TileY so designers can preview without first
// having to set a spawn cell.
//
// Wired into the preview lifecycle via ensureFollower's scene-change
// detection (first frame of a scene === preview restart in v1). A
// future "Restart Preview" button can also call this directly + clear
// the Follower cache to force a clean re-spawn mid-session.
//
// Both Position (legacy pixel coords) and TileX / TileY are updated
// so the camera follower (reads TileX / TileY) and the canvas's
// entity-marker hit-test (reads Position) agree on where the Player
// is after teleport.
func TeleportPlayerToSpawn(scene *pixelforge_project.Scene) {
	if scene == nil {
		return
	}
	if scene.SpawnTile == (pixelforge_project.SpawnCell{}) {
		// Default spawn → preserve authored position.
		return
	}
	player := findPlayerEntity(scene)
	if player == nil {
		return
	}
	player.TileX = scene.SpawnTile.Col
	player.TileY = scene.SpawnTile.Row
	player.Position.X = float64(scene.SpawnTile.Col * pixelforge_entity.TileW)
	player.Position.Y = float64(scene.SpawnTile.Row * pixelforge_entity.TileH)
}

// scenePixelBounds returns the (width, height) of the level in
// scene-space pixels. GridWidthScreens * 256 (ScreenWidth) for the
// horizontal axis, GridHeightScreens * 240 (ScreenHeight) for the
// vertical. Falls back to the project's ScreenWidth/Height when the
// scene's grid fields are zero (pre-defaults scenes built via
// NewProject without going through applyDefaults).
func scenePixelBounds(scene *pixelforge_project.Scene, project *pixelforge_project.Project) (int, int) {
	const screenW, screenH = 256, 240
	widthScreens := scene.GridWidthScreens
	heightScreens := scene.GridHeightScreens
	if widthScreens <= 0 {
		widthScreens = 1
	}
	if heightScreens <= 0 {
		heightScreens = 1
	}
	boundsW := widthScreens * screenW
	boundsH := heightScreens * screenH
	// Belt-and-braces: if the project still uses a non-256/240 screen
	// override (legacy pre-v1 fixtures), prefer the larger of the two
	// so the camera bounds never accidentally clip below the project's
	// declared viewport size.
	if project != nil {
		if project.ScreenWidth > boundsW {
			boundsW = project.ScreenWidth
		}
		if project.ScreenHeight > boundsH {
			boundsH = project.ScreenHeight
		}
	}
	return boundsW, boundsH
}

// findPlayerEntity returns the first entity in scene whose Archetype
// matches the v1 Player tag. v1 enforces mutual exclusivity in U7's
// right-click handler so "first match" is also the only match; if a
// future bug leaves multiple Players the renderer simply tracks the
// first.
func findPlayerEntity(scene *pixelforge_project.Scene) *pixelforge_project.Entity {
	if scene == nil {
		return nil
	}
	for i := range scene.Entities {
		if scene.Entities[i].Archetype == pixelforge_project.ArchetypePlayer {
			return &scene.Entities[i]
		}
	}
	return nil
}

// Draw paints the scene texture for this frame. cimgui-go calls this
// against an off-screen ebiten.Image sized (scenePreviewW × H). The
// world layers render first (tilemap → entity sprites) so the editor's
// own marker/selection overlay (drawn by canvas.Draw) sits on top —
// designers always see the selection outline even when an entity's
// sprite would otherwise hide it.
func (g *sceneGame) Draw(screen *ebiten.Image) {
	if g.editor == nil || g.editor.canvas == nil {
		return
	}
	screen.Fill(themeBackground)

	g.drawWorld()

	area := widgets.Rect{X: 0, Y: 0, W: scenePreviewW, H: scenePreviewH}
	g.editor.canvas.Draw(screen, area, g.editor)
}

// drawWorld renders the tilemap layers + entity sprites through the
// engine renderers. Pulled out of Draw so tests can exercise the
// renderer-dispatch path without an *ebiten.Image.
//
// The renderers are gated behind indirection points (tilemapRenderFn,
// entityRenderFn) so tests can swap them for spies, and the tilemap
// sheet resolution is gated behind sheetResolver so a layer with an
// undecoded sheet is silently skipped rather than crashing the
// preview. The sheet-decoder pipeline lives outside U5; when it
// lands it wires through sheetResolver.
func (g *sceneGame) drawWorld() {
	if g == nil || g.editor == nil {
		return
	}
	scene := g.editor.activeScene()
	if scene == nil {
		return
	}
	project := g.editor.project

	// Tilemap layers first (drawn back-to-front in slice order).
	if g.tilemapRenderFn != nil && project != nil {
		for i := range scene.Tilemaps {
			layer := &scene.Tilemaps[i]
			sheet, ok := g.resolveSheet(layer, project)
			if !ok {
				continue
			}
			g.tilemapRenderFn(layer, sheet)
		}
	}

	// Entity sprites on top of the tilemap. The Z-ordering hint
	// inside RenderAll handles per-entity layering.
	if g.entityRenderFn != nil && project != nil {
		g.entityRenderFn(scene.Entities, project.Sprites)
	}
}

// resolveSheet looks up the SpriteAsset bound to layer's
// SpriteSheetRef and returns its decoded *pixelforge.Canvas. The
// editor's sprite-decoder pipeline does not exist yet (a U5-out-of-
// scope concern), so the default resolver returns (nil, false) for
// every lookup and the tilemap render is a silent no-op. Tests
// install a fake resolver so the renderer-dispatch path is still
// observable.
func (g *sceneGame) resolveSheet(layer *pixelforge_project.TilemapLayer, project *pixelforge_project.Project) (*pixelforge.Canvas, bool) {
	if layer == nil || layer.SpriteSheetRef == "" || project == nil {
		return nil, false
	}
	for i := range project.Sprites {
		if project.Sprites[i].Name == layer.SpriteSheetRef {
			if g.sheetResolver == nil {
				return nil, false
			}
			return g.sheetResolver(&project.Sprites[i])
		}
	}
	return nil, false
}

// Layout returns the fixed preview resolution. The ImGui image scales
// to fit whatever size the docked Scene window has.
func (g *sceneGame) Layout(_, _ int) (int, int) {
	return scenePreviewW, scenePreviewH
}

// SceneTextureRef exposes the editor's scene preview texture so the
// SceneWorkspace.Render can hand it to imgui.Image. Returns the zero
// TextureRef when no live backend has registered one (test stubs,
// pre-attach state).
func (e *Editor) SceneTextureRef() (imgui.TextureRef, int, int, bool) {
	if e == nil || e.imgui == nil || !e.imgui.live || e.sceneTexW == 0 {
		return imgui.TextureRef{}, 0, 0, false
	}
	return e.sceneTexture, e.sceneTexW, e.sceneTexH, true
}

// mapMouseToSceneTexture converts a window-pixel mouse coordinate
// inside the Scene image rect into texture-local pixel coordinates
// (origin top-left, scaled to the texture's native resolution). The
// canvas tool dispatch consumes texture-local coords as if they were
// the captured panel rect, so existing viewBox + entity-marker math
// keeps working without modification.
func mapMouseToSceneTexture(imageRect widgets.Rect, texW, texH, mx, my int) (int, int, bool) {
	if imageRect.W <= 0 || imageRect.H <= 0 {
		return 0, 0, false
	}
	if mx < imageRect.X || mx >= imageRect.X+imageRect.W ||
		my < imageRect.Y || my >= imageRect.Y+imageRect.H {
		return 0, 0, false
	}
	localX := mx - imageRect.X
	localY := my - imageRect.Y
	scaledX := localX * texW / imageRect.W
	scaledY := localY * texH / imageRect.H
	return scaledX, scaledY, true
}
