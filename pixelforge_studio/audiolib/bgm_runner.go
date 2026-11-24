// MP3 and Vorbis imports validate real decoded audio before registering it.
package audiolib

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
)

// BGMContainer names the recognised long-form audio containers.
type BGMContainer string

const (
	// BGMContainerOGG is Vorbis-in-OGG. Magic "OggS" at offset 0.
	BGMContainerOGG BGMContainer = "ogg"

	// BGMContainerMP3 is MPEG-1/2 Layer III. Detected by either an
	// "ID3" tag at offset 0 or an MPEG sync word (0xFF Ex) anywhere
	// in the first few hundred bytes; we check both.
	BGMContainerMP3 BGMContainer = "mp3"
)

// BGMImportResult reports the decoded duration and persisted source.
type BGMImportResult struct {
	SampleName   string
	RelativePath string
	DurationMs   int
	Container    BGMContainer
}

// ImportBGMFromFile is the on-disk BGM-import entry point. Mirrors
// ImportFromFile (WAV) so the audiolib's per-kind import surface
// stays uniform: decode-validate → copy-to-assets-dir → append
// AudioSample.
func ImportBGMFromFile(path string, p *pixelforge_project.Project, projectSourcePath string) (BGMImportResult, error) {
	if path == "" {
		return BGMImportResult{}, errors.New("bgm import: empty path")
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return BGMImportResult{}, fmt.Errorf("bgm import: read %s: %w", path, err)
	}
	container, err := detectBGMContainer(path, bytes)
	if err != nil {
		return BGMImportResult{}, err
	}
	baseName := stripExt(filepath.Base(path))
	return importBGMCore(bytes, baseName, container, p, projectSourcePath)
}

// importBGMCore is the shared validate-copy-append flow. Name
// collisions get _2, _3, … suffixes mirroring importCore (WAV).
func importBGMCore(
	bytes []byte,
	baseName string,
	container BGMContainer,
	p *pixelforge_project.Project,
	projectSourcePath string,
) (BGMImportResult, error) {
	if p == nil {
		return BGMImportResult{}, errors.New("bgm import: nil project")
	}
	if baseName == "" {
		baseName = "imported_bgm"
	}

	sample, err := pixelforge_audio.DecodeAudio(bytes, string(container))
	if err != nil {
		return BGMImportResult{}, fmt.Errorf("bgm import: %w", err)
	}

	uniqueName, relPath := chooseUniqueBGMName(baseName, string(container), p, projectSourcePath)

	if projectSourcePath != "" {
		if err := writeAudioFile(projectSourcePath, relPath, bytes); err != nil {
			return BGMImportResult{}, err
		}
	}

	asset := pixelforge_project.AudioSample{
		Name:                     uniqueName,
		RelativePath:             relPath,
		SuggestedChannelPriority: "bgm",
		Loop:                     true,
		// Rate reflects the actual decoded mixer sample.
		SampleRateHz: int(sample.SampleRate()),
	}
	p.Audio = append(p.Audio, asset)

	return BGMImportResult{
		SampleName:   uniqueName,
		RelativePath: relPath,
		DurationMs:   sample.Len() * 1000 / int(sample.SampleRate()),
		Container:    container,
	}, nil
}

// detectBGMContainer pairs the path extension with the file's
// magic bytes so a renamed .txt → .ogg drop errors cleanly.
// Returns the canonical container token used by relative-path
// construction.
func detectBGMContainer(path string, bytes []byte) (BGMContainer, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".ogg":
		if !looksLikeOGG(bytes) {
			return "", fmt.Errorf("bgm import: %s has .ogg extension but missing OggS magic", path)
		}
		return BGMContainerOGG, nil
	case ".mp3":
		if !looksLikeMP3(bytes) {
			return "", fmt.Errorf("bgm import: %s has .mp3 extension but missing ID3/MPEG sync", path)
		}
		return BGMContainerMP3, nil
	}
	return "", fmt.Errorf("bgm import: %s: unrecognised extension %q (want .ogg or .mp3)", path, ext)
}

// looksLikeOGG reports whether bytes start with the OggS magic.
// OGG bitstream pages always carry "OggS" at offset 0 (RFC 3533).
func looksLikeOGG(b []byte) bool {
	return len(b) >= 4 && string(b[:4]) == "OggS"
}

// looksLikeMP3 reports whether bytes carry either an ID3 tag at
// offset 0 ("ID3") or an MPEG-1/2 sync word (0xFF + 0xE0..0xFF in
// the second byte) somewhere in the first 1024 bytes. The latter
// scan handles MP3s lacking an ID3 header.
func looksLikeMP3(b []byte) bool {
	if len(b) >= 3 && string(b[:3]) == "ID3" {
		return true
	}
	limit := len(b)
	if limit > 1024 {
		limit = 1024
	}
	for i := 0; i+1 < limit; i++ {
		if b[i] == 0xFF && (b[i+1]&0xE0) == 0xE0 {
			return true
		}
	}
	return false
}

// chooseUniqueBGMName mirrors chooseUniqueAudioName but keys the
// extension off the container. Suffixes _2, _3, … on collision.
func chooseUniqueBGMName(baseName, ext string, p *pixelforge_project.Project, projectSourcePath string) (string, string) {
	candidate := baseName
	relPath := filepath.ToSlash(filepath.Join("audio", candidate+"."+ext))
	suffix := 2
	for nameInUse(candidate, p) || fileAtRelPathExists(projectSourcePath, relPath) {
		candidate = fmt.Sprintf("%s_%d", baseName, suffix)
		relPath = filepath.ToSlash(filepath.Join("audio", candidate+"."+ext))
		suffix++
		if suffix > 9999 {
			break
		}
	}
	return candidate, relPath
}

// bgmImportRunner bridges editor.BGMImportHandler to
// ImportBGMFromFile. Same shape as wavImportRunner.
type bgmImportRunner struct {
	editor *editor.Editor
}

func newBGMImportRunner(e *editor.Editor) *bgmImportRunner {
	return &bgmImportRunner{editor: e}
}

// ImportPath runs the on-disk BGM pipeline. The editor's project +
// project-source-path are read at call time so a designer's
// save-as mid-session reflects in the destination path.
func (r *bgmImportRunner) ImportPath(bgmPath string) (*editor.BGMImportResult, error) {
	res, err := ImportBGMFromFile(bgmPath, r.editor.Project(), r.editor.CurrentProjectPath())
	if err != nil {
		return nil, err
	}
	return adaptBGMImportResult(&res), nil
}

func adaptBGMImportResult(res *BGMImportResult) *editor.BGMImportResult {
	return &editor.BGMImportResult{
		SampleName:   res.SampleName,
		RelativePath: res.RelativePath,
		DurationMs:   res.DurationMs,
		Container:    string(res.Container),
	}
}
