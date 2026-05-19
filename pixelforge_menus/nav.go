package pixelforge_menus

// Nav coordinates the cursor movement + selection state for the
// top menu on the stack. The runtime wires input intents to
// Nav.OnDpadUp/Down/Use; Nav clamps the cursor to the active
// template's SelectionCount and reports the selected option index
// when the player presses A.

// OnDpadUp moves the cursor up (decrement, wrapping to last).
// Returns the new cursor position.
func (s *MenuStack) OnDpadUp() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.entries)
	if n == 0 {
		return 0
	}
	top := &s.entries[n-1]
	count := s.activeSelectionCountLocked()
	if count == 0 {
		top.Cursor = 0
		return 0
	}
	top.Cursor = (top.Cursor - 1 + count) % count
	return top.Cursor
}

// OnDpadDown moves the cursor down (increment, wrapping to 0).
// Returns the new cursor position.
func (s *MenuStack) OnDpadDown() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.entries)
	if n == 0 {
		return 0
	}
	top := &s.entries[n-1]
	count := s.activeSelectionCountLocked()
	if count == 0 {
		top.Cursor = 0
		return 0
	}
	top.Cursor = (top.Cursor + 1) % count
	return top.Cursor
}

// OnUse invokes the active menu's Apply on the cursor. Returns
// (verbName, verbArgs, true) for the host engine to dispatch. The
// host typically then calls Pop to close the menu if Apply
// represents a transition.
func (s *MenuStack) OnUse() (string, map[string]any, bool) {
	if s == nil {
		return "", nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.entries)
	if n == 0 {
		return "", nil, false
	}
	top := s.entries[n-1]
	tmpl, ok := LookupTemplate(top.TemplateName)
	if !ok || tmpl.Apply == nil {
		return "", nil, false
	}
	verb, args := tmpl.Apply(top.Cursor, top.Parameters)
	return verb, args, verb != ""
}

// activeSelectionCountLocked looks up the active menu's
// SelectionCount. Caller must hold mu.
func (s *MenuStack) activeSelectionCountLocked() int {
	n := len(s.entries)
	if n == 0 {
		return 0
	}
	top := s.entries[n-1]
	tmpl, ok := LookupTemplate(top.TemplateName)
	if !ok || tmpl.SelectionCount == nil {
		return 0
	}
	return tmpl.SelectionCount(top.Parameters)
}

// Cursor returns the top menu's current cursor position.
func (s *MenuStack) Cursor() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.entries)
	if n == 0 {
		return 0
	}
	return s.entries[n-1].Cursor
}
