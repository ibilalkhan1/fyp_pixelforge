package ingest

import (
	"fmt"
	"log"
	"sync"
)

// Runner is the contract per-kind importers satisfy. One method —
// Run(path) — keeps the boundary minimal so the editor's PNG /
// WAV / OGG / MP3 import handlers can wrap their existing entry
// points without restructuring.
//
// Cycle-break pattern: ingest defines the interface; outer
// packages (palette, audiolib, future BGM library) inject
// implementations via Dispatcher.SetSpriteRunner etc. at studio
// startup, mirroring PNGImportRunner / WAVImportRunner from
// plan-005.
type Runner interface {
	Run(path string) error
}

// RunnerFunc adapts a bare function to the Runner interface.
// Tests use this to wire in-line capture closures without
// declaring a separate type per case.
type RunnerFunc func(path string) error

// Run satisfies Runner.
func (f RunnerFunc) Run(path string) error { return f(path) }

// Dispatcher is the per-kind routing table the watcher + drag-drop
// paths converge on. SetSpriteRunner / SetSFXRunner / SetBGMRunner
// install per-kind implementations; Ingest classifies + routes.
type Dispatcher struct {
	mu     sync.RWMutex
	sprite Runner
	sfx    Runner
	bgm    Runner
}

// NewDispatcher returns an empty dispatcher. Runners are wired
// later via the per-kind setters; missing runners cause Ingest
// to no-op with a debug log so a half-configured studio doesn't
// crash on a drop.
func NewDispatcher() *Dispatcher { return &Dispatcher{} }

// SetSpriteRunner installs the PNG ingest runner. Replacing an
// existing runner is safe; tests use this to swap fakes between
// cases.
func (d *Dispatcher) SetSpriteRunner(r Runner) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sprite = r
}

// SetSFXRunner installs the WAV ingest runner.
func (d *Dispatcher) SetSFXRunner(r Runner) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sfx = r
}

// SetBGMRunner installs the OGG / MP3 ingest runner.
func (d *Dispatcher) SetBGMRunner(r Runner) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.bgm = r
}

// Ingest classifies path by extension and routes to the matching
// runner. Returns the runner's error (so the watcher loop can
// toast it); KindUnknown + missing runners both no-op without
// error. Empty path is rejected up-front so a stray fsnotify
// event for "" doesn't fan out into a runner call.
func (d *Dispatcher) Ingest(path string) error {
	if path == "" {
		return fmt.Errorf("ingest: empty path")
	}
	kind := Classify(path)
	d.mu.RLock()
	defer d.mu.RUnlock()
	switch kind {
	case KindSprite:
		if d.sprite == nil {
			log.Printf("[ingest] %s classified as sprite but no runner registered", path)
			return nil
		}
		return d.sprite.Run(path)
	case KindSFX:
		if d.sfx == nil {
			log.Printf("[ingest] %s classified as sfx but no runner registered", path)
			return nil
		}
		return d.sfx.Run(path)
	case KindBGM:
		if d.bgm == nil {
			log.Printf("[ingest] %s classified as bgm but no runner registered", path)
			return nil
		}
		return d.bgm.Run(path)
	}
	// Unknown extension — silent no-op.
	return nil
}

// HasSpriteRunner reports whether a sprite runner has been
// installed. Test helper + diagnostic.
func (d *Dispatcher) HasSpriteRunner() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.sprite != nil
}

// HasSFXRunner reports whether a SFX runner has been installed.
func (d *Dispatcher) HasSFXRunner() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.sfx != nil
}

// HasBGMRunner reports whether a BGM runner has been installed.
func (d *Dispatcher) HasBGMRunner() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.bgm != nil
}
