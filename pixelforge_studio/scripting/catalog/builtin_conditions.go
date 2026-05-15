package catalog

import (
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
)

func init() {
	RegisterCondition("event_fired", buildEventFired)
	RegisterCondition("key_held", buildKeyHeld)
	RegisterCondition("value_lt", buildValueLt)
	RegisterCondition("value_gt", buildValueGt)
	RegisterCondition("value_eq", buildValueEq)
}

func buildEventFired(args map[string]any, _ Context) (Predicate, error) {
	wantEvent, err := ArgString(args, "event")
	if err != nil {
		return nil, err
	}
	return func(payload any) bool {
		switch p := payload.(type) {
		case string:
			return p == wantEvent
		case fmt_Stringer:
			return p.String() == wantEvent
		default:
			return false
		}
	}, nil
}

// fmt_Stringer is the standard fmt.Stringer interface in a local
// alias so we don't add an import to fmt for one line.
type fmt_Stringer interface{ String() string }

func buildKeyHeld(args map[string]any, _ Context) (Predicate, error) {
	keyName, err := ArgString(args, "key")
	if err != nil {
		return nil, err
	}
	return func(_ any) bool {
		return pixelforge_key.Duration(pixelforge_key.Key(keyName)) > 0
	}, nil
}

func buildValueLt(args map[string]any, ctx Context) (Predicate, error) {
	return buildValueCompare(args, ctx, func(a, b float64) bool { return a < b })
}

func buildValueGt(args map[string]any, ctx Context) (Predicate, error) {
	return buildValueCompare(args, ctx, func(a, b float64) bool { return a > b })
}

func buildValueEq(args map[string]any, ctx Context) (Predicate, error) {
	return buildValueCompare(args, ctx, func(a, b float64) bool { return a == b })
}

func buildValueCompare(args map[string]any, ctx Context, cmp func(a, b float64) bool) (Predicate, error) {
	left, err := ArgString(args, "value")
	if err != nil {
		return nil, err
	}
	right, err := ArgFloat(args, "compare_to")
	if err != nil {
		return nil, err
	}
	return func(_ any) bool {
		if ctx == nil {
			return false
		}
		ref := ctx.ValueRef(left)
		if ref == nil {
			return false
		}
		v := ref.Get()
		f, ok := toFloat(v)
		if !ok {
			return false
		}
		return cmp(f, right)
	}, nil
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
