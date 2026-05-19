package buildpipeline

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
)

// runtimeGOOS is the GOOS literal used in cross-compile error
// messages. Indirected through a var so tests on a single host can
// override it to simulate the rejection path for foreign targets.
var runtimeGOOS = func() string { return runtime.GOOS }

// Phase names where a per-target build currently is. Walks
// Queued → Generating → Compiling → Packaging → Done / Failed.
type Phase int

const (
	PhaseQueued Phase = iota
	PhaseGenerating
	PhaseCompiling
	PhasePackaging
	PhaseDone
	PhaseFailed
)

// String returns the user-facing label for the phase.
func (p Phase) String() string {
	switch p {
	case PhaseQueued:
		return "queued"
	case PhaseGenerating:
		return "generating"
	case PhaseCompiling:
		return "compiling"
	case PhasePackaging:
		return "packaging"
	case PhaseDone:
		return "done"
	case PhaseFailed:
		return "failed"
	}
	return "unknown"
}

// BuildRequest bundles the per-build inputs the orchestrator
// hands to each per-target builder.
type BuildRequest struct {
	Project     *pixelforge_project.Project
	ProjectPath string
	OutputDir   string

	// EnginePath overrides modulepath.Detect's auto-discovery for
	// the engine source location. Required for long-tag builds
	// when the studio binary's path can't be walked up to a
	// containing engine repo (e.g., installed studio at
	// /opt/pixelforge/bin/, or test binaries living under /tmp).
	// Empty falls back to codegen's auto-detect (works for
	// in-tree development builds).
	EnginePath string

	// EngineVersion pairs with EnginePath when the strategy is a
	// require + replace combination. Empty falls back to
	// auto-detect.
	EngineVersion string

	// EngineStrategy lets callers override the modulepath strategy
	// the generator uses (vendor / replace / published-version).
	// Empty (zero value = StrategyVendor) keeps the current
	// auto-detect default. Long-tag tests pin StrategyDevReplace
	// so `go mod tidy` resolves the engine via a `replace`
	// directive instead of trying to fetch v0.0.0 from the
	// module proxy.
	EngineStrategy ModuleStrategy

	// Credits is the CC-BY attribution data the build pipeline
	// embeds into the capsule's auto-injected Credits screen
	// (plan-008 U10). Callers that have a populated
	// assetlibrary.Library pre-fill this via
	// assetlibrary.AssembleCredits(p, lib); empty produces a no-op
	// credits literal in the generated capsule.
	Credits []capsuleruntime.CreditEntry

	// PlayerBinaryPath, when non-empty, points the host build at a
	// pre-built pixelforge-player binary on disk. Skips the entire
	// player-binary discovery chain (cache → embed → developer go
	// build) and uses this path verbatim. Used by long-tag tests to
	// pin a fixture binary + by advanced callers that ship their
	// own player build.
	//
	// Discovery chain when this is empty:
	//   1. <userCacheDir>/pixelforge/player-cache/<...> (with SHA-256
	//      sidecar verification)
	//   2. playerbins.PlayerBinaryFor(GOOS, GOARCH) — the no-Go user
	//      path, embedded via go:embed at studio build time
	//   3. `go build -tags=long -o $TMP ./cmd/pixelforge-player` —
	//      the developer fallback (slow, requires a Go toolchain)
	PlayerBinaryPath string

	// WasmBinaryPath mirrors PlayerBinaryPath for the WASM target.
	// Non-empty short-circuits the discovery chain for
	// GOOS=js/GOARCH=wasm; empty triggers the same cache → embed →
	// developer-fallback walk.
	WasmBinaryPath string

	// ForceLargeWASM, when true, allows the WASM build to proceed
	// even when the bundled .html exceeds the
	// WASMErrorThresholdMB hard limit. Designers shipping an
	// intentionally chunky build (e.g., a debug build with a
	// massive sprite atlas) flip this from the UI's "build
	// anyway" affordance. Default false: oversized builds fail
	// with ErrWASMTooLarge so the cost is visible upstream.
	ForceLargeWASM bool
}

// ModuleStrategy is the build-pipeline-side mirror of the
// modulepath package's Strategy enum. Kept here so callers
// outside the codegen package can name a strategy without
// importing modulepath transitively. Values must match
// modulepath.Strategy 1:1.
type ModuleStrategy int

