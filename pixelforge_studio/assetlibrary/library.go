package assetlibrary

import (
	"sort"
	"sync"
)

// Library is the in-memory index of packs available on this host.
// Populated by EnsureBootstrap after the downloader unpacks each
// pack; queried by the workspace browser (U12), the credits
// assembler (U10), and the ingest dispatcher (U11).
//
// Concurrent reads are safe; the workspace's Render loop and the
// background download goroutine both touch the index.
type Library struct {
	cacheRoot string

	mu       sync.RWMutex
	packs    map[string]Pack
	manifest *Manifest
}

// NewLibrary returns an empty Library rooted at cacheRoot. Test
// callers pass t.TempDir(); main.go passes the studio's
// os.UserCacheDir()/pixelforge.
func NewLibrary(cacheRoot string) *Library {
	return &Library{
		cacheRoot: cacheRoot,
		packs:     map[string]Pack{},
	}
}

// CacheRoot exposes the directory the library indexes. Helpful
// for diagnostics + the workspace's "where do my assets live?"
// status pane.
func (l *Library) CacheRoot() string {
	if l == nil {
		return ""
	}
	return l.cacheRoot
}

// SetManifest records the resolved manifest the downloader
// fetched. Called by EnsureBootstrap once parsing succeeds.
func (l *Library) SetManifest(m *Manifest) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.manifest = m
}

// Manifest returns the most recently set manifest, or nil when
// EnsureBootstrap hasn't completed yet.
func (l *Library) Manifest() *Manifest {
	if l == nil {
		return nil
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.manifest
}

// MarkInstalled records that pack has been verified + unpacked
// under PackDir(cacheRoot, pack.ID). Idempotent — re-marking the
// same id overwrites the prior entry.
func (l *Library) MarkInstalled(pack Pack) {
	if l == nil || pack.ID == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.packs[pack.ID] = pack
}

// IsInstalled reports whether the named pack is recorded as
// installed in the in-memory index. Doesn't re-check disk; the
// downloader is the source of truth and updates this index.
func (l *Library) IsInstalled(id string) bool {
	if l == nil {
		return false
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, ok := l.packs[id]
	return ok
}

// Installed returns every installed pack, sorted by ID. The
// workspace browser walks this to render its tabs.
func (l *Library) Installed() []Pack {
	if l == nil {
		return nil
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]Pack, 0, len(l.packs))
	for _, p := range l.packs {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// PacksForGame returns every installed pack whose Game field
// matches the supplied game id. Empty game string returns every
// pack (the workspace's "All" tab path). game="custom" returns
// no curated packs — the Custom tab reads from the user-library
// watcher's index instead (U11).
func (l *Library) PacksForGame(game string) []Pack {
	all := l.Installed()
	if game == "" || game == "all" {
		return all
	}
	if game == "custom" {
		return nil
	}
	out := make([]Pack, 0)
	for _, p := range all {
		if p.Game == game {
			out = append(out, p)
		}
	}
	return out
}

// LookupAsset resolves (packID, relPath) to the Asset entry the
// pack's manifest declares. Returns nil when either lookup
// misses. The credits assembler uses this to map a project's
// asset references to license/author metadata.
func (l *Library) LookupAsset(packID, relPath string) *Asset {
	if l == nil {
		return nil
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	pack, ok := l.packs[packID]
	if !ok {
		return nil
	}
	for i := range pack.Assets {
		if pack.Assets[i].Path == relPath {
			return &pack.Assets[i]
		}
	}
	return nil
}

// FindAsset scans every installed pack looking for an asset
// matching name (without extension or path prefix). Returns
// (packID, asset, true) on first match. Used by the credits
// assembler when the project records only a sprite name and
// needs to back-reference its source pack.
func (l *Library) FindAsset(assetName string) (string, *Asset, bool) {
	if l == nil || assetName == "" {
		return "", nil, false
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, pack := range l.packs {
		for i := range pack.Assets {
			a := &pack.Assets[i]
			if assetNameMatches(a.Path, assetName) {
				return pack.ID, a, true
			}
		}
	}
	return "", nil, false
}

// assetNameMatches reports whether the supplied bare name
// matches the pack-relative path's basename (with or without
// extension). "ship" matches "sprites/ship.png" and
// "sprites/ship".
func assetNameMatches(packPath, name string) bool {
	if packPath == name {
		return true
	}
	// Strip extension + directory prefix.
	base := packPath
	for i := len(base) - 1; i >= 0; i-- {
		if base[i] == '/' {
			base = base[i+1:]
			break
		}
	}
	for i := len(base) - 1; i >= 0; i-- {
		if base[i] == '.' {
			base = base[:i]
			break
		}
	}
	return base == name
}
