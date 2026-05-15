package widgets

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
)

// FilePickerMode mirrors the native bank's pick mode.
type FilePickerMode int

const (
	// PickOpen lets the user choose an existing file.
	PickOpen FilePickerMode = iota
	// PickSave lets the user type a filename.
	PickSave
	// PickDir lets the user choose a directory.
	PickDir
)

// FilePickerOptions configures a FilePicker.
type FilePickerOptions struct {
	StartPath  string
	Mode       FilePickerMode
	Extensions []string
	Title      string
	ShowHidden bool

	OnConfirm func(path string)
	OnCancel  func()
}

// FilePicker is the canvas-resident file/directory chooser. Mirrors
// the native bank's behaviour: scrollable directory list, breadcrumb,
// save-mode name input.
type FilePicker struct {
	X, Y, W, H int

	opts FilePickerOptions

	visible     bool
	current     string
	entries     []fileEntry
	selectedIdx int
	scrollOff   int

	saveName    []rune
	nameFocused bool

	// hit rects populated each Draw.
	listRect    IntRect
	upRect      IntRect
	nameRect    IntRect
	cancelRect  IntRect
	confirmRect IntRect
}

type fileEntry struct {
	Name  string
	IsDir bool
	Err   error
}

// NewFilePicker builds a hidden picker.
func NewFilePicker() *FilePicker {
	return &FilePicker{}
}

// SetBounds positions the picker (typically full-canvas or workspace-area).
func (p *FilePicker) SetBounds(x, y, w, h int) {
	p.X, p.Y, p.W, p.H = x, y, w, h
}

// Open shows the picker rooted at opts.StartPath. Falls back to $HOME
// when StartPath is empty or unreadable.
func (p *FilePicker) Open(opts FilePickerOptions) {
	p.opts = opts
	start := opts.StartPath
	if start == "" {
		start, _ = os.UserHomeDir()
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		abs = start
	}
	if _, err := os.Stat(abs); err != nil {
		abs, _ = os.UserHomeDir()
	}
	p.current = abs
	p.refreshEntries()
	p.visible = true
	p.selectedIdx = 0
	p.scrollOff = 0
	p.saveName = []rune{}
	p.nameFocused = false
}

// Close dismisses the picker. Equivalent to user pressing Cancel.
func (p *FilePicker) Close() {
	p.visible = false
}

// Visible reports whether the picker is shown.
func (p *FilePicker) Visible() bool { return p.visible }

// Current returns the directory currently being browsed.
func (p *FilePicker) Current() string { return p.current }

func (p *FilePicker) refreshEntries() {
	p.entries = nil
	d, err := os.Open(p.current)
	if err != nil {
		p.entries = []fileEntry{{Name: err.Error(), Err: err}}
		return
	}
	defer d.Close()
	infos, err := d.Readdir(-1)
	if err != nil {
		p.entries = []fileEntry{{Name: err.Error(), Err: err}}
		return
	}
	for _, info := range infos {
		name := info.Name()
		if !p.opts.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}
		if !info.IsDir() && !p.matchesExtension(name) {
			continue
		}
		p.entries = append(p.entries, fileEntry{Name: name, IsDir: info.IsDir()})
	}
	sort.Slice(p.entries, func(i, j int) bool {
		if p.entries[i].IsDir != p.entries[j].IsDir {
			return p.entries[i].IsDir
		}
		return p.entries[i].Name < p.entries[j].Name
	})
}

func (p *FilePicker) matchesExtension(name string) bool {
	if p.opts.Mode == PickDir {
		return false
	}
	if len(p.opts.Extensions) == 0 {
		return true
	}
	for _, ext := range p.opts.Extensions {
		if strings.HasSuffix(strings.ToLower(name), strings.ToLower(ext)) {
			return true
		}
	}
	return false
}

// Up navigates to the parent directory.
func (p *FilePicker) Up() {
	parent := filepath.Dir(p.current)
	if parent == p.current {
		return
	}
	p.current = parent
	p.refreshEntries()
}

