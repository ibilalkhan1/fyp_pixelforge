# PixelForge

PixelForge is an educational 2D pixel game engine in Go, with a native Studio and a focused browser demonstrator. The core provides indexed-colour drawing, sprites, events, input, audio, routines and developer tools.

## Try the browser demo

```sh
./web-demo/build.sh
python3 -m http.server 8765 --bind 127.0.0.1 --directory web-demo
```

Open http://127.0.0.1:8765. Edit a tile or an 8x8 character, choose Play scene, and use arrow keys or WASD to collect the crystals. Save a scene in the browser, download its JSON, or export a standalone playable HTML.

The browser demo uses the actual Go renderer through WebAssembly. It uses a separate, limited scene format and does not replace the full native Studio. Audio is outside the creator. The game library includes Snake, Pac-Man and three Studio-compatible projects: Vault of Ember (dungeon crawler), River Hopper (Frogger-style) and Neon Courier (top-down city driving). The three new projects share the native Studio/player runtime and offer project downloads. See [arcade project details](docs/fyp2/arcade-games.md).

Faculty link: https://ibilalkhan1.github.io/fyp_pixelforge/ . GitHub Pages serves the demo without keeping the development laptop online.

## Native Studio

```sh
go build -o pf-studio ./pixelforge_studio
```

The native application uses Ebitengine and Dear ImGui bindings and needs the applicable platform build tools. Real export builders are available in ordinary builds. Referenced sprites and supported WAV/MP3/Vorbis assets travel in new universal carts. Rebuild packaged players when shipping the updated cart format.

## Readiness and validation

Read [FYP-II delivery status](docs/fyp2/readiness.md) for supported workflows, verified checks, limitations and the temporary demo procedure. Basic native F5 Play, authored keyboard movement, F6 Stop restoration and an independently exported game were verified on Mac. See [native/browser acceptance](docs/fyp2/native-browser-acceptance.md). Broader native authoring and cross-platform interactive launches remain outstanding. This is an educational prototype with a bounded browser creator.
