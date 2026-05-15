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
}

// AnimationClip names a sequence of frame indices played at a given TPS
// rate. Loops by default; LoopMode = "once" plays the clip a single time
// then holds the last frame.
type AnimationClip struct {
	Name string `json:"name"`

	// Frames is an ordered list of frame indices into the parent
	// SpriteAsset's frame grid. The grid is row-major
	// (Width/FrameW columns × Height/FrameH rows).
	Frames []int `json:"frames"`

	// FPS is the per-clip playback rate. Independent of the project
	// TPS so a slow animation can run inside a fast game.
	FPS float64 `json:"fps"`

	// LoopMode is "loop" (default), "once", or "ping_pong".
	LoopMode string `json:"loop_mode"`
}
