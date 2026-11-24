//go:build js && wasm

package pixelforge_gametree

import (
	"encoding/json"
	"syscall/js"
)

var browserCallbacks []js.Func

func installWebGameControls(state func() map[string]any, action func(string)) {
	snapshot := js.FuncOf(func(_ js.Value, _ []js.Value) any { data, _ := json.Marshal(state()); return string(data) })
	control := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 {
			action(args[0].String())
		}
		return nil
	})
	browserCallbacks = append(browserCallbacks, snapshot, control)
	js.Global().Set("pfExampleState", snapshot)
	js.Global().Set("pfExampleControl", control)
}
