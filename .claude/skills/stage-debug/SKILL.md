---
name: stage-debug
description: Checks and tunes a battle stage of the ver5 SRPG in three passes — numbers (cmd/_balance at the level policy), a text playthrough (srpg-cli with numbered menus and letter maps) for story order and placement, then GUI capture (_uidemo-vm) for what only the screen shows. Use for "스테이지 디버깅", "난이도 확인", "B0n 밸런스", "스테이지 이벤트가 제대로 나오는지", "단순 AI로 깰 수 있는지", or after editing a stage file, its map or its story nodes.
---

# Stage debugging

Work from `ver5/`. Read `docs/stage_lv_balance.md` first (levels, enemy count, what the numbers must reach) and the
stage's part of `docs/conceptual_scenario.md` (what the story says must happen and in what order). Report in Korean
with the evidence (numbers, CLI excerpts, frames).

## 1. Numbers — `cmd/_balance`

```sh
go run ./cmd/_balance -stage B03 -seeds 10              # careless player: nearest enemy only
go run ./cmd/_balance -stage B03 -seeds 10 -player ai   # the game's own AI
go run ./cmd/_balance -stage B03 -seeds 1 -show         # unit numbers at the start + retreats/duels/damage per round
go test ./internal/ai -run Rush -v                      # B01→B03 with growth carried over (the real campaign)
```

- It starts the player's officers at the policy level 3 + 2×(n − 1). Targets: rush ≥ 4/5 on the first three stages,
  ai ≥ 4/5 everywhere, officers +1.5~2.5 levels.
- When it loses, read `-show`: who retreats first, who dealt the blow. Typical causes found so far: a strategist's
  magic on a low-Mind officer (Li Ru 622 on Zhang Fei), an L+1 officer hitting for 40% HP, everything engaging at
  once, a duel that needs an officer who already retreated.
- Fix in this order: placement and groups (`react` posts along a narrow road so the player meets one group at a
  time, `hold` rear guards), a story event that changes the fight (a withdrawal, a sortie, a fire that drives troops
  out), the map's shape (narrow and long splits the enemy), and only then levels and counts within the policy table.
  Write any departure from the table and its reason in `docs/history.md`.
- Keep variants in a scratch script that rewrites the stage file from a saved copy, so each run starts from the same
  base. `cmd/_m8check` plays at the characters' own levels, not the policy's; use it only to see the event order.

## 2. Text playthrough — `srpg-cli`

```sh
go build -o $SCRATCH/srpg-cli ./cmd/srpg-cli
(printf '1\n'; for i in $(seq 1 30); do printf '\n'; done; printf '1\n') | $SCRATCH/srpg-cli -data $SCRATCH/cli -slot 9
printf '4\n4\n4\n' | $SCRATCH/srpg-cli -data $SCRATCH/cli -slot 9   # continue from the slot: end three turns
```

- Numbered menus: the title (1 시작하기 · 2 불러오기 · 3 설정), Enter for the next line, `(n)` to pick an officer, then
  이동 → `x,y` (cells marked `o`), 공격/병법/도구 → numbered targets with HP·MP·stats·traits·position and the
  predicted damage. `-slot 9` loads the slot at start and saves it when the input ends, so a few moves per run.
- Maps are four columns a cell: three letters and a side mark (`' '` player, `+` AI ally, `!` enemy). `Liu`/`Gua`
  officers, `lin`/`arc` troops, `FNC` fence, `FIR` flame, `EMB` ember, `CMP` camp, `###` wall, `[ ]` gate. North is the
  top-left corner (0,0), south the bottom-right.
- Read for: the story lines in the agreed order and one speaker at a time, scene casts where the story puts them,
  armies in formation (officers in front, rows), the first turn's cutscene before its result, enemies reacting when
  they should, a passive player still seeing the story through, sub-objectives reachable.

## 3. Screen — `_uidemo-vm`

The CLI shows rules and order, not timing or drawing. Capture what the player sees:

```sh
caffeinate -d -i go run ./cmd/_uidemo-vm -ui <demo> -frames 2400 -every 30 -size 2560x1700 -half -mp4 $SCRATCH/x.mp4 -out $SCRATCH/x
```

- To play a stage's opening from the story, add a temporary demo `internal/gui/uidemo_zz_tmp*.go` that starts a new
  game, pages the dialogue (`g.advanceDialogue()` every few frames), sends `start` at the battle node and, a couple
  of seconds later, answers the briefing the way a player does: press the full-screen button the drawn modal
  registered (`g.buttons`, rect = the whole UI). Never call `closeModal()` directly — that skips what the player
  must go through and once hid a deadlock (the briefing waited for the replay, the replay for the briefing).
  Delete the demo afterwards. `-ui deploy -stage B03` shows a stage's map at deployment.
- Put the frames on one contact sheet (PIL) and look at it, then enlarge the key frames. Check: the cutscene plays
  before its result (fire, positions), bubbles one at a time, night light round fires, torches and units, briefing
  first and the opening after it.

## When done

Record changes in `docs/history.md`, move settled rules into the classified docs, and give the user a page with the
numbers, CLI excerpts and frames.
