// Command pixelforge-player is the universal Pixelforge runtime
// binary. It carries no game data of its own; instead, the studio's
// Build → Host pipeline appends a .pforge cart (and, eventually, the
// game's asset tree) to a copy of this binary via
// pixelforge_cart.Append. At startup the player reads its own
// appended cart, hands the bytes to pixelforge_project.LoadReader,
// boots the capsule runtime, and surrenders the main goroutine to
// pixelforge_ebiten.Run.
//
// No build tag: the cart-read path is build-tagged inside
// pixelforge_cart (selfread_native.go vs selfread_js.go), so this
// entry point compiles unchanged for both native and GOOS=js
// GOARCH=wasm targets — the same source produces the desktop
// pixelforge-player[.exe] and the WASM module that the build host
// inlines into the HTML shell.
//
// Error surface:
//   - ErrNoCart        → "this is the universal player; ship a game
//                        with pixelforge-studio Build → Host" + exit 1
//   - ErrCorruptCart   → diagnostic wrapping the corruption cause
//   - ErrVersionMismatch → diagnostic naming the observed magic
//   - any other error  → wrapped with the failing stage
//
// The exit-code split (0 vs 1) is intentional: ErrNoCart still exits
// non-zero because a player launched without a cart cannot do
// anything useful, but the message is friendlier than a crash so a
// user who double-clicks the wrong binary gets pointed at the
// Build → Host flow.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cart"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_render"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
)

// readCartFn returns the appended cart bytes from the running
// executable. Production binds this to pixelforge_cart.ReadSelf; the
// smoke test in main_test.go replaces it with a fixture-driven
// stub so we can exercise the diagnostic branches without rebuilding
// the binary for every scenario.
type readCartFn func() ([]byte, error)

// bootFn wires the loaded project + asset FS into the capsule
// runtime and then enters the Ebitengine game loop. Production binds
// this to the real Boot + Run sequence; tests substitute a stub that
// returns nil after asserting it received a non-nil Project, which
// lets us cover the cart-read → load → boot wiring without invoking
// the actual Ebitengine main loop (which would block on a window).
type bootFn func(p *pixelforge_project.Project, assets fs.FS) error

// defaultBoot is the production capsule-boot path: Boot the runtime
// against the embedded placeholder FS, install the arcade-shipping U4
// single-render-path seam (so every per-tick advance + draw cycle
// routes through pixelforge_render.RenderTickAtScreen — the same
// function the studio preview and the CI replay harness use), then
// surrender to pixelforge_ebiten.Run. Run blocks until the game
// window closes; the bootFn return is reached only on a graceful
// shutdown.
func defaultBoot(p *pixelforge_project.Project, assets fs.FS) error {
	rt, err := capsuleruntime.Boot(p, assets, capsuleruntime.Options{})
	if err != nil {
		return fmt.Errorf("boot runtime: %w", err)
	}

	// Install the U4 single-render-path seam. The closure captures
	// `rt` plus a scratch input-key slice (re-used per tick to avoid
	// allocation churn) and delegates the actual per-tick simulation
	// + draw work to pixelforge_render.RenderTickAtScreen. The
	// destination *ebiten.Image is the runtime's pixelforge canvas
	// surface, which pixelforge_ebiten owns; we draw into the
	// pixelforge.Screen()-backed canvas indirectly via the engine's
	// pixelforge.Draw path that RenderTickAtScreen invokes.
	//
	// Because the seam draws into the same pixelforge.Screen() the
	// EbitenGame.Draw call subsequently blits to the GPU surface,
	// the hook does not itself receive the GPU screen — it operates
	// on the existing CPU-side paletted canvas. This keeps the
	// player binary one-render-path-aligned without re-architecting
	// the canvas-to-GPU blit that pixelforge_ebiten already owns.
	var keysScratch []ebiten.Key
	pixelforge_ebiten.SetTickHook(func(tick uint64) {
		keysScratch = inpututil.AppendPressedKeys(keysScratch[:0])
		// Snapshot the keys into a fresh slice — RenderTickAtScreen
		// stashes the InputFrame on the seam's ActiveInputFrame
		// accessor (consumed by the U5 replay recorder when it
		// lands), so handing out a slice that gets clobbered next
		// tick would silently break the recorder.
		keys := make([]ebiten.Key, len(keysScratch))
		copy(keys, keysScratch)

		inputs := pixelforge_render.InputFrame{Keys: keys}
		if drawErr := pixelforge_render.RenderTickAtScreen(rt, playerOffscreenSurface(), tick, inputs); drawErr != nil {
			// RenderTickAtScreen does not return errors during the
			// happy path — only on nil-runtime / nil-screen. We
			// log and continue rather than crashing the player on
			// what would be a programmer error.
			log.Printf("pixelforge-player: render tick %d: %v", tick, drawErr)
		}
	})

	pixelforge_ebiten.Run()
	return nil
}

