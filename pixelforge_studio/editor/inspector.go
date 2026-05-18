// inspector.go drives the Inspector ImGui window. U4 of the ImGui
// migration plan replaced the pgui widget bank + per-(entity, comp,
// field) widget cache with a single immediate-mode switch on
// pfcomponent.WidgetKind. ImGui reads + writes component fields
// directly through pointer arguments — no widget state survives
// between frames.
package editor

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_input"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/archetype"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/runtime"
)

// verbBindingUnsetLabel is the placeholder dropdown item that, when
// picked, removes the entity's VerbBinding for the slot (letting the
// archetype template's DefaultVerb show through again).
const verbBindingUnsetLabel = "(unset)"

// verbBindingDefaultSuffix is appended to a slot's dropdown current
// label when the slot is showing the archetype's DefaultVerb rather
// than a designer-set binding. The italic styling can come later; the
// suffix communicates the state in plain text right now.
const verbBindingDefaultSuffix = " (default)"

// advancedComposedLabel is the read-only label rendered in place of a
// verb-pick dropdown when the slot's binding compiles to a rule the
// round-trip detector flags as advanced (composed) — unknown recipe or
// power-user-edited.
const advancedComposedLabel = "advanced (composed)"

// debugLineHeight is the vertical pixel height of one line of
// ebitenutil.DebugPrintAt output. Native widget callers (canvas,
// asset_browser) carry this over from the deleted chrome.go so their
// layout math stays stable until U5/U7/U8 retire them.
const debugLineHeight = 16

// vectorFill outlines a fill rect in c. Used by remaining native
// widget code (canvas / asset_browser) — Inspector itself no longer
// reaches for ebiten/vector.
func vectorFill(dst *ebiten.Image, r widgets.Rect, c color.RGBA) {
	vector.DrawFilledRect(dst, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), c, false)
}

func debugPrint(dst *ebiten.Image, s string, x, y int) {
	ebitenutil.DebugPrintAt(dst, s, x, y)
}

// strokeRectAt outlines the supplied widgets.Rect with a 1-pixel
// border in c. Native widget helper carried over from the deleted
// chrome.go so callers (canvas.go, asset_browser.go) keep their
// existing signatures until U5 retires their native path.
func strokeRectAt(dst *ebiten.Image, r widgets.Rect, c color.RGBA) {
	vector.StrokeRect(dst, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), 1, c, false)
}

// Inspector renders the right panel via ImGui immediate-mode calls.
// U4 dropped the M2 widget cache — ImGui owns transient widget state
// (drag start, focus, dropdown open) keyed by the labels we pass.
//
// U8 adds one cross-frame map: showAdvanced tracks which (entity,
// trigger) pairs have the "Show advanced" parameter-form toggle open.
// ImGui's immediate-mode toggles need a backing bool the widget can
// flip; the map keeps that bool per slot without leaking it into the
// .pforge schema.
type Inspector struct {
	showAdvanced map[string]bool
}

// NewInspector returns the singleton inspector. The constructor stays
// for symmetry with the M0 API; the per-slot showAdvanced map is the
// only piece of state the inspector carries between frames.
func NewInspector() *Inspector {
	return &Inspector{showAdvanced: map[string]bool{}}
}

// advancedKey composes the showAdvanced map key from the entity ID +
// trigger slot name. Stable across frames so a toggle survives until
// the user flips it again or switches entities.
func advancedKey(entityID, trigger string) string {
	return entityID + "::" + trigger
}

// Render emits the Inspector ImGui window for the editor's selected
// entity. Returns true if any field was edited this frame so the
// caller marks the project dirty.
//
// Skipped when no live ImGui backend is attached so unit tests that
// rely on the test stub continue to drive Update/Draw without
// segfaulting in cgo.
func (i *Inspector) Render(e *Editor) bool {
	if e == nil || e.imgui == nil || !e.imgui.live {
		return false
	}
	flags := imgui.WindowFlagsNoCollapse
	if !imgui.BeginV("Inspector", nil, flags) {
		imgui.End()
		e.SetPanelRect("Inspector", widgets.Rect{})
		return false
	}
	defer imgui.End()
	e.CaptureCurrentWindowRect("Inspector")

	ent := e.selectedEntity()
	if ent == nil {
		// Idea #1 v1 U8: when no entity is selected but a Scene is
		// the current selection target, fall through to the scene-
		// settings panel (grid spinners, default-tile picker, camera
		// config). Entity selection still takes precedence — a
		// designer who clicks an entity inside the scene sees the
		// character sheet first.
		if scene := e.selectedScene(); scene != nil {
			return RenderSceneSettings(scene, e)
		}
		imgui.TextDisabled("(no selection)")
		return false
	}
	imgui.Text(ent.Name)
	imgui.Separator()

	return i.renderEntity(e, ent)
}

