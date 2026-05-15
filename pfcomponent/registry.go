package pfcomponent

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"sync"
)

// registry is the package-level singleton. Tests reset it via
// ResetForTest; production code never touches it directly.
var (
	regMu  sync.RWMutex
	regMap = map[string]TypeMetadata{}
	// byType lets us catch a developer trying to register the same
	// Go type under two different names — a likely copy-paste bug.
	byType = map[reflect.Type]string{}
)

// Register parses T's struct fields and stores the resulting
// metadata under name. Re-registering the same (T, name) pair is a
// no-op so packages can safely re-init. Re-registering T under a
// different name panics — it's almost always a developer mistake
// caught best at startup.
//
// Panics on bad pf:"..." tags so register-time mistakes (slider min
// greater than max, malformed range, ...) surface immediately rather
// than as mysterious widget bugs at edit time.
func Register[T any](name string) {
	if name == "" {
		panic("pfcomponent.Register: name must be non-empty")
	}

	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		panic(fmt.Sprintf("pfcomponent.Register: %s has nil type — register a concrete struct, not an interface", name))
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("pfcomponent.Register: %s: only struct types are registerable, got %v", name, t.Kind()))
	}

	regMu.Lock()
	defer regMu.Unlock()

	if existingName, ok := byType[t]; ok {
		if existingName == name {
			return // idempotent — same type, same name
		}
		panic(fmt.Sprintf(
			"pfcomponent.Register: type %v already registered as %q, cannot re-register as %q",
			t, existingName, name))
	}
	if _, ok := regMap[name]; ok {
		panic(fmt.Sprintf("pfcomponent.Register: name %q already registered to a different type", name))
	}

	md, err := parseTypeMetadata(name, t)
	if err != nil {
		panic("pfcomponent.Register: " + err.Error())
	}
	regMap[name] = md
	byType[t] = name
}

// Get returns the registered metadata for name. ok is false if no
// type with that name is registered.
func Get(name string) (TypeMetadata, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
	md, ok := regMap[name]
	return md, ok
}

// All returns every registered TypeMetadata in alphabetical-by-name
// order. Useful for the inspector's component-add menu.
func All() []TypeMetadata {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]TypeMetadata, 0, len(regMap))
	for _, md := range regMap {
		out = append(out, md)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ResetForTest clears the registry. Test-only — production code never
// unregisters. Exported with the conventional test-only suffix so a
// vet rule or reviewer can flag misuse.
func ResetForTest() {
	regMu.Lock()
	defer regMu.Unlock()
	regMap = map[string]TypeMetadata{}
	byType = map[reflect.Type]string{}
}

// Marshal serializes value (any registered component instance) to JSON.
// The wire form is exactly encoding/json's default for the underlying
// struct — Marshal exists so the editor has a single call site even
// when M5 adds custom per-component encoding hooks.
func Marshal(value any) ([]byte, error) {
	if value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(value)
}

// Unmarshal decodes raw into a fresh instance of the type registered
// under name. Returns ok=false if name is unknown so callers can
// surface a clear migration error rather than panicking.
func Unmarshal(name string, raw json.RawMessage) (any, bool, error) {
	md, ok := Get(name)
	if !ok {
		return nil, false, nil
	}
	ptr := reflect.New(md.Type)
	if len(raw) == 0 || string(raw) == "null" {
		return ptr.Elem().Interface(), true, nil
	}
	if err := json.Unmarshal(raw, ptr.Interface()); err != nil {
		return nil, true, fmt.Errorf("pfcomponent.Unmarshal %s: %w", name, err)
	}
	return ptr.Elem().Interface(), true, nil
}
