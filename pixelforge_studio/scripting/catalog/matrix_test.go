package catalog_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
)

// syntheticJSON is one line per `go test -json` event. Built up as a
// string in each test so the inputs are local + obvious.
const passingLinuxJSON = `{"Action":"run","Package":"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/integration_test/asteroids_proof","Test":"TestAsteroidsProof"}
{"Action":"pass","Package":"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/integration_test/asteroids_proof","Test":"TestAsteroidsProof","Elapsed":1.23}
{"Action":"run","Package":"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/integration_test/mario_proof","Test":"TestMarioProof"}
{"Action":"pass","Package":"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/integration_test/mario_proof","Test":"TestMarioProof","Elapsed":2.34}
{"Action":"run","Package":"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/integration_test/bomberman_proof","Test":"TestBombermanProof"}
{"Action":"pass","Package":"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/integration_test/bomberman_proof","Test":"TestBombermanProof","Elapsed":3.45}
{"Action":"run","Package":"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/integration_test/donkey_kong_proof","Test":"TestDonkeyKongProof"}
{"Action":"pass","Package":"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/integration_test/donkey_kong_proof","Test":"TestDonkeyKongProof","Elapsed":4.56}
`

// writeTempJSON drops contents into a tempdir + returns the path.
func writeTempJSON(t *testing.T, name, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
	return path
}

// TestParseTestResults_AllPass exercises the happy path: four PASS
// events in, four MatrixResultPass entries out.
func TestParseTestResults_AllPass(t *testing.T) {
	path := writeTempJSON(t, "linux-amd64.json", passingLinuxJSON)
	leg, err := catalog.ParseTestResults(path, "linux-amd64")
	require.NoError(t, err)

	assert.Equal(t, "linux-amd64", leg.Label)
	assert.Len(t, leg.Results, 4)

	// Spot-check each game looks up to PASS via suffix matching.
	for _, g := range catalog.DefaultMatrixGames() {
		var found bool
		for k, v := range leg.Results {
			if k.Test == g.TestName && strings.HasSuffix(k.Package, g.PackageSuffix) {
				assert.Equal(t, catalog.MatrixResultPass, v,
					"game %q should be PASS", g.DisplayName)
				found = true
				break
			}
		}
		assert.True(t, found, "missing entry for %q (%s)", g.TestName, g.PackageSuffix)
	}
}

// TestParseTestResults_OneFail asserts a FAIL event flips the cell to
// MatrixResultFail without affecting the other three games.
func TestParseTestResults_OneFail(t *testing.T) {
	failingJSON := strings.ReplaceAll(passingLinuxJSON,
		`"Action":"pass","Package":"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/integration_test/mario_proof","Test":"TestMarioProof"`,
		`"Action":"fail","Package":"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/integration_test/mario_proof","Test":"TestMarioProof"`,
	)
	path := writeTempJSON(t, "fail.json", failingJSON)
	leg, err := catalog.ParseTestResults(path, "linux-amd64")
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, catalog.WriteCapabilityMatrix(&buf, []catalog.MatrixLeg{leg}, nil))
	out := buf.String()

	// Mario row must contain FAIL marker.
	marioLine := findLineWithPrefix(t, out, "| Mario |")
	assert.Contains(t, marioLine, "**FAIL**",
		"Mario row should render FAIL marker")
	// Asteroids row must contain PASS.
	asteroidsLine := findLineWithPrefix(t, out, "| Asteroids |")
	assert.Contains(t, asteroidsLine, "PASS")
	assert.NotContains(t, asteroidsLine, "FAIL")
}

// TestParseTestResults_MissingTest asserts a game with no event in the
// JSON renders as "n/a" rather than silently passing.
func TestParseTestResults_MissingTest(t *testing.T) {
	// JSON that only covers Asteroids — Mario/Bomberman/DK absent.
	partial := `{"Action":"pass","Package":"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/integration_test/asteroids_proof","Test":"TestAsteroidsProof"}
`
	path := writeTempJSON(t, "partial.json", partial)
	leg, err := catalog.ParseTestResults(path, "linux-amd64")
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, catalog.WriteCapabilityMatrix(&buf, []catalog.MatrixLeg{leg}, nil))
	out := buf.String()

	// Mario row should show n/a.
	marioLine := findLineWithPrefix(t, out, "| Mario |")
	assert.Contains(t, marioLine, "n/a",
		"Mario row should render n/a when JSON has no event for it")
}

