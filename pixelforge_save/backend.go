package pixelforge_save

import (
	"errors"
	"time"
)

// ErrNotFound is returned by Backend.Read when the requested slot
// does not exist. The WASM backend (backend_js.go) returns this
// sentinel directly; the native backend wraps os.ErrNotExist via
// fmt.Errorf and callers wanting cross-backend uniformity can
// check both with errors.Is.
var ErrNotFound = errors.New("pixelforge_save: slot not found")

// ErrQuotaExceeded is returned by Backend.Write when the storage
// layer rejects the payload because the per-origin quota is full.
// The WASM backend surfaces browser QuotaExceededError this way;
// the native backend never returns it (filesystem quota errors
// surface as raw fs errors instead). Engine UX layers compare
// with errors.Is.
var ErrQuotaExceeded = errors.New("pixelforge_save: storage quota exceeded")

// JSKeyPrefix returns the localStorage key prefix BackendJS uses
// for the given game title. Exposed (and host-testable) so callers
// that need to enumerate or migrate WASM-side keys can derive the
// namespace without instantiating the build-tagged Backend.
func JSKeyPrefix(gameTitle string) string {
	return "pixelforge.save." + Sanitize(gameTitle) + "."
}

// SlotMeta is the lightweight per-slot summary List returns.
// Includes a Saved-at timestamp so the Save / Load menu can sort
// entries by recency.
type SlotMeta struct {
	Name    string
	SavedAt time.Time
	Size    int64
}

// Backend is the swappable storage adapter. Concrete backends:
// BackendNative (filesystem under os.UserConfigDir) and
// BackendWASM (browser localStorage, build-tagged js).
type Backend interface {
	// Read fetches the bytes stored under slotName.
	Read(slotName string) ([]byte, error)
	// Write stores bytes under slotName, replacing any existing
	// data.
	Write(slotName string, data []byte) error
	// Delete removes the supplied slot. Missing slots return nil
	// (idempotent).
	Delete(slotName string) error
	// List enumerates every slot the backend currently carries.
	List() ([]SlotMeta, error)
}
