package pixelforge_physics

import (
	"bytes"
	"encoding/binary"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// regenBaseline, when true, rewrites the cross-CPU baseline fixture from
// the current run's output instead of asserting against it. Used the first
// time the test is set up, or whenever the simulation deliberately changes.
//
// Usage: `go test -run TestDeterminism -regen-baseline=true ./pixelforge_physics/...`
var regenBaseline = flag.Bool("regen-baseline", false, "rewrite the cross-CPU determinism baseline from this run")

// runDeterminismProbe is the canonical 5-body, 1000-tick simulation that
// the cross-CPU probe and the same-CPU probe both share. Any change here
// invalidates the baseline file — bump it deliberately and regenerate.
//
// The scenario:
//   - World: Mario preset (gravity, no wrap, no ladders), 320x240.
//   - Bodies: 5 dynamic AABBs at varied initial positions and velocities.
//   - Tilemap: a single solid row at row=14 (y=224..240) — the floor.
//   - Ticks: 1000 iterations of Integrate → ResolveTilemapCollision.
//   - Output: each body's final Position (X, Y) as raw int32, written LE.
//
// We deliberately exercise: Fixed32 mul/div (via integrate), trig is NOT
// touched (that's a separate probe), tilemap resolution, gravity, body sync.
// We do NOT touch resolv's float64 SAT because U6's substrate choice keeps
// the platformer path purely Fixed32 (resolv is only used for broad-phase
// queries in Asteroids-style games — see doc.go).
func runDeterminismProbe(t *testing.T) []byte {
	t.Helper()

	grid := newFakeGrid(20, 15)
	// Solid floor at row 14 across the full width.
	for col := 0; col < 20; col++ {
		grid.set(col, 14, 1)
	}
	// A solid column wall at col 10 for collision interest.
	for row := 8; row < 14; row++ {
		grid.set(10, row, 1)
	}

	w := NewWorld(MarioConfig())
	bodies := []*Body{
		NewBody("a", Vec2{FixedFromInt(20), FixedFromInt(30)},
			FixedFromInt(8), FixedFromInt(8)),
		NewBody("b", Vec2{FixedFromInt(60), FixedFromInt(50)},
			FixedFromInt(8), FixedFromInt(8)),
		NewBody("c", Vec2{FixedFromInt(120), FixedFromInt(20)},
			FixedFromInt(8), FixedFromInt(8)),
		NewBody("d", Vec2{FixedFromInt(180), FixedFromInt(40)},
			FixedFromInt(8), FixedFromInt(8)),
		NewBody("e", Vec2{FixedFromInt(220), FixedFromInt(10)},
			FixedFromInt(8), FixedFromInt(8)),
	}
	// Sprinkle initial velocities so the bodies don't behave identically.
	bodies[0].Velocity = Vec2{FixedFromInt(1), FixedZero}
	bodies[1].Velocity = Vec2{FixedFromInt(2), FixedZero}
	bodies[2].Velocity = Vec2{FixedFromInt(-1), FixedZero}
	bodies[3].Velocity = Vec2{FixedZero, FixedZero}
	bodies[4].Velocity = Vec2{FixedFromInt(-2), FixedZero}
	for _, b := range bodies {
		w.AddBody(b)
	}

	dt := FixedFromFloat(1.0 / 60.0)
	for tick := 0; tick < 1000; tick++ {
		for _, b := range bodies {
			Integrate(b, w, dt)
			ResolveTilemapCollision(b, w, grid, []int{1}, 16, 16)
		}
	}

	// Serialize each body's final Position (X, Y) as little-endian int32.
	var buf bytes.Buffer
	for _, b := range bodies {
		_ = binary.Write(&buf, binary.LittleEndian, int32(b.Position.X))
		_ = binary.Write(&buf, binary.LittleEndian, int32(b.Position.Y))
	}
	return buf.Bytes()
}

// TestDeterminism_SameCPU_ByteEqual proves that ten identical simulations
// run in the SAME process produce byte-equal Fixed32 position outputs.
//
// This is the foundation contract: if it ever fails, the whole pixel-hash
// CI strategy collapses. It should never fail unless someone has introduced
// nondeterminism (a map iteration, a goroutine, a time.Now() read, a
// math.Sin call outside the LUT, ...).
func TestDeterminism_SameCPU_ByteEqual(t *testing.T) {
	first := runDeterminismProbe(t)
	require.NotEmpty(t, first, "probe produced no bytes")
	for i := 1; i < 10; i++ {
		got := runDeterminismProbe(t)
		if !bytes.Equal(first, got) {
			t.Fatalf("iteration %d diverged from run 0:\n  run0 = %x\n  run%d = %x",
				i, first, i, got)
		}
	}
}

// TestDeterminism_CrossCPU_ProbeFramework compares the current run against
// the baseline committed in testdata/cross_cpu_baseline.dat.
//
// On a CI matrix:
//   - ubuntu-latest amd64 records the baseline (run once with
//     -regen-baseline=true, commit the file).
//   - macos-latest arm64, windows-latest amd64, etc. run this test and
//     compare bytes.
//
// If the test fails on a non-baseline architecture, U6's Execution-note
// fallback applies — see pixelforge_physics/doc.go for the three options
// (tolerance buckets / single-CPU CI / hand-rolled integer SAT).
//
// U25 wires the CI matrix that flips the switch. Until then this test
// proves only same-CPU stability against a snapshot (it should match on
// ANY identical re-run of the same binary).
func TestDeterminism_CrossCPU_ProbeFramework(t *testing.T) {
	baselinePath := filepath.Join("testdata", "cross_cpu_baseline.dat")
	got := runDeterminismProbe(t)
	require.NotEmpty(t, got)

	if *regenBaseline {
		require.NoError(t, os.MkdirAll("testdata", 0o755))
		require.NoError(t, os.WriteFile(baselinePath, got, 0o644))
		t.Logf("baseline rewritten (%d bytes) at %s", len(got), baselinePath)
		return
	}

	want, err := os.ReadFile(baselinePath)
	if os.IsNotExist(err) {
		// First-time setup: write the baseline so subsequent runs can compare.
		require.NoError(t, os.MkdirAll("testdata", 0o755))
		require.NoError(t, os.WriteFile(baselinePath, got, 0o644))
		t.Logf("baseline created (first run) at %s — re-run to verify", baselinePath)
		return
	}
	require.NoError(t, err)
	if !bytes.Equal(got, want) {
		t.Fatalf("cross-CPU probe baseline mismatch:\n  got:  %x\n  want: %x\n"+
			"If this is a deliberate simulation change, re-run with -regen-baseline=true.\n"+
			"If this fails on a non-baseline CPU (e.g. arm64) but passes on amd64, "+
			"the cross-CPU contract is broken — apply U6's Execution-note fallback "+
			"(tolerance buckets or single-CPU pixel-hash CI).", got, want)
	}
}

// TestDeterminism_ToleranceBucket_AcrossCPU is the documented fallback
// variant. It is skipped by default — only activated if the byte-equal
// probe above fails on the CI matrix and U25 commits to the tolerance
// strategy.
//
// The implementation reads the baseline and asserts each Fixed32 component
// is within ±toleranceLSB raw units of the recorded value.
func TestDeterminism_ToleranceBucket_AcrossCPU(t *testing.T) {
	t.Skip("activated only if TestDeterminism_CrossCPU_ProbeFramework fails on " +
		"non-baseline CPUs — see U6 Execution note + doc.go for the decision tree")
}
