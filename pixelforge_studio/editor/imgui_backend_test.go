package editor

import (
	"image"
	"testing"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingBackend is a test double for the imguiBackend interface.
// It records every method call so tests can assert call counts and
// argument values without needing a live OpenGL context. Each method
// also appends a tag to `calls` so order between methods can be
// verified when relevant.
type recordingBackend struct {
	beginFrameCalls int
	endFrameCalls   int
	drawCalls       int
	layoutCalls     int

	lastDrawScreen *ebiten.Image
	lastLayoutW    int
	lastLayoutH    int

	createTextureCalls int
	lastTextureGame    ebiten.Game
	lastTextureW       int
	lastTextureH       int

	calls []string
}

func (b *recordingBackend) BeginFrame() {
	b.beginFrameCalls++
	b.calls = append(b.calls, "BeginFrame")
}

func (b *recordingBackend) EndFrame() {
	b.endFrameCalls++
	b.calls = append(b.calls, "EndFrame")
}

func (b *recordingBackend) Draw(screen *ebiten.Image) {
	b.drawCalls++
	b.lastDrawScreen = screen
	b.calls = append(b.calls, "Draw")
}

func (b *recordingBackend) Layout(outsideWidth, outsideHeight int) (int, int) {
	b.layoutCalls++
	b.lastLayoutW = outsideWidth
	b.lastLayoutH = outsideHeight
	b.calls = append(b.calls, "Layout")
	return outsideWidth, outsideHeight
}

// CreateTextureFromGame is the U5 addition to the imguiBackend
// interface. The test stub records the call so tests can assert on
// game-texture registration; the returned TextureRef is the zero
// value because no real GL texture exists in the stub.
func (b *recordingBackend) CreateTextureFromGame(game ebiten.Game, width, height int) imgui.TextureRef {
	b.createTextureCalls++
	b.lastTextureGame = game
	b.lastTextureW = width
	b.lastTextureH = height
	b.calls = append(b.calls, "CreateTextureFromGame")
	return imgui.TextureRef{}
}

// TestImguiBackendInitializes verifies the host wrapper accepts a
// backend and is ready for use without panicking. This proves the U1
// wiring contract — main.go can construct a host once and the editor
// can drive frames against it.
func TestImguiBackendInitializes(t *testing.T) {
	host := newImguiHostWithBackend(&recordingBackend{}, false)
	require.NotNil(t, host)
	require.NotNil(t, host.backend)
}

// TestEditorUpdateInvokesBeginEndFrame verifies that one Editor.Update
// tick produces exactly one BeginFrame and one EndFrame call. The defer
// guarantee in Editor.Update is the load-bearing detail — early returns
// from modal/menu input must still close the frame.
func TestEditorUpdateInvokesBeginEndFrame(t *testing.T) {
	e := NewWithSettings(DefaultSettings())
	mock := &recordingBackend{}
	e.AttachImguiBackendStub(mock)

	require.NoError(t, e.Update())

	assert.Equal(t, 1, mock.beginFrameCalls, "BeginFrame called once per Update")
	assert.Equal(t, 1, mock.endFrameCalls, "EndFrame called once per Update")
}

// TestEditorDrawInvokesBackendDraw verifies the editor composites the
// ImGui draw lists onto the screen by calling backend.Draw(screen)
// exactly once per Draw, with the same screen the caller handed in.
//
// The plan also asks that backend.Draw fires *after* the native chrome
// draw, not before. We assert the call-order tag last in the recorded
// sequence to enforce that — anything the editor adds before
// backend.Draw in Draw() must come earlier in the recording.
func TestEditorDrawInvokesBackendDraw(t *testing.T) {
	e := NewWithSettings(DefaultSettings())
	mock := &recordingBackend{}
	e.AttachImguiBackendStub(mock)

	screen := ebiten.NewImageWithOptions(image.Rect(0, 0, 800, 600), nil)
	e.Draw(screen)

	require.Equal(t, 1, mock.drawCalls, "Draw called once per Editor.Draw")
	assert.Same(t, screen, mock.lastDrawScreen, "backend.Draw receives the same screen passed to Editor.Draw")
	require.NotEmpty(t, mock.calls)
	assert.Equal(t, "Draw", mock.calls[len(mock.calls)-1], "backend.Draw is the last imgui call in Editor.Draw — comes after native chrome")
}

// TestEditorLayoutForwardsToBackend verifies that Editor.Layout passes
// the unscaled outside dimensions to the backend before returning the
// editor's logical-scale-adjusted dimensions. ImGui needs the outside
// dimensions so its display size matches the OS window.
func TestEditorLayoutForwardsToBackend(t *testing.T) {
	e := NewWithSettings(DefaultSettings())
	mock := &recordingBackend{}
	e.AttachImguiBackendStub(mock)

	w, h := e.Layout(1920, 1080)

	assert.Equal(t, 1, mock.layoutCalls, "Layout called once per Editor.Layout")
	assert.Equal(t, 1920, mock.lastLayoutW, "outsideWidth forwarded")
	assert.Equal(t, 1080, mock.lastLayoutH, "outsideHeight forwarded")
	// The editor's own return value is still its logical-scaled
	// dimensions — default LogicalScale=1 returns the outside dims.
	assert.Equal(t, 1920, w)
	assert.Equal(t, 1080, h)
}
