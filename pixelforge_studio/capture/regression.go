package capture

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	pirand "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_rand"
)

// ErrMissingProjectHash is returned when the regression directory's
// recorded project.pforge hashes differently from the active project
// at replay time. Indicates the project has changed since the
// regression was promoted.
var ErrMissingProjectHash = errors.New("regression replay: project hash mismatch")

// Result is the outcome of a single ReplayRegression call.
type Result struct {
	Passed bool
	// Detail carries human-readable failure context — e.g. a pixel
	// diff summary or "missing project file". Empty on success.
	Detail string
}

// PromoteFrameToRegression promotes frame index `frameIdx` from rec
// into a regression-test directory under regressionsRoot. The
// directory layout matches the M4 plan:
//
//	tests/regressions/<project-hash>/<testName>/
//	  golden.png         the captured frame as a PNG
//	  input.log          JSON lines, one per recorded input event up to frameIdx
//	  events.log         JSON lines, one per recorded non-input event up to frameIdx
//	  project.pforge     the project snapshot at promotion time
//	  seed.txt           the pirand seed captured at recorder.Start()
//
// Returns the absolute test directory.
func PromoteFrameToRegression(rec *Recorder, frameIdx int, project *pixelforge_project.Project, projectPath, regressionsRoot, testName string) (string, error) {
	if rec == nil {
		return "", fmt.Errorf("recorder is nil")
	}
	if project == nil {
		return "", fmt.Errorf("project is nil")
	}
	if testName == "" {
		return "", fmt.Errorf("testName must not be empty")
	}
	frame := rec.FrameAt(frameIdx)
	if frame == nil {
		return "", fmt.Errorf("frameIdx %d out of range (%d frames)", frameIdx, rec.FrameCount())
	}

	if err := os.MkdirAll(regressionsRoot, 0o755); err != nil {
		return "", fmt.Errorf("mkdir regressions root: %w", err)
	}
	// Project hash is intentionally a hash of stable identity (name +
	// screen dims + asset count). Hashing the entire serialized
	// project would drift on every save because ModifiedAt updates
	// each time — and the same project rerunning regressions back-to-
	// back must land in the same directory.
	identity := fmt.Sprintf("%s|%dx%d|sprites=%d|audio=%d",
		project.Name, project.ScreenWidth, project.ScreenHeight,
		len(project.Sprites), len(project.Audio))
	hash := sha256.Sum256([]byte(identity))
	hashStr := hex.EncodeToString(hash[:])[:12]

	testDir := filepath.Join(regressionsRoot, hashStr, testName)
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir test dir: %w", err)
	}

	// golden.png
	if err := writeFramePNG(filepath.Join(testDir, "golden.png"), frame); err != nil {
		return "", fmt.Errorf("write golden.png: %w", err)
	}

	// input.log + events.log: walk every frame [0..frameIdx] and
	// dump entries with their tick index.
	if err := writeLog(filepath.Join(testDir, "input.log"), rec, frameIdx, true); err != nil {
		return "", fmt.Errorf("write input.log: %w", err)
	}
	if err := writeLog(filepath.Join(testDir, "events.log"), rec, frameIdx, false); err != nil {
		return "", fmt.Errorf("write events.log: %w", err)
	}

	// project.pforge — saved via the regular saver. project.Save will
	// also create the *-assets/ directory; we copy referenced assets
	// from the original project's *-assets/ into it so the replay
	// loader can resolve every sprite/audio referenced in the JSON.
	destProjectPath := filepath.Join(testDir, "project.pforge")
	if err := project.Save(destProjectPath); err != nil {
		return "", fmt.Errorf("save project snapshot: %w", err)
	}
	if projectPath != "" {
		srcAssets := pixelforge_project.AssetsDir(projectPath)
		dstAssets := pixelforge_project.AssetsDir(destProjectPath)
		if err := copyTree(srcAssets, dstAssets); err != nil {
			return "", fmt.Errorf("copy assets: %w", err)
		}
	}

	// seed.txt
	seed := rec.InitialSeed()
	if err := os.WriteFile(filepath.Join(testDir, "seed.txt"), []byte(strconv.FormatUint(seed, 10)), 0o644); err != nil {
		return "", fmt.Errorf("write seed.txt: %w", err)
	}
	// projectPath is not stored — the loader uses the bundled snapshot.
	_ = projectPath
	return testDir, nil
}

// logEntry is the JSON shape written to input.log / events.log.
type logEntry struct {
	Tick   int    `json:"tick"`
	Target string `json:"target"`
	Value  string `json:"value"`
}

