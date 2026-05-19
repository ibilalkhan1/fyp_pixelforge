package buildpipeline

import (
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
)

//go:embed wasm_template.html
var wasmTemplateSource string

// wasmTemplate is the compiled HTML template the bundler renders.
// Parsed once at package init; reused for every BundleWASM call.
var wasmTemplate = template.Must(template.New("wasm").Parse(wasmTemplateSource))

// wasmTemplateData is what BundleWASM feeds into the HTML template.
type wasmTemplateData struct {
	GameName      string
	WasmExecJS    template.JS // raw JS inserted into <script> body
	WasmBase64    template.JS // base64-encoded wasm; template.JS so quotes don't get escaped
	FaviconBase64 string

	// Credits is the CC-BY attribution data the splash's "View
	// Credits" button reveals. Empty slice suppresses the button
	// + credits div entirely — CC0-only projects don't carry an
	// attribution duty so we don't waste pixels.
	Credits []capsuleruntime.CreditEntry
}

// BundleWASM produces a single self-contained HTML file at outPath
// that carries the compiled WASM (base64-encoded), the Go runtime
// glue (wasm_exec.js inline), and a click-to-start splash. The
// resulting file works offline because every dependency is
// embedded — no <script src=""> / <link rel="stylesheet">.
//
// Click-to-start satisfies modern browsers' autoplay policy
// (AudioContext can't start without a user gesture). Browsers
// across desktop + mobile honour the click handler the template
// installs on the splash overlay.
//
// Failure modes: missing wasm file, missing wasm_exec.js, template
// execute errors. All return a wrapped error; outPath is not
// written on failure.
func BundleWASM(wasmPath, wasmExecJSPath, gameName, faviconBase64, outPath string, credits []capsuleruntime.CreditEntry) error {
	if wasmPath == "" {
		return errors.New("buildpipeline.BundleWASM: wasmPath is empty")
	}
	if wasmExecJSPath == "" {
		return errors.New("buildpipeline.BundleWASM: wasmExecJSPath is empty")
	}
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return fmt.Errorf("buildpipeline.BundleWASM: read wasm: %w", err)
	}
	if len(wasmBytes) == 0 {
		return errors.New("buildpipeline.BundleWASM: wasm file is empty")
	}
	wasmExecBytes, err := os.ReadFile(wasmExecJSPath)
	if err != nil {
		return fmt.Errorf("buildpipeline.BundleWASM: read wasm_exec.js: %w", err)
	}
	data := wasmTemplateData{
		GameName:      gameName,
		WasmExecJS:    template.JS(string(wasmExecBytes)),
		WasmBase64:    template.JS(base64.StdEncoding.EncodeToString(wasmBytes)),
		FaviconBase64: faviconBase64,
		Credits:       credits,
	}
	var sb strings.Builder
	if err := wasmTemplate.Execute(&sb, data); err != nil {
		return fmt.Errorf("buildpipeline.BundleWASM: template execute: %w", err)
	}
	if err := os.WriteFile(outPath, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("buildpipeline.BundleWASM: write %s: %w", outPath, err)
	}
	return nil
}

// EstimateWASMHTMLSize predicts the bundler's output size for the
// supplied wasm byte count. Used by the Build workspace to warn
// designers when their WASM ships above the practical 30MB UX
// threshold. Base64 inflates by ~4/3; the template + wasm_exec.js
// add a ~30KB overhead.
func EstimateWASMHTMLSize(wasmBytes int64) int64 {
	const templateAndExecOverhead int64 = 35 * 1024 // wasm_exec.js + template chrome
	return wasmBytes*4/3 + templateAndExecOverhead
}
