// sprite_thumbnail.go owns idea #6 v1 U9's tiny preview widget:
// given a sprite name, render a small swatch (the sprite's first
// frame or a placeholder). The Items workspace uses it in the
// Icon column.
package widgets

// SpriteThumbnailLookup is the callback the widget uses to resolve
// a sprite name to a renderable identifier (or an empty value when
// the sprite isn't in the project). Concrete renderers (the
// editor's chrome compositor) supply this so the widget doesn't
// have to know about pixelforge_project or pixelforge.Image.
type SpriteThumbnailLookup func(name string) (any, bool)

// SpriteThumbnailWidget carries the lookup callback + the cached
// last-rendered sprite identifier. State-only; the actual draw is
// driven by the editor's chrome compositor when the widget is
// surfaced inside a panel.
type SpriteThumbnailWidget struct {
	lookup     SpriteThumbnailLookup
	SpriteName string

	lastResolved any
	lastOk       bool
}

// NewSpriteThumbnail constructs a widget bound to lookup. The
// caller sets SpriteName per-row; Resolve refreshes the cached
// identifier.
func NewSpriteThumbnail(lookup SpriteThumbnailLookup) *SpriteThumbnailWidget {
	return &SpriteThumbnailWidget{lookup: lookup}
}

// Resolve refreshes the cached identifier from the bound lookup.
// Returns (identifier, ok). The identifier is opaque to this
// widget — concrete renderers cast it to their texture type.
func (w *SpriteThumbnailWidget) Resolve() (any, bool) {
	if w == nil || w.lookup == nil {
		return nil, false
	}
	resolved, ok := w.lookup(w.SpriteName)
	w.lastResolved = resolved
	w.lastOk = ok
	return resolved, ok
}

// Cached returns the most-recent Resolve result without re-fetching.
func (w *SpriteThumbnailWidget) Cached() (any, bool) {
	if w == nil {
		return nil, false
	}
	return w.lastResolved, w.lastOk
}
