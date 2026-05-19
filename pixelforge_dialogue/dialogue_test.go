package pixelforge_dialogue_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pdialogue "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_dialogue"
)

// ===== Parser tests =====

func TestParse_ScreenplayLine(t *testing.T) {
	tree, errs := pdialogue.Parse("HERO: Hello world")
	require.Empty(t, errs)
	require.Equal(t, 1, tree.Len())
	n := tree.Nodes[0]
	assert.Equal(t, pdialogue.NodeLine, n.Kind)
	assert.Equal(t, "HERO", n.Speaker)
	assert.Equal(t, "Hello world", n.Text)
}

func TestParse_LabelDeclaration(t *testing.T) {
	tree, errs := pdialogue.Parse(":: intro\nHERO: Hi")
	require.Empty(t, errs)
	require.Equal(t, 2, tree.Len())
	assert.Equal(t, pdialogue.NodeLabel, tree.Nodes[0].Kind)
	assert.Equal(t, "intro", tree.Nodes[0].Label)
	assert.Equal(t, 0, tree.Labels["intro"])
}

func TestParse_EmptyLabelNameIsError(t *testing.T) {
	_, errs := pdialogue.Parse("::")
	require.Len(t, errs, 1)
}

func TestParse_ChoiceWithTarget(t *testing.T) {
	tree, errs := pdialogue.Parse("HERO: choose\n[[yes -> accept]]\n[[no -> decline]]")
	require.Empty(t, errs)
	// Line node + one choice node bundling both options.
	require.Equal(t, 2, tree.Len())
	choice := tree.Nodes[1]
	assert.Equal(t, pdialogue.NodeChoice, choice.Kind)
	require.Len(t, choice.Choices, 2)
	assert.Equal(t, "yes", choice.Choices[0].Text)
	assert.Equal(t, "accept", choice.Choices[0].TargetLabel)
	assert.Equal(t, "no", choice.Choices[1].Text)
}

func TestParse_ChoiceWithCondition(t *testing.T) {
	tree, errs := pdialogue.Parse("[[fight -> battle | if hp > 0]]")
	require.Empty(t, errs)
	require.Equal(t, 1, tree.Len())
	c := tree.Nodes[0].Choices[0]
	assert.Equal(t, "fight", c.Text)
	assert.Equal(t, "battle", c.TargetLabel)
	assert.Equal(t, "hp > 0", c.Condition)
}

func TestParse_ChoiceMissingArrowIsError(t *testing.T) {
	_, errs := pdialogue.Parse("[[broken]]")
	require.Len(t, errs, 1)
}

func TestParse_StageDirectionWalkLeft(t *testing.T) {
	tree, errs := pdialogue.Parse("walk_left 3")
	require.Empty(t, errs)
	require.Equal(t, 1, tree.Len())
	n := tree.Nodes[0]
	assert.Equal(t, pdialogue.NodeStageDirection, n.Kind)
	assert.Equal(t, "walk_left", n.StageVerb)
	assert.Equal(t, 3, n.StageArg)
}

func TestParse_StageDirectionPause(t *testing.T) {
	tree, _ := pdialogue.Parse("pause 30")
	require.Equal(t, 1, tree.Len())
	assert.Equal(t, "pause", tree.Nodes[0].StageVerb)
	assert.Equal(t, 30, tree.Nodes[0].StageArg)
}

func TestParse_BlankLinesIgnored(t *testing.T) {
	tree, errs := pdialogue.Parse("\nHERO: hi\n\n\nNPC: bye\n")
	require.Empty(t, errs)
	assert.Equal(t, 2, tree.Len())
}

func TestParse_UnrecognisedLineSurfacesError(t *testing.T) {
	_, errs := pdialogue.Parse("this is not a valid line")
	require.NotEmpty(t, errs)
}

func TestParse_FullScript(t *testing.T) {
	script := `:: intro
KING: Welcome traveller
KING: What say you?
[[Accept -> accept]]
[[Decline -> decline | if hp > 0]]
:: accept
KING: A wise choice
:: decline
KING: So be it`
	tree, errs := pdialogue.Parse(script)
	require.Empty(t, errs)
	assert.Contains(t, tree.Labels, "intro")
	assert.Contains(t, tree.Labels, "accept")
	assert.Contains(t, tree.Labels, "decline")
}

// ===== Interpolator tests =====

func TestInterpolate_ReplacesKnownKey(t *testing.T) {
	got := pdialogue.Interpolate("Hello {state.name}",
		func(k string) (any, bool) { return "hero", true })
	assert.Equal(t, "Hello hero", got)
}

func TestInterpolate_UnknownKeyLeavesPlaceholder(t *testing.T) {
	got := pdialogue.Interpolate("Hello {state.name}",
		func(k string) (any, bool) { return nil, false })
	assert.Equal(t, "Hello {state.name}", got)
}

func TestInterpolate_NumericValueFormatsAsString(t *testing.T) {
	got := pdialogue.Interpolate("Score: {state.score}",
		func(k string) (any, bool) { return 100, true })
	assert.Equal(t, "Score: 100", got)
}

func TestInterpolate_NoLookupReturnsTextAsIs(t *testing.T) {
	got := pdialogue.Interpolate("Hello {state.name}", nil)
	assert.Equal(t, "Hello {state.name}", got)
}

