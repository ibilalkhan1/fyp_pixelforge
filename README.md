# PixelForge

PixelForge is an educational 2D pixel game engine in Go, with a native Studio and a focused browser demonstrator. The core provides indexed-colour drawing, sprites, events, input, audio, routines and developer tools.

## Try the browser demo

```sh
./web-demo/build.sh
python3 -m http.server 8765 --bind 127.0.0.1 --directory web-demo
```

Open http://127.0.0.1:8765. Edit a tile or an 8x8 character, choose Play scene, and use arrow keys or WASD to collect the crystals. Save a scene in the browser, download its JSON, or export a standalone playable HTML.

The browser demo uses the actual Go renderer through WebAssembly. It uses a separate, limited scene format and does not replace the full native Studio. Audio is outside this browser demonstration.

## Native Studio

```sh
go build -o pf-studio ./pixelforge_studio
```

The native application uses Ebitengine and Dear ImGui bindings and needs the applicable platform build tools. Real export builders are available in ordinary builds. Referenced sprite/WAV assets travel in new universal carts. Rebuild packaged players when shipping the updated cart format.

## Readiness and validation

Read [FYP-II delivery status](docs/fyp2/readiness.md) for supported workflows, verified checks, limitations and the temporary demo procedure. Full native Studio workflow acceptance and cross-platform launches remain outstanding. This repository should not be presented as a production-complete no-code engine.
