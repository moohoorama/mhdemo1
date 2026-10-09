---
name: hero-sprite
description: Draws officer (장수) dot sprites in the ver4 dot editor format from the 경보병/경기병 base — combat sheets (8 directions × idle/walk/attack/hit/exhausted/block) and non-combat sheets (NW·SW idle, two steps, one action) — with feature extraction, agent drawing, independent review and the viewer.html check. Use for requests like "유비 경보병 걷는장군", "관우 경기병 장군", "장비 비전투", "장수 도트 다시 그려줘".
---

# Hero dot workflow

The design and rules are in `ver4/tools/doteditor/HERO_DESIGN.md`; read it first, then
`tools/spritetool/assets/SPRITE_ART_GUIDE.md` and `tools/spritetool/assets/UNIT_ART_GUIDE.md` (sections 1, 3, 4, 6).
Drawing and review are done by agents looking at the pictures. The scripts only move pixels and render; never write
code that edits or judges the drawing.

## 0. Parse the request

| Request | Base | Output id |
|---|---|---|
| `<장수> 경보병 [걷는장군]` | infantry | `<id>` |
| `<장수> 경기병 [장군]` | cavalry | `<id>` |
| `<장수> 비전투` | infantry (dismounted, weapon removed) | `<id>_noncombat` |

If a combat request does not name the base, ask. `<id>` is the officer key in `ver4/internal/gui/view.go` `officers`
(유비 `liubei`, 관우 `guanyu`, 장비 `zhangfei`, 여포 `lubu`).

## 1. Tools (`ver4/tools/doteditor/heroes/tools/`, run from there)

```
python3 grid.py new <base> <id> <이름> [--noncombat --steps L,R]   # units/<id>.yaml from the base
python3 grid.py export <id> [--anim A] [--dir D] [--frame N] [--force]
python3 grid.py colors <id> themeA1=#rrggbb ...                      # per-hero tag colors (not faction1-3)
python3 render.py frame <id> <anim> <dir> <n>                        # 10x, hero beside the base frame
python3 render.py sheet <id> [--anims a,b] [--dirs SW,NW]
python3 render.py recolor <id> [--groups themeA,faction] [--anims ...]
python3 grid.py import <id>                                          # grid working copy -> units/<id>.yaml
```

- The working copy is `heroes/<id>/grid/<anim>/<dir>-<n>.txt`: one character per pixel, legend in the header,
  pixel lines `NN|...`. Edit those files (Edit tool on whole rows). `render.py` reads the grid directly, so parallel
  artists on different frames never touch the yaml. Only the orchestrator runs `import`, once at the end.
- Look at every render with the Read tool before deciding anything.
- Grid coordinates: x counts from the first character after `NN|` (the header ruler), y is the row number.
- faction1–3 colours are the game team keys and never change per hero; a hero colour that should follow the faction
  is drawn with faction tags (e.g. 관우 녹포), everything else uses themeA/B/C with per-hero ramps.

Progress page: `python3 live.py <ids…> --every 30` (run it through the Monitor tool; each line is a progress event)
renders the grid working copies to `heroes/live/` for `progress.html`. Serve `ver4/tools/doteditor` with
`python3 -m http.server <port>`. Every progress report to the user carries the progress URL and the viewer URL of
each finished unit.

## 2. Agents

Run them with the Agent tool; each prompt names the hero id, base, profile path and the frames it owns.
Artists follow `motion-artist.md`, reviewers follow `reviewer.md` (both in this skill folder). Tell every agent to
keep its helper scripts in its own scratchpad subfolder — the scratchpad is shared and agents overwrite each other.

1. **Profile** (`heroes/<id>/profile.md`): from the old sprite (`tools/spritetool/assets/ver2-units/<id>/`, look at
   the PNGs), the civilian sprite (`ver4-civilians/`), portraits (`ver4/assets/graphics/portraits/`), `tools/unit3d`
   head/beard/kit definitions and `ver4/design.md`. Use the template in HERO_DESIGN.md §4.1, including the tag
   assignment table, weapon shape (which gaps are intended) and the per-direction visibility table.
2. **Model sheet artist**: sets tag colors, draws idle frame 0 in every row, then idle 1–3. Report the idle sheet.
   **User gate:** the orchestrator looks at it, then shows the user the idle 8-direction sheet (and the progress/viewer
   URL) and waits for approval. Feedback goes back to the model sheet artist and the sheet is shown again. Do not
   start motion artists (step 3) or the non-combat artist before the user approves the idle sheet.
3. **Motion artists** (parallel, one per group: walk · attack · block · hit+exhausted; non-combat: one artist):
   start from the base frame already in the grid, keep the model sheet's head/body/weapon pixels, redraw what the
   motion needs. Render each edited frame and the group's sheet; fix until it reads right.
4. **Reviewer** (fresh agent, never the artist): start one per animation group as soon as its artist reports done.
   Checks per HERO_DESIGN.md §4.3 and writes `heroes/<id>/review-<anim>-N.md` as `anim/dir/frame: symptom`.
   Send findings back to the owning artist (resume it with SendMessage), then resume the same reviewer for the next
   round; at most 4 rounds. Small 권장 leftovers (1px outline gaps) still go back to the artist.
5. **Orchestrator**: when every animation of a unit has passed, `grid.py import <id>`, open `viewer.html?unit=<id>`
   and screenshot it with headless Chrome; report the viewer URL. A non-combat unit is imported as soon as it passes.

User changes mid-run:
- A model-sheet change (outfit, head, stance) goes to the model sheet artist first; when its idle is final, send
  "apply v2" to every motion artist and re-review every animation.
- A tag-group decision (e.g. robe → faction) is applied once, after all artists of that unit finish, by a single
  agent that decides per frame which pixels change (rider vs horse tack) and verifies with recolor sheets — never a
  blind character swap while artists are still editing.

After an interruption (usage limit, stop): check every grid file still parses (`dots.apply_grid` per file), then
resume the stopped agents with SendMessage telling them to re-read their files first, and re-arm the monitor.

## 3. Stop line

Stop at the viewer. Do not run `make apply`, edit game code, or commit unless the user asks.
