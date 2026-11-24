# Browser game creator acceptance — 24 November 2026

The faculty link must enable visitors to make a game, not only play a fixed demonstration. This update adds blank projects, collector and escape templates, configurable win rules, hazard tiles, a movable finish, and preserved rules in standalone HTML exports.

## Observed browser results

- Created a blank game and placed its finish near the player; keyboard movement reached it and displayed **Level complete**.
- Named a new project **My maze adventure**, placed a hazard at cell (5,9), and played into it; the engine stopped and displayed **Game over**.
- Reset, paused, saved and restored the project. The JSON download contained the exact authored name, escape mode, hazard and finish coordinates.
- Changed the win rule to collection and replaced the hazard with a crystal; collecting it displayed **Level complete**, 1/1 crystals.
- Downloaded the authored game as HTML. Opened it over a local HTTP server, started its embedded Go engine, and verified that the authored hazard still ended the run.
- Earlier browser acceptance covered tile editing, sprite painting, collision, movement, saving and restoration. The new engine tests cover legacy collector files, escape completion, optional bonus crystals, hazard priority, stopping terminal runs, reset and invalid scene rejection.

The public deployment is a separate static `gh-pages` branch. No backend or author accounts are needed. Downloading HTML does not automatically publish a visitor's game. Scene JSON is separate from Studio .pforge. This creator supports one top-down level with two win rules; audio, authored scripts and multiple levels are not included. It does not prove a level is solvable. Browser storage has one local save slot; download JSON to preserve multiple projects.

The native Studio's full interactive edit/play/export flow, audio decoder completion, full reference games and platform launch acceptance remain separate from this bounded browser acceptance. Passing this workflow does not establish 100% completion of the original FYP-I scope. Manually rehearse exported HTML from a local file on the actual presentation laptop; this session verified its embedded engine through HTTP.

Implementation and review were AI-assisted under Muhammad Bilal's direction. No teammate contribution history was fabricated.
