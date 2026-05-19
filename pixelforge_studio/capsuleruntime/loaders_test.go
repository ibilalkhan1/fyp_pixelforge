package capsuleruntime_test

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_dialogue"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
)

// tinyPNG produces a 1x1 PNG the sprite loader can decode.
// Indexed PNGs against the default palette are the fast path; a
// 1x1 RGBA image takes the slow path but still produces a valid
// Canvas. We use RGBA so the test stays portable when the global
// palette changes shape.
func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// tinyWAV produces a 4-sample 8-bit mono PCM WAV at 8 kHz that
// DecodeWavOrErr accepts. WAV stores unsigned 8-bit samples (0..255);
// the decoder re-centres to signed int8 (-128..127) so we feed
// midpoint bytes here.
func tinyWAV(t *testing.T) []byte {
	t.Helper()
	const sampleRate = uint32(8000)
	const dataLen = uint32(4)
	const fmtChunkSize = uint32(16)
	const audioFormatPCM = uint16(1)
	const numChannels = uint16(1)
	const bitsPerSample = uint16(8)
	byteRate := sampleRate * uint32(numChannels) * uint32(bitsPerSample) / 8
	blockAlign := numChannels * bitsPerSample / 8

	var buf bytes.Buffer
	buf.WriteString("RIFF")
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, uint32(36+dataLen)))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, fmtChunkSize))
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, audioFormatPCM))
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, numChannels))
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, sampleRate))
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, byteRate))
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, blockAlign))
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, bitsPerSample))
	buf.WriteString("data")
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, dataLen))
	buf.Write([]byte{128, 128, 128, 128})
	return buf.Bytes()
}

// fakeAssets builds an fs.FS rooted at "assets/" with the supplied
// files. Mirrors the layout codegen.copyAssets produces inside the
// generated outDir.
func fakeAssets(files map[string][]byte) fstest.MapFS {
	out := fstest.MapFS{}
	for relPath, data := range files {
		out["assets/"+relPath] = &fstest.MapFile{Data: data}
	}
	return out
}

func resetAll() {
	capsuleruntime.ResetRegistriesForTest()
	pixelforge_audio.ResetSampleRegistryForTest()
	pixelforge_dialogue.ResetScriptRegistryForTest()
}

func TestBoot_LoadsSpritesRegistersByName(t *testing.T) {
	resetAll()
	p := pixelforge_project.NewProject("t")
	p.Sprites = []pixelforge_project.SpriteAsset{
		{Name: "ship", RelativePath: "sprites/ship.png", Width: 1, Height: 1},
		{Name: "rock", RelativePath: "sprites/rock.png", Width: 1, Height: 1},
		{Name: "shot", RelativePath: "sprites/shot.png", Width: 1, Height: 1},
	}
	assets := fakeAssets(map[string][]byte{
		"sprites/ship.png": tinyPNG(t),
		"sprites/rock.png": tinyPNG(t),
		"sprites/shot.png": tinyPNG(t),
	})

	_, err := capsuleruntime.Boot(p, assets, capsuleruntime.Options{SkipSubscribers: true})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"ship", "rock", "shot"}, capsuleruntime.AllSpriteNames())
	entry, ok := capsuleruntime.LookupSprite("ship")
	require.True(t, ok)
	assert.Equal(t, "ship", entry.Asset.Name)
}

func TestBoot_LoadsAudioRegistersByName(t *testing.T) {
	resetAll()
	p := pixelforge_project.NewProject("t")
	p.Audio = []pixelforge_project.AudioSample{
		{Name: "blast", RelativePath: "audio/blast.wav"},
		{Name: "ping", RelativePath: "audio/ping.wav"},
	}
	assets := fakeAssets(map[string][]byte{
		"audio/blast.wav": tinyWAV(t),
		"audio/ping.wav":  tinyWAV(t),
	})

	_, err := capsuleruntime.Boot(p, assets, capsuleruntime.Options{SkipSubscribers: true})
	require.NoError(t, err)
	assert.NotNil(t, pixelforge_audio.LookupSample("blast"))
	assert.NotNil(t, pixelforge_audio.LookupSample("ping"))
}

