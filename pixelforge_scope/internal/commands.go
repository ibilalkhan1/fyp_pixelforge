package internal

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_debug"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_snap"
	"log"
)

func enterConsoleMode() {
	log.Println("Entering console")
	prev := pixelforge.SetColor(*bgColor)
	pixelforge.Rect(0, 0, pixelforge.Screen().W()-1, pixelforge.Screen().H()-1)
	pixelforge.SetColor(prev)
	consoleMode = true
}

func exitConsoleMode() {
	log.Println("Exiting console")
	theScreenRecorder.ShowPrev()
	theScreenRecorder.Reset()
	consoleMode = false
	pixelforge_debug.SetPaused(false)
}

func captureSnapshot() {
	f, err := pixelforge_snap.CaptureOrErr()
	if err != nil {
		log.Println("Error capturing screenshot:", err)
	} else {
		log.Println("Screenshot saved to", f)
	}
}

func showPrevSnapshot() {
	pixelforge_debug.SetPaused(true)
	theScreenRecorder.ShowPrev()
}

func showNextSnapshot() {
	if !theScreenRecorder.ShowNext() {
		pixelforge_debug.SetPaused(false)
		pauseOnNextFrame = true
	}
}

func pauseOrResume() {
	if consoleMode {
		theScreenRecorder.GoToLast()

		pixelforge_debug.SetPaused(!pixelforge_debug.Paused())
	}
}
