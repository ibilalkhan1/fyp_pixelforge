package catalog

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// BaselineCheckpoint is the JSON shape the integration-test fixtures
// use to record one tick's expected frame hash + bus events. Mirrors
// the writer at pixelforge_studio/integration_test/*_proof_test.go;
// duplicated here so the coverage tool doesn't import the long-tag
// test package (which would drag the entire integration harness into
// a tiny standalone binary).
type BaselineCheckpoint struct {
	Tick         int                 `json:"tick"`
	FrameSHA256  string              `json:"frame_sha256"`
	EventsAtTick []BaselineEventArgs `json:"events_at_tick"`
}

// BaselineEventArgs is one per-tick bus event entry inside a
// BaselineCheckpoint.
type BaselineEventArgs struct {
	Topic string         `json:"topic"`
	Args  map[string]any `json:"args"`
}

// Baseline is the top-level JSON document at
// pixelforge_studio/integration_test/fixtures/*_proof.baseline.json.
type Baseline struct {
	SchemaVersion int                  `json:"schema_version"`
	Game          string               `json:"game"`
	Checkpoints   []BaselineCheckpoint `json:"checkpoints"`
}

// CoverageReport summarises which catalog topics are observed in
// baseline.json fixtures vs. which are not. UsedBy maps topic →
// sorted-unique game names that touched the topic; Unused lists
// topics the catalog defines but no proof fixture references;
// UnknownTopics lists topics observed in fixtures that no recipe
// publishes (signals a drift between fixture + catalog).
type CoverageReport struct {
	// Catalog enumerates every topic the registry publishes.
	Catalog []string
	// Used is the sorted set of catalog topics observed in at
	// least one baseline.
	Used []string
	// Unused is the sorted set of catalog topics not observed in
	// any baseline. Informational per plan R19.
	Unused []string
	// UnknownTopics is the sorted set of topics seen in baselines
	// that don't map back to any catalog recipe — drift signal.
	UnknownTopics []string
	// UsedBy maps each Used topic to the set of game names whose
	// baseline referenced it, sorted.
	UsedBy map[string][]string
	// MissingFixtures lists baseline files the caller requested
	// but couldn't read (warning, not fatal).
	MissingFixtures []string
}

// BuildCoverageReport reads each baseline path in turn, extracts the
// set of bus topics observed per game, and cross-references against
// the catalog's registered publish_event topics.
//
// Missing baseline files are recorded under MissingFixtures and skipped
// — Phase 5 may run before every reference game ships; we want the
// tool informative rather than brittle.
func BuildCoverageReport(baselinePaths []string) (*CoverageReport, error) {
	catalogTopics := AllVerbTopics()
	catalogSet := make(map[string]struct{}, len(catalogTopics))
	for _, t := range catalogTopics {
		catalogSet[t] = struct{}{}
	}

	observed := make(map[string]map[string]struct{}) // topic → set(game)
	report := &CoverageReport{
		Catalog: catalogTopics,
		UsedBy:  make(map[string][]string),
	}

	for _, path := range baselinePaths {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				report.MissingFixtures = append(report.MissingFixtures, path)
				continue
			}
			return nil, fmt.Errorf("read baseline %q: %w", path, err)
		}
		var b Baseline
		if err := json.Unmarshal(data, &b); err != nil {
			return nil, fmt.Errorf("parse baseline %q: %w", path, err)
		}
		game := b.Game
		if game == "" {
			// Fall back to the file's basename when the JSON omits
			// `game` — keeps the report readable for synthetic
			// fixtures in tests.
			game = baselineGameFromPath(path)
		}
		for _, cp := range b.Checkpoints {
			for _, ev := range cp.EventsAtTick {
				if ev.Topic == "" {
					continue
				}
				set, ok := observed[ev.Topic]
				if !ok {
					set = make(map[string]struct{})
					observed[ev.Topic] = set
				}
				set[game] = struct{}{}
			}
		}
	}

	// Partition observed topics into used (in catalog) vs unknown
	// (not in catalog), and the inverse for unused.
	for topic, games := range observed {
		gameList := make([]string, 0, len(games))
		for g := range games {
			gameList = append(gameList, g)
		}
		sort.Strings(gameList)
		if _, inCatalog := catalogSet[topic]; inCatalog {
			report.Used = append(report.Used, topic)
			report.UsedBy[topic] = gameList
		} else {
			report.UnknownTopics = append(report.UnknownTopics, topic)
		}
	}
	for _, topic := range catalogTopics {
		if _, used := observed[topic]; !used {
			report.Unused = append(report.Unused, topic)
		}
	}

	sort.Strings(report.Used)
	sort.Strings(report.Unused)
	sort.Strings(report.UnknownTopics)
	return report, nil
}

// baselineGameFromPath strips the directory + ".baseline.json" suffix
// from a path so a fixture whose JSON omits the game field still
// renders under a recognisable name.
func baselineGameFromPath(path string) string {
	// Find last "/" — keep behaviour OS-independent without
	// importing filepath; the integration_test fixtures use forward
	// slashes in source paths anyway.
	if i := strings.LastIndexAny(path, "/\\"); i >= 0 {
		path = path[i+1:]
	}
	if strings.HasSuffix(path, ".baseline.json") {
		path = strings.TrimSuffix(path, ".baseline.json")
	}
	return path
}

// WriteCoverageReport emits a human-readable view of r to w. Stable
// ordering (sorted slices populated by BuildCoverageReport) keeps the
// output diff-friendly across runs.
func WriteCoverageReport(w io.Writer, r *CoverageReport) error {
	bw := &writeErr{w: w}
	bw.WriteString("Pixelforge verb coverage\n")
	bw.WriteString("========================\n\n")

	if len(r.MissingFixtures) > 0 {
		bw.WriteString("Warning: the following baseline fixtures were missing and skipped:\n")
		for _, p := range r.MissingFixtures {
			bw.WriteString(fmt.Sprintf("  - %s\n", p))
		}
		bw.WriteString("\n")
	}

	bw.WriteString("Verbs USED by at least one proof:\n")
	if len(r.Used) == 0 {
		bw.WriteString("  (none observed)\n")
	} else {
		for _, topic := range r.Used {
			games := r.UsedBy[topic]
			bw.WriteString(fmt.Sprintf("  - %-32s (%s)\n", topic, strings.Join(games, ", ")))
		}
	}
	bw.WriteString("\n")

	bw.WriteString("Verbs NOT used by any proof (review for relevance):\n")
	if len(r.Unused) == 0 {
		bw.WriteString("  (all catalog topics observed)\n")
	} else {
		for _, topic := range r.Unused {
			bw.WriteString(fmt.Sprintf("  - %s\n", topic))
		}
	}
	bw.WriteString("\n")

	if len(r.UnknownTopics) > 0 {
		bw.WriteString("Topics in fixtures but NOT in catalog (drift signal):\n")
		for _, topic := range r.UnknownTopics {
			bw.WriteString(fmt.Sprintf("  - %s\n", topic))
		}
		bw.WriteString("\n")
	}

	bw.WriteString(fmt.Sprintf("Catalog size: %d topics total. Used: %d. Unused: %d.\n",
		len(r.Catalog), len(r.Used), len(r.Unused)))
	if len(r.UnknownTopics) > 0 {
		bw.WriteString(fmt.Sprintf("Unknown (drift): %d.\n", len(r.UnknownTopics)))
	}
	return bw.err
}
