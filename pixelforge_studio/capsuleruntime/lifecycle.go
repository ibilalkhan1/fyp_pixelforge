package capsuleruntime

import (
	piaudio "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	piinput "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_input"
	pikey "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	pipad "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_pad"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	scripting "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/runtime"
)

// Start connects authored behaviours and semantic input before the first frame.
// It is deliberately separate from Boot, which also serves asset-only tools.
func (rt *Runtime) Start() {
	if rt == nil || rt.Project == nil || rt.behaviours != nil {
		return
	}
	rt.inputCompiler = piinput.NewCompiler(pikey.Target(), pipad.ButtonTarget())
	bindings := rt.Project.InputMap
	if len(bindings) == 0 {
		bindings = pixelforge_project.DefaultInputMap()
	}
	rt.inputCompiler.Compile(bindings)
	rt.behaviours = scripting.New(rt.Project)
	for _, name := range piaudio.AllSampleNames() {
		rt.behaviours.Context().SetSample(name, piaudio.LookupSample(name))
	}
	rt.behaviours.Start()
}

// Close removes subscriptions so repeated preview sessions do not multiply rules.
func (rt *Runtime) Close() {
	if rt == nil {
		return
	}
	if rt.behaviours != nil {
		rt.behaviours.Stop()
		rt.behaviours = nil
	}
	if rt.inputCompiler != nil {
		rt.inputCompiler.Unsubscribe()
		rt.inputCompiler = nil
	}
	if rt.verbHandler != 0 {
		piloop.VerbsBus().Unsubscribe(rt.verbHandler)
		rt.verbHandler = 0
	}
}

// Behaviours exposes the live scripting engine to Studio diagnostics.
func (rt *Runtime) Behaviours() *scripting.Engine {
	if rt == nil {
		return nil
	}
	return rt.behaviours
}
