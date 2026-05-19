// Package build is idea #7 v1 U7's studio Build workspace.
// Plan-008 U4 collapses the original 5-target checkbox matrix into
// two large buttons — Host (the native OS the studio is running on)
// and WASM (single-file self-contained .html). Per-target status
// pills + Open output / Show error controls share the same path the
// build-on-save daemon mirrors through.
package build

import (
	"context"
	"sync"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/buildpipeline"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
)

// uiTargets is the ordered list the Render path walks — Host first,
// WASM second. Other targets (Source, foreign-OS native) remain
// callable via direct buildpipeline.Build API for tests, but are
// not surfaced in the UI.
var uiTargets = []buildpipeline.Target{
	buildpipeline.HostTarget(),
	buildpipeline.TargetWASM,
}

// Workspace implements editor.Workspace.
type Workspace struct {
	editor *editor.Editor

	mu       sync.Mutex
	statuses map[buildpipeline.Target]buildpipeline.BuildStatus
	cancel   context.CancelFunc
}

// NewWorkspace constructs the Build workspace bound to editor.
func NewWorkspace(e *editor.Editor) *Workspace {
	return &Workspace{
		editor:   e,
		statuses: map[buildpipeline.Target]buildpipeline.BuildStatus{},
	}
}

func (w *Workspace) Name() string        { return "build" }
func (w *Workspace) DisplayName() string { return "Build" }

// RegisterWith installs the workspace on editor.
func RegisterWith(e *editor.Editor) *Workspace {
	w := NewWorkspace(e)
	e.RegisterWorkspace(w)
	return w
}

// Status returns the most-recent BuildStatus for the supplied
// target (zero value when no build has run yet).
func (w *Workspace) Status(t buildpipeline.Target) buildpipeline.BuildStatus {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.statuses[t]
}

// AllStatuses returns a snapshot of every recorded status, sorted
// by canonical target order. Test helper + telemetry.
func (w *Workspace) AllStatuses() []buildpipeline.BuildStatus {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]buildpipeline.BuildStatus, 0, len(w.statuses))
	for _, t := range buildpipeline.AllTargets {
		if s, ok := w.statuses[t]; ok {
			out = append(out, s)
		}
	}
	return out
}

// BuildHost kicks off the orchestrator for the host platform's
// native target. Returns the same done-channel shape StartBuild
// used so tests can block on completion.
func (w *Workspace) BuildHost() <-chan struct{} {
	return w.runBuild([]buildpipeline.Target{buildpipeline.HostTarget()})
}

// BuildWASM kicks off the orchestrator for the WASM target.
func (w *Workspace) BuildWASM() <-chan struct{} {
	return w.runBuild([]buildpipeline.Target{buildpipeline.TargetWASM})
}

// runBuild is the shared dispatch path the two-button UI's
// BuildHost / BuildWASM funnel through. Cancels any in-flight
// build first so a fresh click supersedes the prior run.
func (w *Workspace) runBuild(targets []buildpipeline.Target) <-chan struct{} {
	done := make(chan struct{})
	if len(targets) == 0 {
		close(done)
		return done
	}
	w.mu.Lock()
	if w.cancel != nil {
		w.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.mu.Unlock()

	req := buildpipeline.BuildRequest{
		Project:     w.editor.Project(),
		ProjectPath: w.editor.CurrentProjectPath(),
	}
	statusCh := buildpipeline.Build(ctx, req, targets)
	go func() {
		defer close(done)
		for s := range statusCh {
			w.RecordStatus(s)
		}
	}()
	return done
}

// RecordStatus updates the per-target status map. Exposed for
// tests + the build-on-save daemon's mirror path.
func (w *Workspace) RecordStatus(s buildpipeline.BuildStatus) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.statuses[s.Target] = s
}

// CancelBuild aborts any in-flight build. Safe to call when no
// build is running.
func (w *Workspace) CancelBuild() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
}

// Render emits the workspace UI inside the current ImGui frame.
// Two large buttons replace the prior 5-checkbox matrix; per-target
// status pills sit beside each button so designers see queued ->
// generating -> compiling -> packaging -> done progress without
// switching panes.
func (w *Workspace) Render(e *editor.Editor) {
	if e == nil {
		return
	}
	if !imgui.Begin(w.DisplayName()) {
		imgui.End()
		return
	}
	defer imgui.End()
	imgui.TextUnformatted("Build target")
	imgui.Separator()
	for _, t := range uiTargets {
		label := buttonLabelFor(t)
		if imgui.Button(label) {
			switch t {
			case buildpipeline.TargetWASM:
				_ = w.BuildWASM()
			default:
				_ = w.BuildHost()
			}
		}
		imgui.SameLine()
		imgui.TextDisabled(statusLabel(w.Status(t)))
	}
	imgui.Separator()
	if imgui.Button("Cancel") {
		w.CancelBuild()
	}
}

// buttonLabelFor returns the user-facing label for the supplied
// target. Host gets a labelled-by-OS button so designers see
// "Build for Windows" / "Build for macOS" / "Build for Linux" —
// the cross-compile rejection error message uses the same
// vocabulary.
func buttonLabelFor(t buildpipeline.Target) string {
	if t == buildpipeline.TargetWASM {
		return "Build WASM"
	}
	return "Build " + t.DisplayName()
}

// statusLabel composes the per-target status pill text.
func statusLabel(s buildpipeline.BuildStatus) string {
	switch s.Phase {
	case buildpipeline.PhaseQueued:
		return "queued"
	case buildpipeline.PhaseGenerating:
		return "generating..."
	case buildpipeline.PhaseCompiling:
		return "compiling..."
	case buildpipeline.PhasePackaging:
		return "packaging..."
	case buildpipeline.PhaseDone:
		return "done"
	case buildpipeline.PhaseFailed:
		if s.Err != nil {
			return "failed: " + s.Err.Error()
		}
		return "failed"
	}
	return "—"
}
