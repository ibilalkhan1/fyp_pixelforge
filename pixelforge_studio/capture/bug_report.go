package capture

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	pirand "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_rand"
)

// DefaultBugReportFrames is the default number of trailing frames the
// packager includes. Two seconds at 30 TPS keeps the ZIP small while
// preserving enough context for the bug reporter.
const DefaultBugReportFrames = 60

// PackageReproZip writes a self-contained bug-repro ZIP to w.
//
// Layout (matches the M4 plan):
//
//	README.md                         project name, frame count, ffmpeg available?, system info
//	project.pforge                    the active project
//	project.pforge-assets/...         full recursive copy of the project's assets
//	capture/
//	  frames/0000.png ... NNNN.png    last `framesBack` frames
//	  input.log                       JSON lines, one per recorded input
//	  events.log                      JSON lines, one per non-input event
//	  seed.txt                        the pirand seed at recorder.Start()
//	system.txt                        runtime.GOOS/GOARCH + dependency versions
func PackageReproZip(rec *Recorder, project *pixelforge_project.Project, projectPath string, framesBack int, w io.Writer) error {
	if rec == nil {
		return fmt.Errorf("recorder is nil")
	}
	if project == nil {
		return fmt.Errorf("project is nil")
	}
	if w == nil {
		return fmt.Errorf("writer is nil")
	}
	if framesBack <= 0 {
		framesBack = DefaultBugReportFrames
	}
	frames := rec.Frames()
	if framesBack > len(frames) {
		framesBack = len(frames)
	}
	startIdx := len(frames) - framesBack

	zw := zip.NewWriter(w)
	defer zw.Close()

	// README.md
	readme, err := zw.Create("README.md")
	if err != nil {
		return fmt.Errorf("zip create README: %w", err)
	}
	if _, err := readme.Write(generateReadme(project, len(frames), framesBack)); err != nil {
		return fmt.Errorf("write README: %w", err)
	}

	// project.pforge — serialise via the project saver to a temp dir
	// then stream into the zip. The temp dir hosts the assets the
	// saver also writes.
	tmp, err := os.MkdirTemp("", "pf-repro-")
	if err != nil {
		return fmt.Errorf("mkdir tmp: %w", err)
	}
	defer os.RemoveAll(tmp)
	tmpProject := filepath.Join(tmp, "project.pforge")
	if err := project.Save(tmpProject); err != nil {
		return fmt.Errorf("save project snapshot: %w", err)
	}
	if err := addFileToZip(zw, tmpProject, "project.pforge"); err != nil {
		return err
	}

	// project.pforge-assets/ — recurse the *real* assets directory
	// alongside the live project file. The temp-dir's assets dir from
	// project.Save only carries what the saver writes itself.
	if projectPath != "" {
		srcAssets := pixelforge_project.AssetsDir(projectPath)
		if err := addTreeToZip(zw, srcAssets, "project.pforge-assets"); err != nil {
			return err
		}
	}

	// capture/frames + capture/input.log + capture/events.log + capture/seed.txt
	for i := 0; i < framesBack; i++ {
		frame := frames[startIdx+i]
		name := fmt.Sprintf("capture/frames/%04d.png", i)
		entry, err := zw.Create(name)
		if err != nil {
			return fmt.Errorf("zip create %s: %w", name, err)
		}
		// PNG-encode the frame.
		tmpFile := filepath.Join(tmp, fmt.Sprintf("frame-%04d.png", i))
		if err := writeFramePNG(tmpFile, frame); err != nil {
			return err
		}
		f, err := os.Open(tmpFile)
		if err != nil {
			return err
		}
		_, err = io.Copy(entry, f)
		f.Close()
		if err != nil {
			return err
		}
	}

	// Logs derived from the recorder.
	inputLog, err := zw.Create("capture/input.log")
	if err != nil {
		return err
	}
	if err := encodeFrameLog(inputLog, rec, startIdx, len(frames)-1, true); err != nil {
		return err
	}
	eventLog, err := zw.Create("capture/events.log")
	if err != nil {
		return err
	}
	if err := encodeFrameLog(eventLog, rec, startIdx, len(frames)-1, false); err != nil {
		return err
	}

	// seed.txt
	seedEntry, err := zw.Create("capture/seed.txt")
	if err != nil {
		return err
	}
	if _, err := seedEntry.Write([]byte(fmt.Sprintf("%d", rec.InitialSeed()))); err != nil {
		return err
	}

	// system.txt
	sysEntry, err := zw.Create("system.txt")
	if err != nil {
		return err
	}
	if _, err := sysEntry.Write(generateSystemInfo()); err != nil {
		return err
	}

	return nil
}

// encodeFrameLog walks frames[start..end] and writes either input or
// event entries as JSONL into w.
func encodeFrameLog(w io.Writer, rec *Recorder, start, end int, inputs bool) error {
	frames := rec.Frames()
	for i := start; i <= end && i < len(frames); i++ {
		frame := frames[i]
		if inputs {
			for _, e := range frame.Inputs {
				line := fmt.Sprintf("{\"tick\":%d,\"target\":%q,\"value\":%q}\n", frame.TickNumber, e.Target, e.Value)
				if _, err := w.Write([]byte(line)); err != nil {
					return err
				}
			}
		} else {
			for _, e := range frame.Events {
				line := fmt.Sprintf("{\"tick\":%d,\"target\":%q,\"value\":%q}\n", frame.TickNumber, e.Target, e.Value)
				if _, err := w.Write([]byte(line)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func generateReadme(project *pixelforge_project.Project, totalFrames, capturedFrames int) []byte {
	ffmpeg := "yes"
	if !FFmpegAvailable() {
		ffmpeg = "no (install from https://ffmpeg.org for MP4 export)"
	}
	return []byte(fmt.Sprintf(`# Pixelforge bug-repro

Project: %s
Screen: %dx%d
TPS: %d

Captured frames: %d (of %d total in the recorder ring)
Initial pirand seed: %d
ffmpeg available on origin machine: %s

## What to provide

When opening an issue, please describe:

- What you did just before the bug surfaced
- What you expected to happen
- What actually happened

The capture/ directory contains the last %d frames as PNGs plus the
recorded input and event logs. Replaying these via
"+"pf-studio-test"+" should reproduce the issue locally.
`, project.Name, project.ScreenWidth, project.ScreenHeight, project.TPS,
		capturedFrames, totalFrames, pirand.CurrentSeed(), ffmpeg, capturedFrames))
}

func generateSystemInfo() []byte {
	out := fmt.Sprintf("os=%s\narch=%s\nversion=%s\n",
		runtime.GOOS, runtime.GOARCH, runtime.Version())
	if bi, ok := debug.ReadBuildInfo(); ok {
		out += fmt.Sprintf("main_module=%s\n", bi.Main.Path)
		for _, m := range bi.Deps {
			out += fmt.Sprintf("dep=%s@%s\n", m.Path, m.Version)
		}
	}
	return []byte(out)
}

func addFileToZip(zw *zip.Writer, src, name string) error {
	entry, err := zw.Create(name)
	if err != nil {
		return err
	}
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(entry, f)
	return err
}

func addTreeToZip(zw *zip.Writer, srcRoot, prefix string) error {
	info, err := os.Stat(srcRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return addFileToZip(zw, srcRoot, prefix)
	}
	return filepath.WalkDir(srcRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		zipName := filepath.ToSlash(filepath.Join(prefix, rel))
		return addFileToZip(zw, path, zipName)
	})
}
