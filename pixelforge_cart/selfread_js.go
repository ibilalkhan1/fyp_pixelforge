//go:build js

package pixelforge_cart

import (
	"fmt"
	"syscall/js"
)

// ReadSelf returns the cart payload published by the WASM
// bootstrapping shim. The build host writes the cart bytes into
// window.__pixelforgeCart (as a Uint8Array) BEFORE the Go runtime
// starts; this function copies them into a Go []byte.
//
// Returns ErrNoCart cleanly when the global is undefined — that's
// how a fresh, unshipped WASM player surfaces.
//
// Returns ErrPayloadTooLarge when the published array's length
// exceeds MaxPayloadSize.
//
// Returns a wrapped error if the global is the wrong JS type
// (e.g. someone published a number) since js.CopyBytesToGo would
// otherwise panic on a non-Uint8Array source.
func ReadSelf() ([]byte, error) {
	global := js.Global().Get("__pixelforgeCart")
	if global.IsUndefined() || global.IsNull() {
		return nil, ErrNoCart
	}

	// Defensive type check — js.CopyBytesToGo expects an
	// ArrayBufferView-like object and panics on a primitive.
	if t := global.Type(); t != js.TypeObject {
		return nil, fmt.Errorf("pixelforge_cart: __pixelforgeCart is %s, expected Uint8Array", t.String())
	}

	length := global.Get("byteLength").Int()
	if length < 0 {
		// JS exposed a negative byteLength — extremely unusual but
		// defensive against pathological globals.
		return nil, fmt.Errorf("%w: negative byteLength", ErrCorruptCart)
	}
	if uint64(length) > MaxPayloadSize {
		return nil, fmt.Errorf("%w: cart is %d bytes, limit %d", ErrPayloadTooLarge, length, MaxPayloadSize)
	}

	payload := make([]byte, length)
	if length > 0 {
		n := js.CopyBytesToGo(payload, global)
		if n != length {
			return nil, fmt.Errorf("%w: copied %d of %d bytes", ErrCorruptCart, n, length)
		}
	}
	return payload, nil
}