// Enter descends into the directory at idx, or selects the file there.
func (p *FilePicker) Enter(idx int) string {
	if idx < 0 || idx >= len(p.entries) {
		return ""
	}
	e := p.entries[idx]
	full := filepath.Join(p.current, e.Name)
	if e.IsDir {
		p.current = full
		p.refreshEntries()
		return ""
	}
	return full
}

// Confirm fires OnConfirm with the user's selection.
func (p *FilePicker) Confirm() {
	var path string
	switch p.opts.Mode {
	case PickSave:
		name := strings.TrimSpace(string(p.saveName))
		if name == "" {
			return // OK is disabled when name is empty
		}
		if len(p.opts.Extensions) > 0 && !strings.HasSuffix(strings.ToLower(name), strings.ToLower(p.opts.Extensions[0])) {
			name = name + p.opts.Extensions[0]
		}
		path = filepath.Join(p.current, name)
	case PickDir:
		path = p.current
	default:
		if p.selectedIdx >= 0 && p.selectedIdx < len(p.entries) && !p.entries[p.selectedIdx].IsDir {
			path = filepath.Join(p.current, p.entries[p.selectedIdx].Name)
		}
	}
	if path == "" {
		return
	}
	p.visible = false
	if p.opts.OnConfirm != nil {
		p.opts.OnConfirm(path)
	}
}

// Cancel dismisses the picker.
func (p *FilePicker) Cancel() {
	p.visible = false
	if p.opts.OnCancel != nil {
		p.opts.OnCancel()
	}
}

// HandleClick processes a click at (px, py). Returns true when consumed.
func (p *FilePicker) HandleClick(px, py int) bool {
	if !p.visible {
		return false
	}
	if p.upRect.Contains(px, py) {
		p.Up()
		return true
	}
	if p.cancelRect.Contains(px, py) {
		p.Cancel()
		return true
	}
	if p.confirmRect.Contains(px, py) {
		p.Confirm()
		return true
	}
	if p.nameRect.W > 0 && p.nameRect.Contains(px, py) {
		p.nameFocused = true
		return true
	}
	if p.listRect.Contains(px, py) {
		idx := (py-p.listRect.Y)/rowHeight + p.scrollOff
		if idx >= 0 && idx < len(p.entries) {
			result := p.Enter(idx)
			if result != "" && p.opts.Mode != PickDir {
				p.selectedIdx = idx
				if p.opts.Mode == PickOpen {
					p.Confirm()
				}
			}
		}
		return true
	}
	return true // consume any click while visible
}

// HandleEscape dismisses the picker.
func (p *FilePicker) HandleEscape() bool {
	if !p.visible {
		return false
	}
	p.Cancel()
	return true
}

// SetSaveName replaces the typed filename buffer (PickSave mode).
func (p *FilePicker) SetSaveName(s string) {
	p.saveName = []rune(s)
}

// SaveName returns the current typed filename buffer.
func (p *FilePicker) SaveName() string { return string(p.saveName) }

const rowHeight = 14

