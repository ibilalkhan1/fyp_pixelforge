package buildpipeline

import (
	"runtime"
	"sync"
	"time"
)

// BuildOnSaveDebounce is the interval the daemon waits after a
// save before kicking off a build. 1.5s sits between VS Code's
// ~1s default and the brainstorm's 2-3s suggestion — short enough
// to feel responsive, long enough to coalesce rapid saves.
const BuildOnSaveDebounce = 1500 * time.Millisecond

// HostTarget returns the build target matching the studio's
// current host platform. The daemon dispatches builds only for
// this target so background work stays fast (no cross-compile).
func HostTarget() Target {
	switch runtime.GOOS {
	case "windows":
		return TargetWindows
	case "darwin":
		return TargetMacOS
	case "linux":
		return TargetLinux
	}
	return TargetLinux
}

// CanBuildOnHost reports whether t can be built from the studio's
// current host platform. Cross-OS native builds (Windows .exe from
// macOS, etc.) need Ebitengine's CGO toolchains for the foreign
// OS, which the studio doesn't ship. WASM + Source are pure-Go and
// always build; the host's matching native target also builds.
//
// Two-button Build UI (U4) renders only Host + WASM; the
// orchestrator's preflight uses this helper to reject any other
// target programmatically with a clear error.
func CanBuildOnHost(t Target) bool {
	switch t {
	case TargetWASM, TargetSource:
		return true
	}
	return t == HostTarget()
}

// BuildOnSaveDaemon coordinates the debounced background build
// the studio fires after every project save. The daemon holds the
// debounce timer, the latest status from the most recent build,
// and a per-save trigger function. Caller supplies the "is build
// allowed?" check + the kick-off callback so the daemon stays
// decoupled from settings + orchestrator wiring.
type BuildOnSaveDaemon struct {
	mu             sync.Mutex
	timer          *time.Timer
	debounce       time.Duration
	latestStatus   BuildStatus
	latestSeen     bool
	allowedCheck   func() bool
	triggerBuild   func()
	now            func() time.Time
	pendingTriggered bool
}

// NewBuildOnSaveDaemon returns a daemon bound to the supplied
// "is build allowed?" check + the build-trigger callback. The
// daemon uses BuildOnSaveDebounce by default; tests inject a
// shorter interval via SetDebounce.
func NewBuildOnSaveDaemon(allowedCheck func() bool, triggerBuild func()) *BuildOnSaveDaemon {
	if allowedCheck == nil {
		allowedCheck = func() bool { return true }
	}
	if triggerBuild == nil {
		triggerBuild = func() {}
	}
	return &BuildOnSaveDaemon{
		debounce:     BuildOnSaveDebounce,
		allowedCheck: allowedCheck,
		triggerBuild: triggerBuild,
		now:          time.Now,
	}
}

// SetDebounce overrides the debounce interval (tests use this to
// avoid sleeping the full 1.5s).
func (d *BuildOnSaveDaemon) SetDebounce(dur time.Duration) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.debounce = dur
}

// OnSave is invoked from FileMenu.SaveTo after the project's bytes
// have been written. Resets the debounce timer; the build fires
// when the timer elapses without another save interrupting.
func (d *BuildOnSaveDaemon) OnSave() {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.allowedCheck != nil && !d.allowedCheck() {
		// Build-on-save disabled in settings — clear any pending
		// timer + drop the trigger.
		if d.timer != nil {
			d.timer.Stop()
			d.timer = nil
		}
		return
	}

	if d.timer != nil {
		d.timer.Stop()
	}
	d.pendingTriggered = false
	d.timer = time.AfterFunc(d.debounce, func() {
		d.mu.Lock()
		alreadyTriggered := d.pendingTriggered
		d.pendingTriggered = true
		trigger := d.triggerBuild
		d.mu.Unlock()
		if alreadyTriggered || trigger == nil {
			return
		}
		trigger()
	})
}

// UpdateStatus records the most-recent BuildStatus from the
// orchestrator. The chrome status pill reads via LatestStatus.
func (d *BuildOnSaveDaemon) UpdateStatus(s BuildStatus) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.latestStatus = s
	d.latestSeen = true
}

// LatestStatus returns the most-recent BuildStatus + whether one
// has been recorded. Used by the chrome status pill (U8) to
// render the per-frame "Build ready · 2s ago" line.
func (d *BuildOnSaveDaemon) LatestStatus() (BuildStatus, bool) {
	if d == nil {
		return BuildStatus{}, false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.latestStatus, d.latestSeen
}

// HasPendingTimer reports whether a debounce timer is currently
// armed (a save fired recently, build hasn't kicked off yet).
// Test helper.
func (d *BuildOnSaveDaemon) HasPendingTimer() bool {
	if d == nil {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.timer != nil && !d.pendingTriggered
}

// Trigger forces the debounce timer to fire immediately. Test
// helper for assertions that don't want to sleep through the
// 1.5s default debounce.
func (d *BuildOnSaveDaemon) Trigger() {
	if d == nil {
		return
	}
	d.mu.Lock()
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	alreadyTriggered := d.pendingTriggered
	d.pendingTriggered = true
	trigger := d.triggerBuild
	d.mu.Unlock()
	if alreadyTriggered || trigger == nil {
		return
	}
	trigger()
}
