package pixelforge_render

import (
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	pf "github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_tilemap"
)

// RunProject is shared by generated source exports and the universal player.
func RunProject(project *pixelforge_project.Project, assets fs.FS) error {
	ApplyProject(project)
	rt, err := capsuleruntime.Boot(project, assets, capsuleruntime.Options{})
	if err != nil {
		return fmt.Errorf("boot project: %w", err)
	}
	defer rt.Close()
	rt.Start()
	releaseControls := installArcadeWebControls(rt.Arcade)
	defer releaseControls()
	previousDraw := pf.Draw
	pf.Draw = func() { DrawProject(rt) }
	defer func() { pf.Draw = previousDraw; pixelforge_ebiten.SetTickHook(nil) }()
	var keys []ebiten.Key
	pixelforge_ebiten.SetTickHook(func(tick uint64) {
		keys = inpututil.AppendPressedKeys(keys[:0])
		_ = RenderTickAtScreen(rt, projectSurface(), tick, InputFrame{Keys: keys, NativeEvents: true})
	})
	if err := pixelforge_ebiten.RunOrErr(); err != nil {
		return fmt.Errorf("run project: %w", err)
	}
	return nil
}

var projectFrameSurface *ebiten.Image

func projectSurface() *ebiten.Image {
	w, h := pf.Screen().W(), pf.Screen().H()
	if projectFrameSurface == nil || projectFrameSurface.Bounds().Dx() != w || projectFrameSurface.Bounds().Dy() != h {
		projectFrameSurface = ebiten.NewImage(w, h)
	}
	return projectFrameSurface
}

// ApplyProject installs the logical screen, frame rate and authored palette.
func ApplyProject(project *pixelforge_project.Project) {
	if project == nil {
		return
	}
	if project.ScreenWidth > 0 && project.ScreenHeight > 0 {
		pf.SetScreenSize(project.ScreenWidth, project.ScreenHeight)
	}
	if project.TPS > 0 {
		pf.SetTPS(project.TPS)
	}
	for i, value := range project.Palette.Base {
		rgb, err := strconv.ParseUint(strings.TrimPrefix(value, "#"), 16, 32)
		if err == nil && len(value) == 7 {
			pf.Palette[i] = pf.RGB(rgb)
		}
	}
}

// DrawProject renders the active runtime scene with actual decoded sprite data.
func DrawProject(rt *capsuleruntime.Runtime) {
	if rt == nil || rt.Project == nil || rt.CurrentScene == nil {
		return
	}
	pf.SetDrawTarget(pf.Screen())
	if rt.Arcade != nil {
		rt.Arcade.Draw()
		return
	}
	pf.Screen().Clear(0)
	scene := rt.CurrentScene
	for i := range scene.TileAtlases {
		layer := &scene.TileAtlases[i]
		if sheet, ok := capsuleruntime.LookupSprite(layer.SpriteSheetRef); ok {
			pixelforge_tilemap.Render(layer, &sheet.Canvas)
		}
	}
	for _, entity := range scene.Entities {
		if effect := rt.VisualEffects[entity.ID]; effect != nil && effect.Hidden {
			continue
		}
		name := ""
		for _, component := range entity.Components {
			if component.Type == "Sprite" {
				name, _ = component.Values["sprite"].(string)
				break
			}
		}
		if effect := rt.VisualEffects[entity.ID]; effect != nil && effect.SpriteSwap != "" {
			name = effect.SpriteSwap
		}
		entry, ok := capsuleruntime.LookupSprite(name)
		if !ok {
			continue
		}
		w, h := entry.Asset.FrameW, entry.Asset.FrameH
		if w <= 0 {
			w = entry.Canvas.W()
		}
		if h <= 0 {
			h = entry.Canvas.H()
		}
		x, y := entity.TileX*8, entity.TileY*8
		if body := rt.Bodies[entity.ID]; body != nil {
			x, y = body.Position.X.Int(), body.Position.Y.Int()
		}
		pf.DrawSprite(pf.SpriteFrom(entry.Canvas, 0, 0, w, h), x, y)
	}
}
