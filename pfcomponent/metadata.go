// Package pfcomponent is the reflection-driven component registry that
// powers Pixelforge Studio's auto-generated inspector, undo entries,
// serialization, and visual-script node catalog.
//
// Game authors register their component structs once:
//
//	pfcomponent.Register[Player]("Player")
//
// and the editor inspects the struct's pf:"..." tags to decide which
// widget to render per field. The same metadata feeds visual scripting
// (auto-generated Get/Set nodes) and serialization (round-trip via
// reflection).
package pfcomponent

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// WidgetKind names the inspector widget used to render a field. The
// editor's widgets package switches on this; visual scripting uses it
// to filter Get/Set node generation.
type WidgetKind string

const (
	WidgetDefault      WidgetKind = "default"
	WidgetUnknown      WidgetKind = "unknown"
	WidgetSlider       WidgetKind = "slider"
	WidgetPaletteColor WidgetKind = "color"
	WidgetSpriteRef    WidgetKind = "sprite"
	WidgetAudioRef     WidgetKind = "audio"
	WidgetEventTopic   WidgetKind = "event"
	WidgetEnum         WidgetKind = "enum"
	WidgetText         WidgetKind = "text"
	WidgetVector2      WidgetKind = "vector2"
	WidgetIntField     WidgetKind = "int_field"
	WidgetFloatField   WidgetKind = "float_field"
	WidgetCheckbox     WidgetKind = "checkbox"
)

// FieldMetadata describes one field on a registered component as far as
// the editor needs to know. Built once at Register time from the
// struct's field tags + Go type.
type FieldMetadata struct {
	// Name is the Go field name. JSON keys derive from this via the
	// usual lowercase-first-letter rule unless the json: tag overrides.
	Name string

	// JSONKey is the wire-format key for this field. Pre-resolved so
	// the inspector doesn't have to consult json tags at render time.
	JSONKey string

	// Type is the field's reflect.Type. Carried so widgets can do
	// type-aware fallbacks (e.g. int vs uint vs float).
	Type reflect.Type

	WidgetKind WidgetKind

	// Min / Max bound numeric and text widgets. Zero when unbounded.
	Min float64
	Max float64

	// Options enumerate values for WidgetEnum. Empty for other kinds.
	Options []string

	// Required marks a field that must be set before Save. Reserved
	// for future validation passes; the M1 saver does not enforce.
	Required bool

	// Index is the field's position inside the struct; used by the
	// reflection marshal/unmarshal helpers to locate the value.
	Index int

	// Anonymous reports whether this field came from an embedded
	// struct. Inspector treats anonymous fields like flattened
	// composites — the parent's fields list already includes them.
	Anonymous bool
}

// TypeMetadata is the full per-type registry entry.
type TypeMetadata struct {
	// Name is the registered type identifier (e.g. "Player"). Used as
	// EntityComponent.Type in projects.
	Name string

	// Type is the Go type. Always a struct.
	Type reflect.Type

	// Fields enumerates the inspectable fields in source order. Embedded
	// struct fields are inlined here so the inspector renders one flat
	// list per component.
	Fields []FieldMetadata
}

// parseTypeMetadata walks a struct type and returns its FieldMetadata
// list. Embedded structs flatten into the parent's list. Panics on
// developer mistakes (slider min > max, etc.) — registration is a
// build-time discipline, so failing fast is the right behaviour.
func parseTypeMetadata(name string, t reflect.Type) (TypeMetadata, error) {
	if t.Kind() != reflect.Struct {
		return TypeMetadata{}, fmt.Errorf("pfcomponent: %s: only struct types are registerable, got %v", name, t.Kind())
	}
	md := TypeMetadata{Name: name, Type: t}
	fields, err := collectFields(t, "")
	if err != nil {
		return TypeMetadata{}, err
	}
	md.Fields = fields
	return md, nil
}

// collectFields walks a struct (recursing into anonymous fields) and
// emits one FieldMetadata per inspectable field. Unexported fields and
// json:"-" fields are skipped.
func collectFields(t reflect.Type, _ string) ([]FieldMetadata, error) {
	var out []FieldMetadata
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		jsonTag := f.Tag.Get("json")
		jsonKey, jsonOpts := splitJSONTag(jsonTag)
		if jsonKey == "-" {
			continue
		}
		// Anonymous (embedded) struct: flatten its fields into ours.
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			children, err := collectFields(f.Type, "")
			if err != nil {
				return nil, err
			}
			for j := range children {
				children[j].Anonymous = true
			}
			out = append(out, children...)
			continue
		}
		if jsonKey == "" {
			jsonKey = defaultJSONKey(f.Name)
		}
		_ = jsonOpts // omitempty etc. — not consumed by the inspector

		md := FieldMetadata{
			Name:    f.Name,
			JSONKey: jsonKey,
			Type:    f.Type,
			Index:   i,
		}
		if err := applyPFTag(&md, f.Tag.Get("pf")); err != nil {
			return nil, fmt.Errorf("pfcomponent: field %s: %w", f.Name, err)
		}
		if md.WidgetKind == "" {
			md.WidgetKind = widgetFromType(f.Type)
		}
		out = append(out, md)
	}
	return out, nil
}

