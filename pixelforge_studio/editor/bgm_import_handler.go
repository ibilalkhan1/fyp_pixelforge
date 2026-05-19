// bgm_import_handler.go owns plan-009 U19's editor-side BGM-import
// orchestration. BGM (OGG / MP3) drops + future File → Import BGM…
// menu entries dispatch into BGMImportHandler, which delegates to a
// BGMImportRunner registered by audiolib.RegisterWith.
//
// Same cycle-breaking shape as the PNG / WAV import paths: audiolib
// already imports editor, so the editor declares the contract and
// audiolib injects the implementation at studio startup.
//
// OGG and MP3 share the BGMImportResult shape — the runner detects
// container type via path extension. A single BGMImportRunner
// interface keeps the editor's surface area small; separate decoder
// backends live on the audiolib side.
package editor

// BGMImportResult mirrors audiolib.BGMImportResult at the editor's
// import boundary so this package never has to import audiolib.
// IsBGM is always true on this path (mirrors WAVImportResult.IsBGM
// for shape consistency with the audio surface).
type BGMImportResult struct {
	SampleName   string
	RelativePath string
	DurationMs   int
	Container    string // "ogg" or "mp3"
}

// BGMImportRunner is the contract audiolib.RegisterWith satisfies
// for plan-009 R17. ImportPath drives the on-disk decode pipeline;
// the runner decides container type from the path extension.
type BGMImportRunner interface {
	ImportPath(bgmPath string) (*BGMImportResult, error)
}

// BGMImportHandler carries the editor's pending BGM-import state.
// Mirrors AudioImportHandler so the per-kind audio handlers share a
// uniform feel from the editor's perspective.
type BGMImportHandler struct {
	editor *Editor
	runner BGMImportRunner

	lastResult *BGMImportResult
	lastErr    error
}

// NewBGMImportHandler binds a handler to editor.
func NewBGMImportHandler(e *Editor) *BGMImportHandler {
	return &BGMImportHandler{editor: e}
}

// BGMImportHandler returns the editor's bound handler. Nil-safe.
func (e *Editor) BGMImportHandler() *BGMImportHandler {
	if e == nil {
		return nil
	}
	return e.bgmImportHandler
}

// SetRunner installs the BGMImportRunner. Called by
// audiolib.RegisterWith at studio startup.
func (h *BGMImportHandler) SetRunner(r BGMImportRunner) {
	if h == nil {
		return
	}
	h.runner = r
}

// Import is the entry point the ingest dispatcher (and a future
// File → Import BGM menu) calls. Runs the on-disk pipeline through
// the wired runner; mirrors AudioImportHandler.Import.
func (h *BGMImportHandler) Import(bgmPath string) (*BGMImportResult, error) {
	if h == nil || h.editor == nil || h.runner == nil {
		return nil, errBGMImportPreconditions
	}
	res, err := h.runner.ImportPath(bgmPath)
	if err != nil {
		h.editor.SetStatusMessage("import BGM failed: " + err.Error())
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
func (h *BGMImportHandler) LastResult() *BGMImportResult {
	if h == nil {
		return nil
	}
	return h.lastResult
}

// LastErr exposes the most-recent import error (cleared on next
// successful import).
func (h *BGMImportHandler) LastErr() error {
	if h == nil {
		return nil
	}
	return h.lastErr
}

// errBGMImportPreconditions surfaces when the dispatcher fires
// before the runner has been wired (expected in tests that don't
// register audiolib).
var errBGMImportPreconditions = &bgmImportError{
	msg: "bgm import: editor or runner not ready",
}

type bgmImportError struct{ msg string }

func (e *bgmImportError) Error() string { return e.msg }
