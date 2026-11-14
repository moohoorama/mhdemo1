---
name: isofx
description: Makes or reworks the ver5 SRPG's spell and battle effects (책략 — water 탁류·격류·수진·수룡·해일, fire 초열·업화·화진·화룡·폭염, earth 낙석·산사태·지진·거암·지룡, and new ones) by simulating them in the battle map's iso space — water as a fluid (SPH), fire as hot rising particles, earth as tumbling rock voxels that shatter into grit and dust — and rendering the result as iso pixel art with a renderer made for each element (tools/spellsim.py), shown as draft GIFs on a page, then baked into assets/graphics/effects/spells and played over the spell's whole area in the game. Use for "책략 효과", "이펙트 만들어줘/다시", "물/불/땅 효과", "화계/수계/지계", "효과가 밋밋해/연기 같아/우박 같아", "isofx", or any new effect that would otherwise be drawn with vector shapes or hand-placed ASCII grids.
---

# isofx: simulated iso pixel effects

Spell effects are simulated, not drawn. Particles and rocks move in the battle map's own space, are turned into
voxels and rendered as iso pixel art by a renderer made for each element. The tool is `ver5/tools/spellsim.py`; its
docstring and `ver5/docs/tools-build.md` describe the pipeline, and this skill is how to work with it.

How it got here (2026-11-02), so the same mistakes aren't made again:
- Hand ASCII grids from shape formulas (the `ver5-fx` seed.py way) came out stiff and board-like: "다 무지 별로".
  Keep that pipeline only for small UI-scale bits (rings, sparks, icons; `tools/fx_preview.py` previews them).
- The user's idea: simulate voxels in iso space and render them. Plain particle splats read as hail: "우박 같다".
- Blurring the density into one smooth body with lit shading read as smoke: "연기 같고 물 같지 않다". The cause was
  particles pulling together and sharing velocity, which made them drift like a cloud.
