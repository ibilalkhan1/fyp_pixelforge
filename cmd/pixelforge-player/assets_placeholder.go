// Package main — pixelforge-player asset placeholder.
//
// The universal player binary does not ship any assets of its own:
// the .pforge cart appended at Build → Host carries everything the
// runtime needs. capsuleruntime.Boot still requires an fs.FS argument
// though, so this file supplies a guaranteed-empty embed.FS rooted at
// an "assets_placeholder/" directory containing a single .keep file.
//
// U3 will pivot this seam: once the cart payload carries its own
// asset blobs, Boot will receive an fs.FS view of the in-cart
// asset tree instead of this placeholder. The placeholder stays in
// the source so the player still builds when the cart-side asset FS
// is absent (e.g. an empty smoke-test cart).
package main

import "embed"

//go:embed all:assets_placeholder
var embeddedAssets embed.FS
