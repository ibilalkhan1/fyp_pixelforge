package pixelforge_dialogue

import (
	"fmt"
	"strings"
)

// Interpolate resolves {state.key} substrings inside text against
// the supplied lookup. Unknown keys render as `{state.key}`
// literal so designers spot typos quickly. Nested braces aren't
// supported in v1.
func Interpolate(text string, lookup func(key string) (any, bool)) string {
	if lookup == nil {
		return text
	}
	var sb strings.Builder
	i := 0
	for i < len(text) {
		// Look for the literal {state. prefix.
		start := strings.Index(text[i:], "{state.")
		if start < 0 {
			sb.WriteString(text[i:])
			break
		}
		sb.WriteString(text[i : i+start])
		// Find matching '}'.
		open := i + start
		close := strings.Index(text[open:], "}")
		if close < 0 {
			// Unterminated — emit the rest verbatim.
			sb.WriteString(text[open:])
			break
		}
		full := text[open : open+close+1]
		key := text[open+len("{state.") : open+close]
		if v, ok := lookup(key); ok {
			sb.WriteString(formatInterpolated(v))
		} else {
			sb.WriteString(full)
		}
		i = open + close + 1
	}
	return sb.String()
}

// formatInterpolated renders one blackboard value as text. Numeric
// values use the locale-independent default (%v); strings pass
// through; booleans render as true/false.
func formatInterpolated(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	}
	return fmt.Sprintf("%v", v)
}
