#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
GOOS=js GOARCH=wasm go build -buildvcs=false -trimpath -ldflags='-s -w' -o web-demo/engine.wasm ./cmd/pixelforge-web
install -m 644 "$(go env GOROOT)/lib/wasm/wasm_exec.js" web-demo/wasm_exec.js
mkdir -p web-demo/games/snake web-demo/games/pacman
GOOS=js GOARCH=wasm go build -buildvcs=false -trimpath -ldflags='-s -w' -o web-demo/games/snake/game.wasm ./pixelforge_examples/snake
GOOS=js GOARCH=wasm go build -buildvcs=false -trimpath -ldflags='-s -w' -o web-demo/games/pacman/game.wasm ./pixelforge_examples/pacman

GOOS=js GOARCH=wasm go build -buildvcs=false -trimpath -ldflags='-s -w' -o web-demo/games/arcade.wasm ./cmd/pixelforge-arcade
printf 'Built creator and five game examples. Serve web-demo/ over HTTP.\n'
