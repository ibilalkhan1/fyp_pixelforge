package pixelforge_project

import "fmt"

// SchemaVersion is the wire-format version this package writes. Loaders
// dispatch on the saved file's schema_version field; future versions
// bump this constant and add a forward-migration in MigrateForward.
const SchemaVersion = 1

// ErrUnsupportedSchemaVersion is returned by Load when a file requests
// a schema newer than this binary understands. Callers should display
// the editor version + schema range to the user so they know which
// build to upgrade to.
type ErrUnsupportedSchemaVersion struct {
	Got, Want int
}

func (e *ErrUnsupportedSchemaVersion) Error() string {
	return fmt.Sprintf(
		"pforge schema version %d is newer than this build (max %d) — upgrade Pixelforge Studio",
		e.Got, e.Want)
}

// MigrateForward upgrades p from its on-disk SchemaVersion to the
// current SchemaVersion. No migrations exist at v1 — this is a
// placeholder for future versions so the loader has a single dispatch
// point.
func MigrateForward(p *Project) error {
	if p.SchemaVersion > SchemaVersion {
		return &ErrUnsupportedSchemaVersion{Got: p.SchemaVersion, Want: SchemaVersion}
	}
	// v1 — no migrations yet.
	p.SchemaVersion = SchemaVersion
	return nil
}
