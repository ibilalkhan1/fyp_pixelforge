package pixelforge_input_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_input"
)

// All 9 canonical intents must be registered with the pievent target
// registry by the time package init completes. Subscribers wire up
// against these names via pievent.LookupTarget.
func TestIntents_AllRegisteredOnInit(t *testing.T) {
	want := []string{
		pixelforge_input.IntentJump,
		pixelforge_input.IntentAttack,
		pixelforge_input.IntentUp,
		pixelforge_input.IntentDown,
		pixelforge_input.IntentLeft,
		pixelforge_input.IntentRight,
		pixelforge_input.IntentUse,
		pixelforge_input.IntentMenu,
		pixelforge_input.IntentStart,
	}

	for _, name := range want {
		got := pievent.LookupTarget(name)
		require.NotNil(t, got, "intent %q must be registered after package import", name)

		// Typed access via package helper must return the same target.
		typed := pixelforge_input.Target(name)
		require.NotNil(t, typed, "Target(%q) must return non-nil", name)
		assert.Equal(t, got, typed.(pievent.Inspectable), "registry and typed accessor must return the same target")
	}

	assert.Equal(t, want, pixelforge_input.AllIntents(),
		"AllIntents() must enumerate the canonical 9 in declaration order")
}

// AllIntents must return a fresh slice on every call so callers
// mutating the result cannot corrupt subsequent lookups.
func TestIntents_AllIntents_ReturnsFreshSlice(t *testing.T) {
	a := pixelforge_input.AllIntents()
	a[0] = "MUTATED"

	b := pixelforge_input.AllIntents()
	assert.NotEqual(t, "MUTATED", b[0],
		"AllIntents must not share backing storage between calls")
}

// Target returns nil for intent names that were never registered.
func TestIntents_Target_UnknownIntentReturnsNil(t *testing.T) {
	assert.Nil(t, pixelforge_input.Target("input/nonexistent"))
}