// renderEntity walks the selected entity's components and emits one
// CollapsingHeader per component, with one ImGui field widget per
// FieldMetadata entry. Returns true if any widget reported a value
// change this frame.
func (i *Inspector) renderEntity(e *Editor, ent *pixelforge_project.Entity) bool {
	ctx := buildWidgetContext(e.project)
	mutated := false
	if i.renderCharacterSheet(e, ent, ctx) {
		mutated = true
	}
	for ci := range ent.Components {
		comp := &ent.Components[ci]
		md, ok := pfcomponent.Get(comp.Type)
		if !ok {
			imgui.PushIDInt(int32(ci))
			imgui.TextDisabled("(unknown component) " + comp.Type)
			imgui.PopID()
			continue
		}
		imgui.PushIDInt(int32(ci))
		if !imgui.CollapsingHeaderTreeNodeFlagsV(md.Name, imgui.TreeNodeFlagsDefaultOpen) {
			imgui.PopID()
			continue
		}
		if comp.Values == nil {
			comp.Values = map[string]any{}
		}
		for fi, f := range md.Fields {
			imgui.PushIDInt(int32(fi))
			if i.renderField(comp.Values, f, ctx) {
				mutated = true
			}
			imgui.PopID()
		}
		imgui.PopID()
	}
	return mutated
}

// renderCharacterSheet emits U8's archetype-aware character-sheet
// block: the archetype dropdown, then one row per trigger slot in the
// matching archetype template with verb-pick dropdown + "Show advanced"
// fallback. Returns true if any widget mutated the entity this frame.
//
// The block is additive — it renders before the existing per-component
// pfcomponent loop so Sprite/Position/etc. surfaces stay visible below.
// Entities with empty Archetype + empty VerbBindings still get the
// World template's minimal slots (when_scene_starts + every_tick),
// matching U2's load-time sanitize convention.
func (i *Inspector) renderCharacterSheet(e *Editor, ent *pixelforge_project.Entity, ctx widgets.Context) bool {
	mutated := false

	// ---- Archetype dropdown ---------------------------------------
	options := pixelforge_project.AllArchetypes()
	current := ent.Archetype
	if current == "" {
		current = pixelforge_project.DefaultArchetype
	}
	if renderArchetypePicker(ent, options, current) {
		e.MarkDirty()
		mutated = true
	}

	// Resolve the effective archetype the inspector will render
	// against. Unknown archetype on load falls back to World so a
	// forward-compat .pforge with a seventh archetype name still
	// surfaces something useful.
	effectiveArch := ent.Archetype
	if effectiveArch == "" {
		effectiveArch = pixelforge_project.DefaultArchetype
	}
	tmpl, ok := archetype.LookupArchetype(effectiveArch)
	if !ok {
		tmpl, _ = archetype.LookupArchetype(pixelforge_project.DefaultArchetype)
	}

	// ---- Trigger slots --------------------------------------------
	for slotIdx, slot := range tmpl.TriggerSlots {
		if i.renderTriggerSlot(e, ent, slot, slotIdx, ctx) {
			mutated = true
		}
	}

	if e.imgui != nil && e.imgui.live {
		imgui.Separator()
	}

	return mutated
}

// renderArchetypePicker emits the archetype combo and mutates
// ent.Archetype on selection change. Returns true on change. Extracted
// so tests can drive the selection logic without a live ImGui frame.
func renderArchetypePicker(ent *pixelforge_project.Entity, options []string, current string) bool {
	// Backing map keeps comboField's mutation contract uniform: the
	// helper writes the picked string into the map at the key we
	// pass, and the caller copies it back to the strongly-typed
	// entity field.
	backing := map[string]any{"archetype": current}
	if !comboField(backing, "archetype", "Archetype", options) {
		return false
	}
	picked, _ := backing["archetype"].(string)
	if picked == "" {
		picked = pixelforge_project.DefaultArchetype
	}
	if picked == ent.Archetype {
		return false
	}
	ent.Archetype = picked
	return true
}

