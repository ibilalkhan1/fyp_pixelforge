// Package pixelforge_genassets generates the sprite sheet variations
// and chiptune sound effects used by the snake example. Import it
// and call GenerateSnakeAssets() before starting the game so that
// all asset files are guaranteed to exist on disk.
package pixelforge_genassets

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
)

// GenerateSnakeAssets creates the three extra sprite sheets and the
// three sound effects inside pixelforge_examples/snake/.
// It is safe to call every launch; existing files are overwritten.
func GenerateSnakeAssets() {
	pal := pico8Palette()

	// Sprite sheets use indexed PNGs so that DecodePalette extracts
	// the full 32-colour palette and every index (0-31) is valid.
	genSpriteIndexed("pixelforge_examples/snake/sprites2.png", pal, spriteTheme{
		fruit: 8,  // red
		head:  11, // green
		body:  3,  // dark green
		eye:   7,  // white
	})
	genSpriteIndexed("pixelforge_examples/snake/sprites3.png", pal, spriteTheme{
		fruit: 10, // yellow
		head:  12, // blue
		body:  1,  // dark blue
		eye:   7,  // white
	})
	genSpriteIndexed("pixelforge_examples/snake/sprites4.png", pal, spriteTheme{
		fruit: 9,  // orange
		head:  18, // purple (Picotron)
		body:  2,  // dark purple
		eye:   7,  // white
	})

	// Sound effects land in the same folder.
	genWAV("pixelforge_examples/snake/eat.wav", squareWave(880, 0.10, 0.5))
	genWAV("pixelforge_examples/snake/crash.wav", noiseBurst(0.15, 0.8))
	genWAV("pixelforge_examples/snake/restart.wav",
		arpeggio([]float64{523.25, 659.25, 783.99, 1046.50}, 0.08))
}

// ─── Indexed Sprite Generator ────────────────────────────────────

type spriteTheme struct {
	fruit uint8
	head  uint8
	body  uint8
	eye   uint8
}

func genSpriteIndexed(path string, pal color.Palette, t spriteTheme) {
	img := image.NewPaletted(image.Rect(0, 0, 32, 8), pal)

	// Fruit at (0,0) – 8×8 circle-ish shape
	drawCircleIdx(img, 3, 3, 3, t.fruit)
	// Head vertical at (8,0) – 8×8 block with eye
	drawRectIdx(img, 8, 0, 8, 8, t.head)
	drawCircleIdx(img, 12, 3, 2, t.eye)
	// Head horizontal at (16,0) – 8×8 block with eye
	drawRectIdx(img, 16, 0, 8, 8, t.head)
	drawCircleIdx(img, 20, 3, 2, t.eye)
	// Body at (24,0) – 8×6 centred strip
	drawRectIdx(img, 24, 1, 8, 6, t.body)

	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func drawRectIdx(img *image.Paletted, x, y, w, h int, idx uint8) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			img.SetColorIndex(x+dx, y+dy, idx)
		}
	}
}

func drawCircleIdx(img *image.Paletted, cx, cy, r int, idx uint8) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				img.SetColorIndex(cx+x, cy+y, idx)
			}
		}
	}
}

