package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// chromeTopBandH is the total vertical pixels the menu bar + title bar +
// tab strip claim above the body region. M1.5 added the menu bar and
// tab strip; tests below account for them when computing offsets.
func chromeTopBandH(titleH int) int {
	return widgetsMenuBarHeight() + titleH + tabStripH
}

// widgetsMenuBarHeight pulls in the constant from the widgets package
// through a tiny helper so the test file doesn't need an extra import.
func widgetsMenuBarHeight() int { return 22 }

// Happy path: a default-sized window splits cleanly into menu bar,
// title bar, tab strip, left panel, canvas, right panel, status bar.
func TestChromeLayout_Default1280x800(t *testing.T) {
	l := computeChromeLayout(1280, 800,
		defaultTitleBarH, defaultLeftPanelW, defaultRightPanelW, defaultStatusBarH)

	assert.Equal(t, defaultTitleBarH, l.TitleBar.H)
	assert.Equal(t, 1280, l.TitleBar.W)
	assert.Equal(t, 0, l.TitleBar.X)
	assert.Equal(t, widgetsMenuBarHeight(), l.TitleBar.Y)

	assert.Equal(t, defaultStatusBarH, l.StatusBar.H)
	assert.Equal(t, 800-defaultStatusBarH, l.StatusBar.Y)

	bodyY := chromeTopBandH(defaultTitleBarH)
	assert.Equal(t, defaultLeftPanelW, l.LeftPanel.W)
	assert.Equal(t, 0, l.LeftPanel.X)
	assert.Equal(t, bodyY, l.LeftPanel.Y)
	assert.Equal(t, 800-bodyY-defaultStatusBarH, l.LeftPanel.H)

	assert.Equal(t, defaultRightPanelW, l.RightPanel.W)
	assert.Equal(t, 1280-defaultRightPanelW, l.RightPanel.X)

	wantCanvasW := 1280 - defaultLeftPanelW - defaultRightPanelW
	wantCanvasH := 800 - bodyY - defaultStatusBarH
	assert.Equal(t, defaultLeftPanelW, l.Canvas.X)
	assert.Equal(t, bodyY, l.Canvas.Y)
	assert.Equal(t, wantCanvasW, l.Canvas.W)
	assert.Equal(t, wantCanvasH, l.Canvas.H)
}

// Regions tile the window with no gaps and no overlap. This is the
// invariant chrome.draw relies on for clean repaints.
func TestChromeLayout_RegionsTileWindow(t *testing.T) {
	cases := []struct{ w, h int }{
		{1280, 800},
		{800, 600},
		{1920, 1080},
		{640, 480},
	}
	for _, c := range cases {
		l := computeChromeLayout(c.w, c.h,
			defaultTitleBarH, defaultLeftPanelW, defaultRightPanelW, defaultStatusBarH)

		// vertical: menu + title + tabstrip + body + status = window height
		body := l.LeftPanel.H
		require.Equal(t, body, l.Canvas.H, "%dx%d canvas height", c.w, c.h)
		require.Equal(t, body, l.RightPanel.H, "%dx%d right panel height", c.w, c.h)
		assert.Equal(t, c.h, l.MenuBar.H+l.TitleBar.H+l.TabStrip.H+body+l.StatusBar.H,
			"vertical tile for %dx%d", c.w, c.h)

		// horizontal: left + canvas + right = window width
		assert.Equal(t, c.w, l.LeftPanel.W+l.Canvas.W+l.RightPanel.W,
			"horizontal tile for %dx%d", c.w, c.h)

		// no overlap on adjacencies
		bodyY := l.MenuBar.H + l.TitleBar.H + l.TabStrip.H
		assert.Equal(t, l.LeftPanel.W, l.Canvas.X, "canvas X after left panel")
		assert.Equal(t, l.LeftPanel.W+l.Canvas.W, l.RightPanel.X, "right panel X after canvas")
		assert.Equal(t, bodyY, l.Canvas.Y, "canvas Y after top chrome band")
		assert.Equal(t, bodyY+l.Canvas.H, l.StatusBar.Y, "status bar Y after canvas")
	}
}

// Small window: panels clamp down so the canvas keeps at least
// minCanvasW horizontal pixels.
func TestChromeLayout_SmallWindowClampsToMinimumCanvas(t *testing.T) {
	l := computeChromeLayout(600, 400,
		defaultTitleBarH, defaultLeftPanelW, defaultRightPanelW, defaultStatusBarH)

	assert.GreaterOrEqual(t, l.Canvas.W, minCanvasW,
		"canvas width must stay >= %d on small window", minCanvasW)
	// Panels never grow above defaults
	assert.LessOrEqual(t, l.LeftPanel.W, defaultLeftPanelW)
	assert.LessOrEqual(t, l.RightPanel.W, defaultRightPanelW)
}

// Large 4K window: panels stay at their default widths (no scaling)
// and the canvas absorbs all the extra space.
func TestChromeLayout_LargeWindowKeepsFixedPanels(t *testing.T) {
	l := computeChromeLayout(3840, 2160,
		defaultTitleBarH, defaultLeftPanelW, defaultRightPanelW, defaultStatusBarH)

	assert.Equal(t, defaultLeftPanelW, l.LeftPanel.W)
	assert.Equal(t, defaultRightPanelW, l.RightPanel.W)
	assert.Equal(t, defaultTitleBarH, l.TitleBar.H)
	assert.Equal(t, defaultStatusBarH, l.StatusBar.H)

	// Canvas takes everything else (after subtracting the menu bar +
	// tab strip + title bar + status bar).
	assert.Equal(t, 3840-defaultLeftPanelW-defaultRightPanelW, l.Canvas.W)
	bodyH := 2160 - chromeTopBandH(defaultTitleBarH) - defaultStatusBarH
	assert.Equal(t, bodyH, l.Canvas.H)
}

// New() returns a usable Editor with a chrome layout attached. Update
// returns nil; Layout passes through the window dimensions.
func TestEditor_NewAndLayoutPassThrough(t *testing.T) {
	e := New()
	require.NotNil(t, e)
	require.NotNil(t, e.chrome)

	assert.NoError(t, e.Update())

	w, h := e.Layout(800, 600)
	assert.Equal(t, 800, w)
	assert.Equal(t, 600, h)
}

// clampMax behaves as documented: target inside [min, max] passes through;
// outside is clamped; max < min collapses to min.
func TestClampMax(t *testing.T) {
	assert.Equal(t, 50, clampMax(50, 0, 100))
	assert.Equal(t, 0, clampMax(-1, 0, 100))
	assert.Equal(t, 100, clampMax(200, 0, 100))
	// max < min: result is min (we never go below the floor)
	assert.Equal(t, 50, clampMax(10, 50, 5))
}
