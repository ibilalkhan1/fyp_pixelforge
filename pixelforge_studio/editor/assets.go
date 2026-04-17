package editor

import (
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type AssetManager struct {
	Sprites    map[string]*SpriteAsset
	AudioFiles map[string]string
}

type SpriteAsset struct {
	Name   string
	Path   string
	Image  *ebiten.Image
	Width  int
	Height int
	FrameW int
	FrameH int
}

func NewAssetManager() *AssetManager {
	return &AssetManager{
		Sprites:    make(map[string]*SpriteAsset),
		AudioFiles: make(map[string]string),
	}
}

func (am *AssetManager) LoadSprite(path string) (*SpriteAsset, error) {
	// Check if already loaded
	for _, s := range am.Sprites {
		if s.Path == path {
			return s, nil
		}
	}

	// Load image
	img, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Determine frame size (default 8x8 for retro style)
	frameW := 8
	frameH := 8

	// If image is small, use that as frame size
	if width <= 16 && height <= 16 {
		frameW = width
		frameH = height
	}

	// Extract name from path
	name := GetBaseName(path)

	sprite := &SpriteAsset{
		Name:   name,
		Path:   path,
		Image:  img,
		Width:  width,
		Height: height,
		FrameW: frameW,
		FrameH: frameH,
	}

	am.Sprites[name] = sprite
	return sprite, nil
}

func (am *AssetManager) LoadAudio(path string) (string, error) {
	// Check if already loaded
	for name, p := range am.AudioFiles {
		if p == path {
			return name, nil
		}
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		return "", err
	}

	name := GetBaseName(path)
	am.AudioFiles[name] = path
	return name, nil
}

func (am *AssetManager) GetSprite(name string) *SpriteAsset {
	return am.Sprites[name]
}

func (am *AssetManager) GetFrameCount(name string) int {
	sprite := am.Sprites[name]
	if sprite == nil {
		return 0
	}
	framesX := sprite.Width / sprite.FrameW
	framesY := sprite.Height / sprite.FrameH
	return framesX * framesY
}

func (am *AssetManager) GetAllSpriteNames() []string {
	names := make([]string, 0, len(am.Sprites))
	for name := range am.Sprites {
		names = append(names, name)
	}
	return names
}

func (am *AssetManager) GetAllAudioNames() []string {
	names := make([]string, 0, len(am.AudioFiles))
	for name := range am.AudioFiles {
		names = append(names, name)
	}
	return names
}

func (am *AssetManager) RemoveSprite(name string) {
	delete(am.Sprites, name)
}

func (am *AssetManager) RemoveAudio(name string) {
	delete(am.AudioFiles, name)
}
