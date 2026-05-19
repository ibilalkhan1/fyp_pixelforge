// Command coverage cross-references the verb-recipe catalog against the
// four proof-game baseline.json fixtures and prints used + unused
// verbs.
//
// Per plan-009 U24 / brainstorm R19 the report is INFORMATIONAL — it
// does not exit non-zero when verbs are unused. Unused verbs are
// flagged for review, not removed, because future games may exercise
// them; the catalog deliberately ships richer than any single proof
// game needs.
//
// Usage:
//
//	go run ./pixelforge_studio/scripting/catalog/cmd/coverage
//	go run ./pixelforge_studio/scripting/catalog/cmd/coverage -baselines path1,path2
//
// With no -baselines flag the tool defaults to the four proof
// fixtures under pixelforge_studio/integration_test/fixtures/.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
)

// defaultBaselinePaths returns the canonical four-game proof fixtures.
// Paths are relative to the repo root; the binary is normally invoked
// from there via `go run`. Callers running from a different working
// directory can pass -baselines explicitly.
func defaultBaselinePaths() []string {
	const base = "pixelforge_studio/integration_test/fixtures"
	return []string{
		base + "/asteroids_proof.baseline.json",
		base + "/mario_proof.baseline.json",
		base + "/bomberman_proof.baseline.json",
		base + "/donkey_kong_proof.baseline.json",
	}
}

func main() {
	baselinesFlag := flag.String("baselines", "",
		"comma-separated list of baseline.json paths (default: the four proof fixtures)")
	flag.Parse()

	var paths []string
	if *baselinesFlag != "" {
		for _, p := range strings.Split(*baselinesFlag, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				paths = append(paths, p)
			}
		}
	} else {
		paths = defaultBaselinePaths()
	}

	report, err := catalog.BuildCoverageReport(paths)
	if err != nil {
		log.Fatalf("coverage: %v", err)
	}
	if err := catalog.WriteCoverageReport(os.Stdout, report); err != nil {
		log.Fatalf("coverage: write report: %v", err)
	}

	// Always exit 0 — coverage is informational per U24. The CI
	// completeness gate is the catalog-vs-fixtures drift detection
	// (UnknownTopics) which is reported but does not fail.
	if len(report.UnknownTopics) > 0 {
		fmt.Fprintf(os.Stderr, "warning: %d topic(s) appear in baselines but not in the catalog\n",
			len(report.UnknownTopics))
	}
}
