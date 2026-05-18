package runtime

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/archetype"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
)

// PlaceholderActionKind marks an Action produced from a verb binding
// whose recipe could not be resolved at compile time. The runtime
// dispatcher (compile.go:compileRule) treats unknown action kinds as
// no-ops + a log warning, so a placeholder rule is harmless at run
// time while still preserving the designer's binding intent across
// load/save round-trips. The inspector (U8) reads this Kind back via
// DetectRoundtripRecipe to render the slot as "advanced (composed)".
const PlaceholderActionKind = "unknown_recipe"

// placeholderRecipeNameKey is the Args key the placeholder Action
// uses to carry the unresolved RecipeName forward. DetectRoundtripRecipe
// reads it when reversing a placeholder rule back into a VerbBinding.
const placeholderRecipeNameKey = "recipe_name"

// CompileVerbBindings translates the entity's VerbBindings slice into
// one EventSheetRule per binding. The produced rules are appended to
// the entity's BehaviorGraph EventSheet by the caller (see compile.go
// hook helper attachVerbBindingsToGraph) so they flow through the
// existing compileRule dispatch — no new runtime path is introduced
// (the R15 promise: verb-sheet authoring compiles down to the same
// EventSheetRule shape power-user authoring produces).
//
// Each binding produces a rule with exactly one Condition and one
// Action:
//
//   Condition.Kind = "event_fired"
//   Condition.Args = {"target": triggerTarget, "event": triggerEvent}
//   Action.Kind    = recipe.ActionKind
//   Action.Args    = merged(recipe.DefaultArgs, binding.ArgOverrides)
//
// Bindings whose RecipeName is not registered in catalog.LookupRecipe
// are preserved as placeholder rules with Action.Kind ==
// PlaceholderActionKind so the inspector can mark the slot "advanced
// (composed)" (U8) without losing the data. The placeholder Action
// carries the unresolved RecipeName + ArgOverrides forward through
// save round-trips.
//
// Triggers whose name does not match any slot in archetypeTemplate
// produce a warning log but still compile — designer content is never
// silently dropped. Pass archetype.Template{} (zero value) when the
// caller does not have or care about an archetype context.
func CompileVerbBindings(entity *pixelforge_project.Entity, archetypeTemplate archetype.Template) ([]pixelforge_project.EventSheetRule, error) {
	if entity == nil || len(entity.VerbBindings) == 0 {
		return nil, nil
	}

	knownSlots := slotNameSet(archetypeTemplate)

	rules := make([]pixelforge_project.EventSheetRule, 0, len(entity.VerbBindings))
	for _, binding := range entity.VerbBindings {
		// Validate trigger against the archetype slot list (advisory
		// only — never drop the binding).
		if archetypeTemplate.Archetype != "" && len(knownSlots) > 0 {
			if _, ok := knownSlots[binding.Trigger]; !ok {
				log.Printf("[runtime/verb-binding] entity %q (archetype %q): trigger %q does not match any archetype slot; compiling anyway",
					entity.ID, archetypeTemplate.Archetype, binding.Trigger)
			}
		}

		targetName, eventName := triggerNameToTarget(binding.Trigger)
		condition := pixelforge_project.Condition{
			Kind: "event_fired",
			Args: map[string]any{
				"target": targetName,
				"event":  eventName,
			},
		}

		recipe, ok := catalog.LookupRecipe(binding.RecipeName)
		if !ok {
			log.Printf("[runtime/verb-binding] entity %q: recipe %q not registered; preserving as placeholder",
				entity.ID, binding.RecipeName)
			rules = append(rules, pixelforge_project.EventSheetRule{
				Conditions: []pixelforge_project.Condition{condition},
				Actions: []pixelforge_project.Action{{
					Kind: PlaceholderActionKind,
					Args: placeholderArgs(binding),
				}},
			})
			continue
		}

		rules = append(rules, pixelforge_project.EventSheetRule{
			Conditions: []pixelforge_project.Condition{condition},
			Actions: []pixelforge_project.Action{{
				Kind: recipe.ActionKind,
				Args: mergeRecipeArgs(recipe.DefaultArgs, binding.ArgOverrides),
			}},
		})
	}

	return rules, nil
}

