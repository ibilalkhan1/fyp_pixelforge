package editor

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bgm_import_handler_test.go covers plan-009 U19's editor-side BGM-
// import handler. The runner is mocked so the tests don't touch
// audiolib (which imports editor and would create a cycle).

type stubBGMRunner struct {
	imports []string
	nextRes *BGMImportResult
	nextErr error
}

func (s *stubBGMRunner) ImportPath(path string) (*BGMImportResult, error) {
	s.imports = append(s.imports, path)
	if s.nextErr != nil {
		err := s.nextErr
		s.nextErr = nil
		return nil, err
	}
	out := s.nextRes
	s.nextRes = nil
	return out, nil
}

func newBGMHandlerWithStub(t *testing.T) (*Editor, *BGMImportHandler, *stubBGMRunner) {
	t.Helper()
	e := New()
	require.NotNil(t, e.BGMImportHandler())
	stub := &stubBGMRunner{}
	e.BGMImportHandler().SetRunner(stub)
	return e, e.BGMImportHandler(), stub
}

func TestBGMImportHandler_ImportDelegatesToRunner(t *testing.T) {
	_, h, stub := newBGMHandlerWithStub(t)
	stub.nextRes = &BGMImportResult{
		SampleName:   "title_loop",
		RelativePath: "audio/title_loop.ogg",
		Container:    "ogg",
	}
	res, err := h.Import("/tmp/title_loop.ogg")
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "title_loop", res.SampleName)
	assert.Equal(t, "ogg", res.Container)
	assert.Equal(t, []string{"/tmp/title_loop.ogg"}, stub.imports)
}

func TestBGMImportHandler_ImportMarksDirtyOnSuccess(t *testing.T) {
	e, h, stub := newBGMHandlerWithStub(t)
	require.False(t, e.IsDirty())
	stub.nextRes = &BGMImportResult{SampleName: "ok", Container: "mp3"}
	_, err := h.Import("/tmp/ok.mp3")
	require.NoError(t, err)
	assert.True(t, e.IsDirty())
}

func TestBGMImportHandler_ImportErrorBubblesAndDoesNotDirty(t *testing.T) {
	e, h, stub := newBGMHandlerWithStub(t)
	stub.nextErr = errors.New("decode failed")
	_, err := h.Import("/tmp/bad.ogg")
	require.Error(t, err)
	assert.False(t, e.IsDirty(), "failed import does not mark dirty")
	assert.NotNil(t, h.LastErr())
	assert.Nil(t, h.LastResult(), "failed import leaves LastResult cleared")
}

func TestBGMImportHandler_ImportWithoutRunnerErrors(t *testing.T) {
	e := New()
	h := e.BGMImportHandler()
	require.NotNil(t, h)
	_, err := h.Import("/tmp/x.ogg")
	require.Error(t, err)
}

func TestBGMImportHandler_LastResultExposedAfterSuccess(t *testing.T) {
	_, h, stub := newBGMHandlerWithStub(t)
	stub.nextRes = &BGMImportResult{SampleName: "stash", Container: "ogg"}
	_, _ = h.Import("/tmp/stash.ogg")
	require.NotNil(t, h.LastResult())
	assert.Equal(t, "stash", h.LastResult().SampleName)
}
