package catalog_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
)

// writeBaseline serialises b to <dir>/<name>.baseline.json and returns
// the full path. Helper for the synthetic-fixture coverage tests so
// each scenario produces a clean, isolated tempdir.
func writeBaseline(t *testing.T, dir, name string, b catalog.Baseline) string {
	t.Helper()
	path := filepath.Join(dir, name+".baseline.json")
	data, err := json.MarshalIndent(b, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))
	return path
}

// TestBuildCoverageReport_DetectsUsedTopics builds a synthetic
// baseline that exercises apply_thrust + screen_wrap and asserts both
// appear in the Used set.
func TestBuildCoverageReport_DetectsUsedTopics(t *testing.T) {
	dir := t.TempDir()
	path := writeBaseline(t, dir, "synthetic_used", catalog.Baseline{
		SchemaVersion: 1,
		Game:          "synthetic_used",
		Checkpoints: []catalog.BaselineCheckpoint{
			{
				Tick: 0,
				EventsAtTick: []catalog.BaselineEventArgs{
					{Topic: catalog.EventTopicMotionApplyThrust, Args: map[string]any{}},
					{Topic: catalog.EventTopicMotionScreenWrap, Args: map[string]any{}},
				},
			},
		},
	})

	report, err := catalog.BuildCoverageReport([]string{path})
	require.NoError(t, err)

	assert.Contains(t, report.Used, catalog.EventTopicMotionApplyThrust)
	assert.Contains(t, report.Used, catalog.EventTopicMotionScreenWrap)
	assert.Equal(t, []string{"synthetic_used"},
		report.UsedBy[catalog.EventTopicMotionApplyThrust])
}

// TestBuildCoverageReport_FlagsUnused asserts every catalog topic the
// synthetic fixture does NOT touch lands in Unused. The full
// catalog covers ~40+ topics; the synthetic fixture exercises one;
// the unused set must contain the catalog's other publish_event
// topics.
func TestBuildCoverageReport_FlagsUnused(t *testing.T) {
	dir := t.TempDir()
	path := writeBaseline(t, dir, "synthetic_partial", catalog.Baseline{
		SchemaVersion: 1,
		Game:          "synthetic_partial",
		Checkpoints: []catalog.BaselineCheckpoint{
			{
				Tick: 0,
				EventsAtTick: []catalog.BaselineEventArgs{
					{Topic: catalog.EventTopicMotionApplyThrust, Args: map[string]any{}},
				},
			},
		},
	})

	report, err := catalog.BuildCoverageReport([]string{path})
	require.NoError(t, err)

	// motion/screen_wrap exists in the catalog but wasn't touched
	// by the synthetic baseline — it must be unused.
	assert.Contains(t, report.Unused, catalog.EventTopicMotionScreenWrap,
		"screen_wrap topic should be unused for the partial-fixture run")
	// Used + Unused are disjoint by construction.
	usedSet := make(map[string]struct{}, len(report.Used))
	for _, t := range report.Used {
		usedSet[t] = struct{}{}
	}
	for _, topic := range report.Unused {
		_, present := usedSet[topic]
		assert.False(t, present,
			"topic %q should not appear in both Used and Unused", topic)
	}
}

// TestBuildCoverageReport_FlagsUnknownTopic asserts a topic seen in
// a baseline but not registered in the catalog lands under
// UnknownTopics (drift signal).
func TestBuildCoverageReport_FlagsUnknownTopic(t *testing.T) {
	dir := t.TempDir()
	const ghost = "ghost/never_registered"
	path := writeBaseline(t, dir, "synthetic_ghost", catalog.Baseline{
		SchemaVersion: 1,
		Game:          "synthetic_ghost",
		Checkpoints: []catalog.BaselineCheckpoint{
			{Tick: 0, EventsAtTick: []catalog.BaselineEventArgs{{Topic: ghost}}},
		},
	})

	report, err := catalog.BuildCoverageReport([]string{path})
	require.NoError(t, err)
	assert.Contains(t, report.UnknownTopics, ghost,
		"unknown topic should appear in the drift list")
}

// TestBuildCoverageReport_HandlesMissingBaseline asserts a missing
// path is recorded under MissingFixtures rather than erroring. U24's
// "informational" gate must keep running even when one of the four
// games hasn't shipped its fixture yet.
func TestBuildCoverageReport_HandlesMissingBaseline(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "does_not_exist.baseline.json")

	report, err := catalog.BuildCoverageReport([]string{missing})
	require.NoError(t, err)
	assert.Contains(t, report.MissingFixtures, missing)
	assert.Empty(t, report.Used)
	assert.Empty(t, report.UnknownTopics)
}

// TestBuildCoverageReport_RealFixturesParse exercises the four
// integration_test fixtures so the test catches drift between the
// baseline schema + the coverage tool's parser. The assertion is
// permissive: as long as the read succeeds + at least one Phase 1-4
// topic surfaces, the parse is correct.
func TestBuildCoverageReport_RealFixturesParse(t *testing.T) {
	const base = "../../integration_test/fixtures"
	paths := []string{
		filepath.Join(base, "asteroids_proof.baseline.json"),
		filepath.Join(base, "mario_proof.baseline.json"),
		filepath.Join(base, "bomberman_proof.baseline.json"),
		filepath.Join(base, "donkey_kong_proof.baseline.json"),
	}
	// Skip cleanly if the fixtures aren't co-located (e.g.,
	// running the test from a copied tree). Phase 1-4 baselines
	// always exist in-tree; this is a defence-in-depth guard.
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("integration_test fixture not reachable from catalog package (%v)", err)
		}
	}

	report, err := catalog.BuildCoverageReport(paths)
	require.NoError(t, err)
	require.Empty(t, report.MissingFixtures)

	// At least asteroids' screen_wrap topic must be observed.
	assert.Contains(t, report.Used, catalog.EventTopicMotionScreenWrap,
		"asteroids baseline references motion/screen_wrap; it must appear under Used")
}

// TestWriteCoverageReport_StableOutput asserts the report writer
// produces byte-identical output across two runs against the same
// inputs. The CLI's diff-friendliness depends on this.
func TestWriteCoverageReport_StableOutput(t *testing.T) {
	dir := t.TempDir()
	path := writeBaseline(t, dir, "synthetic_stable", catalog.Baseline{
		SchemaVersion: 1,
		Game:          "synthetic_stable",
		Checkpoints: []catalog.BaselineCheckpoint{
			{
				Tick: 0,
				EventsAtTick: []catalog.BaselineEventArgs{
					{Topic: catalog.EventTopicMotionApplyThrust},
					{Topic: catalog.EventTopicMotionScreenWrap},
				},
			},
		},
	})

	report, err := catalog.BuildCoverageReport([]string{path})
	require.NoError(t, err)

	var first, second bytes.Buffer
	require.NoError(t, catalog.WriteCoverageReport(&first, report))
	require.NoError(t, catalog.WriteCoverageReport(&second, report))
	assert.Equal(t, first.String(), second.String(),
		"WriteCoverageReport must produce deterministic output")
}
