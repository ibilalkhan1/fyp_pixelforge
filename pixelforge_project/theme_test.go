package pixelforge_project_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

func TestDefaultTheme_NonZeroSlots(t *testing.T) {
	th := pixelforge_project.DefaultTheme()
	assert.NotEqual(t, 0, th.PanelSlot, "panel slot should default to something other than slot 0")
	assert.NotEqual(t, 0, th.TextSlot, "text slot should default to something other than slot 0")
	assert.Equal(t, "cofont", th.FontName)
}

func TestThemeSanitize_ClampsOutOfRange(t *testing.T) {
	th := pixelforge_project.Theme{BackgroundSlot: 99, PanelSlot: -3}
	th.SanitizeSlots()
	assert.Equal(t, 0, th.BackgroundSlot)
	assert.Equal(t, 0, th.PanelSlot)
}

func TestNewProject_PopulatesDefaultTheme(t *testing.T) {
	p := pixelforge_project.NewProject("demo")
	assert.Equal(t, pixelforge_project.DefaultTheme(), p.Theme)
}

func TestLoadReader_LegacyProjectGetsDefaultTheme(t *testing.T) {
	// JSON without a theme field — simulates an M0-M2 save.
	body := `{"schema_version":1,"name":"legacy","screen_width":320,"screen_height":180,"tps":30,"palette":{},"scenes":[{"id":"main","name":"Main","entities":[]}]}`
	p, err := pixelforge_project.LoadReader(strings.NewReader(body))
	require.NoError(t, err)
	// Theme should fall back to default.
	assert.Equal(t, pixelforge_project.DefaultTheme().FontName, p.Theme.FontName)
}

func TestThemeRoundTripsThroughEncodeDecode(t *testing.T) {
	p := pixelforge_project.NewProject("demo")
	p.Theme.BackgroundSlot = 14
	p.Theme.FontName = "ttf-demo"

	data, err := pixelforge_project.Encode(p)
	require.NoError(t, err)

	loaded, err := pixelforge_project.LoadReader(strings.NewReader(string(data)))
	require.NoError(t, err)
	assert.Equal(t, 14, loaded.Theme.BackgroundSlot)
	assert.Equal(t, "ttf-demo", loaded.Theme.FontName)
}
