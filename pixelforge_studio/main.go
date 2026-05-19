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
	"context"
	"log"
	"os"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	ebitenaudio "github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/assetlibrary"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/audiolib"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/blackboard"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/build"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capture"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/dialogue"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/ingest"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/input"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/items"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/menus"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/palette"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting"
)

// studioAudioSampleRate is the sample rate the Paula backend
// requires (`pixelforge_ebiten/internal/audio.CtxSampleRate` is the
// internal source of truth at 48000). Duplicated here as a local
// const so studio main.go can build the ebitenaudio.Context without
// importing the internal package; piebiten.StartAudioBackend asserts
// the rate matches and panics otherwise, so a drift would be loud.
const studioAudioSampleRate = 48000

const windowTitle = "Pixelforge Studio"

// imguiDemoFlag toggles the cimgui-go demo window. U1 of the ImGui
// migration plan uses this as the smoke signal that cimgui-go links and
// composes correctly. Removed once real chrome ships in U2.
const imguiDemoFlag = "--imgui-demo"

// noAssetLibraryFlag opts out of the plan-008 U9 first-launch
// asset-library bootstrap. Useful for fully-offline operation
// (the bootstrap silently no-ops on network failure but still
// issues an HTTP request); pass at the command line to skip
// entirely.
const noAssetLibraryFlag = "--no-asset-library"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "test" {
		runTestSubcommand(os.Args[2:])
		return
	}

	settings := editor.LoadSettings()

	// Detect (and strip) the --imgui-demo flag so it doesn't confuse
	// any downstream argv consumers.
	showImguiDemo := slices.Contains(os.Args[1:], imguiDemoFlag)
	skipAssetLibrary := slices.Contains(os.Args[1:], noAssetLibraryFlag)

	// Kick off the asset-library bootstrap in the background so the
	// studio starts immediately; pack downloads land asynchronously
	// (and surface in the Library workspace's installed list). The
	// bootstrap is non-fatal on network failure — the studio is
	// fully usable without a populated library.
	var assetLib *assetlibrary.Library
	if !skipAssetLibrary {
		assetLib = assetlibrary.LaunchBackgroundBootstrap(context.Background())
	}
	_ = assetLib // U12 wires the workspace; for now we hold the handle

	// Plan-008 U11: custom-asset ingest. The dispatcher routes
	// classified extensions to the editor's per-kind import
	// runners. The watcher picks up files dropped into the cache's
	// user-library directory (the sibling of the curated
	// library/); the drag-drop poller (wired via editor hook after
	// the editor is constructed below) drains
	// ebiten.DroppedFiles each frame. Both layers run only when
	// the asset-library bootstrap wasn't suppressed —
	// --no-asset-library implies no ingest either.
	var ingestDispatcher *ingest.Dispatcher
	var ingestUserLibDir string
	if !skipAssetLibrary {
		ingestDispatcher = ingest.NewDispatcher()
		ingestUserLibDir = assetlibrary.UserLibraryDir(assetlibrary.StudioCacheRoot())
		watcher := ingest.NewWatcher(ingestUserLibDir, ingestDispatcher, ingest.DefaultDebounce)
		if err := watcher.Start(); err != nil {
			log.Printf("[ingest] watcher start failed (custom-asset ingest disabled): %v", err)
		}
	}

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// Build the cimgui-go Ebiten backend before RunGame. CreateWindow
	// inside the backend drives ebiten.SetWindowTitle/Size, so the
	// editor's existing window-sizing path moves here.
	imguiBackend, err := editor.NewEbitenImguiBackend(windowTitle, settings.WindowWidth, settings.WindowHeight)
	if err != nil {
		log.Fatalf("pixelforge studio: %v", err)
	}

	// Idea #4 v1 U1 — init Paula audio backend so the studio's
	// audition path (and any future Play call from the editor side)
	// works. Without this every Play crashes via panicBackend.
	// pixelforge_ebiten.StartAudioBackend asserts the sample rate
	// matches its internal constant; mismatches panic loudly so
	// drift surfaces immediately.
	audioCtx := ebitenaudio.NewContext(studioAudioSampleRate)
	pixelforge_audio.Backend = pixelforge_ebiten.StartAudioBackend(audioCtx)

	e := editor.NewWithSettings(settings)
	e.AttachImguiBackend(imguiBackend, showImguiDemo)
	palette.RegisterWith(e)
	capture.RegisterWith(e)
	scripting.RegisterWith(e)
	input.RegisterWith(e)
	blackboard.RegisterWith(e)
	audiolib.RegisterWith(e)
	dialogue.RegisterWith(e)
	items.RegisterWith(e)
	menus.RegisterWith(e)
	build.RegisterWith(e)
	if assetLib != nil {
		// Plan-008 U12: Library workspace. Registers after every
		// other workspace so its Ctrl+9 shortcut closes the
		// canonical Ctrl+5/6/7/8 series. Wired only when the
		// asset-library bootstrap is active.
		assetlibrary.RegisterWith(e, assetLib)
	}

	// Drag-drop poller — drains ebiten.DroppedFiles each tick
	// into the ingest dispatcher. Skipped when the ingest layer
	// is disabled (--no-asset-library).
	if ingestDispatcher != nil {
		dispatcher := ingestDispatcher
		userLibDir := ingestUserLibDir
		e.SetDragDropPoller(func() {
			_ = ingest.NewDragDrop(dispatcher, userLibDir, nil).Poll()
		})
	}

	if err := ebiten.RunGame(e); err != nil {
		log.Fatalf("pixelforge studio: %v", err)
	}
}
