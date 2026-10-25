# AGENTS.md

Read `ver5/README.md` before modifying this repository. The game lives in `ver5/`.

Sprites the game reads are `ver5/assets/sprites/*.spr`, built from the dot editor YAML
(`ver5/tools/doteditor/units/*.yaml` via `ver5/tools/spritebuild/build.py`; see `ver5/docs/presentation-sprite.md`).
Do not add project-specific sprite transformations to `tools/spritetool`. Keep runtime
assets under `ver5/assets/` and source artwork under `tools/spritetool/assets/`.

Before creating or modifying any sprite, read
`tools/spritetool/assets/SPRITE_ART_GUIDE.md` and the target asset's settings
or README. Apply the common rules and the relevant asset-type rules to both
candidate previews and final artwork.

For unit sprites (soldiers, cavalry, archers, heroes), also read
`tools/spritetool/assets/UNIT_ART_GUIDE.md`. `tools/unit3d/` is the old 3D cel-shading pipeline, kept as reference;
new dot work goes through the dot editor (`hero-sprite` skill).

Illustrations (officer portraits, event CGs, backgrounds, UI art) are not
sprites: their sources live under `tools/illustrations/`. Read
`tools/illustrations/README.md` first and keep the chosen ink-wash style.

To check UI or on-screen output, capture it instead of guessing from code. `ver5/cmd/_uidemo-vm`
plays a named UI demo (`ver5/internal/gui/uidemo_*.go`) headlessly and saves PNG frames and/or a
60 fps MP4; run it from `ver5/` wrapped in `caffeinate -d -i`, e.g.
`go run ./cmd/_uidemo-vm -ui dialogue -frames 300 -at 30,96 -mp4 reports/uidemo/dialogue.mp4`.
`-size 2400x1080 -uiscale 160` plays it on a phone-sized screen, and `-sheet -at 260,380,720` puts the
same frames from the four reference screens (docs_old/plans/PLAN_ui-scale-ppi.md) on one image. Demos that
cover whole screens: `uiscale` (story, battle HUD), `popups` (title, windows, modals, preparation),
`deploy`, `duel`, `editor`.
Add a new demo file there when no existing demo shows the screen. `cmd/_uidemo` is the windowed version.

A **scene** (장면) is what the story screen shows between battles: the scene map, the cast and extras standing
on it, their beats (walks, turns, motions, emotes, bubbles, effects) and the camera, for one stretch of story
nodes (`ver5/assets/scenes/chNN.json`, `ver5/docs/presentation-scene.md`). Say "scene" for this, not for a story node,
a battle map or an event CG. `go run ./cmd/_uidemo-vm -ui scenenode -stage c01-t1,c01-t2` plays the given story
nodes' scenes line by line for capture.

Before writing or editing story text (explore people's lines, events, scenes before a sortie, battle dialogue), read
`ver5/docs/story-writing.md`: it holds the tone the confirmed chapters keep (연의's gravity with everyday humour,
standing shown before it is claimed, actions over explanations), the staging rules and the order of work (two or three
drafts side by side in an html page, the user picks, then data, then a game capture). Per-officer speech is in
`ver5/docs/data-speech-style.md`.

Where every person looks matters as much as where they stand: facing is read before any line is. Whenever you
write or change a battle event, a cutscene, a scene, a stage's placement or its opening, decide each person's facing
with their position — the engine turns an unset battle spawn toward its side's objective, and every event move should
name the `Facing` it ends with — and check it on a game capture. Read `ver5/docs/presentation-facing.md` first.

When you add or edit a stage (`ver5/assets/scenario/*/stageNN.json`, its map or its story nodes), read
`ver5/docs/stage_lv_balance.md` first and set the ranks and the enemy count by it: stage files never hold
levels — each spawn has a `Rank` (troop, officer, commander, boss) and an optional `Shift`, and the engine
computes the level from the stage's place in the story (3 + 2×(n − 1)) with a lead that is a share of
strength, not a fixed number of levels. Joining officers are raised to the stage level. Then measure with `go run ./cmd/_balance`,
play it through `go run ./cmd/srpg-cli -slot 9` (numbered menus, letter maps; the slot carries the game
between runs) to check the placement and the order of the story, and only then capture the GUI.

Every time you change something, record it in `ver5/docs/history.md`. Each entry lists the changed
file name, what was changed, and the intent (or reason): `파일이름: 수정내역 — 의도`.
