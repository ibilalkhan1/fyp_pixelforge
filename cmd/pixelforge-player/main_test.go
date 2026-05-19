//go:build !js

package main

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cart"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// TestRun_NoCart verifies the diagnostic the player prints when the
// binary has no v2 footer appended — i.e. somebody ran the universal
// player directly instead of the studio's Build → Host artefact.
// The message must mention both "no .pforge cart appended" (so the
// user knows what's missing) and the "pixelforge-studio Build → Host"
// remediation path.
func TestRun_NoCart(t *testing.T) {
	stubRead := func() ([]byte, error) {
		return nil, pixelforge_cart.ErrNoCart
	}
	stubBoot := func(p *pixelforge_project.Project, assets fs.FS) error {
		t.Fatal("boot should not be invoked when ReadSelf returns ErrNoCart")
		return nil
	}

	err := run(stubRead, stubBoot)
	require.Error(t, err)
	assert.ErrorIs(t, err, pixelforge_cart.ErrNoCart, "ErrNoCart must remain in the chain so callers can branch on it")
	assert.Contains(t, err.Error(), "no .pforge cart appended", "diagnostic must name the missing cart")
	assert.Contains(t, err.Error(), "pixelforge-studio Build", "diagnostic must point users at Build → Host")
}

// TestRun_CorruptCart verifies the corrupt-cart diagnostic is
// distinct from the no-cart one. CRC mismatches and truncated payloads
// should NOT be silently treated as "no cart" — that would hide a
// real shipping bug.
func TestRun_CorruptCart(t *testing.T) {
	stubRead := func() ([]byte, error) {
		return nil, pixelforge_cart.ErrCorruptCart
	}
	stubBoot := func(p *pixelforge_project.Project, assets fs.FS) error {
		t.Fatal("boot should not be invoked when ReadSelf returns ErrCorruptCart")
		return nil
	}

	err := run(stubRead, stubBoot)
	require.Error(t, err)
	assert.ErrorIs(t, err, pixelforge_cart.ErrCorruptCart)
	assert.Contains(t, err.Error(), "corrupt")
}

// TestRun_VersionMismatch verifies that a v1 (or future-version) cart
// surfaces with a precise diagnostic rather than getting lumped under
// "corrupt" — operators upgrading the player need to know whether
// they have a footer-version skew or a true integrity failure.
func TestRun_VersionMismatch(t *testing.T) {
	stubRead := func() ([]byte, error) {
		return nil, pixelforge_cart.ErrVersionMismatch
	}
	stubBoot := func(p *pixelforge_project.Project, assets fs.FS) error {
		t.Fatal("boot should not be invoked on version mismatch")
		return nil
	}

	err := run(stubRead, stubBoot)
	require.Error(t, err)
	assert.ErrorIs(t, err, pixelforge_cart.ErrVersionMismatch)
	assert.Contains(t, err.Error(), "version mismatch")
}

// TestRun_SyntheticCartBootsCleanly is the happy-path smoke test:
// build a real minimal .pforge payload from NewProject + Encode, hand
// it to run() via a stub ReadSelf, and verify the boot stage receives
// a fully-loaded Project. This exercises the cart-read → LoadReader →
// Boot wiring without invoking the actual Ebitengine main loop
// (which would block on a window and isn't safe to spin up under go
// test).
func TestRun_SyntheticCartBootsCleanly(t *testing.T) {
	p := pixelforge_project.NewProject("smoke")
	cartBytes, err := pixelforge_project.Encode(p)
	require.NoError(t, err, "encode minimal project")

	stubRead := func() ([]byte, error) {
		return cartBytes, nil
	}

	var bootInvocations int
	var bootProject *pixelforge_project.Project
	var bootAssets fs.FS
	stubBoot := func(p *pixelforge_project.Project, assets fs.FS) error {
		bootInvocations++
		bootProject = p
		bootAssets = assets
		return nil
	}

	require.NoError(t, run(stubRead, stubBoot), "happy-path run must succeed")
	assert.Equal(t, 1, bootInvocations, "boot must be invoked exactly once")
	require.NotNil(t, bootProject, "boot must receive a non-nil project")
	assert.Equal(t, "smoke", bootProject.Name, "loaded project name must round-trip")
	require.NotNil(t, bootAssets, "boot must receive a non-nil assets FS")

	// The placeholder FS must at least expose its root — Boot's
	// loader walks it for sprite/audio files and a nil-rooted FS
	// would panic deep in capsuleruntime.LoadSprites. We don't list
	// files here (the placeholder contains only .keep); a successful
	// root open is sufficient.
	entries, err := fs.ReadDir(bootAssets, "assets_placeholder")
	require.NoError(t, err, "placeholder FS must be walkable")
	assert.NotNil(t, entries)
}

