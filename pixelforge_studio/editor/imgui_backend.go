// imgui_backend.go bridges cimgui-go's Ebiten backend into the editor's
// frame lifecycle. U1 of the ImGui migration plan
// (docs/plans/2026-05-17-001-refactor-pixelforge-studio-imgui-migration-plan.md)
// adds the integration without yet replacing any native chrome — the
// existing render path is unchanged; ImGui composes on top as the final
// step of Draw, and contributes only a gated demo window for U1.
package editor

import (
	"errors"
	"fmt"

	"github.com/AllenDang/cimgui-go/backend"
	ebitenbackend "github.com/AllenDang/cimgui-go/backend/ebiten-backend"
	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/hajimehoshi/ebiten/v2"
)

// imguiBackend abstracts the subset of cimgui-go's Ebiten backend that
// the editor drives each frame. The concrete production type is
// *ebitenbackend.EbitenBackend; tests inject a stub to avoid spinning
// up an OpenGL context.
type imguiBackend interface {
	BeginFrame()
	EndFrame()
	Draw(screen *ebiten.Image)
	Layout(outsideWidth, outsideHeight int) (int, int)
	// CreateTextureFromGame registers an ebiten.Game as a backend
	// texture. The backend's render pipeline calls game.Update + Draw
	// each frame, blits the resulting image into the texture, and
	// makes the TextureRef usable in imgui.Image calls. Used by U5 to
	// render the Pixelforge scene preview as a docked image panel.
	CreateTextureFromGame(game ebiten.Game, width, height int) imgui.TextureRef
}

// imguiHost wraps an imguiBackend with editor-specific concerns: the
// --imgui-demo gate, nil-safety so existing tests that never attach a
// backend keep working, the live flag (true only when wrapping a real
// cimgui-go backend), and the ImGui-frame content the editor wants to
// build between BeginFrame and EndFrame.
type imguiHost struct {
	backend  imguiBackend
	showDemo bool
	// live distinguishes a real cimgui-go backend (whose
	// BeginFrame/EndFrame stand up a live C-side ImGui context) from
	// a test stub. Editor.buildChrome must not issue imgui.* C calls
	// when live is false — those would segfault without a context.
	live bool
}

// newImguiHostWithBackend wraps an already-constructed test stub
// backend. live=false because no real ImGui context exists.
func newImguiHostWithBackend(b imguiBackend, showDemo bool) *imguiHost {
	return &imguiHost{backend: b, showDemo: showDemo, live: false}
}

// newImguiHostWithLiveBackend wraps the real cimgui-go Ebiten backend.
// Used by AttachImguiBackend; once this is in place, Editor.buildChrome
// is free to call imgui.* C functions.
func newImguiHostWithLiveBackend(b imguiBackend, showDemo bool) *imguiHost {
	return &imguiHost{backend: b, showDemo: showDemo, live: true}
}

// NewEbitenImguiBackend builds the cimgui-go Ebiten backend, registers
// it as the global cimgui backend, and creates the ImGui window/context.
// main.go calls this once before ebiten.RunGame.
//
// The returned backend is the concrete *ebitenbackend.EbitenBackend so
// callers can drive ImGui-specific APIs (texture creation, etc.) in
// later units; it also satisfies the imguiBackend interface used inside
// the editor.
func NewEbitenImguiBackend(title string, width, height int) (*ebitenbackend.EbitenBackend, error) {
	eb := ebitenbackend.NewEbitenBackend()
	if _, err := backend.CreateBackend(eb); err != nil && !errors.Is(err, backend.CExposerError) {
		return nil, fmt.Errorf("pixelforge studio: cimgui-go backend init: %w", err)
	}
	// CreateWindow allocates the imgui.Context, sets backend flags, and
	// drives ebiten.SetWindowTitle/Size. main.go relies on it for the
	// initial window dimensions instead of calling ebiten.SetWindow*
	// separately.
	eb.CreateWindow(title, width, height)
	return eb, nil
}

// AttachImguiBackend wires the editor to a real cimgui-go backend.
// Safe to call once after NewWithSettings; calling twice replaces the
// host (the previous backend is dropped — its lifecycle is owned by
// the caller). Called by main.go with the production Ebiten backend;
// tests should use AttachImguiBackendStub when they want to assert
// against BeginFrame/EndFrame/Draw/Layout without standing up a real
// ImGui context.
//
// showDemo toggles the Dear ImGui demo window, used as the U1 smoke
// signal that cimgui-go links and composes correctly.
func (e *Editor) AttachImguiBackend(b imguiBackend, showDemo bool) {
	e.imgui = newImguiHostWithLiveBackend(b, showDemo)
	// U5: register the scene preview as a backend texture so the Scene
	// workspace can render the game canvas via imgui.Image. The texture
	// dimensions come from the project's logical screen — fallback to
	// the editor canvas size when no project is loaded.
	e.sceneGame = newSceneGame(e)
	e.sceneTexture = b.CreateTextureFromGame(e.sceneGame, scenePreviewW, scenePreviewH)
	e.sceneTexW, e.sceneTexH = scenePreviewW, scenePreviewH
	// U6: route ImGui's layout persistence file at the user config
	// dir so dock arrangement survives restarts (R6). Safe to call
	// after AttachImguiBackend because CreateBackend stood up the
	// imgui.Context first.
	configureImguiIniPath()
}

// AttachImguiBackendStub wires the editor to a test stub. The chrome
// build path (which issues imgui.* C calls) is gated off so unit tests
// can drive Update/Draw/Layout without segfaulting in cgo.
func (e *Editor) AttachImguiBackendStub(b imguiBackend) {
	e.imgui = newImguiHostWithBackend(b, false)
}

// beginFrame opens an ImGui frame. No-op when no backend is attached so
// tests that don't exercise ImGui continue to work.
func (h *imguiHost) beginFrame() {
	if h == nil || h.backend == nil {
		return
	}
	h.backend.BeginFrame()
}

// buildContent builds the ImGui content for this frame. U1 ships only
// the gated demo window; subsequent units extend this to draw the menu
// bar, dock space, panels, inspector, and workspaces.
func (h *imguiHost) buildContent() {
	if h == nil || h.backend == nil {
		return
	}
	if h.showDemo {
		open := true
		imgui.ShowDemoWindowV(&open)
	}
}

// endFrame closes the ImGui frame opened by beginFrame.
func (h *imguiHost) endFrame() {
	if h == nil || h.backend == nil {
		return
	}
	h.backend.EndFrame()
}

// draw composites the ImGui draw lists onto screen. Called from
// Editor.Draw as the last step so ImGui overlays sit above native chrome.
func (h *imguiHost) draw(screen *ebiten.Image) {
	if h == nil || h.backend == nil {
		return
	}
	h.backend.Draw(screen)
}

// layout forwards the outside dimensions to the backend so ImGui knows
// the display size when laying out windows.
func (h *imguiHost) layout(outsideWidth, outsideHeight int) {
	if h == nil || h.backend == nil {
		return
	}
	h.backend.Layout(outsideWidth, outsideHeight)
}
