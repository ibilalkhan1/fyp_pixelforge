package templates

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// findBinding walks an InputMap looking for the intent. Returns the
// matching binding + true, or zero + false.
func findBinding(t *testing.T, p *pixelforge_project.Project, intent string) (pixelforge_project.InputBinding, bool) {
	t.Helper()
	for _, b := range p.InputMap {
		if b.Intent == intent {
			return b, true
		}
	}
	return pixelforge_project.InputBinding{}, false
}

// bindingHasKey checks whether any key in a binding's Keyboard slice
// matches keyName.
func bindingHasKey(b pixelforge_project.InputBinding, keyName string) bool {
	for _, k := range b.Keyboard {
		if k == keyName {
			return true
		}
	}
	return false
}

// TestBlankPlatformer_HasMarioPreset is the per-template baseline:
// returns a project; physics_preset == "mario"; entity count >= 1;
// Space → jump binding present. Mirrors the plan's Track A test list.
func TestBlankPlatformer_HasMarioPreset(t *testing.T) {
	p := BlankPlatformer()
	require.NotNil(t, p)
	assert.Equal(t, "mario", p.PhysicsPreset)
	require.NotEmpty(t, p.Scenes, "platformer should have at least one scene")
	assert.GreaterOrEqual(t, len(p.Scenes[0].Entities), 1, "platformer should have at least one entity")

	jump, ok := findBinding(t, p, "input/jump")
	require.True(t, ok, "platformer should bind input/jump")
	assert.True(t, bindingHasKey(jump, " "), "platformer jump should bind Space (= \" \")")

	// AutoSnapshot save config is the v2 default.
	assert.True(t, p.SaveConfig.AutosaveEnabled, "platformer should ship with autosave on")
}

func TestBlankArcadeShooter_HasAsteroidsPreset(t *testing.T) {
	p := BlankArcadeShooter()
	require.NotNil(t, p)
	assert.Equal(t, "asteroids", p.PhysicsPreset)
	require.NotEmpty(t, p.Scenes)
	assert.GreaterOrEqual(t, len(p.Scenes[0].Entities), 1)

	rl, ok := findBinding(t, p, "input/rotate_left")
	require.True(t, ok, "shooter should bind input/rotate_left")
	assert.True(t, bindingHasKey(rl, "Left"), "shooter rotate_left should bind ArrowLeft (= \"Left\")")

	fire, ok := findBinding(t, p, "input/fire")
	require.True(t, ok)
	assert.True(t, bindingHasKey(fire, " "), "shooter fire should bind Space")

	assert.True(t, p.SaveConfig.AutosaveEnabled)
}

func TestBlankGridGame_HasBombermanPreset(t *testing.T) {
	p := BlankGridGame()
	require.NotNil(t, p)
	assert.Equal(t, "bomberman", p.PhysicsPreset)
	require.NotEmpty(t, p.Scenes)

	// 13×11 board check.
	require.NotEmpty(t, p.Scenes[0].TileAtlases, "grid game should have a tile atlas")
	atlas := p.Scenes[0].TileAtlases[0]
	require.Len(t, atlas.Grid, 11, "board should be 11 rows tall")
	require.Len(t, atlas.Grid[0], 13, "board should be 13 cols wide")

	// Border walls.
	for c := 0; c < 13; c++ {
		assert.Equal(t, 1, atlas.Grid[0][c], "top border col %d should be wall", c)
		assert.Equal(t, 1, atlas.Grid[10][c], "bottom border col %d should be wall", c)
	}
	for r := 0; r < 11; r++ {
		assert.Equal(t, 1, atlas.Grid[r][0], "left border row %d should be wall", r)
		assert.Equal(t, 1, atlas.Grid[r][12], "right border row %d should be wall", r)
	}

	place, ok := findBinding(t, p, "input/place_bomb")
	require.True(t, ok, "grid game should bind input/place_bomb")
	assert.True(t, bindingHasKey(place, " "), "grid game place_bomb should bind Space")

	assert.True(t, p.SaveConfig.AutosaveEnabled)
}

func TestBlankLadderPlatformer_HasDKPreset(t *testing.T) {
	p := BlankLadderPlatformer()
	require.NotNil(t, p)
	assert.Equal(t, "dk", p.PhysicsPreset)
	require.NotEmpty(t, p.Scenes)
	require.NotEmpty(t, p.Scenes[0].TileAtlases)

	// Ladder tiles (value 2) present in the atlas.
	atlas := p.Scenes[0].TileAtlases[0]
	hasLadder := false
	for _, row := range atlas.Grid {
		for _, v := range row {
			if v == 2 {
				hasLadder = true
				break
			}
		}
		if hasLadder {
			break
		}
	}
	assert.True(t, hasLadder, "ladder platformer atlas should contain ladder tiles (value 2)")

	climb, ok := findBinding(t, p, "input/climb_up")
	require.True(t, ok, "ladder platformer should bind input/climb_up")
	assert.True(t, bindingHasKey(climb, "W"), "ladder climb_up should bind W")

	assert.True(t, p.SaveConfig.AutosaveEnabled)
}

