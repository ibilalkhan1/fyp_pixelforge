# Asset-library manifest — schema & publishing checklist

Pixelforge Studio ships with an empty asset library: the curated
art/audio packs live in a separate GitHub Release that the studio
fetches on first launch. This document defines the manifest
schema that drives that fetch, lists what an actual release
needs to contain, and records the security posture for v2.

The wiring (manifest parser, downloader, in-memory library,
embedded starter fallback) is implemented in
`pixelforge_studio/assetlibrary/` and `pixelforge_studio/starterpack/`
per plan-009 U20. Publishing the manifest itself is a deployment
step — not part of the automated codebase — and is owned by the
release content workstream.

---

## Schema (v1)

The manifest is a JSON document hosted at the URL referenced by
`assetlibrary.DefaultManifestURL` (overridable per-host via the
`PIXELFORGE_ASSET_LIBRARY_URL` env var, or per-call via
`DownloadOptions.ManifestURL`).

```jsonc
{
  "schema_version": "1",
  "packs": [
    {
      "id": "asteroids-starter",         // stable id, also on-disk dir
      "version": "1.0.0",                // bumping triggers re-download
      "title": "Asteroids Starter",      // human-readable
      "game": "asteroids",               // "" for genre-neutral packs
      "url": "https://.../asteroids-1.0.0.tar.gz",
      "sha256": "deadbeef…",             // hex, verified before unpack
      "size_bytes": 1048576,              // optional, progress UI only
      "assets": [
        {
          "path": "sprites/ship.png",
          "kind": "sprite",              // "sprite" | "sfx" | "bgm"
          "license": "CC0",
          "author": "Kenney",            // optional
          "source_url": "https://kenney.nl/"
        }
      ]
    }
  ],
  "examples": [                          // plan-009 U20 addition
    {
      "id": "asteroids_proof",
      "version": "1.0.0",
      "title": "Asteroids — Reference",
      "game": "asteroids",               // optional
      "url": "https://.../asteroids_proof-1.0.0.pforge",
      "sha256": "feedface…",
      "size_bytes": 2048,
      "label": "Asteroids (reference build)",  // optional menu label
      "description": "Thrust, fire, screen-wrap demo."
    }
  ]
}
```

### Field reference

- **`schema_version`** (string, required) — gates major-format changes. Major-version bump errors with `ErrUnsupportedSchema`; the schema is otherwise additive forever (unknown fields are silently dropped).
- **`packs[]`** — downloadable asset bundles. Each pack carries the metadata above plus an `assets[]` array that powers the in-studio license badges and the auto-generated credits screen.
- **`examples[]`** — reference `.pforge` projects the studio's `File → Open Example` menu can fetch. New in U20; older manifests without the field parse cleanly as empty.

### Embedded vs downloaded packs

The studio ships an always-available CC0 **starter pack** embedded
in the binary via `pixelforge_studio/starterpack/embed.go`. It
appears in `Library.Packs()` even on a fresh install with no
network. Downloaded packs from the manifest layer on top — both
sources flow through the same `assetlibrary.Library` API, so
callers don't distinguish them.

The starter pack's metadata (`StarterPackID`, `StarterPackVersion`)
is hand-maintained alongside the embedded asset directory. Adding
sprites/SFX/BGM requires:

1. Drop the new file under `pixelforge_studio/starterpack/assets/{sprites,sfx,bgm}/`.
2. Append the asset to `starterAssets` in `embed.go` with the
   correct `License` + `Author`.
3. Bump `StarterPackVersion`.

Tests in `embed_test.go` enforce that every embedded file is
declared and every declared file is embedded.

---

## Publishing checklist (release content workstream)

Treat this as the runbook for cutting a `manifest.json` release.
It is **outside the automated plan-009 work** — the codebase ships
the wiring; the actual art curation + URL publication is a manual
deployment step.

1. **Curate the art.** Source CC0 (or clearly CC-BY-4.0 with author
   attribution) art + audio. Kenney Game Assets is the default
   bulk source; freesound.org for audio. Per-asset license must be
   captured in the manifest's `assets[].license` field — missing
   license = the credits assembler records "Unknown" and logs a
   warning.

