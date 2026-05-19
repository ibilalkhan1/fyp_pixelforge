package assetlibrary_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/assetlibrary"
)

// buildTarGz produces an in-memory .tar.gz containing the
// supplied files (path → bytes). Used to back the test HTTP
// server's pack responses.
func buildTarGz(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for path, data := range files {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name:     path,
			Mode:     0o644,
			Size:     int64(len(data)),
			Typeflag: tar.TypeReg,
		}))
		_, err := tw.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// fixtureServer wires a httptest.NewServer that serves a
// manifest at /manifest.json plus each pack at /<id>.tar.gz.
// Tests pass the returned URL prefix as DownloadOptions.ManifestURL.
type fixtureServer struct {
	srv   *httptest.Server
	hits  map[string]*atomic.Int64
	mutex sync.Mutex
}

func newFixtureServer(t *testing.T, packs map[string][]byte, manifestJSON string) *fixtureServer {
	t.Helper()
	fs := &fixtureServer{hits: map[string]*atomic.Int64{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		fs.count("manifest")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, manifestJSON)
	})
	for name, body := range packs {
		name, body := name, body
		mux.HandleFunc("/"+name, func(w http.ResponseWriter, r *http.Request) {
			fs.count(name)
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(body)
		})
	}
	mux.HandleFunc("/missing.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	fs.srv = httptest.NewServer(mux)
	t.Cleanup(fs.srv.Close)
	return fs
}

func (fs *fixtureServer) count(name string) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	c, ok := fs.hits[name]
	if !ok {
		c = &atomic.Int64{}
		fs.hits[name] = c
	}
	c.Add(1)
}

func (fs *fixtureServer) hitCount(name string) int64 {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	c, ok := fs.hits[name]
	if !ok {
		return 0
	}
	return c.Load()
}

func TestEnsureBootstrap_HappyPath(t *testing.T) {
	tarBytes := buildTarGz(t, map[string][]byte{
		"sprites/ship.png": []byte("fake-png"),
		"audio/blast.wav":  []byte("fake-wav"),
	})
	hash := sha256Hex(tarBytes)
	manifest := fmt.Sprintf(`{"schema_version":"1","packs":[
		{"id":"asteroids","version":"1.0.0","title":"Asteroids","game":"asteroids",
		 "url":"%%s/asteroids.tar.gz","sha256":%q,
		 "assets":[
		   {"path":"sprites/ship.png","kind":"sprite","license":"CC0"},
		   {"path":"audio/blast.wav","kind":"sfx","license":"CC-BY-4.0","author":"X"}
		 ]}
	]}`, hash)
	srv := newFixtureServer(t, map[string][]byte{"asteroids.tar.gz": tarBytes}, "PLACEHOLDER")
	// Patch manifest now that we know srv URL.
	mux := http.NewServeMux()
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(manifest, srv.srv.URL)))
	})
	mux.HandleFunc("/asteroids.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		w.Write(tarBytes)
	})
	srv2 := httptest.NewServer(mux)
	t.Cleanup(srv2.Close)

	cacheRoot := t.TempDir()
	lib := assetlibrary.NewLibrary(cacheRoot)
	err := assetlibrary.EnsureBootstrap(context.Background(), lib, assetlibrary.DownloadOptions{
		ManifestURL: srv2.URL + "/manifest.json",
	})
	require.NoError(t, err)

	require.True(t, lib.IsInstalled("asteroids"))
	packDir := assetlibrary.PackDir(cacheRoot, "asteroids")
	_, err = os.Stat(filepath.Join(packDir, "sprites", "ship.png"))
	assert.NoError(t, err, "ship.png must exist under unpacked pack dir")
	_, err = os.Stat(filepath.Join(packDir, "audio", "blast.wav"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(packDir, ".pack-version"))
	assert.NoError(t, err, ".pack-version sentinel must be stamped after install")
}

func TestEnsureBootstrap_ChecksumMismatchSurfacesError(t *testing.T) {
	tarBytes := buildTarGz(t, map[string][]byte{"x.png": []byte("hi")})
	wrongHash := sha256Hex([]byte("wrong"))
	manifest := fmt.Sprintf(`{"schema_version":"1","packs":[
		{"id":"bad","version":"1","title":"Bad","url":"%%s/bad.tar.gz","sha256":%q,"assets":[]}
	]}`, wrongHash)

	mux := http.NewServeMux()
	var srvURL string
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(manifest, srvURL)))
	})
	mux.HandleFunc("/bad.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		w.Write(tarBytes)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	srvURL = srv.URL

	cacheRoot := t.TempDir()
	lib := assetlibrary.NewLibrary(cacheRoot)
	var events []assetlibrary.Event
	err := assetlibrary.EnsureBootstrap(context.Background(), lib, assetlibrary.DownloadOptions{
		ManifestURL: srv.URL + "/manifest.json",
		Progress:    func(ev assetlibrary.Event) { events = append(events, ev) },
	})
	require.NoError(t, err, "bootstrap returns nil even when individual packs fail")

	assert.False(t, lib.IsInstalled("bad"), "checksum-mismatched pack must not install")
	// No pack dir on disk.
	_, err = os.Stat(assetlibrary.PackDir(cacheRoot, "bad"))
	assert.True(t, os.IsNotExist(err), "no partial pack dir for failed install")

	// Event with ErrChecksumMismatch surfaces.
	var sawMismatch bool
	for _, ev := range events {
		if ev.Err != nil && errors.Is(ev.Err, assetlibrary.ErrChecksumMismatch) {
			sawMismatch = true
			break
		}
	}
	assert.True(t, sawMismatch, "progress callback must report ErrChecksumMismatch")
}

