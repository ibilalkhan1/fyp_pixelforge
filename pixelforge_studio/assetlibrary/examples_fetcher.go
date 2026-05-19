package assetlibrary

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// ErrExampleUnknown is returned by FetchExample when the supplied
// id does not appear in the library's current manifest. Surfaces
// via the File → Open Example menu's error toast.
var ErrExampleUnknown = errors.New("assetlibrary: unknown example id")

// ExampleFetchOptions tunes FetchExample. Zero-value is the
// production posture: default HTTP client, no progress callback,
// the library's own cache root. Tests inject a custom client +
// override the cache root for hermetic operation.
type ExampleFetchOptions struct {
	// HTTPClient overrides http.DefaultClient. Tests inject a
	// client whose Transport points at a httptest.NewServer so
	// they don't hit the network.
	HTTPClient *http.Client

	// CacheRoot overrides lib.CacheRoot(). When empty, the
	// library's own root is used. Tests typically pin this to
	// t.TempDir() for hermetic operation.
	CacheRoot string

	// Progress is invoked once with EventManifestFetched-class
	// events as the fetch progresses. Optional — nil means "no
	// UI mirror"; the menu's toast wrapper sets this to a
	// recorder that publishes to the status bar.
	Progress func(ExampleEvent)
}

// ExampleEventKind classifies progress events FetchExample emits.
type ExampleEventKind int

const (
	// ExampleEventCacheHit fires when the on-disk cache holds a
	// byte-equivalent .pforge — the function returns the path
	// immediately without touching the network.
	ExampleEventCacheHit ExampleEventKind = iota

	// ExampleEventDownloadStarted fires just before the HTTP GET
	// against the example URL begins.
	ExampleEventDownloadStarted

	// ExampleEventDownloadComplete fires after the bytes have
	// been written + verified + are ready to open.
	ExampleEventDownloadComplete
)

// ExampleEvent is one progress callback payload.
type ExampleEvent struct {
	Kind      ExampleEventKind
	ExampleID string
	Path      string
	Err       error
}

// ExamplesDir returns the on-disk directory examples are cached
// under inside cacheRoot. Sibling to library/ (where packs land)
// so the cache hierarchy stays self-documenting.
func ExamplesDir(cacheRoot string) string {
	return filepath.Join(cacheRoot, "examples")
}

// ExampleCachePath returns the canonical filesystem path the
// example with id caches to. Cache layout is
// <cacheRoot>/examples/<id>/<id>.pforge so each example owns its
// own directory (a future v2 might cache sibling asset bundles
// next to the .pforge — having a dedicated dir keeps that
// expansion additive).
func ExampleCachePath(cacheRoot, id string) string {
	return filepath.Join(ExamplesDir(cacheRoot), id, id+".pforge")
}

