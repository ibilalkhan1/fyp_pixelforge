package assetlibrary

import (
	"io/fs"
	"sort"
	"sync"
)

// Library is the in-memory index of packs available on this host.
// Populated by EnsureBootstrap after the downloader unpacks each
// pack; queried by the workspace browser (U12), the credits
// assembler (U10), and the ingest dispatcher (U11).
//
// Two layers coexist in the same index:
//   - Embedded packs (registered via MarkEmbedded): always
//     available, byte-content resolved through an fs.FS supplied
//     at registration (plan-009 U20's starterpack).
//   - Downloaded packs (registered via MarkInstalled): land after
//     the bootstrap fetches them, byte-content resolved on-disk
//     under PackDir.
//
// Callers that just want metadata (Library.Pack, LookupAsset)
// don't care which layer a pack came from; callers that need
// bytes (the workspace's preview render, the credits scanner)
// dispatch on EmbeddedFS to pick the right resolver.
//
// Concurrent reads are safe; the workspace's Render loop and the
// background download goroutine both touch the index.
type Library struct {
	cacheRoot string

	mu          sync.RWMutex
	packs       map[string]Pack
	embeddedFS  map[string]fs.FS
	manifest    *Manifest
}

// NewLibrary returns an empty Library rooted at cacheRoot. Test
// callers pass t.TempDir(); main.go passes the studio's
// os.UserCacheDir()/pixelforge.
//
// Embedded packs (the always-available starter set) are wired in
// after construction via MarkEmbedded so the assetlibrary package
// doesn't take a dependency on starterpack (which itself imports
// assetlibrary for the Pack/Asset types — calling NewLibrary
// from inside starterpack would close an import cycle).
func NewLibrary(cacheRoot string) *Library {
	return &Library{
		cacheRoot:  cacheRoot,
		packs:      map[string]Pack{},
		embeddedFS: map[string]fs.FS{},
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

// MarkEmbedded registers an always-available pack whose bytes
// live in source rather than under PackDir. Source is typically
// the starter pack's embed.FS (plan-009 U20); callers retrieve
// it later via EmbeddedFS. The pack's metadata flows through the
// same packs map as downloaded packs so Installed/Packs/Pack
// don't need to merge two slices.
//
// Idempotent — re-marking the same id overwrites both the
// metadata and the FS handle. nil source is permitted (lets a
// test register pack metadata without bytes) but
// EmbeddedFS(id) will then return (nil, false).
func (l *Library) MarkEmbedded(pack Pack, source fs.FS) {
	if l == nil || pack.ID == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.packs[pack.ID] = pack
	if source != nil {
		l.embeddedFS[pack.ID] = source
	}
}

// EmbeddedFS returns the fs.FS registered for an embedded pack
// id, or (nil, false) when the id either isn't registered or was
// registered without bytes. Callers use this to distinguish
// "resolve via fs.FS" from "resolve via os.Open(PackDir/...)".
func (l *Library) EmbeddedFS(id string) (fs.FS, bool) {
	if l == nil {
		return nil, false
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	src, ok := l.embeddedFS[id]
	return src, ok
}

// IsEmbedded reports whether the named pack was registered via
// MarkEmbedded (and therefore should be byte-resolved through
// EmbeddedFS rather than the on-disk PackDir).
func (l *Library) IsEmbedded(id string) bool {
	if l == nil {
		return false
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, ok := l.embeddedFS[id]
	return ok
}

// Pack returns the registered pack with the given id, or nil
// when no pack matches. Embedded + downloaded packs share the
// same map so callers don't need to ask twice.
func (l *Library) Pack(id string) *Pack {
	if l == nil {
		return nil
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	p, ok := l.packs[id]
	if !ok {
		return nil
	}
	// Return a defensive copy so callers can't mutate our internal
	// state. Pack.Assets is a slice; deep-copy it too.
	out := p
	if len(p.Assets) > 0 {
		out.Assets = make([]Asset, len(p.Assets))
		copy(out.Assets, p.Assets)
	}
	return &out
}

// Packs is an alias for Installed that reads naturally for
// callers that don't distinguish "installed" vs "embedded" — both
// layers appear in the returned slice. Plan-009 U20+U21 uses
// this name when surfacing the unified pack list.
func (l *Library) Packs() []Pack {
	return l.Installed()
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

// Examples returns the example list from the currently-loaded
// manifest, sorted by ID for stable menu rendering. Returns an
// empty slice when the manifest hasn't loaded yet or carries no
// examples — callers can range without nil-checking. Plan-009
// U21's File → Open Example menu walks this slice each frame.
func (l *Library) Examples() []Example {
	if l == nil {
		return nil
	}
	l.mu.RLock()
	m := l.manifest
	l.mu.RUnlock()
	if m == nil || len(m.Examples) == 0 {
		return []Example{}
	}
	out := make([]Example, len(m.Examples))
	copy(out, m.Examples)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// LookupExample returns the example with the given ID from the
// currently-loaded manifest, or nil when no example matches or
// the manifest hasn't loaded yet. Plan-009 U21's File → Open
// Example menu uses this to resolve a clicked row to its Example
// metadata before kicking off FetchExample.
func (l *Library) LookupExample(id string) *Example {
	if l == nil {
		return nil
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.manifest == nil {
		return nil
	}
	for i := range l.manifest.Examples {
		if l.manifest.Examples[i].ID == id {
			ex := l.manifest.Examples[i]
			return &ex
		}
	}
	return nil
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
