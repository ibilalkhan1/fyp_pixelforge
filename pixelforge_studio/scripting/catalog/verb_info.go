package catalog

import (
	"fmt"
	"sort"
)

// VerbKind classifies a registered verb recipe so doc-generation +
// coverage tooling can group them.
//
// Action recipes resolve to an ActionBuilder via ActionKind.
// Condition recipes resolve to a ConditionBuilder via ConditionKind.
// The verb-binding compiler (U6) dispatches the two through different
// paths; the generated docs render them under separate headings.
type VerbKind string

const (
	VerbKindAction    VerbKind = "action"
	VerbKindCondition VerbKind = "condition"
)

// ArgInfo describes a single default-arg entry on a Recipe. The
// gendocs tool renders one of these per row under "Arguments"; the
// matrix tool consumes them when explaining what a verb publishes.
//
// Type is a human-readable shape ("string", "float", "bool",
// "number") derived from the default value's Go type. Required is
// always false in v1 — recipe defaults always populate every arg —
// but the struct carries the field so future schema extensions can
// flip it on without changing call sites.
type ArgInfo struct {
	Name     string
	Type     string
	Default  any
	Required bool
}

// VerbInfo is the doc-friendly projection of a registered Recipe.
// Returned by ListVerbs in stable (name-sorted) order so generated
// output is byte-stable across runs.
//
// Topic is the event-bus topic the recipe publishes when its
// ActionKind is "publish_event", extracted from DefaultArgs["event"].
// Empty for recipes that wrap concrete actions (set_value,
// move_entity, play_sample) and for condition recipes.
type VerbInfo struct {
	Name             string
	Kind             VerbKind
	ActionKind       string
	ConditionKind    string
	Topic            string
	Args             []ArgInfo
	RelevantTriggers []string
}

// ListVerbs returns every registered verb recipe as a VerbInfo, sorted
// by name. The slice is freshly allocated on every call so callers may
// mutate it. Used by gendocs (docs generation) + coverage (used/unused
// reporting).
//
// The function snapshots the registry under the read lock so concurrent
// re-registration does not race; the returned VerbInfo values are
// value-typed and safe to retain after the lock is released.
func ListVerbs() []VerbInfo {
	recipeMu.RLock()
	names := make([]string, 0, len(recipes))
	snap := make(map[string]Recipe, len(recipes))
	for k, v := range recipes {
		names = append(names, k)
		snap[k] = v
	}
	recipeMu.RUnlock()

	sort.Strings(names)
	out := make([]VerbInfo, 0, len(names))
	for _, name := range names {
		out = append(out, recipeToVerbInfo(name, snap[name]))
	}
	return out
}

// recipeToVerbInfo projects a Recipe into the doc-friendly VerbInfo
// shape. Kept package-private; the public surface is ListVerbs.
func recipeToVerbInfo(name string, r Recipe) VerbInfo {
	info := VerbInfo{
		Name:             name,
		ActionKind:       r.ActionKind,
		ConditionKind:    r.ConditionKind,
		RelevantTriggers: append([]string(nil), r.RelevantTriggers...),
	}
	if r.IsCondition() {
		info.Kind = VerbKindCondition
	} else {
		info.Kind = VerbKindAction
	}

	// Extract the event-bus topic when the recipe is a
	// publish_event wrapper. Other action kinds (set_value,
	// move_entity, play_sample) leave Topic empty.
	if r.ActionKind == "publish_event" {
		if t, ok := r.DefaultArgs["event"].(string); ok {
			info.Topic = t
		}
	}

	// Sort arg keys for stable output. Filter "target" + "event"
	// out of the argument list when the recipe is publish_event —
	// those two are bookkeeping for the action dispatcher, not
	// user-facing parameters. They're surfaced via VerbInfo.Topic.
	argKeys := make([]string, 0, len(r.DefaultArgs))
	for k := range r.DefaultArgs {
		if r.ActionKind == "publish_event" && (k == "target" || k == "event") {
			continue
		}
		argKeys = append(argKeys, k)
	}
	sort.Strings(argKeys)

	for _, k := range argKeys {
		v := r.DefaultArgs[k]
		info.Args = append(info.Args, ArgInfo{
			Name:    k,
			Type:    argTypeName(v),
			Default: v,
		})
	}
	return info
}

// argTypeName returns a human-readable type label for v. JSON-style
// numerics collapse to "number" so the docs don't lie about int vs.
// float64; the recipe author should override the default value to
// signal "int-like" usage in their description.
func argTypeName(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	case float64, float32:
		return "number"
	case int, int32, int64:
		return "number"
	case nil:
		return "any"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// AllVerbTopics returns every non-empty event-bus topic published by
// a registered action recipe, sorted. Used by the coverage tool to
// cross-reference observed bus traffic against the catalog's known
// topic surface.
func AllVerbTopics() []string {
	verbs := ListVerbs()
	seen := make(map[string]struct{}, len(verbs))
	out := make([]string, 0, len(verbs))
	for _, v := range verbs {
		if v.Topic == "" {
			continue
		}
		if _, dup := seen[v.Topic]; dup {
			continue
		}
		seen[v.Topic] = struct{}{}
		out = append(out, v.Topic)
	}
	sort.Strings(out)
	return out
}
