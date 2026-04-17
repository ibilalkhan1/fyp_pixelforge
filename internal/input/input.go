package input

import "github.com/ibilalkhan1/fyp_pixelforge"

type State[T comparable] struct {
	pressedInputs map[T]*pressedInput
}

func (s *State[T]) Duration(input T) int {
	return s.pressedInput(input).duration()
}

func (s *State[T]) SetDownFrame(input T, frame int) {
	s.pressedInput(input).downFrame = frame
}

func (s *State[T]) SetUpFrame(input T, frame int) {
	s.pressedInput(input).upFrame = frame
}

func (s *State[T]) pressedInput(input T) *pressedInput {
	if s.pressedInputs == nil {
		s.pressedInputs = map[T]*pressedInput{}
	}
	p, ok := s.pressedInputs[input]
	if !ok {
		p = &pressedInput{downFrame: -1, upFrame: -1}
		s.pressedInputs[input] = p
	}
	return p
}

type pressedInput struct {
	downFrame, upFrame int
}

func (p pressedInput) duration() int {
	if p.downFrame < 0 {
		return 0
	}
	if p.downFrame > p.upFrame {
		return pixelforge.Frame - p.downFrame + 1
	}
	if p.downFrame == p.upFrame && p.upFrame == pixelforge.Frame {
		return 1
	}
	return 0
}
