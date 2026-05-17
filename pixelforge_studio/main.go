// Pixelforge Studio — no-code visual game editor.
//
// The studio main.go is intentionally tiny: it loads user settings,
// sizes the window, and asks Ebitengine to drive the editor. All editor
// logic lives in the editor package.
//
// `pf-studio test [dir]` dispatches to the regression replayer; any
// other argv runs the editor.
package main

import (
	"log"
	"os"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capture"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/palette"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting"
)

const windowTitle = "Pixelforge Studio"

// imguiDemoFlag toggles the cimgui-go demo window. U1 of the ImGui
// migration plan uses this as the smoke signal that cimgui-go links and
// composes correctly. Removed once real chrome ships in U2.
const imguiDemoFlag = "--imgui-demo"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "test" {
		runTestSubcommand(os.Args[2:])
		return
	}

	settings := editor.LoadSettings()

	// Detect (and strip) the --imgui-demo flag so it doesn't confuse
	// any downstream argv consumers.
	showImguiDemo := slices.Contains(os.Args[1:], imguiDemoFlag)

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// Build the cimgui-go Ebiten backend before RunGame. CreateWindow
	// inside the backend drives ebiten.SetWindowTitle/Size, so the
	// editor's existing window-sizing path moves here.
	imguiBackend, err := editor.NewEbitenImguiBackend(windowTitle, settings.WindowWidth, settings.WindowHeight)
	if err != nil {
		log.Fatalf("pixelforge studio: %v", err)
	}

	e := editor.NewWithSettings(settings)
	e.AttachImguiBackend(imguiBackend, showImguiDemo)
	palette.RegisterWith(e)
	capture.RegisterWith(e)
	scripting.RegisterWith(e)

	if err := ebiten.RunGame(e); err != nil {
		log.Fatalf("pixelforge studio: %v", err)
	}
}
