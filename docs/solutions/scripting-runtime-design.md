# Scripting Runtime Design (M5)

## Context

The `.pforge` schema reserved `BehaviorGraph`, `StepNode`,
`EventSheetRule`, `Condition`, and `Action` at M1, but no runtime
walked those structures. From M1 through M4 every loaded project
shipped with `Project.Behaviors` populated but inert — the "no-code
editor" pitch was half-built: users could author content, not
behaviour. M5 fills the gap by giving the studio a single Engine
per-project that compiles the schema into running engine primitives.

## What We Did

- **One `Engine` per project, owned by the studio.** `runtime.New(p)`
  binds an engine to the project; `Start()` compiles every non-empty
  `BehaviorGraph` and schedules the resulting `pixelforge_routine.Routine`s
  on `piloop.EventUpdate`; `Stop()` walks every instance and
  unsubscribes / stops cleanly. `Reload(graphName)` swaps a single
  behaviour in place after an editor edit.
- **Kind catalogs mirror `pfcomponent.Register`.** A
  `catalog.RegisterStep("Wait", builder)` call in a package-init block
  hands the runtime a builder that turns `StepNode.Args` into a
  `pixelforge_routine.Step`. The same pattern services `Condition`
  (predicates) and `Action` (effects) Kinds. Built-ins: seven Step
  Kinds, five Conditions, five Actions.
- **Non-generic `Inspectable` sidecar for `pievent.Target`
  enumeration.** Go generics make uniform iteration over `Target[T]`
  impossible (each instance has a different `T`). We added
  `pievent.RegisterTarget(name string, target Inspectable)` plus
  `EnumerateTargets() []TargetInfo`. Engine packages (`loop`, `mouse`,
  `key`, `pad`, `debug`) register themselves at `init()`.
- **Reflection-driven `SubscribeAny`.** Event-sheet rules can subscribe
  on any `Target[T]` regardless of `T` by using `reflect.Value.MethodByName`
  to call `SubscribeAll` and `Unsubscribe`. The wrapper boxes the
  payload as `any` so catalog predicates can pattern-match generically.
- **Input-log replay v1 for recorded-demo synthesis.** A stateless
  `SynthesiseFromInputLog([]*capture.Frame, start, end) BehaviorGraph`
  walks the recorder's frame buffer and emits one `Wait(n)` between
  distinct `TickNumber`s and one `Publish(target, value)` per logged
  input. Deterministic, trivially testable, no state-diff heuristics.
- **`text/template`-driven View-as-Go emitter.** One template renders
  every `BehaviorGraph` to a readable Go function. One-way (no parser);
  the output is for users escaping to code, not for round-trip
  authoring. ~100 lines of template, no custom code generator.
- **Engine-owned breakpoint store + Step/Continue semantics.** The
  runtime gates step Resume and rule evaluation on a `breakpoints
  map[string]bool` keyed by event path. The debugger UI sets entries;
  `Engine.Step()` lets one event through and re-pauses;
  `Engine.Continue()` clears pause state until the next breakpoint.

## Why It Works

- **Lifecycle pinned to project.** The studio's
  `ProjectListener.OnProjectChanged` hook fires the scripting workspace
  on every project replacement; stop-then-start is deterministic.
- **Catalogs are testable and extensible.** Builders are pure functions
  over `(args, ctx)`. Tests don't need a project, an engine, or a loop
  — they call the builder directly. Custom user Kinds plug in via the
  same `Register` seam (deferred follow-up — the seam exists today).
- **Reflection cost is bounded.** Subscribe-time reflection runs once
  per rule per compile, not per event. The hot path (publish →
  predicate → effect) is plain Go.
- **Emitter has zero parser surface.** Because View-as-Go is read-only,
  the template can take any liberties for clarity; we never have to
  parse the output back. If a future plan wants round-trip Go, it owns
  the grammar.

## Alternatives Considered

- **Per-scene engine.** Multi-scene runtime switching is interesting
  but adds lifecycle complexity (which scene's behaviours run after a
  scene swap?). Deferred — see plan Scope Boundaries.
- **Bidirectional View-as-Go.** A constrained-Go parser that
  round-trips edits back into `BehaviorGraph` is the largest cut item.
  Deferred to its own plan when users have authored enough graphs to
  inform the grammar.
- **State-diff recorded-demo synthesis.** Heuristics that diff entity
  state across frames and propose declarative rules. Requires capturing
  entity-state snapshots (which the recorder doesn't ship today) plus
  the synthesis algorithm. Deferred.
- **Registry-via-codegen instead of reflection.** A code-generated
  `SubscribeAny` per known `Target[T]` would dodge reflection at the
  cost of a build step and an "enumerate all target types" maintenance
  burden. Reflection wins for v1 — bounded use, single seam.

## When to Apply This Pattern

Any subsystem that needs to compile declarative graph-shaped data into
running engine objects with deterministic lifecycle:

- Schema reservation at the earliest milestone (M1 did this).
- One owner per top-level entity (one Engine per project).
- Kind catalogs with `Register(name, builder)` for extensibility.
- `TrackingTarget` (or its non-generic equivalent) for clean teardown.
- A non-generic inspection interface when generic types block uniform
  iteration.

## References

- Plan: [docs/plans/2026-05-16-001-feat-pixelforge-visual-scripting-and-event-sheets-plan.md](../plans/2026-05-16-001-feat-pixelforge-visual-scripting-and-event-sheets-plan.md)
- Engine: `pixelforge_studio/scripting/runtime/engine.go`
- Registry: `pixelforge_event/registry.go`
- Catalog: `pixelforge_studio/scripting/catalog/`
- Workspace integration: `pixelforge_studio/scripting/workspace.go`
- Related learnings:
  - [editor-pforge-schema-shape.md](editor-pforge-schema-shape.md) — additive-on-load + sanitize idiom adopted by the catalog's unknown-Kind handling.
  - [ring-buffer-snapshot-store.md](ring-buffer-snapshot-store.md) — the recorder pattern the debugger's time-travel scrub composes against.
