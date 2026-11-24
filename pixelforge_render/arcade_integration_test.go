//go:build !js

package pixelforge_render_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	pf "github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_arcade"
	render "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_render"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
)

func TestArcadePresetsUseSharedInputAndDrawing(t *testing.T) {
	for _, kind := range []string{"dungeon", "frogger", "city"} {
		t.Run(kind, func(t *testing.T) {
			oldW, oldH, oldTPS := pf.Screen().W(), pf.Screen().H(), pf.TPS()
			oldDraw, oldUpdate, oldInit, oldPalette := pf.Draw, pf.Update, pf.Init, pf.Palette
			t.Cleanup(func() {
				pf.Draw, pf.Update, pf.Init, pf.Palette = oldDraw, oldUpdate, oldInit, oldPalette
				pf.SetScreenSize(oldW, oldH)
				pf.SetTPS(oldTPS)
				render.ResetInputState()
			})
			project, assets, err := pixelforge_arcade.Project(kind)
			if err != nil {
				t.Fatal(err)
			}
			render.ApplyProject(project)
			rt, err := capsuleruntime.Boot(project, assets, capsuleruntime.Options{})
			if err != nil {
				t.Fatal(err)
			}
			defer rt.Close()
			pf.Draw = func() { render.DrawProject(rt) }
			pf.Update = func() {}
			pf.Init = nil
			keys := []ebiten.Key{ebiten.KeyArrowRight}
			if kind == "city" {
				keys = []ebiten.Key{ebiten.KeySpace}
			}
			if _, err := render.RenderTickAtRGBA(rt, 0, render.InputFrame{Keys: keys}); err != nil {
				t.Fatal(err)
			}
			if kind == "city" && !rt.Arcade.Driving {
				t.Fatal("Space did not enter the car")
			}
			x := rt.Arcade.X
			for tick := uint64(1); tick <= 12; tick++ {
				if _, err := render.RenderTickAtRGBA(rt, tick, render.InputFrame{
					Keys: []ebiten.Key{ebiten.KeyArrowRight},
				}); err != nil {
					t.Fatal(err)
				}
			}
			if rt.Arcade.X <= x {
				t.Fatal("shared input did not move preset player")
			}
			textPixels := 0
			for y := 0; y < 24; y++ {
				for x := 0; x < 320; x++ {
					if pf.Screen().Get(x, y) == 1 {
						textPixels++
					}
				}
			}
			if textPixels < 20 {
				t.Fatal("shared renderer omitted the HUD")
			}
		})
	}
}