func TestBoot_LoadsDialogueRegistersByName(t *testing.T) {
	resetAll()
	p := pixelforge_project.NewProject("t")
	p.Dialogues = map[string]pixelforge_project.DialogueScript{
		"intro": {Name: "intro", Script: "ALICE: hello\nBOB: hi"},
	}

	_, err := capsuleruntime.Boot(p, fakeAssets(nil), capsuleruntime.Options{SkipSubscribers: true})
	require.NoError(t, err)
	tree := pixelforge_dialogue.LookupScript("intro")
	require.NotNil(t, tree)
	assert.Equal(t, 2, tree.Len())
}

func TestBoot_EmptyProjectIsNoErr(t *testing.T) {
	resetAll()
	p := pixelforge_project.NewProject("t")
	_, err := capsuleruntime.Boot(p, fakeAssets(nil), capsuleruntime.Options{SkipSubscribers: true})
	require.NoError(t, err)
	assert.Empty(t, capsuleruntime.AllSpriteNames())
	assert.Empty(t, pixelforge_audio.AllSampleNames())
	assert.Empty(t, pixelforge_dialogue.AllScriptNames())
}

func TestBoot_MalformedSpriteSurfacesPath(t *testing.T) {
	resetAll()
	p := pixelforge_project.NewProject("t")
	p.Sprites = []pixelforge_project.SpriteAsset{
		{Name: "bad", RelativePath: "sprites/bad.png", Width: 1, Height: 1},
	}
	assets := fakeAssets(map[string][]byte{
		"sprites/bad.png": []byte("not a png"),
	})

	_, err := capsuleruntime.Boot(p, assets, capsuleruntime.Options{SkipSubscribers: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sprites/bad.png")
	assert.Contains(t, err.Error(), "bad")
}

func TestBoot_MalformedWAVSurfacesName(t *testing.T) {
	resetAll()
	p := pixelforge_project.NewProject("t")
	p.Audio = []pixelforge_project.AudioSample{
		{Name: "broken", RelativePath: "audio/broken.wav"},
	}
	assets := fakeAssets(map[string][]byte{
		"audio/broken.wav": []byte("definitely not a riff"),
	})

	_, err := capsuleruntime.Boot(p, assets, capsuleruntime.Options{SkipSubscribers: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "broken")
}

func TestBoot_IdempotentReregistersSameNames(t *testing.T) {
	resetAll()
	p := pixelforge_project.NewProject("t")
	p.Sprites = []pixelforge_project.SpriteAsset{
		{Name: "ship", RelativePath: "sprites/ship.png", Width: 1, Height: 1},
	}
	p.Audio = []pixelforge_project.AudioSample{
		{Name: "blast", RelativePath: "audio/blast.wav"},
	}
	assets := fakeAssets(map[string][]byte{
		"sprites/ship.png": tinyPNG(t),
		"audio/blast.wav":  tinyWAV(t),
	})

	_, err := capsuleruntime.Boot(p, assets, capsuleruntime.Options{SkipSubscribers: true})
	require.NoError(t, err)
	_, err = capsuleruntime.Boot(p, assets, capsuleruntime.Options{SkipSubscribers: true})
	require.NoError(t, err)
	assert.Equal(t, []string{"ship"}, capsuleruntime.AllSpriteNames())
	assert.Equal(t, []string{"blast"}, pixelforge_audio.AllSampleNames())
}

func TestBoot_RegistersScenesAndItems(t *testing.T) {
	resetAll()
	p := pixelforge_project.NewProject("t")
	p.Scenes = append(p.Scenes, pixelforge_project.Scene{ID: "level_2", Name: "Level 2"})
	p.Items = []pixelforge_project.ItemDefinition{{ID: "key", Name: "Key"}}

	_, err := capsuleruntime.Boot(p, fakeAssets(nil), capsuleruntime.Options{SkipSubscribers: true})
	require.NoError(t, err)
	scene, ok := capsuleruntime.LookupScene("level_2")
	require.True(t, ok)
	assert.Equal(t, "Level 2", scene.Name)
	item, ok := capsuleruntime.LookupItem("key")
	require.True(t, ok)
	assert.Equal(t, "Key", item.Name)
}

func TestBoot_NilProjectErrors(t *testing.T) {
	resetAll()
	_, err := capsuleruntime.Boot(nil, fakeAssets(nil), capsuleruntime.Options{SkipSubscribers: true})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "nil"))
}
