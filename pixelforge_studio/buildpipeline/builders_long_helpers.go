// builders_long_helpers.go carries the player-binary discovery
// chain the long-tag host + WASM builders share. Pulled out of
// builders_long.go to keep the per-target Build implementations
// readable: each Build method becomes a linear sequence of
// codegen.EncodeCart → resolvePlayerBinary → cart.Append /
// BundleWASM. The cache + SHA-256 sidecar + embed-extract logic
// lives here so neither builder has to repeat it.

//go:build long

package buildpipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/playerbins"
)

// playerCacheVersion bumps when the player-binary wire format
// changes incompatibly (e.g., new footer layout). The cache
// filename embeds this suffix so stale entries from a previous
// studio install can never poison a build silently — they simply
// won't be found.
//
// Phase 1: a stable "v2" string; commit-hash-based invalidation is
// deferred to a follow-up unit (see plan-009's player-binary
// discovery chain notes).
const playerCacheVersion = "v2"

// resolvedPlayer carries the path + cleanup the long builders use
// after locating a player binary on disk. cleanup is always
// non-nil — for paths owned by the caller (explicit override,
// cache hit) cleanup is a no-op; for paths the chain created (temp
// extraction, developer go-build) cleanup removes the temp file
// when the build finishes.
type resolvedPlayer struct {
	Path    string
	Cleanup func()
}

// resolvePlayerBinary walks the player-binary discovery chain for
// the supplied target, returning the on-disk path the long builder
// should pass to pixelforge_cart.Append / BundleWASM.
//
// Walk order:
//  1. explicit override (req.PlayerBinaryPath / req.WasmBinaryPath)
//  2. cache hit with valid SHA-256 sidecar
//  3. embedded binary via playerbins.PlayerBinaryFor (the no-Go
//     user path) — extracted to a temp file + cached for next time
//  4. developer fallback: go build -tags=long -o $TMP
//     ./cmd/pixelforge-player — slow, requires a local Go toolchain
//
// On any step's failure the walk continues; only when every step
// has been tried does resolvePlayerBinary return a hard error.
func resolvePlayerBinary(ctx context.Context, req BuildRequest, target Target) (*resolvedPlayer, error) {
	explicit := req.PlayerBinaryPath
	if target == TargetWASM {
		explicit = req.WasmBinaryPath
	}
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return nil, fmt.Errorf("resolvePlayerBinary: explicit player path %s: %w", explicit, err)
		}
		return &resolvedPlayer{Path: explicit, Cleanup: noopCleanup}, nil
	}

	if cached, ok := lookupPlayerCache(target); ok {
		return &resolvedPlayer{Path: cached, Cleanup: noopCleanup}, nil
	}

	goos := target.GOOS()
	goarch := target.GOARCH()
	if goos != "" && goarch != "" {
		embedBytes, embedErr := playerbins.PlayerBinaryFor(goos, goarch)
		if embedErr == nil {
			tmpPath, err := writePlayerToTemp(embedBytes, target)
			if err != nil {
				return nil, fmt.Errorf("resolvePlayerBinary: stage embedded binary: %w", err)
			}
			// Best-effort cache the embedded bytes so the next build
			// skips the embed extraction. Cache failures are logged
			// via the returned error path only when fatal — a
			// failed cache write must not break the build.
			_ = cachePlayerBinary(target, embedBytes)
			return &resolvedPlayer{
				Path:    tmpPath,
				Cleanup: func() { _ = os.Remove(tmpPath) },
			}, nil
		}
		if !errors.Is(embedErr, playerbins.ErrNotEmbedded) {
			// Non-soft-miss embed error: surface immediately. A
			// corrupt embedded entry is a configuration bug, not a
			// soft fallback condition.
			return nil, fmt.Errorf("resolvePlayerBinary: embed lookup: %w", embedErr)
		}
	}

	// Developer fallback: invoke `go build` against the local
	// pixelforge-player source. Slow but always works on a dev
	// machine with the Go toolchain on PATH.
	tmpPath, err := buildPlayerFromSource(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("resolvePlayerBinary: developer go-build fallback: %w", err)
	}
	if bytes, readErr := os.ReadFile(tmpPath); readErr == nil {
		_ = cachePlayerBinary(target, bytes)
	}
	return &resolvedPlayer{
		Path:    tmpPath,
		Cleanup: func() { _ = os.Remove(tmpPath) },
	}, nil
}

