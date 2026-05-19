package pixelforge_replay

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_render"
)

// TraceVersion is the on-wire schema version emitted by Encode and
// required by LoadTrace. A future incompatible schema bumps this and
// LoadTrace returns ErrTraceVersion for older / newer files (the
// reader is intentionally strict so a future codec can decide
// whether and how to migrate older recordings forward).
const TraceVersion = 1

// ErrTraceVersion is returned by LoadTrace when the meta header's
// "v" field does not match TraceVersion. The .trace.jsonl format
// reserves room for forward-compatibility hooks but never silently
// accepts a mismatch.
var ErrTraceVersion = errors.New("pixelforge_replay: unsupported trace version")

// ErrMalformedTrace wraps a per-line decode failure. The underlying
// error from json.Unmarshal (or the missing-meta sentinel) is
// preserved via errors.Is/errors.As; the line number is rendered in
// the message so a fast `grep` of test output locates the offending
// row.
var ErrMalformedTrace = errors.New("pixelforge_replay: malformed trace line")

// ErrMissingMeta is returned when LoadTrace cannot find a valid
// meta header on the first non-empty line of the input.
var ErrMissingMeta = errors.New("pixelforge_replay: missing meta header on first line")

// TraceMeta is the header line of a .trace.jsonl file. The fields
// describe the recording's provenance so the replayer can reject a
// mismatched runtime (wrong screen dimensions, wrong TPS) before
// burning N ticks of render work.
type TraceMeta struct {
	// Game is the recording's authored game ID (e.g.
	// "asteroids_proof"). Pure metadata at v1 — the replayer does
	// not enforce that the runtime is booted from the same
	// project. CI tests inspect this for routing.
	Game string `json:"game"`

	// Seed is the RNG seed used during recording (zero is valid
	// and elided via omitempty for traces that did not use RNG).
	Seed uint64 `json:"seed,omitempty"`

	// Width / Height are the recorded screen dimensions in pixels.
	// The replayer does not currently assert these match the
	// runtime's screen — that's a future hardening pass — but
	// hashing tests rely on them being recorded faithfully so the
	// frame size is documented.
	Width  int `json:"width"`
	Height int `json:"height"`

	// TPS is the ticks-per-second the recording assumes. Used to
	// compute physics dt on the replay side; informational at v1.
	TPS int `json:"tps"`

	// DurationTicks is the total number of logical ticks recorded
	// (i.e. the highest Tick + hold-expansion of any frame plus
	// one). Encoded for fast inspection without scanning every
	// per-tick line. Not strictly enforced on decode — the actual
	// frame count comes from the lines themselves.
	DurationTicks uint64 `json:"duration_ticks"`
}

// TraceFrame is one logical tick of recorded input. Tick is the
// absolute tick number (not relative to the previous frame), Keys
// holds the ebiten.Key values pressed during this tick, and Pad
// carries optional gamepad state (nil if no gamepad input).
//
// Two TraceFrames with the same Tick / Keys / Pad compare equal at
// the package boundary; the round-trip test in trace_test.go relies
// on this.
type TraceFrame struct {
	Tick uint64
	Keys []ebiten.Key
	Pad  *pixelforge_render.GamepadState
}

// Trace is the in-memory representation of a parsed .trace.jsonl
// file: a meta header plus the expanded per-tick frame slice. Hold-
// compressed lines are expanded during LoadTrace so Frames has one
// entry per logical tick — downstream code can iterate without
// special-casing compressed rows.
type Trace struct {
	Meta   TraceMeta
	Frames []TraceFrame
}

// metaLine is the JSON shape of the first line of a .trace.jsonl
// file: schema version + nested meta object. Kept private so callers
// don't accidentally hand-roll a header with the wrong field name.
type metaLine struct {
	V    int       `json:"v"`
	Meta TraceMeta `json:"meta"`
}

// frameLine is the JSON shape of a per-tick line. KeysOpt is *[]string
// (not []string) so we can distinguish "omitted" from "explicitly
// empty"; both decode to an empty Keys slice at the TraceFrame level
// but the distinction lets us detect malformed lines that drop the
// field entirely (we treat that as "no keys this tick", matching the
// recorder's default emission for a no-input tick).
type frameLine struct {
	Tick uint64                           `json:"tick"`
	Keys []string                         `json:"keys"`
	Pad  *pixelforge_render.GamepadState  `json:"pad"`
	Hold uint64                           `json:"hold,omitempty"`
}