const (
	// ModuleStrategyDefault keeps the codegen package's
	// auto-detect choice (vendor unless a git checkout supplies a
	// dev replace).
	ModuleStrategyDefault ModuleStrategy = iota
	// ModuleStrategyPublishedVersion uses a versioned require.
	ModuleStrategyPublishedVersion
	// ModuleStrategyDevReplace emits a `replace` directive pointing
	// at the local engine checkout.
	ModuleStrategyDevReplace
)

// BuildStatus is one event on the orchestrator's status channel.
// Phase / Err / OutputPath update over the build's lifetime.
type BuildStatus struct {
	Target     Target
	Phase      Phase
	OutputPath string
	Err        error
	BuiltAt    time.Time

	// SizeReport is populated on the terminal PhaseDone event of
	// a WASM build with the raw + gzip byte counts of the .html
	// artifact. Nil for non-WASM builds and for non-terminal
	// phases. The studio Build workspace reads this to surface
	// "(18.0MB, gzip 6.4MB)" in the success toast plus the
	// warn/error indicator.
	SizeReport *WASMSizeReport
}

// Builder is the contract each per-target file implements.
// The orchestrator calls Build with the request + a context that
// fires when the caller cancels the overall build.
type Builder interface {
	Build(ctx context.Context, req BuildRequest, emit func(BuildStatus)) error
}

// builderRegistry maps Target → Builder. Populated by per-target
// files' init() functions so the orchestrator stays decoupled
// from the concrete implementations.
var (
	builderMu       sync.RWMutex
	builderRegistry = map[Target]Builder{}
)

// RegisterBuilder installs a per-target builder. Panics on
// duplicate registration to surface ordering bugs immediately.
func RegisterBuilder(t Target, b Builder) {
	builderMu.Lock()
	defer builderMu.Unlock()
	if _, exists := builderRegistry[t]; exists {
		panic(fmt.Sprintf("buildpipeline: target %s already has a registered builder", t))
	}
	builderRegistry[t] = b
}

// LookupBuilder returns the registered builder + true; or nil +
// false when the target has no registration. Targets without a
// builder fail the orchestrator's preflight check with a clear
// error rather than panicking mid-build.
func LookupBuilder(t Target) (Builder, bool) {
	builderMu.RLock()
	defer builderMu.RUnlock()
	b, ok := builderRegistry[t]
	return b, ok
}

// ResetBuildersForTest clears the registry; test-only.
func ResetBuildersForTest() {
	builderMu.Lock()
	defer builderMu.Unlock()
	builderRegistry = map[Target]Builder{}
}

// Build dispatches one goroutine per target. Status events stream
// to the returned channel; the channel closes when every target
// reaches a terminal phase (Done or Failed).
//
// Cancellation: callers wrap their own context around the inner
// build; passing a cancelled ctx tears down in-flight builds via
// the per-target Builder's context-aware exec calls.
func Build(ctx context.Context, req BuildRequest, targets []Target) <-chan BuildStatus {
	statusCh := make(chan BuildStatus, 4*len(targets))
	if len(targets) == 0 {
		close(statusCh)
		return statusCh
	}
	if req.OutputDir == "" {
		req.OutputDir = filepath.Join(filepath.Dir(req.ProjectPath), "exports")
	}

	// Preflight: split callable targets from cross-OS rejects. A
	// rejected target emits PhaseFailed{Err: CrossCompileNotSupportedError}
	// immediately so the caller sees a clean terminal phase on the
	// status channel without spawning a goroutine that can't do
	// anything useful. Plan U4 codifies this as a single seam so the
	// two-button UI + API callers share one rejection path.
	hostOS := runtimeGOOS()
	var callable []Target
	for _, t := range targets {
		if !CanBuildOnHost(t) {
			statusCh <- BuildStatus{
				Target:  t,
				Phase:   PhaseFailed,
				Err:     &CrossCompileNotSupportedError{Target: t, HostOS: hostOS},
				BuiltAt: time.Now(),
			}
			continue
		}
		callable = append(callable, t)
	}
	if len(callable) == 0 {
		close(statusCh)
		return statusCh
	}

	var wg sync.WaitGroup
	for _, target := range callable {
		wg.Add(1)
		go func(t Target) {
			defer wg.Done()
			runOne(ctx, t, req, statusCh)
		}(target)
	}
	go func() {
		wg.Wait()
		close(statusCh)
	}()
	return statusCh
}

