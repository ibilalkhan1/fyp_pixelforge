// Package catalog hosts the verb-recipe + Step/Condition/Action Kind
// registries used by the M5 visual scripting runtime + studio
// authoring surfaces.
//
// The catalog's authoritative reference doc lives at
// docs/verb-catalog.md and is regenerated from this package's
// registries by the cmd/gendocs sub-binary. CI verifies the doc is
// up-to-date by running the directive below and asserting the file
// has not drifted; designers + LLM-assisted authoring agents read
// the regenerated file as the source of truth for "what verbs exist".
//
// Regenerate after registry changes:
//
//	go generate ./pixelforge_studio/scripting/catalog/...
//
// (The Makefile target `verb-catalog` wraps the same invocation.)

//go:generate go run ./cmd/gendocs -out ../../../docs/verb-catalog.md

package catalog
