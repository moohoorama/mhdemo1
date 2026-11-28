---
name: faction-look
description: Makes a faction or group of the ver5 SRPG read as itself at a glance (황건적, 서량군, 남만, 산월, 백파적 …) across five layers — non-combat dots for its named leaders (civ3d 3D conversion, hair/headgear/robe, kept clear of their Look.Palette), the mark on its troops' class sheets (headbands, plumes, hand-edited frame by frame), its faction colours in battle and in scenes (Colors, OfficerColors, PlainExtras), its stage (camp or hall props laid as map objects), and its voice (Shout cries over heads, a crowd falling silent first) — then checks them in game captures and tests. Use for "황건적 느낌이 안 나", "<세력>답게", "<세력> 느낌 살려", "병사 표식", "머리띠/깃발 색", "구호 외치게", "진채 만들어", or when a new faction appears in a scene, the title reel or a battle and does not stand apart.
---

# Faction look

Work from `ver5/`. Read `docs/presentation-faction-look.md` first: it is the worked 황건적 case (2026-11-28) — each
layer, its tools, the commands and the traps met. The scene format is `docs/presentation-scene.md`, sheets and tag
groups `docs/presentation-sprite.md`. Report to the user in Korean.

## 1. Find what does not read

Capture the faction where it shows: its scenes (`uidemo-vm -ui scenenode -stage <노드>`), the title reel shot
(`-ui titlereel -node <컷>`), a battle (`-ui battleview -stage <스테이지 ID>`), and a trial scene file at zoom 2
(`-ui scenefile -stage <file.json>`; give extras a `#` in their names). One capture at a time (`lockf` on the
scratchpad's capture.lock). For each of the five layers note what a viewer sees now and what would make it read.
Before building, show the user ideas, or a mock on a page over a real capture, and let them choose.

## 2. Build layer by layer

1. **Named leaders** — `tools/doteditor/tools/civ3d.py` `STORY` entries (head from `story_heads.py`, kit parts,
   `work`/`ACTION` for a characteristic motion). Look up the person's `Look.Palette` first and move any part the
   palette would recolour into free groups (`KEY_TAGS`, horse tags on a sheet without a horse). Render the sheet,
   build the `.spr`, set `Look.Scene`, regenerate the golden PNGs.
2. **Troop mark** — on the class sheets the faction's extras use. Edits are drawn and checked frame by frame by
   looking (never a rule-based transform over all frames); one agent per sheet, 8 directions × 6 motions × 4 frames,
   before/after picture saved. Classes with no room for the mark (a helmet) are swapped out of the crowds instead.
3. **Colours** — `assets/data/game.json` `Factions`: troops `Colors`, leaders `OfficerColors`; scene extras get the
   leaders' shades unless `PlainExtras` is set. Change one faction, not the rule for all.
4. **Stage** — props as map objects in a map copy made by a script (`tools/title_reel/turban_camp.py` pattern), not
   scene `Prop` steps (those float bright over night and dusk). A scene cell is 64×32 px at zoom 1.
5. **Voice** — `Shout` / `ShoutSize` steps (leader ~34, those beside ~24, a few of the crowd ~15, front ranks
   first), crowd `Chatter` set to `"-"` just before. The words are data strings, so a translation changes only them.

## 3. Check and record

- Game captures of every place the faction shows, at zoom 2 for the dots and in place (zoom 1, film) for the reel.
- `go test -p 1 ./internal/sprite/ ./internal/content/` and `./internal/gui/ -run 'TestTitleReel|TestScene'`.
- Results in `~/orca/mhdemo1/result/YYYYMMDD/<일>/`, an entry in `docs/history.md`, and anything new learnt added to
  `docs/presentation-faction-look.md`.