// runOne dispatches one target to its registered builder.
// Surfaces Queued + a terminal Done/Failed event around the
// builder's own phase emissions. Builders that emit their own
// Done/Failed are not double-stamped — the orchestrator's
// trailing Done only fires when the builder didn't reach a
// terminal phase on its own.
func runOne(ctx context.Context, target Target, req BuildRequest, ch chan<- BuildStatus) {
	emittedTerminal := false
	emit := func(s BuildStatus) {
		s.Target = target
		// Best-effort send on the buffered channel; the buffer
		// scales 4× target count so orchestrator goroutines never
		// block on a typical drain rate.
		ch <- s
		if s.Phase == PhaseDone || s.Phase == PhaseFailed {
			emittedTerminal = true
		}
	}
	emit(BuildStatus{Phase: PhaseQueued, BuiltAt: time.Now()})

	builder, ok := LookupBuilder(target)
	if !ok {
		emit(BuildStatus{
			Phase:   PhaseFailed,
			Err:     fmt.Errorf("buildpipeline: no builder registered for %s", target),
			BuiltAt: time.Now(),
		})
		return
	}

	if ctx.Err() != nil {
		emit(BuildStatus{Phase: PhaseFailed, Err: ctx.Err(), BuiltAt: time.Now()})
		return
	}

	if err := builder.Build(ctx, req, emit); err != nil {
		if !emittedTerminal {
			emit(BuildStatus{Phase: PhaseFailed, Err: err, BuiltAt: time.Now()})
		}
		return
	}
	if !emittedTerminal {
		emit(BuildStatus{Phase: PhaseDone, BuiltAt: time.Now()})
	}
}

// ErrBuildCancelled signals the caller cancelled mid-build.
var ErrBuildCancelled = errors.New("buildpipeline: build cancelled")

// ErrWASMTooLarge fires when the bundled WASM .html exceeds the
// WASMErrorThresholdMB (30MB) hard cap and the request did not
// set ForceLargeWASM. The error carries the size report so
// callers can show the exact byte count in the failure message.
//
// errors.Is comparisons against ErrWASMTooLarge succeed via the
// Is method below — call sites can match on the kind without
// parsing the message.
type WASMTooLargeError struct {
	Report WASMSizeReport
}

func (e *WASMTooLargeError) Error() string {
	return fmt.Sprintf("buildpipeline: WASM .html size %s exceeds %dMB hard limit (pass ForceLargeWASM to override)",
		e.Report.Format(), e.Report.ErrorThresholdMB)
}

// Is reports whether target matches ErrWASMTooLarge so callers
// using errors.Is can match the kind without parsing the message.
func (e *WASMTooLargeError) Is(target error) bool {
	return target == ErrWASMTooLarge
}

// ErrWASMTooLarge is the sentinel callers errors.Is against.
var ErrWASMTooLarge = errors.New("buildpipeline: WASM bundle exceeds hard size limit")

// CrossCompileNotSupportedError is returned for any target the
// host can't currently build. Carries both the requested target
// and the host's actual OS so the error message stays diagnosable
// even after it's wrapped further up the stack.
//
// errors.Is comparisons against the sentinel ErrCrossCompileNotSupported
// also succeed via the Is method below — call sites can match on the
// kind without parsing the message.
type CrossCompileNotSupportedError struct {
	Target Target
	HostOS string
}

func (e *CrossCompileNotSupportedError) Error() string {
	return fmt.Sprintf("buildpipeline: target %s is not buildable from host %s (cross-compile not supported)",
		e.Target.String(), e.HostOS)
}

// Is reports whether target matches ErrCrossCompileNotSupported so
// callers using errors.Is can match the kind without parsing the
// message.
func (e *CrossCompileNotSupportedError) Is(target error) bool {
	return target == ErrCrossCompileNotSupported
}

// ErrCrossCompileNotSupported is the sentinel callers errors.Is
// against. The concrete CrossCompileNotSupportedError instances
// returned by the preflight name the specific (target, host) pair.
var ErrCrossCompileNotSupported = errors.New("buildpipeline: cross-compile not supported")
