// Package main — pixelforge-player asset placeholder.
//
// The universal player binary does not ship any assets of its own:
// the .pforge cart appended at Build → Host carries everything the
// runtime needs. capsuleruntime.Boot still requires an fs.FS argument
// though, so this file supplies a guaranteed-empty embed.FS rooted at
// an "assets_placeholder/" directory containing a single .keep file.
//
// Asset-bearing carts supply a read-only ZIP filesystem in run(). This
// empty filesystem is retained only for legacy JSON-only carts.
package main

import "embed"

//go:embed all:assets_placeholder
var embeddedAssets embed.FS
