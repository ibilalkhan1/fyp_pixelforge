package pixelforge_save_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"
)

// JSKeyPrefix is pure-Go (no syscall/js) so the host test suite
// can verify the WASM backend's namespacing contract without
// wasmbrowsertest. The full Read/Write/Delete/List round-trip is
// covered by backend_js_test.go under //go:build js (runs via
// wasmbrowsertest in CI when configured).

func TestJSKeyPrefix_NamespacesUnderPixelforge(t *testing.T) {
	prefix := pisave.JSKeyPrefix("My Game")
	assert.True(t, strings.HasPrefix(prefix, "pixelforge.save."),
		"all WASM save keys must live under the pixelforge.save namespace")
	assert.True(t, strings.HasSuffix(prefix, "."),
		"prefix must end with '.' so the slot name concatenates without ambiguity")
}

func TestJSKeyPrefix_SanitizesTitle(t *testing.T) {
	prefix := pisave.JSKeyPrefix("My Game!!!")
	// Sanitize lower-cases letters and strips non-alphanumerics
	// (spaces collapse to underscores).
	assert.Equal(t, "pixelforge.save.my_game.", prefix)
}

func TestJSKeyPrefix_IsolatesDifferentGames(t *testing.T) {
	a := pisave.JSKeyPrefix("game_a")
	b := pisave.JSKeyPrefix("game_b")
	assert.NotEqual(t, a, b,
		"two different game titles must derive different prefixes so saves don't collide on a shared origin")
}

func TestJSKeyPrefix_EmptyTitleDefaultsToUntitled(t *testing.T) {
	prefix := pisave.JSKeyPrefix("")
	assert.Equal(t, "pixelforge.save.untitled.", prefix)
}
