// import_handler.go owns the editor's PNG-import orchestration. The
// File → Import PNG… menu entry opens the file picker, then on
// confirm dispatches into ImportHandler.Import which delegates to a
// PNGImportRunner registered by an outside package (palette.
// RegisterWith) and queues the resulting diff for the U4 modal.
//
// The runner indirection keeps the editor package at the bottom of
// the import graph: palette already imports editor, so the inverse
// dependency would create a cycle. The editor exposes the registry;
// palette injects the implementation at studio startup.
package editor

import "image"

// PNGImportResult mirrors palette.ImportResult at the editor's
// import boundary so this package never has to import palette.
type PNGImportResult struct {
	SpriteName    string
	RegisteredIdx int
	Width         int
	Height        int
	Diff          *PNGImportDiff
}

// PNGImportDiff mirrors palette.ImportDiff. Carries the side-by-
// side data the U4 diff modal renders.
type PNGImportDiff struct {
	OriginalImage          image.Image
	QuantizedReconstructed image.Image
	ChosenSubPalette       string
	AutoPicked             bool
	MeanDeltaE             float64
	Width                  int
	Height                 int
}

// PNGImportRunner is the contract palette.RegisterWith satisfies.
// Two methods: ImportPath drives the on-disk pipeline; ImportImage
// drives the in-memory pipeline for tests + future drag-drop.
// Re-quantize re-runs the import against a different sub-palette
// after reusing the previously-loaded image.
type PNGImportRunner interface {
	ImportPath(pngPath string) (*PNGImportResult, error)
	ImportImage(img image.Image, pngPath string) (*PNGImportResult, error)
	Requantize(img image.Image, pngPath, targetSubPalette string) (*PNGImportResult, error)
	RollbackLastImport(idx int)
}

// ImportHandler carries the editor's pending PNG-import state.
type ImportHandler struct {
	editor *Editor
	runner PNGImportRunner

	pendingResult     *PNGImportResult
	pendingSpriteIdx  int
	pendingSourcePath string
}

// NewImportHandler binds a handler to editor. runner is nil at
// startup; palette.RegisterWith populates it via SetRunner.
func NewImportHandler(e *Editor) *ImportHandler {
	return &ImportHandler{editor: e, pendingSpriteIdx: -1}
}

// ImportHandler returns the editor's bound handler. Nil-safe.
func (e *Editor) ImportHandler() *ImportHandler {
	if e == nil {
		return nil
	}
	return e.importHandler
}

// SetRunner installs the PNGImportRunner implementation. Called by
// palette.RegisterWith at studio startup. Replacing the runner mid-
// session is allowed (tests may swap in a stub).
func (h *ImportHandler) SetRunner(r PNGImportRunner) {
	if h == nil {
		return
	}
	h.runner = r
}

// Import is the entry point the File menu (and drag-drop, when
// wired) calls. Delegates to the wired runner; stashes the result
// for the diff modal.
func (h *ImportHandler) Import(pngPath string) (*PNGImportResult, error) {
	if h == nil || h.editor == nil || h.runner == nil {
		return nil, errImportPreconditions
	}
	res, err := h.runner.ImportPath(pngPath)
	if err != nil {
		h.editor.SetStatusMessage("import failed: " + err.Error())
		return nil, err
	}
	h.pendingResult = res
	h.pendingSpriteIdx = res.RegisteredIdx
	h.pendingSourcePath = pngPath
	h.editor.SetStatusMessage("imported " + res.SpriteName + "; review the diff")
	return res, nil
}

// ImportImage is the in-memory variant. Used by tests and the
// future drag-drop handler.
func (h *ImportHandler) ImportImage(img image.Image, pngPath string) (*PNGImportResult, error) {
	if h == nil || h.editor == nil || h.runner == nil {
		return nil, errImportPreconditions
	}
	res, err := h.runner.ImportImage(img, pngPath)
	if err != nil {
		return nil, err
	}
	h.pendingResult = res
	h.pendingSpriteIdx = res.RegisteredIdx
	h.pendingSourcePath = pngPath
	return res, nil
}

// PendingResult exposes the queued ImportResult so the U4 diff
// modal reads from one source of truth.
func (h *ImportHandler) PendingResult() *PNGImportResult {
	if h == nil {
		return nil
	}
	return h.pendingResult
}

// Accept commits the pending import. The sprite is already on the
// project (runner.ImportPath appended it); Accept just clears the
// pending state + marks the project dirty.
func (h *ImportHandler) Accept() {
	if h == nil || h.pendingResult == nil {
		return
	}
	h.editor.MarkDirty()
	h.editor.SetStatusMessage("imported " + h.pendingResult.SpriteName)
	h.clearPending()
}

// Reject rolls back the pending import via the runner's rollback
// hook. The runner is responsible for the specific truncation
// because only it knows the underlying SpriteAsset shape.
func (h *ImportHandler) Reject() {
	if h == nil || h.pendingResult == nil || h.runner == nil {
		return
	}
	h.runner.RollbackLastImport(h.pendingSpriteIdx)
	h.editor.SetStatusMessage("import rejected")
	h.clearPending()
}

// Requantize re-runs the import with a different target sub-palette.
// Rolls back the original sprite first so the re-import lands at
// the same logical "freshly imported, awaiting review" state.
func (h *ImportHandler) Requantize(targetSubPalette string) (*PNGImportResult, error) {
	if h == nil || h.editor == nil || h.runner == nil {
		return nil, errImportPreconditions
	}
	if h.pendingResult == nil {
		return nil, errNoPendingImport
	}

	var img image.Image
	if h.pendingResult.Diff != nil && h.pendingResult.Diff.OriginalImage != nil {
		img = h.pendingResult.Diff.OriginalImage
	} else {
		return nil, errNoPendingImport
	}

	// Capture the source path before clearPending wipes it.
	srcPath := h.pendingSourcePath

	h.runner.RollbackLastImport(h.pendingSpriteIdx)
	h.clearPending()

	res, err := h.runner.Requantize(img, srcPath, targetSubPalette)
	if err != nil {
		return nil, err
	}
	h.pendingResult = res
	h.pendingSpriteIdx = res.RegisteredIdx
	h.pendingSourcePath = srcPath
	if res.Diff != nil {
		res.Diff.AutoPicked = false
	}
	return res, nil
}

func (h *ImportHandler) clearPending() {
	h.pendingResult = nil
	h.pendingSpriteIdx = -1
	h.pendingSourcePath = ""
}

// Sentinel error values so tests can assert specifically.
var (
	errImportPreconditions = &importError{msg: "import: editor or runner not ready"}
	errNoPendingImport     = &importError{msg: "import: no pending import to act on"}
)

type importError struct{ msg string }

func (e *importError) Error() string { return e.msg }
