---
name: scene-staging
description: Stages the story scenes (장면) of the ver5 SRPG — who stands where on which scene map, the extras, the beats (walks, entrances, turns, motions, emotes, effects) for each line — from the story text, as a storyboard first, then in assets/scenes/chNN.json, checked by a placement script and by game captures. Use for "장면 연출", "씬 그려줘", "장면 개선", "스토리 장면 배치", "N장 장면 다시", after writing or editing story nodes, or when a scene looks empty, crowded, overlapping or static.
---

# Scene staging

Work from `ver5/`. A scene is what the story screen shows between battles: the scene map, the cast and extras on it,
their beats and the camera, for one stretch of story nodes (a `scene` node up to the next `scene` or `battle` node).
Read first:

- `docs/presentation-scene.md` — the scene file format (`assets/scenes/chNN.json`: one scene per stretch with Map,
  Time, Nodes, Ambient, Cast, Beats), the scene maps and where one can stand on each, every step the engine knows
  (Move, Enter/Exit, Speed, After, Face, Look, Play, Emote, Say, Wait, Intro, Effect, Shake, Cam/CamAt, Zoom, Time,
  Sprite, Prop, Pose, Patrol, Chatter), a scene's Backdrop, and the scene people and poses (officials that bow and
  prostrate, eunuchs, wounded lying or slumped, helper pairs, stretchers, prone crossbowmen, white horses; officers
  that sit, the emperor on his throne). Use only those. When the story needs something they cannot show, write it down as an engine request
  in the report; do not fake it with something that reads wrong.
- `docs/story-writing.md` "자리와 방향" — units (scene cells vs explore/object small tiles), which axis each Face
  points along, how to face someone and how to line people up in front of a thing. Check every At/Move/Face against it.
- `assets/scenes/ch02.json`, `ch03.json`, `ch05.json`, `ch06.json` as worked examples (a council, a court, arrivals on a road, a night camp).
- The stretch's story nodes in `assets/scenario/hero/storyNN.json` and its part of `docs/fullstory.md`.
- Reuse battle sprites, motions and terrain; draw new art only where nothing fits (scholar/strategist sprites stand in
  for court officials, class sprites for soldiers and riders, `bandit` for Yellow Turbans).

Report in Korean, with before/after frames and the storyboard on a page.

## 1. Read the stretch as a director

For every stretch, list before touching data:

