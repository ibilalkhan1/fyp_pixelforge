package pixelforge_project

import (
	"encoding/json"
	"log"
	"math"
)

// sceneJSONRaw mirrors the on-disk shape of a Scene with each field
// kept as json.RawMessage so the custom UnmarshalJSON can detect
// presence vs absence of the legacy `tilemaps` key and the v1+
// `tile_atlases` key independently. Keeping the raw shape avoids
// double-decoding the same payload.
type sceneJSONRaw struct {
	ID                json.RawMessage `json:"id"`
	Name              json.RawMessage `json:"name"`
	Entities          json.RawMessage `json:"entities"`
	Tilemaps          json.RawMessage `json:"tilemaps"`
	TileAtlases       json.RawMessage `json:"tile_atlases"`
	GridWidthScreens  json.RawMessage `json:"grid_width_screens"`
	GridHeightScreens json.RawMessage `json:"grid_height_screens"`
	SpawnTile         json.RawMessage `json:"spawn_tile"`
	DefaultTileID     json.RawMessage `json:"default_tile_id"`
	Camera            json.RawMessage `json:"camera"`
	World             json.RawMessage `json:"world"`
	Zones             json.RawMessage `json:"zones"`
	Warps             json.RawMessage `json:"warps"`
	CameraMode        json.RawMessage `json:"camera_mode"`
}

// UnmarshalJSON migrates legacy `tilemaps` JSON keys to the v1+
// TileAtlases field on load. The shim is the only persistent surface
// of the idea #2 schema reframe — it lets pre-v1 .pforge files load
// cleanly without a one-time data-migration tool. If both the legacy
// key and the new key are present, the new key wins and the legacy
// key is logged + dropped (defensive: hand-edited files could
// produce this).
//
// All other Scene fields are decoded through the stdlib path via a
// throwaway alias type that suppresses this UnmarshalJSON (otherwise
// the call would recurse).
func (s *Scene) UnmarshalJSON(data []byte) error {
	var raw sceneJSONRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Decode the bulk of the scene through the stdlib path. The
	// alias suppresses Scene.UnmarshalJSON so we don't recurse.
	type sceneStdlib Scene
	var sd sceneStdlib
	if err := json.Unmarshal(data, &sd); err != nil {
		return err
	}
	*s = Scene(sd)

	hasNew := len(raw.TileAtlases) > 0 && string(raw.TileAtlases) != "null"
	hasLegacy := len(raw.Tilemaps) > 0 && string(raw.Tilemaps) != "null"

	switch {
	case hasNew && hasLegacy:
		log.Printf("pixelforge_project: scene %q has both `tile_atlases` "+
			"and legacy `tilemaps` keys; using `tile_atlases`, "+
			"dropping `tilemaps`", s.ID)
	case hasLegacy:
		var legacy []TileAtlas
		if err := json.Unmarshal(raw.Tilemaps, &legacy); err != nil {
			return err
		}
		s.TileAtlases = legacy
		log.Printf("pixelforge_project: migrated scene %q legacy "+
			"`tilemaps` (%d layers) to `tile_atlases`", s.ID, len(legacy))
	}

	sanitizeReservedFields(s)
	return nil
}

// sanitizeReservedFields clamps the idea-#2-reserved tile-atlas
// fields to the ranges their pf-tag widgets advertise so the schema
// and the inspector never disagree. A future inspector frame that
// renders e.g. AnimationFps as a slider with max=30 must never see
// an on-disk value above 30; clamping at load is the cheapest
// guarantee.
//
// Clamps:
//   - AnimationFps: 0..30 (matches pf:"slider,0..30")
//   - ParallaxFactor: 0.0..2.0 (matches pf:"slider,0.0..2.0")
//
// Out-of-range values are silently corrected rather than rejected, in
// line with the additive-omitempty + sanitize discipline documented
// in docs/solutions/editor-pforge-schema-shape.md.
func sanitizeReservedFields(s *Scene) {
	for i := range s.TileAtlases {
		a := &s.TileAtlases[i]
		if a.AnimationFps < 0 {
			a.AnimationFps = 0
		}
		if a.AnimationFps > AnimationFpsMax {
			a.AnimationFps = AnimationFpsMax
		}
		if a.ParallaxFactor < 0 {
			a.ParallaxFactor = 0
		}
		if a.ParallaxFactor > ParallaxFactorMax {
			a.ParallaxFactor = ParallaxFactorMax
		}
		// Defensive: NaN/Inf in a hand-edited file falls back to 0.
		if math.IsNaN(a.ParallaxFactor) || math.IsInf(a.ParallaxFactor, 0) {
			a.ParallaxFactor = 0
		}
	}
}

// Reserved-field declared ranges. Kept in lockstep with the pf-tag
// declarations on TileAtlas so the sanitize clamp and the inspector
// slider agree on a single source of truth.
const (
	AnimationFpsMax   = 30
	ParallaxFactorMax = 2.0
)
