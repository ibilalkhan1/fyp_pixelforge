package editor

import (
	"errors"
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// import_handler_test.go covers idea #3 v1 U3's editor-side
// orchestration: Import → Accept / Reject / Re-quantize. The runner
// is mocked so the tests don't touch the palette package (which
// would create an import cycle); the runner contract is the same
// interface palette.RegisterWith implements.

type stubImportRunner struct {
	imports    []string
	imageCalls int
	requantizes []string
	rollbacks   []int

	nextResult *PNGImportResult
	nextErr    error
}

func (s *stubImportRunner) ImportPath(pngPath string) (*PNGImportResult, error) {
	s.imports = append(s.imports, pngPath)
	return s.deliver()
}

func (s *stubImportRunner) ImportImage(img image.Image, pngPath string) (*PNGImportResult, error) {
	s.imageCalls++
	return s.deliver()
}

func (s *stubImportRunner) Requantize(img image.Image, pngPath, target string) (*PNGImportResult, error) {
	s.requantizes = append(s.requantizes, target)
	return s.deliver()
}

func (s *stubImportRunner) RollbackLastImport(idx int) {
	s.rollbacks = append(s.rollbacks, idx)
}

func (s *stubImportRunner) deliver() (*PNGImportResult, error) {
	if s.nextErr != nil {
		err := s.nextErr
		s.nextErr = nil
		return nil, err
	}
	res := s.nextResult
	s.nextResult = nil
	return res, nil
}

func newHandlerWithStubRunner(t *testing.T) (*Editor, *ImportHandler, *stubImportRunner) {
	t.Helper()
	e := New()
	require.NotNil(t, e.ImportHandler())
	stub := &stubImportRunner{}
	e.ImportHandler().SetRunner(stub)
	return e, e.ImportHandler(), stub
}

// TestImportHandler_ImportDelegatesToRunner: Import dispatches to
// the wired runner and stashes the result for the diff modal.
func TestImportHandler_ImportDelegatesToRunner(t *testing.T) {
	_, h, stub := newHandlerWithStubRunner(t)
	stub.nextResult = &PNGImportResult{SpriteName: "hero", RegisteredIdx: 0}

	res, err := h.Import("/tmp/hero.png")
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "hero", res.SpriteName)
	assert.Equal(t, []string{"/tmp/hero.png"}, stub.imports)
	assert.Same(t, res, h.PendingResult(),
		"PendingResult exposes the stashed result for the modal")
}

// TestImportHandler_ImportRunnerErrorBubblesUp: runner errors
// surface to the caller + leave no pending state.
func TestImportHandler_ImportRunnerErrorBubblesUp(t *testing.T) {
	_, h, stub := newHandlerWithStubRunner(t)
	stub.nextErr = errors.New("decode failed")
	_, err := h.Import("/tmp/bad.png")
	require.Error(t, err)
	assert.Nil(t, h.PendingResult())
}

// TestImportHandler_AcceptMarksProjectDirty: Accept flips the dirty
// flag and clears pending state. The runner already appended the
// sprite during Import; Accept just commits.
func TestImportHandler_AcceptMarksProjectDirty(t *testing.T) {
	e, h, stub := newHandlerWithStubRunner(t)
	require.False(t, e.IsDirty())
	stub.nextResult = &PNGImportResult{SpriteName: "hero", RegisteredIdx: 0}
	_, _ = h.Import("/tmp/hero.png")

	h.Accept()
	assert.True(t, e.IsDirty(), "Accept marks dirty")
	assert.Nil(t, h.PendingResult(), "Accept clears pending")
}

// TestImportHandler_RejectCallsRollback: Reject dispatches the
// runner's RollbackLastImport with the pending sprite index.
func TestImportHandler_RejectCallsRollback(t *testing.T) {
	_, h, stub := newHandlerWithStubRunner(t)
	stub.nextResult = &PNGImportResult{SpriteName: "hero", RegisteredIdx: 3}
	_, _ = h.Import("/tmp/hero.png")

	h.Reject()
	assert.Equal(t, []int{3}, stub.rollbacks,
		"rollback called with the pending sprite index")
	assert.Nil(t, h.PendingResult())
}

