// Package catalog provides a process-wide registry of Step, Condition,
// and Action Kind builders used by the M5 visual scripting runtime.
//
// Each Kind is a string name registered alongside a builder function
// that converts a serialised StepNode/Condition/Action Args map into a
// concrete runtime primitive (pixelforge_routine.Step, a predicate, or
// an effect).
//
// The registry mirrors pfcomponent.Register's shape — package-level
// init() blocks call Register*; the runtime engine looks up names at
// compile time and falls back to a no-op + warning when a Kind is
// unknown.
package catalog

import (
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_routine"
)

// Context is the runtime context passed to every builder.
//
// Builders look up project assets (sprites, audio, entities) through
// this interface so the catalog package doesn't take a hard dependency
// on *pixelforge_project.Project. The runtime engine package
// implements Context; tests can supply their own fakes.
type Context interface {
	// Sample returns the loaded audio sample with the given name, or
	// nil if no sample is registered. Used by play_sample / Play.
	Sample(name string) any

	// Entity returns the scene-resident entity by its ID, or nil if
	// none matches. Used by move_entity / set_value.
	Entity(id string) any

	// ValueRef returns a pointer-like accessor for a named project
	// value (component field). Returns nil when unresolved. v1 keeps
	// the type abstract; the runtime walks the path string.
	ValueRef(path string) ValueRef

	// LookupTarget returns the pievent target registered under name,
	// or nil if unknown. Wraps pievent.LookupTarget so tests can
	// inject fakes.
	LookupTarget(name string) any
}

// ValueRef abstracts mutable access to a project value (typically a
// component field). The runtime resolves these lazily; builders
// receive a ValueRef rather than a concrete *T.
type ValueRef interface {
	Get() any
	Set(v any)
}

// StepBuilder constructs a runtime Step from a serialised Args map.
type StepBuilder func(args map[string]any, ctx Context) (pixelforge_routine.Step, error)

// Predicate is the runtime form of a Condition: returns true when the
// condition is satisfied for the given payload.
type Predicate func(payload any) bool

// ConditionBuilder constructs a Predicate from a serialised Args map.
type ConditionBuilder func(args map[string]any, ctx Context) (Predicate, error)

// Effect is the runtime form of an Action: applies a side effect for
// the given payload.
type Effect func(payload any)

// ActionBuilder constructs an Effect from a serialised Args map.
type ActionBuilder func(args map[string]any, ctx Context) (Effect, error)

var (
	regMu        sync.RWMutex
	stepBuilders = map[string]StepBuilder{}
	condBuilders = map[string]ConditionBuilder{}
	actBuilders  = map[string]ActionBuilder{}
)

// RegisterStep adds (name, builder) to the Step Kind registry.
// Duplicate names log a warning and overwrite.
func RegisterStep(name string, builder StepBuilder) {
	if name == "" || builder == nil {
		return
	}
	regMu.Lock()
	defer regMu.Unlock()
	if _, exists := stepBuilders[name]; exists {
		log.Printf("[catalog] step kind %q re-registered; overwriting", name)
	}
	stepBuilders[name] = builder
}

// RegisterCondition adds (name, builder) to the Condition Kind registry.
func RegisterCondition(name string, builder ConditionBuilder) {
	if name == "" || builder == nil {
		return
	}
	regMu.Lock()
	defer regMu.Unlock()
	if _, exists := condBuilders[name]; exists {
		log.Printf("[catalog] condition kind %q re-registered; overwriting", name)
	}
	condBuilders[name] = builder
}

// RegisterAction adds (name, builder) to the Action Kind registry.
func RegisterAction(name string, builder ActionBuilder) {
	if name == "" || builder == nil {
		return
	}
	regMu.Lock()
	defer regMu.Unlock()
	if _, exists := actBuilders[name]; exists {
		log.Printf("[catalog] action kind %q re-registered; overwriting", name)
	}
	actBuilders[name] = builder
}