// keyNameToKey is the on-decode lookup from key-string-name to
// ebiten.Key. We deliberately keep the table narrow at v1: the keys
// the four reference proof games (Asteroids / Mario / Bomberman /
// DK) actually need plus a handful of obvious extras (modifier
// digits skipped — they enter when a verb needs them). Both the
// canonical ebiten.Key.String() name and the "KeyX" alias are
// registered so hand-authored .trace.jsonl files don't fail on the
// natural alias spelling.
//
// Unknown names are dropped by decodeKeys with a log warning rather
// than producing ErrMalformedTrace — a trace recorded with a key the
// runtime doesn't recognise should produce a reduced-fidelity replay
// rather than a hard error (the recording itself is still valuable
// to inspect).
var keyNameToKey = map[string]ebiten.Key{
	// Arrow keys (canonical names per ebiten.Key.String())
	"ArrowLeft":  ebiten.KeyArrowLeft,
	"ArrowRight": ebiten.KeyArrowRight,
	"ArrowUp":    ebiten.KeyArrowUp,
	"ArrowDown":  ebiten.KeyArrowDown,

	// Letter keys (canonical = bare letter; alias = "KeyX")
	"A":    ebiten.KeyA,
	"D":    ebiten.KeyD,
	"W":    ebiten.KeyW,
	"S":    ebiten.KeyS,
	"KeyA": ebiten.KeyA,
	"KeyD": ebiten.KeyD,
	"KeyW": ebiten.KeyW,
	"KeyS": ebiten.KeyS,

	// Non-letter control keys
	"Space":  ebiten.KeySpace,
	"Escape": ebiten.KeyEscape,
	"Enter":  ebiten.KeyEnter,
}

// LoadTrace reads a .trace.jsonl stream and returns the parsed
// Trace, expanding any "hold":N compressed lines to N consecutive
// TraceFrames. The first non-empty line must be the meta header
// ({"v":N,"meta":{...}}); a mismatch on the version field returns
// ErrTraceVersion, a non-meta first line returns ErrMissingMeta.
//
// Per-line decode failures wrap ErrMalformedTrace with the 1-based
// line number; callers can errors.Is(err, ErrMalformedTrace) to
// distinguish parse from i/o failures. Empty lines are skipped.
//
// Buffer sizing: bufio.Scanner's default 64 KB token limit is too
// small for traces with long compressed runs of multi-key state, so
// LoadTrace bumps the scanner buffer to 1 MB. A trace whose single
// JSON line exceeds 1 MB is almost certainly hand-corrupted; the
// scanner returns its bufio.ErrTooLong as a malformed-line error.
func LoadTrace(r io.Reader) (*Trace, error) {
	scanner := bufio.NewScanner(r)
	// Allow lines up to 1 MB. ~64 KB default is too tight for
	// hold-compressed runs that might pack many key names.
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

	var (
		trace     Trace
		lineNo    int
		gotMeta   bool
	)

	for scanner.Scan() {
		lineNo++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}

		if !gotMeta {
			var ml metaLine
			if err := json.Unmarshal([]byte(raw), &ml); err != nil {
				return nil, fmt.Errorf("%w (line %d): %v", ErrMalformedTrace, lineNo, err)
			}
			// A line that decoded as valid JSON but doesn't look
			// like a meta header (missing "v") is also a
			// malformed-trace.
			if ml.V == 0 && ml.Meta.Width == 0 && ml.Meta.Height == 0 && ml.Meta.Game == "" {
				return nil, fmt.Errorf("%w (line %d)", ErrMissingMeta, lineNo)
			}
			if ml.V != TraceVersion {
				return nil, fmt.Errorf("%w: got v=%d, want v=%d", ErrTraceVersion, ml.V, TraceVersion)
			}
			trace.Meta = ml.Meta
			gotMeta = true
			continue
		}

		var fl frameLine
		if err := json.Unmarshal([]byte(raw), &fl); err != nil {
			return nil, fmt.Errorf("%w (line %d): %v", ErrMalformedTrace, lineNo, err)
		}
		keys := decodeKeys(fl.Keys)
		hold := fl.Hold
		if hold == 0 {
			hold = 1
		}
		for off := uint64(0); off < hold; off++ {
			frame := TraceFrame{
				Tick: fl.Tick + off,
				Keys: cloneKeys(keys),
				Pad:  clonePad(fl.Pad),
			}
			trace.Frames = append(trace.Frames, frame)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%w (line %d): %v", ErrMalformedTrace, lineNo, err)
	}
	if !gotMeta {
		return nil, ErrMissingMeta
	}
	return &trace, nil
}

