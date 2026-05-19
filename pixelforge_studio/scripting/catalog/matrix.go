package catalog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// MatrixGameSpec maps a logical reference-game name (the column header
// the markdown matrix renders) to the test package + test name the
// long-tag suite executes.
//
// The four entries below are hard-coded for plan-009 U25; if the test
// names ever change the matrix tool will skip the renamed test and the
// corresponding cell will show "n/a" rather than silently passing.
//
// TODO(plan-010+): when the reference-game set grows beyond four,
// promote this from a hard-coded slice to a JSON manifest under
// pixelforge_studio/scripting/catalog/matrix_games.json so adding a new
// game does not require recompiling the matrix tool.
type MatrixGameSpec struct {
	// DisplayName is the human-readable label rendered in the
	// markdown's "Game" column.
	DisplayName string
	// PackageSuffix is matched against `go test -json`'s Package
	// field with HasSuffix. Using a suffix tolerates the module
	// prefix (github.com/ibilalkhan1/fyp_pixelforge/...) without the
	// matrix tool needing to know it.
	PackageSuffix string
	// TestName is matched against the Test field exactly.
	TestName string
	// BaselineFixture is the path (relative to the repo root) of the
	// game's baseline.json. The per-recipe coverage section reads
	// these to compute which verbs each game exercises.
	BaselineFixture string
}

// DefaultMatrixGames is the canonical four-game proof set wired into
// plan-009 U25. Order matters: matrix columns render in this order so
// reviewers see games in the same sequence as the plan's Phased
// Delivery Summary (Asteroids → Mario → Bomberman → Donkey Kong).
func DefaultMatrixGames() []MatrixGameSpec {
	const fixtureBase = "pixelforge_studio/integration_test/fixtures"
	return []MatrixGameSpec{
		{
			DisplayName:     "Asteroids",
			PackageSuffix:   "pixelforge_studio/integration_test/asteroids_proof",
			TestName:        "TestAsteroidsProof",
			BaselineFixture: fixtureBase + "/asteroids_proof.baseline.json",
		},
		{
			DisplayName:     "Mario",
			PackageSuffix:   "pixelforge_studio/integration_test/mario_proof",
			TestName:        "TestMarioProof",
			BaselineFixture: fixtureBase + "/mario_proof.baseline.json",
		},
		{
			DisplayName:     "Bomberman",
			PackageSuffix:   "pixelforge_studio/integration_test/bomberman_proof",
			TestName:        "TestBombermanProof",
			BaselineFixture: fixtureBase + "/bomberman_proof.baseline.json",
		},
		{
			DisplayName:     "Donkey Kong",
			PackageSuffix:   "pixelforge_studio/integration_test/donkey_kong_proof",
			TestName:        "TestDonkeyKongProof",
			BaselineFixture: fixtureBase + "/donkey_kong_proof.baseline.json",
		},
	}
}

// MatrixResult is the per-test pass/fail/skip/missing outcome the
// matrix renders into one cell. Missing covers the case where a CI leg
// did not run the test at all (the JSON has no Action=pass|fail event
// for that Package/Test pair).
type MatrixResult string

const (
	MatrixResultPass    MatrixResult = "pass"
	MatrixResultFail    MatrixResult = "fail"
	MatrixResultSkip    MatrixResult = "skip"
	MatrixResultMissing MatrixResult = "missing"
)

// MatrixLeg is one CI matrix leg's contribution: a label (such as
// "linux-amd64" or "macos-arm64") plus the per-test outcome map keyed
// by (PackageSuffix, TestName). The matrix tool reads one of these per
// CI leg + composes them into one markdown table.
type MatrixLeg struct {
	Label   string
	Results map[matrixTestKey]MatrixResult
}

// matrixTestKey identifies one (package, test) pair inside a leg's
// result map. Package matching is exact here — the suffix match runs
// at JSON-parse time, see ParseTestResults.
type matrixTestKey struct {
	Package string
	Test    string
}

// goTestEvent mirrors the subset of `go test -json` event fields the
// matrix tool consumes. Other fields (Time, Output, Elapsed) are
// ignored — matrix cells only care about pass/fail/skip outcomes.
//
// Reference: `go doc cmd/test2json` (Action ∈ {start, run, pause,
// cont, pass, bench, fail, output, skip}).
type goTestEvent struct {
	Action  string `json:"Action"`
	Package string `json:"Package"`
	Test    string `json:"Test"`
}

