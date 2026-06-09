// Package pixelforge_enemyai provides the decision-making layer for
// hostile non-player characters. Every enemy evaluates the world
// through one of four geometric tracking primitives and then
// transitions through a four-state behaviour machine that mirrors
// the perceptual cycle of a guard: patrol, chase, search, alert.
package pixelforge_enemyai

import (
	"container/heap"
	"container/list"
	"math"
)

// Vec2 is a floating-point coordinate pair used for world-space
// positions. The engine stores sprite centres as Vec2 so that
// sub-pixel movement stays smooth between frames.
type Vec2 struct {
	X float64
	Y float64
}

// Distance returns the Euclidean distance between two points.
func Distance(a, b Vec2) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// ─── 1. Coordinate Tracking ──────────────────────────────────────

// CoordinateTracker moves the enemy one step toward the player by
// independently comparing the X and Y axes. It is the cheapest
// tracking mode—only two comparisons and two additions—and is
// ideal for enemies that move on axis-aligned rails or grid
// boundaries.
func CoordinateTracker(enemy, player Vec2, speed float64) Vec2 {
	next := enemy
	if player.X > enemy.X {
		next.X += speed
	} else if player.X < enemy.X {
		next.X -= speed
	}
	if player.Y > enemy.Y {
		next.Y += speed
	} else if player.Y < enemy.Y {
		next.Y -= speed
	}
	return next
}

// ─── 2. Trigonometry Tracking ────────────────────────────────────

// TrigonometryTracker computes a true directional vector toward the
// player using atan2, then decomposes that angle into horizontal and
// vertical push components via cos and sin. The result is diagonal
// pursuit that looks natural for flying or free-roaming enemies.
func TrigonometryTracker(enemy, player Vec2, speed float64) Vec2 {
	dx := player.X - enemy.X
	dy := player.Y - enemy.Y
	// Angle measured from the positive Y axis in standard game math.
	angle := math.Atan2(dx, dy)
	hPush := math.Sin(angle) * speed
	vPush := math.Cos(angle) * speed
	return Vec2{
		X: enemy.X + hPush,
		Y: enemy.Y + vPush,
	}
}

// ─── 3. Line-of-Sight ────────────────────────────────────────────

// TileSolidFn is the signature of the collision probe used by the
// line-of-sight and pathfinding routines.
type TileSolidFn func(tx, ty int) bool

// LineOfSight casts a ray from the enemy toward the player, stepping
// pixel-by-pixel along the dominant axis. If any solid tile is
// intersected before the player is reached, the player is deemed
// hidden and the function returns false.
func LineOfSight(enemy, player Vec2, tileSize int, solid TileSolidFn) bool {
	dx := player.X - enemy.X
	dy := player.Y - enemy.Y
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist == 0 {
		return true
	}
	// Unit step along the line.
	stepX := dx / dist
	stepY := dy / dist

	steps := int(dist)
	for i := 0; i <= steps; i++ {
		px := enemy.X + stepX*float64(i)
		py := enemy.Y + stepY*float64(i)
		tx := int(px / float64(tileSize))
		ty := int(py / float64(tileSize))
		if solid(tx, ty) {
			return false
		}
	}
	return true
}

// ─── 4. Pathfinding ──────────────────────────────────────────────

// GridNode is a single cell in the tilemap graph.
type GridNode struct {
	X int
	Y int
}

// nodeKey packs a grid coordinate into a string for map lookups.
func nodeKey(n GridNode) string {
	// Simple stringification; real engine uses a fixed-size int64 key.
	return string(rune(n.X)) + ":" + string(rune(n.Y))
}

