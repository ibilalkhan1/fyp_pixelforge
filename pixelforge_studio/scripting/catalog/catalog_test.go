package catalog_test

import (
	"testing"

	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_routine"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCtx is a minimal Context for builder tests.
type fakeCtx struct {
	samples map[string]any
	entities map[string]any
	values  map[string]catalog.ValueRef
	targets map[string]any
}

func (c *fakeCtx) Sample(name string) any           { return c.samples[name] }
func (c *fakeCtx) Entity(id string) any             { return c.entities[id] }
func (c *fakeCtx) ValueRef(path string) catalog.ValueRef { return c.values[path] }
func (c *fakeCtx) LookupTarget(name string) any     { return c.targets[name] }

type valueCell struct{ v any }

func (c *valueCell) Get() any   { return c.v }
func (c *valueCell) Set(v any)  { c.v = v }

func TestArgHelpers(t *testing.T) {
	args := map[string]any{
		"n":   float64(30),
		"f":   3.14,
		"s":   "hello",
		"b":   true,
		"bad": "not a number",
	}

	t.Run("ArgInt accepts float64 and int", func(t *testing.T) {
		n, err := catalog.ArgInt(args, "n")
		require.NoError(t, err)
		assert.Equal(t, 30, n)
	})

	t.Run("ArgInt rejects strings", func(t *testing.T) {
		_, err := catalog.ArgInt(args, "bad")
		assert.Error(t, err)
	})

	t.Run("ArgFloat returns float", func(t *testing.T) {
		f, err := catalog.ArgFloat(args, "f")
		require.NoError(t, err)
		assert.InDelta(t, 3.14, f, 0.001)
	})

	t.Run("ArgString returns string", func(t *testing.T) {
		s, err := catalog.ArgString(args, "s")
		require.NoError(t, err)
		assert.Equal(t, "hello", s)
	})

	t.Run("ArgBool returns bool", func(t *testing.T) {
		b, err := catalog.ArgBool(args, "b")
		require.NoError(t, err)
		assert.True(t, b)
	})

	t.Run("missing arg returns error", func(t *testing.T) {
		_, err := catalog.ArgInt(args, "missing")
		assert.Error(t, err)
	})

	t.Run("ArgIntOr returns fallback on missing", func(t *testing.T) {
		assert.Equal(t, 42, catalog.ArgIntOr(args, "missing", 42))
	})
}

func TestBuiltinSteps(t *testing.T) {
	t.Run("Wait builds a Wait step", func(t *testing.T) {
		builder := catalog.LookupStep("Wait")
		require.NotNil(t, builder)
		step, err := builder(map[string]any{"ticks": float64(2)}, nil)
		require.NoError(t, err)
		assert.False(t, step())
		assert.False(t, step())
		assert.True(t, step())
	})

	t.Run("Tween with no target ticks deterministically", func(t *testing.T) {
		builder := catalog.LookupStep("Tween")
		require.NotNil(t, builder)
		step, err := builder(map[string]any{
			"from":  float64(0),
			"to":    float64(10),
			"ticks": float64(2),
			"ease":  "linear",
		}, nil)
		require.NoError(t, err)
		assert.False(t, step())
		assert.True(t, step())
	})

	t.Run("Publish on string Target fires the event", func(t *testing.T) {
		target := pievent.NewTarget[string]()
		received := []string{}
		target.SubscribeAll(func(e string, _ pievent.Handler) {
			received = append(received, e)
		})
		ctx := &fakeCtx{targets: map[string]any{"loop.main": target}}

		builder := catalog.LookupStep("Publish")
		require.NotNil(t, builder)
		step, err := builder(map[string]any{
			"target": "loop.main",
			"event":  "ping",
		}, ctx)
		require.NoError(t, err)
		assert.True(t, step())
		assert.Equal(t, []string{"ping"}, received)
	})

	t.Run("Publish on unknown target is a no-op", func(t *testing.T) {
		ctx := &fakeCtx{targets: map[string]any{}}
		builder := catalog.LookupStep("Publish")
		step, err := builder(map[string]any{
			"target": "missing",
			"event":  "x",
		}, ctx)
		require.NoError(t, err)
		assert.True(t, step())
	})

	t.Run("Custom builds a no-op step (extension hooks deferred)", func(t *testing.T) {
		builder := catalog.LookupStep("Custom")
		step, err := builder(map[string]any{"hook": "demo"}, nil)
		require.NoError(t, err)
		assert.True(t, step())
	})

	t.Run("malformed args return descriptive error", func(t *testing.T) {
		builder := catalog.LookupStep("Wait")
		_, err := builder(map[string]any{"ticks": "thirty"}, nil)
		assert.Error(t, err)
	})

	t.Run("LookupStep returns nil for missing kinds", func(t *testing.T) {
		assert.Nil(t, catalog.LookupStep("DoesNotExist"))
	})
}

func TestBuiltinConditions(t *testing.T) {
	t.Run("event_fired matches the named event", func(t *testing.T) {
		builder := catalog.LookupCondition("event_fired")
		require.NotNil(t, builder)
		pred, err := builder(map[string]any{"event": "Jump"}, nil)
		require.NoError(t, err)
		assert.True(t, pred("Jump"))
		assert.False(t, pred("Walk"))
	})

	t.Run("value_lt compares to a constant", func(t *testing.T) {
		ctx := &fakeCtx{values: map[string]catalog.ValueRef{
			"hp": &valueCell{v: float64(5)},
		}}
		builder := catalog.LookupCondition("value_lt")
		pred, err := builder(map[string]any{
			"value":      "hp",
			"compare_to": float64(10),
		}, ctx)
		require.NoError(t, err)
		assert.True(t, pred(nil))
	})

	t.Run("value_gt and value_eq are registered", func(t *testing.T) {
		assert.NotNil(t, catalog.LookupCondition("value_gt"))
		assert.NotNil(t, catalog.LookupCondition("value_eq"))
	})

	t.Run("key_held is registered", func(t *testing.T) {
		assert.NotNil(t, catalog.LookupCondition("key_held"))
	})
}

func TestBuiltinActions(t *testing.T) {
	t.Run("set_value writes through the ValueRef", func(t *testing.T) {
		cell := &valueCell{}
		ctx := &fakeCtx{values: map[string]catalog.ValueRef{"hp": cell}}
		builder := catalog.LookupAction("set_value")
		eff, err := builder(map[string]any{
			"target": "hp",
			"value":  float64(99),
		}, ctx)
		require.NoError(t, err)
		eff(nil)
		assert.Equal(t, float64(99), cell.v)
	})

	t.Run("publish_event publishes on the named target", func(t *testing.T) {
		target := pievent.NewTarget[string]()
		got := []string{}
		target.SubscribeAll(func(e string, _ pievent.Handler) { got = append(got, e) })
		ctx := &fakeCtx{targets: map[string]any{"loop.main": target}}
		builder := catalog.LookupAction("publish_event")
		eff, err := builder(map[string]any{
			"target": "loop.main",
			"event":  "echoed",
		}, ctx)
		require.NoError(t, err)
		eff(nil)
		assert.Equal(t, []string{"echoed"}, got)
	})

	t.Run("all five builtin actions are registered", func(t *testing.T) {
		for _, name := range []string{"play_sample", "set_value", "publish_event", "move_entity", "branch"} {
			assert.NotNil(t, catalog.LookupAction(name), "missing action: %s", name)
		}
	})
}

func TestCatalog_AllNames(t *testing.T) {
	steps := catalog.AllSteps()
	assert.Contains(t, steps, "Wait")
	assert.Contains(t, steps, "Tween")
	assert.Contains(t, steps, "Move")
	assert.Contains(t, steps, "Play")
	assert.Contains(t, steps, "Publish")
	assert.Contains(t, steps, "Branch")
	assert.Contains(t, steps, "Custom")

	conditions := catalog.AllConditions()
	assert.Len(t, conditions, 5)

	actions := catalog.AllActions()
	assert.Len(t, actions, 5)
}

func TestCatalog_DuplicateRegistrationDoesNotPanic(t *testing.T) {
	// Re-registering an existing Kind should log a warning and
	// overwrite — no panic.
	assert.NotPanics(t, func() {
		catalog.RegisterStep("Wait", func(args map[string]any, _ catalog.Context) (pixelforge_routine.Step, error) {
			return func() bool { return true }, nil
		})
	})
}
