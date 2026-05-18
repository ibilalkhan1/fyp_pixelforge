// dockspace.go owns the editor's ImGui DockSpace. U3 of the ImGui
// migration plan replaces the M2 tab-strip workspace switcher with a
// dockspace that fills the main viewport — workspaces register
// themselves as ImGui windows and the user docks/floats them freely.
//
// On first run (no imgui.ini present), the dockspace lays out a default
// arrangement (Assets left, Inspector right, Scene centre, other
// workspaces tabbed into the centre node) using ImGui's DockBuilder.
// On subsequent runs ImGui restores the user's saved layout from
// imgui.ini, which U6 wires to the user-config dir.
package editor

import (
	"github.com/AllenDang/cimgui-go/imgui"
)

// dockspaceWindowID is the stable ID the editor passes to
// DockSpaceOverViewport. ImGui keys layout persistence against this in
// imgui.ini so the user's dock arrangement survives restarts.
const dockspaceWindowID = "PixelforgeStudioDockSpace"

// dockspaceState carries the editor's dockspace bookkeeping. The first
// frame's buildDockSpace seeds the default layout via DockBuilder;
// subsequent frames simply re-bind the dockspace ID and let ImGui
// restore from imgui.ini.
type dockspaceState struct {
	// initialised flips true after the first DockBuilder pass so the
	// editor doesn't keep rebuilding the default layout every frame.
	initialised bool
	// rootID is the dockspace's root node ID. Captured so SetActive-
	// WorkspaceByName can compute child node IDs if it ever needs to
	// re-dock a window programmatically.
	rootID imgui.ID
}

// buildDockSpace installs the editor's central DockSpace inside the
// main viewport. Called from buildChrome before any panel skeletons or
// workspace Render() calls — those windows then dock into this space
// either via the user's saved imgui.ini layout or, on first run, via
// the DockBuilder defaults seeded below.
func (e *Editor) buildDockSpace() {
	if e.imgui == nil || !e.imgui.live {
		return
	}
	if e.dockspace == nil {
		e.dockspace = &dockspaceState{}
	}
	// PassthruCentralNode keeps the central dock node transparent so
	// native widget content rendered into PanelRect("Scene") shows
	// through ImGui's background fill until U5 swaps it for an
	// imgui.Image-based scene.
	dockID := imgui.DockSpaceOverViewportV(
		imgui.IDStr(dockspaceWindowID),
		imgui.MainViewport(),
		imgui.DockNodeFlagsPassthruCentralNode,
		nil,
	)
	e.dockspace.rootID = dockID
	if !e.dockspace.initialised {
		e.applyDefaultDockLayout(dockID)
		e.dockspace.initialised = true
	}
}

// applyDefaultDockLayout seeds the dockspace on first run with a sane
// default arrangement: Assets on the left, Inspector on the right,
// Scene + the rest of the workspaces tabbed into the central node.
// Subsequent runs skip this entirely — ImGui restores from imgui.ini.
//
// The implementation uses ImGui's internal DockBuilder helpers, which
// is the supported way to bootstrap a default layout before any user
// arrangement exists.
func (e *Editor) applyDefaultDockLayout(rootID imgui.ID) {
	// Skip when ImGui has already restored a layout from imgui.ini. The
	// dockspace node will have child nodes in that case; DockBuilderGet-
	// Node returns non-nil for an existing layout.
	if node := imgui.InternalDockBuilderGetNode(rootID); node != nil && node.InternalIsSplitNode() {
		return
	}

	// Reset + recreate the node so the default layout is deterministic.
	imgui.InternalDockBuilderRemoveNode(rootID)
	imgui.InternalDockBuilderAddNodeV(rootID, imgui.DockNodeFlagsNone)

	// Size the root to the current viewport so the splits compute
	// correct widths.
	viewport := imgui.MainViewport()
	imgui.InternalDockBuilderSetNodeSize(rootID, viewport.WorkSize())
	imgui.InternalDockBuilderSetNodePos(rootID, viewport.WorkPos())

	// Split off left and right side panels, leaving the centre for the
	// scene + workspace tabs. Width ratios come from the editor.pforge
	// theme's DefaultLayout in U6; the hard-coded values here are the
	// fallback (200 px on a 1280-wide viewport ≈ 0.18).
	var leftID, rightID, centerLeft, centerID imgui.ID
	leftID = imgui.InternalDockBuilderSplitNode(rootID, imgui.DirLeft, defaultLeftPanelRatio, nil, &centerLeft)
	rightID = imgui.InternalDockBuilderSplitNode(centerLeft, imgui.DirRight, defaultRightPanelRatio, nil, &centerID)

	// Dock the editor's persistent panels into their default slots.
	imgui.InternalDockBuilderDockWindow(PanelAssets, leftID)
	imgui.InternalDockBuilderDockWindow(PanelInspector, rightID)

	// Dock every registered workspace into the central node. Multiple
	// workspaces docked into the same node render as tabs — exactly the
	// behaviour the M2 tab strip used to provide explicitly.
	for _, w := range e.workspaces {
		imgui.InternalDockBuilderDockWindow(w.DisplayName(), centerID)
	}

	imgui.InternalDockBuilderFinish(rootID)
}

// defaultLeftPanelRatio / defaultRightPanelRatio set the initial
// fraction of viewport width each side panel takes. U6 sources these
// from editor.pforge's DefaultLayout section.
const (
	defaultLeftPanelRatio  = 0.18
	defaultRightPanelRatio = 0.22
)
