package pixelforge_loop

type Event string

const (
	EventInit        Event = "init"         // when the game is started, just before the first frame
	EventFrameStart  Event = "frame_start"  // beginning of the frame
	EventUpdate      Event = "update"       // after pixelforge.Update
	EventLateUpdate  Event = "late_update"  // after EventUpdate
	EventDraw        Event = "draw"         // after pixelforge.Draw
	EventLateDraw    Event = "late_draw"    // after EventDraw
	EventWindowClose Event = "window_close" // when a user closes the window (desktop only)
)
