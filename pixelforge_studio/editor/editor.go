package editor

import (
	"image"
	"image/color"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	defaultUpdateCode = `func update() {
	// Handle input with pixelforge_key, pixelforge_pad, pixelforge_mouse
	
	// Example: Exit with Escape
	// if pixelforge_key.Duration(pixelforge_key.Escape) > 0 {
	//     os.Exit(0)
	// }
}`

	defaultDrawCode = `func draw() {
	// Your drawing code here
	pixelforge.Screen().Clear(0) // Clear with color 0 (black)
	
	// Example: Draw all scene objects
	// for _, obj := range sceneObjects {
	//     sprite := getSprite(obj.SpriteName)
	//     if sprite != nil {
	//         pixelforge.DrawSprite(sprite, obj.X, obj.Y)
	//     }
	// }
}`
)

// EditorState - main state for editor
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

// SpriteInfo - sprite data
type SpriteInfo struct {
	Name   string
	Path   string
	Width  int
	Height int
	FrameW int
	FrameH int
	Image  *ebiten.Image
}

// SceneObject - object in scene
type SceneObject struct {
	ID            int
	SpriteName    string
	X, Y          int
	Width, Height int
	Layer         int
	Collision     *CollisionBox
	Visible       bool
}

// CollisionBox - collision data
type CollisionBox struct {
	X, Y int
	W, H int
}

// AudioInfo - audio file data
type AudioInfo struct {
	Name string
	Path string
}

// MenuItem - menu button
type MenuItem struct {
	Label         string
	X, Y          int
	Width, Height int
}

// Tool - current tool
type Tool int

const (
	ToolSelect Tool = iota
	ToolPlace
	ToolDelete
)

// Editor - main editor struct
type Editor struct {
	state    *EditorState
	assets   *AssetManager
	menus    []MenuItem
	menuOpen int
	tool     Tool
}

const (
	TitleBarH  = 40
	LeftPanelW = 200
	CanvasX    = LeftPanelW
	CanvasY    = TitleBarH
	CanvasW    = 880
	CanvasH    = 715
)

// NewEditor - create new editor
func NewEditor() *Editor {
	return &Editor{
		state: &EditorState{
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
		},
		assets: NewAssetManager(),
		menus: []MenuItem{
			{Label: "File", X: 10, Y: 10, Width: 50, Height: 20},
			{Label: "Edit", X: 65, Y: 10, Width: 50, Height: 20},
			{Label: "View", X: 120, Y: 10, Width: 50, Height: 20},
			{Label: "Project", X: 175, Y: 10, Width: 60, Height: 20},
			{Label: "Help", X: 240, Y: 10, Width: 40, Height: 20},
		},
		menuOpen: -1,
	}
}

