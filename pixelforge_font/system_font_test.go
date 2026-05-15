package pixelforge_font

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSystemSheet_HasAsciiPrintables(t *testing.T) {
	s := NewSystemSheet()
	for ch := rune(' '); ch < rune(''); ch++ {
		_, ok := s.Chars[ch]
		assert.True(t, ok, "missing rune %d (%q)", ch, string(ch))
	}
	assert.Equal(t, 13, s.Height)
}

func TestNewSystemSheet_HasFallbackRune(t *testing.T) {
	s := NewSystemSheet()
	_, ok := s.Chars['�']
	require.True(t, ok)
}

func TestNewSystemSheet_NonAsciiReturnsBlankCanvas(t *testing.T) {
	s := NewSystemSheet()
	// 'à' (U+00E0) is outside the printable ASCII range. The sheet
	// won't have an entry for it; that's the expected behaviour (the
	// cofont default sheet handles those characters separately).
	_, ok := s.Chars['à']
	assert.False(t, ok)
}

func TestNewSystemSheet_GlyphDimensions(t *testing.T) {
	s := NewSystemSheet()
	a := s.Chars['A']
	assert.Equal(t, 6, a.Area.W)
	assert.Equal(t, 13, a.Area.H)
}