// triggerNameToTarget converts a trigger-slot name into the pievent
// target + event-string pair the runtime's "event_fired" condition
// subscribes on.
//
// v1 conventions:
//
//   - "when_input/<intent>" — published by the U3 intent layer on the
//     "input/<intent>" pievent target. The intent layer publishes the
//     intent name as the event string (see pixelforge_input package),
//     so we subscribe to "input/<intent>" and match the intent name.
//
//   - All other triggers (when_stepped_on, when_shot, when_touched,
//     when_damaged, every_tick, etc.) — published on the well-known
//     event-bus target "loop.main" (the shared fan-out target other
//     subsystems already use; matches catalog.EventBusTarget). The
//     event string carries the bare trigger name with the leading
//     "when_" prefix stripped. Example: "when_stepped_on" subscribes
//     on "loop.main" matching event "stepped_on".
//
// Returns (targetName, eventName). targetName is what compile.go's
// condTarget reads from Condition.Args["target"]; eventName is what
// buildEventFired matches against the published payload.
func triggerNameToTarget(trigger string) (targetName string, eventName string) {
	t := strings.TrimSpace(trigger)
	if t == "" {
		return "loop.main", ""
	}
	if strings.HasPrefix(t, "when_input/") {
		intent := strings.TrimPrefix(t, "when_input/")
		return "input/" + intent, "input/" + intent
	}
	// Strip the conventional "when_" prefix so verbs publishing the
	// raw event name (e.g., "stepped_on" on loop.main) match.
	eventName = strings.TrimPrefix(t, "when_")
	return "loop.main", eventName
}

// DetectRoundtripRecipe inspects an EventSheetRule and reports
// whether it round-trips cleanly back to a single VerbBinding.
//
// Returns (recipeName, overrides, false) when the rule has exactly
// one Action whose Kind matches a registered recipe.ActionKind AND
// whose Args equal recipe.DefaultArgs ∪ overrides — that is, every
// recipe default is present unchanged, and any extra/different keys
// are reported as overrides. The inspector renders such rules under
// their named recipe.
//
// Returns ("", nil, true) when:
//   - the rule has zero or 2+ Actions (composed),
//   - the single Action's Kind matches no registered recipe (custom
//     or power-user composed),
//   - the Action.Args is missing a recipe default key or has a default
//     key with a non-equal value of different type (off-shape).
//
// Placeholder rules (Action.Kind == PlaceholderActionKind) are also
// "isAdvancedComposed" — the inspector keeps the read-only "advanced
// (composed)" label but reads the preserved RecipeName from the
// placeholder Args so the slot can still surface the original intent.
//
// Equality semantics: scalar comparisons use Go's == operator;
// slice/map values use reflect.DeepEqual. Numeric JSON values are
// kept as float64 throughout the verb-recipe codebase so cross-type
// number comparisons (int vs float64) are intentionally NOT made
// equal — a designer who edits an Args value to int from float64 has
// produced a "different" rule and the inspector should reflect that.
func DetectRoundtripRecipe(rule pixelforge_project.EventSheetRule) (recipeName string, overrides map[string]any, isAdvancedComposed bool) {
	if len(rule.Actions) != 1 {
		return "", nil, true
	}
	action := rule.Actions[0]

	// Placeholder rules: surface the preserved recipe name but report
	// as advanced so the inspector renders the read-only label.
	if action.Kind == PlaceholderActionKind {
		name, _ := action.Args[placeholderRecipeNameKey].(string)
		return name, nil, true
	}

	// Walk every registered recipe; the first whose ActionKind matches
	// and whose DefaultArgs are a subset of the action Args (with the
	// same values) wins. Recipe iteration is deterministic
	// (AllRecipes returns sorted names) so the same rule always maps
	// back to the same recipe across runs.
	for _, name := range catalog.AllRecipes() {
		recipe, ok := catalog.LookupRecipe(name)
		if !ok || recipe.ActionKind != action.Kind {
			continue
		}
		extracted, matched := extractOverrides(recipe.DefaultArgs, action.Args)
		if !matched {
			continue
		}
		return name, extracted, false
	}

	return "", nil, true
}

// mergeRecipeArgs returns a fresh map carrying every entry from
// defaults overlaid with overrides. Override keys win on collision;
// nil inputs are tolerated. The result is a fresh map — callers may
// mutate the return without side-effects on inputs.
//
// The map is unconditionally allocated (rather than returning nil
// when both inputs are nil) so downstream code that ranges over the
// map's keys does not need a nil guard.
func mergeRecipeArgs(defaults, overrides map[string]any) map[string]any {
	out := make(map[string]any, len(defaults)+len(overrides))
	for k, v := range defaults {
		out[k] = v
	}
	for k, v := range overrides {
		out[k] = v
	}
	return out
}

