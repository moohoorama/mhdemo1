---
name: pixel-iso-kit
description: Draws flat-faced map pieces (walls, gates, parapets, room walls, floors, paving, any box-shaped structure) of the ver5 SRPG as pixel art straight onto the 2:1 small-tile grid — ASCII-style dot patterns laid on the faces of boxes and projected so every course steps one pixel up per two across — with assets_candidate/tools/gen_pixel_castle.py (scene castle) and gen_pixel_kit.py (battle walls/gate, room walls/floors), then compares old and new on a page and wires them in. Use for "도트로 성벽", "가로2 세로1로 쌓아", "벽면을 도트로 찍어", "아스키로 벽돌 무늬", "전투맵 성벽/성문 다시", "실내 벽지/바닥 다시", "타일 대각선 맞춰서 그려", or when a 3D render of a structure does not line up with the tile guide.
---

# Pixel iso kit: dot patterns on 2:1 boxes

The user's method (2026-10-24, after the 3D wall kit kept missing the tile guide):
> 현재 타일이 가로2, 세로 1씩 올라가잖아. 이걸 이용, 도트를 아스키 아트식으로 찍고, 그걸 가로2 세로1에 맞춰서 위로 쌓는거야.

Results the user approved: the scene castle (`gen_pixel_castle.py`, 10-24), then battle walls/gate and room
walls/floors (`gen_pixel_kit.py`, 10-28, "훨씬 깨끗하다"), then the courtyard gate shut and open (10-28, "디자인 딱
좋아"). Rejected the same day: a lone four-gated fort and a tall pointed government hall for the castle cell (`o`) —
"기존게 나은데"; the castle site keeps its old picture. Use this for anything made of flat faces. Organic things
(trees, people, props with curves) go other routes (3D render, codex painting).

## The method

- **Grid.** One small tile is a 32×16 diamond: canvas `x = 16(u − v)`, `y = 8(u + v) − z`, z in pixels up. Battle maps
  and scenes share it (a game cell = 2×2 small tiles). One dot along u or v is one pixel across, half a pixel down.
- **Solids.** Every piece is convex bodies (`Body(box(u0,u1,v0,v1,z0,z1), paint, tile)`, hips via `Line.hip`). The
  renderer `gen_pixel_castle.render` casts each pixel's ray, takes the nearest body and face, and asks the body's
  `paint(face, u, v, z, s, h)` for a palette key. `s` is dots across the face, `h` pixels up — so a pattern written in
  `(s, h)` (bricks: `brick(s, h, a, b, w, rows)`) lands on the 2:1 slope by itself. Top faces (`"+z"`) use
  `i = floor(16u)`, `j = floor(16v)`. Floors use the same `(i, j)` dots, one per pixel.
- **Light.** `+v` faces lit, `+u` faces shaded, top faces a third tone (`pc.lit(face)`). Keep both orientations' tones
  matched; a mirrored piece needs its own render, not a flip.
- **Outline.** The renderer inks a pixel beside a nearer one (by more than a quarter tile) or an empty one beside a
  solid one. To join pieces without a seam, render **context bodies** (`tile=-1`) where the neighbours stand — the
  closed sides, and the diagonal between two closed sides (else a dot of ink appears at the corner) — and keep only
  the pixels the piece owns.
- **Cutting.** Big pieces (gates) are rendered whole and cut by owning small tile (`along(u, v)` → index), each
  piece's pivot at its own tile centre, so the game's per-tile depth sort stacks them back together. A piece can come
  out empty (a back tile wholly under a roof): ship a 1×1 transparent picture.
- **Palette.** Name keys in `pc.PAL` (`pc.PAL.update({...})`), 2 tones per material per face plus joint/seam tones.
  Battle stone is the battle map's blue-grey; the scene castle uses the gatehouse stone the user chose.

## Steps

1. Read the current pictures and how the game places them (pivot, footprint, anchor, mask meaning, sort) — e.g.
   `internal/gui/field_native.go` (`smallBlock`, `gateSprite`), `assets/graphics/scene.json` (`footprints`, `walls`),
   `index.json` (`props`, `extra`). Match their size and anchor so only the picture changes.
2. Add the piece to `gen_pixel_kit.py` (or `gen_pixel_castle.py` for the scene castle): bodies, paint functions,
   palette, a `kit()` entry with its tiles.json fields (`layer`, `scale`, `shadow`).
3. Compare before touching the game: `python3 gen_pixel_kit.py --compare DIR` composes the current game pictures and
   the new ones the same way (a castle corner, four rooms, piece strips). Look at it at 4× and fix what shows: ink
   dots at joins, dark grazing roof slopes (make the roof steeper so back slopes face away), tones off between faces.
   Publish an HTML page with old/new side by side (data URIs) and ask the user.
4. Apply: `python3 gen_pixel_kit.py --write` (pictures into `assets_candidate/tiles`, entries into `tiles.json`,
   replaced files removed), then from the repo root `python3 ver5/tools/build_extra.py` (packs `map/extra.png`;
   `extra` overrides `props` by name, so props.png pictures are replaced by giving the same name).
   `build_all.py` rewrites tiles.json from scratch — after it, run both `--write`s again.
5. Where a unit can stand in the piece, cut it by depth, not by tile: what lies behind the cell's centre line goes to
   the back tile's piece and the rest to the front tile's, or a side tile that sorts level with the unit lays the front
   wall under its feet. Units behind castle walls and gates ghost at 30% (`drawItem.hides`).
6. Code only where the shape changed (e.g. the gate became 2×2 pieces: `gateSprite` picks `gatehouse_closed_<axis>_<du+2dv>`
   with the axis read from the game cell's neighbours). Update tests that name pictures
   (`internal/gui/field_state_test.go`), `go test ./internal/gui/`.
7. Check in the game, not only the composite: `go run ./cmd/_uidemo-vm -ui mapfx -frames 360 -at 30,90,150,210,270,330`
   (battle walls, gates) and `-ui scenemap -stage S_house` etc. (rooms). Then update `docs/tools-build.md`,
   `docs/presentation-scene.md` if a rule changed, `assets_candidate/README.md`, and `docs/history.md`.

## Pieces now

| script | pieces | notes |
|---|---|---|
| `gen_pixel_castle.py` | scene `swall2`, `sparapet_{mu,mv,pu,pv}`, `swall2_shade_*`, `sgate2_{open,closed}_{u,v}_{0..9}` | wall 66px, gate 10 tiles, open passage veiled black |
| `gen_pixel_kit.py` | battle `wall_clean_00..15` (mask bits nw −u, ne −v, se +u, sw +v = open sides), `gatehouse_{closed,open}_{u,v}_{0..3}` | wall 16px, slabs on the walkway; gate = one game cell as a walled courtyard, a wide arch in each gate wall under a dark slab on four red posts, leaves shut or swung back (open also draws the pass `p`); cut at the cell's centre line into piece 0 (behind) and 3 (in front), 1·2 empty, so a unit in the gate sorts between them |
| `gen_pixel_kit.py` | rooms `wall_{plaster,lacquer,canvas}_{nw,ne}`, `floor_{wood,carpet,palace,tent}` | wall 34px, 0.12 tile thick outside the cell edge, anchored at the cell's back corner; floor 64×32, pivot at its centre |
| `gen_pixel_kit.py` | battle `fence_00..15`, `village_0..3`, `camp`, `burnt_village_0..3`, `burnt_camp` | one game cell at the map's scale, pivot at its centre (0, 16 from the back corner); fence = post at the centre, two rails out to the middle of each joined side (mask as the walls'), the next cell's arm as context; houses by `house()` (burnt: `ruin()`, a broken shell); villages are four layouts picked by (x+2y) mod 4 (`siteVariants`), each with its smoke in scenery.json `village_<n>`; village and camp override `sites.png` by name, the camp's flag and the village's chimney smoke are scenery.json overlays; burnt = the same bodies without roofs/tents, recoloured by `charred` (approved 10-29) |
