// backend_native.go is the os-filesystem backend the desktop
// runtime + the studio preview use. WASM builds (//go:build js) use
// backend_wasm.go instead via the build-tag matrix.
//
// Concretely: SaveToSlot writes to
// `<os.UserConfigDir()>/pixelforge-games/<sanitised-title>/<slot>.json`.
// The directory is created lazily on first write; tests use a
// per-test tmpdir via NewBackendNativeAtPath so they don't pollute
// the user's actual config dir.

//go:build !js

package pixelforge_save

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BackendNative is the on-disk Backend.
type BackendNative struct {
	dir string
}

// NewBackendNative binds a backend rooted at the per-game
// directory derived from the title via Sanitize. Created lazily
// (no I/O on construction; the first write triggers MkdirAll).
func NewBackendNative(gameTitle string) (*BackendNative, error) {
	dir, err := GameDir(gameTitle)
	if err != nil {
		return nil, err
	}
	return &BackendNative{dir: dir}, nil
}

// NewBackendNativeAtPath is the test-friendly variant — callers
// pass an explicit directory (typically t.TempDir()) so tests
// don't write into the real user-config-dir.
func NewBackendNativeAtPath(dir string) *BackendNative {
	return &BackendNative{dir: dir}
}

// Dir exposes the backend's root directory. Tests assert on this;
// the production loader uses it to compose a "where are saves
// stored?" line for the Save menu.
func (b *BackendNative) Dir() string { return b.dir }

func (b *BackendNative) Read(slotName string) ([]byte, error) {
	path := filepath.Join(b.dir, slotName+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("save read %s: %w", path, err)
	}
	return data, nil
}

func (b *BackendNative) Write(slotName string, data []byte) error {
	if err := os.MkdirAll(b.dir, 0o755); err != nil {
		return fmt.Errorf("save mkdir %s: %w", b.dir, err)
	}
	path := filepath.Join(b.dir, slotName+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("save write %s: %w", path, err)
	}
	return nil
}

func (b *BackendNative) Delete(slotName string) error {
	path := filepath.Join(b.dir, slotName+".json")
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("save delete %s: %w", path, err)
	}
	return nil
}

func (b *BackendNative) List() ([]SlotMeta, error) {
	entries, err := os.ReadDir(b.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("save list %s: %w", b.dir, err)
	}
	var out []SlotMeta
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, SlotMeta{
			Name:    strings.TrimSuffix(name, ".json"),
			SavedAt: info.ModTime(),
			Size:    info.Size(),
		})
	}
	return out, nil
}
