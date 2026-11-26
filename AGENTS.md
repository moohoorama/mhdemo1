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
For new officer portraits, use the README's current full-body-to-upper-body workflow:
give each character a pose and expression suited to their story and role, preserve the
full-body original, crop the upper body for the 256×256 game portrait, and check a game capture.

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

To check what the story already says (who said what where, earlier foreshadowing, how someone was addressed), search
the story index instead of reading whole files: from `ver5/`, `uv run tools/storydb/storydb.py search "<what happened>"`
finds scenes by meaning across `docs/fullstory.md` and the story, explore and stage files, `... grep <word> [--speaker 관우]`
lists exact lines, both with file:line. Run `... index` first after story edits (only changed chunks are embedded;
needs Ollama with `bge-m3`).

The core of this game's story (`ver5/docs/fullstory.md` and every scene, explore line and battle line drawn from it) is
that it satisfies both the 삼국지 enthusiast and the player who enjoys 삼국지 lightly — in its wording and in its plot:
- Wording keeps a fitting distance from the period. Not the language of papers, reference works or specialist
  sources (no bare official titles like 상시 or 낭중, no 태학생 or 당고 in a passing line), but words a player today
  understands that still sit naturally next to 연의's phrasing (십상시, 환관, 대장군, 벼슬을 돈 주고 산다).
  Titles 연의 readers know well and a newcomer still reads as "some post" stay (기도위, 북부위, 태수, 자사, and also 상서, 사공, 사도, 거기장군, 상국, 승상, 삼공).
  What to avoid is dictionary-only old words and forms (목백, 부중, 막하, 약낭, "공 북해"); the user's keep/replace
  list is `ver5/tools/storydb/jargon.tsv` and `storydb.py jargon` lists lines that use a replace word (see
  `ver5/docs/story-writing.md` 어휘).
- Show the times as a feeling, not a thesis. The player only needs to sense that things are rotten (벼슬을 사고팔고,
  뇌물을 바라고, 그 값을 세금으로 짜낸다); don't lecture on causes such as "the Yellow Turbans rose because of the
  십상시".
- The plot must hold up for the enthusiast: each person acts from their real standing, interests and the customs of
  the late Han (check 정사 and 연의 before writing; don't belittle anyone without reason, don't invent schemes their
  position would never allow). People exchange names before they use each other's 자.
- Titles and forms of address follow the year of the scene and each speaker's standing. Check the office a person
  holds at that moment (도겸 is 서주목 by 194, not 서주자사), and use the honorific a period reader expects (a 자사 or 목
  is 사군: "사군께서"). A subordinate never calls their own lord by 자 in front of others (손건 says "서주목 도겸 공의
  막료", never "도공조"). A 자 is given when peers meet or when the two will call each other by it from now on; a
  low-ranking officer or envoy introducing himself to a general gives only his name, and his 자 comes out later, when
  the relationship begins (when he joins, for example).
- Well-known anecdotes (조조's 오색 몽둥이, his feigned stroke, 허소's verdict) go in as townsfolk gossip a fan smiles
  at and a newcomer still follows. Optional talk carries this colour; the line that opens the next event stays one,
  short and plain.

Where every person looks matters as much as where they stand: facing is read before any line is. Whenever you
write or change a battle event, a cutscene, a scene, a stage's placement or its opening, decide each person's facing
with their position — the engine turns an unset battle spawn toward its side's objective, and every event move should
name the `Facing` it ends with — and check it on a game capture. Read `ver5/docs/presentation-facing.md` first.

When you add or edit a stage (`ver5/assets/scenario/*/stageNN.json`, its map or its story nodes), read
`ver5/docs/stage_lv_balance.md` first and set the ranks and the enemy count by it: stage files never hold
levels — each spawn has a `Rank` (rabble, recruit, troop, officer, commander, boss) and an optional `Shift`, and the engine
computes the level from the stage's place in the story (3 + round(1.5×(n − 1))) with a lead that is a share of
strength, not a fixed number of levels. Joining officers are raised to the stage level, shifted by the difficulty like a troop (`Rules.JoinLevel`). Then measure with `go run ./cmd/_balance`,
play it through `go run ./cmd/srpg-cli -slot 9` (numbered menus, letter maps; the slot carries the game
between runs) to check the placement and the order of the story, and only then capture the GUI.

Never force a duel because an officer's HP dropped (a battle event that opens a duel at "HP under half", or a
victory goal turned into a duel so that only that officer can finish the fight). The user rejected it outright
(2026-10-30): it breeds bugs (the blow that would have felled the officer, counters and area strikes, the duel's
own outcome against the HP left) and it breaks the battle's flow. A duel opens when its officers meet as the
stage sets it (adjacency, `DuelFirst`, a scripted event at a story moment). If another officer defeats the duel's
partner first, the duel simply does not happen; write the story after the battle so it does not depend on who
struck the last blow.

Every task leaves a result folder: make `~/orca/mhdemo1/result/YYYYMMDD/<task-name>/` (today's date, one
directory per task) and save what was done there as md, html, mp4, PNG or whatever fits (a summary of what changed
and why, plus captures, videos or comparison pages). Do this for every task, not only art or UI work, and give the
user the folder path when you report.

Every time you change something, record it in `ver5/docs/history.md`. Each entry lists the changed
file name, what was changed, and the intent (or reason): `파일이름: 수정내역 — 의도`.

For each user request, create a descriptively named Markdown file under `~/orca/mhdemo1/request/`
that summarizes the request, preserving its requirements and linking its result folder when known.
The request-watcher moves each file from `request/` to `~/orca/mhdemo1/process/<name>/<name>.md` shortly
after it appears (and writes `process/<name>/progress.md` every minute with the real Asia/Seoul time). When the work
is done the file moves to `~/orca/mhdemo1/test/<name>/<name>.md` and is tested there (`test/<name>/test-log.md`), and
only after every required test passes it moves to `result/YYYYMMDD/<name>/<name>.md`; a request that is not in
`result/` is not finished. So when pointing the user to a request record, find where the file is now
(`python3 tools/request_mcp/requestflow.py where <name>` tells the stage) and link that path (check that the file
exists first), never the `request/` path.
When asked how a request was carried out, find the corresponding records under
`~/orca/mhdemo1/result/YYYYMMDD/` and explain the recorded work, noting any gaps.