func writeLog(path string, rec *Recorder, lastFrame int, inputs bool) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for i := 0; i <= lastFrame; i++ {
		frame := rec.FrameAt(i)
		if frame == nil {
			break
		}
		if inputs {
			for _, e := range frame.Inputs {
				if err := enc.Encode(logEntry{Tick: frame.TickNumber, Target: e.Target, Value: e.Value}); err != nil {
					return err
				}
			}
		} else {
			for _, e := range frame.Events {
				if err := enc.Encode(logEntry{Tick: frame.TickNumber, Target: e.Target, Value: e.Value}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func writeFramePNG(path string, frame *Frame) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, frameToPalettedImage(frame))
}

func frameToPalettedImage(frame *Frame) *image.Paletted {
	palette := make(color.Palette, pixelforge.MaxColors)
	for i := range palette {
		mapped := frame.PaletteMapping[i] & (pixelforge.MaxColors - 1)
		rgb := frame.Palette[mapped]
		r, g, b := rgb.RGB()
		palette[i] = color.NRGBA{R: r, G: g, B: b, A: 255}
	}
	w := frame.Canvas.W()
	h := frame.Canvas.H()
	img := image.NewPaletted(image.Rect(0, 0, w, h), palette)
	copy(img.Pix, frame.Canvas.Data())
	return img
}

// ReplayRegression replays the regression rooted at testDir.
//
// M4 ships an in-process replay: the project loads, pirand seeds from
// seed.txt, and the live screen state is compared against golden.png.
// For now this is a structural check — the runtime hasn't yet
// reconstructed gameplay from input.log; that's a follow-up tracked in
// the master plan. The current behaviour:
//
//   - Verifies project.pforge loads cleanly.
//   - Verifies seed.txt is a valid uint64 (and reseeds pirand).
//   - Decodes golden.png and compares it against the active screen.
//
// Failures dump diff.png + diff.txt next to the regression.
func ReplayRegression(testDir string) (Result, error) {
	if _, err := os.Stat(testDir); err != nil {
		return Result{}, fmt.Errorf("regression dir %s: %w", testDir, err)
	}
	projectPath := filepath.Join(testDir, "project.pforge")
	if _, err := os.Stat(projectPath); err != nil {
		return Result{}, fmt.Errorf("missing project.pforge in %s", testDir)
	}
	if _, err := pixelforge_project.Load(projectPath); err != nil {
		// Surface ErrMissingAsset etc. as a passing-through error.
		return Result{}, fmt.Errorf("load project: %w", err)
	}
	// Reseed pirand from seed.txt.
	seedBytes, err := os.ReadFile(filepath.Join(testDir, "seed.txt"))
	if err != nil {
		return Result{}, fmt.Errorf("read seed.txt: %w", err)
	}
	seed, err := strconv.ParseUint(strings.TrimSpace(string(seedBytes)), 10, 64)
	if err != nil {
		return Result{}, fmt.Errorf("parse seed.txt: %w", err)
	}
	pirand.Seed(seed)

	// Decode golden.png.
	goldenPath := filepath.Join(testDir, "golden.png")
	gf, err := os.Open(goldenPath)
	if err != nil {
		return Result{}, fmt.Errorf("open golden.png: %w", err)
	}
	defer gf.Close()
	golden, err := png.Decode(gf)
	if err != nil {
		return Result{}, fmt.Errorf("decode golden.png: %w", err)
	}

	// Read the current screen and compare. The expectation here is
	// that the caller has either:
	//   (a) Run the project through enough ticks for the screen to
	//       match the golden — true headless replay (deferred).
	//   (b) Loaded the project + manually called ApplyFrameToScreen
	//       with the golden — what the M4 in-process replay does in
	//       the absence of a real game-loop reconstruction.
	// (b) is what shipping today covers; (a) is the M6+ headless work.
	pal, ok := golden.(*image.Paletted)
	if !ok {
		// Treat any non-paletted golden as a structural failure: we
		// promote paletted images only.
		return Result{Passed: false, Detail: "golden.png is not a paletted PNG"}, nil
	}

	// Cheap structural compare: equal dimensions + same first byte.
	// A full pixel compare is what the diff routine does on
	// hand-supplied screen state. For the CLI happy path the project
	// load + seed restore is the meaningful invariant.
	screen := pixelforge.Screen()
	if screen.W() != pal.Bounds().Dx() || screen.H() != pal.Bounds().Dy() {
		return Result{
			Passed: false,
			Detail: fmt.Sprintf("screen dimensions %dx%d differ from golden %dx%d",
				screen.W(), screen.H(), pal.Bounds().Dx(), pal.Bounds().Dy()),
		}, nil
	}

	// Pixel-level compare.
	diff := pixelDiff(screen.Data(), pal.Pix)
	if diff == 0 {
		return Result{Passed: true}, nil
	}

	// Dump diff.png + diff.txt.
	if err := writeDiffPNG(filepath.Join(testDir, "diff.png"), screen.Data(), pal); err == nil {
		_ = os.WriteFile(filepath.Join(testDir, "diff.txt"),
			[]byte(fmt.Sprintf("differing pixels: %d / %d\n", diff, len(pal.Pix))), 0o644)
	}
	return Result{
		Passed: false,
		Detail: fmt.Sprintf("%d pixel(s) differ; see diff.png", diff),
	}, nil
}

func pixelDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	d := 0
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			d++
		}
	}
	return d
}

func writeDiffPNG(path string, current []byte, golden *image.Paletted) error {
	w := golden.Bounds().Dx()
	h := golden.Bounds().Dy()
	diffImg := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(golden.Pix) && i < len(current); i++ {
		x := i % w
		y := i / w
		if current[i] == golden.Pix[i] {
			diffImg.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 255})
		} else {
			diffImg.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, diffImg)
}

// copyTree recursively copies src into dst. Missing src is a silent
// no-op so projects with no assets directory still round-trip.
func copyTree(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return copyFile(src, dst)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if err := copyTree(s, d); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// readLogEntries decodes a JSONL log file into entries.
func readLogEntries(path string) ([]logEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []logEntry
	dec := json.NewDecoder(f)
	for {
		var e logEntry
		if err := dec.Decode(&e); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}
