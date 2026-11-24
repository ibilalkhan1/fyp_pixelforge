package pixelforge_render

import (
	"github.com/hajimehoshi/ebiten/v2"
	pikey "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
	"strings"
)

var syntheticKeys = map[pikey.Key]bool{}

func inputKey(key ebiten.Key) pikey.Key {
	switch key { //nolint:exhaustive // Named non-navigation keys use the string fallback below.
	case ebiten.KeyArrowUp:
		return pikey.Up
	case ebiten.KeyArrowDown:
		return pikey.Down
	case ebiten.KeyArrowLeft:
		return pikey.Left
	case ebiten.KeyArrowRight:
		return pikey.Right
	case ebiten.KeySpace:
		return pikey.Space
	case ebiten.KeyEnter:
		return pikey.Enter
	case ebiten.KeyEscape:
		return pikey.Esc
	case ebiten.KeyShiftLeft:
		return pikey.ShiftLeft
	case ebiten.KeyShiftRight:
		return pikey.ShiftRight
	case ebiten.KeyControlLeft:
		return pikey.CtrlLeft
	case ebiten.KeyControlRight:
		return pikey.CtrlRight
	case ebiten.KeyAltLeft:
		return pikey.AltLeft
	case ebiten.KeyAltRight:
		return pikey.AltRight
	}
	name := key.String()
	if len(name) == 1 {
		return pikey.Key(strings.ToUpper(name))
	}
	if strings.HasPrefix(name, "Digit") {
		return pikey.Key(strings.TrimPrefix(name, "Digit"))
	}
	return pikey.Key(name)
}
func publishInputSnapshot(frame InputFrame) {
	if frame.NativeEvents {
		return
	}
	next := map[pikey.Key]bool{}
	for _, key := range frame.Keys {
		mapped := inputKey(key)
		next[mapped] = true
		if !syntheticKeys[mapped] {
			pikey.Target().Publish(pikey.Event{Type: pikey.EventDown, Key: mapped})
		}
	}
	for key := range syntheticKeys {
		if !next[key] {
			pikey.Target().Publish(pikey.Event{Type: pikey.EventUp, Key: key})
		}
	}
	syntheticKeys = next
}
