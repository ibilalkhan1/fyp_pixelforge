//go:build js && wasm

package main

import "syscall/js"

func selectedGame(fallback string) string {
	value := js.Global().Get("document").Get("body").Get("dataset").Get("game")
	if value.Type() == js.TypeString {
		return value.String()
	}
	return fallback
}
