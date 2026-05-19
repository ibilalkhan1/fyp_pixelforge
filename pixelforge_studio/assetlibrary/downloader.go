package assetlibrary

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DefaultManifestURL is the production manifest location. Tests
// + local dev override via the PIXELFORGE_ASSET_LIBRARY_URL env
// var (or by passing a custom URL through DownloadOptions).
//
// The location lives in the same GitHub Release that hosts the
// pack tarballs; the manifest is the first file the downloader
// fetches, which then directs it to each pack URL.
const DefaultManifestURL = "https://github.com/ibilalkhan1/fyp_pixelforge/releases/download/asset-library-v1/manifest.json"

// ManifestURLEnvVar is the env-var name that overrides
// DefaultManifestURL at startup. Used by local dev (point at a
// http.FileServer-served manifest) and by tests (point at a
// httptest.NewServer fixture).
const ManifestURLEnvVar = "PIXELFORGE_ASSET_LIBRARY_URL"

// ErrChecksumMismatch is returned when a downloaded pack's
// SHA-256 differs from the manifest-declared hash. The tmp file
// is removed before the error surfaces so partial / poisoned
// downloads never land in the library cache.
var ErrChecksumMismatch = errors.New("assetlibrary: downloaded pack failed SHA-256 verification")

// ErrPackUnavailable is returned when the pack URL responds with
// a non-2xx HTTP status. The pack id appears in the wrapped
// error so the failure is diagnosable.
var ErrPackUnavailable = errors.New("assetlibrary: pack URL unavailable")

// DownloadOptions tunes EnsureBootstrap. Zero-value is the
// production posture: default manifest URL, default HTTP client,
// no per-pack progress callback. Tests inject a mock URL +
// progress recorder.
type DownloadOptions struct {
	// ManifestURL overrides DefaultManifestURL + the env-var
	// override. Tests pin this to a httptest.NewServer URL.
	ManifestURL string

	// HTTPClient overrides http.DefaultClient. Tests inject a
	// client with a custom Transport so they don't hit the real
	// network.
	HTTPClient *http.Client

	// Progress is invoked at major boundaries: manifest fetched,
	// each pack downloaded, each pack unpacked. Optional — nil
	// means "no UI mirror"; the daemon hook (U12 toast) sets
	// this to a recorder that publishes to the chrome status pane.
	Progress func(Event)
}

// EventKind classifies progress events the downloader emits.
// Workspace toasts switch on this to format the message.
type EventKind int

const (
	// EventManifestFetched fires after the manifest HTTP fetch
	// completes (success or failure — Err is non-nil on failure).
	EventManifestFetched EventKind = iota

	// EventPackDownloaded fires after a single pack's tarball is
	// written + verified. PackID + Path are populated.
	EventPackDownloaded

	// EventPackInstalled fires after a single pack is unpacked
	// + registered in the in-memory library index.
	EventPackInstalled

	// EventPackSkipped fires when a pack's manifest version
	// matches the version already on disk — no re-download.
	EventPackSkipped
)

// Event is one progress callback payload.
type Event struct {
	Kind   EventKind
	PackID string
	Path   string // pack tarball or unpacked dir, depending on Kind
	Err    error
}

// bootstrapLock serialises concurrent EnsureBootstrap callers so
// two startup goroutines don't race the same download. The lock
// is per-cacheRoot in the map below.
var (
	bootstrapMu      sync.Mutex
	bootstrapPerRoot = map[string]*sync.Mutex{}
)

