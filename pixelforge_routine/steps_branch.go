package pixelforge_routine

// Branch creates a Routine step that, on first tick, evaluates the
// predicate and delegates the rest of its lifetime to a freshly
// constructed inner routine built from the chosen []Step. Returns
// true once the inner routine completes.
//
// Either substep slice may be nil/empty; an empty selected branch
// completes immediately.
func Branch(predicate func() bool, ifTrue, ifFalse []Step) Step {
	var inner *Routine
	chose := false
	return func() bool {
		if !chose {
			chose = true
			var steps []Step
			if predicate != nil && predicate() {
				steps = ifTrue
			} else {
				steps = ifFalse
			}
			if len(steps) == 0 {
				return true
			}
			inner = New(steps...)
		}
		if inner == nil {
			return true
		}
		// Resume returns true while more work remains. We translate
		// to Step semantics: return true (done) when the inner
		// routine has finished.
		return !inner.Resume()
	}
}