// TestImportHandler_RequantizeReplacesPending: Re-quantize rolls
// back the original sprite + invokes the runner's Requantize with
// the new target; the new result replaces PendingResult.
func TestImportHandler_RequantizeReplacesPending(t *testing.T) {
	_, h, stub := newHandlerWithStubRunner(t)
	originalImg := image.NewRGBA(image.Rect(0, 0, 1, 1))
	originalImg.Set(0, 0, color.RGBA{R: 0xff, A: 0xff})
	stub.nextResult = &PNGImportResult{
		SpriteName:    "hero",
		RegisteredIdx: 0,
		Diff: &PNGImportDiff{
			OriginalImage:    originalImg,
			ChosenSubPalette: "sprite_0",
			AutoPicked:       true,
		},
	}
	_, _ = h.Import("/tmp/hero.png")

	stub.nextResult = &PNGImportResult{
		SpriteName:    "hero",
		RegisteredIdx: 0,
		Diff: &PNGImportDiff{
			OriginalImage:    originalImg,
			ChosenSubPalette: "sprite_2",
		},
	}
	res, err := h.Requantize("sprite_2")
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, []int{0}, stub.rollbacks, "Re-quantize rolls back first")
	assert.Equal(t, []string{"sprite_2"}, stub.requantizes)
	assert.False(t, res.Diff.AutoPicked,
		"Re-quantize flags the new result as manually chosen")
	assert.Same(t, res, h.PendingResult())
}

// TestImportHandler_RequantizeNoPendingReturnsError: calling
// Requantize without an active import returns a sentinel error.
func TestImportHandler_RequantizeNoPendingReturnsError(t *testing.T) {
	_, h, _ := newHandlerWithStubRunner(t)
	_, err := h.Requantize("sprite_2")
	require.Error(t, err)
}

// TestImportHandler_ImportWithoutRunnerErrors: a handler whose
// runner has never been set returns errImportPreconditions rather
// than panicking.
func TestImportHandler_ImportWithoutRunnerErrors(t *testing.T) {
	e := New()
	h := e.ImportHandler()
	require.NotNil(t, h)
	// Don't call SetRunner.
	_, err := h.Import("/tmp/hero.png")
	require.Error(t, err)
}

// TestFileMenu_ImportPNG_OpensPicker: clicking File → Import PNG…
// opens the file picker filtered to .png.
func TestFileMenu_ImportPNG_OpensPicker(t *testing.T) {
	e := New()
	e.fileMenu.ImportPNG()
	assert.True(t, e.filePicker.Visible(),
		"Import PNG opens the file picker")
}

// TestFileMenu_BuildMenuDefs_ContainsImportPNG: the File menu
// definition includes an "Import PNG..." entry.
func TestFileMenu_BuildMenuDefs_ContainsImportPNG(t *testing.T) {
	e := New()
	defs := e.buildMenuDefs()
	found := false
	for _, def := range defs {
		if def.Label != "File" {
			continue
		}
		for _, item := range def.Items {
			if item.Label == "Import PNG..." {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "File menu includes Import PNG entry")
}

// TestImportHandler_RoundTripsPendingProjectMutations: a successful
// import that lands on the project is visible via Project().Sprites
// before Accept (the runner mutates the project directly). This
// pins the contract palette.RegisterWith satisfies.
func TestImportHandler_RoundTripsPendingProjectMutations(t *testing.T) {
	e, h, stub := newHandlerWithStubRunner(t)
	// Simulate runner appending a sprite into the project before
	// returning the result, just like palette.ImportWithDiff does.
	e.project.Sprites = append(e.project.Sprites, pixelforge_project.SpriteAsset{
		Name: "hero", SubPalette: "sprite_0",
	})
	stub.nextResult = &PNGImportResult{
		SpriteName:    "hero",
		RegisteredIdx: 0,
	}
	_, err := h.Import("/tmp/hero.png")
	require.NoError(t, err)
	assert.Len(t, e.Project().Sprites, 1,
		"runner-side append is visible on the editor's project")
}

