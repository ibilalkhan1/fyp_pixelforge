package pixelforge_physics

// LadderDir is the direction the player is currently climbing.
type LadderDir int8

const (
	LadderIdle LadderDir = 0
	LadderUp   LadderDir = -1
	LadderDown LadderDir = 1
)

// DefaultLadderTileID is the canonical tile-id used to mark a ladder
// in a TileAtlas. The substrate hard-codes this for U17; later units
// will lift it to a per-tile attribute / a configurable list on
// PhysicsConfig (PhysicsConfig.LadderTileIDs is the reserved field).
//
// Convention: solid tiles use id=1 (the Mario fixture's choice) so
// ladder gets the next free id. Designers painting ladders into a
// DK-class .pforge fixture stamp id=2 for ladder cells; the tilemap
// resolver treats those cells as non-solid (only non-zero AND
// non-ladder ids are considered solid by the DK callsite), while the
// ladder-overlap helper picks them up.
const DefaultLadderTileID = 2

// LadderState is the per-body climbing state machine. It tracks whether the
// body is actively climbing and which direction. Transition fns gate
// gravity and let the integrator know not to apply it while climbing.
type LadderState struct {
	Climbing  bool
	Direction LadderDir
}

// BeginClimb transitions the state to Climbing in the given direction.
// Used by the runtime when the player presses up/down while OnLadder.
func (s *LadderState) BeginClimb(dir LadderDir) {
	s.Climbing = true
	s.Direction = dir
}

// EndClimb transitions out of climbing. Used when the player lets go of
// up/down or leaves the ladder.
func (s *LadderState) EndClimb() {
	s.Climbing = false
	s.Direction = LadderIdle
}

// ApplyLadderMovement moves a body along the ladder at the given climb
// speed (per tick). Returns true if the body moved.
//
// Caller is responsible for verifying OnLadder + Climbing. This helper just
// applies the motion deterministically.
func ApplyLadderMovement(body *Body, state LadderState, speed Fixed32) bool {
	if body == nil || !state.Climbing {
		return false
	}
	switch state.Direction {
	case LadderUp:
		body.Position.Y = body.Position.Y.Sub(speed)
	case LadderDown:
		body.Position.Y = body.Position.Y.Add(speed)
	default:
		return false
	}
	// Zero out vertical velocity while climbing — gravity is suspended.
	body.Velocity.Y = FixedZero
	body.Sync()
	return true
}

// BodyOverlapsLadderTile reports whether the body's AABB overlaps any
// cell in grid whose TileID matches ladderID. tileW/tileH are the
// atlas's cell dimensions in pixels.
//
// U17 uses this to gate "can this entity start climbing?". The
// substrate doesn't mutate body.OnLadder here — callers in the
// runtime layer write that flag after consulting this helper, so the
// physics package stays free of an opinion on per-tick state ordering.
//
// Returns false when grid is nil, the body has zero footprint, or no
// ladder cell intersects the body.
func BodyOverlapsLadderTile(body *Body, grid TileGrid, ladderID, tileW, tileH int) bool {
	if body == nil || grid == nil || tileW <= 0 || tileH <= 0 {
		return false
	}
	bx := body.Position.X.Int()
	by := body.Position.Y.Int()
	bw := body.Width.Int()
	bh := body.Height.Int()
	if bw <= 0 || bh <= 0 {
		return false
	}
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
	for row := minRow; row <= maxRow; row++ {
		for col := minCol; col <= maxCol; col++ {
			if grid.TileID(col, row) == ladderID {
				return true
			}
		}
	}
	return false
}

// LadderTopReached reports whether the body's TOP edge has cleared
// the topmost ladder cell underneath it — i.e. the climber has
// climbed up onto a platform. Used by the runtime to auto-release
// the climb state and re-ground the body once it reaches the top.
//
// The check is: the body's top edge sits ABOVE the top-most ladder
// tile that touches the body's column. If no ladder tile overlaps
// the body at all the function returns false (a climber who has
// stepped clean off the ladder is "released", not "at top").
func LadderTopReached(body *Body, grid TileGrid, ladderID, tileW, tileH int) bool {
	if body == nil || grid == nil || tileW <= 0 || tileH <= 0 {
		return false
	}
	bx := body.Position.X.Int()
	by := body.Position.Y.Int()
	bw := body.Width.Int()
	if bw <= 0 {
		return false
	}
	minCol := bx / tileW
	maxCol := (bx + bw - 1) / tileW
	if minCol < 0 {
		minCol = 0
	}
	if maxCol >= grid.GridWidth() {
		maxCol = grid.GridWidth() - 1
	}
	// Find the topmost ladder tile in any column the body overlaps.
	topRow := -1
	for col := minCol; col <= maxCol; col++ {
		for row := 0; row < grid.GridHeight(); row++ {
			if grid.TileID(col, row) == ladderID {
				topRow = row
				break
			}
		}
		if topRow >= 0 {
			break
		}
	}
	if topRow < 0 {
		return false
	}
	ladderTopY := topRow * tileH
	// Body has reached the top when its TOP edge has risen above the
	// ladder's topmost tile (i.e. by <= ladderTopY).
	return by <= ladderTopY
}
