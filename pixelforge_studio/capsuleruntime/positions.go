package capsuleruntime

import (
	ph "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_physics"
	project "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// SyncPositions reconciles authored movement actions and physics motion once per tick.
func (rt *Runtime) SyncPositions() {
	if rt == nil || rt.CurrentScene == nil {
		return
	}
	if rt.lastPositions == nil {
		rt.lastPositions = map[string]project.EntityPosition{}
	}
	for i := range rt.CurrentScene.Entities {
		entity := &rt.CurrentScene.Entities[i]
		body := rt.Bodies[entity.ID]
		if body == nil {
			continue
		}
		previous, known := rt.lastPositions[entity.ID]
		if known && (entity.Position.X != previous.X || entity.Position.Y != previous.Y) {
			body.Position.X = ph.FixedFromFloat(entity.Position.X)
			body.Position.Y = ph.FixedFromFloat(entity.Position.Y)
			body.Sync()
		} else {
			entity.Position.X = body.Position.X.Float()
			entity.Position.Y = body.Position.Y.Float()
		}
		rt.lastPositions[entity.ID] = entity.Position
	}
}
