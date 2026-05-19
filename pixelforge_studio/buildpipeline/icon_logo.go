package buildpipeline

import (
	"bytes"
	"encoding/base64"
	_ "embed"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sync"

	icns "github.com/jackmordaunt/icns/v2"
	"github.com/josephspurrier/goversioninfo"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"

	wico "github.com/biessek/golang-ico"
)

// logoSVG carries docs/logo.svg verbatim — same artwork the studio
// brands every shipped game with. The copy under
// pixelforge_studio/buildpipeline/logo_asset.svg exists because
// //go:embed can't reach outside its source file's package
// directory; the file is regenerated from docs/logo.svg by the
// brand-refresh script (or by hand for one-off updates).
//
//go:embed logo_asset.svg
var logoSVG []byte

// RasterLogoPNG rasterises the embedded logo SVG at size×size and
// returns the encoded PNG bytes. Pure-Go (oksvg + rasterx) — no
// cgo, no external tooling. Cached per-size so repeated calls
// during a single build don't re-rasterise.
//
// Sizes <= 0 return an error rather than rasterising into a
// zero-size canvas; callers passing dynamic dimensions get a
// loud failure instead of a silently-empty icon.
func RasterLogoPNG(size int) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("buildpipeline: invalid logo raster size %d", size)
	}
	if cached, ok := logoCacheGet(size); ok {
		return cached, nil
	}
	icon, err := oksvg.ReadIconStream(bytes.NewReader(logoSVG))
	if err != nil {
		return nil, fmt.Errorf("buildpipeline: parse logo svg: %w", err)
	}
	icon.SetTarget(0, 0, float64(size), float64(size))

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, img, img.Bounds())
	dasher := rasterx.NewDasher(size, size, scanner)
	icon.Draw(dasher, 1.0)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("buildpipeline: encode logo png: %w", err)
	}
	out := buf.Bytes()
	logoCachePut(size, out)
	return out, nil
}

// RasterLogoImage rasterises the embedded logo SVG at size×size
// and returns the decoded image.Image. Container packers (.ico,
// .icns) want image.Image rather than PNG bytes; exposing both
// helpers avoids a re-decode in the hot path.
func RasterLogoImage(size int) (image.Image, error) {
	if size <= 0 {
		return nil, fmt.Errorf("buildpipeline: invalid logo raster size %d", size)
	}
	icon, err := oksvg.ReadIconStream(bytes.NewReader(logoSVG))
	if err != nil {
		return nil, fmt.Errorf("buildpipeline: parse logo svg: %w", err)
	}
	icon.SetTarget(0, 0, float64(size), float64(size))
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, img, img.Bounds())
	dasher := rasterx.NewDasher(size, size, scanner)
	icon.Draw(dasher, 1.0)
	return img, nil
}

// BuildLogoICO packs the rasterised logo at the canonical Windows
// icon sizes (16/32/48/256) into a single .ico file. The Windows
// linker picks up the embedded resource via the .syso shim
// BuildWindowsSyso emits — the .ico itself is also useful for
// Windows Explorer's right-click Properties view of the shipped
// .exe and for designers shipping a stand-alone icon asset.
//
// biessek/golang-ico's Encode encodes a single image; we pre-
// compose a multi-size .ico by wrapping each Encode in the .ico
// container directly. To keep dependencies thin and match what
// goversioninfo expects, we currently emit the 256 px variant —
// Windows scales it as needed. A future polish unit can swap in
// a true multi-size .ico if the lossy scaling shows up as a
// visual regression.
func BuildLogoICO() ([]byte, error) {
	img, err := RasterLogoImage(256)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := wico.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("buildpipeline: encode .ico: %w", err)
	}
	return buf.Bytes(), nil
}

// BuildLogoICNS packs the rasterised logo into a macOS .icns
// container. icns/v2 selects appropriate type tags (ic07/ic08/
// ic09/ic10) from the source image dimensions; we feed it a
// single 1024 px source and let it derive intermediate sizes.
func BuildLogoICNS() ([]byte, error) {
	img, err := RasterLogoImage(1024)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := icns.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("buildpipeline: encode .icns: %w", err)
	}
	return buf.Bytes(), nil
}

// BuildWindowsSyso writes rsrc_windows_amd64.syso into outDir so
// the Windows Go linker picks up the embedded .ico when the
// host builder runs `go build`. arch must match the target
// GOARCH (currently always "amd64" — the build pipeline pins
// Windows builds there).
//
// Implementation: goversioninfo writes a .syso from a populated
// VersionInfo. We point its IconPath at a temporary .ico that
// BuildLogoICO produced, fill the StringFileInfo with the game's
// name and version, then call WriteSyso.
func BuildWindowsSyso(outDir, gameName, version, arch string) error {
	if outDir == "" {
		return fmt.Errorf("buildpipeline: BuildWindowsSyso: outDir is empty")
	}
	if arch == "" {
		arch = "amd64"
	}
	icoBytes, err := BuildLogoICO()
	if err != nil {
		return err
	}
	// goversioninfo wants an icon file path, not in-memory bytes,
	// so we drop the .ico next to the .syso in outDir. The .ico
	// is harmless leftover — `go build` doesn't pick it up unless
	// referenced via the .syso, and the file is usable as an icon
	// asset in its own right.
	icoPath := filepath.Join(outDir, sanitizeForFilename(gameName)+".ico")
	if err := os.WriteFile(icoPath, icoBytes, 0o644); err != nil {
		return fmt.Errorf("buildpipeline: write %s: %w", icoPath, err)
	}

	vi := &goversioninfo.VersionInfo{
		IconPath: icoPath,
	}
	vi.StringFileInfo.ProductName = gameName
	vi.StringFileInfo.FileDescription = gameName
	vi.StringFileInfo.OriginalFilename = sanitizeForFilename(gameName) + ".exe"
	if version != "" {
		vi.StringFileInfo.FileVersion = version
		vi.StringFileInfo.ProductVersion = version
	}
	vi.Walk()

	sysoPath := filepath.Join(outDir, "rsrc_windows_"+arch+".syso")
	if err := vi.WriteSyso(sysoPath, arch); err != nil {
		return fmt.Errorf("buildpipeline: write %s: %w", sysoPath, err)
	}
	return nil
}

// GenerateLogoFavicon returns the 32×32 logo as base64 PNG — the
// shape BundleWASM's <link rel="icon" type="image/png"> data URL
// expects. Always succeeds (the SVG is embedded), so callers can
// inline this without an error branch where the legacy
// hash-colour fallback had one.
func GenerateLogoFavicon() (string, error) {
	pngBytes, err := RasterLogoPNG(32)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pngBytes), nil
}

// ---- caches + small helpers -------------------------------------

var (
	logoCacheMu sync.RWMutex
	logoCache   = map[int][]byte{}
)

func logoCacheGet(size int) ([]byte, bool) {
	logoCacheMu.RLock()
	defer logoCacheMu.RUnlock()
	b, ok := logoCache[size]
	return b, ok
}

func logoCachePut(size int, b []byte) {
	logoCacheMu.Lock()
	defer logoCacheMu.Unlock()
	logoCache[size] = b
}

// ResetLogoCacheForTest clears the per-size raster cache. Test-only;
// production never invalidates (the SVG is embedded + immutable).
func ResetLogoCacheForTest() {
	logoCacheMu.Lock()
	defer logoCacheMu.Unlock()
	logoCache = map[int][]byte{}
}