// renderTriggerSlot renders one TriggerSlot row: label + verb dropdown
// (or read-only "advanced (composed)" label) + "Show advanced" toggle.
// Returns true if any widget mutated the entity.
func (i *Inspector) renderTriggerSlot(e *Editor, ent *pixelforge_project.Entity, slot archetype.TriggerSlot, slotIdx int, ctx widgets.Context) bool {
	mutated := false
	live := e.imgui != nil && e.imgui.live

	binding, hasBinding := findVerbBinding(ent.VerbBindings, slot.Name)

	// Advanced-composed detection: only meaningful for designer-set
	// bindings, since defaults are by construction clean recipe shapes.
	advancedComposed := false
	composedRecipe := ""
	if hasBinding {
		composedRecipe, advancedComposed = detectBindingAdvancedComposed(binding)
	}

	if live {
		imgui.PushIDInt(int32(slotIdx))
		defer imgui.PopID()
		imgui.Text(slot.DisplayName)
	}

	if advancedComposed {
		// Read-only label + hint. The slot is preserved on save; the
		// designer edits it via the scripting workspace.
		label := advancedComposedLabel
		if composedRecipe != "" {
			label = advancedComposedLabel + " (" + composedRecipe + ")"
		}
		if live {
			imgui.TextDisabled(label)
			imgui.TextDisabled("View in scripting workspace")
		}
		return false
	}

	// Effective current verb: binding wins; else slot default.
	effectiveVerb, isDefault := effectiveVerbForSlot(slot, binding, hasBinding)
	if renderVerbPicker(ent, slot, effectiveVerb, isDefault) {
		e.MarkDirty()
		mutated = true
		// Re-resolve binding after the picker mutated VerbBindings.
		binding, hasBinding = findVerbBinding(ent.VerbBindings, slot.Name)
	}

	// "Show advanced" toggle + parameter form (simplified: scalar
	// args render as text fields; complex shapes defer to v2 polish
	// in the scripting workspace).
	key := advancedKey(ent.ID, slot.Name)
	advancedOn := i.showAdvanced[key]
	if renderShowAdvancedToggle(i, key, slot.DisplayName, advancedOn) {
		// Toggle state itself is not a project mutation; don't mark
		// dirty for it.
		advancedOn = i.showAdvanced[key]
	}
	if advancedOn && hasBinding {
		if i.renderAdvancedParameterForm(ent, &binding, slot.Name) {
			e.MarkDirty()
			mutated = true
		}
	}
	_ = ctx // Context is plumbed for future intent-name dropdowns.

	return mutated
}

// renderVerbPicker renders the verb-pick combo and upserts the
// resulting VerbBinding on change. Returns true on change. Extracted
// for testability: it mutates ent.VerbBindings synchronously.
func renderVerbPicker(ent *pixelforge_project.Entity, slot archetype.TriggerSlot, currentVerb string, currentIsDefault bool) bool {
	// Dropdown options: "(unset)" first, then the slot's relevant
	// recipes. Stable order so frames don't reshuffle items mid-pick.
	items := make([]string, 0, len(slot.RelevantRecipes)+1)
	items = append(items, verbBindingUnsetLabel)
	items = append(items, slot.RelevantRecipes...)
	// If the current verb (binding or default) isn't in the relevant
	// list, prepend it so the dropdown can still surface it.
	if currentVerb != "" && !contains(items, currentVerb) {
		items = append([]string{verbBindingUnsetLabel, currentVerb}, slot.RelevantRecipes...)
	}

	display := currentVerb
	if display == "" {
		display = verbBindingUnsetLabel
	} else if currentIsDefault {
		display = currentVerb + verbBindingDefaultSuffix
	}

	backing := map[string]any{"verb": display}
	// Substitute the prettified display name for matching purposes.
	matchItems := make([]string, len(items))
	for i, it := range items {
		if it == currentVerb && currentIsDefault {
			matchItems[i] = currentVerb + verbBindingDefaultSuffix
		} else {
			matchItems[i] = it
		}
	}
	if !comboField(backing, "verb", "  verb", matchItems) {
		return false
	}
	picked, _ := backing["verb"].(string)
	// Strip the "(default)" suffix back to the raw recipe name.
	picked = strings.TrimSuffix(picked, verbBindingDefaultSuffix)
	if picked == verbBindingUnsetLabel {
		picked = ""
	}

	// Translate to the resulting VerbBindings mutation.
	switch {
	case picked == "":
		// Designer chose "(unset)" — remove any existing binding so
		// the slot falls back to the archetype's DefaultVerb.
		ent.VerbBindings = removeVerbBinding(ent.VerbBindings, slot.Name)
	case picked == slot.DefaultVerb && currentIsDefault:
		// No-op: designer re-picked the already-active default.
		return false
	default:
		ent.VerbBindings = upsertVerbBinding(ent.VerbBindings, pixelforge_project.VerbBinding{
			Trigger:    slot.Name,
			RecipeName: picked,
		})
	}
	return true
}

