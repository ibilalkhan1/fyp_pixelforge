package pixelforge_physics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLadderState_BeginEndClimb(t *testing.T) {
	var s LadderState
	assert.False(t, s.Climbing)

	s.BeginClimb(LadderUp)
	assert.True(t, s.Climbing)
	assert.Equal(t, LadderUp, s.Direction)

	s.EndClimb()
	assert.False(t, s.Climbing)
	assert.Equal(t, LadderIdle, s.Direction)
}

func TestApplyLadderMovement_Up(t *testing.T) {
	b := NewBody("p", Vec2{FixedFromInt(50), FixedFromInt(100)},
		FixedFromInt(8), FixedFromInt(16))
	b.Velocity = Vec2{FixedZero, FixedFromInt(5)} // arbitrary y velocity to clear
	state := LadderState{Climbing: true, Direction: LadderUp}
	moved := ApplyLadderMovement(b, state, FixedFromInt(2))
	assert.True(t, moved)
	assert.Equal(t, FixedFromInt(98), b.Position.Y)
	assert.Equal(t, FixedZero, b.Velocity.Y, "y velocity zeroed while climbing")
}

func TestApplyLadderMovement_Down(t *testing.T) {
	b := NewBody("p", Vec2{FixedFromInt(50), FixedFromInt(100)},
		FixedFromInt(8), FixedFromInt(16))
	state := LadderState{Climbing: true, Direction: LadderDown}
	moved := ApplyLadderMovement(b, state, FixedFromInt(3))
	assert.True(t, moved)
	assert.Equal(t, FixedFromInt(103), b.Position.Y)
}

func TestApplyLadderMovement_NotClimbing_NoOp(t *testing.T) {
	b := NewBody("p", Vec2{FixedFromInt(50), FixedFromInt(100)},
		FixedFromInt(8), FixedFromInt(16))
	moved := ApplyLadderMovement(b, LadderState{Climbing: false}, FixedFromInt(2))
	assert.False(t, moved)
	assert.Equal(t, FixedFromInt(100), b.Position.Y)
}
