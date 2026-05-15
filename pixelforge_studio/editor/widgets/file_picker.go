package widgets

import (
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// FilePickerMode is the kind of selection the picker requests.
type FilePickerMode int

const (
	// PickOpen lets the user choose an existing file.
	PickOpen FilePickerMode = iota
	// PickSave lets the user type a filename (extension auto-appended).
	PickSave
	// PickDir lets the user choose a directory.
	PickDir
)

// FilePickerOptions configures a FilePicker.
type FilePickerOptions struct {
	StartPath  string
	Mode       FilePickerMode
	Extensions []string // e.g. {".pforge"}. Empty = no filter.
	Title      string
	// ShowHidden, when true, lists files/dirs whose name begins with ".".
	ShowHidden bool

	// OnConfirm is called with the resolved absolute path when the user
	// confirms. The picker dismisses itself after invoking the callback.
	OnConfirm func(path string)

	// OnCancel fires when the user dismisses without picking.
	OnCancel func()
}

// FilePicker is a self-contained in-editor file/directory chooser.
// Renders a centered modal panel with breadcrumb + scrolling list +
// (in PickSave) a filename input + Cancel/Confirm buttons.
type FilePicker struct {
	opts FilePickerOptions

	modal    Modal
	current  string // absolute path of the directory being browsed
	entries  []fileEntry
	listErr  error

	selectedIdx int
	scrollOff   int

	// saveName holds the typed filename when Mode == PickSave.
	saveName []rune

	// nameFocused is true once the user has clicked the name input;
	// typed runes flow into saveName.
	nameFocused bool
}

type fileEntry struct {
	Name  string
	IsDir bool
	// Err marks an entry that could not be opened; rendered with a
	// warning glyph instead of being selectable.
	Err error
}

// NewFilePicker builds a FilePicker ready to be Open()ed.
func NewFilePicker() *FilePicker {
	return &FilePicker{}
}

// Open shows the picker rooted at opts.StartPath. Falls back to the
// user's home directory when StartPath is empty or unreadable.
func (p *FilePicker) Open(opts FilePickerOptions) {
	p.opts = opts
	p.modal.Visible = true
	p.modal.OnDismiss = func() {
		if p.opts.OnCancel != nil {
			p.opts.OnCancel()
		}
	}
	p.saveName = nil
	p.nameFocused = false
	p.selectedIdx = 0
	p.scrollOff = 0
	p.setCurrent(resolveStartPath(opts.StartPath))
}

// Visible reports whether the picker is currently shown.
func (p *FilePicker) Visible() bool { return p.modal.Visible }

// CurrentPath returns the directory the picker is browsing.
func (p *FilePicker) CurrentPath() string { return p.current }

// SetSaveName overrides the filename input. Used by tests; the live
// editor flows runes through Update.
func (p *FilePicker) SetSaveName(name string) {
	p.saveName = []rune(name)
}

// SaveName returns the current Save-mode filename buffer.
func (p *FilePicker) SaveName() string { return string(p.saveName) }

// Entries returns the currently-listed entries (test helper).
func (p *FilePicker) Entries() []fileEntry { return p.entries }

// SelectedEntry returns the highlighted entry pointer (nil if empty).
func (p *FilePicker) SelectedEntry() *fileEntry {
	if p.selectedIdx < 0 || p.selectedIdx >= len(p.entries) {
		return nil
	}
	return &p.entries[p.selectedIdx]
}

// setCurrent reads `dir` and refreshes the entry list. If the read
// fails, the picker stays on the previous directory and surfaces the
// error in the list so the user can navigate elsewhere.
func (p *FilePicker) setCurrent(dir string) {
	dir = filepath.Clean(dir)
	entries, err := readDirEntries(dir, p.opts)
	if err != nil {
		// Keep the previous directory but record the error so the
		// renderer can surface it.
		p.listErr = err
		if p.current == "" {
			p.current = dir
		}
		return
	}
	p.current = dir
	p.entries = entries
	p.listErr = nil
	if p.selectedIdx >= len(entries) {
		p.selectedIdx = 0
	}
}

// readDirEntries reads dir, applies the picker's filters, and returns a
// sorted slice (directories first, then files, both alpha-asc).
func readDirEntries(dir string, opts FilePickerOptions) ([]fileEntry, error) {
	osEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]fileEntry, 0, len(osEntries))
	for _, e := range osEntries {
		name := e.Name()
		if !opts.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}
		isDir := e.IsDir()
		if !isDir && opts.Mode == PickDir {
			continue
		}
		if !isDir && !extensionAllowed(name, opts.Extensions) {
			continue
		}
		out = append(out, fileEntry{Name: name, IsDir: isDir})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

