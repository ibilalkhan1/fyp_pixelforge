package main

import (
	"embed"
	gt "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gametree"
)

//go:embed *.wav
var gameAssets embed.FS

func main() {
	g := gt.NewPacmanGame()
	g.AssetsFS(gameAssets)
	g.PacmanColor(10) // yellow is 10
	g.GridColor(16)   // dark blue walls is 16
	g.Background(0)   // black is 0
	g.Movement("arrowUp", "arrowDown", "arrowLeft", "arrowRight")
	g.LoseCond("catchByEnemy")
	g.PowerPelletDuration(5) // duration in seconds
	g.Lives(3)
	g.Score()
	g.Sound(true)
	g.Loop()
}
