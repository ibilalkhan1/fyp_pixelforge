package capsuleruntime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_physics"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// bombermanProject builds a fixture project with the Bomberman
// physics preset, a single "bomb" entity at (0, 0), and an optional
// 16×16 TileAtlas. Tests that need wall geometry call withGrid below
// to graft on a painted atlas after Boot.
func bombermanProject() *pixelforge_project.Project {
	p := pixelforge_project.NewProject("motion-sink-bomberman-test")
	p.PhysicsPreset = "bomberman"
	p.Scenes = []pixelforge_project.Scene{
		{
			ID:   "main",
			Name: "Main",
			Entities: []pixelforge_project.Entity{
				{ID: "bomb_1", Name: "Bomb", Position: pixelforge_project.EntityPosition{X: 0, Y: 0}},
			},
		},
	}
	return p
}

// withGrid attaches a 16×16 TileAtlas with the supplied grid to the
// scene. Convenience for tests that need a specific wall layout.
func withGrid(scene *pixelforge_project.Scene, grid [][]int) {
	scene.TileAtlases = []pixelforge_project.TileAtlas{
		{
			Name:  "blocks",
			TileW: 16,
			TileH: 16,
			Grid:  grid,
		},
	}
}

// emptyGrid returns a rows×cols grid of zero cells. Used by tests
// that want explicit dimensions but no solid tiles.
func emptyGrid(rows, cols int) [][]int {
	out := make([][]int, rows)
	for r := 0; r < rows; r++ {
		out[r] = make([]int, cols)
	}
	return out
}

func TestMotionSink_PlaceOnGrid_SnapsToTileCoords(t *testing.T) {
	rt := bootDefault(t, bombermanProject())
	body := rt.Bodies["bomb_1"]
	require.NotNil(t, body)

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/place_on_grid",
		Args:  map[string]any{"entity": "bomb_1", "grid_x": float64(3), "grid_y": float64(5)},
	})

	// 16×16 default tile size → (3*16, 5*16) = (48, 80).
	assert.Equal(t, pixelforge_physics.FixedFromInt(48), body.Position.X,
		"X should snap to grid_x*tileW")
	assert.Equal(t, pixelforge_physics.FixedFromInt(80), body.Position.Y,
		"Y should snap to grid_y*tileH")
}

func TestMotionSink_PlaceOnGrid_UsesSceneTileSize(t *testing.T) {
	rt := bootDefault(t, bombermanProject())
	// Override with an 8×8 atlas so the test exercises the
	// atlas-derived tile-size path (not the 16×16 fallback).
	withGrid(rt.CurrentScene, emptyGrid(4, 4))
	rt.CurrentScene.TileAtlases[0].TileW = 8
	rt.CurrentScene.TileAtlases[0].TileH = 8
	body := rt.Bodies["bomb_1"]

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/place_on_grid",
		Args:  map[string]any{"entity": "bomb_1", "grid_x": float64(2), "grid_y": float64(3)},
	})

	// 8×8 atlas → (2*8, 3*8) = (16, 24).
	assert.Equal(t, pixelforge_physics.FixedFromInt(16), body.Position.X)
	assert.Equal(t, pixelforge_physics.FixedFromInt(24), body.Position.Y)
}

func TestMotionSink_PlaceOnGrid_OriginCellSnapsToZero(t *testing.T) {
	rt := bootDefault(t, bombermanProject())
	body := rt.Bodies["bomb_1"]
	// Move the body somewhere off-grid first to prove the snap works.
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromFloat(123.5),
		Y: pixelforge_physics.FixedFromFloat(45.25),
	}

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/place_on_grid",
		Args:  map[string]any{"entity": "bomb_1", "grid_x": float64(0), "grid_y": float64(0)},
	})

	assert.Equal(t, pixelforge_physics.FixedZero, body.Position.X)
	assert.Equal(t, pixelforge_physics.FixedZero, body.Position.Y)
}

func TestMotionSink_PlaceOnGrid_MissingEntityIsNoOp(t *testing.T) {
	rt := bootDefault(t, bombermanProject())
	body := rt.Bodies["bomb_1"]
	startPos := body.Position

	out := captureLog(t, func() {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/place_on_grid",
			Args:  map[string]any{"grid_x": float64(3), "grid_y": float64(5)},
		})
	})

	assert.Contains(t, out, "motion/place_on_grid")
	assert.Equal(t, startPos, body.Position)
}

