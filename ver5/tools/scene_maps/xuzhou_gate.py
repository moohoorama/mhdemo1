#!/usr/bin/env python3
"""Make S_xuzhou_gate, the ground before the gate of 서주성 (chapter 13, 소패의 손님).

  python3 ver5/tools/scene_maps/xuzhou_gate.py

Format 2 (internal/scenemap): small tiles, the playing area from (0, 0) to (W-1, H-1) with a margin of M drawn round
it; u runs down to the right on screen, v down to the left. Ground: d packed earth, i paving, . grass.

Plan: the travellers come down the road from the north-west edge (-u) to the city. 서주성 is a provincial capital, so
its wall runs the whole south-east side (u = WALL) with the gate in the middle, and beyond the wall the paved streets,
the tiled roofs and the governor's great hall show over it. Before the gate the road widens into a trampled square
where a party can stand and talk with the gate guards; the refugees from the west camp along the wall north of the
gate (mats, bundles, a cart, a lean-to, a cooking fire) and a few more south of it. From the square a second road
leaves to the south (+v), a milestone at the fork.
"""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
W, H, M = 28, 24, 12
WALL = 27          # the wall line (u); its small tile 27 is game cell 13
GATE = 13          # the gate point on it (v): its passage, tiles 12-13, is game cell (13, 6)
ROAD = (11, 14)    # the road from the north-west to the gate, v from, to
SOUTH = (14, 16)   # the road to the south, u from, to

g = [['.'] * (W + 2 * M) for _ in range(H + 2 * M)]
obj = []


def paint(u0, v0, u1, v1, ch):
    for v in range(max(-M, v0), min(H + M - 1, v1) + 1):
        for u in range(max(-M, u0), min(W + M - 1, u1) + 1):
            g[v + M][u + M] = ch


def put(name, u, v):
    obj.append([name, u, v])


paint(-M, ROAD[0], WALL, ROAD[1], 'd')            # the road from the north-west
paint(SOUTH[0], ROAD[0], SOUTH[1], H + M, 'd')     # the road to the south
paint(12, 8, 26, 19, 'd')                          # the trampled square before the gate
paint(10, -4, 26, 7, 'd')                          # the refugees' ground along the wall
paint(15, 20, 26, 28, 'd')
paint(WALL + 1, -M, W + M, H + M, 'd')             # the city beyond the wall: its streets,
paint(WALL + 1, ROAD[0] + 1, W + M, ROAD[1] - 1, 'i')   # the paved way in from the gate
paint(WALL + 5, -M, WALL + 6, H + M, 'i')          # and the paved street along the wall

# fields in the open country north-west of the road
for u, v, name in ((-6, -6, 'field_ripe'), (-2, -6, 'field_ripe'), (-6, -2, 'field_ripe'), (-2, -2, 'field_bare'),
                   (2, -6, 'field_sprout'), (-8, 20, 'field_bare'), (-4, 20, 'field_sprout')):
    put(name, u, v)
# the square before the gate: banners on its far corners, a notice board for the newcomers
put('banner', 22, 8), put('banner', 22, 19), put('notice_board', 19, 20)
# the refugees north of the gate, back from the wall: mats and bundles round a cooking fire, carts with their goods,
# a sack shelter under a cloth awning
put('market_stall_cloth', 13, -1), put('hut', 16, -2), put('hut', 19, 0), put('straw_mat', 13, 2), put('straw_mat', 16, 1)
put('straw_mat', 19, 3), put('straw_mat', 15, 4), put('campfire', 17, 4), put('cauldron', 18, 5), put('grain_sack', 12, 0)
put('grain_sack', 20, 2), put('water_jar', 21, 1), put('water_bucket', 14, 6), put('grain_cart', 20, -3)
put('grain_cart_v', 11, 4), put('bloody_cloth', 16, 6), put('wine_jar', 22, 4)
# and a few south of it
put('hut', 20, 25), put('straw_mat', 17, 22), put('straw_mat', 19, 23), put('grain_sack', 22, 22), put('water_bucket', 17, 25)
put('campfire', 19, 21), put('grain_cart', 22, 27)
# the fork of the south road
put('milestone', 13, 17), put('stree_willow', 11, 19)
# willows and trees along the north-west road and in the fields
for u, v, name in ((6, 8, 'stree_willow'), (2, 17, 'stree_willow'), (9, 17, 'stree_bush_0'), (10, 7, 'stree_bush_1'),
                   (-9, 4, 'stree_broad_1'), (-9, 26, 'stree_pine_0'), (6, -8, 'stree_pine_1'), (12, -9, 'stree_broad_0'),
                   (3, 2, 'stree_old'), (8, 27, 'stree_broad_1'), (-3, 30, 'stree_pine_1'), (20, 33, 'stree_broad_0'),
                   (3, 24, 'stree_bush_0'), (8, -4, 'stree_bush_1'), (-10, 13, 'stree_broad_0')):
    put(name, u, v)
# the city over the wall: rows of tiled houses along its streets, the governor's hall up the way from the gate
for v in (-10, -5, 0, 5, 18, 23, 28, 33):
    put('house_tile_x', 30, v)
for v in (-9, -4, 1, 21, 26, 31):
    put('house_tile', 37, v)
put('hall_u', 37, 9), put('banner', 35, 10), put('banner', 35, 15), put('lantern', 30, 11), put('lantern', 30, 15)
put('stree_willow', 34, 2), put('stree_willow', 34, 22), put('well', 34, 27)
put('sgate_open', WALL, GATE)

art = {'id': 'S_xuzhou_gate', 'name': '서주성 앞', 'format': 2, 'size': [W // 2, H // 2], 'origin': [-M, -M],
       'ground': [''.join(r) for r in g], 'objects': obj,
       'walls': [['swall', [WALL, -M], [WALL, H + M - 1]]]}

if __name__ == '__main__':
    json.dump(art, open(GFX / 'maps/S_xuzhou_gate.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if 'maps/S_xuzhou_gate.json' not in index['maps']:
        index['maps'].append('maps/S_xuzhou_gate.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', 'S_xuzhou_gate'], cwd=GFX.parents[1], check=True)
    print('S_xuzhou_gate', art['size'], len(obj), 'objects')
