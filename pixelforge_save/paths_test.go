package pixelforge_save_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"
)

func TestSanitize_RemovesSpacesUppercase(t *testing.T) {
	assert.Equal(t, "my_game", pisave.Sanitize("My Game"))
}

func TestSanitize_StripsSpecialChars(t *testing.T) {
	assert.Equal(t, "heros-quest_2", pisave.Sanitize("Hero's-Quest 2!"))
}

func TestSanitize_EmptyReturnsUntitled(t *testing.T) {
	assert.Equal(t, "untitled", pisave.Sanitize(""))
}

func TestSanitize_AllSpecialCharsReturnsUntitled(t *testing.T) {
	assert.Equal(t, "untitled", pisave.Sanitize("!@#$%^&*()"))
}

func TestGameDir_ComposesPath(t *testing.T) {
	got, err := pisave.GameDir("My Game")
	require.NoError(t, err)
	assert.Contains(t, got, "pixelforge-games")
	assert.Contains(t, got, "my_game")
}

func TestSlotPath_AppendsJSONExtension(t *testing.T) {
	got, err := pisave.SlotPath("test", pisave.Slot1Name)
	require.NoError(t, err)
	assert.Contains(t, got, "slot1.json")
}

func TestSlotConstants_AreCanonical(t *testing.T) {
	assert.Equal(t, "autosave", pisave.AutosaveSlotName)
	assert.Equal(t, "slot1", pisave.Slot1Name)
	assert.Equal(t, "slot2", pisave.Slot2Name)
	assert.Equal(t, "slot3", pisave.Slot3Name)
}
