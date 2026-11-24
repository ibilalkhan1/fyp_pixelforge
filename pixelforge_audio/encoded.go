package pixelforge_audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/hajimehoshi/go-mp3"
	"github.com/jfreymuth/oggvorbis"
)

const maxAudioBytes = 64 << 20
const maxAudioFrames = 48000 * 300

// DecodeAudio decodes supported files to the mixer's signed 8-bit mono format.
// Rates above 48 kHz are resampled. Decode limits bound memory during imports.
func DecodeAudio(data []byte, extension string) (*Sample, error) {
	if len(data) == 0 || len(data) > maxAudioBytes {
		return nil, errors.New("audio must be non-empty and at most 64 MiB")
	}
	switch strings.ToLower(strings.TrimPrefix(extension, ".")) {
	case "wav":
		return DecodeWAV(data)
	case "mp3":
		return DecodeMP3(data)
	case "ogg":
		return DecodeOGG(data)
	default:
		return nil, fmt.Errorf("unsupported audio format %q", extension)
	}
}

// DecodeMP3 decodes MPEG audio, downmixing its stereo PCM to one Paula channel.
func DecodeMP3(data []byte) (*Sample, error) {
	decoder, err := mp3.NewDecoder(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("MP3 decode: %w", err)
	}
	pcm, err := io.ReadAll(io.LimitReader(decoder, maxAudioBytes+1))
	if err != nil {
		return nil, fmt.Errorf("MP3 samples: %w", err)
	}
	if len(pcm) > maxAudioBytes {
		return nil, errors.New("decoded MP3 exceeds 64 MiB")
	}
	return signedPCM(pcm, decoder.SampleRate(), 2, 16, 1)
}

// DecodeOGG decodes Vorbis-in-OGG. Other codecs inside OGG are rejected.
func DecodeOGG(data []byte) (*Sample, error) {
	decoder, err := oggvorbis.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("OGG/Vorbis decode: %w", err)
	}
	channels := decoder.Channels()
	if channels < 1 || channels > 8 {
		return nil, errors.New("invalid Vorbis channel count")
	}
	mono := make([]float64, 0)
	buffer := make([]float32, 4096*channels)
	for {
		n, readErr := decoder.Read(buffer)
		if n%channels != 0 {
			return nil, errors.New("incomplete Vorbis frame")
		}
		for i := 0; i < n; i += channels {
			var sum float64
			for c := 0; c < channels; c++ {
				sum += float64(buffer[i+c])
			}
			mono = append(mono, sum/float64(channels))
		}
		if len(mono) > maxAudioFrames {
			return nil, errors.New("decoded audio exceeds sample limit")
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("vorbis samples: %w", readErr)
		}
		if n == 0 {
			return nil, io.ErrNoProgress
		}
	}
	return quantizeMono(mono, decoder.SampleRate())
}

// DecodeWAV accepts integer PCM (8/16/24/32-bit) and IEEE float (32/64-bit),
// with mono or multichannel input. The strict legacy DecodeWav API is retained.
func DecodeWAV(data []byte) (*Sample, error) {
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, errors.New("not a RIFF/WAVE file")
	}
	var rate, channels, bits, format int
	var pcm []byte
	for offset := 12; offset+8 <= len(data); {
		size := int64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		start := offset + 8
		end := int64(start) + size
		if end > int64(len(data)) {
			return nil, errors.New("WAV chunk out of bounds")
		}
		chunk := data[start:int(end)]
		switch string(data[offset : offset+4]) {
		case "fmt ":
			if len(chunk) < 16 {
				return nil, errors.New("short WAV format chunk")
			}
			format = int(binary.LittleEndian.Uint16(chunk))
			channels = int(binary.LittleEndian.Uint16(chunk[2:]))
			rate = int(binary.LittleEndian.Uint32(chunk[4:]))
			bits = int(binary.LittleEndian.Uint16(chunk[14:]))
		case "data":
			pcm = chunk
		}
		offset = int(end) + int(size%2)
	}
	return signedPCM(pcm, rate, channels, bits, format)
}

func signedPCM(data []byte, rate, channels, bits, format int) (*Sample, error) {
	if channels < 1 || channels > 8 || rate < 1 || rate > 192000 {
		return nil, errors.New("invalid audio channel count or sample rate")
	}
	integerPCM := format == 1 && (bits == 8 || bits == 16 || bits == 24 || bits == 32)
	floatPCM := format == 3 && (bits == 32 || bits == 64)
	if !integerPCM && !floatPCM {
		return nil, errors.New("unsupported PCM encoding")
	}
	stride := bits / 8
	frameBytes := stride * channels
	if len(data) == 0 || len(data)%frameBytes != 0 {
		return nil, errors.New("empty or incomplete PCM frames")
	}
	frames := len(data) / frameBytes
	if frames > maxAudioFrames {
		return nil, errors.New("decoded audio exceeds sample limit")
	}
	mono := make([]float64, frames)
	for frame := range mono {
		for channel := 0; channel < channels; channel++ {
			start := frame*frameBytes + channel*stride
			mono[frame] += pcmValue(data[start:start+stride], bits, format) / float64(channels)
		}
	}
	return quantizeMono(mono, rate)
}
func pcmValue(data []byte, bits, format int) float64 {
	if format == 3 {
		if bits == 32 {
			return float64(math.Float32frombits(binary.LittleEndian.Uint32(data)))
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(data))
	}
	switch bits {
	case 8:
		return float64(int(data[0])-128) / 128
	case 16:
		return float64(int16(binary.LittleEndian.Uint16(data))) / 32768
	case 24:
		value := int32(data[0]) | int32(data[1])<<8 | int32(data[2])<<16
		value = value << 8 >> 8
		return float64(value) / 8388608
	default:
		return float64(int32(binary.LittleEndian.Uint32(data))) / 2147483648
	}
}
func quantizeMono(mono []float64, rate int) (*Sample, error) {
	if rate < 1 || rate > 192000 || len(mono) == 0 {
		return nil, errors.New("empty audio or invalid sample rate")
	}
	target := min(rate, 48000)
	length := len(mono) * target / rate
	if length < 1 {
		return nil, errors.New("audio is too short")
	}
	pcm := make([]int8, length)
	for i := range pcm {
		position := float64(i) * float64(rate) / float64(target)
		index := int(position)
		fraction := position - float64(index)
		value := mono[index]*(1-fraction) + mono[min(index+1, len(mono)-1)]*fraction
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, errors.New("non-finite audio sample")
		}
		pcm[i] = int8(max(-128, min(127, math.Round(value*128))))
	}
	return NewSample(pcm, uint16(target)), nil
}
