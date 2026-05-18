package pixelforge_blackboard

// CanonicalKeyType classifies the value held under a reserved
// blackboard key. The studio inspector workspace (U9) consults the
// declared type to pick an appropriate editor widget (int slider for
// scores, text field for the current scene name, etc.); the
// verb-binding compiler (U6) consults it to validate recipe argument
// shapes at load time.
//
// Designer-added keys outside CanonicalKeys are untyped — their
// concrete value type is whatever the publishing verb stored. This is
// intentional: v1 keeps the schema minimal and avoids tracking
// per-key types for arbitrary names.
type CanonicalKeyType string

const (
	TypeInt    CanonicalKeyType = "int"
	TypeFloat  CanonicalKeyType = "float"
	TypeString CanonicalKeyType = "string"
	TypeBool   CanonicalKeyType = "bool"
)

// CanonicalKeySpec describes a reserved blackboard key: its declared
// value type and the default applied when the runtime initialises a
// fresh project that does not yet carry the key.
type CanonicalKeySpec struct {
	Type    CanonicalKeyType
	Default any
}

// Canonical key names. Verb recipes (U4) reference these constants
// when reading or mutating well-known gameplay state.
const (
	KeyScore        = "score"
	KeyLives        = "lives"
	KeyHealth       = "health"
	KeyCurrentScene = "current_scene"
	KeyPlayerX      = "player_x"
	KeyPlayerY      = "player_y"
)

// CanonicalKeys is the reserved key registry. Order is irrelevant;
// the studio inspector renders canonical keys in a stable
// alphabetical or display order separately.
var CanonicalKeys = map[string]CanonicalKeySpec{
	KeyScore:        {Type: TypeInt, Default: 0},
	KeyLives:        {Type: TypeInt, Default: 3},
	KeyHealth:       {Type: TypeInt, Default: 3},
	KeyCurrentScene: {Type: TypeString, Default: ""},
	KeyPlayerX:      {Type: TypeFloat, Default: 0.0},
	KeyPlayerY:      {Type: TypeFloat, Default: 0.0},
}
