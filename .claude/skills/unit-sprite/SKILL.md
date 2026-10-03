---
name: unit-sprite
description: Creates or changes unit sprites (병종 스프라이트) in this repo — new units/heroes, outfits, heads and faces, attack/hit/idle motions, effects, faction colors — with the tools/unit3d 3D cel-shaded pipeline. Use for requests like "병종 추가", "창병/기병 만들어줘", "관우 모션", "공격 모션 후보", "진영색", "스프라이트 다시 그려줘".
---

# Unit sprite workflow

The standards live in the repo; this skill is only the procedure. Read before changing anything:

1. `tools/spritetool/assets/SPRITE_ART_GUIDE.md` — common sprite rules
2. `tools/spritetool/assets/UNIT_ART_GUIDE.md` — unit art standard (style, faces, motions, effects, faction colors, pitfalls)
3. `tools/spritetool/assets/ver2-units/README.md` — current set: units, selected attacks, backups, cells
4. `tools/unit3d/DESIGN.md` — how the code is organized

## Where to change what

| Change | File |
|---|---|
| Body parts, outfit colors, weapons, horse, per-row poses | `tools/unit3d/units.py` (`KITS`, weapon drawers, `cavalry`) |
| Attack keyframes, swing waypoints, effect geometry, chosen style | `tools/unit3d/attacks.py` (`SWORD`/`SPEAR`/`BOW`, `SELECTED`) |
| Heads, faces, beards, expressions, team-colored head pixels | `tools/unit3d/heads.py` |
| Shading, outlines, palette ramps and extra colors | `tools/unit3d/render.py` (`RAMPS`, `EXTRA`) |
| Translucent effect drawing | `tools/unit3d/effects.py` |
| Faction hue/chroma/lightness | `tools/unit3d/factions.py` |
| Unit list, durations, output | `tools/unit3d/build.py` (`UNITS`, `DURATIONS`) |
| Viewer unit list | `ver2-units-preview.js` (`UNITS`) |

## Steps

1. **Candidates first.** For a new look or motion, render the SW view only and offer 4 candidates
   (`python3 tools/unit3d/attack_candidates.py` for attacks; for looks, render SW idle frames into `output/`).
   Let the user pick. Keep the others as backups (definitions stay in code; only `SELECTED` changes).
2. **Look at the pixels.** Upscale (nearest, 6–10x) and read the image yourself before reporting.
   Check the face rule (rim → shadow row → 2px eyes), weapon visible in the SW wind-up, outlines, team keys only on team parts.
3. **Eight directions, then everything.**
   - `python3 tools/unit3d/build.py` → `output/ver2-units.png` (8-direction idle review; does not overwrite sheets)
   - `python3 tools/unit3d/build.py --full [--unit KEY]` → sheets + `frames.json` (~1.5 min per unit; run in background)
   - Generation asserts the idle ground row and the working-canvas edge; a failure means a pose or effect overflows.
4. **Viewer.** `make preview-knights`, open `http://127.0.0.1:8765/ver2-units-preview.html#UNIT/DIR/ACTION/FACTION`.
   For your own check, serve on another port and screenshot with headless Chrome.
5. **Record.** Update `ver2-units/README.md` (units, attacks, cells) and, when a rule changed, `UNIT_ART_GUIDE.md`.
   Then refresh the packaged assets: `python3 ver3/tools/build_assets.py` (one image per unit + `ver3/asset/assets.yaml`);
   a new unit also needs an entry in `ver3/demo.py` `STATS`/`SIDES` to appear in the demo.
6. Do not wire sprites into `assets/` or the game, and do not commit, unless the user asks.

Codex image generation (`codex-image` skill) is for illustrations, not for these sprites.
