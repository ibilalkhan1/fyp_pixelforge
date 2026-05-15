package pixelforge_project

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Save writes p to path as indented JSON. Side effects:
//   - SchemaVersion is bumped to the current value (in case the in-memory
//     project was loaded from an older schema and migrated).
//   - ModifiedAt is set to time.Now().UTC().
//   - The *-assets/ directory is created if missing so the editor can
//     start dropping sprites/audio in immediately.
//   - Nil slices are normalised to empty slices so the JSON output never
//     contains "null" values.
//
// Save is deterministic: two saves of the same in-memory project produce
// byte-identical files. This is the property the M5 git-merge UX
// depends on.
func (p *Project) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(AssetsDir(path), 0o755); err != nil {
		return err
	}

	p.SchemaVersion = SchemaVersion
	p.touch()
	p.normalizeSlices()

	if p.ModifiedAt.After(time.Now().UTC().Add(24 * time.Hour)) {
		log.Printf("pixelforge_project: %s: ModifiedAt is in the far future (%s)", path, p.ModifiedAt)
	}

	data, err := encodeProject(p)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Encode returns the canonical JSON bytes for p without touching disk.
// Side-effect-free apart from p.SchemaVersion bump + ModifiedAt touch +
// nil-slice normalization, which match what Save would produce.
//
// Used by the code-gen embed pipeline so the bytes shipped in an
// exported binary are byte-identical to the .pforge file on disk.
func Encode(p *Project) ([]byte, error) {
	p.SchemaVersion = SchemaVersion
	p.touch()
	p.normalizeSlices()
	return encodeProject(p)
}

// encodeProject returns the canonical JSON for p. Exposed for tests
// that need byte-for-byte equality without touching disk.
func encodeProject(p *Project) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	// SetEscapeHTML(false) keeps "&", "<", ">" un-escaped, which makes
	// the file friendlier to grep and to humans browsing the diff.
	enc.SetEscapeHTML(false)
	if err := enc.Encode(p); err != nil {
		return nil, err
	}
	// json.Encoder appends a trailing newline. Strip it so the encoded
	// output matches MarshalIndent's behaviour and round-trip tests
	// don't see a phantom newline mismatch.
	out := buf.Bytes()
	if len(out) > 0 && out[len(out)-1] == '\n' {
		out = out[:len(out)-1]
	}
	return out, nil
}
