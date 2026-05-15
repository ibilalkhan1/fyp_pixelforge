package pixelforge_gui

// FocusManager owns keyboard focus for one logical canvas. Widgets that
// participate in Tab traversal register themselves with Register; the
// currently-focused element is returned by Focused.
//
// FocusManager is not concurrency-safe — like the rest of pixelforge_gui
// it expects to be driven by the single update/draw goroutine.
type FocusManager struct {
	order   []*Element
	focused *Element
}

// NewFocusManager creates an empty FocusManager.
func NewFocusManager() *FocusManager {
	return &FocusManager{}
}

// Register adds e to the Tab traversal order. Re-registering an element
// is a no-op so it is safe to call from a widget's constructor.
func (m *FocusManager) Register(e *Element) {
	for _, existing := range m.order {
		if existing == e {
			return
		}
	}
	m.order = append(m.order, e)
}

// Unregister removes e from the traversal order and blurs it when it
// was currently focused.
func (m *FocusManager) Unregister(e *Element) {
	for i, existing := range m.order {
		if existing == e {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
	if m.focused == e {
		m.focused = nil
	}
}

// Focus moves focus to e. e must already be registered, or Focus is a
// no-op (the misuse is silently absorbed because UI code calling Focus
// is rarely in a position to handle errors gracefully).
func (m *FocusManager) Focus(e *Element) {
	if e == nil {
		m.focused = nil
		return
	}
	for _, existing := range m.order {
		if existing == e {
			m.focused = e
			return
		}
	}
}

// Blur clears the current focus.
func (m *FocusManager) Blur() {
	m.focused = nil
}

// Focused returns the currently-focused element, or nil.
func (m *FocusManager) Focused() *Element {
	return m.focused
}

// Tab advances focus to the next or previous element in registration
// order. Wrapping is enabled at both ends.
func (m *FocusManager) Tab(forward bool) {
	if len(m.order) == 0 {
		m.focused = nil
		return
	}
	if m.focused == nil {
		if forward {
			m.focused = m.order[0]
		} else {
			m.focused = m.order[len(m.order)-1]
		}
		return
	}
	idx := -1
	for i, existing := range m.order {
		if existing == m.focused {
			idx = i
			break
		}
	}
	if idx == -1 {
		m.focused = m.order[0]
		return
	}
	if forward {
		idx = (idx + 1) % len(m.order)
	} else {
		idx = (idx - 1 + len(m.order)) % len(m.order)
	}
	m.focused = m.order[idx]
}

// Order returns the registration order. Useful for testing and for
// widgets that need to render focus rings on the active element.
func (m *FocusManager) Order() []*Element {
	return m.order
}
