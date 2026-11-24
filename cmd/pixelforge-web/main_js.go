//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	pf "github.com/ibilalkhan1/fyp_pixelforge"
	"syscall/js"
)

var game Game
var callbacks []js.Func

func expose(name string, fn func(js.Value, []js.Value) any) {
	f := js.FuncOf(fn)
	callbacks = append(callbacks, f)
	js.Global().Set(name, f)
}
func main() {
	pf.SetScreenSize(width, height)
	expose("pfLoad", func(_ js.Value, args []js.Value) any {
		s, e := parseScene([]byte(args[0].String()))
		if e != nil {
			return e.Error()
		}
		game.Reset(s)
		return ""
	})
	expose("pfTick", func(_ js.Value, args []js.Value) any {
		if args[0].Bool() {
			game.Step(args[1].Int(), args[2].Int())
		}
		pixels := game.Render(args[3].Bool())
		v := js.Global().Get("Uint8ClampedArray").New(len(pixels))
		js.CopyBytesToJS(v, pixels)
		return v
	})
	expose("pfState", func(_ js.Value, _ []js.Value) any {
		b, _ := json.Marshal(map[string]any{"tick": game.Tick, "x": game.X.Int(), "y": game.Y.Int(), "score": len(game.Collected), "total": len(game.Scene.Gems), "won": game.Won, "lost": game.Lost})
		return string(b)
	})
	expose("pfPalette", func(_ js.Value, _ []js.Value) any {
		p := []string{}
		for _, c := range pf.Palette {
			p = append(p, fmt.Sprintf("#%06x", uint32(c)))
		}
		b, _ := json.Marshal(p)
		return string(b)
	})
	js.Global().Set("pfReady", true)
	select {}
}