// Update - handle input
func (e *Editor) Update() error {
	mx, my := ebiten.CursorPosition()

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// Menu
		if my < TitleBarH {
			for i, m := range e.menus {
				if mx >= m.X && mx < m.X+m.Width && my >= m.Y && my < m.Y+m.Height {
					e.menuOpen = i
					return nil
				}
			}
			e.menuOpen = -1
		} else if mx >= CanvasX && mx < CanvasX+CanvasW && my >= CanvasY && my < CanvasY+CanvasH {
			canvX := (mx - CanvasX) / int(e.state.Scale)
			canvY := (my - CanvasY) / int(e.state.Scale)

			// If Place tool is active and a sprite is selected, place it
			if e.tool == ToolPlace && e.state.SelectedIndex >= 0 && e.state.SelectedIndex < len(e.state.Sprites) {
				sprite := e.state.Sprites[e.state.SelectedIndex]
				newObj := SceneObject{
					ID:         len(e.state.SceneObjects) + 1,
					SpriteName: sprite.Name,
					X:          canvX - sprite.FrameW/2,
					Y:          canvY - sprite.FrameH/2,
					Width:      sprite.FrameW,
					Height:     sprite.FrameH,
					Layer:      len(e.state.SceneObjects),
					Collision:  &CollisionBox{X: 0, Y: 0, W: sprite.FrameW, H: sprite.FrameH},
					Visible:    true,
				}
				e.state.SceneObjects = append(e.state.SceneObjects, newObj)
				return nil
			}

			// If Delete tool is active, delete object under cursor
			if e.tool == ToolDelete {
				for i := len(e.state.SceneObjects) - 1; i >= 0; i-- {
					obj := &e.state.SceneObjects[i]
					if canvX >= obj.X && canvX < obj.X+obj.Width &&
						canvY >= obj.Y && canvY < obj.Y+obj.Height {
						e.state.SceneObjects = append(e.state.SceneObjects[:i], e.state.SceneObjects[i+1:]...)
						e.state.SelectedIndex = -1
						return nil
					}
				}
				return nil
			}

			// Otherwise, select object under cursor
			e.state.SelectedIndex = -1
			for i := len(e.state.SceneObjects) - 1; i >= 0; i-- {
				obj := &e.state.SceneObjects[i]
				if canvX >= obj.X && canvX < obj.X+obj.Width &&
					canvY >= obj.Y && canvY < obj.Y+obj.Height {
					e.state.SelectedIndex = i
					break
				}
			}
		} else if mx >= 0 && mx < LeftPanelW && my >= 50 {
			idx := (my - 50) / 25
			if idx >= 0 && idx < len(e.state.Sprites) {
				e.state.SelectedIndex = idx
			}
		} else {
			e.menuOpen = -1
		}
	}

	// Drag objects with Select tool
	if e.state.SelectedIndex >= 0 && e.tool == ToolSelect && len(e.state.SceneObjects) > 0 {
		if mx >= CanvasX && mx < CanvasX+CanvasW && my >= CanvasY && my < CanvasY+CanvasH {
			e.state.SceneObjects[e.state.SelectedIndex].X = (mx-CanvasX)/int(e.state.Scale) - e.state.SceneObjects[e.state.SelectedIndex].Width/2
			e.state.SceneObjects[e.state.SelectedIndex].Y = (my-CanvasY)/int(e.state.Scale) - e.state.SceneObjects[e.state.SelectedIndex].Height/2
		}
	}

	// Keyboard shortcuts
	if ebiten.IsKeyPressed(ebiten.KeyV) {
		e.tool = ToolSelect
	} else if ebiten.IsKeyPressed(ebiten.KeyP) {
		e.tool = ToolPlace
	} else if ebiten.IsKeyPressed(ebiten.KeyX) {
		e.tool = ToolDelete
	} else if ebiten.IsKeyPressed(ebiten.KeyDelete) && e.state.SelectedIndex >= 0 {
		// Delete selected object
		if e.state.SelectedIndex >= 0 && e.state.SelectedIndex < len(e.state.SceneObjects) {
			e.state.SceneObjects = append(e.state.SceneObjects[:e.state.SelectedIndex], e.state.SceneObjects[e.state.SelectedIndex+1:]...)
			e.state.SelectedIndex = -1
		}
	}

	return nil
}

func (e *Editor) importSpriteFromPath(path string) error {
	img, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		return err
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	frameW, frameH := 8, 8
	if w <= 16 && h <= 16 {
		frameW, frameH = w, h
	}

	name := filepath.Base(path)
	name = name[:len(name)-4]

	e.state.Sprites = append(e.state.Sprites, SpriteInfo{
		Name:   name,
		Path:   path,
		Width:  w,
		Height: h,
		FrameW: frameW,
		FrameH: frameH,
		Image:  img,
	})
	return nil
}

func (e *Editor) ScanAssetsFolder(folder string) {
	// Recursively scan folder for PNG files
	entries, err := os.ReadDir(folder)
	if err != nil {
		return
	}
	for _, f := range entries {
		subPath := filepath.Join(folder, f.Name())
		if f.IsDir() {
			// Recurse into subdirectories
			e.ScanAssetsFolder(subPath)
		} else {
			name := f.Name()
			if len(name) > 4 && name[len(name)-4:] == ".png" {
				e.importSpriteFromPath(subPath)
			}
		}
	}
}