// playerOffscreenSurface returns the per-frame destination surface
// the TickHook hands to RenderTickAtScreen. RenderTickAtScreen
// performs the canvas-to-Ebiten-image palette blit as the last step
// of its render pipeline; the engine's own EbitenGame.Draw then
// scales + composites the engine's owned ebitenScreen onto the
// window. The two surfaces are different images today: the player's
// hook draws into this scratch surface, and EbitenGame.Draw blits
// pixelforge.Screen() (which the hook also populated via the engine's
// pixelforge.Draw path) onto the visible window.
//
// Sized to match pixelforge.Screen() so the WritePixels inside
// CopyCanvasToEbitenImage doesn't reject a dimension mismatch.
// Allocated lazily on the first hook call (when ebiten is fully
// initialised) and rebuilt when the project's screen size changes.
//
// This is the documented seam asymmetry: the player's TickHook runs
// the unified RenderTickAt simulation, but the on-screen composite
// is owned by pixelforge_ebiten. A future cleanup (post-U4) can
// thread the engine's ebitenScreen through the hook signature so
// RenderTickAtScreen writes straight into the on-window image and
// the scratch surface goes away. Keeping the seam stable was the
// U4 priority.
func playerOffscreenSurface() *ebiten.Image {
	scr := pixelforge.Screen()
	w, h := scr.W(), scr.H()
	if playerOffscreenCache == nil ||
		playerOffscreenCache.Bounds().Dx() != w ||
		playerOffscreenCache.Bounds().Dy() != h {
		playerOffscreenCache = ebiten.NewImage(w, h)
	}
	return playerOffscreenCache
}

var playerOffscreenCache *ebiten.Image

// smokeBoot is the test-mode bootFn the universal player uses when
// invoked with --smoke-tick=N. Plan-009 U12 introduced this seam so
// the build-pipeline integration smoke test can shell out to the
// freshly-built binary, verify the cart parses + the runtime boots
// cleanly, then exit 0 — without opening an Ebitengine window.
//
// The smoke path skips pixelforge_ebiten.Run entirely. We boot the
// capsule runtime, drive rt.AdvanceTick() N times, and return nil
// so run()'s caller can os.Exit(0). The smoke value is "did the
// cart load + the runtime boot cleanly + did N ticks advance
// without panicking?" — not "did Ebitengine render N frames?" For
// the latter (deferred to v3 wasmbrowsertest follow-up), see plan-
// 009's Scope Boundaries section.
//
// Sprites + Audio are zeroed before Boot. The U2/U3 universal player
// ships with a placeholder embed.FS that does NOT carry the cart's
// referenced sprite/audio assets — asset-blob-bundling lands in a
// later unit (see plan-009's asset-bundling follow-up + the doc
// comment in cmd/pixelforge-player/assets_placeholder.go). Until
// then the smoke seam strips asset references so capsuleruntime's
// LoadSprites/LoadAudio don't trip on missing files. The capsule's
// scene+entity+physics state still loads from the cart, which is
// what the smoke is verifying. Once cart-asset-bundling lands the
// strip becomes redundant (Boot reads from the in-cart FS and the
// references resolve); we can drop the assignment then without a
// behavioural change.
//
// SkipSubscribers stays false because the U7/U8/U9 sinks DO want to
// register against the global verb-recipe event bus during smoke —
// that's the cart's "boots cleanly" surface, and any sink wiring
// regression would surface as a panic here.
func smokeBoot(ticks uint64) bootFn {
	return func(p *pixelforge_project.Project, assets fs.FS) error {
		if p != nil {
			p.Sprites = nil
			p.Audio = nil
		}
		rt, err := capsuleruntime.Boot(p, assets, capsuleruntime.Options{})
		if err != nil {
			return fmt.Errorf("smoke-tick boot: %w", err)
		}
		for i := uint64(0); i < ticks; i++ {
			rt.AdvanceTick()
		}
		log.Printf("pixelforge-player: smoke-tick completed %d ticks; exiting 0", ticks)
		return nil
	}
}

