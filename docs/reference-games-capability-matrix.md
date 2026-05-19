# Reference-Games Capability Matrix

Honest assessment of what the four named reference games — **Asteroids**, **Bomberman**, **Mario** (Super Mario Bros-style platformer), and **Donkey Kong** — can be authored as today via the no-code studio's verb-recipe catalog, and what's still missing.

This is the live tracking artifact for plan-008 U7. Each RED row maps directly to a recipe added by U8. Re-read after every catalog change.

## Legend

- **GREEN** — Directly buildable today using only registered recipes.
- **YELLOW** — Buildable with a wrapper / combination of existing recipes; some authorial pain (no single-step verb).
- **RED** — Needs a new primitive. Maps to a U8 recipe.

The "Recipes / Gap" column names every verb the authoring path would use. Pre-U8 names are the existing recipes in `pixelforge_studio/scripting/catalog/builtin_actions.go` and `builtin_rpg.go`; post-U8 names (italics) refer to recipes added by U8 in `builtin_arcade.go`.

## Matrix

After plan-008 U8 every previously-RED row is GREEN. RED entries here would indicate a regression — a recipe was removed without a replacement.

| Game | Mechanic | Status | Recipes / Gap |
|------|----------|--------|---------------|
| Asteroids   | Thrust ship forward                       | GREEN  | `apply_thrust` |
| Asteroids   | Rotate ship left / right                  | GREEN  | `rotate_entity` |
| Asteroids   | Shoot bullet                              | GREEN  | `spawn_entity` + `move_with_intent` |
| Asteroids   | Screen-wrap on edge crossing              | GREEN  | `screen_wrap` |
| Asteroids   | Break asteroid into smaller pieces        | YELLOW | `destroy_self` + N × `spawn_entity` (no size-parameterised splitter) |
| Asteroids   | Score on asteroid destroyed               | GREEN  | `give_points` |
| Bomberman   | Move on a grid (4-directional)            | GREEN  | `move_with_intent` |
| Bomberman   | Place bomb on the player's grid cell      | GREEN  | `place_on_grid` + `spawn_entity` |
| Bomberman   | Bomb fuse → explosion + radius damage     | GREEN  | `explode_radius` |
| Bomberman   | Pick up power-up (range, extra bomb)      | GREEN  | `give_item` + `take_item` |
| Bomberman   | Player dies in explosion                  | GREEN  | `die` + `lose_life` |
| Bomberman   | Win condition (clear all enemies)         | YELLOW | `set_flag` per kill + `check_flag` predicate + `change_scene`; no built-in enemy-counter verb |
| Mario       | Run left / right                          | GREEN  | `move_with_intent` |
| Mario       | Jump + gravity                            | GREEN  | `jump` + `apply_gravity` |
| Mario       | Platform / tile collision (solid tiles)   | GREEN  | `solid_collide` |
| Mario       | Stomp enemy from above                    | GREEN  | `destroy_other` + `give_points` |
| Mario       | Collect coin                              | GREEN  | `give_item` + `give_points` + `destroy_other` |
| Mario       | Power-up (mushroom) spawn + pickup        | GREEN  | `spawn_entity` + `give_item` |
| Donkey Kong | Jump + gravity (Mario character)          | GREEN  | `jump` + `apply_gravity` |
| Donkey Kong | Climb ladder                              | GREEN  | `ladder_climb` |
| Donkey Kong | Barrel rolls down sloped girders          | GREEN  | `barrel_roll` |
| Donkey Kong | Hammer pick-up + smash barrel             | YELLOW | `give_item` + `destroy_other`; no item-effect verb that consumes a held item to trigger an action |
| Donkey Kong | Stage clear advances to next level        | GREEN  | `change_scene` |
| Donkey Kong | Lose life on barrel collision             | GREEN  | `die` + `lose_life` |

## Cross-game primitives

One additional U8 recipe — *fixed_tick_loop* — isn't game-specific. It groups a per-tick condition + action sequence and is the substrate every authored game uses to express "do X every frame while Y is true". Without it, designers fall back to wiring a `when_every_tick` trigger per behaviour, which doesn't compose well across multiple per-tick rules on the same entity.

## Summary

After plan-008 U8:

- **RED rows:** 0. The 9 previously-RED mechanics now resolve via 10 new recipes (`apply_thrust`, `rotate_entity`, `screen_wrap`, `jump`, `apply_gravity`, `solid_collide`, `place_on_grid`, `explode_radius`, `ladder_climb`, `barrel_roll`) plus the cross-cutting `fixed_tick_loop` — **11 recipes** added by U8.
- **YELLOW rows:** 3 — break-asteroid, win-condition, hammer-smash. Authoring works today but takes multiple verbs; future polish can collapse each into a single recipe.
- **GREEN rows:** 21. The four named reference games are authorable end-to-end via the studio's verb-recipe dropdown today.

## Verification rule

Whenever U8 lands a recipe, the matching RED row above flips to GREEN with the new recipe in the "Recipes / Gap" column (no longer in italics). Whenever a recipe in this matrix is renamed in the catalog, this doc gets updated in the same change set — there is no acceptable drift between the two.
