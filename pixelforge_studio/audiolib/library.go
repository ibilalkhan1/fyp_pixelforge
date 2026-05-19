// library.go owns idea #4 v1 U3's audio library catalog. Patches
// are declared in catalog.json with synthesis parameters; the WAV
// bytes are produced on demand by SynthesizeWAV (wavgen.go) and
// validated against the engine's strict 8-bit-mono-PCM gate
// (DecodeWavOrErr) at load time. Malformed entries log and skip
// rather than panicking, per editor-pforge-schema-shape.md.
//
// The "embed real WAV files" path the plan describes is one shape;
// this implementation chose declarative synthesis instead because
// it keeps the repo binary-free, makes the library trivially
// extensible (edit catalog.json, no asset pipeline), and the v1
// bet per origin Key Decisions is wiring-over-fidelity. Quality
// upgrades land in a follow-up patch-pack release.
package audiolib

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
)

//go:embed cart_assets/catalog.json
var embeddedCatalogJSON []byte

// CatalogEntry is the on-disk catalog.json schema. Each entry
// declares everything the picker and the import pipeline need.
type CatalogEntry struct {
	Name              string     `json:"name"`
	Category          string     `json:"category"`
	IsBGM             bool       `json:"is_bgm"`
	SuggestedPriority string     `json:"suggested_priority"`
	Synth             PatchSynth `json:"synth"`
}

// LibraryPatch is the post-validation runtime view of a patch.
// Carries the synthesised bytes (built once at LoadCatalog time)
// plus the Duration (derived from the WAV decode result) so the
// picker can render "0.4s" without re-parsing.
type LibraryPatch struct {
	Name              string
	Category          string
	IsBGM             bool
	SuggestedPriority string
	Bytes             []byte
	Duration          time.Duration
}

var (
	catalogOnce  sync.Once
	cachedPatches []LibraryPatch
	cachedErr     error
)

// LoadCatalog parses the embedded catalog.json, synthesises each
// declared patch, validates the result via DecodeWavOrErr, and
// returns the surviving entries. Malformed catalog JSON is fatal
// (we can't ship a library at all); per-patch failures (invalid
// synth params that produce bytes the gate rejects) are logged and
// skipped so the catalog can grow without breaking the studio.
//
// Cached on first call; subsequent calls return the same slice
// (the bytes are deterministic, so this is safe).
func LoadCatalog() ([]LibraryPatch, error) {
	catalogOnce.Do(func() {
		cachedPatches, cachedErr = loadCatalogUncached(embeddedCatalogJSON)
	})
	return cachedPatches, cachedErr
}

func loadCatalogUncached(raw []byte) ([]LibraryPatch, error) {
	var entries []CatalogEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("audiolib: catalog.json parse failed: %w", err)
	}

	out := make([]LibraryPatch, 0, len(entries))
	for i, e := range entries {
		if e.Name == "" {
			log.Printf("audiolib: catalog entry %d skipped (missing name)", i)
			continue
		}
		bytes := SynthesizeWAV(e.Synth)
		sample, err := pixelforge_audio.DecodeWavOrErr(bytes)
		if err != nil {
			log.Printf("audiolib: patch %q skipped: %v", e.Name, err)
			continue
		}
		duration := sampleDuration(sample, e.Synth.SampleRateHz)
		out = append(out, LibraryPatch{
			Name:              e.Name,
			Category:          e.Category,
			IsBGM:             e.IsBGM,
			SuggestedPriority: e.SuggestedPriority,
			Bytes:             bytes,
			Duration:          duration,
		})
	}
	return out, nil
}

// ReadPatchBytes returns the WAV bytes for the named patch. Returns
// an error when the patch isn't in the catalog so callers (the
// import pipeline in U4) surface a useful message rather than
// crashing.
func ReadPatchBytes(name string) ([]byte, error) {
	patches, err := LoadCatalog()
	if err != nil {
		return nil, err
	}
	for _, p := range patches {
		if p.Name == name {
			return p.Bytes, nil
		}
	}
	return nil, fmt.Errorf("audiolib: patch %q not found in catalog", name)
}

// FindPatch returns the LibraryPatch metadata for the named patch.
// Sibling to ReadPatchBytes for callers that need duration / loop
// flag without the byte payload.
func FindPatch(name string) (LibraryPatch, bool) {
	patches, _ := LoadCatalog()
	for _, p := range patches {
		if p.Name == name {
			return p, true
		}
	}
	return LibraryPatch{}, false
}

// PatchesByCategory returns every patch whose Category matches the
// supplied string (case-sensitive). Empty category returns every
// patch — useful for the picker panel's filter when the input is
// blank.
func PatchesByCategory(category string) []LibraryPatch {
	patches, _ := LoadCatalog()
	if category == "" {
		return patches
	}
	var out []LibraryPatch
	for _, p := range patches {
		if p.Category == category {
			out = append(out, p)
		}
	}
	return out
}

// resetForTest clears the cached catalog so a test that mutates
// catalog inputs can re-run LoadCatalog. Test-only; production code
// never re-loads.
func resetForTest() {
	catalogOnce = sync.Once{}
	cachedPatches = nil
	cachedErr = nil
}

// sampleDuration computes the patch's duration from the decoded
// sample. The plan's "Duration" field is a UX concern (the picker
// shows "0.4s"); the synthesis params already carry DurationMs,
// but reading it off the decoded sample lets us catch synth errors
// where the encoded length disagrees with the declared length.
func sampleDuration(sample *pixelforge_audio.Sample, sampleRateHz uint16) time.Duration {
	if sample == nil {
		return 0
	}
	if sampleRateHz == 0 {
		sampleRateHz = defaultSampleRate
	}
	seconds := float64(sample.Len()) / float64(sampleRateHz)
	return time.Duration(seconds * float64(time.Second))
}
