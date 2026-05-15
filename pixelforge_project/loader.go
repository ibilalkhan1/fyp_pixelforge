package pixelforge_project

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrNotAPforgeFile signals that the file at the given path is JSON
// (or unreadable) but lacks a schema_version field — it's not a
// Pixelforge project. The error message names the path so a sloppy
// drop-target gets fixed quickly.
type ErrNotAPforgeFile struct {
	Path string
}

func (e *ErrNotAPforgeFile) Error() string {
	return fmt.Sprintf("%s: not a Pixelforge project (no schema_version field)", e.Path)
}

// ErrMissingAsset reports a sprite or audio asset referenced by the
// project that doesn't exist on disk. AbsPath is the resolved path the
// loader looked for so the user knows exactly where to put the file.
type ErrMissingAsset struct {
	ProjectPath string
	AbsPath     string
	Asset       string // "sprite:fruit" or "audio:eat"
	Referrer    string // entity ID or behavior name; "" for top-level
}

func (e *ErrMissingAsset) Error() string {
	if e.Referrer != "" {
		return fmt.Sprintf("%s: asset %s (referenced by %s) not found at %s",
			e.ProjectPath, e.Asset, e.Referrer, e.AbsPath)
	}
	return fmt.Sprintf("%s: asset %s not found at %s", e.ProjectPath, e.Asset, e.AbsPath)
}

// Load reads a .pforge file from path, validates the schema version,
// migrates if needed, and returns the in-memory Project.
//
// Assets are validated against the *-assets/ sibling directory: any
// referenced sprite or audio sample whose file is missing yields
// ErrMissingAsset. The Project's path is recorded so subsequent Save
// calls write back to the same location.
func Load(path string) (*Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Peek at schema_version before fully unmarshalling. We do a
	// lightweight decode into a sniffer struct so a future schema
	// surprise doesn't pollute the destination value.
	var sniff struct {
		SchemaVersion *int `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &sniff); err != nil {
		return nil, fmt.Errorf("%s: invalid JSON: %w", path, err)
	}
	if sniff.SchemaVersion == nil {
		return nil, &ErrNotAPforgeFile{Path: path}
	}
	if *sniff.SchemaVersion > SchemaVersion {
		return nil, &ErrUnsupportedSchemaVersion{Got: *sniff.SchemaVersion, Want: SchemaVersion}
	}

	p := &Project{}
	if err := json.Unmarshal(data, p); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := MigrateForward(p); err != nil {
		return nil, err
	}
	p.applyDefaults()
	p.normalizeSlices()

	if err := validateAssets(p, path); err != nil {
		return nil, err
	}
	return p, nil
}

// MustLoad is a Load that panics on error. Intended for test fixtures
// and tooling where a missing project is a fatal configuration bug.
func MustLoad(path string) *Project {
	p, err := Load(path)
	if err != nil {
		panic(err)
	}
	return p
}

// LoadReader is a streaming variant useful for go:embed fixtures and
// in-memory round-trip tests. The project's asset references are NOT
// validated — there's no on-disk anchor to resolve relative paths
// against.
func LoadReader(r io.Reader) (*Project, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	p := &Project{}
	if err := json.Unmarshal(data, p); err != nil {
		return nil, err
	}
	if err := MigrateForward(p); err != nil {
		return nil, err
	}
	p.applyDefaults()
	p.normalizeSlices()
	return p, nil
}

// AssetsDir returns the *-assets/ directory paired with a .pforge file.
// Per the schema docs: my-game.pforge sits next to my-game.pforge-assets/.
func AssetsDir(pforgePath string) string {
	base := strings.TrimSuffix(pforgePath, filepath.Ext(pforgePath))
	return base + ".pforge-assets"
}

// validateAssets checks each Sprite/Audio relative path exists inside
// the project's assets directory. Errors include the resolved absolute
// path so the user can see exactly where the file should live.
func validateAssets(p *Project, pforgePath string) error {
	dir := AssetsDir(pforgePath)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		// No assets directory and no asset references → still valid.
		if len(p.Sprites) == 0 && len(p.Audio) == 0 {
			return nil
		}
	}
	for _, s := range p.Sprites {
		if s.RelativePath == "" {
			continue
		}
		abs := filepath.Join(dir, filepath.FromSlash(s.RelativePath))
		if _, err := os.Stat(abs); errors.Is(err, os.ErrNotExist) {
			return &ErrMissingAsset{
				ProjectPath: pforgePath,
				AbsPath:     abs,
				Asset:       "sprite:" + s.Name,
			}
		}
	}
	for _, a := range p.Audio {
		if a.RelativePath == "" {
			continue
		}
		abs := filepath.Join(dir, filepath.FromSlash(a.RelativePath))
		if _, err := os.Stat(abs); errors.Is(err, os.ErrNotExist) {
			return &ErrMissingAsset{
				ProjectPath: pforgePath,
				AbsPath:     abs,
				Asset:       "audio:" + a.Name,
			}
		}
	}
	return nil
}

// nowFunc is the clock Save() consults when updating ModifiedAt. Tests
// override this to make determinism assertions cheap (no need to mock
// time.Now in every test).
var nowFunc = func() time.Time { return time.Now().UTC() }

// touch updates the ModifiedAt timestamp via nowFunc. Save calls this;
// tests stub the clock by replacing nowFunc.
func (p *Project) touch() {
	p.ModifiedAt = nowFunc()
}
