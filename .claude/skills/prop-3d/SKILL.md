---
name: prop-3d
description: Makes or remakes a story-scene prop or small structure (탁자, 침상, 수레, 우물, 망루 …) of the ver5 SRPG by roughing it out in 3D from boxes, rods and ellipsoids (tools/unit3d/scene_props3d.py), rendering it with the game's cel renderer into dots, retouching the few pixels the render loses, and laying its ground shadow from the model itself (cast_shadow → shadow_mask) so every leg and near edge casts where it stands — then packing it with build_extra.py and checking it in a game capture. Use for "소품 3D로", "구조물 3d로 만들어", "탁자/침상/가구 다시", "다리가 이상해", "그림자가 이상해/떠 있어/안 붙어", or when a 2D-drawn prop (gen_scene_props.py, gen_interior.py) looks wrong in iso.
---

# Scene prop by 3D model

The route the user chose (2026-11-07, war-map table and bed): rough 3D model → cel render to dots → retouch →
shadow cast from the model → pack → game capture. Hand-drawn 2D iso props get legs, corners and shadows wrong;
this route gets the geometry right and keeps the hand only for detail.

## Steps

1. **Look first.** Capture the scene the prop stands in (step 6) and zoom on it, so the fault and the old size are
   clear. Find where the prop is used: `grep -n '"<name>"' ver5/assets/graphics/maps/*.json`. Find its old
   generator: `grep -rn "<name>" assets_candidate/tools/*.py tools/unit3d/*.py`.
2. **Model** in `tools/unit3d/scene_props3d.py` as a function `<name>()` and add it to `PROPS`.
   - Space: one model unit = one 32×16 scene cell edge; +x runs to the lower right (map u), +y to the upper right
     (map v), z up at about 20 px per unit. An officer on foot is 1.9 units tall; a table top ~.65, a bench seat .42,
     a bed platform ~.3. Match the old sprite's footprint unless the user asks for another size.
   - Primitives (`base_model as m`): `m.box(V(centre), V(size), colour)`, `m.rod(a, b, r, colour)`,
     `m.ell(centre, radii, colour)`, `m.poly([V..], colour)`, and `disc()` in the file for flat cylinders.
     Start with `m.meshes = []`.
   - Colours are palette ramps: reuse `P.PLANK`, `P.PLANK_DARK`, `P.PAPER`, `P.INK`, `RED`, `LINEN`, `STRAW`, … or
     add one with `R.add_ramps({hex: 'four palette keys'})` (keys: `python3 -c "import render as R; print(R.COLOR)"`
     in `tools/unit3d`).
   - End with `im, pivot = draw()` (bigger cell for big pieces: `draw(cell=(120, 140), pivot=(60, 110))`).
     The pivot is the footprint centre on the ground — never move it to fix the look; fix the model.
3. **Retouch** what is too fine to model:
   - lines and marks lying on a surface: `on_top(im, pivot, [(x, y, z), ...], key)` projects model points to dots
     (the map table's river);
   - single pixels: `RETOUCH['<name>'] = [(x, y, key), ...]` in trimmed-image pixels (negative counts from the end,
     `'.'` clears).
4. **Shadow:** `return cast_shadow(im), pivot`. It lays every face of `m.meshes` flat along the game's light
   (`SLOPE` .6, .3 per pixel of height) into a mask kept as `im.shadow`; `main()` saves it as
   `assets_candidate/tiles/<name>_shadow.png` with a `shadow_mask` entry in `tiles.json`. The game
   (`sprite.MaskShadow`) softens it exactly as it softens a cast silhouette, so it matches every other shadow.
   Do it for anything with legs, overhangs or depth; a prop without it falls back to the silhouette shadow.
   Props whose outline moves (anims with per-frame shadows) are out of scope.
5. **Preview** before touching the repo's pictures:
   `python3 .claude/skills/prop-3d/preview.py <scratchpad>/p.png <name> [...]` — the render with its softened
   shadow on a floor colour at 5x. `python3 tools/unit3d/scene_props3d.py <name> --review --out=<png>` puts the old
   picture above the new one beside an officer for scale. Show the user and iterate.
6. **Write, pack, check in the game.**
   ```
   python3 tools/unit3d/scene_props3d.py <name> [...]   # ONLY the named props: without names it rewrites every prop
   python3 ver5/tools/build_extra.py                     # packs sprite + shadow mask into map/extra.png, index.json
   cd ver5 && go run ./cmd/_uidemo-vm -ui scenemap -stage <S_scene> -node <u,v> -frames 3 -at 2 -out <scratchpad>/cap
   ```
   Look at the capture zoomed (crop + 2x nearest): legs on the floor, shadow joined to the feet and falling to the
   lower right like the braziers', nothing overlapping. Check each scene that uses the prop, one capture at a time
   (heavy renders back to back slow the machine).
7. **Retire the old picture.** Delete the prop's old 2D generator (and its `reg(...)`/`save(...)` call) from
   `assets_candidate/tools/gen_*.py` and its row in `assets_candidate/README.md`, so `build_all.py` cannot draw it
   back. Never run `build_all.py` for this: it rewrites `tiles.json` from the 2D generators only.
8. **Report** with before/after images (scratchpad paths in full) and which scenes were captured; commit only on request,
   staging only this work's files (the worktree is shared with other sessions; no stash).

## Pieces in the code

- `tools/unit3d/scene_props3d.py`: `cast_shadow`, `on_top`, `RETOUCH`, `PROPS`, `main()` (writes png + tiles.json).
- `ver5/tools/build_extra.py`: packs `<name>#shadow` beside the sprite, lists `shadow_mask: {rect, pivot}`.
- `ver5/internal/gui/assets.go` (`propEntry.ShadowMask`) and `ver5/internal/sprite/render.go` (`MaskShadow`).
- Examples: `map_table()` (apron, four legs, map with markers and an `on_top` river), `bed()`.
