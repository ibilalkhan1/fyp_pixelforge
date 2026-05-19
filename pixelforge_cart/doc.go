// Package pixelforge_cart implements the "cart envelope" — the
// 20-byte footer the build pipeline appends to a pre-compiled
// pixelforge-player binary so the shipped artifact carries its
// game payload inline.
//
// # Wire format
//
// Every shipped binary ends with:
//
//	[CRC32   4 bytes ]   IEEE polynomial over the cart payload
//	[length  8 bytes ]   little-endian uint64, byte count of payload
//	[magic   8 bytes ]   ASCII "PFORGEv2"
//
// Total footer size is FooterSize (= 20). The reader seeks to
// fileSize - FooterSize, validates the magic, reads the declared
// length, seeks back to fileSize - FooterSize - length, reads
// length bytes, and verifies the CRC32. The pre-footer prefix is
// the original player binary; the runtime never touches it once
// loaded.
//
// # Append flow
//
// Append copies the player binary, writes the cart bytes, writes
// the footer, and renames the temp file into place atomically
// (see assetlibrary/downloader.go's matching pattern). On macOS
// it then ad-hoc-signs the result via `codesign --force --sign -
// <path>`; Apple Silicon refuses to launch a binary whose
// signature has been invalidated by post-link modification with
// `killed: 9` and no diagnostic, so re-signing is load-bearing
// even though we're not publishing through the App Store. The
// equivalent Windows-side Authenticode invariant — append BEFORE
// signing — is documented here for the v3 timeline; Authenticode
// signing itself is out of scope for v2.
//
// # Read-self flow
//
// ReadSelf opens the running executable once and holds a single
// *os.File for every seek/read. The single-FD discipline closes
// the TOCTOU window an attacker would otherwise have to swap the
// path between magic-check and payload-read; the file descriptor
// is inode-bound after open. On Linux we prefer /proc/self/exe
// over filepath.EvalSymlinks because /proc/self/exe is inherently
// race-free. The WASM build (selfread_js.go) instead reads the
// cart out of `window.__pixelforgeCart` — Ebitengine's WASM
// runtime publishes the payload there before bootstrapping the
// Go runtime; there's no file to seek inside.
//
// # Trust model (corruption, not tampering)
//
// CRC32 detects accidental corruption: bit-flips on disk, a
// truncated download, a rsync interrupted mid-stream. It does
// NOT defend against a malicious actor with write access to the
// shipped binary — they can recompute a valid CRC trivially.
// Cryptographic signing of the cart payload is deferred to v3;
// the player binary treats any well-formed CRC-valid cart as
// trusted input. Build-host operators who need tamper-evidence
// today should ship the binary inside a code-signed installer
// (Authenticode on Windows, notarized .pkg on macOS) and rely on
// the OS to verify the outer signature.
//
// # Platform split
//
// The package follows the same `//go:build js` vs `//go:build !js`
// split pixelforge_save uses:
//
//	selfread_native.go   //go:build !js
//	selfread_js.go       //go:build js
//	codesign_darwin.go   //go:build darwin
//	codesign_other.go    //go:build !darwin
//
// Append is desktop-only by intent — the build host runs on
// desktop, and the WASM runtime never appends its own cart.
package pixelforge_cart
