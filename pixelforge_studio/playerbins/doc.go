// Package playerbins embeds the pre-built pixelforge-player
// binaries for every shipping target (Windows .exe, macOS .app
// Mach-O, Linux ELF, and the GOOS=js/GOARCH=wasm .wasm module).
//
// The buildpipeline's host + WASM builders look up the matching
// player binary via PlayerBinaryFor(GOOS, GOARCH) and extract it to
// a temp file at build time, then call pixelforge_cart.Append to
// stitch the user's cart payload onto a copy. This is the
// "no-Go user" path: a designer running the studio installer does
// NOT need a local Go toolchain because every shippable player
// already lives inside the embedded binsFS.
//
// Release process owns the binsFS contents — `make playerbins`
// cross-compiles each target's player binary and drops the result
// at pixelforge_studio/playerbins/bins/<os>-<arch>/pixelforge-player[.exe|.wasm].
// The release pipeline runs that target before tagging so the
// shipped studio carries up-to-date players.
//
// During development the binsFS may be empty (only the .gitkeep
// placeholder) — PlayerBinaryFor returns ErrNotEmbedded in that
// case and the buildpipeline falls through to the developer
// `go build -tags=long ./cmd/pixelforge-player` fallback.
package playerbins