func TestInterpolate_MultiplePlaceholders(t *testing.T) {
	got := pdialogue.Interpolate("{state.name}: {state.score}",
		func(k string) (any, bool) {
			switch k {
			case "name":
				return "hero", true
			case "score":
				return 50, true
			}
			return nil, false
		})
	assert.Equal(t, "hero: 50", got)
}

// ===== Renderer tests =====

func TestRenderer_StartsAtFirstLine(t *testing.T) {
	tree, _ := pdialogue.Parse("HERO: first\nHERO: second")
	r := pdialogue.NewTextBoxRenderer(tree, nil, nil)
	speaker, text := r.CurrentLine()
	assert.Equal(t, "HERO", speaker)
	assert.Equal(t, "first", text)
}

func TestRenderer_AdvanceMovesToNextLine(t *testing.T) {
	tree, _ := pdialogue.Parse("HERO: first\nHERO: second")
	r := pdialogue.NewTextBoxRenderer(tree, nil, nil)
	require.True(t, r.Advance())
	_, text := r.CurrentLine()
	assert.Equal(t, "second", text)
}

func TestRenderer_AdvanceMarksDoneAtEnd(t *testing.T) {
	tree, _ := pdialogue.Parse("HERO: only")
	r := pdialogue.NewTextBoxRenderer(tree, nil, nil)
	require.False(t, r.Advance())
	assert.True(t, r.IsDone())
}

func TestRenderer_SkipsLabelNodes(t *testing.T) {
	tree, _ := pdialogue.Parse(":: intro\nHERO: hi")
	r := pdialogue.NewTextBoxRenderer(tree, nil, nil)
	speaker, _ := r.CurrentLine()
	assert.Equal(t, "HERO", speaker,
		"the renderer walks past the label node to the first real line")
}

func TestRenderer_PickChoiceJumpsToLabel(t *testing.T) {
	tree, _ := pdialogue.Parse("[[yes -> accept]]\n[[no -> decline]]\n:: accept\nKING: ok\n:: decline\nKING: no")
	r := pdialogue.NewTextBoxRenderer(tree, nil, nil)
	require.True(t, r.PickChoice(0))
	speaker, text := r.CurrentLine()
	assert.Equal(t, "KING", speaker)
	assert.Equal(t, "ok", text)
}

func TestRenderer_PickChoiceInvalidIdxReturnsFalse(t *testing.T) {
	tree, _ := pdialogue.Parse("[[yes -> accept]]\n:: accept\nKING: ok")
	r := pdialogue.NewTextBoxRenderer(tree, nil, nil)
	assert.False(t, r.PickChoice(99))
	assert.False(t, r.PickChoice(-1))
}

func TestRenderer_CurrentChoicesFiltersByCondition(t *testing.T) {
	tree, _ := pdialogue.Parse("[[fight -> battle | if hp > 0]]\n[[flee -> escape]]\n:: battle\n:: escape")
	r := pdialogue.NewTextBoxRenderer(tree, nil, func(expr string) bool {
		return expr == "hp > 0" // first choice visible
	})
	visible := r.CurrentChoices()
	require.Len(t, visible, 2)
}

func TestRenderer_CurrentChoicesHidesByCondition(t *testing.T) {
	tree, _ := pdialogue.Parse("[[fight -> battle | if hp > 0]]\n[[flee -> escape]]\n:: battle\n:: escape")
	r := pdialogue.NewTextBoxRenderer(tree, nil, func(expr string) bool {
		return false // first choice hidden
	})
	visible := r.CurrentChoices()
	require.Len(t, visible, 1)
	assert.Equal(t, "flee", visible[0].Text)
}

func TestRenderer_InterpolatesCurrentLine(t *testing.T) {
	tree, _ := pdialogue.Parse("HERO: Hello {state.name}")
	r := pdialogue.NewTextBoxRenderer(tree, func(k string) (any, bool) {
		if k == "name" {
			return "stranger", true
		}
		return nil, false
	}, nil)
	_, text := r.CurrentLine()
	assert.Equal(t, "Hello stranger", text)
}

func TestRenderer_StageDirectionsRemainVisible(t *testing.T) {
	tree, _ := pdialogue.Parse("walk_left 3\nHERO: now I'm here")
	r := pdialogue.NewTextBoxRenderer(tree, nil, nil)
	n := r.CurrentNode()
	require.NotNil(t, n)
	assert.Equal(t, pdialogue.NodeStageDirection, n.Kind,
		"stage directions are not skipped — the engine must dispatch them")
	r.Advance()
	speaker, _ := r.CurrentLine()
	assert.Equal(t, "HERO", speaker)
}

func TestRenderer_ResetReturnsCursorToStart(t *testing.T) {
	tree, _ := pdialogue.Parse("HERO: first\nHERO: second")
	r := pdialogue.NewTextBoxRenderer(tree, nil, nil)
	r.Advance()
	r.Reset()
	_, text := r.CurrentLine()
	assert.Equal(t, "first", text)
}

func TestRenderer_NilTreeIsDone(t *testing.T) {
	r := pdialogue.NewTextBoxRenderer(nil, nil, nil)
	assert.True(t, r.IsDone())
	assert.Nil(t, r.CurrentNode())
}
