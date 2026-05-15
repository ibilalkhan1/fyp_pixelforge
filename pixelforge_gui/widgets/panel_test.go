package widgets_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
)

func panelTestSetup(t *testing.T) pixelforge.Canvas {
	t.Helper()
	pixelforge.SetScreenSize(128, 128)
	pixelforge.Cls()
	return pixelforge.Screen()
}

func TestNewPanel(t *testing.T) {
	panelTestSetup(t)

	t.Run("untitled panel uses its element as its body", func(t *testing.T) {
		p := widgets.NewPanel(0, 0, 50, 30, widgets.PanelOptions{})
		assert.Same(t, p.Element, p.Body)
	})

	t.Run("titled panel reserves a strip and exposes Body below it", func(t *testing.T) {
		p := widgets.NewPanel(0, 0, 50, 30, widgets.PanelOptions{Title: "ASSETS"})
		require.NotNil(t, p.Body)
		assert.NotSame(t, p.Element, p.Body)
		assert.Greater(t, p.TitleHeight, 0)
		assert.Equal(t, p.TitleHeight, p.Body.Y)
		assert.Equal(t, p.H-p.TitleHeight, p.Body.H)
	})
}

func TestPanel_Draw(t *testing.T) {
	canvas := panelTestSetup(t)

	root := pgui.New()
	p := widgets.NewPanel(10, 10, 40, 30, widgets.PanelOptions{
		Title:       "HDR",
		BgColor:     5,
		BorderColor: 7,
		TitleColor:  10,
	})
	root.Attach(p.Element)

	root.Draw()

	// Border colour should appear along the top-left corner.
	assert.Equal(t, pixelforge.Color(7), canvas.Get(10, 10),
		"top-left border pixel must be the border colour")
	// Body interior (well below the title strip) should be the bg colour.
	bodyY := 10 + p.TitleHeight + 4
	assert.Equal(t, pixelforge.Color(5), canvas.Get(20, bodyY),
		"body interior must be the bg colour")
}

func hasColorInRect(c pixelforge.Canvas, want pixelforge.Color, x, y, w, h int) bool {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			if c.Get(i, j) == want {
				return true
			}
		}
	}
	return false
}

func TestPanel_TitleDrawn(t *testing.T) {
	canvas := panelTestSetup(t)

	root := pgui.New()
	p := widgets.NewPanel(0, 0, 60, 24, widgets.PanelOptions{
		Title:       "OK",
		BgColor:     1,
		BorderColor: 2,
		TitleColor:  10,
	})
	root.Attach(p.Element)
	root.Draw()

	// Some pixel inside the title strip should be the title text colour.
	assert.True(t,
		hasColorInRect(canvas, 10, 0, 0, p.W, p.TitleHeight),
		"expected at least one title-colour pixel in the title strip")
}
