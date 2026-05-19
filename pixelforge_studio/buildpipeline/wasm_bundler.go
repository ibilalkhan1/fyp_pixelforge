package buildpipeline

import (
	"compress/gzip"
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
	CartBase64    template.JS // base64-encoded cart payload; template.JS for same reason
	FaviconBase64 string

	// Credits is the CC-BY attribution data the splash's "View
	// Credits" button reveals. Empty slice suppresses the button
	// + credits div entirely — CC0-only projects don't carry an
	// attribution duty so we don't waste pixels.
	Credits []capsuleruntime.CreditEntry
}

// BundleWASM produces a single self-contained HTML file at outPath
// that carries the compiled WASM (base64-encoded), the cart payload
// (base64-encoded — the universal-player WASM reads it via the
// browser-side selfread shim that pulls window.__pixelforgeCart),
// the Go runtime glue (wasm_exec.js inline), and a click-to-start
// splash. The resulting file works offline because every dependency
// is embedded — no <script src=""> / <link rel="stylesheet">.
//
// Click-to-start satisfies modern browsers' autoplay policy
// (AudioContext can't start without a user gesture). Browsers
// across desktop + mobile honour the click handler the template
// installs on the splash overlay. The handler ALSO assigns
// window.__pixelforgeCart = Uint8Array.from(atob(CART_B64), ...)
// BEFORE invoking go.run(instance) so the player's selfread call
// sees the cart bytes immediately on startup.
//
// Failure modes: missing wasm file, missing wasm_exec.js, template
// execute errors. All return a wrapped error; outPath is not
// written on failure.
//
// BundleWASM is the backward-compatible entry point — it produces
// the .html but does NOT write a sibling .html.gz, does NOT
// measure size, and does NOT invoke wasm-opt. Callers that want
// those (the studio's wasmLongBuilder) use BundleWASMWithSize
// instead.
func BundleWASM(wasmPath, wasmExecJSPath string, cartBytes []byte, gameName, faviconBase64, outPath string, credits []capsuleruntime.CreditEntry) error {
	_, err := BundleWASMWithSize(wasmPath, wasmExecJSPath, cartBytes, gameName, faviconBase64, outPath, credits, BundleWASMOptions{})
	return err
}

// BundleWASMOptions carries the optional knobs BundleWASMWithSize
// exposes on top of BundleWASM's required arguments. Zero value
// matches the historical BundleWASM behaviour (no gzip sibling,
// no wasm-opt).
type BundleWASMOptions struct {
	// WriteGzipSibling, when true, emits a `<outPath>.gz`
	// gzip-compressed copy of the HTML alongside outPath. The
	// studio always sets this so the resulting size report
	// carries a real GzipBytes measurement.
	WriteGzipSibling bool

	// OptimizeWasm, when true AND IsWasmOptAvailable returns
	// true, runs `wasm-opt -Oz --enable-bulk-memory` on the wasm
	// bytes BEFORE base64-encoding them into the template. Silent
	// skip when wasm-opt is absent — caller doesn't have to
	// branch.
	OptimizeWasm bool
}

