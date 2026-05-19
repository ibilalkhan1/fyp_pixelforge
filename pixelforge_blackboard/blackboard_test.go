package pixelforge_blackboard_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_blackboard"
	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
)

func TestBlackboard_SetGet_RoundTrip(t *testing.T) {
	bb := pixelforge_blackboard.New()

	bb.Set("score", 100)

	got, ok := bb.Get("score")
	require.True(t, ok)
	assert.Equal(t, 100, got)
}

func TestBlackboard_Get_MissingKey(t *testing.T) {
	bb := pixelforge_blackboard.New()

	got, ok := bb.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestBlackboard_Set_PublishesChange(t *testing.T) {
	bb := pixelforge_blackboard.New()

	var received []pixelforge_blackboard.BlackboardChange
	bb.Changes().SubscribeAll(func(c pixelforge_blackboard.BlackboardChange, _ pievent.Handler) {
		received = append(received, c)
	})

	bb.Set("score", 100)

	require.Len(t, received, 1)
	assert.Equal(t, "score", received[0].Key)
	assert.Nil(t, received[0].Old)
	assert.Equal(t, 100, received[0].New)
}

func TestBlackboard_Set_PublishesOldValueOnUpdate(t *testing.T) {
	bb := pixelforge_blackboard.New()

	var received []pixelforge_blackboard.BlackboardChange
	bb.Changes().SubscribeAll(func(c pixelforge_blackboard.BlackboardChange, _ pievent.Handler) {
		received = append(received, c)
	})

	bb.Set("score", 100)
	bb.Set("score", 200)

	require.Len(t, received, 2)
	assert.Equal(t, pixelforge_blackboard.BlackboardChange{Key: "score", Old: nil, New: 100}, received[0])
	assert.Equal(t, pixelforge_blackboard.BlackboardChange{Key: "score", Old: 100, New: 200}, received[1])
}

func TestBlackboard_ConcurrentSet_NoRace(t *testing.T) {
	bb := pixelforge_blackboard.New()

	const goroutines = 100
	const writesPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < writesPerGoroutine; i++ {
				key := keyFor(id, i)
				bb.Set(key, i)
			}
		}(g)
	}
	wg.Wait()

	// Every (goroutine, iteration) pair wrote a unique key, so the
	// final state must contain exactly goroutines * writesPerGoroutine
	// entries.
	assert.Len(t, bb.Keys(), goroutines*writesPerGoroutine)

	// Spot-check: the last write of goroutine 0 stored writesPerGoroutine-1.
	got, ok := bb.Get(keyFor(0, writesPerGoroutine-1))
	require.True(t, ok)
	assert.Equal(t, writesPerGoroutine-1, got)
}

