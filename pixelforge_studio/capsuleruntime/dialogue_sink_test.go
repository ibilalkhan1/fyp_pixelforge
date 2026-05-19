package capsuleruntime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

func TestDialogueSink_OpenSetsActiveScript(t *testing.T) {
	rt := bootDefault(t, pixelforge_project.NewProject("dlg"))
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "ui/open_dialogue",
		Args:  map[string]any{"script_id": "intro"},
	})
	assert.Equal(t, "intro", rt.DialogueScript)
}

func TestDialogueSink_CloseClearsActiveScript(t *testing.T) {
	rt := bootDefault(t, pixelforge_project.NewProject("dlg"))
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "ui/open_dialogue",
		Args:  map[string]any{"script_id": "intro"},
	})
	require.Equal(t, "intro", rt.DialogueScript)

	piloop.VerbsBus().Publish(&piloop.VerbEvent{Topic: "ui/close_dialogue"})
	assert.Equal(t, "", rt.DialogueScript)
}

func TestDialogueSink_CloseWhenNothingOpenNoEvent(t *testing.T) {
	rt := bootDefault(t, pixelforge_project.NewProject("dlg"))
	var seen []string
	piloop.VerbsBus().SubscribeAll(func(ev *piloop.VerbEvent, _ pievent.Handler) {
		if ev != nil && ev.Topic == "ui/dialogue_closed" {
			seen = append(seen, ev.Topic)
		}
	})
	piloop.VerbsBus().Publish(&piloop.VerbEvent{Topic: "ui/close_dialogue"})
	assert.Equal(t, "", rt.DialogueScript)
	assert.Empty(t, seen, "no close-event should fire when no dialogue is open")
}

func TestDialogueSink_OpenPublishesDialogueOpenedEvent(t *testing.T) {
	rt := bootDefault(t, pixelforge_project.NewProject("dlg"))
	var openedSeen []string
	piloop.VerbsBus().SubscribeAll(func(ev *piloop.VerbEvent, _ pievent.Handler) {
		if ev != nil && ev.Topic == "ui/dialogue_opened" {
			if id, ok := ev.Args["script_id"].(string); ok {
				openedSeen = append(openedSeen, id)
			}
		}
	})
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "ui/open_dialogue",
		Args:  map[string]any{"script_id": "intro"},
	})
	require.Equal(t, "intro", rt.DialogueScript)
	require.Len(t, openedSeen, 1)
	assert.Equal(t, "intro", openedSeen[0])
}
