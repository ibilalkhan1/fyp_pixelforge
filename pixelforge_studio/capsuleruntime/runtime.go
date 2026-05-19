package capsuleruntime

import (
	"fmt"
	"io/fs"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"
)

// Options tunes Boot's behaviour. Zero-value Options is the
// production posture: load everything, install subscribers, and
// hand the populated Runtime back to the caller. The capsule's
// CapsuleRun feeds its CapsuleOpts.DataOverride (used by the
// studio preview to inject unsaved edits) through here unchanged.
type Options struct {
	// SkipSubscribers leaves the verb-recipe event-bus subscribers
	// un-installed. Tests that exercise loader semantics in
	// isolation flip this true to avoid touching the global
	// pievent registry.
	SkipSubscribers bool

	// SaveBackendOverride substitutes a custom Backend (typically
	// an in-memory fake) for the file-system or localStorage one
	// the production path selects via build tags. Tests use this
	// to keep capsule-runtime Boot hermetic.
	SaveBackendOverride pisave.Backend

	// SinksOverride substitutes one or more subsystem sinks with
	// test-supplied fakes. Any field left nil receives the
	// production default via Sinks.fillDefaults. Tests that
	// exercise a single handler typically set just the field they
	// care about.
	SinksOverride Sinks
}

// Runtime is the per-capsule wiring Boot returns. Exposes the
// loaded project + the save service the subscriber layer dispatches
// save/load/delete verbs through. The capsule's CapsuleRun keeps a
// reference so the engine packages can reach Runtime state through
// well-known accessors at frame time.
type Runtime struct {
	Project *pixelforge_project.Project
	Save    *pisave.Service
	Sinks   Sinks
}

// Boot is the canonical capsule entry point. Loads sprites + audio
// + dialogue from the embedded assets FS, populates the capsule-
// side scene + item registries, constructs the save service for the
// host platform, then installs the verb-recipe event-bus
// subscribers (U2). Returns the populated *Runtime so the caller
// (the codegen-emitted CapsuleRun) can keep a reference.
//
// On loader failure Boot returns the error without installing
// subscribers — a partially-loaded capsule wouldn't run any
// authored game correctly, so failing fast is preferable to
// shipping a half-functional process.
//
// Boot is idempotent against the registries: re-running with the
// same project + assets produces equivalent post-state because each
// Register* call replaces the prior entry. The save service is
// re-constructed on every call.
func Boot(p *pixelforge_project.Project, assets fs.FS, opts Options) (*Runtime, error) {
	if p == nil {
		return nil, fmt.Errorf("capsuleruntime: project is nil")
	}
	if err := LoadSprites(p, assets); err != nil {
		return nil, err
	}
	if err := LoadAudio(p, assets); err != nil {
		return nil, err
	}
	if err := LoadDialogue(p); err != nil {
		return nil, err
	}
	LoadScenes(p)
	LoadItems(p)

	save, err := newSaveService(p, opts.SaveBackendOverride)
	if err != nil {
		return nil, err
	}
	rt := &Runtime{Project: p, Save: save, Sinks: opts.SinksOverride}

	if !opts.SkipSubscribers {
		InstallSubscribers(rt)
	}
	return rt, nil
}

// newSaveService picks the right backend for the host platform —
// production callers leave opts.SaveBackendOverride nil and the
// build-tagged backend constructor (backend_native.go on desktop,
// backend_js.go on WASM) supplies the real backend. Tests pass an
// in-memory backend through Options to stay hermetic.
//
// gameTitle derives from the project's SaveConfig (designer-set
// override) falling back to Project.Name so two different games on
// the same host don't collide in the save directory.
func newSaveService(p *pixelforge_project.Project, override pisave.Backend) (*pisave.Service, error) {
	if override != nil {
		return pisave.NewService(override), nil
	}
	title := p.SaveConfig.GameTitle
	if title == "" {
		title = p.Name
	}
	backend, err := defaultSaveBackend(title)
	if err != nil {
		return nil, fmt.Errorf("capsuleruntime: save backend: %w", err)
	}
	return pisave.NewService(backend), nil
}
