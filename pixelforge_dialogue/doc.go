// Package pixelforge_dialogue is idea #6 v1 U5's dialogue runtime.
//
// The package centres on two concerns:
//
//   - Parser — converts the designer's authoring script (screenplay-
//     style speaker lines + Twine labels + choices + stage directions)
//     into a tree of Nodes the runtime walks.
//   - Renderer — TextBoxRenderer.Update advances the dialogue based
//     on input intents; .Draw blits the text box into a destination
//     image (the studio preview's scene texture or the shipped
//     binary's screen).
//
// Authoring syntax (one form per line; blank lines ignored):
//
//	:: label                    — Twine-style label declaration
//	SPEAKER: text               — screenplay-style line
//	[[choice text -> label]]    — branching choice
//	[[choice text -> label | if cond]]  — conditional choice
//	{state.key}                 — runtime interpolation of blackboard key
//	walk_left N                 — stage direction (entity moves N tiles)
//
// The parser is hand-rolled recursive-descent (~200 LOC) and the
// runtime is stateful (currentNode pointer + advance/branch
// dispatchers). Both halves are decoupled — tests exercise the
// parser without a runtime and the runtime against a hand-built
// tree without parsing.
package pixelforge_dialogue
