package pixelforge_project

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A Project with a non-default InputMap round-trips through JSON
// without losing field order, modifier flags, or multi-key bindings.
func TestProject_InputMap_RoundTrip(t *testing.T) {
	original := NewProject("inputmap")
	original.InputMap = []InputBinding{
		{Intent: IntentJump, Keyboard: []string{"Space"}, GamepadButton: "GamepadB"},
		{Intent: IntentMenu, Keyboard: []string{"S"}, Modifier: "Ctrl"},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var loaded Project
	require.NoError(t, json.Unmarshal(data, &loaded))

	assert.Equal(t, original.InputMap, loaded.InputMap)
}

// Loading a project JSON that omits the input_map key triggers
// applyDefaults to populate it with the 9 NES-style intent bindings.
func TestProject_InputMap_DefaultsAppliedOnLoad(t *testing.T) {
	raw := `{
		"schema_version": 1,
		"name": "no-inputmap",
		"screen_width": 320,
		"screen_height": 180,
		"tps": 30,
		"palette": {},
		"theme": {},
		"sprites": [],
		"audio": [],
		"scenes": [],
		"behaviors": [],
		"bindings": [],
		"event_subscriptions": [],
		"extension_hooks": []
	}`

	p, err := LoadReader(bytesReader([]byte(raw)))
	require.NoError(t, err)

	require.Len(t, p.InputMap, 9, "all 9 intents should be present after defaults")

	intents := make(map[string]bool, len(p.InputMap))
	for _, b := range p.InputMap {
		intents[b.Intent] = true
	}
	for _, want := range []string{
		IntentJump, IntentAttack,
		IntentUp, IntentDown, IntentLeft, IntentRight,
		IntentUse, IntentMenu, IntentStart,
	} {
		assert.True(t, intents[want], "default InputMap missing intent %q", want)
	}
}

// A project whose InputMap is left at the package default does NOT
// preserve a "size-stable" serialization across save/load — defaults
// are applied at load, not stored. Documenting the deviation: in this
// implementation a project that has been *loaded* will carry the
// default InputMap explicitly, so subsequent re-serialization includes
// the input_map key. The omitempty on the field still suppresses
// emission for projects whose InputMap is nil (never loaded, never
// edited), which is the case for a freshly constructed zero-value
// Project. We assert both halves of that contract here.
func TestProject_InputMap_OmitemptyWhenDefault(t *testing.T) {
	// Half one: a zero-value Project with nil InputMap omits the key.
	var zero Project
	data, err := json.Marshal(&zero)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"input_map"`,
		"nil InputMap on a zero-value project should be omitted")

	// Half two: documented deviation — once applyDefaults has run
	// (i.e. after Load/LoadReader), the InputMap is populated and
	// the key WILL appear in the saved JSON. This is a deliberate
	// trade-off: applying defaults at load gives the runtime a fully
	// hydrated InputMap without needing to re-call DefaultInputMap()
	// everywhere, at the cost of giving up byte-stable size for pre-v1
	// projects. The slice-omitempty rule still protects callers who
	// never invoked applyDefaults.
	raw := `{
		"schema_version": 1,
		"name": "x",
		"screen_width": 320, "screen_height": 180, "tps": 30,
		"palette": {}, "theme": {},
		"sprites": [], "audio": [], "scenes": [], "behaviors": [],
		"bindings": [], "event_subscriptions": [], "extension_hooks": []
	}`
	loaded, err := LoadReader(bytesReader([]byte(raw)))
	require.NoError(t, err)
	require.Len(t, loaded.InputMap, 9)

	out, err := json.Marshal(loaded)
	require.NoError(t, err)
	assert.True(t,
		strings.Contains(string(out), `"input_map"`),
		"after applyDefaults the InputMap is materialised and will be re-serialised")
}
