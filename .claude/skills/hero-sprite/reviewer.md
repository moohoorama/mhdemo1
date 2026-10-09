# Reviewer brief

You review a hero sprite you did not draw. You judge by looking at renders, not by scripts: a script may count or
list pixels to help you look, but every finding is your judgement of the picture. Do not edit grid files, yaml or
the artists' notes.

Read first: `heroes/<id>/profile.md` (for `_noncombat` ids: the hero's profile §7), `model-sheet.md` and the
artists' notes, `ver4/tools/doteditor/HERO_DESIGN.md` §4.3, `tools/spritetool/assets/SPRITE_ART_GUIDE.md`,
`tools/spritetool/assets/UNIT_ART_GUIDE.md` §2–6. The notes list intended gaps; a gap the profile/notes call
intended is not a defect.

Coordinates: x is the column index after the `NN|` row prefix (the header ruler shows it), y is the row number.

Look at (tools in `ver4/tools/doteditor/heroes/tools`, renders read the grid working copy):
- `python3 render.py sheet <id> --anims <a> --zoom 4` per animation, and `--dirs <d> --zoom 8` where something
  looks off; `python3 render.py frame <id> <anim> <dir> <n>` for 10x with the base beside it.
- `python3 render.py recolor <id> --anims <a>` for every tag group used.

Check, per animation × direction × frame
1. Features: the profile's identity items are there per the visibility table, the same drawing as the model sheet.
2. Theme colours: in each recolor sheet only that group's parts change; the change looks natural (no stray pixels of
   the group elsewhere, no part that should change staying put, shading 1→3 ordered).
3. Dot quality: weapon not cut where it should be continuous, not stamped twice/overlapping; head–body and
   arm–hand–weapon joints connected; no floating or stray pixels; outline 1px and closed; head size constant;
   feet/hooves on the pivot.
4. Motion: the frame sequence reads as its action (wind-up → strike → follow-through; block contact; recoil;
   exhaustion; walking steps) and continues from idle.
5. Occlusion: what is in front is drawn over what is behind for this pose and direction (blocking weapon in front
   of the head where it must be, far arm/weapon hidden behind the body in back views, etc.).

Write `heroes/<id>/review-<N>.md` (N = next free number): a verdict line (통과 / 수정 필요), then findings as
`- <anim>/<dir>/<frame>: <symptom> → <what should change>` grouped by severity (반드시 / 권장), each specific enough
for the artist to fix without asking. Reply with the verdict, the count of findings by severity and the file path.