func keyFor(goroutineID, iteration int) string {
	return "g" + itoa(goroutineID) + "-" + itoa(iteration)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func TestBlackboard_Inspectable_RegistersOnInit(t *testing.T) {
	got := pievent.LookupTarget(pixelforge_blackboard.TargetName)
	require.NotNil(t, got, "package import should self-register the default blackboard")

	// The registered target must be the package's Default instance and
	// must implement the Inspectable contract (which it does by virtue
	// of being a *Blackboard).
	bb, ok := got.(*pixelforge_blackboard.Blackboard)
	require.True(t, ok, "registered target must be *Blackboard")
	assert.Same(t, pixelforge_blackboard.Default(), bb)

	// Inspectable surface: both methods are callable without panic.
	_ = got.SubscriberCount()
	_ = got.PublishCount()
}

func TestBlackboard_CanonicalKeys_TypeDeclared(t *testing.T) {
	required := []string{
		pixelforge_blackboard.KeyScore,
		pixelforge_blackboard.KeyLives,
		pixelforge_blackboard.KeyHealth,
		pixelforge_blackboard.KeyCurrentScene,
		pixelforge_blackboard.KeyPlayerX,
		pixelforge_blackboard.KeyPlayerY,
	}

	for _, key := range required {
		spec, ok := pixelforge_blackboard.CanonicalKeys[key]
		require.True(t, ok, "canonical key %q must be declared", key)
		assert.NotEmpty(t, string(spec.Type), "canonical key %q must declare a Type", key)
		assert.NotNil(t, spec.Default, "canonical key %q must declare a Default", key)
	}
}

func TestBlackboard_KeysSnapshot_Stable(t *testing.T) {
	bb := pixelforge_blackboard.New()
	bb.Set("score", 100)
	bb.Set("lives", 3)
	bb.Set("health", 3)

	snap := bb.KeysSnapshot()
	require.Len(t, snap, 3)

	// Mutating the returned slice (or appending past its capacity)
	// must not affect the blackboard's internal key set.
	original := append([]string(nil), snap...)
	for i := range snap {
		snap[i] = "MUTATED"
	}
	snap = append(snap, "extra")

	again := bb.KeysSnapshot()
	assert.Equal(t, original, again, "internal state changed after mutating snapshot")
	assert.NotContains(t, again, "extra")
	assert.NotContains(t, again, "MUTATED")
}

// ===== idea #6 v1 U2 — Snapshot / Restore =====

func TestSnapshot_EmptyBlackboardReturnsEmptyMap(t *testing.T) {
	bb := pixelforge_blackboard.New()
	snap := bb.Snapshot()
	assert.NotNil(t, snap)
	assert.Empty(t, snap)
}

func TestSnapshot_PopulatedBlackboardReturnsAllKeys(t *testing.T) {
	bb := pixelforge_blackboard.New()
	bb.Set("score", 100)
	bb.Set("name", "hero")
	bb.Set("alive", true)
	snap := bb.Snapshot()
	assert.Equal(t, map[string]any{"score": 100, "name": "hero", "alive": true}, snap)
}

func TestSnapshot_DeepCopiesNestedMaps(t *testing.T) {
	bb := pixelforge_blackboard.New()
	bb.Set("inventory", map[string]any{"items": []any{"sword"}})
	snap := bb.Snapshot()

	// Mutate the snapshot's nested map.
	inv := snap["inventory"].(map[string]any)
	inv["items"] = append(inv["items"].([]any), "shield")

	// Original blackboard's value is unchanged.
	original, _ := bb.Get("inventory")
	originalItems := original.(map[string]any)["items"].([]any)
	assert.Len(t, originalItems, 1)
}

func TestSnapshot_DoesNotPublishEvents(t *testing.T) {
	bb := pixelforge_blackboard.New()
	bb.Set("score", 100)
	received := 0
	bb.Changes().SubscribeAll(func(c pixelforge_blackboard.BlackboardChange, _ pievent.Handler) { received++ })
	_ = bb.Snapshot()
	assert.Equal(t, 0, received, "Snapshot is read-only and publishes nothing")
}

func TestRestore_ReplacesAllKeys(t *testing.T) {
	bb := pixelforge_blackboard.New()
	bb.Set("a", 1)
	bb.Set("b", 2)
	bb.Restore(map[string]any{"c": 3, "d": 4})

	_, okA := bb.Get("a")
	assert.False(t, okA, "Restore drops keys not in the new map")
	_, okB := bb.Get("b")
	assert.False(t, okB)
	v, _ := bb.Get("c")
	assert.Equal(t, 3, v)
}

func TestRestore_PublishesChangeEventForEveryNewKey(t *testing.T) {
	bb := pixelforge_blackboard.New()
	var seen []string
	var mu sync.Mutex
	bb.Changes().SubscribeAll(func(c pixelforge_blackboard.BlackboardChange, _ pievent.Handler) {
		mu.Lock()
		seen = append(seen, c.Key)
		mu.Unlock()
	})
	bb.Restore(map[string]any{"a": 1, "b": 2})
	mu.Lock()
	defer mu.Unlock()
	assert.ElementsMatch(t, []string{"a", "b"}, seen)
}

func TestRestore_PublishesRemovalEventForDroppedKeys(t *testing.T) {
	bb := pixelforge_blackboard.New()
	bb.Set("a", 1)
	bb.Set("b", 2)
	var removed []string
	var mu sync.Mutex
	bb.Changes().SubscribeAll(func(c pixelforge_blackboard.BlackboardChange, _ pievent.Handler) {
		mu.Lock()
		if c.New == nil {
			removed = append(removed, c.Key)
		}
		mu.Unlock()
	})
	bb.Restore(map[string]any{})
	mu.Lock()
	defer mu.Unlock()
	assert.ElementsMatch(t, []string{"a", "b"}, removed,
		"every dropped key publishes a removal event with New=nil")
}

func TestSnapshotRestore_RoundTripPreservesState(t *testing.T) {
	bb := pixelforge_blackboard.New()
	bb.Set("score", 100)
	bb.Set("inventory", map[string]any{"items": []any{"sword", "shield"}})
	snap := bb.Snapshot()

	// Mutate.
	bb.Set("score", 999)
	bb.Set("name", "intruder")

	// Restore the snapshot — score returns to 100, name removed.
	bb.Restore(snap)
	v, _ := bb.Get("score")
	assert.Equal(t, 100, v)
	_, ok := bb.Get("name")
	assert.False(t, ok)
}

func TestRestore_ConcurrentReadDuringRestoreSafe(t *testing.T) {
	bb := pixelforge_blackboard.New()
	bb.Set("counter", 0)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_, _ = bb.Get("counter")
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			bb.Restore(map[string]any{"counter": i})
		}
	}()
	wg.Wait()
}
