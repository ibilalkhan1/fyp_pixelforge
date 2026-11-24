package codegen

import (
	"fmt"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cart"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// EncodeCartWithAssets carries every referenced sprite and WAV alongside the
// project. Missing assets fail before a misleading successful export is shown.
func EncodeCartWithAssets(p *pixelforge_project.Project, projectPath string) ([]byte, error) {
	data, e := EncodeCart(p)
	if e != nil {
		return nil, e
	}
	paths := []string{}
	for _, a := range p.Sprites {
		if a.RelativePath != "" {
			paths = append(paths, a.RelativePath)
		}
	}
	for _, a := range p.Audio {
		if a.RelativePath != "" {
			if strings.ToLower(filepath.Ext(a.RelativePath)) != ".wav" {
				return nil, fmt.Errorf("export audio %q: only decoded WAV assets are supported", a.RelativePath)
			}
			paths = append(paths, a.RelativePath)
		}
	}
	if len(paths) == 0 {
		return data, nil
	}
	if projectPath == "" {
		return nil, fmt.Errorf("save the project before exporting its assets")
	}
	root := pixelforge_project.AssetsDir(projectPath)
	resolvedRoot, e := filepath.EvalSymlinks(root)
	if e != nil {
		return nil, fmt.Errorf("cart assets directory: %w", e)
	}
	assets := map[string][]byte{}
	total := int64(len(data))
	for _, name := range paths {
		if !fs.ValidPath(name) || strings.Contains(name, "\\") {
			return nil, fmt.Errorf("unsafe asset path %q", name)
		}
		file, e := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(name)))
		if e != nil {
			return nil, fmt.Errorf("export asset %q: %w", name, e)
		}
		rel, e := filepath.Rel(resolvedRoot, file)
		if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("asset %q leaves project assets directory", name)
		}
		info, e := os.Stat(file)
		if e != nil {
			return nil, e
		}

		if _, exists := assets["assets/"+name]; exists {
			continue
		}
		if info.Size() > (64<<20)-total {
			return nil, fmt.Errorf("cart assets exceed 64 MiB at %q", name)
		}
		total += info.Size()
		b, e := os.ReadFile(file)
		if e != nil {
			return nil, e
		}
		assets["assets/"+name] = b
	}
	return pixelforge_cart.EncodeBundle(data, assets)
}
