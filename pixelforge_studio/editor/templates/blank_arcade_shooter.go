package templates

import "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"

// BlankArcadeShooter returns a fresh project pre-wired with the
// Asteroids-class arcade preset (plan-009 U22, R10):
//
//   - PhysicsPreset = "asteroids" (no gravity, screen-wrap on)
//   - One Ship entity carrying the starter hero sprite, centred on
//     the playfield
//   - No tile atlas (Asteroids has no terrain)
//   - Arrow keys + Space input bindings (rotate_left / rotate_right /
//     thrust / fire)
//   - AutoSnapshot save config
func BlankArcadeShooter() *pixelforge_project.Project {
	p := pixelforge_project.NewProject("Blank Arcade Shooter")
	p.PhysicsPreset = "asteroids"

	p.Sprites = []pixelforge_project.SpriteAsset{
		{
			Name:         "ship",
			RelativePath: "sprites/hero.png",
			Width:        16,
			Height:       16,
			FrameW:       16,
			FrameH:       16,
			OriginX:      8,
			OriginY:      8,
		},
	}

	p.Scenes = []pixelforge_project.Scene{{
		ID:   "main",
		Name: "Main",
		Entities: []pixelforge_project.Entity{{
			ID:   "ship_1",
			Name: "Ship",
			Position: pixelforge_project.EntityPosition{
				// Centre of the 320×180 default screen.
				X: 160, Y: 90, Z: 0,
			},
			Components: []pixelforge_project.EntityComponent{
				{Type: "Sprite", Values: map[string]any{"sprite": "ship"}},
			},
			Archetype: "Player",
		}},
		TileAtlases:       []pixelforge_project.TileAtlas{},
		GridWidthScreens:  1,
		GridHeightScreens: 1,
	}}

	p.InputMap = []pixelforge_project.InputBinding{
		{Intent: "input/rotate_left", Keyboard: []string{"Left"}},
		{Intent: "input/rotate_right", Keyboard: []string{"Right"}},
		{Intent: "input/thrust", Keyboard: []string{"Up"}},
		{Intent: "input/fire", Keyboard: []string{" "}},
	}

	p.SaveConfig = defaultSaveConfig()
	return p
}
