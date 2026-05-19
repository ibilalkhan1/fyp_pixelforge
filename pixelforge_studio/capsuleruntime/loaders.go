package capsuleruntime

import (
	"fmt"
	"io/fs"
	"path"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_dialogue"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// assetsRoot is the directory prefix every embedded asset lives
// under inside the capsule's //go:embed FS. The codegen copies
// files into outDir/assets/<RelativePath> and the capsule embeds
// `assets` as its root — so the lookup path is always
// "assets/<asset.RelativePath>".
const assetsRoot = "assets"

// LoadSprites reads each project sprite from assets, decodes it
// into a pixelforge.Canvas, and registers the result under the
// sprite's Name. Returns the first decode failure wrapped with the
// offending asset's RelativePath so the capsule's startup log
// points at the broken file directly.
//
// Sprites with empty Name or RelativePath are skipped silently —
// the project loader tolerates partial schema rows, so the runtime
// matches that posture.
func LoadSprites(p *pixelforge_project.Project, assets fs.FS) error {
	if p == nil {
		return nil
	}
	for _, s := range p.Sprites {
		if s.Name == "" || s.RelativePath == "" {
			continue
		}
		bytesIn, err := fs.ReadFile(assets, path.Join(assetsRoot, s.RelativePath))
		if err != nil {
			return fmt.Errorf("capsuleruntime: read sprite %q (%s): %w", s.Name, s.RelativePath, err)
		}
		canvas, err := pixelforge.DecodeCanvasOrErr(bytesIn)
		if err != nil {
			return fmt.Errorf("capsuleruntime: decode sprite %q (%s): %w", s.Name, s.RelativePath, err)
		}
		RegisterSprite(s.Name, SpriteEntry{Asset: s, Canvas: canvas})
	}
	return nil
}

// LoadAudio reads each project audio sample from assets, decodes
// the WAV into a *pixelforge_audio.Sample, and registers the
// result under the sample's Name. Returns the first decode failure
// wrapped with the asset's RelativePath.
//
// Samples with empty Name or RelativePath are skipped silently.
func LoadAudio(p *pixelforge_project.Project, assets fs.FS) error {
	if p == nil {
		return nil
	}
	for _, a := range p.Audio {
		if a.Name == "" || a.RelativePath == "" {
			continue
		}
		bytesIn, err := fs.ReadFile(assets, path.Join(assetsRoot, a.RelativePath))
		if err != nil {
			return fmt.Errorf("capsuleruntime: read audio %q (%s): %w", a.Name, a.RelativePath, err)
		}
		sample, err := pixelforge_audio.DecodeWavOrErr(bytesIn)
		if err != nil {
			return fmt.Errorf("capsuleruntime: decode audio %q (%s): %w", a.Name, a.RelativePath, err)
		}
		pixelforge_audio.RegisterSample(a.Name, sample)
	}
	return nil
}

// LoadDialogue walks p.Dialogues and registers each parsed Tree
// under its Name. Parse errors are surfaced wrapped with the
// script's Name. Empty scripts produce empty trees and register
// without error — designers commonly stub out a dialogue before
// filling it in.
//
// Dialogues with empty Name are skipped silently.
func LoadDialogue(p *pixelforge_project.Project) error {
	if p == nil {
		return nil
	}
	for name, d := range p.Dialogues {
		if name == "" {
			continue
		}
		tree, errs := pixelforge_dialogue.Parse(d.Script)
		if len(errs) > 0 {
			return fmt.Errorf("capsuleruntime: parse dialogue %q: %v", name, errs[0])
		}
		pixelforge_dialogue.RegisterScript(name, tree)
	}
	return nil
}

// LoadScenes populates the capsule-side scene registry from
// p.Scenes. The scene/change handler in subscribers.go (U2) reads
// from this registry.
func LoadScenes(p *pixelforge_project.Project) {
	if p == nil {
		return
	}
	for _, s := range p.Scenes {
		RegisterScene(s)
	}
}

// LoadItems populates the capsule-side item registry from p.Items.
// The inventory/give_item and inventory/take_item handlers in
// subscribers.go (U2) read from this registry to validate item IDs.
func LoadItems(p *pixelforge_project.Project) {
	if p == nil {
		return
	}
	for _, it := range p.Items {
		RegisterItem(it)
	}
}
