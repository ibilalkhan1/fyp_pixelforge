// Package starterpack ships the always-available CC0 placeholder
// asset set that the studio falls back to when the curated asset
// library hasn't downloaded yet (fresh install + offline, or the
// background bootstrap is still in flight).
//
// The pack is embedded via `//go:embed all:assets` so it's part of
// the studio binary — no network dependency, no first-launch race
// on whether the user has anything to drag into a scene. The
// downloaded library packs (Asteroids, Mario, Bomberman, DK
// curated art) layer on top of this; both flow through
// assetlibrary.Library so callers don't care which side a pack
// came from.
//
// Placeholders, not production art: the embedded sprites are
// 16x16 solid-color PNGs and the SFX are silent 16-bit PCM WAV
// files. They prove the wiring (embed → fs.FS → library → editor)
// without inflating the binary with real art. Replacing them with
// curated Kenney CC0 art is the v2-actual-release deployment
// step; see docs/asset-library-manifest.md for the publishing
// checklist.
//
// Per plan-009 U20 (docs/plans/2026-05-19-001-feat-arcade-
// shipping-v2-plan.md). The Pack metadata returned by StarterPack()
// is registered into assetlibrary.Library on construction so the
// in-memory index always contains "starter" alongside whatever
// the downloader fetches.
package starterpack
