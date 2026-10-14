---
name: class-sprite-3d
description: Draws a unit-class (병종) dot sheet by modelling the soldier in 3D (tools/unit3d), rendering every frame and converting it into the dot editor's tag grids with tools/grid3d.py — heads and effects copied from the base sheet. Use for "3d로 그려줘", "3D 변환으로 병종 도트", "<병종>을 경보병에서 3D로 만들어", or when an agent-drawn sheet (class-sprite) would be too slow.
---

# Class sprite by 3D conversion

Used for 중보병 (`heavy_infantry`): the user asked for this route explicitly, so scripts here DO generate pixels.
This is the one exception to "dots are drawn by agents, scripts only do I/O" (hero-sprite, class-sprite); say so in
the report. Anything the conversion gets wrong is fixed by hand (artist agent, row edits) afterwards.

## Steps
1. **Design in words.** What changes against the base class (armour, cloak, shield, colours). The user's wording becomes
   3D parts: ellipsoids/boxes with a material colour from `tools/unit3d/units.py` (`STEEL` silver, `BROWN` leather,
   `BLUE` faction cloth, `LIGHT_BLUE` faction light, `SKIN`). Faction colour = `BLUE`/`LIGHT_BLUE`; a material that
   should be a tag colour must use the ramp keys listed in `grid3d.KEY2TAG`.
2. **Quick preview first (idle frame 0, 8 directions).** Do not touch the repo: a scratch script in the scratchpad that
   imports `tools/unit3d` (`build.build_unit(key, build, head_set, False)`), adds the kit, and puts the base
   class beside it, 6x nearest, like `heroes/heavy_infantry/preview/3d-idle0.png`. Show it, adjust (the user iterates:
   colours, slimmer body, cloak length, shield colour).
3. **Tool setup.** `python3 grid.py new <base> <id> <name>`, `grid.py export <id>`, `grid.py export <base>` (reference
   grids). Add the kit to `grid3d.KITS` (`kit dict` + `decorate_<kit>(p)` that adds the extra meshes for pose `p`).
4. **Convert.** `python3 grid3d.py <id> [--kit K] [--base infantry] [--anims ..] [--dirs ..]` (~2.5 min for everything;
   run in the background). It renders all 6 animations × 8 directions × 4 frames, maps palette keys to tags
   (an unknown key aborts), pastes the base's head box (12 rows, found by the `#j#` plume) and effect pixels verbatim,
   applies `FIXES`, and writes `heroes/<id>/grid/...`. Then `grid.py import <id>`.
5. **Per-tag colours.** `grid.py colors <id> themeA1=#.. ...` (leather → gunmetal etc.); the viewer reads the yaml.
6. **Look at the results** (`render.py sheet <id> --anims <a>`, `render.py frame`; check all 8 directions, not only S/SW/W/N)
   and list the faults. Known weak spots of the conversion:
   - the 3D blade foreshortened away in a pose (infantry attack S 1, SW 1): paste the base's blade pixels via `FIXES`
     (`(anim, dir, n): [(row0, row1, col0, col1, replace_all)]`);
   - a head box misplaced when something sits above the plume (detected by the `#j#` pattern, so only a raised sleeve
     with the same pattern can fool it);
   - back-view cloak as one static rectangle (legs/walk invisible), cloak standing up in `exhausted` (seated pose);
   - 3D shading is noisy at 1:1; needs an artist's clean-up pass (use class-sprite's motion-artist/reviewer).
7. **Report** viewer URL (`viewer.html?unit=<id>`), progress URL, the weak-spot list, and that scripts generated the pixels.

## Files
- `ver5/tools/doteditor/tools/grid3d.py` — the converter (reproduces heavy_infantry exactly).
- `tools/unit3d/` — renderer, kits (`units.py`), poses, heads; see its `DESIGN.md`.
- Working copies (`heroes/<id>/grid`, ignored by git) are regenerated with `grid.py export`; before step 4 run `python3 tools/grid.py export <base> --force` (heads/effects come from `heroes/<base>/grid`). Keep a copy of `units/<id>.yaml` before a big change.

## Stop line
Same as hero-sprite: no `make apply`, no game code (`classes.json`), no commit unless the user asks.