func TestEnsureBootstrap_HTTP404SurfacesPackUnavailable(t *testing.T) {
	manifest := `{"schema_version":"1","packs":[
		{"id":"missing","version":"1","title":"Missing","url":"%s/missing.tar.gz","sha256":"abc","assets":[]}
	]}`
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(manifest, srv.URL)))
	})
	mux.HandleFunc("/missing.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	cacheRoot := t.TempDir()
	lib := assetlibrary.NewLibrary(cacheRoot)
	var events []assetlibrary.Event
	err := assetlibrary.EnsureBootstrap(context.Background(), lib, assetlibrary.DownloadOptions{
		ManifestURL: srv.URL + "/manifest.json",
		Progress:    func(ev assetlibrary.Event) { events = append(events, ev) },
	})
	require.NoError(t, err)
	assert.False(t, lib.IsInstalled("missing"))

	var sawUnavailable bool
	for _, ev := range events {
		if ev.Err != nil && errors.Is(ev.Err, assetlibrary.ErrPackUnavailable) {
			sawUnavailable = true
			break
		}
	}
	assert.True(t, sawUnavailable, "404 must surface as ErrPackUnavailable in progress events")
}

func TestEnsureBootstrap_SecondRunSameVersionSkipsDownload(t *testing.T) {
	tarBytes := buildTarGz(t, map[string][]byte{"x.png": []byte("hi")})
	hash := sha256Hex(tarBytes)
	manifest := fmt.Sprintf(`{"schema_version":"1","packs":[
		{"id":"p","version":"1.0.0","title":"P","url":"%%s/p.tar.gz","sha256":%q,"assets":[]}
	]}`, hash)
	mux := http.NewServeMux()
	var srvURL string
	var packHits atomic.Int64
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(manifest, srvURL)))
	})
	mux.HandleFunc("/p.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		packHits.Add(1)
		w.Write(tarBytes)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	srvURL = srv.URL

	cacheRoot := t.TempDir()
	lib := assetlibrary.NewLibrary(cacheRoot)
	url := srv.URL + "/manifest.json"
	require.NoError(t, assetlibrary.EnsureBootstrap(context.Background(), lib, assetlibrary.DownloadOptions{ManifestURL: url}))
	require.Equal(t, int64(1), packHits.Load())

	// Second run hits the manifest but not the pack body.
	lib2 := assetlibrary.NewLibrary(cacheRoot)
	require.NoError(t, assetlibrary.EnsureBootstrap(context.Background(), lib2, assetlibrary.DownloadOptions{ManifestURL: url}))
	assert.Equal(t, int64(1), packHits.Load(), "same-version reruns must not re-download")
	assert.True(t, lib2.IsInstalled("p"), "lib2 sees the pack as installed via the on-disk sentinel")
}

func TestEnsureBootstrap_NilLibraryErrors(t *testing.T) {
	err := assetlibrary.EnsureBootstrap(context.Background(), nil, assetlibrary.DownloadOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil library")
}

func TestEnsureBootstrap_EnvVarOverridesURL(t *testing.T) {
	// Manifest is invalid JSON so we know the env var URL was hit
	// (vs. silently falling through to the default URL which would
	// time out the test). The env var path raises a parse error
	// which surfaces as a returned error from EnsureBootstrap.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not-json"))
	}))
	t.Cleanup(srv.Close)

	t.Setenv("PIXELFORGE_ASSET_LIBRARY_URL", srv.URL+"/manifest.json")
	cacheRoot := t.TempDir()
	lib := assetlibrary.NewLibrary(cacheRoot)
	err := assetlibrary.EnsureBootstrap(context.Background(), lib, assetlibrary.DownloadOptions{})
	require.Error(t, err, "env-var URL must be hit (causing parse failure on bogus content)")
}

func TestEnsureBootstrap_UnsafeTarEntryRejected(t *testing.T) {
	// Build a tar with an entry attempting path traversal.
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "../escape", Mode: 0o644, Size: 3, Typeflag: tar.TypeReg,
	}))
	tw.Write([]byte("hi\n"))
	tw.Close()
	gz.Close()

	hash := sha256Hex(buf.Bytes())
	manifest := fmt.Sprintf(`{"schema_version":"1","packs":[
		{"id":"evil","version":"1","title":"E","url":"%%s/evil.tar.gz","sha256":%q,"assets":[]}
	]}`, hash)
	mux := http.NewServeMux()
	var srvURL string
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(manifest, srvURL)))
	})
	mux.HandleFunc("/evil.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		w.Write(buf.Bytes())
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	srvURL = srv.URL

	cacheRoot := t.TempDir()
	lib := assetlibrary.NewLibrary(cacheRoot)
	var events []assetlibrary.Event
	require.NoError(t, assetlibrary.EnsureBootstrap(context.Background(), lib, assetlibrary.DownloadOptions{
		ManifestURL: srv.URL + "/manifest.json",
		Progress:    func(ev assetlibrary.Event) { events = append(events, ev) },
	}))
	assert.False(t, lib.IsInstalled("evil"), "traversal-attempt pack must not install")

	var rejected bool
	for _, ev := range events {
		if ev.Err != nil {
			rejected = true
			break
		}
	}
	assert.True(t, rejected, "evil pack must surface an error event")
}
