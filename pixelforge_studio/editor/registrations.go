// registrations.go is the central init() block for production
// pfcomponent and widget-registry registrations. Test-only
// registrations (Facing in inspector_test.go, etc) continue to live
// next to their tests; production types declared by the editor's
// schema land here so the registration call site is a single,
// auditable file. Idea #2 v1 U3 establishes this convention; idea #1
// and idea #5 plans extend it when their plans land.
package editor

import (
	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// init runs once at studio startup. Order:
//  1. Register the typed components (so pfcomponent.Get(name)
//     returns metadata as soon as the inspector dispatches).
//  2. Register the custom widget drawers (so WidgetCustom dispatch
//     finds a target).
//
// Drawer implementations live in their own files (tilepainter_widget.go,
// etc); this file owns the registration calls.
func init() {
	RegisterProductionComponents()
}

// RegisterProductionComponents is the idempotent registration site
// the init() block delegates to. Tests that call
// pfcomponent.ResetForTest invoke this helper from their cleanup
// hook so the production registry is restored for any subsequent
// test in the same process. The calls are idempotent — re-running
// is safe.
func RegisterProductionComponents() {
	pfcomponent.Register[pixelforge_project.TileAtlas]("TileAtlas")

	// Skip the widget registration if it's already in place. The
	// widget registry panics on duplicate names rather than
	// idempotent re-runs (RegisterWidget is stricter than
	// Register[T]), so check before calling.
	if _, ok := pfcomponent.LookupWidget("tilepainter"); !ok {
		pfcomponent.RegisterWidget("tilepainter", tilepainterDraw)
	}
}