func main() {
	// Plan-009 U12 added --smoke-tick=N as a CI test seam: when
	// non-zero, the player advances N capsule-runtime ticks and
	// exits 0 instead of surrendering to pixelforge_ebiten.Run.
	// This is the path the build-pipeline integration smoke test
	// shells out to after Build → Host on the Asteroids proof.
	//
	// flag.CommandLine.Parse on os.Args[1:] keeps the production
	// double-click path (no args) byte-equivalent: the flag's zero
	// value selects defaultBoot exactly as before U12.
	smokeTicks := flag.Uint64("smoke-tick", 0,
		"if > 0, advance N capsule-runtime ticks then exit 0 (CI smoke seam; does NOT open an ebiten window)")
	flag.Parse()

	boot := defaultBoot
	if *smokeTicks > 0 {
		boot = smokeBoot(*smokeTicks)
	}

	if err := run(pixelforge_cart.ReadSelf, boot); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

// run is the testable entry point: ReadSelf + LoadReader + Boot +
// Run, with the read and boot stages injectable so tests cover the
// diagnostic and happy-path wiring without touching the real
// /proc/self/exe or starting Ebitengine.
//
// On any error run returns a wrapped error suitable for the stderr
// banner main writes. The sentinel error chain stays intact
// (errors.Is recovers ErrNoCart / ErrCorruptCart / ErrVersionMismatch),
// so downstream tooling that wants to branch on the underlying cause
// still can.
func run(read readCartFn, boot bootFn) error {
	cartBytes, err := read()
	if err != nil {
		switch {
		case errors.Is(err, pixelforge_cart.ErrNoCart):
			return fmt.Errorf("pixelforge-player: no .pforge cart appended to this binary.\n"+
				"This is the universal player; ship a game with `pixelforge-studio Build → Host`. (%w)", err)
		case errors.Is(err, pixelforge_cart.ErrCorruptCart):
			return fmt.Errorf("pixelforge-player: appended cart is corrupt (CRC mismatch or truncated): %w", err)
		case errors.Is(err, pixelforge_cart.ErrVersionMismatch):
			return fmt.Errorf("pixelforge-player: cart version mismatch: %w", err)
		case errors.Is(err, pixelforge_cart.ErrPayloadTooLarge):
			return fmt.Errorf("pixelforge-player: cart payload exceeds size limit: %w", err)
		default:
			return fmt.Errorf("pixelforge-player: read cart: %w", err)
		}
	}

	p, err := pixelforge_project.LoadReader(bytes.NewReader(cartBytes))
	if err != nil {
		return fmt.Errorf("pixelforge-player: load project: %w", err)
	}

	if err := boot(p, embeddedAssets); err != nil {
		return fmt.Errorf("pixelforge-player: %w", err)
	}
	return nil
}
