package widgets

// Draggable is a generic drag-state mixin: a hit region that tracks
// pointer-down → pointer-up and emits OnDrag while pressed. Designed
// to back chrome gutters between panels, but reusable for any drag
// interaction (waveform scroll, behavior-graph pan, etc.).
//
// The mixin doesn't subscribe to mouse events itself — the host
// element calls Press / Move / Release at the right times. That
// keeps Draggable independent of the input layer and trivially
// testable.
type Draggable struct {
	Region IntRect

	OnDragStart func()
	OnDrag      func(dx, dy int)
	OnDragEnd   func()

	pressed      bool
	startX       int
	startY       int
	lastX, lastY int
}

// NewDraggable returns a Draggable bound to the given region.
func NewDraggable(region IntRect) *Draggable {
	return &Draggable{Region: region}
}

// SetRegion repositions the drag hit region.
func (d *Draggable) SetRegion(r IntRect) { d.Region = r }

// Pressed reports whether a drag is currently active.
func (d *Draggable) Pressed() bool { return d.pressed }

// Press registers a pointer-down at (mx, my). Returns true when the
// press started a drag (i.e. the press was inside the region).
func (d *Draggable) Press(mx, my int) bool {
	if !d.Region.Contains(mx, my) {
		return false
	}
	d.pressed = true
	d.startX = mx
	d.startY = my
	d.lastX = mx
	d.lastY = my
	if d.OnDragStart != nil {
		d.OnDragStart()
	}
	return true
}

// Move is called each frame while the pointer is moving. dx/dy of the
// last delta are forwarded to OnDrag if a drag is active.
func (d *Draggable) Move(mx, my int) {
	if !d.pressed {
		return
	}
	dx := mx - d.lastX
	dy := my - d.lastY
	d.lastX = mx
	d.lastY = my
	if d.OnDrag != nil && (dx != 0 || dy != 0) {
		d.OnDrag(dx, dy)
	}
}

// Release ends the drag and fires OnDragEnd. Returns true if a drag
// was active.
func (d *Draggable) Release() bool {
	if !d.pressed {
		return false
	}
	d.pressed = false
	if d.OnDragEnd != nil {
		d.OnDragEnd()
	}
	return true
}

// CumulativeDX returns the horizontal offset from the drag's start.
func (d *Draggable) CumulativeDX() int {
	if !d.pressed {
		return 0
	}
	return d.lastX - d.startX
}

// CumulativeDY returns the vertical offset from the drag's start.
func (d *Draggable) CumulativeDY() int {
	if !d.pressed {
		return 0
	}
	return d.lastY - d.startY
}
