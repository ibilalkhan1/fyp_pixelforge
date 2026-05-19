package pixelforge_project

// SpriteAsset describes one sprite sheet imported into the project. The
// schema carries layout information (frame size, origin, collision mask)
// alongside the relative asset path so the runtime can slice the sheet
// without re-running the import pipeline.
type SpriteAsset struct {
	// Name is the project-scoped identifier used by scripts and scenes
	// to reference this sprite. Unique within Project.Sprites.
	Name string `json:"name"`

	// RelativePath points inside the project's *-assets/ directory,
	// e.g. "sprites/snake.png". Always uses forward slashes for
	// cross-platform consistency.
	RelativePath string `json:"relative_path"`

	Width  int `json:"width"`
	Height int `json:"height"`

	// FrameW / FrameH are the per-frame dimensions for animation. When
	// the sheet has a single frame, FrameW = Width and FrameH = Height.
	FrameW int `json:"frame_w"`
	FrameH int `json:"frame_h"`

	// OriginX / OriginY are the sprite's pivot in frame-local pixels.
	// Used when DrawSprite needs to position around an anchor (e.g. a
	// player's feet).
	OriginX int `json:"origin_x"`
	OriginY int `json:"origin_y"`

	// CollisionMask is an optional packed bitmask: one bit per pixel,
	// row-major, in frame-local coordinates. Empty when collision is
	// handled by axis-aligned bounding boxes only.
	CollisionMask []uint8 `json:"collision_mask,omitempty"`

	// Animations enumerates named animation clips that select frames
	// from this sheet. Populated by the M2 importer or hand-authored.
	Animations []AnimationClip `json:"animations"`

	// SubPalette binds this sprite to one of the project's 4
	// Sprite sub-palettes (idea #3 v1). The renderer indexes
	// against the bound sub-palette's slots instead of the full
	// 64-color base, giving designers NES-style art-direction.
	// Empty on load defaults to DefaultSubPaletteName ("sprite_0")
	// via applyDefaults; omitempty so pre-v1 sprites stay byte-
	// stable through round-trip.
	SubPalette string `json:"sub_palette,omitempty" pf:"subpalette,family=sprite"`
}

// AnimationClip names a sequence of frame indices played at a given TPS
// rate. Loops by default; LoopMode = "once" plays the clip a single time
// then holds the last frame.
type AnimationClip struct {
	Name string `json:"name"`

	// Frames is an ordered list of frame indices into the parent
	// SpriteAsset's frame grid. The grid is row-major
	// (Width/FrameW columns × Height/FrameH rows).
	//
	// When ClipPath is non-empty, Frames index into that clip strip
	// instead of the parent sheet — that's the M4 capture-promoted
	// cliplet path.
	Frames []int `json:"frames"`

	// FPS is the per-clip playback rate. Independent of the project
	// TPS so a slow animation can run inside a fast game.
	FPS float64 `json:"fps"`

	// LoopMode is "loop" (default), "once", or "ping_pong".
	LoopMode string `json:"loop_mode"`

	// ClipPath is an optional override that points to a standalone
	// strip PNG (relative to the project's *-assets/ directory).
	// Captured cliplets (M4 U38) populate this; hand-authored
	// animations leave it empty so the runtime falls back to the
	// parent sheet's frame grid.
	ClipPath string `json:"clip_path,omitempty"`

	// Durations is an optional per-frame override in TPS ticks.
	// Captured cliplets default to one tick per frame; hand-authored
	// animations leave the slice nil and use FPS only.
	Durations []int `json:"durations,omitempty"`
}
