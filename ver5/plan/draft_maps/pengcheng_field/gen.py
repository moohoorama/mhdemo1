"""The Pengcheng field drafts of 2026-10-28 (팽성 싸움 뒤 들판 시안): where Liu Bei meets Gongsun Zan (c05-reunion), at dusk.

  python3 ver5/plan/draft_maps/pengcheng_field/gen.py      -> ver5/plan/draft_maps/pengcheng_field/<draft>.json

The place is the north-east grassland of the B04 map: Pengcheng lies to the south (+v), the hills with the retreat
alley to the north. The recovered grain carts stand on the field. Uses the kit of ../danyang_village/gen.py."""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / 'danyang_village'))
import gen as kit  # noqa: E402  (writes the village drafts too; they are reproducible)

kit.OUT = Path(__file__).resolve().parent


def carts(c, cells):
    for i, (u, v) in enumerate(cells):
        c.put('grain_cart' if i % 2 == 0 else 'grain_cart_v', u, v)


# ---------- A: under the north wall of Pengcheng ----------
c = kit.Map('S_draft_pc_wall', '팽성 북쪽 들판(성벽)', 14, 10, seed=3)
# the wall seen from the field on the NW side, the open north gate at v 9, the paved town behind it
c.paint(-12, -12, 0, 31, 'i')
d_walls = [['swall', [1, -12], [1, 31]]]
c.put('sgate_open', 1, 9)
c.paint(2, 8, 27, 11)
c.paint(6, 4, 22, 15)
carts(c, [(14, 7), (17, 9), (14, 11), (19, 6)])
c.put('grain_sack', 16, 12); c.put('grain_sack', 20, 8)
for u, v in [(6, 6), (10, 5), (21, 12), (5, 11), (24, 9), (12, 13)]:
    c.put('broken_spear', u, v)
c.put('fallen_flag', 7, 9); c.put('fallen_flag', 22, 4)
c.put('banner', 3, 6); c.put('banner', 3, 12)
c.put('campfire', 6, 13)
c.trees(5, (8, -8, 36, 1), gap=4)
c.trees(3, (24, 20, 36, 30), gap=4)
c.save()
import json
p = kit.OUT / 'pc_wall.json'; d = json.load(open(p)); d['walls'] = d_walls; json.dump(d, open(p, 'w'), ensure_ascii=False, indent=1)

# ---------- B: open grass under the northern hills ----------
c = kit.Map('S_draft_pc_hills', '팽성 북쪽 들판(언덕)', 14, 10, seed=9)
c.cliff(-12, -12, 39, -2)
c.cliff(-12, -12, 4, -1)
for u in range(-10, 38, 3):
    c.put(c.rng.choice(['stree_pine_0', 'stree_pine_1', 'stree_broad_1']), u, -3 - c.rng.randint(0, 1))
# hoof-churned track down from the hills towards the city (south, +v)
c.paint(9, -1, 12, 31)
c.paint(4, 6, 23, 14)
carts(c, [(15, 8), (18, 10), (15, 12)])
c.put('grain_sack', 17, 13); c.put('grain_sack', 19, 8)
for u, v in [(6, 7), (11, 4), (21, 13), (7, 12), (24, 10), (13, 15)]:
    c.put('broken_spear', u, v)
c.put('fallen_flag', 8, 9); c.put('fallen_flag', 21, 6)
c.put('campfire', 6, 15); c.put('straw_mat', 4, 16)
c.put('milestone', 13, 20)
c.trees(4, (24, 22, 36, 30), gap=4)
c.trees(4, (-10, 10, 3, 30), gap=4)
c.save()

# ---------- C (chosen): open field with a river winding along the north-east ----------
import math  # noqa: E402

c = kit.Map('S_draft_pc_field', '팽성 북쪽 들판', 14, 10, seed=21)
M = kit.M
river = {}
for u in range(-M, c.W + M):
    vc = round(0.12 * u - 3 + 3.2 * math.sin(u / 5.0))
    river[u] = vc
    c.paint(u, vc - 1, u, vc + 2, '~')
    if u % 5 == 2:
        c.paint(u, vc + 3, u, vc + 3, '~')  # the bank bulges at the bends
bank = lambda u: river[u] + 4


def blob(cu, cv, r):
    for v in range(cv - r, cv + r + 1):
        for u in range(cu - r - 1, cu + r + 2):
            if ((u - cu) / (r + 1)) ** 2 + ((v - cv) / r) ** 2 <= 1 + c.rng.uniform(-.25, .1):
                c.paint(u, v, u, v, 'd')


# trampled ground where the riders turned, not one rectangle
for cu, cv, r in [(15, 11, 3), (9, 13, 2), (20, 14, 2), (11, 8, 2), (22, 9, 1), (6, 16, 1)]:
    blob(cu, cv, r)
# hoof track from the south-west edge to the carts
for i in range(0, 14):
    u, v = 4 + i // 2, 31 - i
    c.paint(u, v, u + 1, v, 'd')
# the recovered grain carts parked together, sacks unloaded beside them
c.put('grain_cart', 14, 10); c.put('grain_cart_v', 17, 11); c.put('grain_cart', 15, 13); c.put('grain_cart_v', 12, 11)
for u, v in [(17, 14), (18, 13), (13, 13)]:
    c.put('grain_sack', u, v)
c.put('campfire', 8, 14); c.put('straw_mat', 7, 16)
for u, v in [(6, 9), (11, 5), (21, 15), (9, 11), (24, 11), (19, 8), (4, 14)]:
    c.put('broken_spear', u, v)
c.put('fallen_flag', 10, 7); c.put('fallen_flag', 22, 12); c.put('fallen_flag', 5, 12)
# willows and reeds on the south bank, bushes round the field's edge
for u in (3, 11, 19, 26, 33):
    c.put('stree_willow', u, bank(u))
for u in (7, 15, 23, 30):
    c.put('stree_bush_1', u, bank(u))
for u, v in [(2, 18), (26, 6), (1, 7), (25, 19)]:
    c.put(c.rng.choice(['stree_bush_0', 'stree_bush_1']), u, v)
c.trees(3, (-10, 6, -2, 30), gap=4)
c.trees(3, (8, 22, 36, 30), gap=4)
c.save()