func TestMotionSink_PlaceOnGrid_UnknownEntityLogsAndNoMutation(t *testing.T) {
	rt := bootDefault(t, bombermanProject())
	body := rt.Bodies["bomb_1"]
	startPos := body.Position

	out := captureLog(t, func() {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/place_on_grid",
			Args:  map[string]any{"entity": "ghost", "grid_x": float64(3), "grid_y": float64(5)},
		})
	})

	assert.Contains(t, out, "unknown entity")
	assert.Contains(t, out, "ghost")
	assert.Equal(t, startPos, body.Position, "unknown-entity place_on_grid must not mutate the real bomb")
}

// ---- grid-explode tests --------------------------------------------

// gridExplodeProject builds a 5-entity fixture for the grid-explode
// tests. Three entities sit on cells the blast will reach; two sit
// off-blast. Each entity carries an HP component so take_damage has
// a target to write into. No tile atlas yet — tests that need walls
// call withGrid.
func gridExplodeProject() *pixelforge_project.Project {
	mkHP := func() []pixelforge_project.EntityComponent {
		return []pixelforge_project.EntityComponent{
			{
				Type: pixelforge_project.HealthComponentType,
				Values: map[string]any{
					"hp":     float64(10),
					"max_hp": float64(10),
				},
			},
		}
	}
	p := pixelforge_project.NewProject("grid-explode-test")
	p.PhysicsPreset = "bomberman"
	p.Scenes = []pixelforge_project.Scene{
		{
			ID: "main", Name: "Main",
			Entities: []pixelforge_project.Entity{
				// Bomb at (5,5) → pixel (80, 80). Range 3 covers
				// cells (5..8, 5) east, (2..4, 5) west, (5, 5..8)
				// south, (5, 2..4) north.
				{ID: "bomb", Name: "Bomb", Position: pixelforge_project.EntityPosition{X: 80, Y: 80}, Components: mkHP()},
				// East 1 tile (cell 6,5) → pixel (96, 80).
				{ID: "east_1", Name: "East1", Position: pixelforge_project.EntityPosition{X: 96, Y: 80}, Components: mkHP()},
				// West 2 tiles (cell 3,5) → pixel (48, 80).
				{ID: "west_2", Name: "West2", Position: pixelforge_project.EntityPosition{X: 48, Y: 80}, Components: mkHP()},
				// South 3 tiles (cell 5,8) → pixel (80, 128).
				{ID: "south_3", Name: "South3", Position: pixelforge_project.EntityPosition{X: 80, Y: 128}, Components: mkHP()},
				// Far away (cell 1,1) → pixel (16, 16), out of range.
				{ID: "far", Name: "Far", Position: pixelforge_project.EntityPosition{X: 16, Y: 16}, Components: mkHP()},
			},
		},
	}
	return p
}

func TestDamageSink_GridExplode_FourCardinalsClear(t *testing.T) {
	bootDefault(t, gridExplodeProject())
	rec := newBusRecorder()
	rec.attach()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/grid_explode",
		Args: map[string]any{
			"origin": map[string]any{"gx": 5, "gy": 5},
			"range":  float64(3),
			"damage": float64(3),
		},
	})

	hits := rec.filterTopic("damage/take_damage")
	// east_1 + west_2 + south_3 each sit on cells the blast reaches.
	// "far" sits at cell (1,1), outside the 3-tile range in every
	// direction. "bomb" is at the origin cell itself (NOT damaged
	// because the blast skips the origin — per Bomberman semantics
	// the bomb is removed first and the blast covers the surrounding
	// rays only).
	got := map[string]bool{}
	for _, h := range hits {
		id, _ := h.Args["entity"].(string)
		got[id] = true
	}
	assert.True(t, got["east_1"], "east_1 should be damaged")
	assert.True(t, got["west_2"], "west_2 should be damaged")
	assert.True(t, got["south_3"], "south_3 should be damaged")
	assert.False(t, got["far"], "far entity must NOT be damaged")
	assert.False(t, got["bomb"], "origin-cell entity must NOT be damaged (blast skips origin)")
}

