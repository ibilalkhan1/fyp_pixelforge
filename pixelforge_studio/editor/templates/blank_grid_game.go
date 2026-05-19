package templates

import "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"

// BlankGridGame returns a fresh project pre-wired with the Bomberman-
// class grid preset (plan-009 U22, R10):
//
//   - PhysicsPreset = "bomberman" (no gravity, grid-AABB)
//   - One Player entity grid-aligned at a clear interior cell
//   - A 13×11 tile atlas (the canonical Bomberman board size) with a
//     wall border framing the play area
//   - Arrow keys + Space input bindings (move_up/down/left/right +
//     place_bomb)
//   - AutoSnapshot save config
//
// The 13×11 board mirrors the reference game's canonical layout (one
// odd column + one odd row so destructible blocks tile cleanly on the
// even cells in v2 designer flows).
func BlankGridGame() *pixelforge_project.Project {
	p := pixelforge_project.NewProject("Blank Grid Game")
	p.PhysicsPreset = "bomberman"

	p.Sprites = []pixelforge_project.SpriteAsset{
		{
			Name:         "player",
			RelativePath: "sprites/hero.png",
			Width:        16,
			Height:       16,
			FrameW:       16,
			FrameH:       16,
			OriginX:      8,
			OriginY:      8,
		},
		{
			Name:         "wall",
			RelativePath: "sprites/wall.png",
			Width:        16,
			Height:       16,
			FrameW:       16,
			FrameH:       16,
		},
	}

	// 13 cols × 11 rows with a wall border. Cell value 1 = wall sprite.
	cols, rows := 13, 11
	grid := make([][]int, rows)
	for r := 0; r < rows; r++ {
		row := make([]int, cols)
		for c := 0; c < cols; c++ {
			if r == 0 || r == rows-1 || c == 0 || c == cols-1 {
				row[c] = 1
			}
		}
		grid[r] = row
	}

	p.Scenes = []pixelforge_project.Scene{{
		ID:   "main",
		Name: "Main",
		Entities: []pixelforge_project.Entity{{
			ID:   "player_1",
			Name: "Player",
			Position: pixelforge_project.EntityPosition{
				// Grid-aligned: cell (1,1) at 16px tile size.
				X: 16, Y: 16, Z: 0,
			},
			Components: []pixelforge_project.EntityComponent{
				{Type: "Sprite", Values: map[string]any{"sprite": "player"}},
			},
			Archetype: "Player",
			TileX:     1,
			TileY:     1,
		}},
		TileAtlases: []pixelforge_project.TileAtlas{{
			Name:           "board",
			TileW:          16,
			TileH:          16,
			Grid:           grid,
			AutoTileRules:  []pixelforge_project.AutoTileRule{},
			SpriteSheetRef: "wall",
		}},
		GridWidthScreens:  1,
		GridHeightScreens: 1,
		SpawnTile:         pixelforge_project.SpawnCell{Col: 1, Row: 1},
	}}

	p.InputMap = []pixelforge_project.InputBinding{
		{Intent: "input/move_up", Keyboard: []string{"Up"}},
		{Intent: "input/move_down", Keyboard: []string{"Down"}},
		{Intent: "input/move_left", Keyboard: []string{"Left"}},
		{Intent: "input/move_right", Keyboard: []string{"Right"}},
		{Intent: "input/place_bomb", Keyboard: []string{" "}},
	}

	p.SaveConfig = defaultSaveConfig()
	return p
}
