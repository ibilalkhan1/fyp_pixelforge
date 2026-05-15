package palette

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Missing sidecar returns an empty Sidecar and no error.
func TestLoadSidecar_Missing(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "img.png")
	sc, err := LoadSidecar(pngPath)
	assert.NoError(t, err)
	assert.Equal(t, Sidecar{}, sc)
}

// Valid sidecar parses correctly.
func TestLoadSidecar_Parses(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "img.png")
	require.NoError(t, os.WriteFile(pngPath+".meta", []byte(`{"frame_w":16,"frame_h":24}`), 0o644))
	sc, err := LoadSidecar(pngPath)
	require.NoError(t, err)
	assert.Equal(t, 16, sc.FrameW)
	assert.Equal(t, 24, sc.FrameH)
}

// Malformed sidecar surfaces an error so the importer can warn.
func TestLoadSidecar_MalformedReturnsError(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "img.png")
	require.NoError(t, os.WriteFile(pngPath+".meta", []byte(`not json`), 0o644))
	_, err := LoadSidecar(pngPath)
	assert.Error(t, err)
}
