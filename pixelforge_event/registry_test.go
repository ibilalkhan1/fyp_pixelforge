package pixelforge_event_test

import (
	"testing"

	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	"github.com/stretchr/testify/assert"
)

func TestRegisterTarget_EnumerateTargets(t *testing.T) {
	t.Run("alphabetical ordering", func(t *testing.T) {
		defer pievent.ResetRegistryForTest()
		pievent.ResetRegistryForTest()

		a := pievent.NewTarget[string]()
		b := pievent.NewTarget[string]()
		c := pievent.NewTarget[string]()

		pievent.RegisterTarget("zeta", a)
		pievent.RegisterTarget("alpha", b)
		pievent.RegisterTarget("mu", c)

		got := pievent.EnumerateTargets()
		var names []string
		for _, info := range got {
			names = append(names, info.Name)
		}
		assert.Equal(t, []string{"alpha", "mu", "zeta"}, names)
	})

	t.Run("subscriber and publish counts reflect underlying target", func(t *testing.T) {
		defer pievent.ResetRegistryForTest()
		pievent.ResetRegistryForTest()

		target := pievent.NewTarget[string]()
		target.SubscribeAll(func(_ string, _ pievent.Handler) {})
		target.Publish("a")
		target.Publish("b")

		pievent.RegisterTarget("demo", target)

		got := pievent.EnumerateTargets()
		require := assert.New(t)
		require.Len(got, 1)
		require.Equal("demo", got[0].Name)
		require.Equal(1, got[0].Subscribers)
		require.Equal(uint64(2), got[0].Publishes)
	})

	t.Run("LookupTarget returns the registered target", func(t *testing.T) {
		defer pievent.ResetRegistryForTest()
		pievent.ResetRegistryForTest()

		target := pievent.NewTarget[string]()
		pievent.RegisterTarget("loop.main", target)

		got := pievent.LookupTarget("loop.main")
		assert.NotNil(t, got)
		assert.Equal(t, 0, got.SubscriberCount())
	})

	t.Run("duplicate registration overwrites without panic", func(t *testing.T) {
		defer pievent.ResetRegistryForTest()
		pievent.ResetRegistryForTest()

		first := pievent.NewTarget[string]()
		second := pievent.NewTarget[string]()
		second.SubscribeAll(func(_ string, _ pievent.Handler) {})

		pievent.RegisterTarget("dup", first)
		pievent.RegisterTarget("dup", second)

		got := pievent.LookupTarget("dup")
		assert.Equal(t, 1, got.SubscriberCount())
	})

	t.Run("LookupTarget on missing name returns nil", func(t *testing.T) {
		defer pievent.ResetRegistryForTest()
		pievent.ResetRegistryForTest()
		assert.Nil(t, pievent.LookupTarget("missing"))
	})

	t.Run("EnumerateTargets returns empty slice (not nil) when empty", func(t *testing.T) {
		defer pievent.ResetRegistryForTest()
		pievent.ResetRegistryForTest()

		got := pievent.EnumerateTargets()
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("RegisterTarget rejects empty name and nil target silently", func(t *testing.T) {
		defer pievent.ResetRegistryForTest()
		pievent.ResetRegistryForTest()
		pievent.RegisterTarget("", pievent.NewTarget[string]())
		pievent.RegisterTarget("noop", nil)
		assert.Empty(t, pievent.EnumerateTargets())
	})
}
