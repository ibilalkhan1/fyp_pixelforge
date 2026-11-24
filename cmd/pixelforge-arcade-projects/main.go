// Command pixelforge-arcade-projects generates the downloadable Studio presets.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	pf "github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_arcade"
	project "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

func main() {
	out := flag.String("out", "pixelforge_arcade/projects", "output directory")
	flag.Parse()
	for _, kind := range []string{"dungeon", "frogger", "city"} {
		if err := generate(*out, kind); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func generate(root, kind string) error {
	g, err := pixelforge_arcade.New(kind, nil)
	if err != nil {
		return fmt.Errorf("create %s: %w", kind, err)
	}
	p := project.NewProject(g.Title)
	p.ScreenWidth, p.ScreenHeight, p.TPS = pixelforge_arcade.Width, pixelforge_arcade.Height, 30
	p.Arcade = &project.ArcadeConfig{Kind: kind, Layout: g.Layout}
	for i, c := range pixelforge_arcade.Colors {
		p.Palette.Base[i] = fmt.Sprintf("#%06x", uint32(c))
		pf.Palette[i] = c
	}
	p.Sprites = []project.SpriteAsset{{Name: "initial-frame", RelativePath: "scene.png", Width: p.ScreenWidth,
		Height: p.ScreenHeight, FrameW: p.ScreenWidth, FrameH: p.ScreenHeight}}
	p.Scenes[0].Name = g.Title
	p.Scenes[0].Entities = []project.Entity{{ID: "preview", Name: "Initial scene preview",
		Components: []project.EntityComponent{
			{Type: "Sprite", Values: map[string]any{"sprite": "initial-frame"}},
		}}}
	dir := filepath.Join(root, kind)
	assets := filepath.Join(dir, "game.pforge-assets")
	if err := os.MkdirAll(assets, 0755); err != nil {
		return fmt.Errorf("create assets: %w", err)
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("encode project: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "game.pforge"), data, 0644); err != nil {
		return fmt.Errorf("write project: %w", err)
	}
	pf.SetScreenSize(p.ScreenWidth, p.ScreenHeight)
	g.Draw()
	im := image.NewRGBA(image.Rect(0, 0, p.ScreenWidth, p.ScreenHeight))
	for y := 0; y < p.ScreenHeight; y++ {
		for x := 0; x < p.ScreenWidth; x++ {
			r, g, b := pf.Palette[pf.Screen().Get(x, y)].RGB()
			im.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	f, err := os.Create(filepath.Join(assets, "scene.png"))
	if err != nil {
		return fmt.Errorf("create preview: %w", err)
	}
	if err := png.Encode(f, im); err != nil {
		_ = f.Close()
		return fmt.Errorf("encode preview: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close preview: %w", err)
	}
	return nil
}