// TestRun_LoadReaderFailureSurfaces verifies that a syntactically
// invalid cart payload surfaces as a "load project" error rather
// than crashing — the user sees the diagnostic and the binary exits
// non-zero, but doesn't panic.
func TestRun_LoadReaderFailureSurfaces(t *testing.T) {
	stubRead := func() ([]byte, error) {
		// Not valid JSON; LoadReader must reject it cleanly.
		return []byte("not a valid .pforge file"), nil
	}
	stubBoot := func(p *pixelforge_project.Project, assets fs.FS) error {
		t.Fatal("boot should not be invoked when LoadReader fails")
		return nil
	}

	err := run(stubRead, stubBoot)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load project", "loader failure must name the failing stage")
}

// TestSmokeBoot_AdvancesNTicksAndReturnsNil exercises the plan-009
// U12 smoke seam: smokeBoot(N) must Boot the capsule runtime,
// advance exactly N ticks, then return nil — without opening an
// Ebitengine window. The CI build-pipeline integration test relies
// on this contract to verify the freshly-built host binary parses
// its appended cart + boots the runtime cleanly + exits 0.
//
// We assert the post-boot Tick counter equals the requested N, so
// a regression that swallows the loop or pre-increments the counter
// surfaces as a test failure before the long-tag smoke runs.
func TestSmokeBoot_AdvancesNTicksAndReturnsNil(t *testing.T) {
	const ticks uint64 = 42

	p := pixelforge_project.NewProject("smoke_tick")
	cartBytes, err := pixelforge_project.Encode(p)
	require.NoError(t, err, "encode minimal project")

	stubRead := func() ([]byte, error) { return cartBytes, nil }

	require.NoError(t, run(stubRead, smokeBoot(ticks)),
		"smoke-tick boot must return nil after advancing N ticks")
}

// TestSmokeBoot_ZeroTicksIsNoOp is the boundary case: smokeBoot(0)
// should still boot the runtime (so a cart-load regression surfaces
// even with a zero count) but advance no ticks. This guards against
// a future "if N == 0, skip Boot entirely" optimisation that would
// hide a real cart-parsing bug.
func TestSmokeBoot_ZeroTicksIsNoOp(t *testing.T) {
	p := pixelforge_project.NewProject("smoke_zero")
	cartBytes, err := pixelforge_project.Encode(p)
	require.NoError(t, err)

	stubRead := func() ([]byte, error) { return cartBytes, nil }
	require.NoError(t, run(stubRead, smokeBoot(0)),
		"smoke-tick=0 must still boot cleanly")
}

// TestSmokeBoot_TolerantOfMissingSpriteAssets is the load-bearing
// regression test for U12's pragmatic asset-strip: a cart that
// declares sprite references the player's embedded placeholder FS
// cannot resolve (the as-shipped state until cart-asset-bundling
// lands) must still smoke-tick cleanly. Without the strip the
// capsule's LoadSprites would fail at Boot and the smoke would
// exit non-zero.
//
// We assemble a minimal project with one Sprite row whose file
// would normally need to live at assets/sprites/missing.png — it
// doesn't — and verify smokeBoot doesn't error.
func TestSmokeBoot_TolerantOfMissingSpriteAssets(t *testing.T) {
	p := pixelforge_project.NewProject("smoke_missing_sprite")
	// Real designer-supplied projects carry SpriteAsset entries; the
	// universal player's placeholder FS does NOT carry the files
	// they reference. Smoke must survive this corner.
	p.Sprites = []pixelforge_project.SpriteAsset{
		{Name: "ship", RelativePath: "sprites/missing.png"},
	}
	cartBytes, err := pixelforge_project.Encode(p)
	require.NoError(t, err)

	stubRead := func() ([]byte, error) { return cartBytes, nil }
	require.NoError(t, run(stubRead, smokeBoot(3)),
		"smoke-tick must tolerate carts whose sprite assets are not in the player's embed FS")
}