// BFS performs an un-informed breadth-first search on the tile grid.
// It returns the shortest path from start to goal as a slice of
// grid nodes, or nil if no path exists. BFS is the default for
// patrol behaviour because it guarantees optimality on uniform
// movement costs and outperforms DFS on open grids.
func BFS(start, goal GridNode, solid TileSolidFn) []GridNode {
	if solid(start.X, start.Y) || solid(goal.X, goal.Y) {
		return nil
	}
	frontier := list.New()
	frontier.PushBack(start)
	cameFrom := map[GridNode]GridNode{start: {}}
	found := false

	for frontier.Len() > 0 {
		elem := frontier.Front()
		current := elem.Value.(GridNode)
		frontier.Remove(elem)

		if current == goal {
			found = true
			break
		}

		neighbors := []GridNode{
			{current.X + 1, current.Y},
			{current.X - 1, current.Y},
			{current.X, current.Y + 1},
			{current.X, current.Y - 1},
		}
		for _, nb := range neighbors {
			if solid(nb.X, nb.Y) {
				continue
			}
			if _, ok := cameFrom[nb]; !ok {
				frontier.PushBack(nb)
				cameFrom[nb] = current
			}
		}
	}

	if !found {
		return nil
	}
	// Reconstruct path.
	var path []GridNode
	at := goal
	for at != start {
		path = append([]GridNode{at}, path...)
		at = cameFrom[at]
	}
	path = append([]GridNode{start}, path...)
	return path
}

// astarNode wraps a grid coordinate with A* bookkeeping.
type astarNode struct {
	pos    GridNode
	g      float64 // cost from start
	h      float64 // heuristic to goal
	parent *astarNode
}

func (a *astarNode) f() float64 { return a.g + a.h }

// astarHeap implements heap.Interface for the A* open set.
type astarHeap []*astarNode

