package scripting

import (
	"fmt"
	"sort"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/runtime"
)

// TopicCatalog is the canvas-resident topic-catalog sub-pane (U11).
type TopicCatalog struct {
	// lastCounts records each target's PublishCount as of the last
	// poll. Used to detect publish bursts and trigger edge flashes.
	lastCounts map[string]uint64

	// flashes tracks per-edge flash intensity (0..1). Keys are the
	// edge's destination target name; values decay each Update tick.
	flashes map[string]float64

	// selected names the currently-focused target in the list (and
	// the corresponding graph node).
	selected string

	pollTickCounter int
}

func newTopicCatalog() *TopicCatalog {
	return &TopicCatalog{
		lastCounts: map[string]uint64{},
		flashes:    map[string]float64{},
	}
}

// Update polls the registry, updates flashes, decays the existing
// ones.
func (c *TopicCatalog) Update(_ *editor.Editor, _ *runtime.Engine) {
	if c == nil {
		return
	}
	c.pollTickCounter++
	// Poll roughly once per 30 ticks (≈1 Hz at 30 TPS).
	if c.pollTickCounter%30 == 0 {
		for _, info := range pievent.EnumerateTargets() {
			prev := c.lastCounts[info.Name]
			if info.Publishes > prev {
				c.flashes[info.Name] = 1.0
			}
			c.lastCounts[info.Name] = info.Publishes
		}
	}
	// Decay flashes by 0.1 per Update tick (the plan's "10 ticks to fade").
	for k, v := range c.flashes {
		v -= 0.1
		if v <= 0 {
			delete(c.flashes, k)
			continue
		}
		c.flashes[k] = v
	}
}

// SelectedTarget returns the name of the focused target, or "".
func (c *TopicCatalog) SelectedTarget() string {
	if c == nil {
		return ""
	}
	return c.selected
}

// Flash returns the current flash intensity for the given target name.
func (c *TopicCatalog) Flash(target string) float64 {
	if c == nil {
		return 0
	}
	return c.flashes[target]
}

// DrawCanvas paints the catalog.
func (c *TopicCatalog) DrawCanvas(rel widgets.Rect, _ *editor.Editor, theme *editor.EditorTheme) {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	pixelforge.SetColor(theme.PanelSlot)
	pixelforge.RectFill(rel.X, rel.Y, rel.X+rel.W-1, rel.Y+rel.H-1)

	targets := pievent.EnumerateTargets()
	if len(targets) == 0 {
		pixelforge.SetColor(theme.TextDimSlot)
		pixelforge_cofont.Print("(no targets)", rel.X+4, rel.Y+4)
		return
	}

	// Left half: stable-sorted list. EnumerateTargets already returns
	// them alphabetically; re-sort defensively in case future
	// versions change.
	sort.SliceStable(targets, func(i, j int) bool { return targets[i].Name < targets[j].Name })

	leftW := rel.W / 2
	pixelforge.SetColor(theme.TextSlot)
	pixelforge_cofont.Print("TARGETS", rel.X+4, rel.Y+2)

	y := rel.Y + 14
	for _, info := range targets {
		line := fmt.Sprintf("%s  s=%d  p=%d", info.Name, info.Subscribers, info.Publishes)
		flash := c.flashes[info.Name]
		if flash > 0 {
			pixelforge.SetColor(theme.AccentSlot)
		} else {
			pixelforge.SetColor(theme.TextDimSlot)
		}
		pixelforge_cofont.Print(line, rel.X+4, y)
		y += 9
		if y > rel.Y+rel.H-10 {
			break
		}
	}

	// Right half: minimal node graph (one node per target).
	rightX := rel.X + leftW + 4
	pixelforge.SetColor(theme.TextSlot)
	pixelforge_cofont.Print("GRAPH", rightX, rel.Y+2)

	ny := rel.Y + 14
	for _, info := range targets {
		flash := c.flashes[info.Name]
		bg := theme.PanelHeaderSlot
		if flash > 0 {
			bg = theme.AccentSlot
		}
		pixelforge.SetColor(bg)
		nodeW := rel.W - leftW - 12
		if nodeW > 96 {
			nodeW = 96
		}
		pixelforge.RectFill(rightX, ny, rightX+nodeW-1, ny+10)
		pixelforge.SetColor(theme.TextSlot)
		pixelforge_cofont.Print(info.Name, rightX+2, ny+1)
		ny += 12
		if ny > rel.Y+rel.H-12 {
			break
		}
	}
}
