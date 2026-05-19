package assetlibrary_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/assetlibrary"
)

// validPforgeBytes returns the minimal byte sequence Load/LoadReader
// accept as a .pforge — schema_version is the sniffer's required
// field. These bytes are what the fetcher's tests stamp into the
// HTTP fixture so a follow-on Open-Project call could succeed.
func validPforgeBytes() []byte {
	return []byte(`{"schema_version":1,"name":"example_proof"}`)
}

// loadExampleManifest constructs a Library with a single example
// pointed at the supplied URL + SHA-256, and returns it. Callers
// inject any URL/hash combination the test needs (matching hash for
// happy path, wrong hash for the bad-checksum case).
func loadExampleManifest(t *testing.T, lib *assetlibrary.Library, id, url, sha256Hex string) {
	t.Helper()
	body := fmt.Sprintf(`{
		"schema_version":"1",
		"packs":[],
		"examples":[
			{"id":%q,"version":"1.0.0","title":%q,"url":%q,"sha256":%q,"size_bytes":42}
		]
	}`, id, id, url, sha256Hex)
	m, err := assetlibrary.ParseManifest([]byte(body))
	require.NoError(t, err)
	lib.SetManifest(m)
}

func TestFetchExample_UnknownIDReturnsErr(t *testing.T) {
	lib := assetlibrary.NewLibrary(t.TempDir())
	_, err := assetlibrary.FetchExample(context.Background(), lib, "nope", assetlibrary.ExampleFetchOptions{})
	require.Error(t, err)
	assert.ErrorIs(t, err, assetlibrary.ErrExampleUnknown)
}

func TestFetchExample_LoadingState_ManifestNotYetSet(t *testing.T) {
	// Pre-bootstrap library (Manifest()==nil) — lookup misses
	// and FetchExample surfaces ErrExampleUnknown. The menu's
	// "Loading examples…" disabled item depends on Library.Examples()
	// returning an empty slice in this state; verify both surfaces
	// agree the example list is empty.
	lib := assetlibrary.NewLibrary(t.TempDir())
	assert.Empty(t, lib.Examples(), "pre-bootstrap library reports no examples")
	assert.Nil(t, lib.LookupExample("asteroids_proof"))

	_, err := assetlibrary.FetchExample(context.Background(), lib, "asteroids_proof", assetlibrary.ExampleFetchOptions{})
	require.Error(t, err)
	assert.ErrorIs(t, err, assetlibrary.ErrExampleUnknown)
}

