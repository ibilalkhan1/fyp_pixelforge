package editor

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// audio_import_handler_test.go covers idea #4 v1 U8: the editor-
// side WAV-import handler + the File / View menu entries. The
// runner is mocked so the tests don't touch audiolib (which
// imports editor and would create a cycle).

type stubWAVRunner struct {
	imports []string
	nextRes *WAVImportResult
	nextErr error
}

func (s *stubWAVRunner) ImportPath(wavPath string) (*WAVImportResult, error) {
	s.imports = append(s.imports, wavPath)
	return s.deliver()
}

func (s *stubWAVRunner) ImportLibraryPatch(patch string) (*WAVImportResult, error) {
	return s.deliver()
}

func (s *stubWAVRunner) deliver() (*WAVImportResult, error) {
	if s.nextErr != nil {
		err := s.nextErr
		s.nextErr = nil
		return nil, err
	}
	out := s.nextRes
	s.nextRes = nil
	return out, nil
}

func newAudioHandlerWithStub(t *testing.T) (*Editor, *AudioImportHandler, *stubWAVRunner) {
	t.Helper()
	e := New()
	require.NotNil(t, e.AudioImportHandler())
	stub := &stubWAVRunner{}
	e.AudioImportHandler().SetRunner(stub)
	return e, e.AudioImportHandler(), stub
}

func TestAudioImportHandler_ImportDelegatesToRunner(t *testing.T) {
	_, h, stub := newAudioHandlerWithStub(t)
	stub.nextRes = &WAVImportResult{SampleName: "jump", RelativePath: "audio/jump.wav"}
	res, err := h.Import("/tmp/jump.wav")
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "jump", res.SampleName)
	assert.Equal(t, []string{"/tmp/jump.wav"}, stub.imports)
}

func TestAudioImportHandler_ImportMarksDirtyOnSuccess(t *testing.T) {
	e, h, stub := newAudioHandlerWithStub(t)
	require.False(t, e.IsDirty())
	stub.nextRes = &WAVImportResult{SampleName: "ok"}
	_, err := h.Import("/tmp/ok.wav")
	require.NoError(t, err)
	assert.True(t, e.IsDirty())
}

func TestAudioImportHandler_ImportErrorBubblesAndDoesNotDirty(t *testing.T) {
	e, h, stub := newAudioHandlerWithStub(t)
	stub.nextErr = errors.New("decode failed")
	_, err := h.Import("/tmp/bad.wav")
	require.Error(t, err)
	assert.False(t, e.IsDirty(), "failed import does not mark dirty")
	assert.NotNil(t, h.LastErr())
	assert.Nil(t, h.LastResult(), "failed import leaves LastResult cleared")
}

func TestAudioImportHandler_ImportWithoutRunnerErrors(t *testing.T) {
	e := New()
	h := e.AudioImportHandler()
	require.NotNil(t, h)
	_, err := h.Import("/tmp/x.wav")
	require.Error(t, err)
}

func TestAudioImportHandler_LastResultExposedAfterSuccess(t *testing.T) {
	_, h, stub := newAudioHandlerWithStub(t)
	stub.nextRes = &WAVImportResult{SampleName: "stash"}
	_, _ = h.Import("/tmp/stash.wav")
	require.NotNil(t, h.LastResult())
	assert.Equal(t, "stash", h.LastResult().SampleName)
}

func TestFileMenu_ImportWAVOpensPicker(t *testing.T) {
	e := New()
	e.fileMenu.ImportWAV()
	assert.True(t, e.filePicker.Visible(),
		"Import WAV opens the file picker")
}

func TestFileMenu_BuildMenuDefsContainsImportWAVAndAudio(t *testing.T) {
	e := New()
	defs := e.buildMenuDefs()
	importFound := false
	audioFound := false
	for _, def := range defs {
		if def.Label == "File" {
			for _, item := range def.Items {
				if item.Label == "Import WAV..." {
					importFound = true
				}
			}
		}
		if def.Label == "View" {
			for _, item := range def.Items {
				if item.Label == "Audio" {
					audioFound = true
				}
			}
		}
	}
	assert.True(t, importFound, "File menu includes Import WAV entry")
	assert.True(t, audioFound, "View menu includes Audio entry")
}
