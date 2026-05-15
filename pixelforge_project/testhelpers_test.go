package pixelforge_project

import (
	"bytes"
	"encoding/json"
	"io"
	"time"
)

// bytesReader wraps a byte slice as an io.Reader; alias kept short for
// the loader tests.
func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }

// jsonUnmarshal is a thin alias so test helpers don't have to import
// encoding/json directly (cuts down on naked package imports in tests).
func jsonUnmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }

// pinClock replaces nowFunc with a fixed timestamp and returns a
// restorer the caller must defer. Used by determinism tests so two
// Save() calls don't pick up different real-clock values.
func pinClock() func() {
	prev := nowFunc
	fixed := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	nowFunc = func() time.Time { return fixed }
	return func() { nowFunc = prev }
}
