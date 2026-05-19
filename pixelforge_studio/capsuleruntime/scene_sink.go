package capsuleruntime

import (
	"log"

	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// sceneSink is the concrete SceneController plan-009 U7 lands. It
// mutates rt.CurrentScene on scene/change, rolls the live scene back
// to its authored entity list on scene/restart, and sets a tick
// countdown on scene/wait. After a successful scene change the sink
// republishes a "scene/changed" event on verbs.bus so any chained
// handler (audio fade-in, save-on-transition, etc.) can subscribe
// without coupling to the sink directly.
//
// All methods are safe to call on a nil rt or a runtime with no
// scenes; the verb-recipe surface contract is "never crash a shipped
// game", and unknown scene names log a warning and no-op rather than
// panic.
type sceneSink struct {
	rt *Runtime
}

// newSceneSink constructs the production sceneSink. Exposed as a
// constructor so fillDefaults stays the only place the rt-pointer
// wiring happens.
func newSceneSink(rt *Runtime) *sceneSink {
	return &sceneSink{rt: rt}
}

// Change swaps rt.CurrentScene to the named scene from
// rt.Project.Scenes. Unknown names log + no-op (graceful degrade).
// On success the sink republishes "scene/changed" so chained
// handlers fire after the swap lands.
func (s *sceneSink) Change(sceneID string) {
	if s == nil || s.rt == nil || s.rt.Project == nil {
		return
	}
	if sceneID == "" {
		debugDrop("scene/change", "empty scene_id", nil)
		return
	}
	for i := range s.rt.Project.Scenes {
		if s.rt.Project.Scenes[i].ID == sceneID {
			s.rt.CurrentScene = &s.rt.Project.Scenes[i]
			piloop.VerbsBus().Publish(&piloop.VerbEvent{
				Topic: "scene/changed",
				Args:  map[string]any{"scene_id": sceneID},
			})
			return
		}
	}
	log.Printf("[capsuleruntime] scene/change: unknown scene %q", sceneID)
}

// Restart rolls the current scene's entity list back to the
// authored snapshot captured at Boot. No-op when rt or CurrentScene
// is nil. Mutates the live Scene in place so any consumer holding a
// pointer to rt.CurrentScene observes the rollback without
// re-resolving the pointer.
func (s *sceneSink) Restart() {
	if s == nil || s.rt == nil || s.rt.CurrentScene == nil {
		return
	}
	snap, ok := s.rt.InitialEntities[s.rt.CurrentScene.ID]
	if !ok {
		return
	}
	restored := make([]pixelforge_project.Entity, len(snap))
	copy(restored, snap)
	s.rt.CurrentScene.Entities = restored
	// Clear all visual effects on restart — they were tied to
	// entities that are being replaced.
	s.rt.VisualEffects = map[string]*VisualEffect{}
}

// Wait sets the scene-wait countdown. Engine drivers decrement
// rt.SceneWaitTicks once per game tick; sinks consulting the field
// can skip downstream dispatch while a wait is active. Negative
// values clamp to 0 so a malformed recipe can't lock the runtime in
// a perpetual wait.
func (s *sceneSink) Wait(ticks int) {
	if s == nil || s.rt == nil {
		return
	}
	if ticks < 0 {
		ticks = 0
	}
	s.rt.SceneWaitTicks = ticks
}
