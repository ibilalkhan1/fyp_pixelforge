---
date: 2026-05-18
topic: audio-library-picker-v1
origin: docs/ideation/2026-05-18-pixelforge-nes-class-no-code-ideation.md (idea #4)
---

# Audio Library Picker + Bindings Table — v1

## Summary

v1 ships a curated audio library inside the studio: ~30 NES-authentic SFX WAVs + 4-5 BGM loops, browsable via a categorized picker with audition. A separate bindings table lets the designer drag a patch onto an event topic to wire it up. No parametric synthesis, no mixer-lane visualization, no advanced composer — patches are pre-rendered WAVs, the engine's existing Paula auto-allocator handles channel routing, and no schema changes are required.

---

## Problem Frame

Pixelforge's engine has a working 4-channel Paula-style audio mixer (`pixelforge_audio`), a project schema with `AudioSample` (imported WAV files with priority/loop hints) and `AudioBinding` (event topic → sample with optional ForceChannel + TriggerCondition), and a `Play` step in the scripting catalog that routes through them. None of it is exposed in the studio:

- The asset browser lists imported audio files but offers no audition, no categorization, no preview.
- No UI walks the `AudioBinding` schema — designers who want a sound to play when an event fires must edit `.pforge` JSON by hand.
- No bundled patch library exists; designers either import their own WAVs or ship games with no sound at all.

This is the largest user-visible gap in the studio. The cross-domain research from the NES-class ideation surfaced a consistent pattern: every no-code game editor's designers stall on audio. FamiStudio / Bosca Ceoil / Beepbox / sfxr / Bfxr / ChipTone all converged on three-panel preset-driven workflows because composing chiptune from scratch is a discipline designers don't have time to learn while also designing a game.

Pixelforge's twist: skip the synthesis entirely for v1. Ship a curated library of pre-rendered NES-authentic WAVs the designer picks from like a sound-effects asset library. Composition lands later (or never — most arcade games need a small handful of distinctive SFX + a loop, all coverable from a 30-patch library). The bet: designers ship games with appropriate audio in their first session without ever opening a synth.

---

## Actors

- **A1. Designer.** Picks audio patches from the library, auditions them, drags them onto event topics. Not pre-trained on trackers, DAWs, or chiptune synthesis. Knows what "the sound when Mario jumps" should feel like; doesn't know what an envelope or a duty cycle is.
- **A2. Pixelforge Studio.** Hosts the new dockable Audio workspace (library picker + bindings table); plays auditions through the engine's Paula mixer; persists bindings to the project file.
- **A3. Pixelforge engine audio pipeline.** `pixelforge_audio` package + the existing Paula auto-allocator that routes samples to channels at runtime based on `SuggestedChannelPriority`.

---

## Key Flows

- **F1. Designer auditions and binds a jump sound**
  - **Trigger:** Designer wants the player to make a sound when jumping
  - **Actors:** A1, A2, A3
  - **Steps:** (1) Designer opens the Audio workspace; (2) filters the library to "jump" category or scrolls; (3) clicks the play icon on three jump variants to audition; (4) drags the chosen patch from the library picker onto the "PlayerJumped" event topic in the bindings table; (5) saves the project
  - **Outcome:** When the PlayerJumped event fires at runtime, the bound sample plays through the Paula mixer.
  - **Covered by:** R1, R2, R3, R4, R5

- **F2. Designer picks a BGM loop for the title screen**
  - **Trigger:** Designer wants background music on the title screen
  - **Actors:** A1, A2, A3
  - **Steps:** (1) Designer filters the library to "BGM" category; (2) auditions 4-5 BGM loops (each loops automatically during audition so the designer can hear the loop point); (3) drags the chosen loop onto the "SceneEntered:title" event topic in the bindings table; (4) saves
  - **Outcome:** When the title scene is entered, the chosen BGM loop plays on a channel the auto-allocator dedicates to it (one of channels 1-2 per `SuggestedChannelPriority`).
  - **Covered by:** R1, R2, R3, R6

- **F3. Designer ships a game with library-sourced audio to a classmate**
  - **Trigger:** Designer has bound several library patches and clicks the build action (full ship UX defined under idea #7)
  - **Actors:** A1, A2, A3
  - **Steps:** (1) Studio copies every library WAV that's referenced by a binding into the project's audio-assets/ directory; (2) build pipeline embeds those WAVs in the shipped binary; (3) classmate runs the binary on the same OS
  - **Outcome:** The shipped game plays audio identically to the studio preview, without any runtime dependency on the studio's bundled library.
  - **Covered by:** R7, R8

- **F4. Designer also imports a custom WAV they made themselves**
  - **Trigger:** Designer has a specific custom sound they recorded or generated externally
  - **Actors:** A1, A2
  - **Steps:** (1) Designer uses the existing AudioSample import path (file picker / drag-drop into asset browser); (2) WAV becomes a regular AudioSample alongside library patches; (3) appears in the bindings table's sound picker on equal footing with library patches
  - **Outcome:** User-imported and library WAVs are first-class peers in the bindings UI.
  - **Covered by:** R9

---

## Requirements

**Library (curated bundle)**

- R1. The studio ships with a **bundled audio library** of approximately 30 SFX WAVs + 4-5 BGM loops. Patches are pre-rendered WAVs, not parametric recipes. All are NES-authentic — synthesized against the same Paula 4-channel mixer the engine ships, mono, sample-rate within Paula's range.
- R2. Library patches are organized by **role-based categories**: SFX categories include `jump`, `shoot`, `hit`, `pickup`, `coin`, `menu-confirm`, `win-jingle`, `lose-stinger`, `damage`, `death`, `ambient`. BGM categories include `town`, `dungeon`, `boss`, `title`, `victory`. The library picker exposes a category filter; same picker holds both SFX and BGM (BGM patches show a loop icon).

**Audio workspace surface**

- R3. A new **dockable Audio workspace** appears alongside the existing Scene / Inspector / Assets / Capture / Behavior / Palette workspaces. Same dockable-panel + ImGui pattern as the rest; persists in `imgui.ini` like other docks.
- R4. The Audio workspace contains **two panels**: a **library picker** on the left and a **bindings table** on the right. **No mixer-lane panel in v1.**
- R5. The library picker shows patches as rows with: name, category, duration, loop indicator (for BGM), and an inline **play button** that auditions the patch through the engine's Paula mixer. Click-to-start, click-again-to-stop. SFX play once; BGM loops automatically during audition so the designer can hear the loop point.

**Bindings table**

- R6. The bindings table shows existing `AudioBinding` rows: event topic, bound sample name, optional `SceneID` filter, optional `TriggerCondition`. Designer adds rows by **dragging a patch from the library picker onto an event topic** (or onto an "add binding" cell), or via a per-row "Pick sound..." button that opens a sound-picker overlay listing both library patches and user-imported AudioSamples.
- R7. When a binding is created with a library patch, the studio **copies the library WAV from its bundled location into the project's audio-assets/ directory** as a regular `AudioSample` entry. The binding references that AudioSample by name — identical shape to a user-imported binding. The shipped game has zero dependency on the studio's bundle.

**Shipping integration**

- R8. The bet (designer ships a game with sound to a classmate) requires the build pipeline (idea #7) to include the project's audio-assets/ directory in the shipped binary. v1 of this brainstorm only requires that *some* build pipeline exists (current-platform native binary is the minimum, per idea #1's R12 dependency on idea #7).

**User-imported audio coexistence**

- R9. User-imported AudioSamples (via the existing import path — file picker / drag-drop into asset browser) appear in the bindings table's sound picker on **equal footing** with library patches. There is no UI distinction between "library sound" and "user sound" once both are in the project's audio-assets/ directory.

**Channel allocation (existing engine behavior)**

- R10. v1 honors the engine's existing **Paula auto-allocator** using `AudioSample.SuggestedChannelPriority` to route playback (BGM locks to channels 1-2; SFX round-robin on 3-4). v1 **does not surface** the `AudioBinding.ForceChannel` override in the UI; the schema field remains for power users who edit `.pforge` directly and for the v2 mixer-lane panel.

---

## Acceptance Examples

- **AE1. Covers R1, R5.** Given the designer opens the Audio workspace, when they click the play button next to a patch named "shoot/laser_small", the studio plays the WAV once through the engine's Paula mixer. Clicking play again on the same patch stops it.
- **AE2. Covers R2.** Given the designer types "BGM" into the library category filter, when the picker updates, only the 4-5 BGM loop patches appear, each showing a loop icon.
- **AE3. Covers R6, R7.** Given the designer drags the "jump/spring" patch from the library picker onto the "PlayerJumped" event-topic cell in the bindings table, when the drop completes: (a) a new AudioSample appears in the project named "jump_spring" (or similar) with `RelativePath` pointing into `audio-assets/`; (b) the bindings table shows a new row mapping "PlayerJumped" → "jump_spring"; (c) the WAV is copied from the studio's bundle into the project's audio-assets/ directory on disk.
- **AE4. Covers R5.** Given the designer auditions a BGM loop named "town/peaceful_loop", when the WAV plays, it loops automatically (so the designer hears the loop transition). Clicking play again stops it.
- **AE5. Covers R9.** Given the designer has both library-sourced and user-imported AudioSamples in the project, when they open the "Pick sound..." overlay on a binding row, both kinds appear in the same scrollable list with no visual distinction between origins.
- **AE6. Covers R8.** Given the designer has bound three library patches in a project, when the build pipeline (idea #7) runs and produces a single-file binary, the binary is self-contained: running it on a clean machine with no Pixelforge installed plays the bound audio identically to the studio preview.

---

## Success Criteria

- **Designer outcome:** A first-time designer with no prior audio-authoring experience opens the Audio workspace, picks a jump SFX in under a minute, binds it, hears it play when the player jumps in the live preview, and moves on to picking a coin SFX. Through the whole session they never see the word "channel," "envelope," "synthesis," or "tracker."
- **Library-coverage outcome:** The bundled 30 SFX + 4-5 BGM loops cover the common events of every game in the NES-class reference set (Mario, Zelda, Metroid, Megaman, Tetris, Final Fantasy, Bubble Bobble, Punch-Out, Excitebike, Double Dragon) at least roughly. A designer building any one of those classes doesn't need to import a single custom sound for the first prototype.
- **Ship-loop outcome:** A game built with library-sourced audio ships as a self-contained binary (per idea #7) — no studio dependency at runtime — and plays audio identically to the studio preview.
- **Downstream handoff outcome:** Planning consumes this doc and does not need to invent library taxonomy, audition behavior, binding UX, or asset-copy semantics. Only implementation specifics (exact category list, exact patch list, exact ImGui drag-drop API, exact path inside audio-assets/) are open for planning.

---

## Scope Boundaries

- **Mixer-lane panel** (live 4-channel Paula visualization with flash-on-voice-stealing) — deferred to v2; debugging surface, not core authoring.
- **Parametric synthesis** (recipe + sliders that synthesize a WAV at edit-time or runtime). Patches are static WAVs in v1.
- **Sheet-music notation advanced editor** (4 voices = 4 staves for composing custom music). Out — composing-from-scratch is what library-first explicitly sidesteps.
- **MIDI / tracker editor.** Out — same rationale.
- **Audio effects chain** (reverb, delay, low-pass, sidechain). Out — Paula didn't have any; adding them undermines the NES-class identity.
- **Designer-built patches.** v1 designers cannot create new library patches; the library is bundle-only. User-imported WAVs still work via the existing import path; they just don't become "library" entries.
- **Community patch gallery** (download patches from the internet). Out of product identity, not just v1.
- **Adaptive / layered BGM** (music intensity rises with combat). Out — would require runtime synthesis or layered mixing infrastructure that doesn't exist.
- **Real-time waveform scrub** (drag through a sample to find a loop point). Out — defers to a future power-user feature.
- **Audio quantizer / format converter** (auto-convert non-Paula WAVs to Paula-compatible). The existing `pixelforge_audio.DecodeWavOrErr` validates format and surfaces errors; that's enough for v1.
- **Per-binding `ForceChannel` UI.** The schema supports it; v1 does not surface it. Designers who need override edit `.pforge` directly or wait for the v2 mixer-lane panel.
- **Mic-capture authoring mode** (hum a melody, system synthesizes). Out — Ebitengine has no mic API; would require a separate audio-input dependency.
- **Pitch / speed / volume modifiers on bound patches.** v1 binds a patch verbatim; if the designer wants the same sound at half speed, they bind a different patch (or import their own variant). Per-binding modifiers are a v2 feature.

---

## Key Decisions

- **Pre-rendered WAVs over parametric recipes.** Simplest path. No synthesis engine needed in the studio or the runtime; no "Bake" UX; no recipe-tweaking loop. Trades flexibility (designers can't tweak patches) for shipping certainty.
- **Library + bindings table, defer mixer lane.** Two panels cover the authoring story end-to-end. Mixer-lane is debugging UI that only matters once designers hit channel-stealing bugs — defer until they do.
- **Category by role, not by instrument family.** Designer thinks "I need a jump sound," not "I need a square-wave with fast decay." The taxonomy matches the audience's mental model.
- **BGM and SFX in the same picker with a filter.** Two patches that play through the same Paula mixer differ in role, not in kind. A unified picker with category filter is simpler than two separate tabs and lets designers browse holistically.
- **Library WAV gets copied into the project on first use.** Trades a tiny duplication cost (a few KB per used patch) for a clean runtime contract — the shipped game never references the studio bundle. Matches the "cart ships standalone" identity from the prior ideation.
- **No schema changes.** Library WAVs are stored as regular `AudioSample` entries (the existing schema). The studio knows about library origin via the source path of the WAV (`<studio-bundle>/audio/library/<category>/<name>.wav`); the project file just sees a regular AudioSample with a path inside its own audio-assets/.
- **No `ForceChannel` UI in v1.** Surfacing it forces designers to understand the channel allocator; the auto-allocator covers the common case. v2 mixer-lane panel exposes both visualization and override together.
- **Audition through the engine's actual Paula mixer, not a separate preview engine.** Ensures what the designer hears is what they ship; no "the patch sounded different in the studio than in the game" bugs.
- **No designer-authored patches in v1.** Adding "Save your own SFX as a library patch" would require either an authoring UI (synth or recorder) or a curation flow. Out of scope; user-imported WAVs serve the same need via the existing import path.

---

## Dependencies / Assumptions

- **Depends on idea #7 (build pipeline)** for the ship-loop outcome. Without a build pipeline producing a self-contained binary, R8 is unsatisfied and the bet ("classmate plays the game with sound") isn't proven.
- **Depends on the existing `pixelforge_audio` package** continuing to work — Paula mixer, sample playback, channel auto-allocation via `SuggestedChannelPriority`. v1 doesn't modify the engine; only adds a studio surface.
- **Depends on existing `AudioSample` / `AudioBinding` schema** in `pixelforge_project/audio.go`. v1 adds no new schema fields.
- **Depends on the existing asset-browser AudioSample import path** continuing to work for user-imported WAVs (R9).
- **Assumes ImGui drag-drop** is achievable for the library-to-bindings drag (R6). cimgui-go's drag-drop bindings deferred uintptr payload was deliberately bypassed for the lane-editor (idea #2's prior brainstorm noted the Move Left/Move Right buttons workaround). v1 of this brainstorm may need the same workaround — a "Bind..." button per library patch as the primary path, with drag-drop as a polish enhancement. Planning resolves.
- **Assumes the bundled library** (30 SFX + 4-5 BGM, NES-authentic, mono, Paula-compatible sample rate) is **curated as part of v1** — sound design work to produce the patches is in scope. This is the make-or-break for the bet: patch quality determines whether designers' games sound good.
- **Assumes `go:embed`** is usable for bundling the library WAVs inside the studio binary. Verified pattern (already used elsewhere in the project for embedded fixtures).

---

## Outstanding Questions

### Resolve Before Planning

- *(none — scope is fully resolved at the brainstorm level)*

### Deferred to Planning

- **[Affects R1] [Needs design]** Exact list of ~30 SFX + 4-5 BGM patches in the v1 library. Planning should produce a draft list (probably grouped by category) and validate by sketching: "for each NES-class reference game, can these patches cover the common events?" If not, expand the list before locking.
- **[Affects R6] [Technical]** Whether drag-from-library-to-bindings ships with full ImGui drag-drop (uintptr payload mechanism) or with a "Bind..." button on each library row as the primary path. The cimgui-go drag-drop deferral established in earlier work suggests the button-path is the safer v1; drag-drop polish later.
- **[Affects R7] [Technical]** Exact directory structure inside the project's audio-assets/ for library-sourced patches. Candidates: flat (`audio-assets/jump_spring.wav`), categorized (`audio-assets/library/jump/spring.wav`), prefixed (`audio-assets/lib-jump-spring.wav`). Planning picks whichever round-trips cleanly through the AudioSample schema and avoids name collisions with user-imported WAVs.
- **[Affects R5] [Technical]** Whether audition uses a dedicated preview channel (e.g., reserved channel 5 if Paula allows, or temporarily suspending channel allocation) or shares the runtime's channels and competes with any currently-playing audio. Most ergonomic is a dedicated preview path; planning verifies whether `pixelforge_audio` supports it.
- **[Affects R3] [Technical]** Whether the Audio workspace renders inside the dockspace via the same `Workspace` interface the Scene / Inspector / etc. use (post-ImGui-migration U3), or needs a custom dispatch. Verified existing convention; planning confirms the workspace registration pattern.
- **[Affects R1] [Needs design]** Sound-design source for the bundled patches. Candidates: hand-crafted by a sound designer; generated via sfxr/Bfxr presets exported to WAV; sourced from CC0 NES-style sound libraries. Affects bundle size, patch quality, and licensing posture.