// LookupStep returns the StepBuilder for name, or nil if unregistered.
func LookupStep(name string) StepBuilder {
	regMu.RLock()
	defer regMu.RUnlock()
	return stepBuilders[name]
}

// LookupCondition returns the ConditionBuilder for name, or nil.
func LookupCondition(name string) ConditionBuilder {
	regMu.RLock()
	defer regMu.RUnlock()
	return condBuilders[name]
}

// LookupAction returns the ActionBuilder for name, or nil.
func LookupAction(name string) ActionBuilder {
	regMu.RLock()
	defer regMu.RUnlock()
	return actBuilders[name]
}

// AllSteps returns every registered Step Kind name in alphabetical order.
func AllSteps() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]string, 0, len(stepBuilders))
	for k := range stepBuilders {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// AllConditions returns every registered Condition Kind name in alphabetical order.
func AllConditions() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]string, 0, len(condBuilders))
	for k := range condBuilders {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// AllActions returns every registered Action Kind name in alphabetical order.
func AllActions() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]string, 0, len(actBuilders))
	for k := range actBuilders {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ResetForTest clears every registry. Test-only.
func ResetForTest() {
	regMu.Lock()
	defer regMu.Unlock()
	stepBuilders = map[string]StepBuilder{}
	condBuilders = map[string]ConditionBuilder{}
	actBuilders = map[string]ActionBuilder{}
}

// Args helpers — JSON-unmarshalled Args maps store numerics as
// float64, strings as string, bools as bool. These helpers convert
// with a clear error on type mismatch so misshaped graphs surface a
// readable diagnostic rather than a runtime panic.

// ArgInt reads an integer-valued arg. Accepts float64 (JSON), int,
// int32, int64 — anything that round-trips to int without precision
// loss.
func ArgInt(args map[string]any, key string) (int, error) {
	v, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("missing arg %q", key)
	}
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case int:
		return n, nil
	case int32:
		return int(n), nil
	case int64:
		return int(n), nil
	default:
		return 0, fmt.Errorf("arg %q: expected number, got %T", key, v)
	}
}

// ArgIntOr returns the integer-valued arg, or fallback if absent or
// malformed.
func ArgIntOr(args map[string]any, key string, fallback int) int {
	n, err := ArgInt(args, key)
	if err != nil {
		return fallback
	}
	return n
}

// ArgFloat reads a float-valued arg. Accepts float64 (JSON), int, etc.
func ArgFloat(args map[string]any, key string) (float64, error) {
	v, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("missing arg %q", key)
	}
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("arg %q: expected number, got %T", key, v)
	}
}

// ArgFloatOr returns the float-valued arg, or fallback if absent or malformed.
func ArgFloatOr(args map[string]any, key string, fallback float64) float64 {
	n, err := ArgFloat(args, key)
	if err != nil {
		return fallback
	}
	return n
}

// ArgString reads a string-valued arg.
func ArgString(args map[string]any, key string) (string, error) {
	v, ok := args[key]
	if !ok {
		return "", fmt.Errorf("missing arg %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("arg %q: expected string, got %T", key, v)
	}
	return s, nil
}

// ArgStringOr returns the string-valued arg, or fallback if absent or malformed.
func ArgStringOr(args map[string]any, key string, fallback string) string {
	s, err := ArgString(args, key)
	if err != nil {
		return fallback
	}
	return s
}

// ArgBool reads a boolean-valued arg.
func ArgBool(args map[string]any, key string) (bool, error) {
	v, ok := args[key]
	if !ok {
		return false, fmt.Errorf("missing arg %q", key)
	}
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("arg %q: expected bool, got %T", key, v)
	}
	return b, nil
}

// ArgBoolOr returns the bool-valued arg, or fallback if absent or malformed.
func ArgBoolOr(args map[string]any, key string, fallback bool) bool {
	b, err := ArgBool(args, key)
	if err != nil {
		return fallback
	}
	return b
}