// EnsureBootstrap is the canonical startup entry point. Fetches
// the manifest, downloads each pack with checksum verification +
// atomic write, unpacks under <cacheRoot>/library/<packID>/, and
// records each pack in lib's in-memory index. Idempotent — re-
// running with the same manifest version no-ops.
//
// Concurrent calls coalesce: if two goroutines invoke
// EnsureBootstrap for the same cacheRoot, the second blocks until
// the first completes. The check is per-cacheRoot so tests using
// t.TempDir() don't serialise across cases.
//
// Errors surface from fundamental failures (manifest unreachable,
// schema unsupported, every pack failed). Per-pack failures are
// reported via Progress and don't fail EnsureBootstrap — a
// partial library is better than no library.
func EnsureBootstrap(ctx context.Context, lib *Library, opts DownloadOptions) error {
	if lib == nil {
		return errors.New("assetlibrary: EnsureBootstrap: nil library")
	}
	cacheRoot := lib.CacheRoot()
	if cacheRoot == "" {
		return errors.New("assetlibrary: EnsureBootstrap: library has empty cacheRoot")
	}
	// Per-cacheRoot serialisation. Multiple startup goroutines
	// for the same studio invocation coalesce here; the second
	// goroutine sees the first's results when MarkInstalled has
	// completed.
	lock := lockFor(cacheRoot)
	lock.Lock()
	defer lock.Unlock()

	url := resolveManifestURL(opts.ManifestURL)
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	manifestBytes, err := fetchURL(ctx, client, url)
	if err != nil {
		emit(opts.Progress, Event{Kind: EventManifestFetched, Err: err})
		return fmt.Errorf("assetlibrary: fetch manifest %s: %w", url, err)
	}
	manifest, err := ParseManifest(manifestBytes)
	if err != nil {
		emit(opts.Progress, Event{Kind: EventManifestFetched, Err: err})
		return err
	}
	lib.SetManifest(manifest)
	emit(opts.Progress, Event{Kind: EventManifestFetched})

	if err := os.MkdirAll(libraryRoot(cacheRoot), 0o755); err != nil {
		return fmt.Errorf("assetlibrary: mkdir library root: %w", err)
	}

	for _, pack := range manifest.Packs {
		pack := pack
		if err := installPack(ctx, client, cacheRoot, &pack, lib, opts.Progress); err != nil {
			// Per-pack errors surface via Progress; the bootstrap
			// itself continues so one bad pack doesn't strand a
			// fresh studio install with zero assets.
			emit(opts.Progress, Event{Kind: EventPackInstalled, PackID: pack.ID, Err: err})
			continue
		}
	}
	return nil
}

