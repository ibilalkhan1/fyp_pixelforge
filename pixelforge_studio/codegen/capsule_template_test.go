package codegen

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// capsule_template_test.go covers idea #7 v1 U1's Capsule
// generation: capsule.go parses, embeds the right resources, and
// stamps typed scene/item constants.

func TestRenderCapsuleGo_ParsesValid(t *testing.T) {
	p := pixelforge_project.NewProject("ok")
	src, err := renderCapsuleGo(p, nil)
	require.NoError(t, err)
	_, err = parser.ParseFile(token.NewFileSet(), "capsule.go", src, 0)
	assert.NoError(t, err, "generated capsule.go parses as valid Go")
}

func TestRenderCapsuleGo_EmbedsProjectAndAssets(t *testing.T) {
	p := pixelforge_project.NewProject("ok")
	src, err := renderCapsuleGo(p, nil)
	require.NoError(t, err)
	s := string(src)
	assert.Contains(t, s, "//go:embed project.pforge",
		"capsule.go embeds project.pforge so the binary is self-contained")
	assert.Contains(t, s, "//go:embed all:assets",
		"capsule.go embeds the assets directory tree")
}

func TestRenderCapsuleGo_ExportsCapsuleRunAndDefaults(t *testing.T) {
	p := pixelforge_project.NewProject("ok")
	src, _ := renderCapsuleGo(p, nil)
	s := string(src)
	assert.Contains(t, s, "func CapsuleRun(opts CapsuleOpts) error")
	assert.Contains(t, s, "func CapsuleDefaults() CapsuleOpts")
	assert.Contains(t, s, "func CapsuleAssets()")
	assert.Contains(t, s, "func CapsuleProjectData()")
}

func TestRenderCapsuleGo_StampsSceneIDConstants(t *testing.T) {
	p := pixelforge_project.NewProject("ok")
	p.Scenes = []pixelforge_project.Scene{
		{ID: "title"},
		{ID: "level_1"},
		{ID: "boss.intro"},
	}
	src, err := renderCapsuleGo(p, nil)
	require.NoError(t, err)
	s := string(src)
	assert.Contains(t, s, `const SceneIDTitle = "title"`)
	assert.Contains(t, s, `const SceneIDLevel1 = "level_1"`)
	assert.Contains(t, s, `const SceneIDBossIntro = "boss.intro"`)
}

func TestRenderCapsuleGo_StampsItemIDConstants(t *testing.T) {
	p := pixelforge_project.NewProject("ok")
	p.Items = []pixelforge_project.ItemDefinition{
		{ID: "potion"}, {ID: "sword_iron"},
	}
	src, err := renderCapsuleGo(p, nil)
	require.NoError(t, err)
	s := string(src)
	assert.Contains(t, s, `const ItemIDPotion = "potion"`)
	assert.Contains(t, s, `const ItemIDSwordIron = "sword_iron"`)
}

func TestRenderCapsuleGo_DataOverrideFieldOnOpts(t *testing.T) {
	p := pixelforge_project.NewProject("ok")
	src, _ := renderCapsuleGo(p, nil)
	s := string(src)
	assert.Contains(t, s, "DataOverride *pixelforge_project.Project",
		"CapsuleOpts.DataOverride lets the editor preview share the Capsule code path")
}

func TestRenderMainGo_DelegatesToCapsuleRun(t *testing.T) {
	p := pixelforge_project.NewProject("ok")
	src, err := renderMainGo(p, "test/module")
	require.NoError(t, err)
	s := string(src)
	assert.Contains(t, s, "CapsuleRun(CapsuleDefaults())")
	assert.False(t, strings.Contains(s, "applyProject"),
		"old applyProject indirection retired in favour of CapsuleRun")
}

func TestToCapsuleCamel_HandlesAllDelimiters(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"title", "Title"},
		{"level_1", "Level1"},
		{"boss.intro", "BossIntro"},
		{"title-screen", "TitleScreen"},
		{"a/b/c", "ABC"},
		{"123abc", "X123abc"},
		{"", "Anonymous"},
		{"___", "Anonymous"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, toCapsuleCamel(c.in), "input: %q", c.in)
	}
}

func TestRenderCapsuleGo_NoScenesNoItemsStillEmits(t *testing.T) {
	p := pixelforge_project.NewProject("empty")
	p.Scenes = nil
	p.Items = nil
	src, err := renderCapsuleGo(p, nil)
	require.NoError(t, err)
	_, err = parser.ParseFile(token.NewFileSet(), "capsule.go", src, 0)
	assert.NoError(t, err, "empty scenes + items still produce valid capsule.go")
}