// pico8Palette returns the 32 colours used by the Pico-8 / Picotron
// fantasy consoles. When DecodePalette reads an indexed PNG built
// from this palette, every index 0–31 maps to the expected colour.
func pico8Palette() color.Palette {
	return color.Palette{
		color.RGBA{0x00, 0x00, 0x00, 0xFF}, // 0  black
		color.RGBA{0x1D, 0x2B, 0x53, 0xFF}, // 1  dark blue
		color.RGBA{0x7E, 0x25, 0x53, 0xFF}, // 2  dark purple
		color.RGBA{0x00, 0x87, 0x51, 0xFF}, // 3  dark green
		color.RGBA{0xAB, 0x52, 0x36, 0xFF}, // 4  brown
		color.RGBA{0x5F, 0x57, 0x4F, 0xFF}, // 5  dark gray
		color.RGBA{0xC2, 0xC3, 0xC7, 0xFF}, // 6  light gray
		color.RGBA{0xFF, 0xF1, 0xE8, 0xFF}, // 7  white
		color.RGBA{0xFF, 0x00, 0x4D, 0xFF}, // 8  red
		color.RGBA{0xFF, 0xA3, 0x00, 0xFF}, // 9  orange
		color.RGBA{0xFF, 0xEC, 0x27, 0xFF}, // 10 yellow
		color.RGBA{0x00, 0xE4, 0x36, 0xFF}, // 11 green
		color.RGBA{0x29, 0xAD, 0xFF, 0xFF}, // 12 blue
		color.RGBA{0x83, 0x76, 0x9C, 0xFF}, // 13 indigo
		color.RGBA{0xFF, 0x77, 0xA8, 0xFF}, // 14 pink
		color.RGBA{0xFF, 0xCC, 0xAA, 0xFF}, // 15 peach
		color.RGBA{0x24, 0x63, 0xB0, 0xFF}, // 16 true-blue
		color.RGBA{0x00, 0xA5, 0xA1, 0xFF}, // 17 teal
		color.RGBA{0x65, 0x46, 0x88, 0xFF}, // 18 purple
		color.RGBA{0x12, 0x53, 0x59, 0xFF}, // 19 dark-teal
		color.RGBA{0x74, 0x2F, 0x29, 0xFF}, // 20 dark-brown
		color.RGBA{0x45, 0x2D, 0x32, 0xFF}, // 21 darker-grey
		color.RGBA{0xA2, 0x88, 0x79, 0xFF}, // 22 medium-grey
		color.RGBA{0xFF, 0xAC, 0xC5, 0xFF}, // 23 light-pink
		color.RGBA{0xB9, 0x00, 0x3E, 0xFF}, // 24 dark-red
		color.RGBA{0xE2, 0x6B, 0x02, 0xFF}, // 25 dark-orange
		color.RGBA{0x95, 0xF0, 0x42, 0xFF}, // 26 lime-green
		color.RGBA{0x00, 0xB2, 0x51, 0xFF}, // 27 medium-green
		color.RGBA{0x64, 0xDF, 0xF6, 0xFF}, // 28 light-blue
		color.RGBA{0xBD, 0x9A, 0xDF, 0xFF}, // 29 mauve
		color.RGBA{0xE4, 0x0D, 0xAB, 0xFF}, // 30 magenta
		color.RGBA{0xFF, 0x85, 0x57, 0xFF}, // 31 peach-2
	}
}

// ─── WAV generator ───────────────────────────────────────────────

const sampleRate = 44100

func genWAV(path string, samples []int8) {
	data := makeWAV(samples, sampleRate)
	if err := os.WriteFile(path, data, 0644); err != nil {
		panic(err)
	}
}

func makeWAV(samples []int8, rate int) []byte {
	data := make([]byte, len(samples))
	for i, s := range samples {
		data[i] = byte(s)
	}
	var buf bytes.Buffer
	// RIFF header
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+len(data)))
	buf.WriteString("WAVE")
	// fmt chunk
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16)) // chunk size
	binary.Write(&buf, binary.LittleEndian, uint16(1))  // PCM
	binary.Write(&buf, binary.LittleEndian, uint16(1))  // mono
	binary.Write(&buf, binary.LittleEndian, uint32(rate))
	binary.Write(&buf, binary.LittleEndian, uint32(rate)) // byte rate
	binary.Write(&buf, binary.LittleEndian, uint16(1))    // block align
	binary.Write(&buf, binary.LittleEndian, uint16(8))    // bits per sample
	// data chunk
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(len(data)))
	buf.Write(data)
	return buf.Bytes()
}

func squareWave(freq float64, durationSec, duty float64) []int8 {
	n := int(sampleRate * durationSec)
	samples := make([]int8, n)
	period := sampleRate / freq
	for i := 0; i < n; i++ {
		phase := float64(i%int(period)) / period
		if phase < duty {
			samples[i] = 64
		} else {
			samples[i] = -64
		}
	}
	return samples
}

func noiseBurst(durationSec, amp float64) []int8 {
	n := int(sampleRate * durationSec)
	samples := make([]int8, n)
	state := uint32(0xACE1)
	for i := 0; i < n; i++ {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		samples[i] = int8((float64(state&0xFF)/255.0 - 0.5) * amp * 127)
	}
	return samples
}

func arpeggio(freqs []float64, noteDurSec float64) []int8 {
	var samples []int8
	for _, f := range freqs {
		samples = append(samples, squareWave(f, noteDurSec, 0.5)...)
	}
	return samples
}
