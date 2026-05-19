// backend_js_test.go runs only under js/wasm (typically via
// `wasmbrowsertest`, e.g. `GOOS=js GOARCH=wasm go test
// -exec="$(go env GOPATH)/bin/wasmbrowsertest" ./pixelforge_save/`).
// When wasmbrowsertest is unavailable, verify manually in a real
// browser: open a generated WASM game, fire a save_now verb, refresh
// the page, fire a load_slot verb, and confirm the saved bytes
// round-trip via the browser's DevTools Application -> Local Storage
// view.

//go:build js

package pixelforge_save_test

import (
	"errors"
	"syscall/js"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"
)

// clearLocalStorage wipes the browser-side localStorage so each
// test starts hermetic.
func clearLocalStorage(t *testing.T) {
	t.Helper()
	js.Global().Get("localStorage").Call("clear")
}

func TestBackendJS_WriteThenReadRoundTrips(t *testing.T) {
	clearLocalStorage(t)
	b := pisave.NewBackendJS("Test Game")
	require.NoError(t, b.Write("slot1", []byte("hello")))
	got, err := b.Read("slot1")
	require.NoError(t, err)
	assert.Equal(t, "hello", string(got))
}

func TestBackendJS_ReadMissingReturnsErrNotFound(t *testing.T) {
	clearLocalStorage(t)
	b := pisave.NewBackendJS("Test Game")
	_, err := b.Read("never_written")
	require.Error(t, err)
	assert.True(t, errors.Is(err, pisave.ErrNotFound),
		"missing slot must surface ErrNotFound, got %v", err)
}

func TestBackendJS_DeleteRemovesSlot(t *testing.T) {
	clearLocalStorage(t)
	b := pisave.NewBackendJS("Test Game")
	require.NoError(t, b.Write("slot1", []byte("hi")))
	require.NoError(t, b.Delete("slot1"))
	_, err := b.Read("slot1")
	assert.True(t, errors.Is(err, pisave.ErrNotFound))
}

func TestBackendJS_DeleteMissingIsNoOp(t *testing.T) {
	clearLocalStorage(t)
	b := pisave.NewBackendJS("Test Game")
	assert.NoError(t, b.Delete("never_existed"))
}

func TestBackendJS_ListReturnsWrittenSlots(t *testing.T) {
	clearLocalStorage(t)
	b := pisave.NewBackendJS("Test Game")
	require.NoError(t, b.Write("slot1", []byte("a")))
	require.NoError(t, b.Write("slot2", []byte("bb")))
	got, err := b.List()
	require.NoError(t, err)
	names := []string{}
	for _, m := range got {
		names = append(names, m.Name)
	}
	assert.ElementsMatch(t, []string{"slot1", "slot2"}, names)
}

func TestBackendJS_IsolatesGamesByTitle(t *testing.T) {
	clearLocalStorage(t)
	a := pisave.NewBackendJS("game_a")
	b := pisave.NewBackendJS("game_b")
	require.NoError(t, a.Write("slot1", []byte("from_a")))
	require.NoError(t, b.Write("slot1", []byte("from_b")))

	// Each backend sees only its own data.
	aList, err := a.List()
	require.NoError(t, err)
	assert.Len(t, aList, 1)

	bList, err := b.List()
	require.NoError(t, err)
	assert.Len(t, bList, 1)

	gotA, _ := a.Read("slot1")
	gotB, _ := b.Read("slot1")
	assert.Equal(t, "from_a", string(gotA))
	assert.Equal(t, "from_b", string(gotB))
}

func TestBackendJS_KeyPrefixHelperMatchesNewBackend(t *testing.T) {
	b := pisave.NewBackendJS("My Game")
	assert.Equal(t, pisave.JSKeyPrefix("My Game"), b.KeyPrefix())
}
