//nolint:testpackage // Tests the PCM downmix helper as well as public format decoding.
package pixelforge_audio

import (
	"encoding/binary"
	"os"
	"testing"
)

func TestDecodeSupportedAudioProducesSamples(t *testing.T) {
	for _, name := range []string{"tone.mp3", "tone.ogg", "tone16.wav"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile("internal/test/" + name)
			if err != nil {
				t.Fatal(err)
			}
			extension := "wav"
			if name == "tone.mp3" {
				extension = "mp3"
			}
			if name == "tone.ogg" {
				extension = "ogg"
			}
			sample, err := DecodeAudio(data, extension)
			if err != nil {
				t.Fatal(err)
			}
			if sample == nil || sample.SampleRate() != 44100 || sample.Len() < 8000 {
				t.Fatalf("bad decoded sample: %#v", sample)
			}
			nonzero := false
			for _, value := range sample.Data() {
				if value != 0 {
					nonzero = true
					break
				}
			}
			if !nonzero {
				t.Fatal("decoded tone is silent")
			}
		})
	}
}
func TestDecodeAudioRejectsInvalidFiles(t *testing.T) {
	for _, ext := range []string{"wav", "mp3", "ogg", "midi"} {
		for _, data := range [][]byte{nil, []byte("OggS"), []byte("ID3"), []byte("not audio")} {
			if sample, err := DecodeAudio(data, ext); err == nil || sample != nil {
				t.Fatalf("accepted invalid %s file", ext)
			}
		}
	}
}
func TestPCMDownmixAndResample(t *testing.T) {
	pcm := make([]byte, 960*4)
	for i := 0; i < 960; i++ {
		binary.LittleEndian.PutUint16(pcm[i*4:], 16384)
		binary.LittleEndian.PutUint16(pcm[i*4+2:], 0)
	}
	sample, err := signedPCM(pcm, 96000, 2, 16, 1)
	if err != nil {
		t.Fatal(err)
	}
	if sample.SampleRate() != 48000 || sample.Len() != 480 || sample.Data()[0] != 32 {
		t.Fatalf("incorrect downmix/resample: %d Hz, %d frames, value %d",
			sample.SampleRate(), sample.Len(), sample.Data()[0])
	}
}
