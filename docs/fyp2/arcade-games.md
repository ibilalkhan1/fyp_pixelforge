# Studio arcade projects and browser game library

The library contains five separate playable games: Pac-Man, Snake and three new original projects.

| Project | Gameplay | Browser route |
| --- | --- | --- |
| Vault of Ember | Top-down dungeon crawler: three gems, slimes, sword, key and locked exit | `games/dungeon/` |
| River Hopper | Frogger-style traffic, moving logs, three different homes, lives and timer | `games/frogger/` |
| Neon Courier | GTA-inspired small top-down city: walking, entering/exiting a car, traffic, building collisions and three parcel deliveries | `games/city/` |

These are original educational prototypes. Neon Courier is a small driving/courier game, not a full GTA implementation. The new presets are silent. Existing Snake and Pac-Man keep their sound controls.

## One project/runtime path

The three new games load the same bundled `.pforge` data through `capsuleruntime.Boot` and `pixelforge_render.RunProject`. Studio F5 Play, generated native exports and the browser call the shared render/tick path. `pixelforge_arcade` implements deterministic Go simulations and renders through PixelForge's indexed pixel API. HTML only hosts the player, forwards keys and reads the real HUD state.

The optional project `arcade` field selects a preset and stores its authored 20x11 layout. Ordinary projects without that field retain their existing path. A snapshot sprite lets the Scene workspace display the authored starting scene. The current scene workspace does **not** graphically edit preset maps or rules. They are code-driven Studio-compatible projects, not a claim of complete no-code genre authoring. Download/extract the Studio project ZIP, open `game.pforge` in the updated Studio and press F5. F6 restores editing. Earlier Studio/player binaries do not implement these presets, so the player cache version was bumped.

## Controls

Arrows or WASD move. Space attacks in the dungeon, and enters/exits the yellow car or handles a nearby parcel in the city. P pauses and R restarts. Browser buttons provide held direction/action input plus Pause/Resume and Restart. Frogger respawns include a grace period so one held input cannot immediately consume all lives.

## Source and regeneration

- `pixelforge_arcade`: simulations, renderer and embedded Studio projects.
- `cmd/pixelforge-arcade`: native/WASM shared-project entry point. Native selection: `go run ./cmd/pixelforge-arcade --game=city`.
- `cmd/pixelforge-arcade-projects`: regenerates `.pforge` files and initial-frame PNGs. After changing authored data, refresh the web project ZIPs and preview images as well.
- `web-demo/build.sh`: builds the creator, Snake, Pac-Man and one shared arcade WASM for the three new projects.

## Validation and boundaries

The full Mac default suite passed 65 test packages. Regression tests walk the full dungeon route through its win condition, check Frogger traffic/water/logs and three-home victory, exercise the city vehicle and three-delivery objective state, and check pause/restart/determinism. The city objective unit test sets positions to isolate interaction gating; it is not a claim of a full manual driving playthrough. The shared-runtime regression boots all three real projects, drives input and verifies visible HUD pixels. Browser checks observed dungeon scoring, Frogger movement/life loss and a city car pickup/drive/parcel delivery worth 200 points. Native Studio displayed and played all three projects: the dungeon collected a gem for 100 points, River Hopper moved and registered traffic/life loss, and Neon Courier entered its car and drove. Current CI and additional captures are recorded with the delivery.

This does not finish the older Mario/Asteroids/Bomberman/Donkey Kong reference-game ambitions or every Studio authoring workspace. Windows/Linux interactive launch acceptance and presentation-equipment checks remain separate.
