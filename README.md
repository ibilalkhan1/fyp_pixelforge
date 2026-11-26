# PixelForge public creator and game library

Main site: https://ibilalkhan1.github.io/fyp_pixelforge/

The home page is the visual creator: make/edit a level and sprite, play, save scene JSON or export a playable HTML game. It retains the original Go/WebAssembly workflow and now uses the supplied offline Studio palette and crisp pixel-art presentation.

The game library retains all five existing Go project/example games and adds four supplied .pf cartridge games: Tetris, Quest, Brickbreaker and Empty House. Their pages run the supplied Emscripten runtime, include actual gameplay previews, and provide Play, Edit in Studio and Download cartridge links. Game launches use a separate __play_ filename to preserve a visitor's named edits in Studio.

Full cartridge Studio is separate at /studio/. Desktop Studio download is separate at /downloads/. The walkthrough is separate at /how-it-works.html. Scene JSON, Go .pforge projects and .pf cartridges are distinct formats.

The supplied runtime JS/WASM are copied byte-for-byte from the local build/web files. studio-host.js seeds 35 cartridges after browser storage restoration, without replacing existing named files, and fixes export asset paths for GitHub project Pages. Native executable export templates are not included. The desktop ZIP is macOS arm64, with Ruby and Gravity disabled.

Visitors run games locally in their browsers. Saving is browser-local and does not publish games or commit to GitHub. No accounts, server database or multiplayer are added by this static deployment.
