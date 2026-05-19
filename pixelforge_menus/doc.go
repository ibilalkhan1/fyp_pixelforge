// Package pixelforge_menus is idea #6 v1 U7's menu runtime.
//
// Three concerns:
//
//   - Template registry — RegisterTemplate(name, Template) stores
//     the nine NES-canonical menu shapes (title, game_over,
//     high_score, pause, save_game, load_game, inventory, status,
//     stage_select). Templates are pure metadata + a Nav handler
//     pair; the runtime composes them with per-instance parameters
//     from MenuConfig.
//   - MenuStack — push/pop semantics with pause coordination. When
//     the first overlay-type menu pushes, the stack calls
//     pixelforge_loop.Pause(); when the last one pops, it Resumes.
//   - Nav state machine — consumes input intents and updates the
//     active menu's selection cursor; emits "selected" signals
//     that the engine binds to verb-recipe dispatch.
//
// The runtime renderer hook (per-template Draw) is a thin func
// pointer the editor preview / shipped Capsule both call. Tests
// exercise the registry, the stack, and the nav state directly
// without driving any draw path.
package pixelforge_menus