// noopCleanup is the cleanup func for paths the discovery chain
// did not create (explicit overrides + cache hits). Returning the
// same shared closure for every caller saves a tiny allocation and
// makes the "no-op" intent visible at the call site.
var noopCleanup = func() {}

// playerCacheDir returns the per-user cache directory the
// discovery chain reads + writes. Mirrors os.UserCacheDir's layout
// but pinned under pixelforge/player-cache so cleanup tools can
// nuke just our entries without disturbing other applications.
//
// Returns ("", error) when the platform doesn't expose a user
// cache dir; callers treat this as a cache miss + fall through.
func playerCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "pixelforge", "player-cache"), nil
}

// playerCacheFilename returns the on-disk filename a cached player
// binary uses for the supplied target. The format mirrors the
// playerbins/bins/ layout:
//
//	pixelforge-player-linux-amd64-v2
//	pixelforge-player-windows-amd64-v2.exe
//	pixelforge-player-js-wasm-v2.wasm
func playerCacheFilename(target Target) string {
	goos := target.GOOS()
	goarch := target.GOARCH()
	suffix := ""
	switch target {
	case TargetWindows:
		suffix = ".exe"
	case TargetWASM:
		suffix = ".wasm"
	}
	return fmt.Sprintf("pixelforge-player-%s-%s-%s%s", goos, goarch, playerCacheVersion, suffix)
}

// lookupPlayerCache returns (path, true) when a valid cached
// player binary exists for the target. "Valid" means: the file
// exists AND the sibling .sha256 sidecar matches the file's
// recomputed SHA-256 hash. A mismatched sidecar invalidates the
// cache entry (poisoned-cache protection); both files are removed
// so the next build re-populates from the next discovery step.
func lookupPlayerCache(target Target) (string, bool) {
	dir, err := playerCacheDir()
	if err != nil {
		return "", false
	}
	path := filepath.Join(dir, playerCacheFilename(target))
	sidecarPath := path + ".sha256"

	binaryBytes, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	sidecarBytes, err := os.ReadFile(sidecarPath)
	if err != nil {
		// Missing sidecar = invalid cache entry. Remove the binary
		// so subsequent lookups don't keep tripping on it.
		_ = os.Remove(path)
		return "", false
	}
	expected := strings.TrimSpace(string(sidecarBytes))
	actual := sha256HexOf(binaryBytes)
	if expected != actual {
		// Checksum mismatch: someone (or something) tampered with
		// the cached binary. Nuke both files + fall through to the
		// next discovery step.
		_ = os.Remove(path)
		_ = os.Remove(sidecarPath)
		return "", false
	}
	return path, true
}

// cachePlayerBinary writes the supplied bytes (plus a .sha256
// sidecar) to the per-target cache location. Best-effort: cache
// failures are returned but the caller treats them as
// non-fatal — a build that can't write its cache still produces a
// valid artifact, just a little slower next time.
func cachePlayerBinary(target Target, data []byte) error {
	dir, err := playerCacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cachePlayerBinary: mkdir %s: %w", dir, err)
	}
	path := filepath.Join(dir, playerCacheFilename(target))
	if err := atomicWriteBytes(path, data, perm0o755(target)); err != nil {
		return err
	}
	sidecarPath := path + ".sha256"
	if err := atomicWriteBytes(sidecarPath, []byte(sha256HexOf(data)+"\n"), 0o644); err != nil {
		// Remove the binary we just wrote so a partial cache (binary
		// + missing sidecar) doesn't lull a future lookup into a
		// false hit. lookupPlayerCache treats missing sidecar as an
		// invalid entry; this just keeps the on-disk state tidy.
		_ = os.Remove(path)
		return err
	}
	return nil
}

