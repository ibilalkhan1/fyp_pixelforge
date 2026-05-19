// Package pixelforge_save is idea #6 v1 U1's save-state pipeline.
//
// The package centres on three concerns:
//
//   - Snapshot — the JSON-serializable shape carrying everything a
//     load must restore (blackboard, current scene, player tile,
//     per-scene entity overrides).
//   - Backend — the swappable storage adapter; BackendNative writes
//     to os.UserConfigDir; BackendJS writes to browser localStorage
//     (under a `//go:build js` build tag). The build-tag matrix
//     selects exactly one per binary so each platform compiles
//     cleanly without dead WASM or filesystem code paths.
//   - Service — the round-trip API the game runtime calls into:
//     SaveToSlot / LoadFromSlot / DeleteSlot, plus an autosave path
//     with a 30s minimum interval throttle.
//
// Save files are JSON with a SchemaVersion field so future format
// shifts can ship a migration step. Per editor-pforge-schema-shape.md
// the loader is forgiving: missing keys default; future versions
// log a warning and best-effort load; corrupt JSON returns an error
// without panicking.
//
// Paths are derived from the game title via Sanitize so titles
// containing spaces or special characters land at predictable
// filesystem locations (e.g. "My Game" → ".../pixelforge-games/my_game/").
package pixelforge_save
