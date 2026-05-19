// Plan-009 U21 — bridge between the assetlibrary's example
// manifest + fetcher and the editor's OpenExampleSource seam.
//
// The editor package never imports assetlibrary (clean dependency
// direction: editor → widgets / project; assetlibrary → editor).
// This file lives in the assetlibrary package and adapts a
// *Library into the editor.OpenExampleSource interface, which
// main.go wires up immediately after editor construction. The
// alternative would be to inline the adapter in main.go itself,
// but keeping it next to the package whose types it adapts means
// future Example-field additions land here in one place.
package assetlibrary

import (
	"context"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
)

// AttachExampleSource binds lib to e.OpenExampleSource. Plan-009
// U21's main.go calls this after editor construction so the
// File → Open Example menu lights up the moment the background
// bootstrap drops the manifest into the library — no per-frame
// polling needed (Library.Examples is the cheap lookup the menu
// calls each frame).
//
// Passing a nil lib is a no-op (the menu then renders the
// "No examples available" disabled row).
func AttachExampleSource(e *editor.Editor, lib *Library) {
	if e == nil {
		return
	}
	if lib == nil {
		e.SetOpenExampleSource(nil)
		return
	}
	e.SetOpenExampleSource(&exampleSourceAdapter{lib: lib})
}

// exampleSourceAdapter wraps *Library to satisfy
// editor.OpenExampleSource. Methods convert assetlibrary.Example
// to editor.OpenExampleEntry and translate FetchExample's return
// shape into OpenExampleResult.
type exampleSourceAdapter struct {
	lib *Library
}

// Examples mirrors Library.Examples through the editor's entry
// type. Sorted by ID for stable menu rendering (Library.Examples
// already sorts; the conversion preserves order).
func (a *exampleSourceAdapter) Examples() []editor.OpenExampleEntry {
	if a == nil || a.lib == nil {
		return nil
	}
	src := a.lib.Examples()
	if len(src) == 0 {
		return nil
	}
	out := make([]editor.OpenExampleEntry, 0, len(src))
	for _, ex := range src {
		out = append(out, editor.OpenExampleEntry{
			ID:          ex.ID,
			Title:       ex.Title,
			Label:       ex.Label,
			Description: ex.Description,
			Game:        ex.Game,
			SizeBytes:   ex.SizeBytes,
		})
	}
	return out
}

// Fetch delegates to FetchExample. The default options posture is
// production: HTTPClient nil → http.DefaultClient, CacheRoot ""
// → lib.CacheRoot(), no progress callback (the editor's toast
// surface owns the user-facing feedback already). A future v3
// could pipe per-byte progress through ExampleFetchOptions.Progress;
// v2 ships with the indeterminate "Downloading…" status the menu
// publishes around the blocking Fetch call.
func (a *exampleSourceAdapter) Fetch(ctx context.Context, exampleID string) editor.OpenExampleResult {
	if a == nil || a.lib == nil {
		return editor.OpenExampleResult{Err: ErrExampleUnknown}
	}
	path, err := FetchExample(ctx, a.lib, exampleID, ExampleFetchOptions{})
	return editor.OpenExampleResult{LocalPath: path, Err: err}
}
