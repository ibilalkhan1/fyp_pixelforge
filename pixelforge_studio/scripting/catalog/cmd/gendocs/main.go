// Command gendocs regenerates docs/verb-catalog.md from the verb-recipe
// registry in pixelforge_studio/scripting/catalog. It is invoked via
// `go generate ./pixelforge_studio/scripting/catalog/...` (the
// directive lives in catalog/doc.go) and ships as a standalone binary
// so CI / Makefile can run it without a build-tag round-trip.
//
// Output is deterministic — two invocations produce byte-identical
// markdown — so a "go generate && check-no-diff" gate is meaningful.
// The deterministic writer lives in
// pixelforge_studio/scripting/catalog/docgen.go so tests can exercise
// it without spawning a subprocess.
//
// Usage:
//
//	go run ./pixelforge_studio/scripting/catalog/cmd/gendocs           # stdout
//	go run ./pixelforge_studio/scripting/catalog/cmd/gendocs -out FILE # write to FILE
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
)

func main() {
	out := flag.String("out", "", "output file path (default stdout)")
	flag.Parse()

	if *out == "" {
		if err := catalog.WriteVerbCatalogMarkdown(os.Stdout); err != nil {
			log.Fatalf("gendocs: write stdout: %v", err)
		}
		return
	}

	// Write to a temp file in the destination's directory then
	// rename — avoids leaving a half-written verb-catalog.md when
	// the generator panics partway through.
	dir := filepath.Dir(*out)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("gendocs: mkdir %q: %v", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".verb-catalog-*.md.tmp")
	if err != nil {
		log.Fatalf("gendocs: create temp: %v", err)
	}
	tmpName := tmp.Name()
	defer func() {
		// Best-effort cleanup — only triggers if the rename below
		// never ran (panic / early return).
		_ = os.Remove(tmpName)
	}()

	if err := catalog.WriteVerbCatalogMarkdown(tmp); err != nil {
		_ = tmp.Close()
		log.Fatalf("gendocs: write: %v", err)
	}
	if err := tmp.Close(); err != nil {
		log.Fatalf("gendocs: close temp: %v", err)
	}
	if err := os.Rename(tmpName, *out); err != nil {
		log.Fatalf("gendocs: rename %q → %q: %v", tmpName, *out, err)
	}
	fmt.Fprintf(os.Stderr, "gendocs: wrote %s\n", *out)
}
