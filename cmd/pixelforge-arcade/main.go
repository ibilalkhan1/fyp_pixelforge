// Command pixelforge-arcade runs the bundled Studio projects through the shared player.
package main

import (
	"flag"
	"log"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_arcade"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_render"
)

func main() {
	kind := flag.String("game", "dungeon", "Studio game: dungeon, frogger or city")
	flag.Parse()
	project, assets, err := pixelforge_arcade.Project(selectedGame(*kind))
	if err != nil {
		log.Fatal(err)
	}
	pixelforge_ebiten.SetMetricsBorder(8, 8, 8, 8)
	if err := pixelforge_render.RunProject(project, assets); err != nil {
		log.Fatal(err)
	}
}
