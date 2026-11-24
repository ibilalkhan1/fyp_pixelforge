#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
GOOS=js GOARCH=wasm go build -buildvcs=false -trimpath -ldflags='-s -w' -o web-demo/engine.wasm ./cmd/pixelforge-web
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web-demo/wasm_exec.js
printf 'Built browser demo. Serve web-demo/ over HTTP.\n'