// renderShowAdvancedToggle emits a checkbox that flips the inspector's
// showAdvanced map entry for (entity, slot). Returns true when the
// checkbox state changed this frame. The toggle is purely transient
// UI state; it does NOT mark the project dirty.
func renderShowAdvancedToggle(i *Inspector, key, slotLabel string, current bool) bool {
	if i.showAdvanced == nil {
		i.showAdvanced = map[string]bool{}
	}
	v := current
	// Live ImGui surface: render the Checkbox. In tests the showAdvanced
	// map is driven by SetShowAdvancedForTest so the logic flow below
	// (read map, render advanced form) is exercisable without a frame.
	if imgui.Checkbox("Show advanced", &v) {
		i.showAdvanced[key] = v
		return true
	}
	return false
}

// renderAdvancedParameterForm renders the per-Action parameter form
// for the binding's recipe. v1 keeps this simple: each merged Arg
// (recipe DefaultArgs ∪ binding ArgOverrides) is rendered as a single
// input row. Numeric values use InputFloat, strings use InputText,
// bools use Checkbox. Edits persist as ArgOverrides on the binding.
// Returns true when any value changed this frame.
//
// The "complex parameter form" full pfcomponent dispatch is deferred
// per the plan's "simplified to toggle if complex" guidance — v2 polish
// can swap this for the full per-Action FieldMetadata walk once the
// catalog ships per-recipe metadata.
func (i *Inspector) renderAdvancedParameterForm(ent *pixelforge_project.Entity, binding *pixelforge_project.VerbBinding, slotName string) bool {
	recipe, ok := catalog.LookupRecipe(binding.RecipeName)
	if !ok {
		// Unknown recipe — should have been caught upstream by
		// detectBindingAdvancedComposed, but be defensive.
		return false
	}
	merged := mergeArgsMap(recipe.DefaultArgs, binding.ArgOverrides)
	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	mutated := false
	for _, k := range keys {
		def := recipe.DefaultArgs[k]
		cur := merged[k]
		// Render strategy depends on the value type.
		switch typedDefault := def.(type) {
		case float64:
			v := toFloat32(cur)
			if imgui.InputFloatV("    "+k, &v, 0, 0, "%.3f", 0) {
				newVal := float64(v)
				if newVal != typedDefault {
					setArgOverride(ent, binding, slotName, k, newVal)
					mutated = true
				} else {
					clearArgOverride(ent, binding, slotName, k)
					mutated = true
				}
			}
		case string:
			s := toString(cur)
			if imgui.InputTextWithHint("    "+k, "", &s, 0, nil) {
				if s != typedDefault {
					setArgOverride(ent, binding, slotName, k, s)
					mutated = true
				} else {
					clearArgOverride(ent, binding, slotName, k)
					mutated = true
				}
			}
		case bool:
			v := toBool(cur)
			if imgui.Checkbox("    "+k, &v) {
				if v != typedDefault {
					setArgOverride(ent, binding, slotName, k, v)
					mutated = true
				} else {
					clearArgOverride(ent, binding, slotName, k)
					mutated = true
				}
			}
		default:
			// Unknown shape — read-only fallback.
			imgui.Text("    " + k + ": " + formatFieldValue(cur))
		}
	}
	return mutated
}

