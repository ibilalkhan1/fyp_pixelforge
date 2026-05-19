package buildpipeline

import "fmt"

// Plan-009 U23 — WASM size reporting.
//
// The .html bundle the WASM builder emits inlines the entire
// player runtime + the game cart as base64. That payload is ~1.33×
// the raw WASM byte count plus a fixed template overhead. For real
// Pixelforge projects we expect 18-25MB raw uncompressed; gzip
// halves that on the wire. The size-report type below is what the
// builder feeds back to the studio so the success toast can name
// the cost (and warn / error at the budget thresholds).

// WASM size thresholds. Warn at 15MB raw HTML, hard-error at 30MB
// (unless BuildRequest.ForceLargeWASM is set). Values are picked
// against the "playable on a phone over 4G" UX budget — see the
// plan's R21 line for the empirical baseline rationale.
const (
	WASMWarnThresholdMB  = 15
	WASMErrorThresholdMB = 30

	WASMWarnThreshold  int64 = WASMWarnThresholdMB * 1024 * 1024  // bytes
	WASMErrorThreshold int64 = WASMErrorThresholdMB * 1024 * 1024 // bytes
)

// WASMSizeReport is the size summary the WASM bundler hands back
// to the build orchestrator. Carried inside BuildStatus.SizeReport
// so the studio Build workspace can surface it in the success
// toast. Nil for non-WASM builds.
type WASMSizeReport struct {
	// RawBytes is the size of the bundled .html on disk.
	RawBytes int64
	// GzipBytes is the size of the sibling .html.gz on disk.
	GzipBytes int64
	// WarnThresholdMB / ErrorThresholdMB are the configured budgets
	// at the time the report was generated. Carried so consumers
	// don't have to consult the package constant — makes
	// per-project overrides (a future polish) a 1-line change.
	WarnThresholdMB  int
	ErrorThresholdMB int
}

// ExceededWarn reports whether the raw HTML size crossed the warn
// budget. False means "everything's fine, no indicator". True
// means "show a yellow indicator".
func (r WASMSizeReport) ExceededWarn() bool {
	return r.RawBytes > int64(r.WarnThresholdMB)*1024*1024
}

// ExceededError reports whether the raw HTML size crossed the
// hard-error budget. True means "show a red indicator AND fail
// the build unless ForceLargeWASM is set".
func (r WASMSizeReport) ExceededError() bool {
	return r.RawBytes > int64(r.ErrorThresholdMB)*1024*1024
}

// Severity classifies the report into one of three buckets the
// toast UI maps to colour codes. Kept as a string so callers don't
// have to import any UI vocabulary.
type WASMSizeSeverity string

const (
	WASMSeverityOK    WASMSizeSeverity = "ok"    // green / no indicator
	WASMSeverityWarn  WASMSizeSeverity = "warn"  // yellow
	WASMSeverityError WASMSizeSeverity = "error" // red
)

// Severity returns the report's classification. Picks the most
// severe label that applies (error > warn > ok).
func (r WASMSizeReport) Severity() WASMSizeSeverity {
	switch {
	case r.ExceededError():
		return WASMSeverityError
	case r.ExceededWarn():
		return WASMSeverityWarn
	default:
		return WASMSeverityOK
	}
}

// Format renders a "(18.0MB, gzip 6.4MB)" style suffix the toast
// composes into its final string. The leading parenthesis +
// trailing parenthesis are part of the output so callers can drop
// it straight after a filename. Single decimal point keeps the
// readout compact without losing meaningful precision at the MB
// scale.
func (r WASMSizeReport) Format() string {
	return fmt.Sprintf("(%.1fMB, gzip %.1fMB)", bytesToMB(r.RawBytes), bytesToMB(r.GzipBytes))
}

// bytesToMB converts a byte count to a float MB value using the
// same 1024-based units the thresholds are expressed in.
func bytesToMB(b int64) float64 {
	return float64(b) / (1024.0 * 1024.0)
}
