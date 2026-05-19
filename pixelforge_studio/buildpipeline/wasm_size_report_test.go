package buildpipeline_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/buildpipeline"
)

// reportAtMB is a test helper that constructs a WASMSizeReport
// whose raw size is the supplied MB count. Threshold defaults
// match the package's production constants.
func reportAtMB(rawMB, gzipMB float64) buildpipeline.WASMSizeReport {
	return buildpipeline.WASMSizeReport{
		RawBytes:         int64(rawMB * 1024 * 1024),
		GzipBytes:        int64(gzipMB * 1024 * 1024),
		WarnThresholdMB:  buildpipeline.WASMWarnThresholdMB,
		ErrorThresholdMB: buildpipeline.WASMErrorThresholdMB,
	}
}

// TestWASMSizeReport_SmallBuildNoWarnNoError covers the happy
// path: an 8MB raw HTML is well below the 15MB warn budget.
func TestWASMSizeReport_SmallBuildNoWarnNoError(t *testing.T) {
	r := reportAtMB(8.0, 2.5)
	assert.False(t, r.ExceededWarn(), "8MB raw must not trip the 15MB warn budget")
	assert.False(t, r.ExceededError(), "8MB raw must not trip the 30MB error budget")
	assert.Equal(t, buildpipeline.WASMSeverityOK, r.Severity())
}

// TestWASMSizeReport_WarnThresholdAt18MB covers AE9 — 18MB raw
// must trip the warn budget but stay below the error budget.
func TestWASMSizeReport_WarnThresholdAt18MB(t *testing.T) {
	r := reportAtMB(18.0, 6.4)
	assert.True(t, r.ExceededWarn(), "18MB raw must exceed the 15MB warn budget")
	assert.False(t, r.ExceededError(), "18MB raw must stay below the 30MB error budget")
	assert.Equal(t, buildpipeline.WASMSeverityWarn, r.Severity())
}

// TestWASMSizeReport_ErrorThresholdAt35MB covers the hard-stop
// branch: a 35MB build trips both warn AND error budgets.
func TestWASMSizeReport_ErrorThresholdAt35MB(t *testing.T) {
	r := reportAtMB(35.0, 12.5)
	assert.True(t, r.ExceededWarn(), "35MB raw must exceed the 15MB warn budget")
	assert.True(t, r.ExceededError(), "35MB raw must exceed the 30MB error budget")
	assert.Equal(t, buildpipeline.WASMSeverityError, r.Severity())
}

// TestWASMSizeReport_FormatStringAt18MB asserts the exact toast
// suffix shape the plan calls out — "(18.0MB, gzip 6.4MB)".
func TestWASMSizeReport_FormatStringAt18MB(t *testing.T) {
	r := reportAtMB(18.0, 6.4)
	got := r.Format()
	assert.True(t, strings.HasPrefix(got, "("), "format suffix must start with (")
	assert.True(t, strings.HasSuffix(got, ")"), "format suffix must end with )")
	assert.Contains(t, got, "18.0MB", "raw MB count must appear in the format string")
	assert.Contains(t, got, "gzip 6.4MB", "gzip MB count must appear in the format string with the gzip label")
}

// TestWASMSizeReport_FormatStringAt8MB checks the small-build
// format too — same shape, different numbers.
func TestWASMSizeReport_FormatStringAt8MB(t *testing.T) {
	r := reportAtMB(8.0, 2.5)
	got := r.Format()
	assert.Contains(t, got, "8.0MB")
	assert.Contains(t, got, "gzip 2.5MB")
}

// TestWASMSizeReport_ThresholdConstantsMatchPlan locks in the
// 15MB/30MB plan values so an accidental edit fails this test
// before reaching CI.
func TestWASMSizeReport_ThresholdConstantsMatchPlan(t *testing.T) {
	assert.Equal(t, 15, buildpipeline.WASMWarnThresholdMB)
	assert.Equal(t, 30, buildpipeline.WASMErrorThresholdMB)
	assert.Equal(t, int64(15*1024*1024), buildpipeline.WASMWarnThreshold)
	assert.Equal(t, int64(30*1024*1024), buildpipeline.WASMErrorThreshold)
}

// TestWASMSizeReport_ExactlyAtWarnBoundary covers the boundary
// edge: exactly 15MB is NOT a warn (strictly greater triggers it).
func TestWASMSizeReport_ExactlyAtWarnBoundary(t *testing.T) {
	r := buildpipeline.WASMSizeReport{
		RawBytes:         buildpipeline.WASMWarnThreshold,
		GzipBytes:        5 * 1024 * 1024,
		WarnThresholdMB:  buildpipeline.WASMWarnThresholdMB,
		ErrorThresholdMB: buildpipeline.WASMErrorThresholdMB,
	}
	assert.False(t, r.ExceededWarn(), "exactly at warn boundary must not trip")
}

// TestWASMSizeReport_OneByteOverWarnTriggers covers the other
// side of the boundary — one byte past the warn threshold trips.
func TestWASMSizeReport_OneByteOverWarnTriggers(t *testing.T) {
	r := buildpipeline.WASMSizeReport{
		RawBytes:         buildpipeline.WASMWarnThreshold + 1,
		GzipBytes:        5 * 1024 * 1024,
		WarnThresholdMB:  buildpipeline.WASMWarnThresholdMB,
		ErrorThresholdMB: buildpipeline.WASMErrorThresholdMB,
	}
	assert.True(t, r.ExceededWarn(), "one byte past warn threshold must trip")
}
