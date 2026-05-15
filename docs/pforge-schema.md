# `.pforge` Schema Reference (v1)

The `.pforge` file is a single JSON document describing every part of a
Pixelforge project — screen size, palette + 4 ColorTables, sprites and
animations, scenes and their entities, audio bindings, behavior graphs,
and event subscriptions. The runtime loads it via `pixelforge_project`
and the editor mutates the same in-memory struct.

Schema version: **1**. Older versions cannot exist; newer versions
trigger a forward migration via `pixelforge_project.MigrateForward`.

## Top-level shape

```json
{
  "schema_version": 1,
  "name": "snake",
  "screen_width": 128,
  "screen_height": 128,
  "tps": 30,
  "created_at": "2026-05-15T12:00:00Z",
  "modified_at": "2026-05-15T12:00:00Z",
  "palette": { ... },
  "sprites": [ ... ],
  "audio": [ ... ],
  "scenes": [ ... ],
  "behaviors": [ ... ],
  "bindings": [ ... ],
  "event_subscriptions": [ ... ],
  "extension_hooks": [ ... ]
}
```

All array fields are non-null: a freshly-created project emits `"sprites": []`
rather than `null`. Field order is preserved across save/reload, so
two saves of the same in-memory project produce byte-identical files
(the git-merge-friendly invariant).

## On-disk layout

```
my-game.pforge          # the project file (this document)
my-game.pforge-assets/  # sibling directory
  sprites/*.png         # source PNGs
  audio/*.wav           # source WAVs
  palettes/*.gpl        # optional Aseprite-compatible palette dumps
```

Sprite and audio entries reference assets via relative paths
(`"sprites/snake.png"`) interpreted under the sibling assets directory.

## Section reference

### `palette`

```json
{
  "base":           ["#000000", "#040404", ...],   // 64 entries
  "color_tables":   [ [...], [...], [...], [...] ], // 4 × 64 × 64 indices
  "presets":        [ { "name": "Dawn", ... } ],
  "animations":     [ { "slot": 8, "keyframes": [...], ... } ]
}
```

Each `color_tables[t][src][dst]` is the palette index Pixelforge
substitutes when drawing `src` over `dst` using ColorTable `t`. The
engine selects a table at draw time via `(source | target) >> 6`.

`presets` are Lightroom-style non-destructive overlays: they list
overrides on top of the base palette/color-tables, toggled live by the
editor. `animations` drive a single palette slot through keyframes,
triggered by an event or by scene start.

### `sprites`

```json
{
  "name": "fruit",
  "relative_path": "sprites/sprites.png",
  "width": 32, "height": 8,
  "frame_w": 8, "frame_h": 8,
  "origin_x": 0, "origin_y": 0,
  "collision_mask": null,
  "animations": [
    { "name": "idle", "frames": [0,1,2,3], "fps": 8, "loop_mode": "loop" }
  ]
}
```

### `audio`

```json
{
  "name": "eat",
  "relative_path": "audio/eat.wav",
  "suggested_channel_priority": "sfx",
  "loop": false,
  "sample_rate_hz": 22050
}
```

`suggested_channel_priority` is `"bgm"`, `"sfx"`, `"voice"`, or
`"ambient"` — used by M6's auto-allocator. The Paula mixer is
4-channel mono; the importer downsamples WAVs to match.

### `scenes`

```json
{
  "id": "main",
  "name": "Main",
  "entities": [
    {
      "id": "player",
      "name": "Player",
      "position": { "x": 32, "y": 32, "z": 0 },
      "components": [
        { "type": "Mover", "values": { "speed": 4.5 } }
      ]
    }
  ]
}
```

Entity IDs are stable strings so behaviors and event subscriptions can
reference them across saves. Components carry their values as a free-
form JSON map; concrete typing happens at runtime via the `pfcomponent`
registry.

### `behaviors`

Reserved for M5 visual scripting. Each `BehaviorGraph` carries:
- `steps`: a serialized lane editor (Wait, Tween, Move, Play, …).
- `event_sheet`: GDevelop-style condition/action rules.

M1 projects have an empty `behaviors` array.

### `bindings` and `event_subscriptions`

Reserved for M6 (audio bindings — sample × topic + scene + condition)
and M5 (event subscriptions — topic × entity × behavior reference).
Both arrays are empty in M1 projects.

### `extension_hooks`

Named callable points where user-supplied Go code can be plugged into a
generated game. The exporter wires these via the generated `main.go`
shim; the editor surfaces them as "[code extension: X]" placeholders.

## Component tag grammar (`pfcomponent`)

Component structs registered via `pfcomponent.Register[T]("Name")` use
the `pf:"..."` struct tag to advise the inspector. Tags recognised at M1:

| Tag                       | Widget kind        | Notes                                |
|---------------------------|--------------------|--------------------------------------|
| `pf:"slider,0..10"`       | slider             | min/max inclusive                    |
| `pf:"color"`              | palette-color grid | 64-swatch picker                     |
| `pf:"sprite"`             | dropdown           | lists project sprites                |
| `pf:"audio"`              | dropdown           | lists project audio samples          |
| `pf:"event"`              | dropdown           | lists project event topics           |
| `pf:"enum,A|B|C"`         | dropdown           | static option list                   |
| `pf:"text,maxlen=64"`     | text input         | clamps to maxlen runes               |
| `pf:"vector2"`            | 2-axis stepper     | works on `[2]float64`, `[2]int`, etc.|

Untagged fields fall back to a widget based on the Go type:
`bool → checkbox`, integer types `→ int-field`, floats `→ float-field`,
`string → text`, everything else `→ default` (read-only label).

Unknown tags are recorded with widget kind `"unknown"` so projects from
a newer editor still open in an older one — values are preserved but
uneditable in this build.

## Encoding rules

- JSON only at v1. CBOR may arrive in M5+ as an opt-in faster format.
- `time.Time` values use RFC 3339 with UTC offset (`"…Z"`).
- Hex colors use lowercase `#rrggbb`.
- Empty arrays are emitted as `[]` (never `null`).
- Map keys serialize in sorted lex order so diffs remain stable across
  Go versions.

## Migration policy

Every saved file declares its `schema_version`. When the project's
version is older than the binary, `pixelforge_project.MigrateForward`
runs an ordered chain of migrators that mutates the in-memory struct
in place; the next `Save()` writes the upgraded form. Newer-than-binary
versions return `ErrUnsupportedSchemaVersion` with a clear "upgrade
Pixelforge Studio" message.

There are no v1-to-v1 migrations because v1 is the initial release.
