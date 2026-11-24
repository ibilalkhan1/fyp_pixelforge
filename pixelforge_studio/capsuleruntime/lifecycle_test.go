//nolint:testpackage // Verifies private subscription ownership and teardown.
package capsuleruntime

import (
	pf "github.com/ibilalkhan1/fyp_pixelforge"
	pikey "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	project "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"testing"
)

func TestAuthoredMovementStartsStopsAndSynchronizes(t *testing.T) {
	p := project.NewProject("play-test")
	p.Scenes = []project.Scene{{ID: "main", Entities: []project.Entity{
		{ID: "hero", Position: project.EntityPosition{X: 16, Y: 16}},
	}}}
	p.Behaviors = []project.BehaviorGraph{{Name: "move", EventSheet: []project.EventSheetRule{{
		Conditions: []project.Condition{
			{Kind: "event_fired", Args: map[string]any{"target": "loop.main", "event": "update"}},
			{Kind: "key_held", Args: map[string]any{"key": "Right"}},
		},
		Actions: []project.Action{
			{Kind: "move_entity", Args: map[string]any{"entity": "hero", "dx": float64(2), "dy": float64(0)}},
		},
	}}}}
	rt, err := Boot(p, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()
	rt.Start()
	pf.Frame = 100
	pikey.Target().Publish(pikey.Event{Type: pikey.EventDown, Key: pikey.Right})
	piloop.Target().Publish(piloop.EventUpdate)
	rt.SyncPositions()
	if p.Scenes[0].Entities[0].Position.X != 18 || rt.Bodies["hero"].Position.X.Int() != 18 {
		t.Fatal("authored move did not update the entity and rendered body")
	}
	rt.Close()
	piloop.Target().Publish(piloop.EventUpdate)
	if p.Scenes[0].Entities[0].Position.X != 18 {
		t.Fatal("closed preview kept executing")
	}
	pikey.ResetState()
}
func TestRuntimeSubscriberInstallationIsIdempotent(t *testing.T) {
	p := project.NewProject("subscriber-test")
	rt, err := Boot(p, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()
	first := rt.verbHandler
	InstallSubscribers(rt)
	if first == 0 || rt.verbHandler == 0 || first == rt.verbHandler {
		t.Fatal("handler was not replaced")
	}
	rt.Close()
	if rt.verbHandler != 0 {
		t.Fatal("handler survived close")
	}
}