2. **Build per-pack archives.** `.tar.gz` is the only format the
   downloader recognises today. Layout inside the archive matches
   the manifest's relative paths exactly (no top-level wrapper
   directory; the tar root is the pack root).

3. **Compute SHA-256 per archive.**
   ```sh
   sha256sum asteroids-1.0.0.tar.gz | awk '{print $1}'
   ```
   Paste each into the manifest's `packs[].sha256` field.

4. **Build the manifest JSON.** Validate locally by pointing the
   studio at the file:
   ```sh
   PIXELFORGE_ASSET_LIBRARY_URL=file:///path/to/manifest.json pf-studio
   ```
   (Note: the downloader currently uses `net/http`; for local
   validation serve from `python3 -m http.server` or similar.)

5. **Upload to GitHub Release.** Create a release tagged
   `asset-library-v<N>` and attach the manifest + every pack
   archive. The release tag is what `DefaultManifestURL` points
   at; bumping `<N>` requires editing
   `pixelforge_studio/assetlibrary/downloader.go`'s
   `DefaultManifestURL` constant.

6. **Verify on a fresh install.** Wipe `~/.cache/pixelforge/library/`,
   launch the studio, confirm the Library workspace shows the
   downloaded packs landing.

7. **For example `.pforge` releases**, the same checklist applies:
   build the `.pforge` (via `pf-studio` save), compute its
   SHA-256, attach to the release, add an entry to the
   manifest's `examples[]` array.

---

## Schema migration policy

**Additive only.** New fields can be added to any existing struct
without bumping `schema_version`. Older studio builds drop unknown
fields silently; newer fields land as zero-value when an old
manifest is parsed by a new build.

**Breaking changes** (renamed/removed/repurposed fields) require
bumping `schema_version`. Old builds then surface
`ErrUnsupportedSchema` instead of silently mis-parsing. This is
intentional: a hard error is better than corrupted asset paths.

---

## Security posture (v2)

- **HTTPS-only fetch.** The default manifest URL and every pack
  URL must be `https://`. The downloader uses `http.DefaultClient`
  which honours system CA roots; no certificate pinning today.
- **SHA-256 verification.** Every pack archive's bytes are hashed
  during download; mismatch surfaces `ErrChecksumMismatch` and the
  tmp file is removed before the error returns. Partial /
  poisoned downloads never land in the cache.
- **Path traversal guard.** `unpackTarGz` rejects any tar entry
  whose cleaned name starts with `..` or is absolute. Malicious
  manifests can't escape the pack's destination directory.

### Deferred to v3

- **The manifest itself is not signed.** A compromised GitHub
  Release (or MITM of the manifest URL on a host without HTTPS
  validation) could substitute pack URLs + their declared hashes,
  shipping malicious bytes to studio users. v2 ships TLS-only
  fetch and acknowledges this gap; v3 will add signed manifests
  (likely an Ed25519 signature alongside the JSON) per the
  doc-review finding in plan-009 §Risks.

---

## File layout

```
pixelforge_studio/
├── assetlibrary/
│   ├── manifest.go        # schema types + ParseManifest
│   ├── downloader.go      # HTTPS fetch + SHA-256 verify + tar.gz unpack
│   ├── library.go         # in-memory index (embedded + downloaded)
│   ├── startup.go         # bootstrap + EmbeddedPack wiring
│   └── …
└── starterpack/
    ├── embed.go           # //go:embed all:assets + StarterPack metadata
    ├── doc.go
    ├── assets/
    │   ├── sprites/       # 8 placeholder 16×16 PNGs (~70-90B each)
    │   ├── sfx/           # 2 silent 16-bit PCM WAVs (~4.5KB each)
    │   └── bgm/           # 1 placeholder OGG (64B)
    └── …
```

Total starter-pack disk footprint (embedded in every studio
binary): **~10KB**. Replacing the placeholders with curated
Kenney art ahead of the v2 release will likely raise this to
~500KB — well below any binary-size threshold.
