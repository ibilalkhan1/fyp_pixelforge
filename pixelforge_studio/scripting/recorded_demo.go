package scripting

import (
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capture"
)

// SynthesiseFromInputLog turns a recorded-frame range into a
// BehaviorGraph using input-log replay v1: one Wait(n) between
// distinct TickNumbers, one Publish per InputEntry. State-diff
// heuristics are deferred (Scope Boundaries).
//
// startIdx and endIdx are inclusive. Out-of-order endpoints swap;
// out-of-range indices clamp to the available range.
func SynthesiseFromInputLog(frames []*capture.Frame, startIdx, endIdx int) pixelforge_project.BehaviorGraph {
	if len(frames) == 0 {
		return pixelforge_project.BehaviorGraph{Name: "recorded"}
	}
	if startIdx > endIdx {
		startIdx, endIdx = endIdx, startIdx
	}
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= len(frames) {
		endIdx = len(frames) - 1
	}
	if startIdx > endIdx {
		return pixelforge_project.BehaviorGraph{Name: "recorded"}
	}

	graph := pixelforge_project.BehaviorGraph{Name: "recorded"}
	// Walk in tick order. The first frame's tick anchors the wait
	// arithmetic — we emit a leading Wait if the first tick is > 0
	// so the replay matches the original timeline.
	lastTick := -1
	for i := startIdx; i <= endIdx; i++ {
		f := frames[i]
		if f == nil {
			continue
		}
		if lastTick == -1 {
			// First frame in the range: emit a leading Wait of the
			// absolute tick value (e.g. tick=5 → Wait(5) before
			// publishing).
			if f.TickNumber > 0 {
				graph.Steps = append(graph.Steps, pixelforge_project.StepNode{
					Kind: "Wait",
					Args: map[string]any{"ticks": float64(f.TickNumber)},
				})
			}
		} else if f.TickNumber > lastTick {
			delta := f.TickNumber - lastTick
			graph.Steps = append(graph.Steps, pixelforge_project.StepNode{
				Kind: "Wait",
				Args: map[string]any{"ticks": float64(delta)},
			})
		}
		lastTick = f.TickNumber

		for _, in := range f.Inputs {
			graph.Steps = append(graph.Steps, pixelforge_project.StepNode{
				Kind: "Publish",
				Args: map[string]any{
					"target": in.Target,
					"event":  in.Value,
				},
			})
		}
	}
	return graph
}
