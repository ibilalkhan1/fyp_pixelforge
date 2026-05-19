package playerbins

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
)

// binsFS holds the pre-built per-target player binaries. The
// release pipeline populates bins/<goos>-<goarch>/pixelforge-player[.exe|.wasm]
// via `make playerbins` before tagging; development checkouts may
// see an empty tree (only the .gitkeep placeholder), in which case
// PlayerBinaryFor returns ErrNotEmbedded and the buildpipeline
// falls through to its developer fallback.
//
//go:embed all:bins
var binsFS embed.FS

// ErrNotEmbedded is returned by PlayerBinaryFor when the requested
// (GOOS, GOARCH) pair has no entry under binsFS. Callers (the
// buildpipeline) treat this as a soft miss and fall through to the
// next step of the player-binary discovery chain (developer
// `go build` fallback) rather than failing the build.
var ErrNotEmbedded = errors.New("playerbins: no embedded player binary for this OS/arch")

// PlayerBinaryFor returns the embedded pixelforge-player bytes for
// the supplied GOOS/GOARCH pair. The lookup convention mirrors the
// Makefile's playerbins target:
//
//	bins/<goos>-<goarch>/pixelforge-player        (Linux, macOS)
//	bins/windows-amd64/pixelforge-player.exe      (Windows)
//	bins/js-wasm/pixelforge-player.wasm           (WASM)
//
// Returns ErrNotEmbedded when the directory or file is missing so
// the buildpipeline can fall through to the developer fallback
// without conflating "no embedded binary shipped" with "tried to
// read the binary and the read failed."
func PlayerBinaryFor(goos, goarch string) ([]byte, error) {
	if goos == "" || goarch == "" {
		return nil, fmt.Errorf("playerbins: goos/goarch must be non-empty (got %q/%q)", goos, goarch)
	}
	dir := goos + "-" + goarch
	filename := playerBinaryFilename(goos)
	entry := path.Join("bins", dir, filename)
	data, err := binsFS.ReadFile(entry)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrNotEmbedded, entry)
		}
		return nil, fmt.Errorf("playerbins: read embedded entry %s: %w", entry, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: %s is empty", ErrNotEmbedded, entry)
	}
	return data, nil
}

// playerBinaryFilename returns the on-disk filename the release
// process produces for the supplied GOOS. Windows carries the .exe
// suffix the Go linker auto-appends; WASM gets .wasm; everything
// else is the bare binary.
func playerBinaryFilename(goos string) string {
	switch goos {
	case "windows":
		return "pixelforge-player.exe"
	case "js":
		return "pixelforge-player.wasm"
	}
	return "pixelforge-player"
}
