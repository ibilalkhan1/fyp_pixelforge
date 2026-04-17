package editor

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Project struct {
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	ScreenWidth  int          `json:"screenWidth"`
	ScreenHeight int          `json:"screenHeight"`
	TPS          int          `json:"tps"`
	Sprites      []SpriteInfo `json:"sprites"`
	AudioFiles   []AudioInfo  `json:"audioFiles"`
	Scene        SceneData    `json:"scene"`
	UpdateCode   string       `json:"updateCode"`
	DrawCode     string       `json:"drawCode"`
}

type SceneData struct {
	Objects []SceneObject `json:"objects"`
}

func NewProject(name string) *Project {
	return &Project{
		Name:         name,
		Version:      "1.0.0",
		ScreenWidth:  320,
		ScreenHeight: 180,
		TPS:          30,
		Sprites:      []SpriteInfo{},
		AudioFiles:   []AudioInfo{},
		Scene: SceneData{
			Objects: []SceneObject{},
		},
		UpdateCode: `// Write your Update code here
func update() {
	// Handle input with pixelforge_key, pixelforge_pad, pixelforge_mouse
}`,
		DrawCode: `// Write your Draw code here
func draw() {
	// Draw your game here
	pixelforge.Screen().Clear(0)
}`,
	}
}

func (p *Project) Save(path string) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadProject(path string) (*Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var proj Project
	if err := json.Unmarshal(data, &proj); err != nil {
		return nil, err
	}
	return &proj, nil
}

func (p *Project) AddSprite(name, path string, w, h, frameW, frameH int) {
	p.Sprites = append(p.Sprites, SpriteInfo{
		Name:   name,
		Path:   path,
		Width:  w,
		Height: h,
		FrameW: frameW,
		FrameH: frameH,
	})
}

func (p *Project) AddAudio(name, path string) {
	p.AudioFiles = append(p.AudioFiles, AudioInfo{
		Name: name,
		Path: path,
	})
}

func (p *Project) AddObject(obj SceneObject) {
	p.Scene.Objects = append(p.Scene.Objects, obj)
}

func (p *Project) RemoveObject(id int) {
	var newObjs []SceneObject
	for _, o := range p.Scene.Objects {
		if o.ID != id {
			newObjs = append(newObjs, o)
		}
	}
	p.Scene.Objects = newObjs
}

func (p *Project) GetObject(id int) *SceneObject {
	for i := range p.Scene.Objects {
		if p.Scene.Objects[i].ID == id {
			return &p.Scene.Objects[i]
		}
	}
	return nil
}

func (p *Project) NextObjectID() int {
	maxID := 0
	for _, o := range p.Scene.Objects {
		if o.ID > maxID {
			maxID = o.ID
		}
	}
	return maxID + 1
}

func GetBaseName(path string) string {
	ext := filepath.Ext(path)
	name := filepath.Base(path)
	return name[:len(name)-len(ext)]
}
