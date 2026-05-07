package internal

import (
	pixelforge_event "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	pixelforge_key "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
)

func handleInputInConsoleMode() {
	right := pixelforge_key.Duration(pixelforge_key.Right)
	if right > 0 {
		if right == 1 || right > 10 {
			showNextSnapshot()
		}
	} else {
		left := pixelforge_key.Duration(pixelforge_key.Left)
		if left == 1 || left > 10 {
			showPrevSnapshot()
		}
	}
}

func registerShortcuts() {
	// CTRL+SHIFT+I
	onCtrlShiftI := func() {
		if !consoleMode {
			enterConsoleMode()
		} else {
			exitConsoleMode()
		}
	}
	pixelforge_key.RegisterShortcut(onCtrlShiftI, pixelforge_key.Ctrl, pixelforge_key.Shift, pixelforge_key.I)

	// F12
	f12Down := pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.F12}
	pixelforge_key.DebugTarget().Subscribe(f12Down, func(pixelforge_key.Event, pixelforge_event.Handler) {
		if consoleMode {
			captureSnapshot()
		}
	})

	// Space
	spaceDown := pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.Space}
	pixelforge_key.DebugTarget().Subscribe(spaceDown, func(pixelforge_key.Event, pixelforge_event.Handler) {
		pauseOrResume()
	})

	// Esc
	escDown := pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.Esc}
	pixelforge_key.DebugTarget().Subscribe(escDown, func(pixelforge_key.Event, pixelforge_event.Handler) {
		if consoleMode {
			exitConsoleMode()
		}
	})
}