func (h astarHeap) Len() int           { return len(h) }
func (h astarHeap) Less(i, j int) bool { return h[i].f() < h[j].f() }
func (h astarHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *astarHeap) Push(x any)        { *h = append(*h, x.(*astarNode)) }
func (h *astarHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

// heuristic uses Manhattan distance on the grid.
func heuristic(a, b GridNode) float64 {
	return math.Abs(float64(a.X-b.X)) + math.Abs(float64(a.Y-b.Y))
}

// AStar performs an informed A* search on the tile grid. It is
// invoked in Alert Mode when the enemy needs to intelligently
// reacquire the player after a prolonged disappearance.
func AStar(start, goal GridNode, solid TileSolidFn) []GridNode {
	if solid(start.X, start.Y) || solid(goal.X, goal.Y) {
		return nil
	}

	open := &astarHeap{}
	heap.Init(open)
	heap.Push(open, &astarNode{pos: start, g: 0, h: heuristic(start, goal)})

	parent := map[GridNode]GridNode{}
	gScore := map[GridNode]float64{start: 0}

	for open.Len() > 0 {
		current := heap.Pop(open).(*astarNode).pos
		if current == goal {
			// Reconstruct.
			var path []GridNode
			at := goal
			for at != start {
				path = append([]GridNode{at}, path...)
				at = parent[at]
			}
			path = append([]GridNode{start}, path...)
			return path
		}

		neighbors := []GridNode{
			{current.X + 1, current.Y},
			{current.X - 1, current.Y},
			{current.X, current.Y + 1},
			{current.X, current.Y - 1},
		}
		for _, nb := range neighbors {
			if solid(nb.X, nb.Y) {
				continue
			}
			tentativeG := gScore[current] + 1
			if existing, ok := gScore[nb]; !ok || tentativeG < existing {
				gScore[nb] = tentativeG
				h := heuristic(nb, goal)
				heap.Push(open, &astarNode{pos: nb, g: tentativeG, h: h})
				parent[nb] = current
			}
		}
	}
	return nil
}

// ─── Four-State Behaviour Machine ────────────────────────────────

// AIState is the current perceptual mode of an enemy.
type AIState int

const (
	// StatePatrol is the start state. The enemy wanders the grid
	// using BFS pathfinding, probing for the player.
	StatePatrol AIState = iota
	// StateChase is entered when the player is spotted via
	// Line-of-Sight. The enemy pursues aggressively.
	StateChase
	// StateSearch is entered when the player escapes LOS. The
	// enemy uses trigonometric tracking to reach the last known
	// position and wait.
	StateSearch
	// StateAlert is entered after 10 seconds without contact. The
	// enemy uses A* for 10 steps and then falls back to Patrol.
	StateAlert
)

// EnemyAI is the brain of a single hostile actor. It holds the
// state machine, positional data, and the last known player
// location.
type EnemyAI struct {
	Pos                Vec2
	Speed              float64
	State              AIState
	LastKnownPlayerPos Vec2
	AlertTimer         float64 // seconds since last sighting
	AlertStepCounter   int     // steps taken in Alert Mode
	TileSize           int
	Solid              TileSolidFn
}

// NewEnemyAI creates an enemy at the given spawn position.
func NewEnemyAI(pos Vec2, speed float64, tileSize int, solid TileSolidFn) *EnemyAI {
	return &EnemyAI{
		Pos:      pos,
		Speed:    speed,
		TileSize: tileSize,
		Solid:    solid,
		State:    StatePatrol,
	}
}

// Update drives the state machine for one tick (dt in seconds).
// It expects the current player position so that LOS and distance
// checks can be evaluated.
func (e *EnemyAI) Update(player Vec2, dt float64) {
	switch e.State {
	case StatePatrol:
		e.updatePatrol(player, dt)
	case StateChase:
		e.updateChase(player, dt)
	case StateSearch:
		e.updateSearch(player, dt)
	case StateAlert:
		e.updateAlert(player, dt)
	}
}

func (e *EnemyAI) updatePatrol(player Vec2, dt float64) {
	if LineOfSight(e.Pos, player, e.TileSize, e.Solid) {
		e.State = StateChase
		e.LastKnownPlayerPos = player
		e.AlertTimer = 0
		return
	}
	// Wander using BFS toward a random patrol waypoint.
	// In a full implementation the waypoint would be selected
	// from a designer-authored patrol route.
}

func (e *EnemyAI) updateChase(player Vec2, dt float64) {
	if !LineOfSight(e.Pos, player, e.TileSize, e.Solid) {
		// Player broke line of sight—switch to search.
		e.State = StateSearch
		e.AlertTimer = 0
		return
	}
	e.LastKnownPlayerPos = player
	// Direct pursuit using coordinate tracking for grid-aligned
	// movement; trigonometry tracking is substituted for
	// free-roaming enemies.
	e.Pos = CoordinateTracker(e.Pos, player, e.Speed)
}

func (e *EnemyAI) updateSearch(player Vec2, dt float64) {
	e.AlertTimer += dt
	if LineOfSight(e.Pos, player, e.TileSize, e.Solid) {
		// Player reacquired during search.
		e.State = StateChase
		e.AlertTimer = 0
		return
	}
	// Move toward last known position using trigonometry.
	e.Pos = TrigonometryTracker(e.Pos, e.LastKnownPlayerPos, e.Speed)
	if Distance(e.Pos, e.LastKnownPlayerPos) < float64(e.TileSize) {
		// Reached the last known spot without reacquiring.
		if e.AlertTimer >= 10.0 {
			e.State = StateAlert
			e.AlertStepCounter = 0
		}
	}
}

func (e *EnemyAI) updateAlert(player Vec2, dt float64) {
	if LineOfSight(e.Pos, player, e.TileSize, e.Solid) {
		e.State = StateChase
		e.AlertTimer = 0
		return
	}
	if e.AlertStepCounter < 10 {
		// Use A* toward the player's last known tile.
		start := GridNode{int(e.Pos.X) / e.TileSize, int(e.Pos.Y) / e.TileSize}
		goal := GridNode{int(e.LastKnownPlayerPos.X) / e.TileSize, int(e.LastKnownPlayerPos.Y) / e.TileSize}
		path := AStar(start, goal, e.Solid)
		if len(path) > 1 {
			next := path[1]
			e.Pos = Vec2{
				X: float64(next.X*e.TileSize) + float64(e.TileSize)/2,
				Y: float64(next.Y*e.TileSize) + float64(e.TileSize)/2,
			}
		}
		e.AlertStepCounter++
	} else {
		// After 10 A* steps with no contact, fall back to patrol.
		e.State = StatePatrol
		e.AlertTimer = 0
		e.AlertStepCounter = 0
	}
}
