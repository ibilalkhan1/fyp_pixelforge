package catalog_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
)

// TestWriteVerbCatalogMarkdown_Stable ensures gendocs produces
// byte-identical output across two consecutive runs. The CI gate "go
// generate && check-no-diff" only makes sense if regeneration is
// deterministic. Map-ranging or unsorted iteration would break this
// invariant; the test pins the contract.
func TestWriteVerbCatalogMarkdown_Stable(t *testing.T) {
	var first, second bytes.Buffer
	require.NoError(t, catalog.WriteVerbCatalogMarkdown(&first))
	require.NoError(t, catalog.WriteVerbCatalogMarkdown(&second))
	assert.Equal(t, first.String(), second.String(),
		"gendocs output must be byte-stable across runs")
}

// TestWriteVerbCatalogMarkdown_CoversEveryVerb asserts the markdown
// mentions every registered recipe name. The doc-generated section
// header is `### \`<name>\`` so we scan for that exact substring per
// recipe.
func TestWriteVerbCatalogMarkdown_CoversEveryVerb(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, catalog.WriteVerbCatalogMarkdown(&buf))
	out := buf.String()

	verbs := catalog.ListVerbs()
	require.NotEmpty(t, verbs, "catalog must register at least one verb for this test")
	for _, v := range verbs {
		needle := "### `" + v.Name + "`"
		assert.Contains(t, out, needle,
			"markdown should carry a section for recipe %q", v.Name)
	}
}

// TestWriteVerbCatalogMarkdown_HasExpectedSections asserts the doc
// renders the headline + group headings expected by U24's plan output
// shape.
func TestWriteVerbCatalogMarkdown_HasExpectedSections(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, catalog.WriteVerbCatalogMarkdown(&buf))
	out := buf.String()

	assert.True(t, strings.HasPrefix(out, "# Pixelforge Verb Catalog"),
		"doc must lead with the headline")
	assert.Contains(t, out, "## Action recipes",
		"doc must group action recipes under a header")
	// Condition recipes exist (has_item) so the heading must appear.
	assert.Contains(t, out, "## Condition recipes",
		"doc must group condition recipes under a header")
}

// TestListVerbs_SortedByName confirms ListVerbs returns the recipe
// set in name-sorted order — the gendocs writer depends on this for
// stable output.
func TestListVerbs_SortedByName(t *testing.T) {
	verbs := catalog.ListVerbs()
	require.NotEmpty(t, verbs)
	for i := 1; i < len(verbs); i++ {
		assert.True(t, verbs[i-1].Name < verbs[i].Name,
			"ListVerbs must return name-sorted output (offenders: %q before %q)",
			verbs[i-1].Name, verbs[i].Name)
	}
}

// TestListVerbs_PopulatesTopicForPublishRecipes asserts ListVerbs
// extracts the event-bus topic from publish_event recipe defaults.
func TestListVerbs_PopulatesTopicForPublishRecipes(t *testing.T) {
	// apply_thrust is a publish_event recipe — its topic must be
	// "motion/apply_thrust".
	for _, v := range catalog.ListVerbs() {
		if v.Name != catalog.RecipeApplyThrust {
			continue
		}
		assert.Equal(t, catalog.EventTopicMotionApplyThrust, v.Topic)
		assert.Equal(t, catalog.VerbKindAction, v.Kind)
		return
	}
	t.Fatalf("expected to find recipe %q in ListVerbs output",
		catalog.RecipeApplyThrust)
}

// TestListVerbs_ConditionRecipeKind asserts the has_item recipe surfaces
// as a condition (ConditionKind set, ActionKind empty).
func TestListVerbs_ConditionRecipeKind(t *testing.T) {
	for _, v := range catalog.ListVerbs() {
		if v.Name != catalog.RecipeHasItem {
			continue
		}
		assert.Equal(t, catalog.VerbKindCondition, v.Kind)
		assert.Empty(t, v.ActionKind)
		assert.NotEmpty(t, v.ConditionKind)
		assert.Empty(t, v.Topic, "condition recipes do not publish bus topics")
		return
	}
	t.Fatalf("expected to find recipe %q in ListVerbs output",
		catalog.RecipeHasItem)
}

// TestAllVerbTopics_NoDuplicates asserts AllVerbTopics returns a
// sorted unique set. Two recipes could in principle publish to the
// same topic; the helper must collapse those.
func TestAllVerbTopics_NoDuplicates(t *testing.T) {
	topics := catalog.AllVerbTopics()
	require.NotEmpty(t, topics)
	seen := make(map[string]struct{}, len(topics))
	for _, top := range topics {
		_, dup := seen[top]
		assert.False(t, dup, "topic %q appears more than once", top)
		seen[top] = struct{}{}
	}
	// Also sorted.
	for i := 1; i < len(topics); i++ {
		assert.True(t, topics[i-1] < topics[i],
			"AllVerbTopics must be sorted")
	}
}
