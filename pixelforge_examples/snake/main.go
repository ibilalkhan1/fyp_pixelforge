package main

import (
	"embed"
	gt "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gametree"
)

//go:embed *.png *.wav
var gameAssets embed.FS

func main() {
	g := gt.NewGame()
	g.AssetsFS(gameAssets)
	g.Player("sprites3.png")
	g.Background(13) // background color from 0 to 22
	g.Wall(16)       // grid size from 8 to 30
	g.Movement("arrowUp", "arrowDown", "arrowLeft", "arrowRight")
	g.LoseCond("crash", "wall")
	g.Score()
	g.Sound(true)
	g.Loop()
}