// TestAllNames_ReturnsExactlyFourGenericNames is the AE10 enforcement:
// the four exposed template names must be the generic genre starters,
// NOT the trademarked literal names (Mario / Asteroids / Bomberman /
// Donkey Kong / DK).
func TestAllNames_ReturnsExactlyFourGenericNames(t *testing.T) {
	names := AllNames()
	require.Len(t, names, 4, "exactly four template names")

	want := []string{
		"Blank Platformer",
		"Blank Arcade Shooter",
		"Blank Grid Game",
		"Blank Ladder Platformer",
	}
	gotSorted := append([]string(nil), names...)
	wantSorted := append([]string(nil), want...)
	sort.Strings(gotSorted)
	sort.Strings(wantSorted)
	assert.Equal(t, wantSorted, gotSorted, "exact match on the four generic names")

	// AE10 negative: the trademarked / literal names must NOT appear.
	for _, banned := range []string{"Mario", "Asteroids", "Bomberman", "Donkey Kong", "DK"} {
		for _, n := range names {
			assert.False(t, strings.EqualFold(n, banned), "template name %q must not be literal %q (AE10)", n, banned)
		}
	}
}

// TestLookup_UnknownReturnsNil pins the AE10 negative contract on the
// Lookup seam: a literal-named template lookup returns nil rather than
// surfacing a project.
func TestLookup_UnknownReturnsNil(t *testing.T) {
	for _, banned := range []string{"Mario", "Asteroids", "Bomberman", "Donkey Kong", "DK"} {
		assert.Nil(t, Lookup(banned), "Lookup(%q) should be nil — only generic names resolve", banned)
	}
	assert.Nil(t, Lookup(""))
	assert.Nil(t, Lookup("nonexistent"))
}

// TestLookup_KnownNamesResolve verifies all four generic names return
// non-nil constructors that each produce a non-nil project.
func TestLookup_KnownNamesResolve(t *testing.T) {
	for _, name := range AllNames() {
		ctor := Lookup(name)
		require.NotNil(t, ctor, "Lookup(%q) should return a constructor", name)
		p := ctor()
		require.NotNil(t, p, "Lookup(%q)() should return a non-nil project", name)
	}
}

// TestPresetsAreDistinct cross-checks that each template carries its
// own genre's physics_preset, so the runtime can branch on the string.
func TestPresetsAreDistinct(t *testing.T) {
	got := map[string]string{
		"Blank Platformer":        BlankPlatformer().PhysicsPreset,
		"Blank Arcade Shooter":    BlankArcadeShooter().PhysicsPreset,
		"Blank Grid Game":         BlankGridGame().PhysicsPreset,
		"Blank Ladder Platformer": BlankLadderPlatformer().PhysicsPreset,
	}

	// All four values must be distinct.
	seen := map[string]string{}
	for name, preset := range got {
		require.NotEmpty(t, preset, "%s must carry a non-empty physics_preset", name)
		if other, ok := seen[preset]; ok {
			t.Fatalf("templates %q and %q share physics_preset %q", name, other, preset)
		}
		seen[preset] = name
	}

	// Exact values match the runtime presets in pixelforge_project's
	// project.go PhysicsPreset doc.
	assert.Equal(t, "mario", got["Blank Platformer"])
	assert.Equal(t, "asteroids", got["Blank Arcade Shooter"])
	assert.Equal(t, "bomberman", got["Blank Grid Game"])
	assert.Equal(t, "dk", got["Blank Ladder Platformer"])
}

// TestNew_AppliesProjectName verifies the New(name, projectName)
// convenience renames the resulting project.
func TestNew_AppliesProjectName(t *testing.T) {
	p, err := New("Blank Platformer", "MyCoolGame")
	require.NoError(t, err)
	assert.Equal(t, "MyCoolGame", p.Name)
	assert.Equal(t, "mario", p.PhysicsPreset)
}

// TestNew_EmptyProjectNameKeepsDefault verifies that passing the empty
// string leaves the template's default name in place.
func TestNew_EmptyProjectNameKeepsDefault(t *testing.T) {
	p, err := New("Blank Arcade Shooter", "")
	require.NoError(t, err)
	assert.Equal(t, "Blank Arcade Shooter", p.Name)
}

// TestNew_UnknownTemplateErrors documents the error path so the menu
// can surface it as a status message.
func TestNew_UnknownTemplateErrors(t *testing.T) {
	_, err := New("Mario", "x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Mario")
}

// TestTemplate_HasSpritesAndScene is a smoke check across all four
// templates: each must carry at least one sprite + scene + entity so
// the editor's first render isn't empty.
func TestTemplate_HasSpritesAndScene(t *testing.T) {
	cases := map[string]func() *pixelforge_project.Project{
		"platformer":    BlankPlatformer,
		"arcade":        BlankArcadeShooter,
		"grid":          BlankGridGame,
		"ladder":        BlankLadderPlatformer,
	}
	for label, ctor := range cases {
		p := ctor()
		assert.NotEmpty(t, p.Sprites, "%s should have sprites", label)
		require.NotEmpty(t, p.Scenes, "%s should have scenes", label)
		assert.NotEmpty(t, p.Scenes[0].Entities, "%s scene should have entities", label)
	}
}
