# PixelForge public Studio

Public URL: https://ibilalkhan1.github.io/fyp_pixelforge/

The root now serves the supplied full cartridge Studio, using its C/C++ runtime compiled with Emscripten to WebAssembly. The runtime JS/WASM are copied byte-for-byte from the supplied build/web files. studio-host.js adds demo-cartridge seeding after IndexedDB restore, respects existing saved files, reports startup failures and fixes /export and /js resource URLs for project Pages.

The previous Go creator and all its existing assets remain at creator.html. It uses scene JSON and is a different implementation from the .pf cartridge Studio. Existing game URLs remain unchanged.

35 cartridges are bundled in cartridges/, with a SHA-256 manifest. The public site is static: there is no server-side account service, visitor-game publishing endpoint or multiplayer service. Saves stay in the visitor's browser; download a cartridge to transfer it.

HTML export templates are supplied under export/1.5/html. Native export templates are not included and must not be claimed as working. Physical phone acceptance is separate from desktop browser tests.

See how-it-works.html for the demo walkthrough and architecture explanation. The source repository's main branch contains earlier Go work; publishing the supplied runtime does not make those two implementations identical.
