package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// U2 deleted chrome.go and its tests (TestChromeLayout_*, TestClampMax,
// chromeTopBandH). ImGui owns chrome geometry now; layout invariants
// live in dear-imgui itself, not in editor code. The surviving editor-
// level test verifies the constructor and Layout pass-through.

// New() returns a usable Editor. Update returns nil; Layout passes
// through the window dimensions when LogicalScale=1.
func TestEditor_NewAndLayoutPassThrough(t *testing.T) {
	e := New()
	require.NotNil(t, e)

	assert.NoError(t, e.Update())

	w, h := e.Layout(800, 600)
	assert.Equal(t, 800, w)
	assert.Equal(t, 600, h)
}
