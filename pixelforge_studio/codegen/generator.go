// Package codegen emits a runnable Go project from a pixelforge_project
// Project. The generator is template-driven (text/template) — every byte
// of generated source is visible in templates.go for review.
//
// Output layout for `Generate(project, outDir, opts)`:
//
//	outDir/
//	  main.go              # thin Ebitengine host (gofmt'd)
//	  go.mod               # module declaration + engine require/replace
//	  project.pforge       # the schema, embedded into main.go via go:embed
//	  assets/
//	    sprites/*.png      # copied from the project's *-assets/ directory
//	    audio/*.wav
//	  vendor/...           # populated when modulepath.StrategyVendor is used
package codegen

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/modulepath"
)

// TemplateVersion stamps every generated file's header. Bump when the
// templates change shape so debugging "why does my old export not
// rebuild?" is unambiguous.
const TemplateVersion = "1"

// Options configures Generate. Zero value works for the common case.
type Options struct {
	// ModulePath is the Go module path the generated main.go declares.
	// Defaults to "exported-game" when empty.
	ModulePath string

	// Strategy overrides the module-path strategy. When zero (the
	// default), Generate calls modulepath.Detect to pick.
	Strategy modulepath.Strategy

	// EnginePath overrides modulepath.Detect's EnginePath result.
	// Useful for tests and for users editing settings.
	EnginePath string

	// EngineVersion overrides modulepath.Detect's Version result.
	EngineVersion string

	// Force overwrites a non-empty output directory. Without Force,
	// Generate refuses to clobber existing files.
	Force bool

	// RunGoModTidy, when true, runs `go mod tidy` in outDir after
	// writing all files so the exported game picks up its transitive
	// dependency graph. Defaults to false because tests rely on
	// deterministic output; production export should set it true.
	RunGoModTidy bool

	// ProjectSourcePath is the user's `.pforge` path on disk. When
	// set, sprite and audio assets are copied from its sibling
	// `*-assets/` directory. When empty, no assets are copied.
	ProjectSourcePath string
}

// Generate writes an exported game to outDir. Returns the resolved
// strategy used so the caller can surface it in editor logs.
func Generate(p *pixelforge_project.Project, outDir string, opts Options) (modulepath.Detection, error) {
	if p == nil {
		return modulepath.Detection{}, errors.New("codegen: Generate: project is nil")
	}
	if outDir == "" {
		return modulepath.Detection{}, errors.New("codegen: Generate: outDir is empty")
	}

	// Validate the destination is empty (or Force).
	if !opts.Force {
		if files, _ := os.ReadDir(outDir); len(files) > 0 {
			names := []string{}
			for _, f := range files {
				names = append(names, f.Name())
			}
			return modulepath.Detection{}, fmt.Errorf(
				"codegen: outDir %s is not empty (%s); pass Options{Force:true} to overwrite",
				outDir, strings.Join(names, ", "))
		}
	}

	// Pre-flight: every referenced sprite/audio asset must exist on
	// source disk. Atomic — we want to fail before writing any output.
	if opts.ProjectSourcePath != "" {
		if err := preflightAssets(p, opts.ProjectSourcePath); err != nil {
			return modulepath.Detection{}, err
		}
	}

	detection := resolveStrategy(opts)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return modulepath.Detection{}, err
	}

	// Render templates first, so a parse error fails before any disk write.
	mainSrc, err := renderMainGo(p)
	if err != nil {
		return modulepath.Detection{}, err
	}
	goModSrc, err := renderGoMod(opts, detection)
	if err != nil {
		return modulepath.Detection{}, err
	}

	// Write generated source.
	if err := os.WriteFile(filepath.Join(outDir, "main.go"), mainSrc, 0o644); err != nil {
		return detection, err
	}
	if err := os.WriteFile(filepath.Join(outDir, "go.mod"), goModSrc, 0o644); err != nil {
		return detection, err
	}

	// Write the embedded .pforge.
	pforgeBytes, err := encodeProject(p)
	if err != nil {
		return detection, err
	}
	if err := os.WriteFile(filepath.Join(outDir, "project.pforge"), pforgeBytes, 0o644); err != nil {
		return detection, err
	}

	// Copy assets.
	if opts.ProjectSourcePath != "" {
		if err := copyAssets(p, opts.ProjectSourcePath, outDir); err != nil {
			return detection, err
		}
	}

	// Module-path strategy file-system work (vendor copy etc.).
	if _, err := modulepath.Apply(detection, outDir); err != nil {
		return detection, fmt.Errorf("codegen: modulepath apply: %w", err)
	}

	if opts.RunGoModTidy {
		if err := runGoModTidy(outDir); err != nil {
			return detection, err
		}
	}

	return detection, nil
}

