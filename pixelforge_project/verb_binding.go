package pixelforge_project

// VerbBinding is one entry in an entity's character-sheet: a trigger
// slot (e.g. "when_stepped_on") bound to a named verb recipe (e.g.
// "die") with optional argument overrides. The verb-binding compiler
// (pixelforge_studio/scripting/runtime) translates each binding into
// one EventSheetRule at project-load time; the runtime treats the
// produced rules identically to power-user-authored EventSheet rules.
//
// Unknown RecipeName values are preserved on load (not dropped) so a
// project authored against a newer recipe library round-trips through
// an older studio without losing data. The inspector renders unknown
// bindings as "advanced (composed)" placeholders.
type VerbBinding struct {
	// Trigger names the archetype trigger slot the binding fills
	// (e.g. "when_stepped_on", "when_input/jump"). Must match a slot
	// declared by the entity's archetype template; mismatches log a
	// warning at compile time but the binding is still preserved.
	Trigger string `json:"trigger"`

	// RecipeName names the verb recipe to fire (e.g. "die",
	// "give_points"). Looked up via the scripting catalog's recipe
	// registry at compile time. Unknown names are preserved.
	RecipeName string `json:"recipe_name"`

	// ArgOverrides patches the recipe's DefaultArgs at compile time.
	// Override keys win over recipe defaults; absent keys inherit the
	// default. Nil/empty means "use defaults". The map's value type is
	// any so JSON numbers / strings / bools all round-trip without a
	// per-recipe type schema.
	ArgOverrides map[string]any `json:"arg_overrides,omitempty"`
}
