package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pf "github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

type previewAsset struct {
	canvas   pf.Canvas
	size     int64
	modified int64
}

func (g *sceneGame) resolveAsset(asset *pixelforge_project.SpriteAsset) (*pf.Canvas, bool) {
	if g.editor == nil || asset == nil || asset.RelativePath == "" || g.editor.currentProjectPath == "" {
		return nil, false
	}
	root := pixelforge_project.AssetsDir(g.editor.currentProjectPath)
	file := filepath.Join(root, filepath.FromSlash(asset.RelativePath))
	rel, err := filepath.Rel(root, file)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, false
	}
	info, err := os.Stat(file)
	if err != nil {
		return nil, false
	}
	if g.assetCache == nil {
		g.assetCache = map[string]previewAsset{}
	}
	cached, ok := g.assetCache[file]
	if ok && cached.size == info.Size() && cached.modified == info.ModTime().UnixNano() {
		return &cached.canvas, true
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, false
	}
	canvas, err := pf.DecodeCanvasOrErr(data)
	if err != nil {
		g.editor.SetStatusMessage(fmt.Sprintf("Sprite %s: %v", asset.Name, err))
		return nil, false
	}
	g.assetCache[file] = previewAsset{canvas: canvas, size: info.Size(), modified: info.ModTime().UnixNano()}
	return &canvas, true
}
func (g *sceneGame) drawEntities(entities []pixelforge_project.Entity, assets []pixelforge_project.SpriteAsset) {
	for _, entity := range entities {
		name := ""
		for _, component := range entity.Components {
			if component.Type == "Sprite" {
				name, _ = component.Values["sprite"].(string)
				break
			}
		}
		for i := range assets {
			asset := &assets[i]
			if asset.Name != name {
				continue
			}
			canvas, ok := g.resolveAsset(asset)
			if !ok {
				continue
			}
			w, h := asset.FrameW, asset.FrameH
			if w <= 0 {
				w = canvas.W()
			}
			if h <= 0 {
				h = canvas.H()
			}
			pf.DrawSprite(pf.SpriteFrom(*canvas, 0, 0, w, h), entity.TileX*8, entity.TileY*8)
		}
	}
}
