// import.go is idea #4 v1 U4's canonical audio import seam. Two
// public entry points (one per source kind) share the same core:
// decode via DecodeWavOrErr, copy bytes to AssetsDir(ppath)/audio/,
// append an AudioSample to the project.
//
// Mirrors palette.Import's shape exactly so the editor's File →
// Import WAV menu (U8) and the library Bind button (U6) wire
// through the same code path. SubPalette is to PaletteAsset what
// SuggestedChannelPriority is to AudioSample — both are inferred
// from the import source's metadata (library declaration when
// available, sensible default otherwise).
package audiolib

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// ImportResult summarises a successful import. Carries enough for
// the diff modal / toast UI (U8) to render a status line.
type ImportResult struct {
	SampleName   string
	RelativePath string
	DurationMs   int
	IsBGM        bool
}

// ImportFromFile reads a WAV from the on-disk pngPath, validates,
// copies into the project's assets dir, and appends an AudioSample
// with sensible defaults (sfx priority, loop=false). Designers can
// edit those defaults in the inspector after import.
func ImportFromFile(path string, p *pixelforge_project.Project, projectSourcePath string) (ImportResult, error) {
	if path == "" {
		return ImportResult{}, errors.New("audio import: empty path")
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return ImportResult{}, fmt.Errorf("audio import: read %s: %w", path, err)
	}
	baseName := stripExt(filepath.Base(path))
	return importCore(bytes, baseName, p, projectSourcePath, "", false)
}

// ImportFromLibrary materializes a library patch into the project.
// Looks up the patch by name, then runs the same core path as
// ImportFromFile but carries the patch's SuggestedPriority + IsBGM
// onto the AudioSample so library-sourced samples behave correctly
// (BGM loops, SFX doesn't) without designer intervention.
func ImportFromLibrary(patchName string, p *pixelforge_project.Project, projectSourcePath string) (ImportResult, error) {
	patch, ok := FindPatch(patchName)
	if !ok {
		return ImportResult{}, fmt.Errorf("audio import: patch %q not found in library", patchName)
	}
	return importCore(patch.Bytes, sanitizePatchName(patch.Name), p, projectSourcePath, patch.SuggestedPriority, patch.IsBGM)
}

// importCore is the shared decode-validate-copy-append flow. Name
// collisions get _2, _3, … suffixes so a designer can Bind the same
// patch multiple times without overwriting.
func importCore(
	bytes []byte,
	baseName string,
	p *pixelforge_project.Project,
	projectSourcePath string,
	priority string,
	isBGM bool,
) (ImportResult, error) {
	if p == nil {
		return ImportResult{}, errors.New("audio import: nil project")
	}
	if baseName == "" {
		baseName = "imported_sound"
	}

	sample, err := pixelforge_audio.DecodeWavOrErr(bytes)
	if err != nil {
		return ImportResult{}, fmt.Errorf("audio import: %w (requires 8-bit mono PCM WAV)", err)
	}

	if priority == "" {
		priority = "sfx"
	}

	// Pick a unique destination filename to avoid clobbering prior
	// imports. The collision check is in two layers: against
	// previously-imported AudioSamples and against the on-disk
	// directory (the second guards against orphan files left over
	// from earlier sessions).
	uniqueName, relPath := chooseUniqueAudioName(baseName, p, projectSourcePath)

	if projectSourcePath != "" {
		if err := writeAudioFile(projectSourcePath, relPath, bytes); err != nil {
			return ImportResult{}, err
		}
	}

	// SampleRateHz lives on the decoded sample but the existing
	// AudioSample schema carries it as int; the decoded value is
	// uint16. Truncation is safe (max declared rate is 48000).
	sampleRateHz := 0
	if sample != nil {
		sampleRateHz = int(sampleRateHzFromBytes(bytes))
	}

	asset := pixelforge_project.AudioSample{
		Name:                     uniqueName,
		RelativePath:             relPath,
		SuggestedChannelPriority: priority,
		Loop:                     isBGM,
		SampleRateHz:             sampleRateHz,
	}
	p.Audio = append(p.Audio, asset)

	durationMs := 0
	if sample != nil && sampleRateHz > 0 {
		durationMs = sample.Len() * 1000 / sampleRateHz
	}

	return ImportResult{
		SampleName:   uniqueName,
		RelativePath: relPath,
		DurationMs:   durationMs,
		IsBGM:        isBGM,
	}, nil
}

// sanitizePatchName converts a library patch name (which may carry
// slashes, e.g. "jump/spring") into a filesystem-safe basename
// ("jump_spring"). Sample's Name field also uses this so the
// inspector / picker shows a stable identifier.
func sanitizePatchName(name string) string {
	return strings.ReplaceAll(name, "/", "_")
}

// chooseUniqueAudioName returns (sampleName, relativePath) such
// that no existing AudioSample carries the same name and no file
// already exists at the destination path. Suffixes _2, _3, … on
// collision.
func chooseUniqueAudioName(baseName string, p *pixelforge_project.Project, projectSourcePath string) (string, string) {
	candidate := baseName
	relPath := filepath.ToSlash(filepath.Join("audio", candidate+".wav"))
	suffix := 2
	for nameInUse(candidate, p) || fileAtRelPathExists(projectSourcePath, relPath) {
		candidate = fmt.Sprintf("%s_%d", baseName, suffix)
		relPath = filepath.ToSlash(filepath.Join("audio", candidate+".wav"))
		suffix++
		if suffix > 9999 {
			break // defensive — never loop forever
		}
	}
	return candidate, relPath
}

func nameInUse(name string, p *pixelforge_project.Project) bool {
	for _, a := range p.Audio {
		if a.Name == name {
			return true
		}
	}
	return false
}

func fileAtRelPathExists(projectSourcePath, relPath string) bool {
	if projectSourcePath == "" {
		return false
	}
	dst := filepath.Join(pixelforge_project.AssetsDir(projectSourcePath), filepath.FromSlash(relPath))
	_, err := os.Stat(dst)
	return err == nil
}

// writeAudioFile creates the audio/ subdir if missing and writes
// the supplied bytes. Mirrors palette/import_pipeline.go's
// copyAssetFile shape; centralised here so the audio side stays
// independent of the palette package.
func writeAudioFile(projectSourcePath, relPath string, bytes []byte) error {
	dst := filepath.Join(pixelforge_project.AssetsDir(projectSourcePath), filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("audio import: mkdir %s: %w", filepath.Dir(dst), err)
	}
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("audio import: create %s: %w", dst, err)
	}
	defer out.Close()
	if _, err := out.Write(bytes); err != nil {
		return fmt.Errorf("audio import: write %s: %w", dst, err)
	}
	return nil
}

// stripExt removes the file extension from a path's basename.
func stripExt(s string) string {
	ext := filepath.Ext(s)
	if ext == "" {
		return s
	}
	return strings.TrimSuffix(s, ext)
}

// sampleRateHzFromBytes reads the WAV header's sample-rate field
// directly. DecodeWavOrErr validates the bytes pass the gate; this
// reads from offset 24 (the canonical fmt-chunk sample-rate field
// position in a 44-byte WAV header).
func sampleRateHzFromBytes(b []byte) uint32 {
	if len(b) < 28 {
		return 0
	}
	return uint32(b[24]) | uint32(b[25])<<8 | uint32(b[26])<<16 | uint32(b[27])<<24
}

// ensure io is referenced even when no test uses it explicitly —
// keeps the import list honest while leaving room for a future
// streaming variant.
var _ = io.Discard
