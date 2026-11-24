# Native runtime, audio and browser examples — acceptance scope

The work is on the existing `codex/fyp2-demo-readiness` branch. The contractor download and `main` are preserved. Changes were implemented with AI assistance and must not be represented as three teammates independently authoring these commits.

## Implemented changes

- Native Play uses a detached project copy, embedded project assets, decoded sprites and a shared render path with the universal player and generated source exports. Stop restores the authored project and closes owned subscriptions.
- Authored event-sheet movement now resolves mutable entity positions. Loop events, semantic input events and keyboard snapshots reach behaviours. Runtime housekeeping advances once per simulated tick.
- Studio services its audio backend around every update, including modal updates. WAV imports accept common integer/float PCM and downmix to the existing signed 8-bit mono mixer; MP3 and OGG/Vorbis use real decoders. Compressed assets travel in exported carts. Header-only invalid files fail. Limits remain 64 MiB input and bounded decoded samples. This is decoded in memory, not streaming audio.
- Snake and Pac-Man browser examples embed their assets and offer Start, Restart, sound toggle and held touch controls. Pac-Man spawn, initial movement and tunnel-row bugs were reproduced and corrected. They are separate examples, not editable creator templates.

## Evidence and limits

The full default Mac test suite passed across 63 packages. The long suite passed 85 checks and skipped six. The shared rendering check verifies real sprite pixels, input-driven movement, key release and one housekeeping advance per tick. The no-new-issues lint check compares against the supplied-source baseline; this is not a claim that inherited lint debt is zero. Browser Snake movement, scoring, restart and sound toggle were exercised. Browser Pac-Man moved and scored 70 with three lives. Native Studio Play was launched on the Mac, and a held Right key moved the authored character. F5 Play, held-arrow movement and F6 Stop were verified in the running Studio; Stop restored the authored scene. Generated native source compiled and its independent app launched and responded to arrow input. The public Pac-Man example reached score 150 with three lives, then Restart reset the score to zero. Latest CI links are recorded in the release handoff.

The public link is a bounded browser game creator: two editable templates plus a blank starting point, tile and sprite editing, two objective types, keyboard/touch play, one local save slot, JSON exchange and standalone HTML download. It does not provide every native Studio workspace, open native `.pforge` projects, or publish visitors’ exports automatically.

## Remaining acceptance before claiming the entire FYP scope complete

- Play through complete authored reference games; the four reference proof fixtures are subsystem evidence and must not be described as complete finished games.
- Verify exported games by launching them on Windows and Linux hardware. CI/build success alone is insufficient.
- Verify complex native scene switching, behaviour combinations, controller input, visual effects/dialogue overlays, and long sessions. The new shared path does not by itself establish full feature parity.
- Listen on the presentation laptop/speakers to confirm audible output and its volume; decoding and mixer command tests cannot establish what the room hears.
- Rehearse the offline HTML by manually opening it on the actual presentation laptop, and test the public HTTPS link from an external device/network.

Do not label the original FYP requirements 100% complete solely because the presentation demonstration works.
