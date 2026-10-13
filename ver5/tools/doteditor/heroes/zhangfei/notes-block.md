# 장비 block (막기) 작업 노트

Grid: `grid/block/<dir>-<n>.txt` (8 directions × frames 0–3). Timing, pivot and horse shape are the base's.
The head, beard, helmet, robe, sash and armour come from idle frame 0 at the base head position. Each direction's
frame 0 has a 장팔사모 drawn by hand. Frames 1–3 move the rider and the 사모 together, by the base head offset
(integer moves). The hips and thighs follow the base frame and are recoloured: base `l` becomes robe `k`/`l`, and
base iron boots `b c` become `n o`.
Coordinates: x is the column after `NN|`, y is the row.

## By direction

| Direction | 사모 placement in frame 0 | Head/weapon offset f1 / f2 / f3 | Notes |
|---|---|---|---|
| S | Shaft at the chin: row 25 at x41–44 (tail `u` x41–43), row 24 at x47–55 (across the beard), then row 23. Ring x59 at rows 22–24, blade x60–66 lying flat to the right with up/down waves, tip outline (67,23). Hands (45–46,24–25) and (56–57,23–24) | (0,−2) / (0,−5) / (0,−1) | Beard shows rows 21–22 above the shaft, spikes on row 25 below it. Eyes not covered. Tassel `p q` (58,25–26) |
| SE | Gentle slope: tail row 26 x40–44 (raised 1 row after review 1 so both sides of the hand match), across the beard on row 25 x47–55, row 24 x58. Ring x59 at rows 23–25, blade x60–66 to the right | (−2,−1) / (−7,−3) / (−1,−1) | Beard rows 22–23 above, spikes on row 26 below. Tassel 1px `p` (58,26), above the horse's ear. In f2 the rider leans back, so robe was added to join the hips at x41–50 |
| SW | Mirror of SE (x′ = 102 − x). Blade x35–42 on the **left**, ring x43, tail x58–62 at row 26 (raised after review 1) on the right | (+2,−1) / (+7,−3) / (+1,−1) | Beard rows 22–23 above, spikes on row 26 below. The rein hand was removed. Tassel (44,26) |
| E | Shaft stands up in front of the face: x55 for rows 15–20, x56 for rows 21–28. Hand (55–56,27–28), tail `u` (56,29–30). Upright blade rows 7–13, ring row 14, tassel (57,15–16) | (−2,0) / (−9,0) / (−1,0) | All 5 beard rows show. The sleeve (x52–54, row 28) joins the hand. In f2 robe (x49–50, rows 27–30) was added to join the leaning torso |
| W | Mirror of E (x′ = 101 − x): shaft x46 for rows 15–20 and x45 for rows 21–28, hand (45–46,27–28), tail (45,29–30). Edges lit `a` on the left | (+2,0) / (+9,0) / (+1,0) | Beard shows. In f2 the area between the horse's neck and the rider was filled with horse `x` |
| N | Shaft runs crosswise behind the head. Left: ring x45 at rows 18–20, blade x37–44. Right: tail x57–60 on row 21 | (0,+1) / (0,+4) / (0,+1) | Hands are hidden. Tassel 1px (45,22) |
| NE | Diagonal at slope 1/2 (blade redrawn after review 1): one unbroken band x34–40, rows 9–15, 3–4px at the base narrowing to a single 1px tip (34,10), no outline inside; the top edge `a` steps out 1px and back along the diagonal stairs for the wave. Ring `ccc` 3px vertical at x41 rows 14–16 (perpendicular to the shaft), shaft (42,15)(43–44,16), hidden behind the head, out at the right shoulder (55–56,22), tail (57–58,23) | (−2,+1) / (−7,+3) / (−1,0) | Tassel `p q` (40,17–18) below the ring, outside. f2: sleeve `kkl` x48–50 rows 27–28 from shoulder toward the horse neck |
| NW | Horizontal on row 18: blade x38–45 above the horse's ears, ring x46, tail on the right x58–60 | (+2,+1) / (+7,+3) / (+1,0) | The base shaft remnants (`aaoo`, `AAAA`, `noo`) left on the horse's head were refilled with horse skin, mane or outline. f2: the gap between horse neck and rider is filled with sleeve `kkl` (x51–54, rows 26–27) under a 1-row outline |

## Tag cleanup
- Base shaft pixels tagged as horse (`w x A` remnants) and the base themeA shaft (`n o`) were removed in every frame.
  The 사모 uses only themeC `t u`, iron `a b c` and themeB `p q`.
- Base iron helmet and boots were replaced with themeA `m n o`.
- `render.py recolor --anims block`: horseSkin and horseMane cover only the horse, themeC only the shaft, and iron
  only the blade and ring.

## Deliberate
- In the back views (N, NE, NW) the middle of the shaft is hidden behind the head. The two visible pieces lie on one
  straight line, and the blade appears once.
- Directly under the shaft in the S, SE and SW frames, the beard spikes touch the shaft with no outline (row 25 or 26).
- In NE and SE, part of the blade or shaft overlaps the horse's ear. The weapon is drawn in front of the horse.

## Doubts
- **SW blade side**: profile §6 says "SW: blade on the left, as in the base". But in the base SW block the spear tip
  is actually on the right ((61,28)). I followed the profile and put the blade on the left (the mirror of SE), the same
  call as the approved Guan Yu block.
- **NE/NW blade side**: the profile only specifies N ("blade at the left end"). I followed the base, which puts the
  blade upper-left (NE) and left (NW) in both views. Guan Yu put the blade on the right for NE/NW, so this differs.
- The NE blade lies on a 1/2 slope, so its waves are weaker than in other directions. It reads as an S shape mainly
  from columns x39–41.
- The SE f2 leaning torso (rows 26–27, x41–48) and the E/W f2 joins (robe or horse fill) were drawn by hand.

## Review 1 fixes
- NE 0–3 (must): blade redrawn as one 2–4px band with no inner outline and a single tip; ring is one 3px `ccc` line; tassel below the ring, outside.
  The f0 block x31–44, rows 8–19 was moved unchanged to f1 (−2,+1), f2 (−7,+3), f3 (−1,0).
- SE/SW 0–3: tail piece raised 1 row (SE f0 row 26, f1 25, f2 23, f3 25; SW mirrored), so tail → hand → across-beard → hand → ring reads as one straight stair.
- NW 2: 3×3 hole filled with sleeve. NE 2: sleeve at x48–50 rows 27–28, outline at x51, horse neck/mane at x52–53.
- E 0–3: 1px holes between beard spikes and shaft filled with outline ((54,26) / (52,26) / (45,26) / (53,26)).
- SW 3: holes (44,26), (46–47,26) on row 26 filled with outline (each is enclosed by outline on all sides, so `#` rather than horse `x`).
- Review 2 (권장): NE 0–3 lower blade edge pushed out 1px in the middle, (35,12) `c` and (36,12) `b` with outline at (34,12) and (35,13) in f0, same cells at each frame offset, so both edges of the blade wave.
