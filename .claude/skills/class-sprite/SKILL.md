---
name: class-sprite
description: Draws a new unit-class (병종) combat dot sheet derived from an existing class sheet — e.g. 적병 from 경보병, 무도가 from 적병 — using the hero-sprite agent pipeline with the cheaper row-range editing method, then wires it into classes.json. Use for "병종 도트 추가", "<병종>을 <기존 병종> 기반으로 그려줘", "무도가/중보병/근위대 도트".
---

# Class sprite (병종 도트) workflow

A class sprite is a combat sheet with id = class id (`bandit`, `brawler`), not an officer. It follows the
`hero-sprite` skill (profile → model sheet → user gate → motion artists → reviewers → import) with the changes below.
Read `hero-sprite/SKILL.md`, `motion-artist.md` and `reviewer.md` first; the stop line is the same.

## 1. Profile
- Base = the nearest existing sheet (`units/<base>.yaml`), not necessarily the infantry sheet. Keep its head size, body
  height, feet and pivot (56×46, pivot 28,38); only the requested parts change.
- Decide and write down in `profile/<id>.md`: what is removed (weapon, armour, hair/headgear), what the faction
  colour moves to (e.g. waist sash when the headband goes), tag assignments, per-motion description, per-direction
  visibility. A request such as "근육 팔" must be turned into pixel terms (e.g. +1px arm width, forearm wedge, biceps
  bump with a highlight row and a shadow row) in the profile.
- Check `ver5/assets/data/classes.json` for which class uses which `Sprite`; a missing class means nothing to wire yet.

## 2. Model sheet (idle) and the user gate
- One artist draws idle 8 directions × 4 frames. Show the 8-direction sheet to the user and wait for approval;
  feedback goes back to the same artist.
- Typical feedback seen: arms/pose too stiff (ㄷ-shaped guard), E/W must be true side views, NE must not just copy NW,
  breathing should move the whole upper body (duplicate/drop one chest row), a missing ear. Ask for these in the first brief.
- Once approved, tell every motion artist the idle is final.

## 3. Motion artists — efficient method (user approved)
Brief every motion artist with:
- Edit **one contiguous changed row range per frame** in one call; do **all 4 frames of a direction per turn**;
  render **one sheet per direction**, not 10× per frame; one full check at the end. Do not retype whole frames.
- Unchanged body/head/legs rows come from the approved idle by verbatim row copy (a copy helper that moves rows only is
  allowed; scripts must never generate, transform or retag pixels). The orchestrator must tell the user which copy
  helpers were used.
- Poses that need real drawing: attack (hardest, usually 3 review rounds), block back views, exhausted back views.
  Give these artists the explicit weak spots from the previous class (stray skin pixels on cloth, outward fist stubs,
  head hiding the biceps, no torso twist, back views reading as a flex pose).
- Skip recolor images for groups that are absent (the renderer leaves stale PNGs for empty groups); confirm absence by grep.

Groups in parallel: walk · attack · block · hit+exhausted. A fresh reviewer per group, ≤4 rounds, findings back to
the owning artist.

## 4. Import and report
- `python3 tools/grid.py import <id>`, serve `ver5/tools/doteditor` (`python3 -m http.server <port>`; never kill
  an existing server on another port), screenshot `viewer.html?unit=<id>` with headless Chrome, report the viewer URL
  and the progress URL, remaining 권장 items and known weak spots, and the token use per artist/reviewer.
- Do not run `make apply` unless the user asks (the user ran it for bandit themselves).

## 5. Wiring (only when asked)
1. `make apply` → `ver5/assets/sprites/<id>.spr`
2. `ver5/assets/data/classes.json`: class `"Sprite": "<id>"` (a class must exist first; stats, terrain, attack, promotion
   chain are a design task)
3. Golden image: `python3 ver5/tools/spritebuild/golden.py` → `ver5/internal/sprite/testdata/units/<id>.png`
4. `cd ver5 && go test ./internal/sprite ./internal/content ./internal/gui`
5. Deleting a class sprite removes `units/<id>.yaml`, `assets/sprites/<id>.spr`, the golden png and `heroes/<id>/`
   (confirm the paths with the user first; tracked files are recoverable with `git restore`).

## Open classes (as of 2026-10)
흉적/의적 still use the infantry sprite; 중보병/근위대 share the 경보병 sprite (plan: draw 중보병 first — bigger shoulder
pads, longer cuirass, cheek guards — then derive 근위대 from it); 무도가→권사→권성 have no class entry yet.
