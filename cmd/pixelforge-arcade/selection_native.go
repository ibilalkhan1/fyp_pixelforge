//go:build !js || !wasm

package main

func selectedGame(fallback string) string { return fallback }
