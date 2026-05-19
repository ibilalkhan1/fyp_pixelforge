package templates

import "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"

// BlankLadderPlatformer returns a fresh project pre-wired with the
// Donkey-Kong-class ladder platformer preset (plan-009 U22, R10):
//
//   - PhysicsPreset = "dk" (Mario gravity + ladder-aware AABB)
//   - One Hero entity standing on the bottom platform
//   - A 20×11 tile atlas with three stacked platforms and a ladder
//     column connecting them
//   - WASD + Space + W/S input bindings (left/right + jump + climb_up/
//     climb_down)
//   - AutoSnapshot save config
//
// Tile value legend (per the grid below):
//
//	0 — empty
//	1 — platform tile
//	2 — ladder tile
//
// The ladder column lives at x=10; platforms sit on rows 10, 6, and 2.
func BlankLadderPlatformer() *pixelforge_project.Project {
	p := pixelforge_project.NewProject("Blank Ladder Platformer")
	p.PhysicsPreset = "dk"

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
			Name:         "platform",
			RelativePath: "sprites/platform.png",
			Width:        16,
			Height:       16,
			FrameW:       16,
			FrameH:       16,
		},
		{
			Name:         "ladder",
			RelativePath: "sprites/wall.png",
			Width:        16,
			Height:       16,
			FrameW:       16,
			FrameH:       16,
		},
	}

	cols, rows := 20, 11
	grid := make([][]int, rows)
	for r := range grid {
		grid[r] = make([]int, cols)
	}
	// Bottom platform — full width.
	for c := 0; c < cols; c++ {
		grid[10][c] = 1
	}
	// Middle platform.
	for c := 2; c < cols-2; c++ {
		grid[6][c] = 1
	}
	// Top platform.
	for c := 0; c < cols-3; c++ {
		grid[2][c] = 1
	}
	// Ladder column linking all three platforms.
	ladderCol := 10
	for r := 3; r <= 10; r++ {
		if grid[r][ladderCol] == 0 {
			grid[r][ladderCol] = 2
		}
	}

	p.Scenes = []pixelforge_project.Scene{{
		ID:   "main",
		Name: "Main",
		Entities: []pixelforge_project.Entity{{
			ID:   "hero_1",
			Name: "Hero",
			Position: pixelforge_project.EntityPosition{
				// Standing on the bottom platform, near the left edge.
				X: 32, Y: 144, Z: 0,
			},
			Components: []pixelforge_project.EntityComponent{
				{Type: "Sprite", Values: map[string]any{"sprite": "hero"}},
			},
			Archetype: "Player",
		}},
		TileAtlases: []pixelforge_project.TileAtlas{{
			Name:           "platforms",
			TileW:          16,
			TileH:          16,
			Grid:           grid,
			AutoTileRules:  []pixelforge_project.AutoTileRule{},
			SpriteSheetRef: "platform",
		}},
		GridWidthScreens:  1,
		GridHeightScreens: 1,
		SpawnTile:         pixelforge_project.SpawnCell{Col: 2, Row: 9},
	}}

	p.InputMap = []pixelforge_project.InputBinding{
		{Intent: "input/left", Keyboard: []string{"A"}},
		{Intent: "input/right", Keyboard: []string{"D"}},
		{Intent: "input/jump", Keyboard: []string{" "}},
		{Intent: "input/climb_up", Keyboard: []string{"W"}},
		{Intent: "input/climb_down", Keyboard: []string{"S"}},
	}

	p.SaveConfig = defaultSaveConfig()
	return p
}
