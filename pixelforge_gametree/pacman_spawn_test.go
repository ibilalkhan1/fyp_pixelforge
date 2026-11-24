//nolint:testpackage // Inspects maze tiles and drives private game updates.
package pixelforge_gametree

import "testing"

func TestPacmanStartsInWalkableCorridorAndCollectsDots(t *testing.T) {
	g := NewPacmanGame()
	g.Sound(false)
	g.resetLevel()
	if g.isWallTile(int(g.pac.x)/pTileSize, int(g.pac.y)/pTileSize) {
		t.Fatal("Pac-Man spawned inside a wall")
	}
	start := g.pac.x
	for range 24 {
		g.update()
	}
	if g.pac.x >= start || g.score == 0 {
		t.Fatalf("Pac-Man did not move and eat dots: x=%v score=%v", g.pac.x, g.score)
	}
}
