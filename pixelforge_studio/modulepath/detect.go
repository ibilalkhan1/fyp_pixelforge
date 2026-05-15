// Package modulepath answers the question "where is the Pixelforge
// engine source so I can wire an exported game's go.mod to it?"
//
// Three strategies cover the realistic deployment shapes:
//
//   - StrategyVendor (default): copy the engine source into the
//     exported game's vendor/ tree. One-click export ships
//     self-contained on any machine, no internet required.
//   - StrategyPublishedVersion: emit `require ... vX.Y.Z` against the
//     engine's published module version. Selectable but disabled by
//     default until the engine is actually published.
//   - StrategyDevReplace: emit a local `replace` directive pointing at
//     the engine checkout the editor was built from. Useful when an
//     engine contributor wants exported games to track their working
//     copy.
//
// Detect() inspects the editor's own runtime environment and picks the
// best default. Apply() does the file-system work for a chosen strategy.
package modulepath

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// EngineModule is the import path used in generated go.mod files.
const EngineModule = "github.com/ibilalkhan1/fyp_pixelforge"

// Strategy picks how the exported game references the engine.
type Strategy int

const (
	// StrategyVendor copies the engine into a vendor/ directory next
	// to the generated main.go. End-user default.
	StrategyVendor Strategy = iota

	// StrategyPublishedVersion emits a versioned require directive.
	// Disabled by default until the engine is published as a Go
	// module; selectable for early adopters who publish a fork.
	StrategyPublishedVersion

	// StrategyDevReplace writes a `replace` directive pointing at a
	// local checkout. Auto-detected when the editor itself was built
	// from a git checkout.
	StrategyDevReplace
)

func (s Strategy) String() string {
	switch s {
	case StrategyVendor:
		return "vendor"
	case StrategyPublishedVersion:
		return "published"
	case StrategyDevReplace:
		return "dev-replace"
	default:
		return fmt.Sprintf("strategy(%d)", int(s))
	}
}

// Detection holds the strategy choice plus the inputs that informed it.
type Detection struct {
	Strategy   Strategy
	EnginePath string // absolute path to engine source (StrategyDevReplace / StrategyVendor)
	Version    string // module version (StrategyPublishedVersion); empty otherwise
	Reason     string // human-readable explanation surfaced in editor logs
}

// Detect picks the best strategy for the current environment. The
// caller can override the choice via the editor settings; this is the
// out-of-the-box default.
//
// editorBinaryPath is the path returned by os.Executable() for the
// running studio binary; tests pass a synthetic path to exercise the
// detection logic without a real running binary.
func Detect(editorBinaryPath string) Detection {
	enginePath, ok := findEnginePath(editorBinaryPath)
	if !ok {
		return Detection{
			Strategy: StrategyVendor,
			Reason:   "engine source not located beside editor binary — defaulting to vendor",
		}
	}
	if isGitCheckout(enginePath) {
		return Detection{
			Strategy:   StrategyDevReplace,
			EnginePath: enginePath,
			Reason:     "engine source is a git checkout — using local replace directive",
		}
	}
	return Detection{
		Strategy:   StrategyVendor,
		EnginePath: enginePath,
		Reason:     "engine source found — defaulting to vendor snapshot",
	}
}

// findEnginePath walks upward from binaryPath looking for a directory
// containing a go.mod with EngineModule. Returns ok=false if the
// engine source cannot be located within five directory levels.
func findEnginePath(binaryPath string) (string, bool) {
	dir := binaryPath
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", false
		}
	} else {
		stat, err := os.Stat(dir)
		if err == nil && !stat.IsDir() {
			dir = filepath.Dir(dir)
		}
	}
	for i := 0; i < 6; i++ {
		gomod := filepath.Join(dir, "go.mod")
		if hasEngineModule(gomod) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

// hasEngineModule reports whether gomod's `module` line points at
// EngineModule. Cheap line scan — no need to pull in golang.org/x/mod.
func hasEngineModule(gomod string) bool {
	data, err := os.ReadFile(gomod)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")) == EngineModule
		}
	}
	return false
}

// isGitCheckout reports whether the engine path has a .git child —
// our signal that the user is running from a checkout rather than an
// installed binary.
func isGitCheckout(enginePath string) bool {
	_, err := os.Stat(filepath.Join(enginePath, ".git"))
	return err == nil
}

// Apply executes the chosen strategy against an export directory.
//
//   - StrategyVendor copies all pixelforge*/ packages from EnginePath
//     into outputDir/vendor/<EngineModule>/.
//   - StrategyDevReplace writes nothing to disk here; the caller's
//     code-gen template renders the `replace` directive in go.mod.
//     Apply still verifies the path exists so failures surface early.
//   - StrategyPublishedVersion is similarly handled in the template.
//
// Returns the absolute path the strategy wrote to (vendor root), or
// "" when no file-system work was required.
func Apply(d Detection, outputDir string) (string, error) {
	switch d.Strategy {
	case StrategyVendor:
		if d.EnginePath == "" {
			return "", errors.New("modulepath: StrategyVendor requires an EnginePath")
		}
		dst := filepath.Join(outputDir, "vendor", filepath.FromSlash(EngineModule))
		if err := copyEngineTree(d.EnginePath, dst); err != nil {
			return "", err
		}
		return dst, nil
	case StrategyDevReplace:
		if d.EnginePath == "" {
			return "", errors.New("modulepath: StrategyDevReplace requires an EnginePath")
		}
		if _, err := os.Stat(filepath.Join(d.EnginePath, "go.mod")); err != nil {
			return "", fmt.Errorf("modulepath: StrategyDevReplace path %s has no go.mod: %w", d.EnginePath, err)
		}
		return "", nil
	case StrategyPublishedVersion:
		if d.Version == "" {
			return "", errors.New("modulepath: StrategyPublishedVersion requires a Version")
		}
		return "", nil
	default:
		return "", fmt.Errorf("modulepath: unknown strategy %v", d.Strategy)
	}
}

// copyEngineTree copies all pixelforge*/ subdirectories plus the
// engine's go.mod, go.sum, and root .go files into dst. We deliberately
// skip examples, binaries, and the studio package itself — the
// generated game only needs the engine subsystems.
func copyEngineTree(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			if !shouldVendorDir(name) {
				continue
			}
			if err := copyDir(filepath.Join(src, name), filepath.Join(dst, name)); err != nil {
				return err
			}
			continue
		}
		// Copy the engine's top-level .go files and go.mod / go.sum.
		if name == "go.mod" || name == "go.sum" || strings.HasSuffix(name, ".go") {
			if err := copyFile(filepath.Join(src, name), filepath.Join(dst, name)); err != nil {
				return err
			}
		}
	}
	return nil
}

// shouldVendorDir reports whether a directory at the engine root
// should be carried into the exported game's vendor tree. We pick the
// pixelforge_* engine packages plus internal/, and skip examples,
// tooling, build artefacts.
func shouldVendorDir(name string) bool {
	switch name {
	case "internal", "pixelforge_pool", "pixelforge_math":
		return true
	}
	if strings.HasPrefix(name, "pixelforge_") {
		// Skip the studio itself (the exported game doesn't need it).
		if name == "pixelforge_studio" {
			return false
		}
		// Skip ideation/test/demo packages we identified by name.
		if name == "pixelforge_examples" || name == "pixelforge_test_helpers" {
			return false
		}
		return true
	}
	return false
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		// Skip test files and binaries; they bloat the vendor tree
		// and are never needed at runtime.
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
