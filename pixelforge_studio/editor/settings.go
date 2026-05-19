package editor

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MaxRecentProjects caps the recent-project list. Beyond this we drop the
// oldest entries, keeping most-recent-first order.
const MaxRecentProjects = 10

// settingsFileName is the JSON filename inside the editor's config
// directory. We avoid dotfiles since os.UserConfigDir already returns a
// hidden config root on each platform.
const settingsFileName = "settings.json"

// settingsAutosaveDelay is the idle period after a mutation before
// settings are written to disk. Coalescing repeated edits (e.g. a window
// drag) into one write keeps disk traffic low.
const settingsAutosaveDelay = 500 * time.Millisecond

// Settings is the editor's user-scoped preferences. Persisted to disk in
// the platform user-config directory.
//
// The struct intentionally exposes plain JSON-tagged fields so a power
// user can edit settings.json by hand if needed.
type Settings struct {
	WindowWidth    int      `json:"window_width"`
	WindowHeight   int      `json:"window_height"`
	Theme          string   `json:"theme"`
	RecentProjects []string `json:"recent_projects"`

	// CaptureBudgetFrames sizes the M4 continuous-capture ring buffer.
	// Default 300 frames (10s @ 30 TPS). Resizing in the Capture
	// workspace persists here.
	CaptureBudgetFrames int `json:"capture_budget_frames"`

	// LogicalScale is the integer multiplier the editor's chrome
	// renders at against the window resolution. 1 keeps M0 behaviour;
	// 2/3/4 give crisper text on Hi-DPI displays. See U47.
	LogicalScale int `json:"logical_scale"`

	// LeftPanelW / RightPanelW were the M2 native chrome's drag-
	// resize gutter widths. U6 of the ImGui migration handed dock
	// geometry persistence over to imgui.ini; these fields persist
	// only for backwards compatibility with settings.json files
	// written by older builds, and are otherwise unused.
	LeftPanelW  int `json:"left_panel_width,omitempty"`
	RightPanelW int `json:"right_panel_width,omitempty"`

	// BuildOnSaveDisabled gates the build-on-save daemon (idea #7
	// v1 U8). Default false: every save triggers a debounced
	// host-platform build so the binary is always one click away.
	// Designers who want quieter saves toggle this off via the
	// View menu; the value persists across studio sessions.
	BuildOnSaveDisabled bool `json:"build_on_save_disabled,omitempty"`

	path string // resolved on Load; empty when only in-memory

	mu          sync.Mutex
	pendingSave bool
	saveTimer   *time.Timer
}

// DefaultSettings returns the settings used on a fresh install or when
// the on-disk file is missing/corrupt.
func DefaultSettings() *Settings {
	return &Settings{
		WindowWidth:         defaultWindowWidth,
		WindowHeight:        defaultWindowHeight,
		Theme:               "dark",
		RecentProjects:      []string{},
		CaptureBudgetFrames: 300,
		LogicalScale:        1,
	}
}

// LoadSettings reads settings from the platform user-config directory.
// A missing file is not an error — defaults are returned. A malformed
// file is logged and defaults are returned, never propagated as a panic.
func LoadSettings() *Settings {
	path, err := SettingsPath()
	if err != nil {
		log.Printf("pixelforge studio: could not resolve settings path: %v", err)
		s := DefaultSettings()
		return s
	}
	return LoadSettingsAt(path)
}

// LoadSettingsAt loads from an explicit path. Used by tests to keep
// each run hermetic; production code calls LoadSettings.
func LoadSettingsAt(path string) *Settings {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		s := DefaultSettings()
		s.path = path
		return s
	}
	if err != nil {
		log.Printf("pixelforge studio: read settings %s: %v (using defaults)", path, err)
		s := DefaultSettings()
		s.path = path
		return s
	}
	s := DefaultSettings()
	if err := json.Unmarshal(data, s); err != nil {
		log.Printf("pixelforge studio: settings %s is malformed: %v (using defaults)", path, err)
		s = DefaultSettings()
	}
	s.path = path
	s.sanitize()
	return s
}

// SettingsPath returns the canonical settings file location on this
// platform. Created on demand by Save.
func SettingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pixelforge-studio", settingsFileName), nil
}

// Save persists the current settings to disk. Creates any missing
// parent directories. Synchronous — callers wanting debounced writes
// should use MarkDirty instead.
func (s *Settings) Save() error {
	s.mu.Lock()
	path := s.path
	s.mu.Unlock()
	if path == "" {
		resolved, err := SettingsPath()
		if err != nil {
			return err
		}
		path = resolved
		s.mu.Lock()
		s.path = path
		s.mu.Unlock()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// MarkDirty schedules a save after settingsAutosaveDelay of idle time.
// Repeated calls within the window collapse into a single write.
func (s *Settings) MarkDirty() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingSave = true
	if s.saveTimer != nil {
		s.saveTimer.Reset(settingsAutosaveDelay)
		return
	}
	s.saveTimer = time.AfterFunc(settingsAutosaveDelay, func() {
		if err := s.Save(); err != nil {
			log.Printf("pixelforge studio: autosave settings: %v", err)
		}
		s.mu.Lock()
		s.pendingSave = false
		s.saveTimer = nil
		s.mu.Unlock()
	})
}

// PushRecentProject records a freshly opened project at the front of
// the recents list, deduping prior entries and clamping length.
// Returns true if the list mutated.
func (s *Settings) PushRecentProject(path string) bool {
	if path == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Dedupe: remove any existing entry for this path.
	filtered := s.RecentProjects[:0]
	for _, p := range s.RecentProjects {
		if p != path {
			filtered = append(filtered, p)
		}
	}
	// Prepend the new entry.
	s.RecentProjects = append([]string{path}, filtered...)
	if len(s.RecentProjects) > MaxRecentProjects {
		s.RecentProjects = s.RecentProjects[:MaxRecentProjects]
	}
	return true
}

// sanitize repairs out-of-range fields that survived an old or
// hand-edited settings file. Called on load.
func (s *Settings) sanitize() {
	if s.WindowWidth <= 0 {
		s.WindowWidth = defaultWindowWidth
	}
	if s.WindowHeight <= 0 {
		s.WindowHeight = defaultWindowHeight
	}
	if s.Theme == "" {
		s.Theme = "dark"
	}
	if len(s.RecentProjects) > MaxRecentProjects {
		s.RecentProjects = s.RecentProjects[:MaxRecentProjects]
	}
	if s.RecentProjects == nil {
		s.RecentProjects = []string{}
	}
	if s.CaptureBudgetFrames <= 0 {
		s.CaptureBudgetFrames = 300
	}
	if s.LogicalScale < 1 {
		s.LogicalScale = 1
	} else if s.LogicalScale > 4 {
		s.LogicalScale = 4
	}
	if s.LeftPanelW < 0 {
		s.LeftPanelW = 0
	}
	if s.RightPanelW < 0 {
		s.RightPanelW = 0
	}
}
