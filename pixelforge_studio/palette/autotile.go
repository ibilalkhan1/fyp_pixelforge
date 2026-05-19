package palette

import (
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// AutoTileRuleSynth observes paint strokes and synthesizes transition
// rules using LDtk-style neighbor-pattern matching. When the same 3×3
// pattern is painted twice, the synth promotes it into an active rule
// that auto-applies on subsequent strokes producing the same pattern.
type AutoTileRuleSynth struct{}

// NewAutoTileRuleSynth returns a fresh synth (it carries no per-call
// state — rules live on the TileAtlas).
func NewAutoTileRuleSynth() *AutoTileRuleSynth { return &AutoTileRuleSynth{} }

// PaintCell is one tile-coordinate + value pair captured during a paint
// stroke. The painter accumulates these into a slice for the stroke
// then hands them to RecordStroke on stroke end.
type PaintCell struct {
	X, Y  int
	Value int
}

// RecordStroke analyses each painted cell's 3×3 neighborhood (on the
// post-paint grid) and either increments an existing rule's count or
// records the pattern as a new rule. Patterns whose count reaches
// AutoTileActivationThreshold become active and apply on subsequent
// strokes. Kept for callers that don't care about promotion events;
// new code should prefer RecordStrokeWithPromotions.
func (s *AutoTileRuleSynth) RecordStroke(layer *pixelforge_project.TileAtlas, stroke []PaintCell) {
	_ = s.RecordStrokeWithPromotions(layer, stroke)
}

// PromotedRule names a rule that crossed AutoTileActivationThreshold
// during the stroke just observed. The studio surfaces promotions
// through the toast UX (idea #2 v1 U7) so designers can opt the rule
// into session-active state. RuleIndex points into
// layer.AutoTileRules at the time of the call; the slice never
// shrinks within a session so the index is stable.
type PromotedRule struct {
	RuleIndex int
	Pattern   [9]int
	Output    int
}

// RecordStrokeWithPromotions runs the existing rule-counting logic
// and additionally reports which rules crossed
// AutoTileActivationThreshold during this stroke. A rule only
// promotes once per session — subsequent strokes that touch the same
// rule keep incrementing Count but do not re-surface.
//
// Implementation: capture each rule's pre-stroke Count, run the
// observation, then collect rules whose Count crossed the threshold.
// A pattern that lands at the threshold for the first time in a
// stroke surfaces as one PromotedRule entry. New rules introduced by
// this stroke that already happen to land at >= threshold (cannot
// happen given the increment-by-one logic, but defensive) also count
// as a promotion.
func (s *AutoTileRuleSynth) RecordStrokeWithPromotions(layer *pixelforge_project.TileAtlas, stroke []PaintCell) []PromotedRule {
	if layer == nil || len(stroke) == 0 {
		return nil
	}
	if len(stroke) == 1 {
		return nil
	}

	// Snapshot pre-stroke counts so the post-stroke diff identifies
	// fresh promotions. The key is (Pattern, Output); we store the
	// pre-stroke Count for each.
	pre := make(map[ruleKey]int, len(layer.AutoTileRules))
	for _, r := range layer.AutoTileRules {
		pre[keyOf(r.Pattern, r.Output)] = r.Count
	}

	for _, cell := range stroke {
		pattern := neighborhood(layer, cell.X, cell.Y)
		// Skip if the cell has no painted neighbors at all (the 3×3
		// pattern is just one cell). This is the plan's "single tile
		// does not trigger rule synthesis" edge case extended to mean
		// "no neighbor signal".
		if !patternHasNeighbors(pattern) {
			continue
		}
		s.recordRule(layer, pattern, cell.Value)
	}

	var promotions []PromotedRule
	for i, r := range layer.AutoTileRules {
		preCount := pre[keyOf(r.Pattern, r.Output)]
		if preCount < AutoTileActivationThreshold &&
			r.Count >= AutoTileActivationThreshold {
			promotions = append(promotions, PromotedRule{
				RuleIndex: i,
				Pattern:   r.Pattern,
				Output:    r.Output,
			})
		}
	}
	return promotions
}

// ruleKey identifies a rule by its content (Pattern + Output) so the
// pre-/post-stroke diff doesn't depend on slice indices that may
// shift if the synth ever reorders rules.
type ruleKey struct {
	Pattern [9]int
	Output  int
}

func keyOf(p [9]int, out int) ruleKey { return ruleKey{Pattern: p, Output: out} }

// Apply looks up an active rule for the given 3×3 pattern at (x, y)
// and returns the synthesised tile + true. Returns (0, false) when no
// rule matches.
func (s *AutoTileRuleSynth) Apply(layer *pixelforge_project.TileAtlas, x, y int) (int, bool) {
	if layer == nil {
		return 0, false
	}
	pattern := neighborhood(layer, x, y)
	for _, rule := range layer.AutoTileRules {
		if rule.Count < AutoTileActivationThreshold {
			continue
		}
		if patternMatches(rule.Pattern, pattern) {
			return rule.Output, true
		}
	}
	return 0, false
}

// recordRule updates the count of an existing matching rule or
// appends a new entry.
func (s *AutoTileRuleSynth) recordRule(layer *pixelforge_project.TileAtlas, pattern [9]int, output int) {
	for i := range layer.AutoTileRules {
		if layer.AutoTileRules[i].Pattern == pattern && layer.AutoTileRules[i].Output == output {
			layer.AutoTileRules[i].Count++
			return
		}
	}
	layer.AutoTileRules = append(layer.AutoTileRules, pixelforge_project.AutoTileRule{
		Pattern: pattern,
		Output:  output,
		Count:   1,
	})
}

// neighborhood returns the 3×3 window centered on (x, y) of layer.
// Out-of-range cells are -1 (wildcard / missing).
func neighborhood(layer *pixelforge_project.TileAtlas, x, y int) [9]int {
	var out [9]int
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			out[(dy+1)*3+(dx+1)] = cellAt(layer, x+dx, y+dy)
		}
	}
	return out
}

// cellAt returns the tile value at (x, y) or -1 outside the grid.
func cellAt(layer *pixelforge_project.TileAtlas, x, y int) int {
	if y < 0 || y >= len(layer.Grid) {
		return -1
	}
	row := layer.Grid[y]
	if x < 0 || x >= len(row) {
		return -1
	}
	return row[x]
}

// patternMatches reports whether candidate matches rule (treating -1
// in rule as a wildcard).
func patternMatches(rule, candidate [9]int) bool {
	for i := range rule {
		if rule[i] == -1 {
			continue
		}
		if rule[i] != candidate[i] {
			return false
		}
	}
	return true
}

// patternHasNeighbors reports whether at least one non-center cell is
// non-empty in the pattern.
func patternHasNeighbors(p [9]int) bool {
	for i, v := range p {
		if i == 4 { // center
			continue
		}
		if v >= 0 {
			return true
		}
	}
	return false
}

// AutoTileActivationThreshold is the count a rule must reach before
// the synth applies it. v1 of idea #2 aligns this with the
// docs/solutions/auto-tile-heuristic.md invariant: the third
// matching stroke is what promotes a pattern to active. Bumped from
// 2 to 3 with U4 so the heuristic doc and the code agree on a
// single source of truth.
const AutoTileActivationThreshold = 3
