package editor

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type Editor struct {
	state *EditorState
}

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

func NewEditor() *Editor {
	return &Editor{
		state: NewEditorState(),
	}
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
		UpdateCode:   "",
		DrawCode:     "",
	}
}

func (e *Editor) Update() error {
	return nil
}

func (e *Editor) Draw(screen *ebiten.Image) {
	// Dark background
	screen.Fill(color.RGBA{R: 30, G: 30, B: 35, A: 255})

	// Title bar background
	screen.SubImage(image.Rect(0, 0, 1280, 40)).(*ebiten.Image).Fill(color.RGBA{R: 45, G: 45, B: 50, A: 255})

	// Left panel (project browser)
	screen.SubImage(image.Rect(0, 40, 200, 755)).(*ebiten.Image).Fill(color.RGBA{R: 40, G: 40, B: 45, A: 255})

	// Right panel (properties)
	screen.SubImage(image.Rect(1080, 40, 1280, 755)).(*ebiten.Image).Fill(color.RGBA{R: 40, G: 40, B: 45, A: 255})

	// Center panel (scene canvas)
	screen.SubImage(image.Rect(200, 40, 1080, 755)).(*ebiten.Image).Fill(color.RGBA{R: 25, G: 25, B: 30, A: 255})

	// Status bar
	screen.SubImage(image.Rect(0, 755, 1280, 800)).(*ebiten.Image).Fill(color.RGBA{R: 35, G: 35, B: 40, A: 255})

	// Draw menu items placeholder (File, Edit, View, Project, Help)
	menuItems := []string{"File", "Edit", "View", "Project", "Help"}
	xPos := 10
	for _, item := range menuItems {
		// Simple text would require font - using rectangles as placeholders
		_ = xPos
		_ = item
	}
}

func (e *Editor) Layout(width, height int) (int, int) {
	return width, height
}

func (e *Editor) State() *EditorState {
	return e.state
}
