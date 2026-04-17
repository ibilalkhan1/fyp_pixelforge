// Example of using the high-level audio API in Pixelforge.
//
// This program plays a sample at different pitches depending on the key pressed.
package main

import (
	_ "embed"
	"math"
	"slices"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
)

var (
	//go:embed "piano.raw"
	sampleRAW []byte
	//go:embed "piano.png"
	pianoPNG []byte
)

func main() {
	// Decode a raw sample file (8-bit mono, no header, no compression).
	sample := pixelforge_audio.DecodeRaw(sampleRAW, 16726)

	var selectedKey pixelforge_key.Key

	// Load palette and canvas from PNG file
	pixelforge.Palette = pixelforge.DecodePalette(pianoPNG)
	pianoCanvas := pixelforge.DecodeCanvas(pianoPNG)

	pixelforge.SetScreenSize(113, 46)
	pixelforge.SetTransparency(0, false) // disable transparency for color 0

	pixelforge.Init = func() {
		// The sample must be loaded before use,
		// but communication with the audio backend is only possible after starting the game.
		pixelforge_audio.LoadSample(sample)
	}

	pixelforge.Update = func() {
		// Check if any of the piano keys were just pressed
		for key, pitch := range buttonPitch {
			if pixelforge_key.Duration(key) == 1 {
				selectedKey = key
				// Play the sample on two channels: left and right — for stereo effect
				ch := pixelforge_audio.Chan1 | pixelforge_audio.Chan2
				vol := 1.0
				pixelforge_audio.Play(ch, sample, pitch, vol)
				break
			}
		}
	}

	pixelforge.Draw = func() {
		pixelforge.Cls()

		// Map each key color to either white (tone) or black (semitone)
		for key, color := range keyColors {
			if slices.Contains(semitoneLetters, key) {
				pixelforge.RemapColor(color, 0)
			} else {
				pixelforge.RemapColor(color, 7)
			}
		}

		// Highlight the last pressed key in gray
		if selectedKey != "" {
			pixelforge.RemapColor(keyColors[selectedKey], 6)
		}

		// Draw the piano image with updated color tables
		pixelforge.DrawCanvas(pianoCanvas, 0, 0)

		// Draw labels for each piano key
		printLetters()
	}

	pixelforge_ebiten.Run()
}

// cPitch = 1.0 is the base pitch (e.g. middle C).
// Change to 2.0 to play one octave higher, or 0.5 for one octave lower.
const cPitch = 1.0

// Maps keyboard keys to pitch multipliers.
var buttonPitch = map[pixelforge_key.Key]float64{
	pixelforge_key.Z:     cPitch,          // C
	pixelforge_key.S:     adjustPitch(1),  // C#
	pixelforge_key.X:     adjustPitch(2),  // D
	pixelforge_key.D:     adjustPitch(3),  // D#
	pixelforge_key.C:     adjustPitch(4),  // E
	pixelforge_key.V:     adjustPitch(5),  // F
	pixelforge_key.G:     adjustPitch(6),  // F#
	pixelforge_key.B:     adjustPitch(7),  // G
	pixelforge_key.H:     adjustPitch(8),  // G#
	pixelforge_key.N:     adjustPitch(9),  // A
	pixelforge_key.J:     adjustPitch(10), // A#
	pixelforge_key.M:     adjustPitch(11), // H
	pixelforge_key.Comma: adjustPitch(12), // C
}

func adjustPitch(i int) float64 {
	return cPitch * math.Pow(2, float64(i)/12.0)
}

// Layout of tone and semitone key labels on the piano image.
var (
	toneLetters     = []pixelforge_key.Key{"Z", "X", "C", "V", "B", "N", "M", ","}
	semitoneLetters = []pixelforge_key.Key{"S", "D", "", "G", "H", "J"}
)

const keyWidth = 13

func printLetters() {
	pixelforge.SetColor(16)
	for i, letter := range toneLetters {
		pixelforge_cofont.Print(string(letter), 9+i*keyWidth, 31)
	}

	for i, letter := range semitoneLetters {
		pixelforge_cofont.Print(string(letter), 16+i*keyWidth, 13)
	}
}

// Each key on the image uses a unique color
var keyColors = map[pixelforge_key.Key]pixelforge.Color{
	pixelforge_key.Z:     1,
	pixelforge_key.S:     2,
	pixelforge_key.X:     3,
	pixelforge_key.D:     4,
	pixelforge_key.C:     6,
	pixelforge_key.V:     8,
	pixelforge_key.G:     9,
	pixelforge_key.B:     10,
	pixelforge_key.H:     11,
	pixelforge_key.N:     12,
	pixelforge_key.J:     13,
	pixelforge_key.M:     14,
	pixelforge_key.Comma: 15,
}
