package widgets_test

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
	"github.com/stretchr/testify/assert"
)

func TestRuleRow_HitTest_Conditions(t *testing.T) {
	r := widgets.NewRuleRow(0, 0, 200, 18, widgets.RuleRowOptions{
		Indent:     0,
		Conditions: []string{"A", "B"},
		Actions:    []string{"X"},
	})
	col, idx := r.HitTest(2, 0)
	assert.Equal(t, 0, col)
	assert.Equal(t, 0, idx)

	col, idx = r.HitTest(2, 9)
	assert.Equal(t, 0, col)
	assert.Equal(t, 1, idx)
}

func TestRuleRow_HitTest_AddAffordance(t *testing.T) {
	r := widgets.NewRuleRow(0, 0, 200, 27, widgets.RuleRowOptions{
		Conditions: []string{"A"},
		Actions:    []string{},
	})
	col, idx := r.HitTest(2, 9)
	assert.Equal(t, 0, col)
	assert.Equal(t, 1, idx, "+ sentinel after 1 condition")
}

func TestRuleRow_HitTest_Actions(t *testing.T) {
	r := widgets.NewRuleRow(0, 0, 200, 18, widgets.RuleRowOptions{
		Conditions: []string{},
		Actions:    []string{"A", "B"},
	})
	col, idx := r.HitTest(110, 9)
	assert.Equal(t, 1, col)
	assert.Equal(t, 1, idx)
}

func TestRuleRow_Indent_ShiftsHitArea(t *testing.T) {
	r := widgets.NewRuleRow(0, 0, 200, 18, widgets.RuleRowOptions{
		Indent:     2,
		Conditions: []string{"A"},
	})
	// (10, 0) sits inside the indent padding.
	col, idx := r.HitTest(10, 0)
	assert.Equal(t, -1, col)
	assert.Equal(t, -1, idx)
	// After indent (2 * 16 = 32 px), conditions begin.
	col, idx = r.HitTest(34, 0)
	assert.Equal(t, 0, col)
	assert.Equal(t, 0, idx)
}

func TestRuleRow_ZeroItemsRenders(t *testing.T) {
	r := widgets.NewRuleRow(0, 0, 200, 18, widgets.RuleRowOptions{})
	// "+" affordance at index 0 for both columns.
	col, idx := r.HitTest(2, 0)
	assert.Equal(t, 0, col)
	assert.Equal(t, 0, idx)
}
