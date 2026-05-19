package ingest

import (
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// DropSource is the injection seam for Ebitengine's DroppedFiles.
// Production binds it to DefaultDropSource (which returns the
// current drop set as a single fs.FS, or nil when no drops are
// pending); tests substitute a fake that returns canned drops
// without needing an Ebitengine window.
type DropSource func() fs.FS

// DefaultDropSource adapts Ebitengine 2.9.9's DroppedFiles() into
// the DropSource shape. Returns the live virtual filesystem
// Ebitengine populates each frame; an empty FS (no drops) walks
// to zero files.
func DefaultDropSource() fs.FS {
	return ebiten.DroppedFiles()
}

// DragDrop is the per-frame drop reader. Poll() drains the
// source, walks each manifest's regular files, copies them into
// stagingDir, and routes through dispatcher.Ingest.
//
// Drag-drop drops aren't paths on disk — Ebitengine wraps them
// in an io.FS abstraction (mobile-portable). We materialise each
// dropped file into the staging directory so the per-kind runners
// receive a stable on-disk path.
type DragDrop struct {
	mu         sync.Mutex
	source     DropSource
	dispatcher *Dispatcher
	stagingDir string
}

// NewDragDrop binds a poller to dispatcher. stagingDir is where
// dropped files materialise before ingest — pass the same
// user-library directory the watcher reads so drops land
// alongside watched-folder additions.
//
// source defaults to DefaultDropSource when nil; tests pass a
// canned source.
func NewDragDrop(dispatcher *Dispatcher, stagingDir string, source DropSource) *DragDrop {
	if source == nil {
		source = DefaultDropSource
	}
	return &DragDrop{
		source:     source,
		dispatcher: dispatcher,
		stagingDir: stagingDir,
	}
}

// Poll drains the source, copies each dropped file into the
// staging directory, and dispatches via dispatcher.Ingest.
// Returns the count of successfully-dispatched files (useful
// for tests + the eventual UI toast).
//
// Per-file errors log + continue so one malformed drop doesn't
// stall the batch.
func (d *DragDrop) Poll() int {
	if d == nil {
		return 0
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	drop := d.source()
	if drop == nil {
		return 0
	}
	if err := os.MkdirAll(d.stagingDir, 0o755); err != nil {
		log.Printf("[ingest] dragdrop: mkdir staging %s: %v", d.stagingDir, err)
		return 0
	}
	return d.processManifest(drop)
}

// processManifest walks every regular file in the dropped fs.FS,
// materialises asset-extension files into stagingDir, and
// dispatches each via the registered Runner.
func (d *DragDrop) processManifest(fsys fs.FS) int {
	if fsys == nil {
		return 0
	}
	count := 0
	walkErr := fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			log.Printf("[ingest] dragdrop walk %s: %v", name, walkErr)
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !IsAssetExtension(name) {
			return nil
		}
		dst, err := d.materialise(fsys, name)
		if err != nil {
			log.Printf("[ingest] dragdrop materialise %s: %v", name, err)
			return nil
		}
		if err := d.dispatcher.Ingest(dst); err != nil {
			log.Printf("[ingest] dragdrop ingest %s: %v", dst, err)
			return nil
		}
		count++
		return nil
	})
	if walkErr != nil {
		log.Printf("[ingest] dragdrop walk root: %v", walkErr)
	}
	return count
}

// materialise reads the named file out of fsys and writes a copy
// into d.stagingDir, preserving the basename. Subdirectories in
// the source manifest collapse into the staging dir (we don't
// re-create them) so the dispatcher always sees a flat layout.
// Returns the absolute on-disk path of the materialised file.
func (d *DragDrop) materialise(fsys fs.FS, name string) (string, error) {
	src, err := fsys.Open(name)
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst := filepath.Join(d.stagingDir, filepath.Base(name))
	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, src); err != nil {
		return "", err
	}
	return dst, nil
}
