package capture

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspace_NameAndDisplay(t *testing.T) {
	w := NewWorkspace(8)
	assert.Equal(t, "capture", w.Name())
	assert.Equal(t, "Capture", w.DisplayName())
}

func TestWorkspace_HasRecorder(t *testing.T) {
	w := NewWorkspace(8)
	require.NotNil(t, w.Recorder())
	assert.Equal(t, 8, w.Recorder().Budget())
}

func TestWorkspace_DefaultBudgetClampsZero(t *testing.T) {
	w := NewWorkspace(0)
	assert.Equal(t, DefaultBudgetFrames, w.Recorder().Budget())
}

func TestWorkspace_MarkRangeOrders(t *testing.T) {
	w := NewWorkspace(8)
	w.SetMarkRange(20, 5)
	s, e := w.MarkRange()
	assert.Equal(t, 5, s)
	assert.Equal(t, 20, e)
}

func TestWorkspace_ClearMark(t *testing.T) {
	w := NewWorkspace(8)
	w.SetMarkRange(2, 6)
	w.ClearMark()
	s, e := w.MarkRange()
	assert.Equal(t, -1, s)
	assert.Equal(t, -1, e)
}

func TestRegisterWith_ReplacesStubByName(t *testing.T) {
	e := editor.New()
	// Before RegisterWith the stub is in place (installed by the
	// editor's installStubWorkspaces).
	stub := findWorkspace(e, "capture")
	require.NotNil(t, stub, "M3 capture stub must be registered by default")

	w := RegisterWith(e)
	require.NotNil(t, w)

	// After RegisterWith the slot is the real workspace, same name,
	// not the stub.
	live := findWorkspace(e, "capture")
	require.NotNil(t, live)
	_, isStub := live.(interface{ Recorder() *Recorder })
	assert.True(t, isStub, "after RegisterWith, capture slot should expose Recorder()")
}

func TestRegisterWith_RegistersCaptureKeymap(t *testing.T) {
	e := editor.New()
	RegisterWith(e)
	km := e.KeyMap()
	require.NotNil(t, km)
	for _, action := range []string{
		"capture.toggle_mark",
		"capture.clear_mark",
		"capture.save_clip",
		"capture.export_gif",
		"capture.export_mp4",
		"capture.promote_regression",
		"capture.bug_report",
	} {
		assert.NotEmpty(t, km.BindingsFor(action), "missing binding for %s", action)
	}
}

func TestRegisterWith_RecorderRunningAfterRegistration(t *testing.T) {
	e := editor.New()
	w := RegisterWith(e)
	assert.True(t, w.Recorder().Running())
}

func findWorkspace(e *editor.Editor, name string) editor.Workspace {
	for _, w := range e.Workspaces() {
		if w.Name() == name {
			return w
		}
	}
	return nil
}

func TestWorkspace_Status(t *testing.T) {
	w := NewWorkspace(8)
	w.SetStatus("clip saved")
	assert.Equal(t, "clip saved", w.Status())
}
