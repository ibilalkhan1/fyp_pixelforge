# Browser demonstrator

Run `./web-demo/build.sh` from the repository root, then serve this folder over HTTP. Requires Go 1.24.2 or newer. `engine.wasm` and its matching Go `wasm_exec.js` are generated together and ignored by Git.

The browser handles UI, keyboard collection, saving and downloads. `cmd/pixelforge-web` runs scene validation, fixed-point movement, collision and PixelForge drawing in Go. The core canvas is 320x180 and uses PixelForge's 64-colour palette. The UI exposes the first 16 colours for painting.

This is a bounded demonstration. It does not load Studio `.pforge` files or provide full native Studio workspaces. The JSON version is 1. Only browser-local scenes are stored. No backend account, database, telemetry or external assets are required.

HTML export embeds the current scene, compiled engine and matching Go runtime into a single downloaded file. Keep an exported copy before the viva and rehearse opening it manually on the presentation machine.

## Game creation workflow

Choose Blank game, Crystal quest example or Maze escape example, then click New game. Name the project, paint walls/crystals/hazards, move the player, and use Finish to set an escape objective. Choose the win rule: collect every crystal or reach the finish. Hazards end the run, and Reset starts it again. Escape-mode crystals are optional bonus points. The creator does not check whether a level is solvable; play-test your layout before sharing.

Scene version 1 remains compatible with earlier collector scenes. Optional `mode` defaults to `collect`; `hazards` and `goal` extend the same format. Imports validate bounds and disallow player/finish overlap with dangerous or blocked tiles. JSON downloads preserve editable projects; playable HTML downloads embed their rules and support keyboard and touch controls. There is one browser-local save slot. Export before replacing it.

The free public creator is hosted at https://ibilalkhan1.github.io/fyp_pixelforge/ on a separate `gh-pages` branch. No account, server-side code or running presentation laptop is required. Hosting a visitor's exported game is a separate action; the creator downloads the file and does not automatically publish visitors' work.