// runGoModTidy resolves the exported game's transitive deps so a fresh
// `go build` succeeds. Skipped by default for deterministic test output.
func runGoModTidy(outDir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = outDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("codegen: go mod tidy: %w\n%s", err, string(out))
	}
	return nil
}

// resolveStrategy honors caller overrides; falls back to autodetect.
func resolveStrategy(opts Options) modulepath.Detection {
	if opts.EnginePath != "" || opts.EngineVersion != "" {
		return modulepath.Detection{
			Strategy:   opts.Strategy,
			EnginePath: opts.EnginePath,
			Version:    opts.EngineVersion,
			Reason:     "explicit override via Options",
		}
	}
	bin, _ := os.Executable()
	return modulepath.Detect(bin)
}

// mainTemplateData is the value passed to mainGoTemplate.
type mainTemplateData struct {
	TemplateVersion string
	ProjectName     string
}

// goModTemplateData is the value passed to goModTemplate.
type goModTemplateData struct {
	ModulePath           string
	EngineRequireVersion string
	EngineReplacePath    string
}

func renderMainGo(p *pixelforge_project.Project) ([]byte, error) {
	tpl, err := template.New("main").Parse(mainGoTemplate)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, mainTemplateData{
		TemplateVersion: TemplateVersion,
		ProjectName:     p.Name,
	}); err != nil {
		return nil, err
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Surface the offending source alongside the gofmt error so
		// the human debugging a template change can see what we
		// fed gofmt.
		return nil, fmt.Errorf("codegen: gofmt failed on generated main.go: %w\nsource:\n%s", err, buf.String())
	}
	return formatted, nil
}

func renderGoMod(opts Options, d modulepath.Detection) ([]byte, error) {
	modulePath := opts.ModulePath
	if modulePath == "" {
		modulePath = "exported-game"
	}
	data := goModTemplateData{ModulePath: modulePath}

	switch d.Strategy {
	case modulepath.StrategyVendor:
		// Vendored — pin to v0.0.0 with no replace. The vendor/
		// directory satisfies the require at build time.
		data.EngineRequireVersion = "v0.0.0"
	case modulepath.StrategyPublishedVersion:
		if d.Version == "" {
			return nil, errors.New("codegen: StrategyPublishedVersion requires a Version")
		}
		data.EngineRequireVersion = d.Version
	case modulepath.StrategyDevReplace:
		if d.EnginePath == "" {
			return nil, errors.New("codegen: StrategyDevReplace requires an EnginePath")
		}
		data.EngineRequireVersion = "v0.0.0"
		data.EngineReplacePath = d.EnginePath
	}
	tpl, err := template.New("gomod").Parse(goModTemplate)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// preflightAssets verifies every referenced sprite and audio asset is
// on disk in the source project's *-assets/ directory before we touch
// the output. Atomic-on-failure is the contract.
func preflightAssets(p *pixelforge_project.Project, projectPath string) error {
	dir := pixelforge_project.AssetsDir(projectPath)
	missing := []string{}
	for _, s := range p.Sprites {
		if s.RelativePath == "" {
			continue
		}
		abs := filepath.Join(dir, filepath.FromSlash(s.RelativePath))
		if _, err := os.Stat(abs); err != nil {
			missing = append(missing, "sprite "+s.Name+": "+abs)
		}
	}
	for _, a := range p.Audio {
		if a.RelativePath == "" {
			continue
		}
		abs := filepath.Join(dir, filepath.FromSlash(a.RelativePath))
		if _, err := os.Stat(abs); err != nil {
			missing = append(missing, "audio "+a.Name+": "+abs)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("codegen: missing assets:\n  %s", strings.Join(missing, "\n  "))
	}
	return nil
}

// copyAssets duplicates each referenced sprite/audio file into the
// output's assets/ subtree, mirroring the relative path inside the
// project's *-assets/ directory.
func copyAssets(p *pixelforge_project.Project, projectPath, outDir string) error {
	srcRoot := pixelforge_project.AssetsDir(projectPath)
	dstRoot := filepath.Join(outDir, "assets")
	for _, s := range p.Sprites {
		if s.RelativePath == "" {
			continue
		}
		if err := copyOne(filepath.Join(srcRoot, filepath.FromSlash(s.RelativePath)),
			filepath.Join(dstRoot, filepath.FromSlash(s.RelativePath))); err != nil {
			return err
		}
	}
	for _, a := range p.Audio {
		if a.RelativePath == "" {
			continue
		}
		if err := copyOne(filepath.Join(srcRoot, filepath.FromSlash(a.RelativePath)),
			filepath.Join(dstRoot, filepath.FromSlash(a.RelativePath))); err != nil {
			return err
		}
	}
	return nil
}

func copyOne(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// encodeProject defers to pixelforge_project.Encode so the embedded
// copy matches the file the user saves to disk byte-for-byte.
func encodeProject(p *pixelforge_project.Project) ([]byte, error) {
	return pixelforge_project.Encode(p)
}