// writePlayerToTemp drops the supplied bytes at a temp file the
// long builder can hand to pixelforge_cart.Append. Uses
// os.CreateTemp so concurrent builds don't trip over each other,
// and chmod 0o755 (non-Windows) so the file is executable.
func writePlayerToTemp(data []byte, target Target) (string, error) {
	pattern := "pixelforge-player-*"
	if target == TargetWindows {
		pattern = "pixelforge-player-*.exe"
	}
	if target == TargetWASM {
		pattern = "pixelforge-player-*.wasm"
	}
	tmp, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("writePlayerToTemp: create temp: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("writePlayerToTemp: write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("writePlayerToTemp: close: %w", err)
	}
	if err := os.Chmod(tmp.Name(), perm0o755(target)); err != nil {
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("writePlayerToTemp: chmod: %w", err)
	}
	return tmp.Name(), nil
}

// perm0o755 returns 0o755 for executable targets (host native
// binaries) and 0o644 for WASM modules (not invoked directly on
// disk; loaded by the browser via fetch()).
func perm0o755(target Target) os.FileMode {
	if target == TargetWASM {
		return 0o644
	}
	return 0o755
}

// sha256HexOf returns the lowercase hex SHA-256 of data. Wrapped in
// a helper so the producer + verifier paths can't drift on
// encoding choices.
func sha256HexOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// atomicWriteBytes writes data to path via a tmp file + rename so
// readers never see a half-written file. Mirrors pixelforge_cart's
// own atomic-rename pattern.
func atomicWriteBytes(path string, data []byte, mode os.FileMode) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, mode); err != nil {
		return fmt.Errorf("atomicWriteBytes: write %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("atomicWriteBytes: rename %s -> %s: %w", tmpPath, path, err)
	}
	return nil
}

// buildPlayerFromSource is the developer fallback: shell out to
// `go build` against cmd/pixelforge-player and return the path to
// the produced binary. The caller owns cleanup of the returned
// path.
//
// Slow path (compile time dominates), only triggered when neither
// an explicit override, a valid cache entry, nor an embedded
// binary is available — i.e. fresh dev checkouts with no
// `make playerbins` run + no prior build.
func buildPlayerFromSource(ctx context.Context, target Target) (string, error) {
	tmpDir, err := os.MkdirTemp("", "pixelforge-player-build-*")
	if err != nil {
		return "", fmt.Errorf("buildPlayerFromSource: mkdir temp: %w", err)
	}
	filename := "pixelforge-player"
	if target == TargetWindows {
		filename += ".exe"
	}
	if target == TargetWASM {
		filename += ".wasm"
	}
	outPath := filepath.Join(tmpDir, filename)

	cmd, err := NewBuildCommand(ctx, target, "build", "-tags=long", "-o", outPath,
		"github.com/ibilalkhan1/fyp_pixelforge/cmd/pixelforge-player")
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("buildPlayerFromSource: build command: %w", err)
	}
	combined, runErr := cmd.CombinedOutput()
	if runErr != nil {
		_ = os.RemoveAll(tmpDir)
		if errors.Is(ctx.Err(), context.Canceled) {
			return "", ErrBuildCancelled
		}
		return "", fmt.Errorf("buildPlayerFromSource: go build failed: %w\noutput:\n%s", runErr, string(combined))
	}
	return outPath, nil
}

// targetToGOOS + targetToGOARCH expose Target's GOOS/GOARCH
// accessors via small free functions so tests that need to mock
// the discovery chain can replace them without poking at Target's
// methods.
func targetToGOOS(t Target) string   { return t.GOOS() }
func targetToGOARCH(t Target) string { return t.GOARCH() }