// FetchExample resolves an example id to a local `.pforge` path,
// downloading + SHA-256-verifying on cache miss. The contract:
//
//   - Cache hit (bytes on disk match Example.SHA256): the cached
//     path is returned immediately; ExampleEventCacheHit fires.
//   - Cache miss / checksum drift: the URL is fetched, verified
//     against the manifest's expected SHA-256, and atomically
//     written to ExampleCachePath. ExampleEventDownloadStarted +
//     ExampleEventDownloadComplete bracket the work.
//   - Unknown id: returns ErrExampleUnknown without touching the
//     network or filesystem.
//   - Checksum mismatch on the freshly-downloaded body: returns
//     ErrChecksumMismatch and removes the partial file before
//     surfacing the error — nothing poisoned lands in the cache.
//
// The function is safe to call from a UI click handler — it
// blocks on the HTTP call so callers that need non-blocking
// behaviour should wrap it in a goroutine. Plan-009 U21's menu
// does exactly that + surfaces a "Downloading…" toast around the
// call.
func FetchExample(ctx context.Context, lib *Library, exampleID string, opts ExampleFetchOptions) (string, error) {
	if lib == nil {
		return "", errors.New("assetlibrary: FetchExample: nil library")
	}
	if exampleID == "" {
		return "", fmt.Errorf("%w: (empty)", ErrExampleUnknown)
	}

	example := lib.LookupExample(exampleID)
	if example == nil {
		return "", fmt.Errorf("%w: %q", ErrExampleUnknown, exampleID)
	}

	cacheRoot := opts.CacheRoot
	if cacheRoot == "" {
		cacheRoot = lib.CacheRoot()
	}
	if cacheRoot == "" {
		return "", errors.New("assetlibrary: FetchExample: no cache root resolved")
	}

	cachePath := ExampleCachePath(cacheRoot, exampleID)

	// Cache hit fast path — verify the bytes match the manifest
	// hash before trusting the cached file. A checksum drift on
	// disk (corrupted download from a prior session, manifest
	// bumped the SHA-256 without bumping Version, malicious local
	// modification) falls through to the re-download path so the
	// user transparently gets fresh-and-valid bytes.
	if matches, _ := verifyFileSHA256(cachePath, example.SHA256); matches {
		emitExampleEvent(opts.Progress, ExampleEvent{
			Kind:      ExampleEventCacheHit,
			ExampleID: exampleID,
			Path:      cachePath,
		})
		return cachePath, nil
	}

	// Cache miss — download + verify + atomic write.
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return "", fmt.Errorf("assetlibrary: FetchExample %s: mkdir: %w", exampleID, err)
	}

	emitExampleEvent(opts.Progress, ExampleEvent{
		Kind:      ExampleEventDownloadStarted,
		ExampleID: exampleID,
	})

	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	data, err := fetchExampleBody(ctx, client, example.URL, exampleID)
	if err != nil {
		emitExampleEvent(opts.Progress, ExampleEvent{
			Kind:      ExampleEventDownloadComplete,
			ExampleID: exampleID,
			Err:       err,
		})
		return "", err
	}

	if example.SHA256 != "" {
		got := sha256Hex(data)
		if got != example.SHA256 {
			err := fmt.Errorf("%w: example %s got %s want %s",
				ErrChecksumMismatch, exampleID, got, example.SHA256)
			emitExampleEvent(opts.Progress, ExampleEvent{
				Kind:      ExampleEventDownloadComplete,
				ExampleID: exampleID,
				Err:       err,
			})
			return "", err
		}
	}

	// Atomic write: write to a tmp sibling, then rename into
	// place so concurrent readers never see a partial file.
	tmpPath := cachePath + ".dl"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		emitExampleEvent(opts.Progress, ExampleEvent{
			Kind:      ExampleEventDownloadComplete,
			ExampleID: exampleID,
			Err:       err,
		})
		return "", fmt.Errorf("assetlibrary: FetchExample %s: write: %w", exampleID, err)
	}
	if err := os.Rename(tmpPath, cachePath); err != nil {
		_ = os.Remove(tmpPath)
		emitExampleEvent(opts.Progress, ExampleEvent{
			Kind:      ExampleEventDownloadComplete,
			ExampleID: exampleID,
			Err:       err,
		})
		return "", fmt.Errorf("assetlibrary: FetchExample %s: rename: %w", exampleID, err)
	}

	emitExampleEvent(opts.Progress, ExampleEvent{
		Kind:      ExampleEventDownloadComplete,
		ExampleID: exampleID,
		Path:      cachePath,
	})
	return cachePath, nil
}

// fetchExampleBody runs the HTTP GET and returns the body bytes
// on 2xx. Errors wrap ErrPackUnavailable for non-2xx so callers
// can errors.Is-detect the network-unavailable case.
func fetchExampleBody(ctx context.Context, client *http.Client, url, exampleID string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("assetlibrary: FetchExample %s: build request: %w", exampleID, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("assetlibrary: FetchExample %s: %w", exampleID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("%w: example %s returned HTTP %d", ErrPackUnavailable, exampleID, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// verifyFileSHA256 returns true when the file at path exists and
// its SHA-256 matches expectedHex. An empty expectedHex disables
// verification (returns false so the caller treats the file as
// missing — the manifest's contract is "SHA-256 mandatory for
// trust"). I/O errors return (false, err); the caller can ignore
// the error and treat the file as a cache miss.
func verifyFileSHA256(path, expectedHex string) (bool, error) {
	if expectedHex == "" {
		return false, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}
	return hex.EncodeToString(h.Sum(nil)) == expectedHex, nil
}

// sha256Hex returns the hex-encoded SHA-256 of b.
func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// emitExampleEvent forwards an event to the optional progress
// callback. nil callback is the no-op posture.
func emitExampleEvent(cb func(ExampleEvent), ev ExampleEvent) {
	if cb == nil {
		return
	}
	cb(ev)
}
