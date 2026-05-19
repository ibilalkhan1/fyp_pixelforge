// audio_import_handler.go owns idea #4 v1 U8's editor-side WAV
// import orchestration. The File → Import WAV menu opens the file
// picker and dispatches into AudioImportHandler, which delegates
// to a WAVImportRunner registered by audiolib.RegisterWith.
//
// Same cycle-breaking shape as the PNG import path: audiolib
// already imports editor, so the editor must hold the contract
// and audiolib injects the implementation.
package editor

// WAVImportResult mirrors audiolib.ImportResult at the editor's
// import boundary. The editor surfaces SampleName + RelativePath
// in the toast / status bar; DurationMs + IsBGM are informational.
type WAVImportResult struct {
	SampleName   string
	RelativePath string
	DurationMs   int
	IsBGM        bool
}

// WAVImportRunner is the contract audiolib.RegisterWith satisfies.
// ImportPath drives the on-disk pipeline; ImportLibraryPatch
// materializes a bundled patch by name (the picker's Bind button
// uses this path).
type WAVImportRunner interface {
	ImportPath(wavPath string) (*WAVImportResult, error)
	ImportLibraryPatch(patchName string) (*WAVImportResult, error)
}

// AudioImportHandler carries the editor's pending WAV-import
// state. Bound to *Editor at construction; the menu entry reaches
// for it via *Editor.AudioImportHandler().
type AudioImportHandler struct {
	editor *Editor
	runner WAVImportRunner

	lastResult *WAVImportResult
	lastErr    error
}

// NewAudioImportHandler binds a handler to editor.
func NewAudioImportHandler(e *Editor) *AudioImportHandler {
	return &AudioImportHandler{editor: e}
}

// AudioImportHandler returns the editor's bound handler. Nil-safe.
func (e *Editor) AudioImportHandler() *AudioImportHandler {
	if e == nil {
		return nil
	}
	return e.audioImportHandler
}

// SetRunner installs the WAVImportRunner. Called by
// audiolib.RegisterWith at studio startup.
func (h *AudioImportHandler) SetRunner(r WAVImportRunner) {
	if h == nil {
		return
	}
	h.runner = r
}

// Import is the entry point the File → Import WAV menu calls. Runs
// the on-disk pipeline through the wired runner.
func (h *AudioImportHandler) Import(wavPath string) (*WAVImportResult, error) {
	if h == nil || h.editor == nil || h.runner == nil {
		return nil, errAudioImportPreconditions
	}
	res, err := h.runner.ImportPath(wavPath)
	if err != nil {
		h.editor.SetStatusMessage("import WAV failed: " + err.Error())
		h.lastErr = err
		h.lastResult = nil
		return nil, err
	}
	h.editor.MarkDirty()
	h.editor.SetStatusMessage("imported " + res.SampleName)
	h.lastResult = res
	h.lastErr = nil
	return res, nil
}

// LastResult exposes the most-recent successful import for tests +
// the status bar.
func (h *AudioImportHandler) LastResult() *WAVImportResult {
	if h == nil {
		return nil
	}
	return h.lastResult
}

// LastErr exposes the most-recent import error (cleared on next
// successful import).
func (h *AudioImportHandler) LastErr() error {
	if h == nil {
		return nil
	}
	return h.lastErr
}

// errAudioImportPreconditions surfaces when the menu fires before
// the runner has been wired (which should never happen in
// production but is the expected outcome in tests that haven't
// registered audiolib).
var errAudioImportPreconditions = &audioImportError{
	msg: "audio import: editor or runner not ready",
}

type audioImportError struct{ msg string }

func (e *audioImportError) Error() string { return e.msg }