- A real fluid (Clavet's double density relaxation) plus a water renderer gave "너무 좋아. 진짜 물같아".
- Fire and earth followed the same route and were accepted after tuning, as listed under the element sections below.

## World and layout

- x, y run along the map's two cell axes, 32 units to a cell (the 64×32 diamond). h is screen pixels up.
  Screen x = x − y, screen y = (x + y)/2 − h, and +x is screen down-right, the same as `field.Center(u, v)`.
- Gravity is 900 units/s². A unit stands about 40 px tall. The target stands on the cell at the origin.
- Layouts come from the skill's shape in `assets/data/skills.json`. Check it before designing:
  - `single`
  - `cross`: the five cells
  - `line`: `Length` cells from the cell in front of the caster (폭염 is 6). It needs a `_back` twin for up-left;
    mirroring covers the other two directions.
- Names carry meaning; ask when unsure. 탁류 is "탁 하고 친다" (a smack), not muddy water.

## Elements

**Water** (`Water`):
- Simulation: Clavet's double density relaxation keeps the spacing incompressible, and the pull below rest
  density gives surface tension. Viscosity comes from impulses.
- Ground: water that lands hard throws a crown with ¼ of the particles. Water flattens into puddles that drain.
- Units are cylinders the water parts around (`unit()`).
- `render_water`:
  - puddles flat and see-through, with a bright rim and glints;
  - the body splatted stretched along its motion, cut sharp, toned deep with a narrow lit band and a white glint,
    a bright upper-left rim and a dark edge;
  - free drops drawn as short streaks.

**Fire** (`Fire`):
- Heat = start heat (0.55–1, varied per particle; all 1 turns it white and doughy) × (1 − age/life)^1.4.
- Particles rise by their heat, sway in a moving noise field and slow by drag.
- Heavy particles are burning lumps that fall and catch fire where they land. Embers are sparks.
- `render_fire`:
  - colour by heat: white → yellow → orange → red → dark red;
  - smoke where it cooled, a ground glow, hotter where the flame is thick, the silhouette one step darker;
  - embers as streaks.
- Users wanted lots of sparks, around 300 for a big blow. 업화 and 초열 (`soft_embers`) throw them from all through
  the flame (`embers(radius=, h=)`), in batches for as long as it burns, fewer and shorter-lived as it dies so the
  last go out with it. They cool white → yellow → orange and fade, and over the flame they add light to the pixel
  rather than paint over it: opaque sparks with fixed colours stood apart from the fire. Dim red sparks over grass
  read as dirt, so they are dropped.

**Earth** (`Earth`):
- Rocks are angular chunks: blocks cut by twelve random planes. Round bumps looked like Mickey Mouse ears.
  `parts=3` heaps several into one boulder.
- Rocks tumble. A hard landing shatters a rock into pieces (which never shatter again), grit and a dust burst.
  `stay=True` beds a boulder in and spreads a wide dust ring instead.
- `render_earth`: faces lit top-left, a speckle, a dark outline. Dust is a see-through cloud in two tones, and it
  must be dense enough to read on grass.

**Gas, seals, formations** (`spellsim_gas.py`, `spellsim_seal.py`, `spellsim_formation.py`): the status spells.
- Gas (혼란, 대혼란, 독연, 독무) is a see-through cloud. Use flat alpha steps, a fainter outer band and no outline.
  Checker-dithered wisps were rejected ("너무 디더링된듯"), and a dark outline made the clouds read as rocks or
  bushes.
- Poison is a purple body with lime tops. Green alone vanishes into the grass.
- Seals:
  - talismans: yellowed paper plates with red marks, no characters;
  - chains: Verlet links that wrap the unit's cylinder, drawn behind the body where they pass behind it;
  - stakes, plus `Earth` for dust.
- 팔진도 bakes as three pieces:
  - `spell_eightfold`: the formation, drawn on the field's ground layer under the units
    (`spellOnGround` → `groundSpells`);
  - `spell_eightfold_ally` and `_enemy`: put on each unit the spell reaches, by side (`spellPiece`). Any spell
    with `<look>_ally` and `_enemy` sprites gets the same handling.

**Shared**:
- `Particles.guide` moves particles along a path (`helix` is the whirl used by all three dragons), then flings them.
- `on_landing(cells, fn)` sets off an effect in each cell when the falling stream first reaches it. The dragons use
  it: 화룡 → 초열 per cell, 수룡 → a small 격류 per cell.
- `fade` empties the effect at the end.

## Steps

1. **Read the ask as motion and timing.** For each spell, say what moves, from where, when it lands and how hard.
   If they gave a reference ("업화가 연달아 파도타기 타듯"), reuse that spell's function (`scorch`, `geyser`).
2. **Drafts.** For an open direction, write two or three genuinely different functions per spell. Run them with
   `uv run --with numpy --with scipy --with pillow python tools/spellsim.py <names>` (from `ver5/`).
   - Each gives `reports/spellsim/<name>.gif` (3×, over grass, cells outlined, two infantrymen for scale) and
     `<name>.png` (every frame).
   - Look at the PNGs yourself first. To compare many at once, montage a few frames per draft.
   - Publish the GIFs on one page and let the user pick. Never put the user's choice in yourself.
3. **Tune in rounds.** Fix the worst thing and run again. Typical fixes:
   - flies apart or floats: cohesion is too strong, or rest density doesn't match the packing;
   - spreads too far: friction and landing kick;
   - too high: launch speed, since height goes with v², so 0.7× v gives half the height;
   - doughy white fire: lower the start heat;
   - invisible dust: raise its alpha and amount;
   - clipped crop: stray drops; the bake and previews cut beyond ±190 px.
4. **Bake.** Water, fire and earth bake in 2× slow motion (`ELEMENT_SLOW`): at their own speed they were over too
   soon. Preview with `--slow 2` to see the game's length. Run `tools/spellsim.py --bake` to write one sheet per spell, `assets/graphics/effects/spells/<sprite>.{png,json}` (the game reads a spell's sheet the first time it plays). `baked()` lists
   everything that goes in: this file's spells, plus the picks from the other modules.
   - Frames are cropped tight, each with its own anchor, and packed 2048 wide.
   - `hit` is when the blow lands: the first fast landing, or set `s.hit` yourself, as for a geyser whose blow is
     the burst.
   - Add a new spell to `SPELLS` and `SKILL` (sprite `spell_<skill id>`), or for another module's spell to
     `baked()`.
5. **Wire.** A skill whose `spell_<id>` exists plays it automatically, through `effectKind`, `swing` (holds the
   damage until `Hit`), `impact` (no per-target sprite) and `drawSpell`. Only add:
   - shake (`spellShake`);
   - side-mirroring for looks that come from the caster's side (`spellFromSide`);
   - a line's direction handling, if it's a new line spell.
6. **See it in the game.** Capture one run of a temporary demo that casts the spells on troops, with troops placed
   round the target for area spells. Copy the pattern of `cmd/_uidemo-vm` with a `uidemo_zz_tmp*.go`.
   - Target a `#` troop: an officer can trigger a duel event.
   - Reset XP: level-up windows hide the moment.
   - A critical adds 0.75 s of charge before the cast.
   - Report with the page link and captures. Heavy runs slow the user's machine, so one run, frames plus an mp4.
