// Package buildpipeline is idea #7 v1's ship-loop infrastructure.
//
// Three concerns:
//
//   - toolchain.go — locates the Go toolchain (vendored SDK at
//     <execDir>/go-sdk/bin/go, with PATH fallback), composes
//     cross-compile env vars (GOOS/GOARCH/CGO_ENABLED), and gives
//     callers an exec.Cmd they can run.
//   - orchestrator.go — Build(req, targets) <-chan BuildStatus
//     dispatches per-target builders in parallel goroutines and
//     streams status events to a single channel for the Build
//     workspace UI to consume.
//   - per-target builders (builders/) — windows/macos/linux/wasm/
//     source. Each generates the Capsule, runs go build (where
//     applicable), and packages the artifact under
//     <project-dir>/exports/<target>/.
//
// The package's design contract: every concrete build action
// (codegen, compile, package) is reachable from one entry point so
// the Build workspace UI and the build-on-save daemon share the
// same code path. The shipped binary IS the editor preview's
// Capsule, just compiled — there's no separate runtime.
package buildpipeline