// BundleWASMWithSize is the size-aware bundler the studio's
// wasmLongBuilder calls. It does everything BundleWASM does plus:
//   - optionally runs wasm-opt on the wasm bytes (size-first
//     -Oz optimisation) before bundling, when opts.OptimizeWasm
//     is true and the binary is on PATH
//   - optionally writes a sibling `<outPath>.gz` gzip-compressed
//     copy of the HTML when opts.WriteGzipSibling is true
//   - stat's the output and returns a populated WASMSizeReport
//     the orchestrator can route into BuildStatus
//
// Returns a zero-value WASMSizeReport AND an error on any failure;
// outPath may be partially written.
func BundleWASMWithSize(wasmPath, wasmExecJSPath string, cartBytes []byte, gameName, faviconBase64, outPath string, credits []capsuleruntime.CreditEntry, opts BundleWASMOptions) (WASMSizeReport, error) {
	zero := WASMSizeReport{
		WarnThresholdMB:  WASMWarnThresholdMB,
		ErrorThresholdMB: WASMErrorThresholdMB,
	}
	if wasmPath == "" {
		return zero, errors.New("buildpipeline.BundleWASM: wasmPath is empty")
	}
	if wasmExecJSPath == "" {
		return zero, errors.New("buildpipeline.BundleWASM: wasmExecJSPath is empty")
	}
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return zero, fmt.Errorf("buildpipeline.BundleWASM: read wasm: %w", err)
	}
	if len(wasmBytes) == 0 {
		return zero, errors.New("buildpipeline.BundleWASM: wasm file is empty")
	}

	// Optional wasm-opt pass. If the binary's not on PATH we
	// silently skip — users without Binaryen still get a working
	// build. If it IS on PATH but the invocation fails we treat
	// it as a soft failure and fall through to the unoptimized
	// bytes, because shipping a working-but-larger artifact beats
	// blocking the build on a non-essential optimisation step.
	if opts.OptimizeWasm && IsWasmOptAvailable() {
		optimizedPath := wasmPath + ".opt"
		if optErr := OptimizeWasm(wasmPath, optimizedPath); optErr == nil {
			if optimizedBytes, readErr := os.ReadFile(optimizedPath); readErr == nil && len(optimizedBytes) > 0 {
				wasmBytes = optimizedBytes
			}
			_ = os.Remove(optimizedPath)
		}
	}

	wasmExecBytes, err := os.ReadFile(wasmExecJSPath)
	if err != nil {
		return zero, fmt.Errorf("buildpipeline.BundleWASM: read wasm_exec.js: %w", err)
	}
	data := wasmTemplateData{
		GameName:      gameName,
		WasmExecJS:    template.JS(string(wasmExecBytes)),
		WasmBase64:    template.JS(base64.StdEncoding.EncodeToString(wasmBytes)),
		CartBase64:    template.JS(base64.StdEncoding.EncodeToString(cartBytes)),
		FaviconBase64: faviconBase64,
		Credits:       credits,
	}
	var sb strings.Builder
	if err := wasmTemplate.Execute(&sb, data); err != nil {
		return zero, fmt.Errorf("buildpipeline.BundleWASM: template execute: %w", err)
	}
	htmlBytes := []byte(sb.String())
	if err := os.WriteFile(outPath, htmlBytes, 0o644); err != nil {
		return zero, fmt.Errorf("buildpipeline.BundleWASM: write %s: %w", outPath, err)
	}

	report := WASMSizeReport{
		RawBytes:         int64(len(htmlBytes)),
		WarnThresholdMB:  WASMWarnThresholdMB,
		ErrorThresholdMB: WASMErrorThresholdMB,
	}
	if opts.WriteGzipSibling {
		gzipSize, gzipErr := writeGzipSibling(outPath, htmlBytes)
		if gzipErr != nil {
			return report, fmt.Errorf("buildpipeline.BundleWASM: write gzip sibling: %w", gzipErr)
		}
		report.GzipBytes = gzipSize
	}
	return report, nil
}

// writeGzipSibling writes `<htmlPath>.gz` and returns its on-disk
// size. Uses the default compression level — gzip's --best (level
// 9) saves a few percent on these large mostly-base64 payloads at
// roughly 4× the CPU cost; not worth it for the per-build path.
func writeGzipSibling(htmlPath string, htmlBytes []byte) (int64, error) {
	gzPath := htmlPath + ".gz"
	f, err := os.Create(gzPath)
	if err != nil {
		return 0, fmt.Errorf("create %s: %w", gzPath, err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	if _, err := gz.Write(htmlBytes); err != nil {
		return 0, fmt.Errorf("gzip write: %w", err)
	}
	if err := gz.Close(); err != nil {
		return 0, fmt.Errorf("gzip close: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		return 0, fmt.Errorf("stat %s: %w", gzPath, err)
	}
	return info.Size(), nil
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
