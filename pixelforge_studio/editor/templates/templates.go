// Package templates exposes the four genre-starter project templates
// the File → New submenu spawns (plan-009 U22, R10, AE10).
//
// Each template hand-builds a *pixelforge_project.Project ready to load
// into the editor: one scene, one placeholder hero entity, the genre's
// physics preset, genre-appropriate input bindings, an AutoSnapshot
// save config, and (for grid + ladder genres) a tile atlas.
//
// AE10 (R10's "no named-game template" affordance) is enforced by
// AllNames(): the four genre names are intentionally generic — "Blank
// Platformer", "Blank Arcade Shooter", "Blank Grid Game", "Blank Ladder
// Platformer". The literal-game labels (Mario / Asteroids / Bomberman /
// DK) are NOT exposed; physics_preset values still carry the short
// identifier the runtime knows, but designers never see them in menus.
//
// The four template constructors are also exposed as exported funcs so
// integration tests can fork the project shape without going through
// the Lookup table.
package templates

import (
	"fmt"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// Template names. Exported so menu code, tests, and the Lookup table
// reference the same source of truth.
const (
	NameBlankPlatformer       = "Blank Platformer"
	NameBlankArcadeShooter    = "Blank Arcade Shooter"
	NameBlankGridGame         = "Blank Grid Game"
	NameBlankLadderPlatformer = "Blank Ladder Platformer"
)

// AllNames returns the four canonical template names in display order.
// AE10 enforcement lives here: the returned slice contains exactly the
// four generic genre starter names; literal-game labels (Mario,
// Asteroids, Bomberman, Donkey Kong, DK) are NOT included and never
// surface to designers via this seam. Returns a fresh slice on each
// call so callers may mutate it freely.
func AllNames() []string {
	return []string{
		NameBlankPlatformer,
		NameBlankArcadeShooter,
		NameBlankGridGame,
		NameBlankLadderPlatformer,
	}
}

// Lookup returns the constructor matching name, or nil if name is not
// one of the four canonical template names. The nil-on-unknown contract
// is the AE10 reinforcement: even if a caller passes "Mario" by
// accident, Lookup returns nil and the template flow refuses to start.
func Lookup(name string) func() *pixelforge_project.Project {
	switch name {
	case NameBlankPlatformer:
		return BlankPlatformer
	case NameBlankArcadeShooter:
		return BlankArcadeShooter
	case NameBlankGridGame:
		return BlankGridGame
	case NameBlankLadderPlatformer:
		return BlankLadderPlatformer
	}
	return nil
}

// New constructs the named template and renames the resulting project
// to projectName. Returns an error if name is not a known template.
// projectName may be empty — the template's default name then stays.
//
// Callers that need direct control over the resulting Project should
// call the per-genre constructor (BlankPlatformer / ...) and mutate
// the returned value directly.
func New(name, projectName string) (*pixelforge_project.Project, error) {
	ctor := Lookup(name)
	if ctor == nil {
		return nil, fmt.Errorf("templates: unknown template %q", name)
	}
	p := ctor()
	if projectName != "" {
		p.Name = projectName
	}
	return p, nil
}

// defaultSaveConfig returns the AutoSnapshot save config the four
// templates share. Kept here so a future tuning change touches one
// place rather than four template files.
func defaultSaveConfig() pixelforge_project.SaveConfig {
	cfg := pixelforge_project.DefaultSaveConfig()
	// DefaultSaveConfig already enables AutosaveEnabled; restating the
	// invariant defensively in case the default ever flips.
	cfg.AutosaveEnabled = true
	cfg.Set = true
	return cfg
}
