package pixelforge_arcade

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

//go:embed all:projects
var projectFiles embed.FS

// Project loads the same .pforge and asset directory offered for Studio download.
func Project(kind string) (*pixelforge_project.Project, fs.FS, error) {
	if kind != "dungeon" && kind != "frogger" && kind != "city" {
		return nil, nil, fmt.Errorf("unknown arcade project %q", kind)
	}
	data, err := projectFiles.ReadFile("projects/" + kind + "/game.pforge")
	if err != nil {
		return nil, nil, fmt.Errorf("read arcade project: %w", err)
	}
	p, err := pixelforge_project.LoadReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("load arcade project: %w", err)
	}
	assets, err := fs.Sub(projectFiles, "projects/"+kind+"/game.pforge-assets")
	if err != nil {
		return nil, nil, fmt.Errorf("open arcade assets: %w", err)
	}
	return p, projectAssets{FS: assets}, nil
}

// projectAssets exposes the canonical assets/ prefix used by capsule loaders.
type projectAssets struct{ fs.FS }

func (a projectAssets) Open(name string) (fs.File, error) {
	file, err := a.FS.Open(strings.TrimPrefix(name, "assets/"))
	if err != nil {
		return nil, fmt.Errorf("open project asset: %w", err)
	}
	return file, nil
}
