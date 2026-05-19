// Package pixelforge_replay implements the deterministic replay
// harness that powers Pixelforge's pixel-hash CI strategy. Two
// surfaces:
//
//   - Trace + (*Trace).Encode + LoadTrace — the .trace.jsonl format
//     used to persist a recorded input session to disk.
//   - Recorder + Replayer — capture per-tick inputs while the studio
//     preview (or any host that drives pixelforge_render) runs, then
//     deterministically replay them against a freshly-booted
//     *capsuleruntime.Runtime to produce a sequence of *image.RGBA
//     frames + the captured verbs.bus event stream.
//
// The package is the canonical consumer of pixelforge_render's
// RenderTickAtRGBA seam: per the parity contract documented in
// pixelforge_render/doc.go, given equal runtimes + equal traces, the
// replayer is guaranteed to reproduce the recorded session byte-for-
// byte on the same CPU architecture.
//
// # .trace.jsonl schema
//
// A trace is a UTF-8 text file. The first line is the meta header:
//
//	{"v":1,"meta":{"game":"asteroids_proof","seed":42,"width":320,"height":180,"tps":60,"duration_ticks":5400}}
//
// All subsequent lines are per-tick input frames:
//
//	{"tick":0,"keys":[],"pad":null}
//	{"tick":1,"keys":[],"pad":null}
//	...
//	{"tick":47,"keys":["Space"],"pad":null}
//	{"tick":48,"keys":["Space","ArrowLeft"],"pad":null}
//
// Key names follow ebiten.Key.String() — "Space", "ArrowLeft",
// "ArrowRight", "ArrowUp", "ArrowDown", letter keys as their bare
// letter ("A", "D", "W", "S"), plus "Escape" and "Enter". Aliases
// such as "KeyA" / "KeyD" / "KeyW" / "KeyS" are also accepted on
// decode for compatibility with hand-authored traces.
//
// Unknown key names are dropped with a log warning on decode (so a
// future ebiten version's new key constants do not crash an older
// replayer). Unknown gamepad-button indices are similarly tolerant.
//
// ## Run-length compression
//
// A compressed frame line carries a "hold" field:
//
//	{"tick":47,"keys":["Space"],"pad":null,"hold":12}
//
// This expands at decode time to 12 consecutive logical frames
// covering ticks 47..58 inclusive. The decoder always accepts both
// uncompressed and compressed forms; the encoder defaults to
// uncompressed at v1 (file size has not yet been measured against
// hard caps — compression can land as a follow-up if 5400-frame
// traces grow beyond comfort).
//
// # Determinism boundaries
//
// What the replay harness DOES guarantee:
//
//   - Same trace + same booted runtime → byte-equal *image.RGBA
//     sequence across N invocations on the SAME CPU architecture.
//   - Same trace + same booted runtime → identical verbs.bus event
//     sequence (Topic + Args, in publication order) across N
//     invocations.
//   - No goroutines are spawned by (*Replayer).Run; iteration is
//     strictly single-threaded.
//   - No time.Now() reads from the harness itself — all timing comes
//     from the recorded tick numbers.
//
// What the replay harness DOES NOT promise:
//
//   - **Cross-CPU determinism** (amd64 vs arm64). This is a property
//     of the engine layers underneath (pixelforge_physics in
//     particular), measured by the U6 probe. The replayer is a thin
//     iterator over RenderTickAtRGBA — whatever determinism contract
//     the renderer + physics provide is what the replayer surfaces.
//
//   - **Audio sample-accurate output.** Audio sinks are typically
//     stubbed for replay (via capsuleruntime.Options.SinksOverride).
//     The replayer itself does not override them; the caller passes
//     a runtime booted with whatever Sinks they need. Stubbed audio
//     sinks are the conventional choice for CI replay so the test
//     does not depend on an audio device being present.
//
//   - **Reproduction of the recorder's wall-clock pacing.** A
//     recording captured at 60 TPS is replayed as a tight loop, not
//     at 60 TPS; ticks advance as fast as the renderer can complete
//     them. For headless CI this is intentional (faster than real-
//     time replay). Visual playback should sit on top of an outer
//     ticker.
//
// # Usage sketch
//
// Record (during studio preview):
//
//	rec := pixelforge_replay.NewRecorder(pixelforge_replay.TraceMeta{
//	    Game: "asteroids_proof", Width: 320, Height: 180, TPS: 60,
//	})
//	// each frame:
//	rec.Tick(pixelforge_render.InputFrame{Keys: heldKeys, Pad: padState})
//	// when done:
//	f, _ := os.Create("session.trace.jsonl")
//	_ = rec.Flush(f)
//
// Replay (CI):
//
//	f, _ := os.Open("session.trace.jsonl")
//	trace, _ := pixelforge_replay.LoadTrace(f)
//	rt, _ := capsuleruntime.Boot(project, assets, capsuleruntime.Options{
//	    SinksOverride: capsuleruntime.Sinks{Audio: &silentAudioSink{}},
//	})
//	r := pixelforge_replay.NewReplayer()
//	frames, events, err := r.Run(rt, trace)
//
// Each frame in frames is a fresh *image.RGBA the caller owns; each
// event in events is a *piloop.VerbEvent captured in publication
// order with a tick stamp.
package pixelforge_replay