// ---- VerbBinding helpers ----------------------------------------

// findVerbBinding returns the binding matching trigger and whether it
// was found. Iterates rather than maps because v1's binding count per
// entity is small (≤ 9 slots) and we want stable order for save/load.
func findVerbBinding(bindings []pixelforge_project.VerbBinding, trigger string) (pixelforge_project.VerbBinding, bool) {
	for _, b := range bindings {
		if b.Trigger == trigger {
			return b, true
		}
	}
	return pixelforge_project.VerbBinding{}, false
}

// upsertVerbBinding inserts or replaces the binding for b.Trigger.
// Returns a fresh slice so callers may safely reassign without aliasing.
func upsertVerbBinding(bindings []pixelforge_project.VerbBinding, b pixelforge_project.VerbBinding) []pixelforge_project.VerbBinding {
	for i := range bindings {
		if bindings[i].Trigger == b.Trigger {
			bindings[i] = b
			return bindings
		}
	}
	return append(bindings, b)
}

// removeVerbBinding drops any binding matching trigger and returns the
// resulting slice. Returns nil when removing the last entry to keep
// JSON round-trip clean (`omitempty` collapses nil but not empty).
func removeVerbBinding(bindings []pixelforge_project.VerbBinding, trigger string) []pixelforge_project.VerbBinding {
	out := bindings[:0]
	for _, b := range bindings {
		if b.Trigger != trigger {
			out = append(out, b)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// effectiveVerbForSlot returns (verbName, isDefault) for the slot's
// current dropdown selection. Designer-set binding wins; otherwise the
// slot's DefaultVerb shows through and isDefault is true.
func effectiveVerbForSlot(slot archetype.TriggerSlot, binding pixelforge_project.VerbBinding, hasBinding bool) (string, bool) {
	if hasBinding {
		return binding.RecipeName, false
	}
	return slot.DefaultVerb, true
}

// detectBindingAdvancedComposed compiles a single VerbBinding through
// U6's CompileVerbBindings + DetectRoundtripRecipe to determine
// whether the binding represents an "advanced (composed)" shape — an
// unknown recipe name or a rule that no longer round-trips to a clean
// recipe form. Returns (recipeNameHint, true) when advanced, ("", false)
// when the binding maps cleanly back to its recipe.
//
// The single-binding wrapper here lets the inspector check each slot
// independently without recompiling the entity's whole rule list.
func detectBindingAdvancedComposed(binding pixelforge_project.VerbBinding) (string, bool) {
	// Build a transient single-binding entity so we can reuse U6's
	// CompileVerbBindings without duplicating its trigger → target
	// translation.
	ent := &pixelforge_project.Entity{
		ID:           "__inspect__",
		VerbBindings: []pixelforge_project.VerbBinding{binding},
	}
	rules, err := runtime.CompileVerbBindings(ent, archetype.Template{})
	if err != nil || len(rules) != 1 {
		return binding.RecipeName, true
	}
	name, _, advanced := runtime.DetectRoundtripRecipe(rules[0])
	if advanced {
		if name == "" {
			name = binding.RecipeName
		}
		return name, true
	}
	return "", false
}

// setArgOverride writes (key, val) into the binding's ArgOverrides
// and persists the updated binding back into ent.VerbBindings. Lazy-
// allocates the map.
func setArgOverride(ent *pixelforge_project.Entity, binding *pixelforge_project.VerbBinding, trigger, key string, val any) {
	if binding.ArgOverrides == nil {
		binding.ArgOverrides = map[string]any{}
	}
	binding.ArgOverrides[key] = val
	ent.VerbBindings = upsertVerbBinding(ent.VerbBindings, *binding)
}

// clearArgOverride drops key from the binding's ArgOverrides (so the
// recipe default takes effect again) and persists.
func clearArgOverride(ent *pixelforge_project.Entity, binding *pixelforge_project.VerbBinding, trigger, key string) {
	if binding.ArgOverrides == nil {
		return
	}
	delete(binding.ArgOverrides, key)
	if len(binding.ArgOverrides) == 0 {
		binding.ArgOverrides = nil
	}
	ent.VerbBindings = upsertVerbBinding(ent.VerbBindings, *binding)
}

// mergeArgsMap returns a fresh map carrying defaults overlaid with
// overrides. Mirrors the catalog's mergeArgs helper, duplicated here
// to avoid importing the unexported version.
func mergeArgsMap(defaults, overrides map[string]any) map[string]any {
	out := make(map[string]any, len(defaults)+len(overrides))
	for k, v := range defaults {
		out[k] = v
	}
	for k, v := range overrides {
		out[k] = v
	}
	return out
}

// renderField dispatches one FieldMetadata to its matching ImGui
// widget, reading the current value from values[f.JSONKey] and
// writing back the edited value when ImGui reports a change.
func (i *Inspector) renderField(values map[string]any, f pfcomponent.FieldMetadata, ctx widgets.Context) bool {
	label := f.Name
	switch f.WidgetKind {
	case pfcomponent.WidgetSlider:
		v := toFloat32(values[f.JSONKey])
		min, max := float32(f.Min), float32(f.Max)
		if min == 0 && max == 0 {
			min, max = 0, 1
		}
		if imgui.SliderFloatV(label, &v, min, max, "%.3f", 0) {
			values[f.JSONKey] = float64(v)
			return true
		}
	case pfcomponent.WidgetFloatField:
		v := toFloat32(values[f.JSONKey])
		if imgui.InputFloatV(label, &v, 0, 0, "%.3f", 0) {
			values[f.JSONKey] = float64(v)
			return true
		}
	case pfcomponent.WidgetIntField:
		v := toInt32(values[f.JSONKey])
		if imgui.InputIntV(label, &v, 1, 10, 0) {
			values[f.JSONKey] = int(v)
			return true
		}
	case pfcomponent.WidgetCheckbox:
		v := toBool(values[f.JSONKey])
		if imgui.Checkbox(label, &v) {
			values[f.JSONKey] = v
			return true
		}
	case pfcomponent.WidgetText:
		v := toString(values[f.JSONKey])
		if imgui.InputTextWithHint(label, "", &v, 0, nil) {
			values[f.JSONKey] = v
			return true
		}
	case pfcomponent.WidgetVector2:
		x, y := toVec2(values[f.JSONKey])
		arr := [2]float32{x, y}
		if imgui.InputFloat2V(label, &arr, "%.3f", 0) {
			values[f.JSONKey] = map[string]any{"x": float64(arr[0]), "y": float64(arr[1])}
			return true
		}
	case pfcomponent.WidgetPaletteColor:
		v := toInt32(values[f.JSONKey])
		if imgui.InputIntV(label+" (palette slot)", &v, 1, 8, 0) {
			if v < 0 {
				v = 0
			}
			values[f.JSONKey] = int(v)
			return true
		}
	case pfcomponent.WidgetSpriteRef:
		return comboField(values, f.JSONKey, label, ctx.SpriteNames)
	case pfcomponent.WidgetAudioRef:
		return comboField(values, f.JSONKey, label, ctx.AudioNames)
	case pfcomponent.WidgetEventTopic:
		return comboField(values, f.JSONKey, label, ctx.EventTopics)
	case pfcomponent.WidgetEnum:
		return comboField(values, f.JSONKey, label, f.Options)
	default:
		// WidgetDefault / WidgetUnknown — render as read-only text
		// so newer-schema fields still surface their stored value.
		imgui.Text(label + ": " + formatFieldValue(values[f.JSONKey]))
	}
	return false
}

// comboField renders a Combo over options, persists the picked
// string into values[key], and returns true on selection change. A
// blank current value is auto-prepended so "no selection" is a
// legitimate choice.
func comboField(values map[string]any, key, label string, options []string) bool {
	current := toString(values[key])
	items := options
	if current == "" {
		items = append([]string{"(none)"}, options...)
	} else if !contains(options, current) {
		items = append([]string{current}, options...)
	}
	selected := indexOf(items, current)
	if selected < 0 {
		selected = 0
	}
	sel32 := int32(selected)
	if imgui.ComboStrarrV(label, &sel32, items, int32(len(items)), int32(len(items))) {
		picked := items[sel32]
		if picked == "(none)" {
			picked = ""
		}
		values[key] = picked
		return true
	}
	return false
}

// buildWidgetContext snapshots the project's catalogs into the form
// the inspector's combo widgets expect. Sorted lists make the
// dropdown order deterministic and stable across frames.
//
// U8 extends the Context with two cross-project catalogs that don't
// depend on the active *Project but ride with it for ergonomic reasons:
// the input-intent topic list (so any future intent-name dropdowns
// inside verb parameter forms have something to enumerate) and the
// global verb-recipe registry (so the character-sheet inspector can
// filter per-trigger-slot at render time). Both are populated
// unconditionally — they're cheap to gather and the inspector pays
// no draw cost when it doesn't reach for them.
func buildWidgetContext(p *pixelforge_project.Project) widgets.Context {
	ctx := widgets.Context{
		Intents:     append([]string(nil), pixelforge_input.AllIntents()...),
		VerbRecipes: append([]string(nil), catalog.AllRecipes()...),
	}
	sort.Strings(ctx.Intents)
	if p == nil {
		return ctx
	}
	for _, s := range p.Sprites {
		ctx.SpriteNames = append(ctx.SpriteNames, s.Name)
	}
	for _, a := range p.Audio {
		ctx.AudioNames = append(ctx.AudioNames, a.Name)
	}
	for _, sub := range p.EventSubscriptions {
		if sub.Topic != "" {
			ctx.EventTopics = append(ctx.EventTopics, sub.Topic)
		}
	}
	sort.Strings(ctx.SpriteNames)
	sort.Strings(ctx.AudioNames)
	sort.Strings(ctx.EventTopics)
	for _, c := range p.Palette.Base {
		ctx.PaletteColors = append(ctx.PaletteColors, string(c))
	}
	return ctx
}

// formatFieldValue renders a stored map[string]any value as a string
// for the read-only Default/Unknown widget fallback.
func formatFieldValue(v any) string {
	if v == nil {
		return "-"
	}
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		return fmt.Sprintf("%.2f", x)
	case int:
		return fmt.Sprintf("%d", x)
	case map[string]any:
		if xv, ok := x["x"]; ok {
			if yv, ok2 := x["y"]; ok2 {
				return fmt.Sprintf("%v,%v", xv, yv)
			}
		}
		return fmt.Sprintf("%v", x)
	default:
		return fmt.Sprintf("%v", x)
	}
}

// ---- value coercion helpers -------------------------------------

// toFloat32 widens stored numeric values (typically float64 from JSON
// unmarshalling) to ImGui's float32 widget signature.
func toFloat32(v any) float32 {
	switch x := v.(type) {
	case float64:
		return float32(x)
	case float32:
		return x
	case int:
		return float32(x)
	case int32:
		return float32(x)
	case int64:
		return float32(x)
	case string:
		var f float64
		fmt.Sscanf(x, "%f", &f)
		return float32(f)
	default:
		return 0
	}
}

// toInt32 narrows stored numeric values to ImGui's int32 widget
// signature.
func toInt32(v any) int32 {
	switch x := v.(type) {
	case int:
		return int32(x)
	case int32:
		return x
	case int64:
		return int32(x)
	case float64:
		return int32(x)
	case float32:
		return int32(x)
	default:
		return 0
	}
}

// toBool coerces stored truthy values to bool.
func toBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return strings.EqualFold(x, "true")
	case float64:
		return x != 0
	case int:
		return x != 0
	default:
		return false
	}
}

// toString coerces stored values to their string form.
func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", x)
	}
}

// toVec2 extracts (x, y) from a stored map[string]any vector. Falls
// back to (0, 0) when the stored shape isn't a {x, y} map.
func toVec2(v any) (float32, float32) {
	m, ok := v.(map[string]any)
	if !ok {
		return 0, 0
	}
	return toFloat32(m["x"]), toFloat32(m["y"])
}

// contains reports whether s contains target.
func contains(s []string, target string) bool {
	for _, v := range s {
		if v == target {
			return true
		}
	}
	return false
}

// indexOf returns target's position in s, or -1.
func indexOf(s []string, target string) int {
	for i, v := range s {
		if v == target {
			return i
		}
	}
	return -1
}