- **Place and time.** Day, dusk, night, dawn; inside or outside; what the narration says is there (fires, tables,
  thrones, a wall with the enemy's fires beyond it).
- **Who is present**, not only who speaks:
  - speakers, plus everyone the narration names ("조조, 공융, 도겸 … 이 들어와 자리를 잡는다"),
  - retinues the situation implies: Liu Bei comes with Guan Yu and Zhang Fei unless the story has split them; a lord
    comes with his own officers (Cao Cao with 하후돈·하후연·조인·조홍, Sun Jian with 황개·정보·한당); a court has
    officials in rows and guards at the door; a camp has soldiers; a council has seats and guards,
  - characters that exist in `assets/data/characters*.json` stand as themselves; people with no data stand as extras.
- **Who enters, leaves or moves on which line** ("전령이 뛰어든다", "관우가 앞으로 나선다", "말을 몰아 다가온다").
- **Sides.** An enemy who shouts from afar stays afar (the far edge, facing in), never in the allies' row.

Then read the battle on each side of the stretch (`assets/scenario/hero/stageNN.json`: `Events` lines and cutscene
`say` steps, `Duels` texts and beats, `Victory`/`Sub`) in play order with the story, and fix the text before staging:

- nothing is told twice: a moment shown in a battle (a duel's challenge, a famous line) is not replayed in the story,
  and a battle's call-out is not repeated word for word by the duel that follows it;
- no outcome is reported before it is shown: a scene before a battle does not celebrate what the battle's opening
  will show;
- who died, retreated or joined in the battle matches the story after it (a duel's `EnemyResult: death` and a later
  scene with that officer alive cannot both stand), and so do sides, directions and places (south fence vs north);
- everyone who fought in the battle and matters to the story is seen or mentioned after it;
- a `grant` or `join` comes after the line that gives or welcomes, not before;
- each officer keeps one voice across story, battle bubbles and duels.

`docs/fullstory.md` is the plan; when data and plan disagree, say which you followed.

## 2. Pick the map

The head count decides the map. Game cells hold one standing person each; leave room to walk. Use the size table in
`presentation-scene.md`. A council of lords does not fit a 4×4 tent; a court scene needs room for two rows of
officials. When no map fits, propose a larger or variant map (`tools/scene_maps/`, one map per state like the open and
closed castle gates; a Yellow Turban camp is not the Han camp with red Han flags) and stage on the closest existing
map meanwhile, saying so. A missing prop is modelled in 3D in `tools/unit3d/scene_props3d.py` (same camera and cel
shading as the map furniture, a few retouched pixels), never drawn as a flat front-view picture.

## 3. Storyboard before data

Write a plan per stretch: a floor plan (cells, props, who stands where, arrows for walks) and a beat list per node.
Placement by role, not a row:

- seats of rank: the host on the high seat facing the entrance, guests down both sides of the table, newcomers at
  the foot or the door, retinues one cell behind their lord;
- confrontations face each other across open ground; speakers and listeners turn to each other;
- extras in loose groups with their own facing (guards facing out, officials facing the throne, bandits round fires);
- people at rest rest: lords sit at a council, an emperor sits his throne, the wounded lie on mats or slump against
  something, sentries patrol, a crowd murmurs (Chatter) while the speakers talk.

Beats, at least one visible action every two or three lines:

- every action in the narration becomes a beat (enter, walk, kneel, strike the table, pour wine, throw down a head);
- arrivals walk in from the map's edge, groups one after another, not appear in place;
- reactions: emotes (`!`, `…`, `?`), a turn to look at the speaker, a step back;
- the scene's standing effects (campfires, braziers, smoke) start on its first node; outdoors, a Backdrop gives the
  far view the story names (the pass's gate, a burning city);
- a remembered scene inside a line (a flashback) takes Time memory for that line and goes back after.

## 4. Write `assets/scenes/chNN.json`

Hand-staged scenes are the source. Keep each scene's `Nodes` list in story order and complete (a test checks that
every dialogue node is staged). For new story nodes run `tools/scene_maps/wire_story.py`: it adds an empty scene
(map and nodes) for a stretch that has none and never touches staged ones; then stage it. Actors keep where the beats
took them for the rest of the scene, so a later node starts from there. Write from a small script in the scratchpad
with `tools/scene_maps/scene_format.py`'s `write()`, or edit by hand and run `scene_format.py` to keep the layout.

## 5. Check placement

```sh
python3 ../.claude/skills/scene-staging/check_scenes.py c02-3 c02-4   # scene or node id prefixes; none = every scene
go test ./internal/gui -run 'TestScenesData|TestScenarioNodesAreStaged|TestSceneWalksStayOnOpenGround'   # cast names, sprites, factions, steps, every node staged, walks
```

It flags actors closer than 1 cell (their sprites overlap on the iso screen), off the open floor, on a standing prop,
or outside the map picture, at the cast spots and after every beat group (enters, walks and exits played in order). Fix every line for the stretches you staged.
`TestSceneWalksStayOnOpenGround` walks every Move (through its Via, u first then v, as the scene screen does) and
every Patrol both ways on the format 2 scene maps, and fails a walk that crosses a small tile the map's rules close
(`scenemap.Footing`: fences, walls outside gate passages, tents, tables) or climbs between heights. Route round with
whole-cell `Via` corners; a gate's passage is where `scenemap.Walkable` opens the wall, not where the ASCII picture
puts it. Old format maps are not walked.

## 6. Check the screen

```sh
caffeinate -d -i go run ./cmd/_uidemo-vm -ui scenenode -stage c02-31,c02-32,c02-33 -frames 900 -every 30 -out $SCRATCH/s
caffeinate -d -i go run ./cmd/_uidemo-vm -ui scenenode -stage c02-31,c02-32 -frames 900 -mp4 $SCRATCH/s.mp4
```

`scenenode` plays the listed nodes' scenes like a player: each line 2 seconds after its beats have played out. Put the frames on a contact
sheet (PIL), enlarge the cast area (crop round the screen centre, nearest-neighbour ×1.5), and check: nobody overlaps
or stands in a tree or off the floor, everyone the narration names is there, sides are apart, effects stay on, walks
end where planned and read in the right order. Watch the MP4 for movement.

## Done

Record the change in `docs/history.md` (`파일이름: 수정내역 — 의도`), move settled rules into
`docs/presentation-scene.md`, and give the user a page with the storyboard, before/after frames and the engine steps
the stretch still needs.
