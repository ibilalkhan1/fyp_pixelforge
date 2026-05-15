// Package runtime is the M5 visual-scripting engine. It walks a
// loaded *pixelforge_project.Project, compiles every non-empty
// BehaviorGraph into a pixelforge_routine.Routine (Steps) and a tree
// of pievent.Subscribe handlers (EventSheet), and tears everything
// down on Stop/Reload.
//
// The engine is owned by the studio (one instance per loaded
// project). Tests typically wire a minimal Project, build an Engine,
// Start it, exercise the loop manually, then Stop.
package runtime

import (
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
)

// Context is the runtime adapter passed to catalog builders. It
// bridges the catalog package (which knows nothing about Project)
// and the loaded project.
type Context struct {
	project *pixelforge_project.Project

	// samples and entities resolve project assets by name/ID.
	// Both maps are built lazily on first lookup.
	samples  map[string]*pixelforge_audio.Sample
	entities map[string]*pixelforge_project.Entity

	// valueRefs holds an in-memory store for set_value-driven
	// scratch values keyed by the user's path string. v1 of the
	// runtime doesn't yet poke component-field memory directly; the
	// scratch store lets event sheets and tween targets share state
	// across rule evaluations.
	valueRefs map[string]catalog.ValueRef
}

func newContext(p *pixelforge_project.Project) *Context {
	return &Context{
		project:   p,
		samples:   map[string]*pixelforge_audio.Sample{},
		entities:  map[string]*pixelforge_project.Entity{},
		valueRefs: map[string]catalog.ValueRef{},
	}
}

// Sample returns the loaded *pixelforge_audio.Sample for `name`, or
// nil if absent. v1 returns nil for now (asset loading is wired by
// the studio after compile).
func (c *Context) Sample(name string) any {
	if c == nil {
		return nil
	}
	if s, ok := c.samples[name]; ok {
		return s
	}
	return nil
}

// SetSample registers a loaded sample under name. Used by the studio
// after audio decode.
func (c *Context) SetSample(name string, sample *pixelforge_audio.Sample) {
	if c == nil {
		return
	}
	c.samples[name] = sample
}

// Entity returns the *pixelforge_project.Entity with the given ID
// from the project's first scene (v1 scene-runtime isolation
// deferred per the plan's Scope Boundaries).
func (c *Context) Entity(id string) any {
	if c == nil {
		return nil
	}
	if e, ok := c.entities[id]; ok {
		return e
	}
	if c.project == nil || len(c.project.Scenes) == 0 {
		return nil
	}
	for i := range c.project.Scenes[0].Entities {
		e := &c.project.Scenes[0].Entities[i]
		if e.ID == id {
			c.entities[id] = e
			return e
		}
	}
	return nil
}

// ValueRef returns the scratch value-ref for path, creating a fresh
// zero-valued cell on first access.
func (c *Context) ValueRef(path string) catalog.ValueRef {
	if c == nil {
		return nil
	}
	if ref, ok := c.valueRefs[path]; ok {
		return ref
	}
	cell := &scratchValueRef{}
	c.valueRefs[path] = cell
	return cell
}

// LookupTarget defers to the process-wide pievent registry.
func (c *Context) LookupTarget(name string) any {
	t := pievent.LookupTarget(name)
	if t == nil {
		return nil
	}
	return t
}

// Project returns the loaded project for the runtime. Read-only access.
func (c *Context) Project() *pixelforge_project.Project {
	if c == nil {
		return nil
	}
	return c.project
}

type scratchValueRef struct {
	v any
}

func (r *scratchValueRef) Get() any  { return r.v }
func (r *scratchValueRef) Set(v any) { r.v = v }
