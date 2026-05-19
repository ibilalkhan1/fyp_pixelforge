package assetlibrary

import (
	"log"
	"sort"
	"strings"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
)

// AssembleCredits walks every asset reference in the project,
// looks each up in the installed library packs, and returns the
// CC-BY-licensed entries the auto-injected Credits screen
// should display. CC0 entries are excluded — they carry no
// attribution duty and would just dilute the page.
//
// Orphaned assets (named in the project but not present in any
// installed pack) are recorded with Author="Unknown" and a
// warning logged at generate time so designers see what slipped
// through.
//
// Returned slice is sorted by Name for deterministic output.
func AssembleCredits(p *pixelforge_project.Project, lib *Library) []capsuleruntime.CreditEntry {
	if p == nil || lib == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []capsuleruntime.CreditEntry

	collect := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		_, asset, ok := lib.FindAsset(name)
		if !ok {
			log.Printf("[credits] asset %q referenced by project not found in library; recording as Unknown", name)
			out = append(out, capsuleruntime.CreditEntry{Name: name, License: "Unknown", Author: "Unknown"})
			return
		}
		if !licenseRequiresAttribution(asset.License) {
			// CC0 + public-domain entries skip the credits page.
			return
		}
		out = append(out, capsuleruntime.CreditEntry{
			Name:      name,
			License:   asset.License,
			Author:    asset.Author,
			SourceURL: asset.SourceURL,
		})
	}

	for _, s := range p.Sprites {
		collect(s.Name)
	}
	for _, a := range p.Audio {
		collect(a.Name)
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// licenseRequiresAttribution reports whether the supplied SPDX-
// style license identifier requires attribution. CC-BY-* + a few
// adjacent attribution-required licenses qualify; CC0 + plain
// "Public Domain" do not. Unknown / empty licenses default to
// "requires attribution" so designers see the entry rather than
// silently omitting credit for a missing license declaration.
func licenseRequiresAttribution(license string) bool {
	upper := strings.ToUpper(strings.TrimSpace(license))
	switch upper {
	case "", "CC0", "CC0-1.0", "PUBLIC DOMAIN", "PD":
		return false
	}
	return strings.HasPrefix(upper, "CC-BY") ||
		strings.Contains(upper, "ATTRIBUTION") ||
		// Unknown licenses default to "requires attribution" so
		// designers see the row and notice the missing license
		// metadata; safer than silently dropping.
		!strings.HasPrefix(upper, "CC0")
}