// Draw - render editor
func (e *Editor) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 30, G: 30, B: 35, A: 255})

	// Title bar
	screen.SubImage(image.Rect(0, 0, 1280, TitleBarH)).(*ebiten.Image).Fill(color.RGBA{R: 45, G: 45, B: 50, A: 255})

	// Left panel
	screen.SubImage(image.Rect(0, TitleBarH, LeftPanelW, 755)).(*ebiten.Image).Fill(color.RGBA{R: 40, G: 40, B: 45, A: 255})

	// Right panel
	screen.SubImage(image.Rect(1080, TitleBarH, 1280, 755)).(*ebiten.Image).Fill(color.RGBA{R: 40, G: 40, B: 45, A: 255})

	// Canvas
	screen.SubImage(image.Rect(CanvasX, CanvasY, CanvasX+CanvasW, CanvasY+CanvasH)).(*ebiten.Image).Fill(color.RGBA{R: 25, G: 25, B: 30, A: 255})

	// Status bar
	screen.SubImage(image.Rect(0, 755, 1280, 800)).(*ebiten.Image).Fill(color.RGBA{R: 35, G: 35, B: 40, A: 255})

	// Panel borders - left panel
	for y := TitleBarH; y < 755; y++ {
		screen.SubImage(image.Rect(LeftPanelW-1, y, LeftPanelW, y+1)).(*ebiten.Image).Fill(color.RGBA{R: 60, G: 60, B: 70, A: 255})
	}
	// Panel borders - right panel
	for y := TitleBarH; y < 755; y++ {
		screen.SubImage(image.Rect(1080, y, 1081, y+1)).(*ebiten.Image).Fill(color.RGBA{R: 60, G: 60, B: 70, A: 255})
	}
	// Canvas border
	for x := CanvasX; x < CanvasX+CanvasW; x++ {
		screen.SubImage(image.Rect(x, CanvasY+CanvasH-1, x+1, CanvasY+CanvasH)).(*ebiten.Image).Fill(color.RGBA{R: 60, G: 60, B: 70, A: 255})
	}
	for y := CanvasY; y < CanvasY+CanvasH; y++ {
		screen.SubImage(image.Rect(CanvasX+CanvasW-1, y, CanvasX+CanvasW, y+1)).(*ebiten.Image).Fill(color.RGBA{R: 60, G: 60, B: 70, A: 255})
	}

	// Tool buttons with visual indicators
	toolColors := []color.RGBA{
		{R: 80, G: 160, B: 80, A: 255},  // Select - green
		{R: 80, G: 120, B: 200, A: 255}, // Place - blue
		{R: 180, G: 80, B: 80, A: 255},  // Delete - red
	}
	toolLabels := []string{"V", "P", "X"}
	for i := range toolLabels {
		bg := color.RGBA{R: 50, G: 50, B: 55, A: 255}
		indicator := color.RGBA{R: 70, G: 70, B: 80, A: 255}
		if e.tool == Tool(i) {
			bg = color.RGBA{R: 60, G: 60, B: 70, A: 255}
			indicator = toolColors[i]
		}
		btnX := CanvasX + 10 + i*60
		screen.SubImage(image.Rect(btnX, CanvasY+10, btnX+40, CanvasY+30)).(*ebiten.Image).Fill(bg)
		// Color indicator square
		screen.SubImage(image.Rect(btnX+5, CanvasY+15, btnX+10, CanvasY+20)).(*ebiten.Image).Fill(indicator)
	}

	// Grid
	if e.state.ShowGrid {
		gridSize := 8
		for gx := CanvasX; gx < CanvasX+CanvasW; gx += gridSize {
			for gy := CanvasY; gy < CanvasY+CanvasH; gy += gridSize {
				screen.SubImage(image.Rect(gx, gy, gx+1, gy+1)).(*ebiten.Image).Fill(color.RGBA{R: 50, G: 50, B: 55, A: 255})
			}
		}
	}

	// Objects
	for i, obj := range e.state.SceneObjects {
		objX := CanvasX + obj.X*int(e.state.Scale)
		objY := CanvasY + obj.Y*int(e.state.Scale)
		objW := obj.Width * int(e.state.Scale)
		objH := obj.Height * int(e.state.Scale)

		col := color.RGBA{R: 80, G: 130, B: 180, A: 255}
		if i == e.state.SelectedIndex {
			col = color.RGBA{R: 130, G: 180, B: 230, A: 255}
		}
		screen.SubImage(image.Rect(objX, objY, objX+objW, objY+objH)).(*ebiten.Image).Fill(col)
	}

	// Menus
	for _, m := range e.menus {
		bg := color.RGBA{R: 60, G: 60, B: 70, A: 255}
		if e.menuOpen >= 0 && m.X == e.menus[e.menuOpen].X {
			bg = color.RGBA{R: 80, G: 80, B: 90, A: 255}
		}
		screen.SubImage(image.Rect(m.X, m.Y, m.X+m.Width, m.Y+m.Height)).(*ebiten.Image).Fill(bg)
	}

	// Menu dropdown
	if e.menuOpen >= 0 {
		screen.SubImage(image.Rect(0, TitleBarH, 200, 400)).(*ebiten.Image).Fill(color.RGBA{R: 45, G: 45, B: 50, A: 255})
		var items []string
		switch e.menuOpen {
		case 0:
			items = []string{"New Project", "Save", "Export Game", "---", "Exit"}
		case 1:
			items = []string{"Delete", "Duplicate"}
		case 2:
			items = []string{"Zoom In", "Zoom Out", "Grid", "Collision"}
		case 3:
			items = []string{"Run Preview", "Settings"}
		}
		menuY := TitleBarH + 5
		for range items {
			screen.SubImage(image.Rect(5, menuY, 195, menuY+20)).(*ebiten.Image).Fill(color.RGBA{R: 50, G: 50, B: 55, A: 255})
			menuY += 25
		}
	}

	// Sprite list header - "SPRITES" header bar
	screen.SubImage(image.Rect(5, TitleBarH+5, 195, TitleBarH+25)).(*ebiten.Image).Fill(color.RGBA{R: 55, G: 55, B: 65, A: 255})

	// Sprites list with visual thumbnails
	for i := range e.state.Sprites {
		bg := color.RGBA{R: 50, G: 50, B: 55, A: 255}
		if i == e.state.SelectedIndex {
			bg = color.RGBA{R: 70, G: 90, B: 110, A: 255}
		}
		listY := TitleBarH + 30 + i*30
		screen.SubImage(image.Rect(5, listY, 195, listY+25)).(*ebiten.Image).Fill(bg)
		// Thumbnail box
		screen.SubImage(image.Rect(10, listY+5, 25, listY+20)).(*ebiten.Image).Fill(color.RGBA{R: 80, G: 80, B: 90, A: 255})
	}

	// Current tool indicator in status bar
	statusY := 760
	screen.SubImage(image.Rect(10, statusY, 150, statusY+25)).(*ebiten.Image).Fill(color.RGBA{R: 45, G: 45, B: 50, A: 255})
	toolIndicator := toolColors[int(e.tool)]
	screen.SubImage(image.Rect(15, statusY+8, 20, statusY+17)).(*ebiten.Image).Fill(toolIndicator)

	// Object count in status bar
	screen.SubImage(image.Rect(200, statusY, 350, statusY+25)).(*ebiten.Image).Fill(color.RGBA{R: 45, G: 45, B: 50, A: 255})

	// Zoom indicator
	screen.SubImage(image.Rect(400, statusY, 500, statusY+25)).(*ebiten.Image).Fill(color.RGBA{R: 45, G: 45, B: 50, A: 255})

	// Canvas size indicator
	screen.SubImage(image.Rect(600, statusY, 750, statusY+25)).(*ebiten.Image).Fill(color.RGBA{R: 45, G: 45, B: 50, A: 255})
}

// Layout - set size
func (e *Editor) Layout(w, h int) (int, int) {
	return w, h
}

// State - get state
func (e *Editor) State() *EditorState {
	return e.state
}