// TestParseTestResults_MalformedLinesSkipped asserts the parser
// silently skips non-JSON noise + malformed records rather than
// erroring out.
func TestParseTestResults_MalformedLinesSkipped(t *testing.T) {
	noisy := "garbage line\n" + passingLinuxJSON + `{"Action": broken\n` + "\n"
	path := writeTempJSON(t, "noisy.json", noisy)
	leg, err := catalog.ParseTestResults(path, "linux-amd64")
	require.NoError(t, err)
	assert.Len(t, leg.Results, 4,
		"valid lines should be parsed even when surrounding lines are malformed")
}

// TestWriteCapabilityMatrix_TwoLegs renders two columns + asserts both
// headers appear.
func TestWriteCapabilityMatrix_TwoLegs(t *testing.T) {
	linuxPath := writeTempJSON(t, "linux.json", passingLinuxJSON)
	macosPath := writeTempJSON(t, "macos.json", passingLinuxJSON)

	linux, err := catalog.ParseTestResults(linuxPath, "linux-amd64")
	require.NoError(t, err)
	macos, err := catalog.ParseTestResults(macosPath, "macos-arm64")
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, catalog.WriteCapabilityMatrix(&buf,
		[]catalog.MatrixLeg{linux, macos}, nil))
	out := buf.String()

	assert.Contains(t, out, "linux-amd64")
	assert.Contains(t, out, "macos-arm64")
	assert.Contains(t, out, "## Per-game pass/fail")
	assert.Contains(t, out, "## Per-recipe coverage")
}

// TestWriteCapabilityMatrix_Deterministic asserts two consecutive
// renders of the same inputs produce byte-identical output. CI's
// "regenerate + diff" workflow depends on this.
func TestWriteCapabilityMatrix_Deterministic(t *testing.T) {
	path := writeTempJSON(t, "stable.json", passingLinuxJSON)
	leg, err := catalog.ParseTestResults(path, "linux-amd64")
	require.NoError(t, err)

	var first, second bytes.Buffer
	require.NoError(t, catalog.WriteCapabilityMatrix(&first,
		[]catalog.MatrixLeg{leg}, nil))
	require.NoError(t, catalog.WriteCapabilityMatrix(&second,
		[]catalog.MatrixLeg{leg}, nil))
	assert.Equal(t, first.String(), second.String(),
		"matrix output must be byte-stable across runs")
}

// TestWriteCapabilityMatrix_NoLegs renders cleanly when given an empty
// leg slice (the empty-CI case after a brand-new repo / first push).
func TestWriteCapabilityMatrix_NoLegs(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, catalog.WriteCapabilityMatrix(&buf, nil, nil))
	out := buf.String()
	assert.Contains(t, out, "# Reference-Games Capability Matrix")
	assert.Contains(t, out, "_No test-results legs supplied")
}

// TestWriteCapabilityMatrix_CustomGames asserts the games parameter
// lets callers override the four-game default — needed for plan-010+
// when the reference set expands.
func TestWriteCapabilityMatrix_CustomGames(t *testing.T) {
	custom := []catalog.MatrixGameSpec{
		{
			DisplayName:   "TinyGame",
			PackageSuffix: "example/tiny",
			TestName:      "TestTiny",
		},
	}
	leg := catalog.MatrixLeg{Label: "linux-amd64"}
	var buf bytes.Buffer
	require.NoError(t, catalog.WriteCapabilityMatrix(&buf,
		[]catalog.MatrixLeg{leg}, custom))
	out := buf.String()
	assert.Contains(t, out, "| TinyGame |")
	// The standard four must NOT appear when caller passes custom.
	assert.NotContains(t, out, "| Asteroids |")
}

// TestSortedLabelLegs asserts the helper sorts by Label ascending.
func TestSortedLabelLegs(t *testing.T) {
	in := []catalog.MatrixLeg{
		{Label: "macos-arm64"},
		{Label: "linux-amd64"},
	}
	out := catalog.SortedLabelLegs(in)
	require.Len(t, out, 2)
	assert.Equal(t, "linux-amd64", out[0].Label)
	assert.Equal(t, "macos-arm64", out[1].Label)
}

// findLineWithPrefix returns the first line in s starting with prefix,
// failing the test if none is found. Helper for matrix-row inspection.
func findLineWithPrefix(t *testing.T, s, prefix string) string {
	t.Helper()
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	t.Fatalf("no line found with prefix %q in:\n%s", prefix, s)
	return ""
}