// applyPFTag parses one pf:"..." struct tag and populates the matching
// fields on md. Returns an error for syntactically invalid tags
// (slider min > max) so Register can panic with a useful message.
func applyPFTag(md *FieldMetadata, tag string) error {
	if tag == "" {
		return nil
	}
	parts := strings.SplitN(tag, ",", 2)
	kind := strings.TrimSpace(parts[0])
	rest := ""
	if len(parts) > 1 {
		rest = strings.TrimSpace(parts[1])
	}
	switch kind {
	case "slider":
		md.WidgetKind = WidgetSlider
		if rest == "" {
			return nil
		}
		minV, maxV, err := parseRange(rest)
		if err != nil {
			return err
		}
		if minV > maxV {
			return fmt.Errorf("slider min %v > max %v", minV, maxV)
		}
		md.Min, md.Max = minV, maxV
	case "color":
		md.WidgetKind = WidgetPaletteColor
	case "sprite":
		md.WidgetKind = WidgetSpriteRef
	case "audio":
		md.WidgetKind = WidgetAudioRef
	case "event":
		md.WidgetKind = WidgetEventTopic
	case "enum":
		md.WidgetKind = WidgetEnum
		if rest == "" {
			return fmt.Errorf("enum widget requires at least one option")
		}
		md.Options = splitEnumOptions(rest)
	case "text":
		md.WidgetKind = WidgetText
		md.Max = parseMaxLen(rest)
	case "vector2":
		md.WidgetKind = WidgetVector2
	default:
		// Unknown tags are recorded with WidgetUnknown so newer schemas
		// open in older editors without losing the value.
		md.WidgetKind = WidgetUnknown
	}
	return nil
}

// parseRange handles "0..100" and "0.0..1.0" forms.
func parseRange(s string) (float64, float64, error) {
	idx := strings.Index(s, "..")
	if idx < 0 {
		return 0, 0, fmt.Errorf("slider range %q must use min..max syntax", s)
	}
	lo, err := strconv.ParseFloat(strings.TrimSpace(s[:idx]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("slider min %q: %w", s[:idx], err)
	}
	hi, err := strconv.ParseFloat(strings.TrimSpace(s[idx+2:]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("slider max %q: %w", s[idx+2:], err)
	}
	return lo, hi, nil
}

func splitEnumOptions(s string) []string {
	raw := strings.Split(s, "|")
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		r = strings.TrimSpace(r)
		if r != "" {
			out = append(out, r)
		}
	}
	return out
}

// parseMaxLen extracts the "maxlen=N" sub-argument. Returns 0 when
// unspecified, which the text widget treats as "unbounded".
func parseMaxLen(s string) float64 {
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "maxlen=") {
			n, err := strconv.Atoi(part[len("maxlen="):])
			if err == nil {
				return float64(n)
			}
		}
	}
	return 0
}

// splitJSONTag splits a json:"name,omitempty,..." tag into name + opts.
func splitJSONTag(t string) (string, []string) {
	if t == "" {
		return "", nil
	}
	parts := strings.Split(t, ",")
	return parts[0], parts[1:]
}

// defaultJSONKey mirrors encoding/json's default: the Go field name
// verbatim (Go std lib does NOT lowercase by default; we follow suit so
// the schema stays explicit).
func defaultJSONKey(goName string) string {
	return goName
}

// widgetFromType picks a sensible widget for an untagged field based
// on its Go type. Booleans → Checkbox, floats → FloatField, ints →
// IntField, strings → Text; anything else falls through to Default.
func widgetFromType(t reflect.Type) WidgetKind {
	switch t.Kind() {
	case reflect.Bool:
		return WidgetCheckbox
	case reflect.Float32, reflect.Float64:
		return WidgetFloatField
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return WidgetIntField
	case reflect.String:
		return WidgetText
	default:
		return WidgetDefault
	}
}
