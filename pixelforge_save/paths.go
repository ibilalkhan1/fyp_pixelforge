package pixelforge_save

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// GamesRootDirName is the constant directory the save pipeline
// rooted under os.UserConfigDir. Exported so tests + tools can
// compose the path independently.
const GamesRootDirName = "pixelforge-games"

// AutosaveSlotName is the canonical slot identifier the throttled
// autosave path writes to. Constants for the three named slots
// follow the brainstorm's R1.
const (
	AutosaveSlotName = "autosave"
	Slot1Name        = "slot1"
	Slot2Name        = "slot2"
	Slot3Name        = "slot3"
)

// Sanitize returns a filesystem-safe identifier derived from the
// game title. Empty / all-special-chars titles return "untitled"
// so the path is never empty. Spaces collapse to underscores;
// characters outside [a-z0-9-_] are stripped.
func Sanitize(title string) string {
	if title == "" {
		return "untitled"
	}
	var sb strings.Builder
	for _, r := range strings.ToLower(title) {
		switch {
		case unicode.IsLetter(r) && r < 128:
			sb.WriteRune(r)
		case unicode.IsDigit(r):
			sb.WriteRune(r)
		case r == '-' || r == '_':
			sb.WriteRune(r)
		case r == ' ':
			sb.WriteRune('_')
		}
	}
	out := sb.String()
	if out == "" {
		return "untitled"
	}
	return out
}

// GameDir returns the per-game directory under os.UserConfigDir.
// Falls back to the temp dir when UserConfigDir is unavailable
// (e.g. fresh container).
func GameDir(title string) (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		root = os.TempDir()
	}
	return filepath.Join(root, GamesRootDirName, Sanitize(title)), nil
}

// SlotPath returns the file path for the named slot. slotName is
// usually one of the constants above; arbitrary names are accepted
// to keep the API flexible for future "named save" features.
func SlotPath(title, slotName string) (string, error) {
	dir, err := GameDir(title)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, slotName+".json"), nil
}
