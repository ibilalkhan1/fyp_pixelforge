package ingest

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// DefaultDebounce is the window the watcher waits after the last
// observed event before firing Ingest. Tuned per the plan's
// implementation-time unknown (start at 500ms; lower if real OS
// event volume warrants).
const DefaultDebounce = 500 * time.Millisecond

// Watcher monitors a directory via fsnotify, debounces rapid
// events on the same path, and routes the resulting stable file
// to a Dispatcher.Ingest call. Non-asset extensions are filtered
// out before debouncing so a noisy directory (logs, build
// outputs) doesn't queue work.
type Watcher struct {
	dir        string
	dispatcher *Dispatcher
	debounce   time.Duration

	mu       sync.Mutex
	timers   map[string]*time.Timer
	watcher  *fsnotify.Watcher
	stopped  bool // gates scheduleIngest after Stop
	stopOnce sync.Once
	done     chan struct{}
}

// NewWatcher returns a Watcher rooted at dir, routing through
// dispatcher. Construction is pure — Start kicks off the
// goroutine + fsnotify subscription. debounce <= 0 falls back to
// DefaultDebounce.
func NewWatcher(dir string, dispatcher *Dispatcher, debounce time.Duration) *Watcher {
	if debounce <= 0 {
		debounce = DefaultDebounce
	}
	return &Watcher{
		dir:        dir,
		dispatcher: dispatcher,
		debounce:   debounce,
		timers:     map[string]*time.Timer{},
		done:       make(chan struct{}),
	}
}

// Start opens the fsnotify watcher, registers dir, and launches
// the event loop. Returns an error when dir doesn't exist or
// fsnotify rejects the subscription. Idempotent — calling Start
// twice returns nil for the second call once the first has
// established a watcher.
func (w *Watcher) Start() error {
	if w == nil {
		return errors.New("ingest: nil watcher")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.watcher != nil {
		return nil // already running
	}
	if err := os.MkdirAll(w.dir, 0o755); err != nil {
		return fmt.Errorf("ingest: mkdir %s: %w", w.dir, err)
	}
	notifier, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("ingest: fsnotify: %w", err)
	}
	if err := notifier.Add(w.dir); err != nil {
		notifier.Close()
		return fmt.Errorf("ingest: watch %s: %w", w.dir, err)
	}
	w.watcher = notifier
	go w.loop()
	return nil
}

// Stop closes the underlying fsnotify watcher and cancels any
// pending debounce timers. Safe to call multiple times; safe to
// call before Start.
func (w *Watcher) Stop() {
	if w == nil {
		return
	}
	w.stopOnce.Do(func() {
		w.mu.Lock()
		w.stopped = true
		for _, t := range w.timers {
			t.Stop()
		}
		w.timers = map[string]*time.Timer{}
		watcher := w.watcher
		w.watcher = nil
		w.mu.Unlock()
		if watcher != nil {
			watcher.Close()
		}
		close(w.done)
	})
}

// Done returns a channel that closes when Stop completes. Test
// helper — production callers don't typically need this.
func (w *Watcher) Done() <-chan struct{} {
	if w == nil {
		return nil
	}
	return w.done
}

// IngestNow bypasses debouncing and dispatches path immediately.
// The plan's "imperative test API — no input mocking" pattern —
// tests call this to exercise the routing without standing up
// fsnotify timing.
func (w *Watcher) IngestNow(path string) error {
	if w == nil || w.dispatcher == nil {
		return errors.New("ingest: watcher or dispatcher nil")
	}
	return w.dispatcher.Ingest(path)
}

// loop is the fsnotify event-pump goroutine. Each Create / Write
// event for a recognised asset extension restarts a per-path
// debounce timer; on timer fire the path dispatches once. Errors
// surface to log and continue.
func (w *Watcher) loop() {
	w.mu.Lock()
	notifier := w.watcher
	w.mu.Unlock()
	if notifier == nil {
		return
	}
	for {
		select {
		case ev, ok := <-notifier.Events:
			if !ok {
				return
			}
			// Filter to events that bring a new bytes-on-disk
			// state: Create + Write. Rename / Remove get processed
			// only insofar as they cancel pending debounces for the
			// removed path.
			if !ev.Has(fsnotify.Create) && !ev.Has(fsnotify.Write) {
				if ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename) {
					w.cancelTimer(ev.Name)
				}
				continue
			}
			if !IsAssetExtension(ev.Name) {
				continue
			}
			w.scheduleIngest(ev.Name)
		case err, ok := <-notifier.Errors:
			if !ok {
				return
			}
			log.Printf("[ingest] watcher error: %v", err)
		}
	}
}

// scheduleIngest restarts the per-path debounce timer; on
// elapse, the timer fires Dispatcher.Ingest exactly once. Rapid
// successive writes to the same file (e.g. a brush export
// streaming data into the watched folder) coalesce into a single
// ingest call once the writes settle.
func (w *Watcher) scheduleIngest(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		// Stop has been called; don't schedule new timers (they'd
		// fire after Stop and dispatch ingests against a stopped
		// watcher).
		return
	}
	if t, ok := w.timers[path]; ok {
		t.Stop()
	}
	w.timers[path] = time.AfterFunc(w.debounce, func() {
		w.mu.Lock()
		stopped := w.stopped
		delete(w.timers, path)
		dispatcher := w.dispatcher
		w.mu.Unlock()
		if stopped || dispatcher == nil {
			return
		}
		if err := dispatcher.Ingest(path); err != nil {
			log.Printf("[ingest] %s: %v", path, err)
		}
	})
}

// cancelTimer drops any pending debounce for path (the file was
// removed / renamed before the debounce window elapsed).
func (w *Watcher) cancelTimer(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if t, ok := w.timers[path]; ok {
		t.Stop()
		delete(w.timers, path)
	}
}

// PendingCount returns the number of debounce timers currently
// armed. Test helper to assert that rapid writes coalesce.
func (w *Watcher) PendingCount() int {
	if w == nil {
		return 0
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.timers)
}