func TestDamageSink_GridExplode_NoWallsCounts12Tiles(t *testing.T) {
	// Pack 12 entities — one per affected cell — to verify the blast
	// touches exactly 12 tiles when no walls block. 3 tiles in each
	// of 4 cardinal directions = 12.
	p := pixelforge_project.NewProject("grid-explode-count-test")
	p.PhysicsPreset = "bomberman"
	mkHP := func(id string, x, y float64) pixelforge_project.Entity {
		return pixelforge_project.Entity{
			ID: id, Name: id,
			Position: pixelforge_project.EntityPosition{X: x, Y: y},
			Components: []pixelforge_project.EntityComponent{
				{
					Type:   pixelforge_project.HealthComponentType,
					Values: map[string]any{"hp": float64(10), "max_hp": float64(10)},
				},
			},
		}
	}
	ents := []pixelforge_project.Entity{}
	// East: (6,5), (7,5), (8,5).
	for step := 1; step <= 3; step++ {
		ents = append(ents, mkHP("e"+itoa(step), float64((5+step)*16), 80))
	}
	// West: (4,5), (3,5), (2,5).
	for step := 1; step <= 3; step++ {
		ents = append(ents, mkHP("w"+itoa(step), float64((5-step)*16), 80))
	}
	// South: (5,6), (5,7), (5,8).
	for step := 1; step <= 3; step++ {
		ents = append(ents, mkHP("s"+itoa(step), 80, float64((5+step)*16)))
	}
	// North: (5,4), (5,3), (5,2).
	for step := 1; step <= 3; step++ {
		ents = append(ents, mkHP("n"+itoa(step), 80, float64((5-step)*16)))
	}
	p.Scenes = []pixelforge_project.Scene{
		{ID: "main", Name: "Main", Entities: ents},
	}

	bootDefault(t, p)
	rec := newBusRecorder()
	rec.attach()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/grid_explode",
		Args: map[string]any{
			"origin": map[string]any{"gx": 5, "gy": 5},
			"range":  float64(3),
			"damage": float64(1),
		},
	})

	hits := rec.filterTopic("damage/take_damage")
	assert.Len(t, hits, 12, "exactly 12 tiles should be touched with no walls (3 in each of 4 cardinal directions)")
}

func TestDamageSink_GridExplode_BlockedByWallNorth(t *testing.T) {
	p := pixelforge_project.NewProject("grid-explode-wall-test")
	p.PhysicsPreset = "bomberman"
	mkHP := func(id string, x, y float64) pixelforge_project.Entity {
		return pixelforge_project.Entity{
			ID: id, Name: id,
			Position: pixelforge_project.EntityPosition{X: x, Y: y},
			Components: []pixelforge_project.EntityComponent{
				{
					Type:   pixelforge_project.HealthComponentType,
					Values: map[string]any{"hp": float64(10), "max_hp": float64(10)},
				},
			},
		}
	}
	// 12 candidate targets, but a wall blocks the north arm at (5,4)
	// — so the 3 northern cells (5,4)/(5,3)/(5,2) should all be
	// blocked. 9 tiles should still take damage.
	ents := []pixelforge_project.Entity{}
	for step := 1; step <= 3; step++ {
		ents = append(ents, mkHP("e"+itoa(step), float64((5+step)*16), 80))
		ents = append(ents, mkHP("w"+itoa(step), float64((5-step)*16), 80))
		ents = append(ents, mkHP("s"+itoa(step), 80, float64((5+step)*16)))
		ents = append(ents, mkHP("n"+itoa(step), 80, float64((5-step)*16)))
	}
	p.Scenes = []pixelforge_project.Scene{
		{ID: "main", Name: "Main", Entities: ents},
	}

	rt := bootDefault(t, p)
	// Paint a 10×10 grid; mark (col=5, row=4) solid (the immediately-
	// north neighbor of the bomb at (5,5)). All other cells are
	// clear so the east/west/south rays still walk the full range.
	grid := emptyGrid(10, 10)
	grid[4][5] = 1
	withGrid(rt.CurrentScene, grid)

	rec := newBusRecorder()
	rec.attach()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/grid_explode",
		Args: map[string]any{
			"origin": map[string]any{"gx": 5, "gy": 5},
			"range":  float64(3),
			"damage": float64(1),
		},
	})

	hits := rec.filterTopic("damage/take_damage")
	got := map[string]bool{}
	for _, h := range hits {
		id, _ := h.Args["entity"].(string)
		got[id] = true
	}
	// East/west/south arms (3+3+3=9) should fire.
	assert.True(t, got["e1"] && got["e2"] && got["e3"], "east arm should be unblocked")
	assert.True(t, got["w1"] && got["w2"] && got["w3"], "west arm should be unblocked")
	assert.True(t, got["s1"] && got["s2"] && got["s3"], "south arm should be unblocked")
	// North arm cells must NOT fire — even though entities exist on
	// those cells, the solid tile at (5,4) blocks the ray.
	assert.False(t, got["n1"], "n1 must be blocked by wall at (5,4)")
	assert.False(t, got["n2"], "n2 must be blocked by wall at (5,4)")
	assert.False(t, got["n3"], "n3 must be blocked by wall at (5,4)")
	assert.Len(t, hits, 9, "exactly 9 tiles should be damaged with the north arm blocked")
}