func TestFetchExample_NilLibraryErrors(t *testing.T) {
	_, err := assetlibrary.FetchExample(context.Background(), nil, "x", assetlibrary.ExampleFetchOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil library")
}

func TestFetchExample_CacheHitReturnsImmediately(t *testing.T) {
	cacheRoot := t.TempDir()
	lib := assetlibrary.NewLibrary(cacheRoot)

	body := validPforgeBytes()
	hash := sha256Hex(body)

	// Pre-populate the cache before installing the manifest so
	// no HTTP round-trip is needed at all. A server that refuses
	// every request proves the cache-hit short-circuit triggered.
	cachePath := assetlibrary.ExampleCachePath(cacheRoot, "cached_proof")
	require.NoError(t, os.MkdirAll(filepath.Dir(cachePath), 0o755))
	require.NoError(t, os.WriteFile(cachePath, body, 0o644))

	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "should not hit network on cache hit", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	loadExampleManifest(t, lib, "cached_proof", srv.URL+"/example.pforge", hash)

	var events []assetlibrary.ExampleEvent
	got, err := assetlibrary.FetchExample(context.Background(), lib, "cached_proof", assetlibrary.ExampleFetchOptions{
		Progress: func(ev assetlibrary.ExampleEvent) { events = append(events, ev) },
	})
	require.NoError(t, err)
	assert.Equal(t, cachePath, got)
	assert.Equal(t, int64(0), hits.Load(), "cache hit must not touch the network")
	require.Len(t, events, 1)
	assert.Equal(t, assetlibrary.ExampleEventCacheHit, events[0].Kind)
}

func TestFetchExample_CacheMissDownloadsAndOpens(t *testing.T) {
	cacheRoot := t.TempDir()
	lib := assetlibrary.NewLibrary(cacheRoot)

	body := validPforgeBytes()
	hash := sha256Hex(body)

	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	loadExampleManifest(t, lib, "fresh_proof", srv.URL+"/fresh.pforge", hash)

	var events []assetlibrary.ExampleEvent
	got, err := assetlibrary.FetchExample(context.Background(), lib, "fresh_proof", assetlibrary.ExampleFetchOptions{
		HTTPClient: srv.Client(),
		Progress:   func(ev assetlibrary.ExampleEvent) { events = append(events, ev) },
	})
	require.NoError(t, err)

	cachePath := assetlibrary.ExampleCachePath(cacheRoot, "fresh_proof")
	assert.Equal(t, cachePath, got)
	assert.Equal(t, int64(1), hits.Load(), "cache miss must hit the network exactly once")

	onDisk, err := os.ReadFile(cachePath)
	require.NoError(t, err)
	assert.Equal(t, body, onDisk, "cached file holds the downloaded bytes")

	// Started + complete events both fire.
	require.Len(t, events, 2)
	assert.Equal(t, assetlibrary.ExampleEventDownloadStarted, events[0].Kind)
	assert.Equal(t, assetlibrary.ExampleEventDownloadComplete, events[1].Kind)
	assert.NoError(t, events[1].Err)
	assert.Equal(t, cachePath, events[1].Path)
}

func TestFetchExample_BadChecksumReturnsErrChecksumMismatch(t *testing.T) {
	cacheRoot := t.TempDir()
	lib := assetlibrary.NewLibrary(cacheRoot)

	body := validPforgeBytes()
	wrongHash := sha256Hex([]byte("totally different bytes"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	loadExampleManifest(t, lib, "bad_proof", srv.URL+"/bad.pforge", wrongHash)

	var events []assetlibrary.ExampleEvent
	_, err := assetlibrary.FetchExample(context.Background(), lib, "bad_proof", assetlibrary.ExampleFetchOptions{
		HTTPClient: srv.Client(),
		Progress:   func(ev assetlibrary.ExampleEvent) { events = append(events, ev) },
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, assetlibrary.ErrChecksumMismatch)

	// Nothing landed on disk — partial / poisoned downloads must
	// not pollute the cache.
	_, statErr := os.Stat(assetlibrary.ExampleCachePath(cacheRoot, "bad_proof"))
	assert.True(t, os.IsNotExist(statErr), "no cache file on checksum failure")

	// The progress callback saw a completion event carrying the error.
	var sawErr bool
	for _, ev := range events {
		if errors.Is(ev.Err, assetlibrary.ErrChecksumMismatch) {
			sawErr = true
			break
		}
	}
	assert.True(t, sawErr, "progress must report ErrChecksumMismatch on bad bytes")
}

func TestFetchExample_NetworkFailureSurfacesError(t *testing.T) {
	cacheRoot := t.TempDir()
	lib := assetlibrary.NewLibrary(cacheRoot)

	// Server that returns 503 for every request — simulates a
	// CDN outage. Library still surfaces the error to the menu
	// so it can render the red toast + leave the row enabled
	// for retry.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	loadExampleManifest(t, lib, "down_proof", srv.URL+"/down.pforge", sha256Hex(validPforgeBytes()))

	_, err := assetlibrary.FetchExample(context.Background(), lib, "down_proof", assetlibrary.ExampleFetchOptions{
		HTTPClient: srv.Client(),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, assetlibrary.ErrPackUnavailable)
}

func TestFetchExample_StaleCacheRefetches(t *testing.T) {
	// Cached bytes whose SHA-256 doesn't match the manifest must
	// fall through to the download path. Covers the "manifest
	// bumped without bumping Version" + "corrupted prior download"
	// cases.
	cacheRoot := t.TempDir()
	lib := assetlibrary.NewLibrary(cacheRoot)

	freshBody := validPforgeBytes()
	freshHash := sha256Hex(freshBody)

	cachePath := assetlibrary.ExampleCachePath(cacheRoot, "stale_proof")
	require.NoError(t, os.MkdirAll(filepath.Dir(cachePath), 0o755))
	require.NoError(t, os.WriteFile(cachePath, []byte("stale corrupted content"), 0o644))

	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write(freshBody)
	}))
	t.Cleanup(srv.Close)
	loadExampleManifest(t, lib, "stale_proof", srv.URL+"/stale.pforge", freshHash)

	got, err := assetlibrary.FetchExample(context.Background(), lib, "stale_proof", assetlibrary.ExampleFetchOptions{
		HTTPClient: srv.Client(),
	})
	require.NoError(t, err)
	assert.Equal(t, cachePath, got)
	assert.Equal(t, int64(1), hits.Load(), "stale cache must trigger one re-download")

	final, err := os.ReadFile(cachePath)
	require.NoError(t, err)
	assert.Equal(t, freshBody, final)
}

func TestFetchExample_ExamplesDirAndCachePathStable(t *testing.T) {
	// The on-disk layout is part of the public contract — tests +
	// the fork action depend on it. Pin it.
	root := "/tmp/forge-root"
	assert.Equal(t, "/tmp/forge-root/examples", assetlibrary.ExamplesDir(root))
	assert.Equal(t, "/tmp/forge-root/examples/foo/foo.pforge", assetlibrary.ExampleCachePath(root, "foo"))
}

func TestLibrary_ExamplesReturnsSortedCopy(t *testing.T) {
	lib := assetlibrary.NewLibrary(t.TempDir())
	body := `{
		"schema_version":"1",
		"packs":[],
		"examples":[
			{"id":"mario_proof","version":"1","url":"https://x/m.pforge","sha256":"a"},
			{"id":"asteroids_proof","version":"1","url":"https://x/a.pforge","sha256":"b"}
		]
	}`
	m, err := assetlibrary.ParseManifest([]byte(body))
	require.NoError(t, err)
	lib.SetManifest(m)

	got := lib.Examples()
	require.Len(t, got, 2)
	assert.Equal(t, "asteroids_proof", got[0].ID, "Examples() must sort by ID for stable menu rendering")
	assert.Equal(t, "mario_proof", got[1].ID)
}
