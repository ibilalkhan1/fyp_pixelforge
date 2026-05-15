package scripting

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/scanner"
	"go/token"
	"strings"
	"text/template"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

//go:embed view_as_go.tmpl
var viewAsGoTemplate string

// Emit renders the behaviour graph to a readable Go source string.
// One-way (no parser): the output is for users escaping to code,
// not for round-trip authoring (Scope Boundaries).
//
// Unknown Kinds emit as TODO comments rather than failing the
// whole render.
func Emit(graph pixelforge_project.BehaviorGraph) (string, error) {
	tmpl, err := template.New("view_as_go").Parse(viewAsGoTemplate)
	if err != nil {
		return "", fmt.Errorf("template parse: %w", err)
	}
	data := buildEmitData(graph)
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execute: %w", err)
	}
	return buf.String(), nil
}

// emitRule is a per-rule view-model used by the template.
type emitRule struct {
	PredicateLines []string
	ActionLines    []string
}

// emitData is the view-model the template renders.
type emitData struct {
	Name       string
	SafeName   string
	Steps      []string
	EventSheet []emitRule
}

func buildEmitData(g pixelforge_project.BehaviorGraph) emitData {
	d := emitData{
		Name:     g.Name,
		SafeName: safeGoIdentifier(g.Name),
	}
	for _, n := range g.Steps {
		d.Steps = append(d.Steps, emitStep(n))
	}
	for _, r := range g.EventSheet {
		d.EventSheet = append(d.EventSheet, buildEmitRule(r))
	}
	return d
}

func buildEmitRule(r pixelforge_project.EventSheetRule) emitRule {
	out := emitRule{}
	for _, c := range r.Conditions {
		out.PredicateLines = append(out.PredicateLines, emitCondition(c))
	}
	for _, a := range r.Actions {
		out.ActionLines = append(out.ActionLines, emitAction(a))
	}
	return out
}

func emitStep(n pixelforge_project.StepNode) string {
	switch n.Kind {
	case "Wait":
		ticks := intArg(n.Args, "ticks")
		return fmt.Sprintf("piroutine.Wait(%d)", ticks)
	case "Publish":
		t := stringArg(n.Args, "target")
		e := stringArg(n.Args, "event")
		return fmt.Sprintf("piroutine.Publish(/* %s */ targetFor(%q), %q)", t, t, e)
	case "Tween":
		from := floatArg(n.Args, "from")
		to := floatArg(n.Args, "to")
		ticks := intArg(n.Args, "ticks")
		ease := stringArg(n.Args, "ease")
		return fmt.Sprintf("piroutine.Tween(nil, %g, %g, %d, %q)", from, to, ticks, ease)
	case "Move":
		dx := intArg(n.Args, "dx")
		dy := intArg(n.Args, "dy")
		ticks := intArg(n.Args, "ticks")
		return fmt.Sprintf("piroutine.Move(nil, %d, %d, %d)", dx, dy, ticks)
	case "Play":
		return "piroutine.Play(0, nil, 1.0, 1.0)"
	case "Branch":
		return "piroutine.Branch(func() bool { return false }, nil, nil)"
	case "Custom":
		return fmt.Sprintf("/* Custom: hook=%q */ piroutine.Call(func() {})", stringArg(n.Args, "hook"))
	}
	return fmt.Sprintf("/* TODO: unknown kind %q */ piroutine.Call(func() {})", n.Kind)
}

func emitCondition(c pixelforge_project.Condition) string {
	switch c.Kind {
	case "event_fired":
		e := stringArg(c.Args, "event")
		return fmt.Sprintf("payload == %q", e)
	case "key_held":
		k := stringArg(c.Args, "key")
		return fmt.Sprintf("pikey.Duration(%q) > 0", k)
	case "value_lt":
		v := stringArg(c.Args, "value")
		c := floatArg(c.Args, "compare_to")
		return fmt.Sprintf("readValue(%q) < %g", v, c)
	case "value_gt":
		v := stringArg(c.Args, "value")
		c := floatArg(c.Args, "compare_to")
		return fmt.Sprintf("readValue(%q) > %g", v, c)
	case "value_eq":
		v := stringArg(c.Args, "value")
		c := floatArg(c.Args, "compare_to")
		return fmt.Sprintf("readValue(%q) == %g", v, c)
	}
	return fmt.Sprintf("/* TODO: unknown condition %q */ true", c.Kind)
}

func emitAction(a pixelforge_project.Action) string {
	switch a.Kind {
	case "play_sample":
		n := stringArg(a.Args, "name")
		return fmt.Sprintf("playSample(%q)", n)
	case "set_value":
		t := stringArg(a.Args, "target")
		return fmt.Sprintf("setValue(%q, %#v)", t, a.Args["value"])
	case "publish_event":
		t := stringArg(a.Args, "target")
		e := stringArg(a.Args, "event")
		return fmt.Sprintf("targetFor(%q).Publish(%q)", t, e)
	case "move_entity":
		id := stringArg(a.Args, "entity")
		dx := floatArg(a.Args, "dx")
		dy := floatArg(a.Args, "dy")
		return fmt.Sprintf("moveEntity(%q, %g, %g)", id, dx, dy)
	case "branch":
		return "/* branch action: predicate path collapses to no-op in v1 */"
	}
	return fmt.Sprintf("/* TODO: unknown action %q */", a.Kind)
}

func intArg(args map[string]any, key string) int {
	if args == nil {
		return 0
	}
	switch n := args[key].(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

func floatArg(args map[string]any, key string) float64 {
	if args == nil {
		return 0
	}
	switch n := args[key].(type) {
	case float64:
		return n
	case int:
		return float64(n)
	}
	return 0
}

func stringArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	if s, ok := args[key].(string); ok {
		return s
	}
	return ""
}

// safeGoIdentifier converts a graph name into a valid Go identifier.
// Invalid characters become underscores; identifiers that start with a
// digit get a leading "G_". Empty strings become "unnamed". The
// emitter prepends a "// TODO: invalid identifier %q" comment when
// sanitisation was needed.
func safeGoIdentifier(name string) string {
	if name == "" {
		return "unnamed"
	}
	if isValidGoIdent(name) {
		return name
	}
	var b strings.Builder
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			if i == 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "unnamed"
	}
	return out
}

func isValidGoIdent(s string) bool {
	if s == "" {
		return false
	}
	var sc scanner.Scanner
	fset := token.NewFileSet()
	sc.Init(fset.AddFile("ident", fset.Base(), len(s)), []byte(s), nil, 0)
	_, tok, lit := sc.Scan()
	if tok != token.IDENT || lit != s {
		return false
	}
	_, tok2, _ := sc.Scan()
	return tok2 == token.EOF
}