func TestDamageSink_GridExplode_PublishesVisualFlash(t *testing.T) {
	bootDefault(t, gridExplodeProject())
	rec := newBusRecorder()
	rec.attach()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/grid_explode",
		Args: map[string]any{
			"origin": map[string]any{"gx": 5, "gy": 5},
			"range":  float64(3),
			"damage": float64(1),
		},
	})

	flashes := rec.filterTopic("visual/flash")
	require.Len(t, flashes, 1, "grid_explode must publish exactly one visual/flash")
	assert.Equal(t, "_blast_center", flashes[0].Args["entity"])
}

func TestDamageSink_GridExplode_PlaysSFXWhenAudioRegistered(t *testing.T) {
	p := gridExplodeProject()
	p.Audio = []pixelforge_project.AudioSample{
		{Name: "bomb_explode"},
	}
	bootDefault(t, p)
	rec := newBusRecorder()
	rec.attach()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/grid_explode",
		Args: map[string]any{
			"origin": map[string]any{"gx": 5, "gy": 5},
			"range":  float64(3),
			"damage": float64(1),
		},
	})

	sfx := rec.filterTopic("audio/play_sound")
	require.Len(t, sfx, 1)
	assert.Equal(t, "bomb_explode", sfx[0].Args["name"])
}

func TestDamageSink_GridExplode_SkipsSFXWhenSampleMissing(t *testing.T) {
	bootDefault(t, gridExplodeProject())
	rec := newBusRecorder()
	rec.attach()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/grid_explode",
		Args: map[string]any{
			"origin": map[string]any{"gx": 5, "gy": 5},
			"range":  float64(3),
			"damage": float64(1),
		},
	})

	assert.Empty(t, rec.filterTopic("audio/play_sound"), "missing SFX must skip the publish")
}

func TestDamageSink_GridExplode_NonPositiveRangeIsNoOp(t *testing.T) {
	bootDefault(t, gridExplodeProject())
	rec := newBusRecorder()
	rec.attach()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "damage/grid_explode",
		Args: map[string]any{
			"origin": map[string]any{"gx": 5, "gy": 5},
			"range":  float64(0),
			"damage": float64(5),
		},
	})

	assert.Empty(t, rec.filterTopic("damage/take_damage"))
	assert.Empty(t, rec.filterTopic("visual/flash"))
}

func TestDamageSink_GridExplode_MissingOriginIsNoOp(t *testing.T) {
	bootDefault(t, gridExplodeProject())
	rec := newBusRecorder()
	rec.attach()

	out := captureLog(t, func() {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "damage/grid_explode",
			Args: map[string]any{
				"range":  float64(3),
				"damage": float64(5),
			},
		})
	})

	assert.Contains(t, out, "damage/grid_explode")
	assert.Empty(t, rec.filterTopic("damage/take_damage"))
}

// itoa returns a stable digit-only string for small ints. Used by
// the count test above to build entity IDs deterministically.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
