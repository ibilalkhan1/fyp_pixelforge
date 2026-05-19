package pixelforge_menus

import (
	"sync"

	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
)

// StackEntry is one menu instance on the stack — the template
// name + the per-instance parameter map + the current cursor
// position. Top of stack is index len-1.
type StackEntry struct {
	MenuName     string
	TemplateName string
	Parameters   map[string]any
	Cursor       int
}

// MenuStack owns the runtime's open menus. Push opens a menu;
// Pop closes the top one. Overlay menus pause the underlying
// scene via piloop.Pause / Resume coordination — the stack
// tracks an overlay refcount so multiple overlays compose
// correctly.
type MenuStack struct {
	mu             sync.Mutex
	entries        []StackEntry
	overlayDepth   int
}

// NewMenuStack returns an empty stack.
func NewMenuStack() *MenuStack { return &MenuStack{} }

// Push opens a menu. Returns the resulting stack entry so the
// caller can record the cursor position. Overlay-kind menus
// trigger piloop.Pause() when the first one pushes.
func (s *MenuStack) Push(menuName, templateName string, params map[string]any) StackEntry {
	if s == nil {
		return StackEntry{}
	}
	entry := StackEntry{
		MenuName:     menuName,
		TemplateName: templateName,
		Parameters:   params,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entry)
	if isOverlayTemplate(templateName) {
		s.overlayDepth++
		if s.overlayDepth == 1 {
			piloop.Pause()
		}
	}
	return entry
}

// Pop closes the top menu. Returns the popped entry + true; or
// the zero entry + false when the stack was empty. Overlay-kind
// menus that bring the overlay refcount to zero trigger
// piloop.Resume.
func (s *MenuStack) Pop() (StackEntry, bool) {
	if s == nil {
		return StackEntry{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.entries)
	if n == 0 {
		return StackEntry{}, false
	}
	top := s.entries[n-1]
	s.entries = s.entries[:n-1]
	if isOverlayTemplate(top.TemplateName) {
		s.overlayDepth--
		if s.overlayDepth == 0 {
			piloop.Resume()
		}
	}
	return top, true
}

// Top returns the top-of-stack entry. Returns the zero entry +
// false on an empty stack.
func (s *MenuStack) Top() (StackEntry, bool) {
	if s == nil {
		return StackEntry{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.entries) == 0 {
		return StackEntry{}, false
	}
	return s.entries[len(s.entries)-1], true
}

// Depth returns the number of menus on the stack.
func (s *MenuStack) Depth() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

// Entries returns a copy of the stack from bottom to top.
func (s *MenuStack) Entries() []StackEntry {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]StackEntry, len(s.entries))
	copy(out, s.entries)
	return out
}

// Clear pops everything. Used at scene change. Triggers Resume
// when overlay-depth reaches zero.
func (s *MenuStack) Clear() {
	if s == nil {
		return
	}
	for {
		_, ok := s.Pop()
		if !ok {
			return
		}
	}
}

// isOverlayTemplate dispatches into the registry to ask whether
// the template is an overlay. Unknown templates default to full-
// screen (no pause coordination).
func isOverlayTemplate(name string) bool {
	t, ok := LookupTemplate(name)
	if !ok {
		return false
	}
	return t.Kind == KindOverlay
}
