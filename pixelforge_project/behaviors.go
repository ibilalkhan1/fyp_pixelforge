package pixelforge_project

// BehaviorGraph holds the data side of M5 visual scripting: a Step
// sequence (compiled to pixelforge_routine.Step at runtime) plus event
// sheet rules (compiled to pievent.Subscribe handlers). The runtime
// interpreter in M5 walks these structures; the schema reserves them
// at M1 so existing projects don't break when M5 ships.
type BehaviorGraph struct {
	// Name is the project-scoped identifier referenced by
	// EventSubscriptions and code-gen.
	Name string `json:"name"`

	// EntityID, when non-empty, scopes the behavior to a single
	// entity (it's an instance method, in OO terms). Empty means
	// the behavior is a free function callable from anywhere.
	EntityID string `json:"entity_id"`

	// Steps is the lane-editor's serialized sequence. Empty in M1
	// projects; M5 lane editor populates it.
	Steps []StepNode `json:"steps"`

	// EventSheet is the GDevelop-style condition/action table.
	// Empty in M1; M5 event sheet editor populates it.
	EventSheet []EventSheetRule `json:"event_sheet"`
}

// StepNode is one card in the lane editor — a Wait, Tween, Move, Play,
// Publish, Branch, or Custom-extension. Concrete shape per kind is
// encoded in Args; the M5 runtime interpreter switches on Kind.
type StepNode struct {
	Kind string         `json:"kind"`
	Args map[string]any `json:"args"`
}

// EventSheetRule is one "When X, do Y" row. Sub-events nest under
// Children for indent-style grouping; the M5 interpreter walks them
// depth-first.
type EventSheetRule struct {
	Conditions []Condition       `json:"conditions"`
	Actions    []Action          `json:"actions"`
	Children   []EventSheetRule  `json:"children,omitempty"`
}

// Condition is one predicate in a sheet row. Kind names a registered
// predicate (e.g. "event_fired", "key_held", "value_lt"); Args supplies
// its inputs.
type Condition struct {
	Kind string         `json:"kind"`
	Args map[string]any `json:"args"`
}

// Action is one effect in a sheet row. Kind names a registered effect
// (e.g. "play_sample", "set_value", "publish_event"); Args supplies
// its inputs.
type Action struct {
	Kind string         `json:"kind"`
	Args map[string]any `json:"args"`
}