// ParseTestResults reads a `go test -json` output file, extracts the
// per-test outcome (the final pass/fail/skip event for each
// Package/Test pair), and returns a MatrixLeg.
//
// label is the column header (e.g. "linux-amd64") the matrix renders.
//
// Malformed JSON lines are logged to stderr + skipped rather than
// aborting — CI may interleave test output with the JSON stream
// when a test panics, and the matrix should still surface whatever
// pass/fail signal it can extract.
func ParseTestResults(path, label string) (MatrixLeg, error) {
	f, err := os.Open(path) //nolint:gosec // path is operator-supplied at CI time
	if err != nil {
		return MatrixLeg{}, fmt.Errorf("open test-results %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	leg := MatrixLeg{
		Label:   label,
		Results: make(map[matrixTestKey]MatrixResult),
	}
	if err := parseTestResultsStream(f, &leg); err != nil {
		return MatrixLeg{}, fmt.Errorf("parse %q: %w", path, err)
	}
	return leg, nil
}

// parseTestResultsStream is the io.Reader-backed core of
// ParseTestResults. Split out so tests can feed synthetic JSON without
// touching disk.
func parseTestResultsStream(r io.Reader, leg *MatrixLeg) error {
	scanner := bufio.NewScanner(r)
	// Some test-output lines (panic dumps) can be long; bump the
	// scanner buffer ceiling well above the default 64 KiB.
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 || line[0] != '{' {
			// Non-JSON lines (rare; usually CI-runner banners that
			// slipped past tee) — skip silently.
			continue
		}
		var ev goTestEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			// Malformed JSON — log + continue. The matrix tool's
			// resilience requirement is in U25's test scenarios.
			fmt.Fprintf(os.Stderr, "matrix: skipping malformed JSON line: %v\n", err)
			continue
		}
		if ev.Test == "" {
			// Package-level events (e.g. compile-fail) carry no
			// Test field. They affect the overall package result
			// but the matrix is per-test, so skip.
			continue
		}
		key := matrixTestKey{Package: ev.Package, Test: ev.Test}
		switch ev.Action {
		case "pass":
			leg.Results[key] = MatrixResultPass
		case "fail":
			leg.Results[key] = MatrixResultFail
		case "skip":
			// Only record skip if no later pass/fail overrides it.
			if _, present := leg.Results[key]; !present {
				leg.Results[key] = MatrixResultSkip
			}
		default:
			// run/pause/cont/output/start/bench — ignore.
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	return nil
}

// lookupResult finds the outcome for a (PackageSuffix, TestName) pair
// inside a leg, using HasSuffix on the package field so the matrix
// tool tolerates the module prefix. Returns MatrixResultMissing when
// no matching event is present.
func (l MatrixLeg) lookupResult(pkgSuffix, testName string) MatrixResult {
	for key, res := range l.Results {
		if key.Test != testName {
			continue
		}
		if strings.HasSuffix(key.Package, pkgSuffix) {
			return res
		}
	}
	return MatrixResultMissing
}

// renderCell formats one MatrixResult as a markdown table cell.
// Unicode markers (no emoji per repo convention; the user-facing plan
// document used ✅/❌ for readability but the file output keeps the
// catalog-package text-only so docs/verb-catalog.md style + this
// matrix file render consistently in plain terminals).
func renderCell(r MatrixResult) string {
	switch r {
	case MatrixResultPass:
		return "PASS"
	case MatrixResultFail:
		return "**FAIL**"
	case MatrixResultSkip:
		return "skip"
	case MatrixResultMissing:
		return "n/a"
	default:
		return string(r)
	}
}

// WriteCapabilityMatrix renders the markdown capability matrix to w.
// Deterministic: legs render in the order passed in; games in the
// order returned by DefaultMatrixGames; per-recipe rows sorted
// alphabetically by topic.
//
// games is the column set (defaults to DefaultMatrixGames when nil).
// Per-recipe coverage rows are emitted only when the games' baseline
// fixtures are readable; missing fixtures cause the section to omit
// that game's column data (cell = "n/a"), not the whole section.
func WriteCapabilityMatrix(w io.Writer, legs []MatrixLeg, games []MatrixGameSpec) error {
	if games == nil {
		games = DefaultMatrixGames()
	}
	bw := &writeErr{w: w}
	writeMatrixHeader(bw)
	writeMatrixPerGame(bw, legs, games)
	writeMatrixPerRecipe(bw, games)
	return bw.err
}

// writeMatrixHeader emits the headline + do-not-edit notice.
func writeMatrixHeader(bw *writeErr) {
	bw.WriteString("<!-- Generated by `go run ./pixelforge_studio/scripting/catalog/cmd/matrix`. Do not edit by hand. -->\n\n")
	bw.WriteString("# Reference-Games Capability Matrix\n\n")
	bw.WriteString("Regenerated by CI from `go test -tags=long -json` output.\n")
	bw.WriteString("Do not edit by hand.\n\n")
}

// writeMatrixPerGame emits the per-game pass/fail table — one row per
// game, one column per CI matrix leg.
func writeMatrixPerGame(bw *writeErr, legs []MatrixLeg, games []MatrixGameSpec) {
	bw.WriteString("## Per-game pass/fail\n\n")
	if len(legs) == 0 {
		bw.WriteString("_No test-results legs supplied; per-game outcomes unavailable._\n\n")
		return
	}
	// Header row.
	bw.WriteString("| Game |")
	for _, leg := range legs {
		bw.WriteString(" ")
		bw.WriteString(leg.Label)
		bw.WriteString(" |")
	}
	bw.WriteString("\n|---|")
	for range legs {
		bw.WriteString("---|")
	}
	bw.WriteString("\n")

	for _, g := range games {
		bw.WriteString("| ")
		bw.WriteString(g.DisplayName)
		bw.WriteString(" |")
		for _, leg := range legs {
			res := leg.lookupResult(g.PackageSuffix, g.TestName)
			bw.WriteString(" ")
			bw.WriteString(renderCell(res))
			bw.WriteString(" |")
		}
		bw.WriteString("\n")
	}
	bw.WriteString("\n")
}

// writeMatrixPerRecipe emits the recipe×game coverage table. Reads
// each game's baseline.json + cross-references against the catalog's
// publish_event topics. Missing fixtures render as "n/a" in that
// game's column instead of erroring — the matrix is informational and
// must survive a partial reference-game set during incremental rollout.
func writeMatrixPerRecipe(bw *writeErr, games []MatrixGameSpec) {
	bw.WriteString("## Per-recipe coverage\n\n")
	bw.WriteString("Each row is a verb-catalog event topic; each column is one reference game.\n")
	bw.WriteString("`yes` means the game's `*.baseline.json` records the topic in at least one checkpoint.\n\n")

	// Build a set of catalog topics for lookup.
	catalogTopics := AllVerbTopics()
	if len(catalogTopics) == 0 {
		bw.WriteString("_Catalog has no publish_event topics; per-recipe coverage unavailable._\n\n")
		return
	}

	// Read each game's baseline + extract its topic set.
	type gameTopics struct {
		spec    MatrixGameSpec
		topics  map[string]struct{}
		missing bool
	}
	perGame := make([]gameTopics, 0, len(games))
	for _, g := range games {
		topics, ok := readBaselineTopics(g.BaselineFixture)
		perGame = append(perGame, gameTopics{spec: g, topics: topics, missing: !ok})
	}

	// Header row.
	bw.WriteString("| Topic |")
	for _, g := range games {
		bw.WriteString(" ")
		bw.WriteString(g.DisplayName)
		bw.WriteString(" |")
	}
	bw.WriteString("\n|---|")
	for range games {
		bw.WriteString("---|")
	}
	bw.WriteString("\n")

	// One row per catalog topic, sorted (AllVerbTopics already sorts).
	for _, topic := range catalogTopics {
		bw.WriteString("| `")
		bw.WriteString(topic)
		bw.WriteString("` |")
		for _, gt := range perGame {
			cell := " no |"
			switch {
			case gt.missing:
				cell = " n/a |"
			case containsTopic(gt.topics, topic):
				cell = " yes |"
			}
			bw.WriteString(cell)
		}
		bw.WriteString("\n")
	}
	bw.WriteString("\n")

	// Footer: surface any baselines that couldn't be read so the
	// reviewer knows the n/a cells weren't a tooling bug.
	for _, gt := range perGame {
		if gt.missing {
			bw.WriteString(fmt.Sprintf("_Note: baseline fixture %s was not readable; %s column shows `n/a`._\n",
				gt.spec.BaselineFixture, gt.spec.DisplayName))
		}
	}
}

// containsTopic returns true when topic is a key in set. Tiny helper —
// avoids inlining the `_, ok := set[topic]; ok` idiom four times in
// writeMatrixPerRecipe.
func containsTopic(set map[string]struct{}, topic string) bool {
	if set == nil {
		return false
	}
	_, ok := set[topic]
	return ok
}

// readBaselineTopics parses a single baseline.json fixture + returns
// the set of bus topics it references across all checkpoints. The
// second return is false when the file is missing or unparseable — the
// caller renders the column as "n/a".
func readBaselineTopics(path string) (map[string]struct{}, bool) {
	data, err := os.ReadFile(path) //nolint:gosec // path is operator-supplied at CI time
	if err != nil {
		return nil, false
	}
	var b Baseline
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, false
	}
	out := make(map[string]struct{})
	for _, cp := range b.Checkpoints {
		for _, ev := range cp.EventsAtTick {
			if ev.Topic != "" {
				out[ev.Topic] = struct{}{}
			}
		}
	}
	return out, true
}

// SortedLabelLegs returns legs sorted by Label. Convenience for the
// matrix tool when it loads legs from a comma-separated flag and the
// operator passes them out of order; the matrix output is then
// reviewer-friendly (linux-amd64 before macos-arm64 alphabetically).
//
// Kept separate from WriteCapabilityMatrix so tests can assert on the
// caller-provided order when explicit ordering matters.
func SortedLabelLegs(legs []MatrixLeg) []MatrixLeg {
	out := append([]MatrixLeg(nil), legs...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Label < out[j].Label
	})
	return out
}
