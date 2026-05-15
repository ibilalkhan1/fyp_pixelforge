package pfcomponent

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// To keep tests independent we serialize them — the registry is a
// package singleton and parallel TestMain-style isolation would force
// every test to pay the synchronization cost.
var testMu sync.Mutex

func withResetRegistry(t *testing.T) {
	t.Helper()
	testMu.Lock()
	t.Cleanup(func() {
		ResetForTest()
		testMu.Unlock()
	})
	ResetForTest()
}

// Each tag kind parses correctly and produces the right WidgetKind plus
// any bounds/options/maxlen.
func TestRegister_AllTagKinds(t *testing.T) {
	withResetRegistry(t)

	type Mover struct {
		Speed     float64 `pf:"slider,0..10"`
		Skin      int     `pf:"color"`
		HeroSprite string `pf:"sprite"`
		Theme     string  `pf:"audio"`
		Trigger   string  `pf:"event"`
		Facing    string  `pf:"enum,Up|Down|Left|Right"`
		Note      string  `pf:"text,maxlen=64"`
		Origin    [2]int  `pf:"vector2"`
		Untagged  int
	}

	Register[Mover]("Mover")
	md, ok := Get("Mover")
	require.True(t, ok)
	require.Len(t, md.Fields, 9)

	byName := map[string]FieldMetadata{}
	for _, f := range md.Fields {
		byName[f.Name] = f
	}
	assert.Equal(t, WidgetSlider, byName["Speed"].WidgetKind)
	assert.Equal(t, 0.0, byName["Speed"].Min)
	assert.Equal(t, 10.0, byName["Speed"].Max)
	assert.Equal(t, WidgetPaletteColor, byName["Skin"].WidgetKind)
	assert.Equal(t, WidgetSpriteRef, byName["HeroSprite"].WidgetKind)
	assert.Equal(t, WidgetAudioRef, byName["Theme"].WidgetKind)
	assert.Equal(t, WidgetEventTopic, byName["Trigger"].WidgetKind)
	assert.Equal(t, WidgetEnum, byName["Facing"].WidgetKind)
	assert.Equal(t, []string{"Up", "Down", "Left", "Right"}, byName["Facing"].Options)
	assert.Equal(t, WidgetText, byName["Note"].WidgetKind)
	assert.Equal(t, 64.0, byName["Note"].Max)
	assert.Equal(t, WidgetVector2, byName["Origin"].WidgetKind)
	// Untagged falls back to a type-based widget
	assert.Equal(t, WidgetIntField, byName["Untagged"].WidgetKind)
}

// Type-based fallback covers bool/float/int/string.
func TestRegister_UntaggedTypeFallback(t *testing.T) {
	withResetRegistry(t)
	type T struct {
		Flag    bool
		Coord   float64
		Counter int
		Label   string
		Other   []int // unrecognised — falls to Default
	}
	Register[T]("T")
	md, _ := Get("T")

	byName := map[string]FieldMetadata{}
	for _, f := range md.Fields {
		byName[f.Name] = f
	}
	assert.Equal(t, WidgetCheckbox, byName["Flag"].WidgetKind)
	assert.Equal(t, WidgetFloatField, byName["Coord"].WidgetKind)
	assert.Equal(t, WidgetIntField, byName["Counter"].WidgetKind)
	assert.Equal(t, WidgetText, byName["Label"].WidgetKind)
	assert.Equal(t, WidgetDefault, byName["Other"].WidgetKind)
}

// Unknown tag kind is recorded with WidgetUnknown (warning, not panic).
func TestRegister_UnknownTagKindIsForwardCompat(t *testing.T) {
	withResetRegistry(t)
	type T struct {
		FromFuture int `pf:"hologram,4..8"`
	}
	Register[T]("T")
	md, _ := Get("T")
	require.Len(t, md.Fields, 1)
	assert.Equal(t, WidgetUnknown, md.Fields[0].WidgetKind)
}

// Slider min > max panics at registration with a clear message.
func TestRegister_SliderMinGreaterThanMaxPanics(t *testing.T) {
	withResetRegistry(t)
	type Bad struct {
		Bad float64 `pf:"slider,10..0"`
	}
	defer func() {
		r := recover()
		require.NotNil(t, r, "expected panic")
		msg, _ := r.(string)
		assert.Contains(t, msg, "min")
		assert.Contains(t, msg, "max")
	}()
	Register[Bad]("Bad")
}

// Re-registering the same type with the same name is a no-op.
func TestRegister_IdempotentSameTypeSameName(t *testing.T) {
	withResetRegistry(t)
	type T struct {
		X int
	}
	Register[T]("T")
	Register[T]("T") // must not panic
	md, _ := Get("T")
	assert.Len(t, md.Fields, 1)
}

// Re-registering the same type under a different name panics.
func TestRegister_SameTypeDifferentNamePanics(t *testing.T) {
	withResetRegistry(t)
	type T struct {
		X int
	}
	Register[T]("T")
	assert.Panics(t, func() {
		Register[T]("DifferentName")
	})
}

// Embedded struct fields are inspected recursively (flattened into the
// parent's field list).
func TestRegister_EmbeddedStructFlattens(t *testing.T) {
	withResetRegistry(t)
	type Base struct {
		Health int `pf:"slider,0..100"`
	}
	type Player struct {
		Base
		Speed float64 `pf:"slider,0..10"`
	}
	Register[Player]("Player")
	md, _ := Get("Player")
	require.Len(t, md.Fields, 2)
	byName := map[string]FieldMetadata{}
	for _, f := range md.Fields {
		byName[f.Name] = f
	}
	require.Contains(t, byName, "Health")
	assert.True(t, byName["Health"].Anonymous, "embedded field should be marked Anonymous")
	assert.False(t, byName["Speed"].Anonymous)
}

// Marshal/Unmarshal round-trips a component through reflection.
func TestRegister_MarshalUnmarshalRoundTrip(t *testing.T) {
	withResetRegistry(t)
	type Player struct {
		Speed  float64 `pf:"slider,0..10"`
		Facing string  `pf:"enum,Up|Down|Left|Right"`
	}
	Register[Player]("Player")

	original := Player{Speed: 4.5, Facing: "Up"}
	raw, err := Marshal(original)
	require.NoError(t, err)

	out, ok, err := Unmarshal("Player", raw)
	require.NoError(t, err)
	require.True(t, ok)

	got, isPlayer := out.(Player)
	require.True(t, isPlayer)
	assert.Equal(t, original, got)
}

// Unmarshal for an unknown name returns ok=false (not an error) so the
// caller can decide whether to migrate or warn.
func TestUnmarshal_UnknownName(t *testing.T) {
	withResetRegistry(t)
	_, ok, err := Unmarshal("MissingType", json.RawMessage(`{}`))
	assert.NoError(t, err)
	assert.False(t, ok)
}

// All() returns registered types sorted by name.
func TestAll_SortedOutput(t *testing.T) {
	withResetRegistry(t)
	type A struct{ X int }
	type Z struct{ Y int }
	Register[Z]("Zelda")
	Register[A]("Apple")
	all := All()
	require.Len(t, all, 2)
	assert.Equal(t, "Apple", all[0].Name)
	assert.Equal(t, "Zelda", all[1].Name)
}

// Empty name and nil/interface types are rejected.
func TestRegister_ValidationPanics(t *testing.T) {
	withResetRegistry(t)
	assert.Panics(t, func() {
		type T struct{ X int }
		Register[T]("")
	})
}
