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
	pidebug.SetPaused(false)
}

func captureSnapshot() {
	f, err := pisnap.CaptureOrErr()
	if err != nil {
		log.Println("Error capturing screenshot:", err)
	} else {
		log.Println("Screenshot saved to", f)
	}
}

func showPrevSnapshot() {
	pidebug.SetPaused(true)
	theScreenRecorder.ShowPrev()
}

func showNextSnapshot() {
	if !theScreenRecorder.ShowNext() {
		pidebug.SetPaused(false)
		pauseOnNextFrame = true
	}
}

func pauseOrResume() {
	if consoleMode {
		theScreenRecorder.GoToLast()

		pidebug.SetPaused(!pidebug.Paused())
	}
}
