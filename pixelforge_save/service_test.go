//go:build !js

package pixelforge_save_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"
)

func newService(t *testing.T) *pisave.Service {
	t.Helper()
	b := newTempBackend(t)
	return pisave.NewService(b)
}

func TestService_SaveAndLoadRoundTrip(t *testing.T) {
	s := newService(t)
	snap := pisave.Snapshot{
		SchemaVersion: 1,
		GameTitle:     "test",
		Blackboard:    map[string]any{"score": float64(100)},
	}
	require.NoError(t, s.SaveToSlot(snap, pisave.Slot1Name))
	got, err := s.LoadFromSlot(pisave.Slot1Name)
	require.NoError(t, err)
	assert.Equal(t, "test", got.GameTitle)
	assert.Equal(t, float64(100), got.Blackboard["score"])
}

func TestService_SaveStampSchemaVersionAndTimestamp(t *testing.T) {
	s := newService(t)
	require.NoError(t, s.SaveToSlot(pisave.Snapshot{}, pisave.Slot1Name))
	got, err := s.LoadFromSlot(pisave.Slot1Name)
	require.NoError(t, err)
	assert.Equal(t, pisave.CurrentSchemaVersion, got.SchemaVersion)
	assert.False(t, got.SavedAt.IsZero(),
		"SavedAt populated by Service when caller leaves it zero")
}

func TestService_DeleteSlot(t *testing.T) {
	s := newService(t)
	require.NoError(t, s.SaveToSlot(pisave.Snapshot{}, pisave.Slot1Name))
	require.NoError(t, s.DeleteSlot(pisave.Slot1Name))
	_, err := s.LoadFromSlot(pisave.Slot1Name)
	require.Error(t, err)
}

func TestService_ListSlots(t *testing.T) {
	s := newService(t)
	require.NoError(t, s.SaveToSlot(pisave.Snapshot{}, pisave.Slot1Name))
	require.NoError(t, s.SaveToSlot(pisave.Snapshot{}, pisave.AutosaveSlotName))
	slots, err := s.ListSlots()
	require.NoError(t, err)
	names := []string{}
	for _, m := range slots {
		names = append(names, m.Name)
	}
	assert.ElementsMatch(t, []string{"slot1", "autosave"}, names)
}

func TestService_AutosaveFirstCallWrites(t *testing.T) {
	s := pisave.NewServiceWithThrottle(newTempBackend(t), 10*time.Second,
		fixedClock(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)))
	wrote, err := s.Autosave(pisave.Snapshot{})
	require.NoError(t, err)
	assert.True(t, wrote, "first autosave always writes")
}

func TestService_AutosaveThrottleSkipsWithinWindow(t *testing.T) {
	clock := newSteppingClock(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))
	s := pisave.NewServiceWithThrottle(newTempBackend(t), 10*time.Second, clock.Now)
	wrote1, err := s.Autosave(pisave.Snapshot{})
	require.NoError(t, err)
	require.True(t, wrote1)
	clock.advance(5 * time.Second)
	wrote2, err := s.Autosave(pisave.Snapshot{})
	require.NoError(t, err)
	assert.False(t, wrote2, "within-window autosave throttled")
}

func TestService_AutosaveAcceptedAfterThrottleWindow(t *testing.T) {
	clock := newSteppingClock(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))
	s := pisave.NewServiceWithThrottle(newTempBackend(t), 10*time.Second, clock.Now)
	_, _ = s.Autosave(pisave.Snapshot{})
	clock.advance(11 * time.Second)
	wrote, err := s.Autosave(pisave.Snapshot{})
	require.NoError(t, err)
	assert.True(t, wrote, "autosave allowed after throttle window")
}

func TestService_SaveToSlotBypassesThrottle(t *testing.T) {
	clock := newSteppingClock(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))
	s := pisave.NewServiceWithThrottle(newTempBackend(t), 10*time.Second, clock.Now)
	for i := 0; i < 5; i++ {
		require.NoError(t, s.SaveToSlot(pisave.Snapshot{}, pisave.Slot1Name))
	}
	// All 5 writes succeed despite no throttle window elapsing.
	got, err := s.LoadFromSlot(pisave.Slot1Name)
	require.NoError(t, err)
	assert.NotNil(t, got)
}

func TestService_NilSafe(t *testing.T) {
	var s *pisave.Service
	_, err := s.LoadFromSlot("x")
	require.Error(t, err)
	err = s.SaveToSlot(pisave.Snapshot{}, "x")
	require.Error(t, err)
}

// fixedClock returns a clock function that always reports t.
func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

// steppingClock returns a clock function whose value advances by
// the explicit Step calls. Used by throttle tests.
type steppingClock struct{ now time.Time }

func newSteppingClock(start time.Time) *steppingClock { return &steppingClock{now: start} }
func (c *steppingClock) Now() time.Time               { return c.now }
func (c *steppingClock) advance(d time.Duration)      { c.now = c.now.Add(d) }
