---
name: ascii-layout
description: Places people and units on whole cells by drawing and editing ASCII pictures — one cell to three characters (`...` floor, `###` wall, `Cao` 조조, `Tbl` table) — for story scenes (assets/scenes/chNN.json via tools/scene_maps/scene_ascii.py) and battle stages (stageNN.json via tools/mapgen/stage_ascii.py), then loads the picture back and checks it. Use for "배치 바꿔줘", "자리 옮겨", "장면 배치", "전투맵 배치", "유닛 배치", "아스키로 보여줘", "정수 칸으로", when staging a new scene or stage, or before reporting where people stand.
---

# ASCII layout

Work from `ver5/`. Placement is edited as a picture, not as coordinates: dump the picture, move the three-letter
codes with Edit, load it back, check. Story scenes put people on whole cells only (`docs/presentation-scene.md`,
"장면 맵"); battle stages always do.

## Pictures

One game cell is three characters; rows run along v (scenes) or Y (stages), columns along u or X, the north corner
at the top left. The first line gives the origin cell of the top-left character.

```
# c03-y0  S_tent_lux  ch03  origin -1 -1
Pil###############Pil
###............BnrJar
###Cdleu1...Tbl...Jar
######YShTbl...Sol...
```

| Code | Meaning |
|---|---|
| `...` | open floor or grass |
| `,,,` `___` | wasteland, castle floor (stages) |
| `###` `~~~` `===` `^^^` `[ ]` | wall or ground nobody stands on, water, bridge, cliff, gate wall |
| `TRE` `VIL` `CMP` `CST` `PAS` `BRG` `FNC` `FIR` `EMB` | stage structures and states, as srpg-cli draws them |
| `Cao` `Liu` `Gua` `Zha` `XHD` … | named officers (`OFFICER` in scene_ascii.py, else the id's first three letters) |
| `in1` `gu2` `tu3` `so1` | extras and template troops, lower case with a number |
| `pl1` `pl2` … `p10` | a stage's deployment slots, in the order of `Allies` |
| `Tbl` `Dsk` `Brz` `Tnt` `Bnr` `Pil` `Hse` `Tre` `Trc` … | map objects and torches; `load` never moves them |

Under the picture: a legend line per code (who, sprite, pose, off-stage; for stages the list entry `Enemies[3]`,
side, group, policy), then the walks of each story line (`# c03-32: Liu -> 9 3`) or the stage's areas.

## Story scenes

```sh
python3 tools/scene_maps/scene_ascii.py whole ch03          # any fractional spots → nearest free whole cell
python3 tools/scene_maps/scene_ascii.py dump ch03           # assets/scenes/ascii/<scene>.txt for each scene
# edit assets/scenes/ascii/c03-31.txt: move codes, keep every code exactly once
python3 tools/scene_maps/scene_ascii.py load c03-31         # sets those cast members' At
python3 tools/scene_maps/scene_ascii.py dump ch03           # redraw after any JSON change
python3 ../.claude/skills/scene-staging/check_scenes.py c03-31
go test ./internal/gui -run 'TestScenesData|TestScenarioNodesAreStaged'
```

- `load` changes only opening spots (`Cast.At`). Walks (`Move`, `Enter`, `Patrol`) are listed as comments; change
  them in the JSON with whole numbers and dump again.
- Keep two people out of one cell and off the codes of standing objects; `check_scenes.py` reports what is still
  wrong. A seated person (`Pose: sit`, `throne`, `lie`) belongs on the cell of their seat, mat or throne: those
  objects stand at fractional spots on older maps, so put the person on the seat's cell and check the screen.
- Effects (`fire`, `bigfire`, `smoke`) and scene `Prop`s sit on map objects and keep their fractional spots.
- Off-stage cast (entering later, or heard from off screen) is not drawn; its cell is in the legend.
- A chapter another session is editing (check `git status`) is left alone.

## Battle stages

```sh
python3 tools/mapgen/stage_ascii.py dump B01                # tools/mapgen/layouts/hero/stage01.txt
# edit the picture: move unit codes only
python3 tools/mapgen/stage_ascii.py load B01                # sets X, Y of Allies / Party / Enemies entries
python3 tools/mapgen/stage_ascii.py dump B01
```

- Terrain comes from the map document (`maps/mapNN.json`); change it in `tools/mapgen/layouts/hero/mapNN.txt`
  and `tools/mapgen/mapgen.py`, not in the placement picture.
- After a move, run the stage-debug skill's numbers (`go run ./cmd/_balance -stage B01 -seeds 20`, `-player ai`,
  `go test ./internal/ai -run Rush`) and the tests that pin cells (`go test ./internal/core/...`).
- `load` refuses a stage file that is not plain indented JSON instead of reformatting it.

## Reporting

Show the picture (or the changed rows) in a code block in the reply, in Korean, with what moved and why, then the
check results and a game capture for scenes (scene-staging step 6) or the numbers for stages.