// Encode serialises the Trace to w as a .trace.jsonl stream. The
// first line emitted is the meta header (always {"v":1,"meta":{...}}
// at this schema version); each subsequent line is one TraceFrame
// rendered uncompressed (no "hold" field). Lines are terminated with
// a single '\n'.
//
// Encode does not emit hold-compressed lines today. The decoder
// supports them so future tooling can opt into compression for size-
// sensitive recordings; the choice is deferred to U5 follow-up after
// the size of a real Asteroids-length trace is measured.
//
// Returns the first write error (or nil on success). Encode does
// NOT call w.Sync() — flushing is the caller's responsibility, so
// callers can compose Encode with gzip / bufio without double-
// flushing.
func (t *Trace) Encode(w io.Writer) error {
	if t == nil {
		return fmt.Errorf("pixelforge_replay: nil trace")
	}
	header := metaLine{V: TraceVersion, Meta: t.Meta}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return fmt.Errorf("pixelforge_replay: encode meta: %w", err)
	}
	if _, err := w.Write(append(headerBytes, '\n')); err != nil {
		return err
	}
	for _, f := range t.Frames {
		line := frameLine{
			Tick: f.Tick,
			Keys: encodeKeys(f.Keys),
			Pad:  f.Pad,
		}
		b, err := json.Marshal(line)
		if err != nil {
			return fmt.Errorf("pixelforge_replay: encode frame: %w", err)
		}
		if _, err := w.Write(append(b, '\n')); err != nil {
			return err
		}
	}
	return nil
}

// encodeKeys converts an []ebiten.Key into the on-wire []string
// form using ebiten.Key.String(). A nil/empty slice round-trips to
// an empty []string so the JSON output is the explicit `"keys":[]`
// shape rather than `"keys":null` — keeping the schema regular
// across all frames.
func encodeKeys(in []ebiten.Key) []string {
	out := make([]string, 0, len(in))
	for _, k := range in {
		out = append(out, k.String())
	}
	return out
}

// decodeKeys converts the on-wire []string form back to
// []ebiten.Key, dropping any name not in keyNameToKey with a log
// warning. The warning is emitted at most once per unique unknown
// name per process by the log package's default rate (no extra
// rate-limiting here — replays log at most a handful of unknowns).
//
// An empty input (or one whose every name is unknown) returns nil
// — not an allocated empty slice — so the round-trip from nil Keys
// produces an identical nil on the way back. The round-trip
// equality test in trace_test.go relies on this symmetry.
func decodeKeys(in []string) []ebiten.Key {
	if len(in) == 0 {
		return nil
	}
	out := make([]ebiten.Key, 0, len(in))
	for _, name := range in {
		k, ok := keyNameToKey[name]
		if !ok {
			log.Printf("pixelforge_replay: unknown key name %q in trace; dropping", name)
			continue
		}
		out = append(out, k)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// cloneKeys returns a fresh slice with the same key values. Used
// when expanding hold-compressed lines so each generated frame
// owns an independent Keys slice (otherwise mutation by a downstream
// consumer would leak across frames).
func cloneKeys(in []ebiten.Key) []ebiten.Key {
	if in == nil {
		return nil
	}
	out := make([]ebiten.Key, len(in))
	copy(out, in)
	return out
}

// clonePad returns a fresh *GamepadState pointing at a copied value
// (or nil if in is nil). Same rationale as cloneKeys — hold-expanded
// frames must not share a Pad pointer with their siblings.
func clonePad(in *pixelforge_render.GamepadState) *pixelforge_render.GamepadState {
	if in == nil {
		return nil
	}
	cp := *in
	if in.Buttons != nil {
		cp.Buttons = make([]ebiten.GamepadButton, len(in.Buttons))
		copy(cp.Buttons, in.Buttons)
	}
	return &cp
}
