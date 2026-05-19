// session_state.go owns the in-memory session-rule suppression map
// idea #2 v1 U7 introduces. Designers who pick "No" on an auto-rule
// toast insert the rule's (Pattern, Output) signature here; future
// strokes that re-promote the same rule consult the map and stay
// silent for the rest of the session. The map is never persisted —
// reopening a project gives the designer a fresh chance to accept
// every rule the synth has learned.
package editor

// suppressRule inserts a promotion's (Pattern, Output) signature
// into the session-suppression map. Subsequent calls to isSuppressed
// for the same signature return true so the toast queue silently
// drops re-promotions.
func (e *Editor) suppressRule(p AutoTilePromotion) {
	if e == nil {
		return
	}
	if e.sessionRuleSuppression == nil {
		e.sessionRuleSuppression = map[ruleSignature]struct{}{}
	}
	e.sessionRuleSuppression[ruleSignature{Pattern: p.Pattern, Output: p.Output}] = struct{}{}
}

// isSuppressed reports whether a promotion's signature is in the
// session-suppression map.
func (e *Editor) isSuppressed(p AutoTilePromotion) bool {
	if e == nil || e.sessionRuleSuppression == nil {
		return false
	}
	_, ok := e.sessionRuleSuppression[ruleSignature{Pattern: p.Pattern, Output: p.Output}]
	return ok
}

// SuppressedRuleCount reports how many rules the designer has
// silenced this session. Exposed for tests + diagnostics.
func (e *Editor) SuppressedRuleCount() int {
	if e == nil {
		return 0
	}
	return len(e.sessionRuleSuppression)
}

// ResetSessionRuleSuppression clears the suppression map. Called
// when a new project loads so the new session starts clean.
func (e *Editor) ResetSessionRuleSuppression() {
	if e == nil {
		return
	}
	e.sessionRuleSuppression = nil
}
