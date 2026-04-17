package editor

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type EditorState struct {
	ProjectPath   string
	Sprites       []SpriteInfo
	AudioFiles    []AudioInfo
	SceneObjects  []SceneObject
	SelectedIndex int
	Scale         float64
	ShowGrid      bool
	ShowCollision bool
	ScreenWidth   int
	ScreenHeight  int
	TPS           int
	UpdateCode    string
	DrawCode      string
}

type SpriteInfo struct {
	Name   string
	Path   string
	Width  int
	Height int
	FrameW int
	FrameH int
	Image  *ebiten.Image
}

type AudioInfo struct {
	Name string
	Path string
}

type SceneObject struct {
	ID            int
	SpriteName    string
	X, Y          int
	Width, Height int
	Layer         int
	Collision     *CollisionBox
	Visible       bool
}

type CollisionBox struct {
	X, Y int
	W, H int
}

type MenuItem struct {
	Label  string
	X, Y   int
	Width  int
	Height int
}

type Editor struct {
	state    *EditorState
	assets   *AssetManager
	menus    []MenuItem
	menuOpen int
	scrollY  int
}

const (
	TitleBarH  = 40
	StatusBarH = 25
	LeftPanelW = 200
)

func NewEditor() *Editor {
	e := &Editor{
		state:    NewEditorState(),
		assets:   NewAssetManager(),
		menus:    make([]MenuItem, 0),
		menuOpen: -1,
	}
	e.setupMenus()
	return e
}

func NewEditorState() *EditorState {
	return &EditorState{
		Sprites:      []SpriteInfo{},
		AudioFiles:   []AudioInfo{},
		SceneObjects: []SceneObject{},
		Scale:        1.0,
		ShowGrid:     true,
		ScreenWidth:  320,
		ScreenHeight: 180,
		TPS:          30,
		UpdateCode:   defaultUpdateCode,
		DrawCode:     defaultDrawCode,
	}
}

const (
	defaultUpdateCode = `func update() {
	// Handle input with pixelforge_key, pixelforge_pad, pixelforge_mouse
	// Add your game logic here
}`

	defaultDrawCode = `func draw() {
	// Add your drawing code here
	pixelforge.Screen().Clear(0)
}`
)

func (e *Editor) setupMenus() {
	menuItems := []string{"File", "Edit", "View", "Project", "Help"}
	xPos := 10
	for _, label := range menuItems {
		e.menus = append(e.menus, MenuItem{
			Label:  label,
			X:      xPos,
			Y:      10,
			Width:  50,
			Height: 20,
		})
		xPos += 55
	}
}

func (e *Editor) Update() error {
	mx, my := ebiten.CursorPosition()

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if my < TitleBarH && mx < len(e.menus)*55 {
			for i, m := range e.menus {
				if mx >= m.X && mx < m.X+m.Width && my >= m.Y && my < m.Y+m.Height {
					e.menuOpen = i
					return nil
				}
			}
			e.menuOpen = -1
		} else {
			e.menuOpen = -1
		}
	}

	if e.menuOpen >= 0 {
		menuY := TitleBarH
		var items []string
		switch e.menuOpen {
		case 0:
			items = []string{"New", "Open", "Save", "Import Sprite", "Import Audio", "Export", "Exit"}
		case 1:
			items = []string{"Delete", "Duplicate"}
		case 2:
			items = []string{"Zoom In", "Zoom Out", "Toggle Grid", "Toggle Collision"}
		case 3:
			items = []string{"Run Preview", "Settings"}
		case 4:
			items = []string{"Documentation", "About"}
		}

		for range items {
			if mx >= 5 && mx <= 195 && my >= menuY && my < menuY+20 {
				e.handleMenuAction(e.menuOpen)
				e.menuOpen = -1
				return nil
			}
			menuY += 25
		}

		if mx < 0 || mx > LeftPanelW || my < TitleBarH || my > 755 {
			e.menuOpen = -1
		}
	}

	return nil
}

func (e *Editor) handleMenuAction(menuIdx int) {
	switch menuIdx {
	case 0: // File
		e.state = NewEditorState()
		e.assets = NewAssetManager()
	case 2: // View
		if e.state.Scale < 4 {
			e.state.Scale += 0.5
		}
	case 3:
		// Project - would run preview
	}
}

func (e *Editor) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 30, G: 30, B: 35, A: 255})

	// Title bar
	screen.SubImage(image.Rect(0, 0, 1280, TitleBarH)).(*ebiten.Image).Fill(color.RGBA{R: 45, G: 45, B: 50, A: 255})

	// Left panel (project/sprites list)
	screen.SubImage(image.Rect(0, TitleBarH, LeftPanelW, 755)).(*ebiten.Image).Fill(color.RGBA{R: 40, G: 40, B: 45, A: 255})

	// Right panel (properties)
	screen.SubImage(image.Rect(1080, TitleBarH, 1280, 755)).(*ebiten.Image).Fill(color.RGBA{R: 40, G: 40, B: 45, A: 255})

	// Center (scene canvas)
	screen.SubImage(image.Rect(LeftPanelW, TitleBarH, 1080, 755)).(*ebiten.Image).Fill(color.RGBA{R: 25, G: 25, B: 30, A: 255})

	// Status bar
	screen.SubImage(image.Rect(0, 755, 1280, 800)).(*ebiten.Image).Fill(color.RGBA{R: 35, G: 35, B: 40, A: 255})

	// Draw menu buttons
	for _, m := range e.menus {
		bg := color.RGBA{R: 60, G: 60, B: 70, A: 255}
		if e.menuOpen >= 0 && m.X == e.menus[e.menuOpen].X {
			bg = color.RGBA{R: 80, G: 80, B: 90, A: 255}
		}
		screen.SubImage(image.Rect(m.X, m.Y, m.X+m.Width, m.Y+m.Height)).(*ebiten.Image).Fill(bg)
	}

	// Draw dropdown menu when open
	if e.menuOpen >= 0 {
		screen.SubImage(image.Rect(0, TitleBarH, 200, 500)).(*ebiten.Image).Fill(color.RGBA{R: 45, G: 45, B: 50, A: 255})

		var items []string
		switch e.menuOpen {
		case 0:
			items = []string{"New Project", "Open Project", "Save Project", "Import Sprite", "Import Audio", "Export Game"}
		case 1:
			items = []string{"Delete", "Duplicate"}
		case 2:
			items = []string{"Zoom In", "Zoom Out", "Toggle Grid", "Toggle Collision"}
		case 3:
			items = []string{"Run Preview", "Settings"}
		case 4:
			items = []string{"Documentation", "About"}
		}

		menuY := TitleBarH + 5
		for range items {
			screen.SubImage(image.Rect(5, menuY, 195, menuY+20)).(*ebiten.Image).Fill(color.RGBA{R: 50, G: 50, B: 55, A: 255})
			menuY += 25
		}
	}

	// Draw sprites list in left panel
	listY := 50
	for i := range e.state.Sprites {
		bg := color.RGBA{R: 50, G: 50, B: 55, A: 255}
		if i == e.state.SelectedIndex {
			bg = color.RGBA{R: 70, G: 70, B: 80, A: 255}
		}
		screen.SubImage(image.Rect(5, listY, 195, listY+20)).(*ebiten.Image).Fill(bg)
		listY += 25
	}
}

func (e *Editor) Layout(width, height int) (int, int) {
	return width, height
}

func (e *Editor) State() *EditorState {
	return e.state
}

func (e *Editor) Assets() *AssetManager {
	return e.assets
}