// extractOverrides reports whether merged is a valid extension of
// defaults: every key in defaults appears in merged with a
// reflect.DeepEqual-equal value. Returns the override map (keys in
// merged that differ from defaults OR are absent from defaults) and
// a "matched" boolean.
//
// When matched is false, the returned map is nil.
func extractOverrides(defaults, merged map[string]any) (map[string]any, bool) {
	// Every default key must be present in merged with an equal value.
	for k, defVal := range defaults {
		mergedVal, present := merged[k]
		if !present {
			return nil, false
		}
		if !reflect.DeepEqual(defVal, mergedVal) {
			// Default present but designer overrode it — that's a
			// valid override.
			continue
		}
	}

	overrides := map[string]any{}
	for k, mergedVal := range merged {
		defVal, hasDefault := defaults[k]
		if !hasDefault {
			overrides[k] = mergedVal
			continue
		}
		if !reflect.DeepEqual(defVal, mergedVal) {
			overrides[k] = mergedVal
		}
	}
	if len(overrides) == 0 {
		return nil, true
	}
	return overrides, true
}

// placeholderArgs produces the Args map an unknown-recipe placeholder
// Action carries forward so the inspector + save round-trip can
// reconstruct the original VerbBinding. The placeholder stashes the
// RecipeName + ArgOverrides verbatim under reserved keys; the
// placeholder Action kind is treated as a no-op by the runtime
// dispatch (unknown_recipe is not registered in catalog.LookupAction,
// so compileRule logs "[scripting] unknown action kind" and skips).
func placeholderArgs(binding pixelforge_project.VerbBinding) map[string]any {
	args := map[string]any{
		placeholderRecipeNameKey: binding.RecipeName,
	}
	if len(binding.ArgOverrides) > 0 {
		// Copy overrides into a fresh map so later mutations on the
		// binding's overrides don't bleed into the rule.
		copied := make(map[string]any, len(binding.ArgOverrides))
		for k, v := range binding.ArgOverrides {
			copied[k] = v
		}
		args["arg_overrides"] = copied
	}
	return args
}

// slotNameSet returns the set of TriggerSlot names declared by the
// template, for quick O(1) lookup in CompileVerbBindings. An empty
// template (Archetype == "") yields a nil map, which the caller
// treats as "skip slot validation entirely".
func slotNameSet(t archetype.Template) map[string]struct{} {
	if t.Archetype == "" || len(t.TriggerSlots) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(t.TriggerSlots))
	for _, s := range t.TriggerSlots {
		out[s.Name] = struct{}{}
	}
	return out
}

// attachVerbBindingsToGraph is the hook helper called from
// Engine.compileAll (compile.go integration point). For every entity
// in the project's scenes, it compiles the entity's VerbBindings and
// appends the produced rules to a synthetic BehaviorGraph named
// after the entity ID. The synthetic graphs are added to a fresh
// slice the caller merges into the project's compile list — the
// underlying *pixelforge_project.Project is NOT mutated, so
// project.Marshal round-trips remain stable.
//
// This is the additive hook the U6 plan asked for: the existing
// compileAll iterates p.Behaviors; we extend it by appending the
// synthetic verb-binding graphs to the working list before
// compileGraph runs over each.
//
// Synthetic graph naming convention: "__verb_bindings__:<entityID>".
// The double-underscore prefix guards against collisions with
// designer-named behaviour graphs.
func attachVerbBindingsToGraph(p *pixelforge_project.Project) []pixelforge_project.BehaviorGraph {
	if p == nil {
		return nil
	}
	var out []pixelforge_project.BehaviorGraph
	for sceneIdx := range p.Scenes {
		for entIdx := range p.Scenes[sceneIdx].Entities {
			ent := &p.Scenes[sceneIdx].Entities[entIdx]
			if len(ent.VerbBindings) == 0 {
				continue
			}
			tmpl, _ := archetype.LookupArchetype(ent.Archetype)
			rules, err := CompileVerbBindings(ent, tmpl)
			if err != nil {
				log.Printf("[runtime/verb-binding] entity %q: %v", ent.ID, err)
				continue
			}
			if len(rules) == 0 {
				continue
			}
			out = append(out, pixelforge_project.BehaviorGraph{
				Name:       fmt.Sprintf("__verb_bindings__:%s", ent.ID),
				EntityID:   ent.ID,
				EventSheet: rules,
			})
		}
	}
	return out
}
