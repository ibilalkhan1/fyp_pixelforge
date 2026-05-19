package templates

import "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"

// BlankPlatformer returns a fresh project pre-wired with the Mario-class
// platformer preset (plan-009 U22, R10):
//
//   - PhysicsPreset = "mario" (gravity + tile-AABB)
//   - One Hero entity carrying the starter pack hero sprite, positioned
//     near the bottom-centre of the scene
//   - A single 20×11 ground tile-atlas with a floor row
//   - WASD + Space input bindings (left/right/down + jump)
//   - AutoSnapshot save config
//
// The template is hand-built rather than loaded from a fixture so it
// does not depend on the asset library bootstrap — File → New must
// work offline.
func BlankPlatformer() *pixelforge_project.Project {
	p := pixelforge_project.NewProject("Blank Platformer")
	p.PhysicsPreset = "mario"

	p.Sprites = []pixelforge_project.SpriteAsset{
		{
			Name:         "hero",
			RelativePath: "sprites/hero.png",
			Width:        16,
			Height:       16,
			FrameW:       16,
			FrameH:       16,
			OriginX:      8,
			OriginY:      8,
		},
		{
			Name:         "ground_tile",
			RelativePath: "sprites/platform.png",
			Width:        16,
			Height:       16,
			FrameW:       16,
			FrameH:       16,
		},
	}

	// 20×11 atlas with the bottom row filled — a runnable starter
	// floor the hero can stand on.
	grid := make([][]int, 11)
	for r := range grid {
		row := make([]int, 20)
		if r == 10 {
			for c := range row {
				row[c] = 1
			}
		}
		grid[r] = row
	}

	p.Scenes = []pixelforge_project.Scene{{
		ID:   "main",
		Name: "Main",
		Entities: []pixelforge_project.Entity{{
			ID:   "hero_1",
			Name: "Hero",
			Position: pixelforge_project.EntityPosition{
				X: 40, Y: 130, Z: 0,
			},
			Components: []pixelforge_project.EntityComponent{
				{Type: "Sprite", Values: map[string]any{"sprite": "hero"}},
			},
			Archetype: "Player",
		}},
		TileAtlases: []pixelforge_project.TileAtlas{{
			Name:           "ground",
			TileW:          16,
			TileH:          16,
			Grid:           grid,
			AutoTileRules:  []pixelforge_project.AutoTileRule{},
			SpriteSheetRef: "ground_tile",
		}},
		GridWidthScreens:  1,
		GridHeightScreens: 1,
		SpawnTile:         pixelforge_project.SpawnCell{Col: 2, Row: 8},
	}}

	p.InputMap = []pixelforge_project.InputBinding{
		{Intent: "input/left", Keyboard: []string{"A"}},
		{Intent: "input/right", Keyboard: []string{"D"}},
		{Intent: "input/jump", Keyboard: []string{" "}},
		{Intent: "input/down", Keyboard: []string{"S"}},
	}

	p.SaveConfig = defaultSaveConfig()
	return p
}
