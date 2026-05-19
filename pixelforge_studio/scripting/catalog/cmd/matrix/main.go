// Command matrix regenerates docs/reference-games-capability-matrix.md
// from one or more `go test -tags=long -json` output files. Designed
// for CI use: GitHub Actions runs the long-tag suite per platform leg,
// uploads each leg's JSON as a workflow artifact, and the
// capability-matrix job downloads them and invokes this binary.
//
// Per plan-009 U25 scope-guardian finding F-007, the regenerated
// matrix is uploaded as a workflow artifact rather than committed back
// to the repo — that avoids the commit-feedback-loop where every CI
// run would author a new commit.
//
// Output is deterministic — two invocations on the same inputs
// produce byte-identical markdown — so the file can also be
// regenerated locally (via `make capability-matrix` against a saved
// test-results.json) for sanity-checking before pushing.
//
// Usage:
//
//	go run ./pixelforge_studio/scripting/catalog/cmd/matrix \
//	  -in test-results-linux-amd64.json,test-results-macos-arm64.json \
//	  -out docs/reference-games-capability-matrix.md
//
// Each path may optionally include a leg-label prefix separated by '=':
//
//	-in linux-amd64=test-results-linux.json,macos-arm64=test-results-macos.json
//
// Without an explicit prefix the leg label is derived from the
// filename: strip the directory + ".json" suffix, drop a leading
// "test-results-" so "test-results-linux-amd64.json" yields
// "linux-amd64".
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
)

func main() {
	inFlag := flag.String("in", "",
		"comma-separated test-results JSON files (one per CI matrix leg); "+
			"each may be prefixed with 'label=' to override the column label")
	outFlag := flag.String("out", "docs/reference-games-capability-matrix.md",
		"output path; '-' writes to stdout")
	flag.Parse()

	if *inFlag == "" {
		log.Fatal("matrix: -in is required (comma-separated test-results JSON files)")
	}

	legs, err := loadLegs(*inFlag)
	if err != nil {
		log.Fatalf("matrix: %v", err)
	}

	if err := writeOutput(*outFlag, legs); err != nil {
		log.Fatalf("matrix: %v", err)
	}
	if *outFlag != "-" {
		fmt.Fprintf(os.Stderr, "matrix: wrote %s\n", *outFlag)
	}
}

// loadLegs parses the -in flag into a slice of MatrixLegs. Each token
// in the comma-separated list is either "path" or "label=path".
func loadLegs(spec string) ([]catalog.MatrixLeg, error) {
	tokens := strings.Split(spec, ",")
	legs := make([]catalog.MatrixLeg, 0, len(tokens))
	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		label, path := splitLegToken(tok)
		leg, err := catalog.ParseTestResults(path, label)
		if err != nil {
			return nil, fmt.Errorf("load leg %q: %w", tok, err)
		}
		legs = append(legs, leg)
	}
	if len(legs) == 0 {
		return nil, fmt.Errorf("matrix: -in produced no usable legs (got %q)", spec)
	}
	return legs, nil
}

// splitLegToken accepts "path" or "label=path". When no explicit
// label is supplied, derive one from the filename: strip dir +
// ".json" suffix, drop "test-results-" prefix if present.
func splitLegToken(tok string) (label, path string) {
	if i := strings.IndexByte(tok, '='); i >= 0 {
		return tok[:i], tok[i+1:]
	}
	base := filepath.Base(tok)
	base = strings.TrimSuffix(base, ".json")
	base = strings.TrimPrefix(base, "test-results-")
	return base, tok
}

// writeOutput renders the matrix to the configured destination. For
// file outputs we write to a sibling temp file then rename — avoids a
// half-written matrix.md when the renderer panics partway through.
// The same atomic-rename pattern is used by cmd/gendocs.
func writeOutput(path string, legs []catalog.MatrixLeg) error {
	if path == "-" {
		return catalog.WriteCapabilityMatrix(os.Stdout, legs, nil) //nolint:wrapcheck // direct error surface
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %q: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".capability-matrix-*.md.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		// Best-effort cleanup if rename below didn't run.
		_ = os.Remove(tmpName)
	}()

	if err := catalog.WriteCapabilityMatrix(tmp, legs, nil); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write matrix: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename %q → %q: %w", tmpName, path, err)
	}
	return nil
}