// Draw paints the picker. Caller must position via SetBounds beforehand.
func (p *FilePicker) Draw() {
	if !p.visible {
		return
	}
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	// Backdrop dim.
	pixelforge.SetColor(0)
	pixelforge.RectFill(p.X, p.Y, p.X+p.W-1, p.Y+p.H-1)

	// Body — 80% of canvas.
	bodyW := p.W * 4 / 5
	bodyH := p.H * 4 / 5
	bodyX := p.X + (p.W-bodyW)/2
	bodyY := p.Y + (p.H-bodyH)/2

	pixelforge.SetColor(2)
	pixelforge.RectFill(bodyX, bodyY, bodyX+bodyW-1, bodyY+bodyH-1)
	pixelforge.SetColor(6)
	pixelforge.Rect(bodyX, bodyY, bodyX+bodyW-1, bodyY+bodyH-1)

	// Title.
	pixelforge.SetColor(7)
	pixelforge_cofont.Print(p.opts.Title, bodyX+8, bodyY+6)

	// Breadcrumb.
	pixelforge_cofont.Print(p.current, bodyX+8, bodyY+18)

	// Up button (top-right of header).
	p.upRect = IntRect{X: bodyX + bodyW - 40, Y: bodyY + 4, W: 32, H: 14}
	pixelforge.SetColor(5)
	pixelforge.RectFill(p.upRect.X, p.upRect.Y, p.upRect.X+p.upRect.W-1, p.upRect.Y+p.upRect.H-1)
	pixelforge.SetColor(7)
	pixelforge_cofont.Print("..", p.upRect.X+12, p.upRect.Y+3)

	// Footer (cancel + confirm + optional name input).
	footerH := 24
	footerY := bodyY + bodyH - footerH

	// Name input (PickSave only).
	if p.opts.Mode == PickSave {
		p.nameRect = IntRect{X: bodyX + 8, Y: footerY - 22, W: bodyW - 16, H: 18}
		pixelforge.SetColor(1)
		pixelforge.RectFill(p.nameRect.X, p.nameRect.Y, p.nameRect.X+p.nameRect.W-1, p.nameRect.Y+p.nameRect.H-1)
		pixelforge.SetColor(7)
		pixelforge_cofont.Print(string(p.saveName), p.nameRect.X+4, p.nameRect.Y+5)
	} else {
		p.nameRect = IntRect{}
	}

	// Buttons.
	btnW := 64
	btnH := 18
	p.confirmRect = IntRect{X: bodyX + bodyW - btnW - 8, Y: footerY + 3, W: btnW, H: btnH}
	p.cancelRect = IntRect{X: p.confirmRect.X - btnW - 8, Y: footerY + 3, W: btnW, H: btnH}

	// Confirm: dim if save-mode name is empty.
	confirmActive := true
	if p.opts.Mode == PickSave && strings.TrimSpace(string(p.saveName)) == "" {
		confirmActive = false
	}
	confirmColor := pixelforge.Color(12)
	if !confirmActive {
		confirmColor = pixelforge.Color(5)
	}
	pixelforge.SetColor(confirmColor)
	pixelforge.RectFill(p.confirmRect.X, p.confirmRect.Y, p.confirmRect.X+btnW-1, p.confirmRect.Y+btnH-1)
	pixelforge.SetColor(5)
	pixelforge.RectFill(p.cancelRect.X, p.cancelRect.Y, p.cancelRect.X+btnW-1, p.cancelRect.Y+btnH-1)

	pixelforge.SetColor(7)
	pixelforge_cofont.Print("OK", p.confirmRect.X+btnW/2-4, p.confirmRect.Y+5)
	pixelforge_cofont.Print("Cancel", p.cancelRect.X+8, p.cancelRect.Y+5)

	// List region.
	listY := bodyY + 32
	listH := footerY - listY - 6
	if p.opts.Mode == PickSave {
		listH = p.nameRect.Y - listY - 4
	}
	p.listRect = IntRect{X: bodyX + 4, Y: listY, W: bodyW - 8, H: listH}
	pixelforge.SetColor(1)
	pixelforge.RectFill(p.listRect.X, p.listRect.Y, p.listRect.X+p.listRect.W-1, p.listRect.Y+p.listRect.H-1)

	visibleRows := p.listRect.H / rowHeight
	end := p.scrollOff + visibleRows
	if end > len(p.entries) {
		end = len(p.entries)
	}
	for i := p.scrollOff; i < end; i++ {
		e := p.entries[i]
		rowY := p.listRect.Y + (i-p.scrollOff)*rowHeight
		if i == p.selectedIdx {
			pixelforge.SetColor(12)
			pixelforge.RectFill(p.listRect.X, rowY, p.listRect.X+p.listRect.W-1, rowY+rowHeight-1)
		}
		pixelforge.SetColor(7)
		name := e.Name
		if e.IsDir {
			name = name + "/"
		}
		pixelforge_cofont.Print(name, p.listRect.X+4, rowY+3)
	}
}
