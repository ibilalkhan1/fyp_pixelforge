package codegen

import (
	"bytes"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cart"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
	"image"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestCartAssetsExportRequiresAndPreservesFiles(t *testing.T) {
	p := pixelforge_project.NewProject("assets")
	p.Sprites = []pixelforge_project.SpriteAsset{{Name: "hero", RelativePath: "sprites/hero.png"}}
	file := filepath.Join(t.TempDir(), "game.pforge")
	if _, e := EncodeCartWithAssets(p, file); e == nil {
		t.Fatal("missing assets accepted")
	}
	root := pixelforge_project.AssetsDir(file)
	os.MkdirAll(filepath.Join(root, "sprites"), 0755)
	var pngBytes bytes.Buffer
	png.Encode(&pngBytes, image.NewRGBA(image.Rect(0, 0, 8, 8)))
	os.WriteFile(filepath.Join(root, "sprites/hero.png"), pngBytes.Bytes(), 0644)
	blob, e := EncodeCartWithAssets(p, file)
	if e != nil {
		t.Fatal(e)
	}
	_, assets, e := pixelforge_cart.DecodeBundle(blob)
	if e != nil {
		t.Fatal(e)
	}
	b, e := fs.ReadFile(assets, "assets/sprites/hero.png")
	if e != nil || !bytes.Equal(b, pngBytes.Bytes()) {
		t.Fatalf("asset lost: %v", e)
	}
	if err := capsuleruntime.LoadSprites(p, assets); err != nil {
		t.Fatalf("bundled sprite cannot load: %v", err)
	}
	p.Sprites[0].RelativePath = "../secret"
	if _, e := EncodeCartWithAssets(p, file); e == nil {
		t.Fatal("escape accepted")
	}
}
