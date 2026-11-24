package editor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	piaudio "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	pf "github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cart"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	render "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_render"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/codegen"
	scripting "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/runtime"
)

type playPreview struct {
	runtime                                    *capsuleruntime.Runtime
	tick                                       uint64
	accumulator                                int
	previousPalette                            pf.PaletteArray
	previousCamera                             pf.Position
	previousWidth, previousHeight, previousTPS int
	previousDraw                               func()
	surface                                    *ebiten.Image
}

// StartPreview runs a detached copy so playing never mutates the authored scene.
func (e *Editor) StartPreview() error {
	if e.preview != nil {
		return nil
	}
	if e.project == nil {
		return errors.New("open a project before playing")
	}
	encoded, err := json.Marshal(e.project)
	if err != nil {
		return fmt.Errorf("start preview: %w", err)
	}
	project, err := pixelforge_project.LoadReader(bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("start preview: %w", err)
	}
	cart, err := codegen.EncodeCartWithAssets(project, e.currentProjectPath)
	if err != nil {
		e.SetStatusMessage("Play failed: " + err.Error())
		return fmt.Errorf("start preview: %w", err)
	}
	_, assets, err := pixelforge_cart.DecodeBundle(cart)
	if err != nil {
		return fmt.Errorf("start preview: %w", err)
	}
	state := &playPreview{
		previousPalette: pf.Palette, previousCamera: pf.Camera,
		previousWidth: pf.Screen().W(), previousHeight: pf.Screen().H(),
		previousTPS: pf.TPS(), previousDraw: pf.Draw,
	}
	render.ApplyProject(project)
	rt, err := capsuleruntime.Boot(project, assets, capsuleruntime.Options{})
	if err != nil {
		pf.Palette = state.previousPalette
		pf.SetScreenSize(state.previousWidth, state.previousHeight)
		pf.SetTPS(state.previousTPS)
		e.SetStatusMessage("Play failed: " + err.Error())
		return fmt.Errorf("start preview: %w", err)
	}
	state.runtime = rt
	pf.Camera = pf.Position{}
	state.surface = ebiten.NewImage(pf.Screen().W(), pf.Screen().H())
	pf.Draw = func() { render.DrawProject(rt) }
	e.preview = state
	e.SetStatusMessage("Playing · keyboard controls the game · Stop restores your scene")
	return nil
}
func (e *Editor) StopPreview() {
	if e.preview == nil {
		return
	}
	state := e.preview
	state.runtime.Close()
	piaudio.ClearChan(15, 0)
	e.preview = nil
	pf.Draw = state.previousDraw
	pf.Palette = state.previousPalette
	pf.Camera = state.previousCamera
	pf.SetScreenSize(state.previousWidth, state.previousHeight)
	pf.SetTPS(state.previousTPS)
	render.ResetInputState()
	e.SetStatusMessage("Stopped · authored scene restored")
}
func (e *Editor) PreviewRunning() bool { return e != nil && e.preview != nil }
func (e *Editor) PreviewEngine() *scripting.Engine {
	if !e.PreviewRunning() {
		return nil
	}
	return e.preview.runtime.Behaviours()
}
func (e *Editor) updatePreview() {
	if e.preview == nil {
		return
	}
	state := e.preview
	state.accumulator += state.runtime.Project.TPS
	if state.accumulator < ebiten.TPS() {
		return
	}
	state.accumulator -= ebiten.TPS()
	keys := inpututil.AppendPressedKeys(nil)
	if e.imguiCapturesKeyboard() {
		keys = nil
	}
	inputs := render.InputFrame{Keys: keys}
	if err := render.RenderTickAtScreen(state.runtime, state.surface, state.tick, inputs); err != nil {
		e.SetStatusMessage(err.Error())
		e.StopPreview()
		return
	}
	state.tick++
}
