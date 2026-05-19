// backend_js.go is the WASM (js/wasm) Backend, sibling to
// backend_native.go. The build-tag matrix selects exactly one per
// build:
//
//	desktop / server (default):  backend_native.go  (//go:build !js)
//	browser / WASM:              backend_js.go      (//go:build js)
//
// Slots persist via the browser's window.localStorage. Each key is
// namespaced "pixelforge.save.<sanitised-title>.<slot>" so two
// different games hosted on the same origin don't collide.
// Snapshots are base64-encoded JSON because localStorage stores
// strings only; binary frames go through the same Marshal/Unmarshal
// path the native backend uses.

//go:build js

package pixelforge_save

import (
	"encoding/base64"
	"fmt"
	"strings"
	"syscall/js"
)

// BackendJS is the browser-localStorage Backend. Keys live under
// the keyPrefix "pixelforge.save.<sanitised-title>." so a single
// origin serving multiple games keeps each game's slots isolated.
type BackendJS struct {
	keyPrefix string
}

// NewBackendJS binds a backend rooted at the per-game localStorage
// prefix derived from title via Sanitize. Construction is pure —
// no JS calls happen until the first Read/Write/Delete/List.
func NewBackendJS(gameTitle string) *BackendJS {
	return &BackendJS{keyPrefix: JSKeyPrefix(gameTitle)}
}

// KeyPrefix exposes the namespace this backend operates under.
// Used by diagnostics; tests assert on it to verify isolation
// between games.
func (b *BackendJS) KeyPrefix() string { return b.keyPrefix }

// Read returns the bytes stored under slotName, decoded from the
// base64 envelope. Returns ErrNotFound when the key isn't present.
func (b *BackendJS) Read(slotName string) ([]byte, error) {
	storage := js.Global().Get("localStorage")
	val := storage.Call("getItem", b.keyPrefix+slotName)
	if val.IsNull() || val.IsUndefined() {
		return nil, ErrNotFound
	}
	data, err := base64.StdEncoding.DecodeString(val.String())
	if err != nil {
		return nil, fmt.Errorf("save: decode localStorage value for %q: %w", slotName, err)
	}
	return data, nil
}

// Write encodes data as base64 and persists it under slotName.
// Browser QuotaExceededError surfaces as ErrQuotaExceeded so the
// engine UX layer can present a "save full" dialog without parsing
// JS-side strings.
func (b *BackendJS) Write(slotName string, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = mapJSPanic(r)
		}
	}()
	encoded := base64.StdEncoding.EncodeToString(data)
	js.Global().Get("localStorage").Call("setItem", b.keyPrefix+slotName, encoded)
	return nil
}

// Delete removes the slot. Missing slots no-op silently — matches
// the native backend's idempotent contract.
func (b *BackendJS) Delete(slotName string) error {
	js.Global().Get("localStorage").Call("removeItem", b.keyPrefix+slotName)
	return nil
}

// List enumerates every key under this backend's prefix. Browser
// localStorage exposes only key index + name + value — no per-key
// timestamps or sizes — so SlotMeta.SavedAt is the zero time and
// Size reflects the encoded (base64) payload length rather than
// the underlying byte count. The Save / Load menu sorts by Name
// alphabetically when SavedAt is zero across all entries.
func (b *BackendJS) List() ([]SlotMeta, error) {
	storage := js.Global().Get("localStorage")
	length := storage.Get("length").Int()
	var out []SlotMeta
	for i := 0; i < length; i++ {
		key := storage.Call("key", i)
		if key.IsNull() || key.IsUndefined() {
			continue
		}
		k := key.String()
		if !strings.HasPrefix(k, b.keyPrefix) {
			continue
		}
		slot := strings.TrimPrefix(k, b.keyPrefix)
		val := storage.Call("getItem", k)
		size := int64(0)
		if !val.IsNull() && !val.IsUndefined() {
			size = int64(val.Get("length").Int())
		}
		out = append(out, SlotMeta{Name: slot, Size: size})
	}
	return out, nil
}

// mapJSPanic translates a recovered js.Error panic into one of the
// typed Backend sentinels. JS APIs raise exceptions; js.Value.Call
// panics with a *js.Error when the underlying call throws.
// QuotaExceededError is the only one we currently classify — every
// other error surfaces as a wrapped "localStorage: <message>".
func mapJSPanic(r any) error {
	if e, ok := r.(*js.Error); ok && e != nil {
		name := e.Value.Get("name").String()
		if name == "QuotaExceededError" {
			return ErrQuotaExceeded
		}
		return fmt.Errorf("save: localStorage error: %s", e.Error())
	}
	if e, ok := r.(js.Error); ok {
		name := e.Value.Get("name").String()
		if name == "QuotaExceededError" {
			return ErrQuotaExceeded
		}
		return fmt.Errorf("save: localStorage error: %s", e.Error())
	}
	return fmt.Errorf("save: localStorage panic: %v", r)
}
