//go:build js && wasm

package pixelforge_render

import (
	"encoding/json"
	"syscall/js"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_arcade"
)

func installArcadeWebControls(game *pixelforge_arcade.Game) func() {
	if game == nil {
		return func() {}
	}
	state := js.FuncOf(func(_ js.Value, _ []js.Value) any {
		data, err := json.Marshal(game.Snapshot())
		if err != nil {
			return "{}"
		}
		return string(data)
	})
	control := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		switch args[0].String() {
		case "restart":
			game.Reset()
		case "pause":
			game.Paused = !game.Paused
		}
		return nil
	})
	js.Global().Set("pfExampleState", state)
	js.Global().Set("pfExampleControl", control)
	return func() {
		js.Global().Set("pfExampleState", js.Undefined())
		js.Global().Set("pfExampleControl", js.Undefined())
		state.Release()
		control.Release()
	}
}