// installPack is the per-pack pipeline: skip-if-installed → fetch
// → verify → atomic write → unpack → MarkInstalled. Each step
// reports via the Progress callback.
func installPack(ctx context.Context, client *http.Client, cacheRoot string, pack *Pack, lib *Library, progress func(Event)) error {
	dst := PackDir(cacheRoot, pack.ID)
	if existingVersion, ok := readInstalledVersion(dst); ok && existingVersion == pack.Version {
		// Same version already on disk — register in the in-mem
		// index so callers see the pack as installed without
		// re-downloading.
		lib.MarkInstalled(*pack)
		emit(progress, Event{Kind: EventPackSkipped, PackID: pack.ID, Path: dst})
		return nil
	}

	// Atomic write: download to a tmp file in libraryRoot, verify
	// SHA-256, then rename into place. A partial / poisoned
	// download is removed before the error surfaces — nothing
	// stale leaks into the library cache.
	tmpFile, err := os.CreateTemp(libraryRoot(cacheRoot), pack.ID+"-*.dl")
	if err != nil {
		return fmt.Errorf("create temp for pack %s: %w", pack.ID, err)
	}
	tmpPath := tmpFile.Name()
	cleanup := func() { os.Remove(tmpPath) }

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pack.URL, nil)
	if err != nil {
		cleanup()
		tmpFile.Close()
		return fmt.Errorf("pack %s: build request: %w", pack.ID, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		cleanup()
		tmpFile.Close()
		return fmt.Errorf("pack %s: %w", pack.ID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		cleanup()
		tmpFile.Close()
		return fmt.Errorf("%w: pack %s returned HTTP %d", ErrPackUnavailable, pack.ID, resp.StatusCode)
	}

	hasher := sha256.New()
	if _, err := io.Copy(tmpFile, io.TeeReader(resp.Body, hasher)); err != nil {
		cleanup()
		tmpFile.Close()
		return fmt.Errorf("pack %s: copy: %w", pack.ID, err)
	}
	if err := tmpFile.Close(); err != nil {
		cleanup()
		return fmt.Errorf("pack %s: close: %w", pack.ID, err)
	}
	got := hex.EncodeToString(hasher.Sum(nil))
	if pack.SHA256 != "" && got != pack.SHA256 {
		cleanup()
		return fmt.Errorf("%w: pack %s got %s want %s", ErrChecksumMismatch, pack.ID, got, pack.SHA256)
	}
	emit(progress, Event{Kind: EventPackDownloaded, PackID: pack.ID, Path: tmpPath})

	// Replace any existing pack dir with the freshly-unpacked one.
	if err := os.RemoveAll(dst); err != nil {
		cleanup()
		return fmt.Errorf("pack %s: remove old dir: %w", pack.ID, err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		cleanup()
		return fmt.Errorf("pack %s: mkdir dst: %w", pack.ID, err)
	}
	if err := unpackTarGz(tmpPath, dst); err != nil {
		cleanup()
		return fmt.Errorf("pack %s: unpack: %w", pack.ID, err)
	}
	cleanup()

	if err := writeInstalledVersion(dst, pack.Version); err != nil {
		return fmt.Errorf("pack %s: stamp version: %w", pack.ID, err)
	}
	lib.MarkInstalled(*pack)
	emit(progress, Event{Kind: EventPackInstalled, PackID: pack.ID, Path: dst})
	return nil
}

// fetchURL is the GET-and-read-all helper. Returns the raw body
// bytes on 2xx, wraps the error on transport / non-2xx failures.
func fetchURL(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}

// unpackTarGz extracts the .tar.gz at archivePath into dst. Path
// traversal is guarded: any header naming a path outside dst is
// rejected (defends against malicious manifests). Regular files
// and directories are honoured; symlinks + devices are skipped.
func unpackTarGz(archivePath, dst string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		clean := filepath.Clean(hdr.Name)
		if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			return fmt.Errorf("assetlibrary: unsafe tar entry %q", hdr.Name)
		}
		target := filepath.Join(dst, clean)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
}

// readInstalledVersion reads the .pack-version sentinel inside a
// pack dir. Returns ("", false) when the file doesn't exist —
// treated as "not installed" so the downloader fetches fresh.
func readInstalledVersion(packDir string) (string, bool) {
	b, err := os.ReadFile(filepath.Join(packDir, ".pack-version"))
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(b)), true
}

// writeInstalledVersion stamps the pack dir with its manifest
// version so the next EnsureBootstrap can skip re-download.
func writeInstalledVersion(packDir, version string) error {
	return os.WriteFile(filepath.Join(packDir, ".pack-version"), []byte(version+"\n"), 0o644)
}

// resolveManifestURL picks the URL based on (in order): explicit
// override, env var, default.
func resolveManifestURL(override string) string {
	if override != "" {
		return override
	}
	if env := os.Getenv(ManifestURLEnvVar); env != "" {
		return env
	}
	return DefaultManifestURL
}

// lockFor returns the per-cacheRoot mutex. Tests with different
// t.TempDir() roots don't serialise against each other; a single
// studio process always sees the same lock for its single cache.
func lockFor(cacheRoot string) *sync.Mutex {
	bootstrapMu.Lock()
	defer bootstrapMu.Unlock()
	m, ok := bootstrapPerRoot[cacheRoot]
	if !ok {
		m = &sync.Mutex{}
		bootstrapPerRoot[cacheRoot] = m
	}
	return m
}

// emit forwards an event to the optional progress callback.
func emit(cb func(Event), ev Event) {
	if cb == nil {
		return
	}
	cb(ev)
}
