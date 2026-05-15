package editor

// Tool enumerates the mouse tools the canvas dispatches on. The set is
// small and rarely changes; a switch in canvas.go reads this directly.
type Tool int

const (
	ToolSelect Tool = iota
	ToolPlace
	ToolDelete
	ToolPaint
)

// String returns a label suitable for the status bar.
func (t Tool) String() string {
	switch t {
	case ToolSelect:
		return "select"
	case ToolPlace:
		return "place"
	case ToolDelete:
		return "delete"
	case ToolPaint:
		return "paint"
	default:
		return "?"
	}
}
