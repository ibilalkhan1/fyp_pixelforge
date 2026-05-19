package pixelforge_save

import (
	"fmt"
	"sync"
	"time"
)

// AutosaveThrottle is the minimum interval between consecutive
// successful autosaves. Brainstorm prescribes 30s; tunable for
// tests via NewServiceWithThrottle.
const AutosaveThrottle = 30 * time.Second

// Service wraps a Backend with the save-pipeline contract the
// runtime calls into. Owns the autosave-throttle state.
type Service struct {
	backend Backend

	mu            sync.Mutex
	lastAutosave  time.Time
	throttle      time.Duration
	clock         func() time.Time // injectable for tests
}

// NewService returns a Service bound to backend using the default
// AutosaveThrottle + the wall clock.
func NewService(backend Backend) *Service {
	return NewServiceWithThrottle(backend, AutosaveThrottle, time.Now)
}

// NewServiceWithThrottle is the test-friendly constructor — pass a
// shorter throttle and an injectable clock so the autosave-
// throttle tests don't have to sleep.
func NewServiceWithThrottle(backend Backend, throttle time.Duration, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{backend: backend, throttle: throttle, clock: clock}
}

// SaveToSlot writes the snapshot to the named slot. Bypasses the
// autosave throttle — designer-initiated saves always succeed.
func (s *Service) SaveToSlot(snapshot Snapshot, slotName string) error {
	if s == nil || s.backend == nil {
		return fmt.Errorf("save: nil service / backend")
	}
	if snapshot.SchemaVersion == 0 {
		snapshot.SchemaVersion = CurrentSchemaVersion
	}
	if snapshot.SavedAt.IsZero() {
		snapshot.SavedAt = s.clock()
	}
	data, err := snapshot.MarshalToJSON()
	if err != nil {
		return err
	}
	return s.backend.Write(slotName, data)
}

// LoadFromSlot reads + unmarshals the named slot. Returns the
// zero Snapshot + error when the slot is missing or corrupt.
func (s *Service) LoadFromSlot(slotName string) (Snapshot, error) {
	if s == nil || s.backend == nil {
		return Snapshot{}, fmt.Errorf("save: nil service / backend")
	}
	data, err := s.backend.Read(slotName)
	if err != nil {
		return Snapshot{}, err
	}
	return UnmarshalSnapshot(data)
}

// DeleteSlot removes the named slot. Missing slots are not an
// error (idempotent).
func (s *Service) DeleteSlot(slotName string) error {
	if s == nil || s.backend == nil {
		return fmt.Errorf("save: nil service / backend")
	}
	return s.backend.Delete(slotName)
}

// Autosave writes the snapshot to the canonical AutosaveSlotName
// IF the throttle window has elapsed. Returns (true, nil) when
// the write happened, (false, nil) when throttled, (false, err)
// on backend failure.
func (s *Service) Autosave(snapshot Snapshot) (bool, error) {
	if s == nil || s.backend == nil {
		return false, fmt.Errorf("save: nil service / backend")
	}
	s.mu.Lock()
	now := s.clock()
	if !s.lastAutosave.IsZero() && now.Sub(s.lastAutosave) < s.throttle {
		s.mu.Unlock()
		return false, nil
	}
	s.lastAutosave = now
	s.mu.Unlock()
	return true, s.SaveToSlot(snapshot, AutosaveSlotName)
}

// ListSlots returns the backend's slot metadata.
func (s *Service) ListSlots() ([]SlotMeta, error) {
	if s == nil || s.backend == nil {
		return nil, fmt.Errorf("save: nil service / backend")
	}
	return s.backend.List()
}
