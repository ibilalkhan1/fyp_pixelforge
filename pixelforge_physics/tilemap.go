package pixelforge_physics

import "github.com/solarlune/resolv"

// TileGrid is a minimal interface describing what BuildTileGridCollider
// needs to know about a tilemap. The pixelforge_project package supplies a
// thin adapter that satisfies this interface from its TileAtlas type — we
// avoid an import dependency by keeping the contract narrow.
type TileGrid interface {
	// Width in tiles.
	GridWidth() int
	// Height in tiles.
	GridHeight() int
	// TileID returns the tile id at (col, row). Used by the caller to
	// decide which IDs are solid via solidIDs.
	TileID(col, row int) int
}

// BuildTileGridCollider scans the grid for tiles whose ID is in solidIDs
// and emits a *resolv.ConvexPolygon (axis-aligned rectangle) for each solid
// tile, positioned at the right pixel coordinates.
//
// Returned polygons are NOT automatically added to a World — caller does
// `world.Space.Add(poly, ...)`.
//
// We emit one rectangle per solid tile rather than running a greedy merge
// pass. The performance cost is bounded (resolv's broad-phase grid
// dispatches to the right cell in O(1)) and the simpler representation
// makes per-tile metadata (e.g. one-way platform flags) easier in later
// iterations.
func BuildTileGridCollider(grid TileGrid, solidIDs []int, tileW, tileH int) []*resolv.ConvexPolygon {
	if grid == nil || tileW <= 0 || tileH <= 0 {
		return nil
	}
	solidSet := make(map[int]bool, len(solidIDs))
	for _, id := range solidIDs {
		solidSet[id] = true
	}

	gw := grid.GridWidth()
	gh := grid.GridHeight()
	out := make([]*resolv.ConvexPolygon, 0, gw*gh/4)
	for row := 0; row < gh; row++ {
		for col := 0; col < gw; col++ {
			id := grid.TileID(col, row)
			if !solidSet[id] {
				continue
			}
			x := float64(col * tileW)
			y := float64(row * tileH)
			poly := resolv.NewRectangleFromTopLeft(x, y, float64(tileW), float64(tileH))
			out = append(out, poly)
		}
	}
	return out
}

// ResolveTilemapCollision is the hand-rolled 16×16 AABB-vs-tilemap sweeper
// for the platformer collision path. It walks the body's bounding rect
// against the grid (independently of resolv) and resolves overlaps along
// each axis.
//
// The algorithm runs in two passes (X then Y) — this is the standard
// recipe that avoids "stuck in the corner" artefacts and is fully integer.
// All math is Fixed32: the tile bounds are computed by .Int() truncation,
// then re-converted back at the end.
//
// Returns true if a collision was resolved (caller may consult body.Grounded
// for the bottom-touch state).
func ResolveTilemapCollision(body *Body, world *World, grid TileGrid, solidIDs []int, tileW, tileH int) bool {
	if body == nil || world == nil || grid == nil || tileW <= 0 || tileH <= 0 {
		return false
	}
	solidSet := make(map[int]bool, len(solidIDs))
	for _, id := range solidIDs {
		solidSet[id] = true
	}

	// Convert body bounds to pixel-integer extents (truncating, deterministic).
	bx := body.Position.X.Int()
	by := body.Position.Y.Int()
	bw := body.Width.Int()
	bh := body.Height.Int()
	if bw <= 0 || bh <= 0 {
		return false
	}

	// Determine tile-range overlapped by the body.
	minCol := bx / tileW
	maxCol := (bx + bw - 1) / tileW
	minRow := by / tileH
	maxRow := (by + bh - 1) / tileH
	if minCol < 0 {
		minCol = 0
	}
	if minRow < 0 {
		minRow = 0
	}
	if maxCol >= grid.GridWidth() {
		maxCol = grid.GridWidth() - 1
	}
	if maxRow >= grid.GridHeight() {
		maxRow = grid.GridHeight() - 1
	}

	resolved := false
	body.Grounded = false

	for row := minRow; row <= maxRow; row++ {
		for col := minCol; col <= maxCol; col++ {
			if !solidSet[grid.TileID(col, row)] {
				continue
			}
			tx := col * tileW
			ty := row * tileH
			// Compute intersection on each axis.
			interLeft := maxInt(bx, tx)
			interRight := minInt(bx+bw, tx+tileW)
			interTop := maxInt(by, ty)
			interBottom := minInt(by+bh, ty+tileH)
			overlapX := interRight - interLeft
			overlapY := interBottom - interTop
			if overlapX <= 0 || overlapY <= 0 {
				continue
			}
			// Resolve along the smaller-overlap axis (gives "land on top"
			// behaviour when falling, "bump into wall" when running).
			//
			// Tie-break (overlapY == overlapX): pick the axis the body
			// is currently moving along. A grounded body whose gravity
			// step just pushed it into the floor has vy > 0 and vx == 0,
			// so Y-axis resolution is the correct choice — the plan's
			// canonical "Y-first sweep order" hint. Without this tie-
			// break the substrate picks X-resolution on a wide-body /
			// equal-tile-size collision and the body teleports sideways
			// out of the floor instead of landing on it.
			preferY := overlapY < overlapX ||
				(overlapY == overlapX && body.Velocity.Y != 0 && body.Velocity.X == 0)
			if preferY {
				// Y resolution. Push along the axis opposite to motion: if
				// body is falling (vy > 0), push up; if jumping (vy < 0),
				// push down. If vy == 0, decide by the body-vs-tile center
				// (whichever is closer wins).
				pushUp := body.Velocity.Y > 0 ||
					(body.Velocity.Y == 0 && (by+bh/2) < (ty+tileH/2))
				if pushUp {
					// New top = tile top - body height.
					body.Position.Y = FixedFromInt(ty - bh)
					if body.Velocity.Y > 0 {
						body.Velocity.Y = FixedZero
					}
					body.Grounded = true
				} else {
					// New top = tile bottom.
					body.Position.Y = FixedFromInt(ty + tileH)
					if body.Velocity.Y < 0 {
						body.Velocity.Y = FixedZero
					}
				}
				by = body.Position.Y.Int()
			} else {
				// X resolution. Same decision rule: push opposite to motion.
				pushLeft := body.Velocity.X > 0 ||
					(body.Velocity.X == 0 && (bx+bw/2) < (tx+tileW/2))
				if pushLeft {
					// New left = tile left - body width.
					body.Position.X = FixedFromInt(tx - bw)
					if body.Velocity.X > 0 {
						body.Velocity.X = FixedZero
					}
				} else {
					// New left = tile right.
					body.Position.X = FixedFromInt(tx + tileW)
					if body.Velocity.X < 0 {
						body.Velocity.X = FixedZero
					}
				}
				bx = body.Position.X.Int()
			}
			resolved = true
		}
	}
	if resolved {
		body.Sync()
	}
	return resolved
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
