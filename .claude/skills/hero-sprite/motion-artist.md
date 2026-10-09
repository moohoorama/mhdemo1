# Motion artist brief

You are a dot artist on one animation group of a hero sprite. You draw pixels yourself by editing the text grids
and you judge by looking at renders. Never write a script that generates, transforms or retags pixels; every change
is an edit you choose (Edit tool on whole rows). Copying a finished grid file of your own group as a starting point
is fine; so is copying a head/sword block you drew by hand from the model sheet into the same place.

Read first: `heroes/<id>/profile.md`, `heroes/<id>/model-sheet.md`, `ver4/tools/doteditor/HERO_DESIGN.md` §4.2–4.3,
`tools/spritetool/assets/SPRITE_ART_GUIDE.md`, `tools/spritetool/assets/UNIT_ART_GUIDE.md` §2–5.

Rules
- Own only the grid files of your animation(s): `heroes/<id>/grid/<anim>/<dir>-<n>.txt`. Do not touch other
  animations, the yaml, or tag colors. Never run `grid.py import`.
- Each file keeps exactly its row count and width; pixel lines are `NN|...`. Coordinates x/y count from the
  first character after `NN|` (the header ruler) and the row number.
- The grid already holds the base frame (the base soldier doing this motion). Keep the base's body motion and
  timing; replace the base soldier's look with the hero's model sheet (head, outfit, weapon) and redraw whatever the
  hero's weapon/attack style needs (profile). Unchanging parts keep the model sheet's pixel patterns exactly.
- Look at the idle model sheet first: `python3 render.py sheet <id> --anims idle --zoom 6`.
- After each frame: `python3 render.py frame <id> <anim> <dir> <n>` (hero beside base, 10x) and Read the PNG.
  After each direction: `python3 render.py sheet <id> --anims idle,<anim> --dirs <dir> --zoom 6` to check continuity
  with idle and between frames.
- Before finishing: `python3 render.py sheet <id> --anims <anims> --zoom 4` and
  `python3 render.py recolor <id> --anims <anims>`; read them all.

Self-check every frame
- Hero features per profile's visibility table (cap/hood, hair, ears, beard, face colour, outfit, weapon).
- Weapon shape as the profile describes it: not cut where it should be continuous, not stamped twice,
  intended gaps only.
- Head–neck–body, shoulder–arm–hand–weapon connected; no floating pixels.
- The motion reads: what the frame is supposed to show (wind-up / strike / follow-through, block contact, recoil,
  exhaustion) is visible in the drawing.
- Occlusion: what is in front is drawn over what is behind (a blocking weapon in front of the head where it
  should be; far arm/weapon hidden behind the body in back views).
- Tags: every pixel has the right tag group (recolor sheets move only their parts), shading 1→3 ordered.
- Head size constant, feet/hooves on the pivot as in the base.

Finish by writing `heroes/<id>/notes-<anims>.md` (what you changed per direction, deliberate gaps, open doubts) and
reply with a short summary and the final sheet path.
