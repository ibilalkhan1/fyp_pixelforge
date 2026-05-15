package palette

import (
	"encoding/json"
	"errors"
	"os"
)

// Sidecar is the JSON .png.meta payload sitting next to a PNG asset.
// It overrides auto-detected import results.
type Sidecar struct {
	FrameW  int `json:"frame_w,omitempty"`
	FrameH  int `json:"frame_h,omitempty"`
	OriginX int `json:"origin_x,omitempty"`
	OriginY int `json:"origin_y,omitempty"`

	AnimationClips []SidecarClip `json:"animations,omitempty"`
}

// SidecarClip names an animation clip the .png.meta declares.
type SidecarClip struct {
	Name     string  `json:"name"`
	Frames   []int   `json:"frames"`
	FPS      float64 `json:"fps"`
	LoopMode string  `json:"loop_mode"`
}

// LoadSidecar attempts to read a sidecar from `<pngPath>.meta`. If no
// sidecar file exists, returns (Sidecar{}, nil) — the caller treats it
// as the empty default. Malformed sidecars surface as an error so the
// importer can log a warning while still proceeding with the PNG's
// auto-detection.
func LoadSidecar(pngPath string) (Sidecar, error) {
	data, err := os.ReadFile(pngPath + ".meta")
	if errors.Is(err, os.ErrNotExist) {
		return Sidecar{}, nil
	}
	if err != nil {
		return Sidecar{}, err
	}
	var sc Sidecar
	if err := json.Unmarshal(data, &sc); err != nil {
		return Sidecar{}, err
	}
	return sc, nil
}