// extensionAllowed reports whether name matches one of allowed
// extensions. An empty allowed list means "no filter".
func extensionAllowed(name string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	for _, a := range allowed {
		if strings.ToLower(a) == ext {
			return true
		}
	}
	return false
}

// resolveStartPath returns a safe absolute directory to open at. Falls
// back to the user's home dir, then the working dir, then "/".
func resolveStartPath(start string) string {
	if start != "" {
		if info, err := os.Stat(start); err == nil {
			if info.IsDir() {
				abs, _ := filepath.Abs(start)
				return abs
			}
			abs, _ := filepath.Abs(filepath.Dir(start))
			return abs
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return string(filepath.Separator)
}

// Navigate moves into a child directory. Public for tests.
func (p *FilePicker) Navigate(name string) {
	p.setCurrent(filepath.Join(p.current, name))
	p.selectedIdx = 0
	p.scrollOff = 0
}

// NavigateUp moves to the parent directory. No-op at filesystem root.
func (p *FilePicker) NavigateUp() {
	parent := filepath.Dir(p.current)
	if parent == p.current {
		return
	}
	p.setCurrent(parent)
	p.selectedIdx = 0
	p.scrollOff = 0
}

// Confirm resolves the current selection / save-name into an absolute
// path and fires OnConfirm. Returns "" if there is nothing to confirm.
func (p *FilePicker) Confirm() string {
	target := p.resolveConfirmTarget()
	if target == "" {
		return ""
	}
	p.modal.Visible = false
	if p.opts.OnConfirm != nil {
		p.opts.OnConfirm(target)
	}
	return target
}

// resolveConfirmTarget is the pure logic of Confirm — useful for tests.
func (p *FilePicker) resolveConfirmTarget() string {
	switch p.opts.Mode {
	case PickSave:
		name := strings.TrimSpace(string(p.saveName))
		if name == "" {
			return ""
		}
		name = ensureExtension(name, p.opts.Extensions)
		return filepath.Join(p.current, name)
	case PickDir:
		// If a directory entry is selected, return its absolute path;
		// otherwise return the current directory.
		if sel := p.SelectedEntry(); sel != nil && sel.IsDir {
			return filepath.Join(p.current, sel.Name)
		}
		return p.current
	default: // PickOpen
		sel := p.SelectedEntry()
		if sel == nil || sel.IsDir {
			return ""
		}
		return filepath.Join(p.current, sel.Name)
	}
}

// ensureExtension appends the first allowed extension if name has no
// extension or has an extension not in the allow-list.
func ensureExtension(name string, allowed []string) string {
	if len(allowed) == 0 {
		return name
	}
	ext := strings.ToLower(filepath.Ext(name))
	for _, a := range allowed {
		if strings.ToLower(a) == ext {
			return name
		}
	}
	return name + allowed[0]
}

// Dismiss closes the picker without confirming.
func (p *FilePicker) Dismiss() {
	if !p.modal.Visible {
		return
	}
	p.modal.Visible = false
	if p.opts.OnCancel != nil {
		p.opts.OnCancel()
	}
}

// Update routes keyboard + mouse input while the picker is open.
// Returns true when the picker is consuming input this frame.
func (p *FilePicker) Update(windowW, windowH int) bool {
	if !p.modal.Visible {
		return false
	}
	mx, my := ebiten.CursorPosition()
	body := p.bodyRect(windowW, windowH)
	p.modal.Body = body

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.Dismiss()
		return true
	}

	// Mouse outside the picker body dismisses (same UX as system dialogs).
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && !body.Contains(mx, my) {
		p.Dismiss()
		return true
	}

	// Arrow keys navigate the list.
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		if p.selectedIdx < len(p.entries)-1 {
			p.selectedIdx++
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		if p.selectedIdx > 0 {
			p.selectedIdx--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && !p.nameFocused {
		p.NavigateUp()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		sel := p.SelectedEntry()
		if sel != nil && sel.IsDir && p.opts.Mode != PickDir {
			p.Navigate(sel.Name)
		} else {
			p.Confirm()
		}
		return true
	}

	// Save-mode text input.
	if p.opts.Mode == PickSave {
		p.saveName = append(p.saveName, ebiten.AppendInputChars(nil)...)
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(p.saveName) > 0 {
			p.saveName = p.saveName[:len(p.saveName)-1]
		}
	}

	// Mouse click on a list row.
	listRect := p.listRect(body)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && listRect.Contains(mx, my) {
		idx := p.scrollOff + (my-listRect.Y)/filePickerRowHeight
		if idx >= 0 && idx < len(p.entries) {
			if p.selectedIdx == idx {
				// Double-tap semantics on simple click → navigate/confirm.
				sel := &p.entries[idx]
				if sel.IsDir && p.opts.Mode != PickDir {
					p.Navigate(sel.Name)
				} else {
					p.Confirm()
				}
			} else {
				p.selectedIdx = idx
			}
		}
	}

	return true
}

// Draw paints the picker. Caller passes the full-window destination
// image; the picker carves its own centered body.
func (p *FilePicker) Draw(dst *ebiten.Image) {
	if !p.modal.Visible {
		return
	}
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	p.modal.DrawBackdrop(dst)
	body := p.bodyRect(w, h)
	p.modal.Body = body

	fillRect(dst, body, colFilePickerBg)
	strokeRect(dst, body, colWidgetBorder)

	// Title bar.
	title := p.opts.Title
	if title == "" {
		switch p.opts.Mode {
		case PickSave:
			title = "Save As"
		case PickDir:
			title = "Choose Directory"
		default:
			title = "Open"
		}
	}
	header := Rect{X: body.X, Y: body.Y, W: body.W, H: 22}
	fillRect(dst, header, colFilePickerHeader)
	printAt(dst, title, header.X+8, header.Y+3)

	// Breadcrumb (current path).
	crumb := Rect{X: body.X, Y: body.Y + 22, W: body.W, H: 18}
	fillRect(dst, crumb, colWidgetBg)
	printAt(dst, p.current, crumb.X+8, crumb.Y+1)

	// List.
	lr := p.listRect(body)
	fillRect(dst, lr, colWidgetBg)
	strokeRect(dst, lr, colWidgetBorder)
	if p.listErr != nil {
		printAt(dst, "error: "+p.listErr.Error(), lr.X+4, lr.Y+4)
	}
	for i := 0; i < lr.H/filePickerRowHeight && p.scrollOff+i < len(p.entries); i++ {
		e := p.entries[p.scrollOff+i]
		row := Rect{X: lr.X, Y: lr.Y + i*filePickerRowHeight, W: lr.W, H: filePickerRowHeight}
		if p.scrollOff+i == p.selectedIdx {
			fillRect(dst, row, colWidgetHover)
		}
		glyph := "  "
		if e.IsDir {
			glyph = "[D]"
		} else if e.Err != nil {
			glyph = "[!]"
		}
		printAt(dst, glyph+" "+e.Name, row.X+6, row.Y+1)
	}

	// Save-name input.
	if p.opts.Mode == PickSave {
		input := Rect{X: body.X + 8, Y: body.Y + body.H - 56, W: body.W - 16, H: 22}
		fillRect(dst, input, colWidgetBg)
		strokeRect(dst, input, colWidgetBorder)
		printAt(dst, "Name: "+string(p.saveName), input.X+4, input.Y+3)
	}

	// Buttons.
	cancel := Rect{X: body.X + body.W - 180, Y: body.Y + body.H - 28, W: 80, H: 22}
	confirm := Rect{X: body.X + body.W - 90, Y: body.Y + body.H - 28, W: 80, H: 22}
	fillRect(dst, cancel, colWidgetBorder)
	printAt(dst, "Cancel", cancel.X+18, cancel.Y+3)
	fillRect(dst, confirm, colWidgetFill)
	confirmLabel := "Open"
	if p.opts.Mode == PickSave {
		confirmLabel = "Save"
	} else if p.opts.Mode == PickDir {
		confirmLabel = "Choose"
	}
	printAt(dst, confirmLabel, confirm.X+22, confirm.Y+3)

	mx, my := ebiten.CursorPosition()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		switch {
		case cancel.Contains(mx, my):
			p.Dismiss()
		case confirm.Contains(mx, my):
			p.Confirm()
		}
	}
}

func (p *FilePicker) bodyRect(windowW, windowH int) Rect {
	bw := 600
	bh := 400
	if bw > windowW-20 {
		bw = windowW - 20
	}
	if bh > windowH-20 {
		bh = windowH - 20
	}
	return Rect{
		X: (windowW - bw) / 2,
		Y: (windowH - bh) / 2,
		W: bw,
		H: bh,
	}
}

func (p *FilePicker) listRect(body Rect) Rect {
	headerH := 22 + 18 // title + breadcrumb
	bottomReserved := 36
	if p.opts.Mode == PickSave {
		bottomReserved += 26
	}
	return Rect{
		X: body.X + 8,
		Y: body.Y + headerH + 4,
		W: body.W - 16,
		H: body.H - headerH - bottomReserved,
	}
}

const filePickerRowHeight = 16

var (
	colFilePickerBg     = color.RGBA{R: 0x1a, G: 0x1a, B: 0x24, A: 0xff}
	colFilePickerHeader = color.RGBA{R: 0x28, G: 0x28, B: 0x34, A: 0xff}
)
