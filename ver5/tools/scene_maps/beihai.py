#!/usr/bin/env python3
"""Make S_beihai, the inside of 북해성 where 공융 shelters the refugees from 서주 (chapter 12's explore, explore12).

  python3 ver5/tools/scene_maps/beihai.py

It is laid out like 신도성 (tools/scene_maps/xindu.py: walls on the north-west and south-west sides, the gate in the
near-left wall, the main street from the gate up to the 관아, the cross street) but it is the bigger city: a wide
paved 관아 yard with a pavilion, three market stalls, the tavern, and the refugees who keep coming from 서주. Outside
the walls is green field, the road leaving the gate south-west; inside, besides the refugees' mats by the north-west
wall where the explore's people stand, the refugees camp along the roads on the open north and east sides of the city:
mats, huts and an awning, carts and sacks, a cooking fire and pot, water jars, each camp on its patch of trodden earth.

Format 2 (internal/scenemap): small tiles, the playing area from (0, 0) to (W-1, H-1) with a margin of M drawn round
it; u runs down to the right on screen, v down to the left. Ground: d packed earth, i paving, . grass.
"""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
W, H, M = 32, 28, 12
MAIN = (13, 16)    # the main street from the gate to the 관아 and on north, u from, to
CROSS = (14, 16)   # the cross street, v from, to

g = [['.'] * (W + 2 * M) for _ in range(H + 2 * M)]


def paint(u0, v0, u1, v1, ch):
    for v in range(max(-M, v0), min(H + M - 1, v1) + 1):
        for u in range(max(-M, u0), min(W + M - 1, u1) + 1):
            g[v + M][u + M] = ch


paint(MAIN[0], -M, MAIN[1], H + M, 'd')   # the main street, on out of the gate across the fields
paint(0, CROSS[0], W + M, CROSS[1], 'd')
paint(9, 3, 29, 8, 'i')                   # the 관아 yard
paint(21, 9, 29, 9, 'i')

# the town as it stood: the 관아, yard, stalls, houses, the refugees' mats by the north-west wall, trees outside
obj = [
    ["hall_u", 14, 2], ["banner", 10, 4], ["banner", 19, 4], ["lantern", 12, 8], ["lantern", 17, 8], ["notice_board", 18, 10],
    ["pavilion", 25, 5], ["taihu_rock", 22, 7], ["stree_willow", 28, 2], ["desk", 24, 8], ["incense", 27, 8], ["hut", 2, 4],
    ["straw_mat", 1, 8], ["straw_mat", 3, 9], ["straw_mat", 1, 11], ["straw_mat", 4, 12], ["straw_mat", 2, 20], ["straw_mat", 5, 22],
    ["soup_stove", 6, 9], ["cauldron", 7, 11], ["water_jar", 5, 7], ["grain_sack", 8, 8], ["tavern", 5, 17], ["wine_jar", 2, 18],
    ["bench", 9, 18], ["house_tile", 22, 12], ["market_stall_jar", 19, 13], ["market_stall_cloth", 25, 13], ["house_tile_x", 28, 18], ["weapon_rack", 26, 17],
    ["armor_stand", 30, 16], ["market_stall", 10, 12], ["grain_cart_v", 11, 18], ["house_tile_x", 7, 23], ["house_thatch", 22, 22], ["house_thatch_x", 27, 23],
    ["well", 19, 21], ["water_bucket", 20, 22], ["stree_broad_0", 4, 26], ["grain_sack", 11, 26], ["grain_sack", 18, 26], ["broken_spear", 17, 24],
    ["stree_pine_0", 6, -7], ["stree_broad_1", 20, -9], ["stree_broad_0", 37, 3], ["stree_pine_1", 39, 17], ["stree_pine_1", -7, 6], ["stree_broad_0", -6, 22],
    ["stree_old", 26, 35], ["sgate_open", 14, 28]]

# the refugees' camps by the roads on the north and east sides. A big camp: an awning, two huts, mats round a
# cooking fire and its pot, sacks, a water jar, a cart; a small one: a hut, mats, a fire and pot, a sack, a jar, a cart
BIG = [('market_stall_cloth', 1, 0), ('hut', 3, 0), ('hut', 5, 1), ('grain_sack', 0, 2), ('straw_mat', 2, 2),
       ('straw_mat', 4, 3), ('straw_mat', 1, 4), ('campfire', 3, 5), ('cauldron', 4, 6), ('grain_sack', 6, 3),
       ('water_jar', 7, 2), ('straw_mat', 6, 5), ('grain_cart_v', 0, 6)]
SMALL = [('hut', 1, 0), ('hut', 3, 0), ('grain_sack', 4, 1), ('straw_mat', 0, 2), ('straw_mat', 2, 2),
         ('campfire', 2, 4), ('water_jar', 4, 3), ('grain_cart', 6, 0)]
CAMPS = [(BIG, 4, -10),     # north, west of the main street
         (SMALL, 19, -8),   # north, east of it
         (BIG, 32, 6),      # east, north of the cross street
         (SMALL, 32, 20)]   # east, south of it
GONE = {('stree_pine_0', 6, -7), ('stree_broad_1', 20, -9)}   # trees where the northern camps stand
obj = [o for o in obj if tuple(o) not in GONE]
for kit, a, b in CAMPS:
    placed = [(a + du, b + dv) for _, du, dv in kit]
    for v in range(b - 3, b + 10):            # the trodden earth round the camp's things
        for u in range(a - 3, a + 11):
            if any((u - pu) ** 2 + (v - pv) ** 2 <= 2.3 for pu, pv in placed) and g[v + M][u + M] == '.':
                g[v + M][u + M] = 'd'
    obj += [[n, a + du, b + dv] for n, du, dv in kit]

art = {'id': 'S_beihai', 'name': '북해성 안', 'format': 2, 'size': [W // 2, H // 2], 'origin': [-M, -M],
       'grass': {'hue': -16, 'sat': 0.8, 'val': 1.04},
       'ground': [''.join(r) for r in g], 'objects': obj,
       'walls': [['swall', [-1, -M], [-1, H]], ['swall', [0, H], [W + M - 1, H]]]}

if __name__ == '__main__':
    json.dump(art, open(GFX / 'maps/S_beihai.json', 'w'), ensure_ascii=False, indent=1)
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', 'S_beihai'], cwd=GFX.parents[1], check=True)
    print('S_beihai', art['size'], len(obj), 'objects')
