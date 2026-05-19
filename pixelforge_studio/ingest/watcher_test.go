package ingest_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/ingest"
)

// thread-safe recordingRunner for watcher tests that may dispatch
// from the fsnotify goroutine concurrent with the test goroutine
// reading calls.
type syncRunner struct {
	mu    sync.Mutex
	calls []string
}

func (r *syncRunner) Run(path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, path)
	return nil
}

func (r *syncRunner) Calls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.calls))
	copy(out, r.calls)
	return out
}

// waitFor polls until cond returns true or timeout elapses.
// Returns true on success, false on timeout. Used to bridge the
// fsnotify async path without a fixed sleep.
func waitFor(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return cond()
}

func TestWatcher_IngestNowBypassesFSNotify(t *testing.T) {
	dir := t.TempDir()
	disp := ingest.NewDispatcher()
	runner := &syncRunner{}
	disp.SetSpriteRunner(runner)
	w := ingest.NewWatcher(dir, disp, 10*time.Millisecond)
	t.Cleanup(w.Stop)
	require.NoError(t, w.IngestNow("/path/ship.png"))
	assert.Equal(t, []string{"/path/ship.png"}, runner.Calls())
}

func TestWatcher_StartCreatesDirAndWatches(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "user-library")
	disp := ingest.NewDispatcher()
	w := ingest.NewWatcher(dir, disp, 50*time.Millisecond)
	t.Cleanup(w.Stop)
	require.NoError(t, w.Start())
	// Dir must exist post-Start.
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestWatcher_FileAppearanceDispatchesAfterDebounce(t *testing.T) {
	dir := t.TempDir()
	disp := ingest.NewDispatcher()
	runner := &syncRunner{}
	disp.SetSpriteRunner(runner)
	w := ingest.NewWatcher(dir, disp, 80*time.Millisecond)
	t.Cleanup(w.Stop)
	require.NoError(t, w.Start())

	target := filepath.Join(dir, "ship.png")
	require.NoError(t, os.WriteFile(target, []byte("fake-png"), 0o644))

	ok := waitFor(2*time.Second, func() bool {
		return len(runner.Calls()) == 1
	})
	require.True(t, ok, "watcher should dispatch the dropped file once debounce elapses")
	assert.Equal(t, target, runner.Calls()[0])
}

func TestWatcher_UnrecognisedExtensionIgnored(t *testing.T) {
	dir := t.TempDir()
	disp := ingest.NewDispatcher()
	runner := &syncRunner{}
	disp.SetSpriteRunner(runner)
	w := ingest.NewWatcher(dir, disp, 50*time.Millisecond)
	t.Cleanup(w.Stop)
	require.NoError(t, w.Start())

	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644))
	// Wait a bit past the debounce — runner must stay empty.
	time.Sleep(300 * time.Millisecond)
	assert.Empty(t, runner.Calls(), "non-asset extensions must not dispatch")
}

func TestWatcher_RapidWritesCoalesce(t *testing.T) {
	dir := t.TempDir()
	disp := ingest.NewDispatcher()
	runner := &syncRunner{}
	disp.SetSpriteRunner(runner)
	w := ingest.NewWatcher(dir, disp, 200*time.Millisecond)
	t.Cleanup(w.Stop)
	require.NoError(t, w.Start())

	target := filepath.Join(dir, "ship.png")
	// Five rapid writes to the same file — each one restarts the
	// debounce timer, so only the final settled state dispatches.
	for i := 0; i < 5; i++ {
		require.NoError(t, os.WriteFile(target, []byte{byte(i)}, 0o644))
		time.Sleep(30 * time.Millisecond)
	}
	ok := waitFor(2*time.Second, func() bool {
		return len(runner.Calls()) >= 1
	})
	require.True(t, ok)
	// Exactly one dispatch — coalescing worked.
	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, 1, len(runner.Calls()),
		"5 rapid writes to the same file must coalesce into a single dispatch")
}

func TestWatcher_StopCancelsPendingTimers(t *testing.T) {
	dir := t.TempDir()
	disp := ingest.NewDispatcher()
	runner := &syncRunner{}
	disp.SetSpriteRunner(runner)
	w := ingest.NewWatcher(dir, disp, 500*time.Millisecond)
	require.NoError(t, w.Start())

	require.NoError(t, os.WriteFile(filepath.Join(dir, "ship.png"), []byte("x"), 0o644))
	// Stop immediately — pending debounce should never fire.
	w.Stop()
	time.Sleep(700 * time.Millisecond)
	assert.Empty(t, runner.Calls(), "Stop must cancel pending debounces")
}
