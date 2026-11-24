# Browser demonstrator

Run `./web-demo/build.sh` from the repository root, then serve this folder over HTTP. Requires Go 1.24.2 or newer. `engine.wasm` and its matching Go `wasm_exec.js` are generated together and ignored by Git.

The browser handles UI, keyboard collection, saving and downloads. `cmd/pixelforge-web` runs scene validation, fixed-point movement, collision and PixelForge drawing in Go. The core canvas is 320x180 and uses PixelForge's 64-colour palette. The UI exposes the first 16 colours for painting.

This is a bounded demonstration. It does not load Studio `.pforge` files or provide full native Studio workspaces. The JSON version is 1. Only browser-local scenes are stored. No backend account, database, telemetry or external assets are required.

HTML export embeds the current scene, compiled engine and matching Go runtime into a single downloaded file. Keep an exported copy before the viva and rehearse opening it manually on the presentation machine.
