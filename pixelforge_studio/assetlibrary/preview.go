package assetlibrary

import (
	"bytes"
	"image"
	"image/png"
	"os"
)

// PreviewSpriteThumb loads the pack-relative sprite PNG and
// returns the decoded image.Image so the Library workspace can
// blit a thumbnail. Returns the standard image package's error
// when the file isn't a valid PNG; tests assert on dimensions.
//
// Audio audition is intentionally not implemented here —
// audiolib already owns the audition path and the workspace
// reaches through it. Keeping the preview helper sprite-only
// avoids dragging the audio subsystem into the library package.
func PreviewSpriteThumb(cacheRoot, packID, relPath string) (image.Image, error) {
	src := AssetPath(cacheRoot, packID, relPath)
	data, err := os.ReadFile(src)
	if err != nil {
		return nil, err
	}
	return png.Decode(bytes.NewReader(data))
}

// LicenseBadge returns the short user-facing label the
// workspace renders next to each asset. Empty / unknown licenses
// render as "?" so designers see something obviously wrong rather
// than silent blanks.
func LicenseBadge(license string) string {
	if license == "" {
		return "?"
	}
	return license
}
