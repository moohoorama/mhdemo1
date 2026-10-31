---
name: script-edit
description: Lets the user edit a chapter's story script in the browser — lines, speakers, explore talk, and the scene directions written as prose against a rendered picture of the scene with places marked (A)~(D) — then carries the saved edits into storyNN.json, the explore files, assets/scenes/chNN.json and docs/fullstory.md. Uses tools/script_edit/build.py, cmd/_scene_render, cmd/_script and tools/script_edit/apply.py. Use for "대본 편집", "대사 고치게 띄워줘", "N장 대본 웹으로", "스크립트 에디터", "내가 대사 수정할게", "저장했어 반영해", or when the user wants to rewrite a chapter's lines or directions themselves.
---

# Script edit

Work from `ver5/`. The user edits; the agent prepares the page and carries the edits into the game. Four steps:
build the script, write the directions and places, serve the page, apply what the user saved.

## Files

| file | what |
|---|---|
| `plan/script/chNN.json` | the editable script: entries in story order (scene nodes, then explore events, then explore talk) and `Renders` |
| `plan/script/chNN.edited.json` | what the editor saved; exists only between a save and the agent applying it |
| `plan/script/chNN.<scene>.<shot>.png` | the scene pictures `cmd/_scene_render` draws |

Each entry has `Key` (node id, or `exploreNN:area:person:kindI`), `Kind`, `Section` (장면 / 이벤트 / 탐험), `Text`,
`Speaker`, `Stage` (directions in prose), `Note`, and `Hint` (a mechanical summary of the beats or of where an explore
person stands, for the agent; the page does not show it). `Fixed` entries (branch, set, grant, join, explore, back,
battle, event links) have no editable text.

## 1. Build

```
python3 tools/script_edit/build.py N
```

Lines always come fresh from the game data. `Stage`, non-fixed `Note`, fixed `Note` with `"NoteByHand": true`, and
`Renders` are kept across rebuilds by key, so rebuilding is always safe. A new chapter starts with empty `Stage`s.

## 2. Places and directions

**Places (`Renders`).** For each scene of the chapter (`assets/scenes/chNN.json`), pick about four places, never
one per person. Group the cells where things happen: the entrance people walk in by, where the main group stands,
where the extras sit, the building someone comes out of. Read the cells off the scene's ASCII picture
(`assets/scenes/ascii/<scene>.txt`, written by `python3 tools/scene_maps/scene_ascii.py dump chNN`; one cell = three
characters, row = v, column = u; see the ascii-layout skill) and the beats in `Hint`. Label
A, B, C, D in the order the scene uses them, and name each place the way the story would (북서쪽 뒷문, 진영 마당).

```json
"Renders": [
 {"Scene": "c10-start", "Shots": [{"Name": "stage", "After": ""}],
  "Spots": [{"Label": "A", "Name": "북서쪽 뒷문", "Cells": [[-1, 6], [-1, 7], [0, 6], [0, 7]]}, …]}
]
```

A shot draws the cast as it stands once every node's beats up to `After` have played; leave `After` empty for the
end of the scene. Add a second shot (for example after the entrance) only when people move so much that one picture
misleads.

```
go run ./cmd/_scene_render plan/script/chNN.json
```

Look at every picture before going on. The labels must sit on the places they name, and the name tags must be on
the right people.

**Directions (`Stage`).** Edit `plan/script/chNN.json` by hand. Turn each node's `Hint` into one or two plain
sentences that use the places: "유비가 (A) 뒷문으로 걸어 들어와 (B)에 선다. 전해가 (C)를 떠나 바삐 (B)로 와서 일행 앞에
선다." Write who walks where, who appears, who turns to whom, and which gesture. Say the scene's opening layout on the
`scene` entry. Leave out speeds, seconds and coordinates. Directions are staging, not narration
(`docs/story-writing.md`). For explore people, write where they stand in words. The places A~D belong to the scene,
not to the explore walk.

Rebuild once to check the JSON (`build.py N`).

## 3. Serve

```
go run ./cmd/_script -ch N        # http://127.0.0.1:8795/   (run in the background)
```

`_scene` already uses 8790. Give the user the URL and say how the page works. Grey italic text is the directions,
large text is the line. Hovering a card shows add, delete and revert buttons. Saving with ⌘S writes
`chNN.edited.json`. The user tells the agent when they are done. If Claude Code reaps the server for low memory, say
so and restart it only when the user asks.

## 4. Apply

```
python3 tools/script_edit/apply.py N --dry   # look first
python3 tools/script_edit/apply.py N
```

The script replaces a changed line wherever its old text occurs exactly once: in storyNN.json or the explore file,
and in fullstory.md. It edits by exact string replacement, so other sessions' uncommitted changes in the same files
survive. Never rewrite those files whole. It lists everything else for the agent to do by hand:

- **Directions changed**: change the beats in `assets/scenes/chNN.json` to match, using the scene-staging and
  ascii-layout skills (whole cells, facing rules). Check them with
  `python3 ../.claude/skills/scene-staging/check_scenes.py <scene>`.
- **New or deleted lines, changed speakers**: add or remove story nodes with new ids, keep the scene's `Nodes` list
  in story order, and move the beats keyed by a removed node.
- **Not found in fullstory**: find the line that means the same and fix it. Narration and explore talk are often
  only summarised there.
- **Edits that leave something meaningless**, such as two branch lines that became identical, a name used before
  anyone said it (`story-historical-plausibility`), or a direction the engine cannot show: ask the user before
  applying. Do not guess.

Then:

```
python3 tools/script_edit/build.py N --absorb   # keeps the edited directions, removes chNN.edited.json
go test ./internal/gui -run TestUIContentFits -count=1
```

`--absorb` matters. A stale `edited.json` left behind would undo other sessions' later changes on the next apply.

Report a table of the lines changed and the files they went to, the things done by hand, the questions, and the
test results. Commit only when asked.
