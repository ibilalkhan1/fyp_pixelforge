package pixelforge_dialogue

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseError carries one parse failure plus the line it occurred
// on so the Dialogue workspace's error view can render with
// precise locations.
type ParseError struct {
	Line    int    // 1-based
	Message string
}

func (e ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}

// Parse converts a dialogue script into a Tree. Returns the tree
// (best-effort even on errors — recoverable lines still land in
// the tree) plus a list of parse errors the caller can surface to
// the designer.
//
// Each line is dispatched by prefix:
//
//	":: name"               → label declaration
//	"[[choice -> label]]"   → choice node (one per line; choices
//	                          group into a single NodeChoice for
//	                          consecutive lines)
//	"walk_* N" / "pause N"  → stage direction
//	"SPEAKER: text"         → screenplay-style line
//	""                      → blank, ignored
//
// Unrecognised lines surface as parse errors but the parser keeps
// going so the designer sees every problem at once instead of one
// at a time.
func Parse(script string) (*Tree, []ParseError) {
	tree := &Tree{Labels: map[string]int{}}
	var errs []ParseError
	lines := strings.Split(script, "\n")

	var pendingChoices []Choice
	flushChoices := func() {
		if len(pendingChoices) == 0 {
			return
		}
		tree.Nodes = append(tree.Nodes, Node{
			Kind:    NodeChoice,
			Choices: pendingChoices,
		})
		pendingChoices = nil
	}

	for i, raw := range lines {
		lineNum := i + 1
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		// Label
		if strings.HasPrefix(line, "::") {
			flushChoices()
			name := strings.TrimSpace(line[2:])
			if name == "" {
				errs = append(errs, ParseError{Line: lineNum, Message: "empty label name"})
				continue
			}
			tree.Labels[name] = len(tree.Nodes)
			tree.Nodes = append(tree.Nodes, Node{Kind: NodeLabel, Label: name})
			continue
		}

		// Choice — may be one of several consecutive [[ ... ]] lines.
		if strings.HasPrefix(line, "[[") && strings.HasSuffix(line, "]]") {
			c, err := parseChoice(line, lineNum)
			if err != nil {
				errs = append(errs, *err)
				continue
			}
			pendingChoices = append(pendingChoices, c)
			continue
		}

		// Any non-choice line ends the current choice group.
		flushChoices()

		// Stage direction
		if dir, ok := parseStageDirection(line); ok {
			tree.Nodes = append(tree.Nodes, dir)
			continue
		}

		// Speaker line: "SPEAKER: text".
		speaker, text, ok := splitSpeakerLine(line)
		if !ok {
			errs = append(errs, ParseError{
				Line:    lineNum,
				Message: "expected 'SPEAKER: text', label '::', choice '[[..]]', or stage direction",
			})
			continue
		}
		tree.Nodes = append(tree.Nodes, Node{
			Kind:    NodeLine,
			Speaker: speaker,
			Text:    text,
		})
	}
	flushChoices()
	return tree, errs
}

// parseChoice handles "[[text -> label]]" or
// "[[text -> label | if cond]]".
func parseChoice(line string, lineNum int) (Choice, *ParseError) {
	body := strings.TrimSpace(line[2 : len(line)-2])
	// Split off the optional condition first.
	condition := ""
	if idx := strings.Index(body, "| if "); idx >= 0 {
		condition = strings.TrimSpace(body[idx+len("| if "):])
		body = strings.TrimSpace(body[:idx])
	}
	// Split text -> label.
	idx := strings.Index(body, "->")
	if idx < 0 {
		err := ParseError{Line: lineNum, Message: "choice missing '->' separator"}
		return Choice{}, &err
	}
	text := strings.TrimSpace(body[:idx])
	target := strings.TrimSpace(body[idx+2:])
	if text == "" || target == "" {
		err := ParseError{Line: lineNum, Message: "choice text or label is empty"}
		return Choice{}, &err
	}
	return Choice{Text: text, TargetLabel: target, Condition: condition}, nil
}

// parseStageDirection recognises walk_left/right/up/down N and
// pause N. Anything else returns ok=false so the caller falls
// through to the speaker-line path.
func parseStageDirection(line string) (Node, bool) {
	parts := strings.Fields(line)
	if len(parts) != 2 {
		return Node{}, false
	}
	switch parts[0] {
	case "walk_left", "walk_right", "walk_up", "walk_down", "pause":
		n, err := strconv.Atoi(parts[1])
		if err != nil || n < 0 {
			return Node{}, false
		}
		return Node{
			Kind:      NodeStageDirection,
			StageVerb: parts[0],
			StageArg:  n,
		}, true
	}
	return Node{}, false
}

// splitSpeakerLine splits "SPEAKER: text" into ("SPEAKER", "text").
// Returns ok=false when no colon is found. Multi-colon lines split
// on the first colon so dialogue like "BOB: time: 3pm" still parses.
func splitSpeakerLine(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	speaker := strings.TrimSpace(line[:idx])
	text := strings.TrimSpace(line[idx+1:])
	if speaker == "" {
		return "", "", false
	}
	return speaker, text, true
}
