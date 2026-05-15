package capture

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRecorderWithSomeFrames(t *testing.T, n int) *Recorder {
	t.Helper()
	pixelforge.SetScreenSize(8, 8)
	rec := New(120)
	for i := 0; i < n; i++ {
		pixelforge.SetColor(pixelforge.Color((i % 6) + 1))
		pixelforge.RectFill(0, 0, 7, 7)
		rec.RecordInput("key", "down:32")
		rec.SaveFrame()
	}
	return rec
}

func TestPackageReproZip_HappyPath(t *testing.T) {
	rec := setupRecorderWithSomeFrames(t, 100)
	pforge, project := makeProjectWithSprite(t)

	var buf bytes.Buffer
	require.NoError(t, PackageReproZip(rec, project, pforge, 30, &buf))

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	names := zipNames(zr)
	assert.Contains(t, names, "README.md")
	assert.Contains(t, names, "project.pforge")
	assert.Contains(t, names, "system.txt")
	assert.Contains(t, names, "capture/input.log")
	assert.Contains(t, names, "capture/events.log")
	assert.Contains(t, names, "capture/seed.txt")

	frameCount := 0
	for _, name := range names {
		if strings.HasPrefix(name, "capture/frames/") {
			frameCount++
		}
	}
	assert.Equal(t, 30, frameCount)
}

func TestPackageReproZip_ClampsFramesBack(t *testing.T) {
	rec := setupRecorderWithSomeFrames(t, 5)
	pforge, project := makeProjectWithSprite(t)
	var buf bytes.Buffer
	require.NoError(t, PackageReproZip(rec, project, pforge, 200, &buf))

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	frameCount := 0
	for _, name := range zipNames(zr) {
		if strings.HasPrefix(name, "capture/frames/") {
			frameCount++
		}
	}
	assert.Equal(t, 5, frameCount, "clamps to all available frames")
}

func TestPackageReproZip_DefaultFramesBack(t *testing.T) {
	rec := setupRecorderWithSomeFrames(t, 100)
	pforge, project := makeProjectWithSprite(t)
	var buf bytes.Buffer
	require.NoError(t, PackageReproZip(rec, project, pforge, 0, &buf))

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	frameCount := 0
	for _, name := range zipNames(zr) {
		if strings.HasPrefix(name, "capture/frames/") {
			frameCount++
		}
	}
	assert.Equal(t, DefaultBugReportFrames, frameCount)
}

func TestPackageReproZip_ReadmeMentionsProject(t *testing.T) {
	rec := setupRecorderWithSomeFrames(t, 10)
	pforge, project := makeProjectWithSprite(t)
	var buf bytes.Buffer
	require.NoError(t, PackageReproZip(rec, project, pforge, 5, &buf))

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	for _, zf := range zr.File {
		if zf.Name != "README.md" {
			continue
		}
		r, err := zf.Open()
		require.NoError(t, err)
		body, err := io.ReadAll(r)
		require.NoError(t, err)
		r.Close()
		assert.Contains(t, string(body), project.Name)
	}
}

func TestPackageReproZip_WithoutAssetsDirOmitsSection(t *testing.T) {
	rec := setupRecorderWithSomeFrames(t, 5)
	// Use a project not on disk (projectPath empty) — no assets to bundle.
	tmp := t.TempDir()
	_ = tmp
	pforge := ""
	_, project := makeProjectWithSprite(t)
	var buf bytes.Buffer
	require.NoError(t, PackageReproZip(rec, project, pforge, 3, &buf))

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	for _, name := range zipNames(zr) {
		assert.NotEqual(t, "project.pforge-assets/", name)
	}
}

func zipNames(zr *zip.Reader) []string {
	out := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		out = append(out, f.Name)
	}
	return out
}
