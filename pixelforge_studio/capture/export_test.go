package capture

import (
	"bytes"
	"image/gif"
	"os"
	"path/filepath"
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportGIF_HappyPath(t *testing.T) {
	pixelforge.SetScreenSize(8, 8)
	rec := New(40)
	for i := 0; i < 30; i++ {
		pixelforge.SetColor(pixelforge.Color((i % 6) + 1))
		pixelforge.RectFill(0, 0, 7, 7)
		rec.SaveFrame()
	}
	var buf bytes.Buffer
	require.NoError(t, ExportGIF(rec, 0, 29, &buf, GIFOptions{LoopCount: 0}))

	g, err := gif.DecodeAll(&buf)
	require.NoError(t, err)
	assert.Equal(t, 30, len(g.Image))
	assert.Equal(t, 0, g.LoopCount)
}

func TestExportGIF_SingleFrame(t *testing.T) {
	pixelforge.SetScreenSize(4, 4)
	rec := New(4)
	rec.SaveFrame()
	var buf bytes.Buffer
	require.NoError(t, ExportGIF(rec, 0, 0, &buf, GIFOptions{}))

	g, err := gif.DecodeAll(&buf)
	require.NoError(t, err)
	assert.Len(t, g.Image, 1)
}

func TestExportGIF_ClampsRange(t *testing.T) {
	pixelforge.SetScreenSize(4, 4)
	rec := New(8)
	for i := 0; i < 4; i++ {
		rec.SaveFrame()
	}
	var buf bytes.Buffer
	require.NoError(t, ExportGIF(rec, -5, 99, &buf, GIFOptions{}))

	g, err := gif.DecodeAll(&buf)
	require.NoError(t, err)
	assert.Equal(t, 4, len(g.Image))
}

func TestExportGIF_NoFramesErrors(t *testing.T) {
	rec := New(4)
	var buf bytes.Buffer
	err := ExportGIF(rec, 0, 0, &buf, GIFOptions{})
	assert.Error(t, err)
}

func TestExportGIF_WriterFailure(t *testing.T) {
	pixelforge.SetScreenSize(4, 4)
	rec := New(8)
	rec.SaveFrame()
	err := ExportGIF(rec, 0, 0, failingWriter{}, GIFOptions{})
	assert.Error(t, err)
}

type failingWriter struct{}

func (failingWriter) Write(p []byte) (int, error) {
	return 0, assert.AnError
}

func TestFFmpegAvailable_RespectsLookuper(t *testing.T) {
	resetFFmpegCacheForTesting()
	defer resetFFmpegCacheForTesting()
	ffmpegLookuper = func(string) (string, error) { return "/fake/ffmpeg", nil }
	assert.True(t, FFmpegAvailable())

	resetFFmpegCacheForTesting()
	ffmpegLookuper = func(string) (string, error) { return "", os.ErrNotExist }
	assert.False(t, FFmpegAvailable())
}

func TestExportMP4_MissingFFmpegReturnsSentinel(t *testing.T) {
	resetFFmpegCacheForTesting()
	defer resetFFmpegCacheForTesting()
	ffmpegLookuper = func(string) (string, error) { return "", os.ErrNotExist }

	pixelforge.SetScreenSize(4, 4)
	rec := New(8)
	rec.SaveFrame()
	out := filepath.Join(t.TempDir(), "out.mp4")
	err := ExportMP4(rec, 0, 0, out, MP4Options{})
	assert.ErrorIs(t, err, ErrFFmpegMissing)
	_, statErr := os.Stat(out)
	assert.True(t, os.IsNotExist(statErr), "no MP4 file should be written")
}

func TestExportMP4_HappyPathSkipsWithoutFFmpeg(t *testing.T) {
	if !FFmpegAvailable() {
		t.Skip("ffmpeg not available")
	}
	pixelforge.SetScreenSize(8, 8)
	rec := New(8)
	for i := 0; i < 5; i++ {
		pixelforge.SetColor(pixelforge.Color((i % 6) + 1))
		pixelforge.RectFill(0, 0, 7, 7)
		rec.SaveFrame()
	}
	out := filepath.Join(t.TempDir(), "out.mp4")
	require.NoError(t, ExportMP4(rec, 0, 4, out, MP4Options{}))
	info, err := os.Stat(out)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}
