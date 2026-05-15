package capture

import (
	"fmt"
	"image/gif"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// GIFOptions configures ExportGIF.
type GIFOptions struct {
	// LoopCount is the number of times the GIF loops. 0 = loop
	// forever; n > 0 plays n+1 times then stops (matches gif.GIF
	// semantics).
	LoopCount int

	// Delay is the inter-frame delay in 100ths-of-a-second units (the
	// raw gif.GIF.Delay unit). Default = 100/TPS so a 30 TPS recording
	// plays at ~30 fps (delay = 3 ⇒ ~33 ms per frame).
	Delay int
}

// ExportGIF encodes the [start, end] frame range of rec to w as an
// animated GIF. Returns an error if writing fails or the range is
// empty.
func ExportGIF(rec *Recorder, start, end int, w io.Writer, opts GIFOptions) error {
	if rec == nil {
		return fmt.Errorf("recorder is nil")
	}
	if w == nil {
		return fmt.Errorf("writer is nil")
	}
	frames := rec.Frames()
	if len(frames) == 0 {
		return fmt.Errorf("no frames to export")
	}
	if start > end {
		start, end = end, start
	}
	if start < 0 {
		start = 0
	}
	if end >= len(frames) {
		end = len(frames) - 1
	}
	if opts.Delay <= 0 {
		opts.Delay = 100 / 30 // ~33 ms / frame at 30 TPS; gif.GIF.Delay is in 1/100s ticks
		if opts.Delay <= 0 {
			opts.Delay = 1
		}
	}

	g := &gif.GIF{
		LoopCount: opts.LoopCount,
		Image:     nil,
		Delay:     nil,
	}
	for i := start; i <= end; i++ {
		img := frameToPalettedImage(frames[i])
		g.Image = append(g.Image, img)
		g.Delay = append(g.Delay, opts.Delay)
	}
	return gif.EncodeAll(w, g)
}

// MP4Options configures ExportMP4.
type MP4Options struct {
	// Framerate the encoder runs at. Default 30.
	Framerate int
	// Codec is the ffmpeg `-c:v` value. Default "libx264".
	Codec string
	// PixFmt is the ffmpeg `-pix_fmt` value. Default "yuv420p".
	PixFmt string
}

// ErrFFmpegMissing is returned when ExportMP4 cannot find ffmpeg in PATH.
var ErrFFmpegMissing = fmt.Errorf("ffmpeg not found in PATH; install from https://ffmpeg.org")

// ErrFFmpeg wraps a non-zero ffmpeg exit with stderr context.
type ErrFFmpeg struct {
	Stderr string
	Err    error
}

func (e *ErrFFmpeg) Error() string {
	tail := e.Stderr
	if len(tail) > 1024 {
		tail = "..." + tail[len(tail)-1024:]
	}
	return fmt.Sprintf("ffmpeg failed: %v\n%s", e.Err, tail)
}

func (e *ErrFFmpeg) Unwrap() error { return e.Err }

var (
	ffmpegOnce     sync.Once
	ffmpegPath     string
	ffmpegFound    bool
	ffmpegLookuper = exec.LookPath
)

// FFmpegAvailable reports whether ExportMP4 can succeed on this system.
// The result is cached on first call.
func FFmpegAvailable() bool {
	ffmpegOnce.Do(func() {
		p, err := ffmpegLookuper("ffmpeg")
		if err != nil {
			return
		}
		ffmpegPath = p
		ffmpegFound = true
	})
	return ffmpegFound
}

// resetFFmpegCacheForTesting flushes the FFmpegAvailable() cache. Tests
// only — production callers should never need to retry detection.
func resetFFmpegCacheForTesting() {
	ffmpegOnce = sync.Once{}
	ffmpegPath = ""
	ffmpegFound = false
}

// ExportMP4 encodes the [start, end] frame range to outPath via ffmpeg.
// Frames are first written to a temporary directory as PNGs, then
// ffmpeg muxes them into an MP4.
func ExportMP4(rec *Recorder, start, end int, outPath string, opts MP4Options) error {
	if rec == nil {
		return fmt.Errorf("recorder is nil")
	}
	if !FFmpegAvailable() {
		return ErrFFmpegMissing
	}
	frames := rec.Frames()
	if len(frames) == 0 {
		return fmt.Errorf("no frames to export")
	}
	if start > end {
		start, end = end, start
	}
	if start < 0 {
		start = 0
	}
	if end >= len(frames) {
		end = len(frames) - 1
	}
	if opts.Framerate <= 0 {
		opts.Framerate = 30
	}
	if opts.Codec == "" {
		opts.Codec = "libx264"
	}
	if opts.PixFmt == "" {
		opts.PixFmt = "yuv420p"
	}

	tmp, err := os.MkdirTemp("", "pf-mp4-")
	if err != nil {
		return fmt.Errorf("mkdir tmp: %w", err)
	}
	defer os.RemoveAll(tmp)

	for i := start; i <= end; i++ {
		idx := i - start
		framePath := filepath.Join(tmp, fmt.Sprintf("%04d.png", idx))
		if err := writeFramePNG(framePath, frames[i]); err != nil {
			return fmt.Errorf("write frame %d: %w", idx, err)
		}
	}

	args := []string{
		"-y",
		"-framerate", fmt.Sprintf("%d", opts.Framerate),
		"-i", filepath.Join(tmp, "%04d.png"),
		"-c:v", opts.Codec,
		"-pix_fmt", opts.PixFmt,
		outPath,
	}
	cmd := exec.Command(ffmpegPath, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return &ErrFFmpeg{Stderr: stderr.String(), Err: err}
	}
	return nil
}
