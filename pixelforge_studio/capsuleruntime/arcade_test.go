package capsuleruntime_test

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_arcade"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
)

func TestBundledArcadeProjectsBoot(t *testing.T) {
	for _, kind := range []string{"dungeon", "frogger", "city"} {
		t.Run(kind, func(t *testing.T) {
			project, assets, err := pixelforge_arcade.Project(kind)
			if err != nil {
				t.Fatal(err)
			}
			rt, err := capsuleruntime.Boot(project, assets, capsuleruntime.Options{SkipSubscribers: true})
			if err != nil {
				t.Fatal(err)
			}
			defer rt.Close()
			if rt.Arcade == nil || rt.Arcade.Kind != kind {
				t.Fatal("project did not create its arcade runtime")
			}
			if _, err := assets.Open("assets/scene.png"); err != nil {
				t.Fatal("preview asset missing:", err)
			}
		})
	}
}

func TestArcadeRejectsIncompatibleScreen(t *testing.T) {
	p := pixelforge_project.NewProject("invalid arcade screen")
	p.Arcade = &pixelforge_project.ArcadeConfig{Kind: "dungeon"}
	if _, err := capsuleruntime.Boot(p, nil, capsuleruntime.Options{}); err == nil {
		t.Fatal("invalid screen accepted")
	}
}
