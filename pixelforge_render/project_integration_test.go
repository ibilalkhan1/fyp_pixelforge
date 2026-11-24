//go:build !js

package pixelforge_render_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
	"testing/fstest"

	"github.com/hajimehoshi/ebiten/v2"
	pf "github.com/ibilalkhan1/fyp_pixelforge"
	project "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	render "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_render"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
	"github.com/stretchr/testify/require"
)

func TestProjectPipelineDrawsAssetMovesFromInputAndReleases(t *testing.T) {
	oldW, oldH, oldTPS := pf.Screen().W(), pf.Screen().H(), pf.TPS()
	oldCamera := pf.Camera
	oldDraw, oldUpdate, oldInit, oldPalette := pf.Draw, pf.Update, pf.Init, pf.Palette
	t.Cleanup(func() {
		pf.Draw = oldDraw
		pf.Update = oldUpdate
		pf.Init = oldInit
		pf.Palette = oldPalette
		pf.Camera = oldCamera
		pf.SetScreenSize(oldW, oldH)
		pf.SetTPS(oldTPS)
		render.ResetForTest()
	})
	render.ResetForTest()
	p := project.NewProject("pipeline-proof")
	p.ScreenWidth = 32
	p.ScreenHeight = 24
	p.Sprites = []project.SpriteAsset{{Name: "hero", RelativePath: "hero.png", FrameW: 2, FrameH: 2}}
	p.Scenes[0].Entities = []project.Entity{{
		ID: "hero", Position: project.EntityPosition{X: 8, Y: 8},
		Components: []project.EntityComponent{{Type: "Sprite", Values: map[string]any{"sprite": "hero"}}},
	}}
	p.Behaviors = []project.BehaviorGraph{{Name: "move", EventSheet: []project.EventSheetRule{{
		Conditions: []project.Condition{
			{Kind: "event_fired", Args: map[string]any{"target": "loop.main", "event": "update"}},
			{Kind: "key_held", Args: map[string]any{"key": "Right"}},
		},
		Actions: []project.Action{
			{Kind: "move_entity", Args: map[string]any{"entity": "hero", "dx": float64(2), "dy": float64(0)}},
		},
	}}}}
	artwork := image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{color.Black, color.White})
	for y := range 2 {
		for x := range 2 {
			artwork.SetColorIndex(x, y, 1)
		}
	}
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, artwork))
	assets := fstest.MapFS{"assets/hero.png": &fstest.MapFile{Data: encoded.Bytes()}}
	render.ApplyProject(p)
	pf.Camera = pf.Position{}
	pf.ResetPaletteMapping()
	pf.ResetColorTables()
	rt, err := capsuleruntime.Boot(p, assets, capsuleruntime.Options{})
	require.NoError(t, err)
	t.Cleanup(rt.Close)
	render.ApplyProject(p)
	entry, ok := capsuleruntime.LookupSprite("hero")
	require.True(t, ok)
	ink := entry.Canvas.Get(0, 0)
	pf.Palette[ink] = pf.RGB(0xffffff)
	pf.Palette[0] = pf.RGB(0)
	pf.Init = nil
	pf.Update = func() {}
	pf.Draw = func() { render.DrawProject(rt) }
	frame, err := render.RenderTickAtRGBA(rt, 1, render.InputFrame{Keys: []ebiten.Key{ebiten.KeyArrowRight}})
	require.NoError(t, err)
	require.InDelta(t, 10, p.Scenes[0].Entities[0].Position.X, 0.00001)
	require.Equal(t, uint8(255), frame.RGBAAt(10, 8).R, "real decoded sprite must be visible at moved position")
	require.Equal(t, uint8(0), frame.RGBAAt(8, 8).R, "old sprite position must be cleared")
	_, err = render.RenderTickAtRGBA(rt, 2, render.InputFrame{})
	require.NoError(t, err)
	require.InDelta(t, 10, p.Scenes[0].Entities[0].Position.X, 0.00001, "released key must stop authored motion")
	require.Equal(t, uint64(2), rt.Tick, "runtime housekeeping must advance once per simulated tick")
}
