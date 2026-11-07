#!/usr/bin/env python3
"""Make S_xiaopei, the county town of 소패 that 유비 holds for 도겸 (chapter 13, 소패의 손님), walked in explore mode.

  python3 ver5/tools/scene_maps/xiaopei.py

Format 2 (internal/scenemap): small tiles, the playing area from (0, 0) to (W-1, H-1) with a margin of M drawn round
it; u runs down to the right on screen, v down to the left. Ground: d packed earth, i paving, . grass.
A house's door is on its +v face (house_tile, tavern, hall_u) or +u face (house_tile_x, hall_v).

Plan: a small walled county town, smaller than 신도. The walls run along the two back sides; the gate is in the
northern (-v) wall near the north corner and the road north leaves through it, a lane running from it down to the
main street. The main street runs across the town (along u) from the 관아 at its back end, a hall behind a paved yard
against the north-west wall, to the open south-east side. North of the street the 약방 with its jar stall and herb
sacks; east of the north lane the granary yard with its storehouses, sacks and a guard's weapon rack; south of the
street the market corner where refugee merchants have set up stalls and a cart, the well, and the houses. Everything
someone stands by faces a street, so people stand in front of the buildings.
"""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
W, H, M = 28, 24, 12
MAIN = (10, 12)    # the main street across the town, v from, to
LANE = (18, 20)    # the lane from the north gate, u from, to
GATE = 19          # the gate in the northern wall (v = -1): its passage, tiles 18-19, is game cell (9, -1)

g = [['.'] * (W + 2 * M) for _ in range(H + 2 * M)]
obj = []


def paint(u0, v0, u1, v1, ch):
    for v in range(max(-M, v0), min(H + M - 1, v1) + 1):
        for u in range(max(-M, u0), min(W + M - 1, u1) + 1):
            g[v + M][u + M] = ch


def put(name, u, v):
    obj.append([name, u, v])


paint(LANE[0], -M, LANE[1], H + M, 'd')     # the north road through the gate, the lane and on south
paint(4, MAIN[0], W + M, MAIN[1], 'd')      # the main street
paint(3, 6, 8, 16, 'i')                     # the 관아 yard
paint(21, 0, 27, 8, 'd')                    # the granary yard
paint(9, 6, 16, 9, 'd')                     # before the 약방
paint(20, 13, 28, 20, 'd')                  # the market corner
paint(11, 13, 15, 18, 'd')                  # round the well

# 관아: the hall at the back end of the main street, its door to the street, banners and lanterns in its yard
put('hall_v', 2, 11), put('banner', 5, 7), put('banner', 5, 15), put('lantern', 7, 8), put('lantern', 7, 14)
put('stree_old', 1, 3), put('house_tile_x', 2, 19)
# 약방: the herb shop north of the street, its jar stall, herb sacks and the medicine pot out front
put('house_tile', 12, 6), put('market_stall_jar', 9, 7), put('medicine_pot', 14, 8), put('grain_sack', 15, 7)
put('water_jar', 10, 9)
put('house_thatch', 15, 2), put('house_thatch_x', 7, 2)
# the granary yard east of the north lane: two storehouses, sacks and a cart, the guard's weapon rack
put('granary_raised', 23, 2), put('granary_round', 26, 3), put('grain_sack', 22, 5), put('grain_sack', 23, 6)
put('grain_cart', 26, 7), put('weapon_rack', 21, 1), put('banner', 21, 7)
# the market corner south of the street: refugee merchants' stalls, their cart and bundles
put('market_stall', 21, 14), put('market_stall_cloth', 24, 14), put('market_stall_jar', 27, 14)
put('grain_cart_v', 26, 18), put('hut', 23, 19), put('straw_mat', 21, 18), put('grain_sack', 28, 17)
# the well and the houses of the south of town
put('well', 13, 15), put('water_bucket', 14, 16), put('stree_willow', 10, 14)
put('house_thatch', 6, 20), put('house_tile', 12, 21), put('house_thatch_x', 17, 22), put('house_tile_x', 24, 23)
put('house_tile', 31, 9), put('house_thatch', 32, 14), put('stree_bush_0', 29, 12)
put('notice_post', 17, 9), put('stree_bush_1', 16, 18)
# trees and fields outside the walls
for u, v, name in ((-7, 4, 'stree_broad_1'), (-6, 18, 'stree_pine_0'), (-8, 28, 'stree_broad_0'), (6, -8, 'stree_pine_1'),
                   (13, -6, 'stree_broad_0'), (28, -8, 'stree_old'), (36, -5, 'stree_pine_0'), (-5, -6, 'stree_broad_1')):
    put(name, u, v)
for u, v, name in ((-8, 8, 'field_ripe'), (-8, 12, 'field_bare'), (24, -8, 'field_sprout'), (30, -4, 'field_ripe')):
    put(name, u, v)
put('sgate_open', GATE, -1)

art = {'id': 'S_xiaopei', 'name': '소패', 'format': 2, 'size': [W // 2, H // 2], 'origin': [-M, -M],
       'ground': [''.join(r) for r in g], 'objects': obj,
       'walls': [['swall', [-1, -1], [W + M - 1, -1]], ['swall', [-1, 0], [-1, H + M - 1]]]}

if __name__ == '__main__':
    json.dump(art, open(GFX / 'maps/S_xiaopei.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if 'maps/S_xiaopei.json' not in index['maps']:
        index['maps'].append('maps/S_xiaopei.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', 'S_xiaopei'], cwd=GFX.parents[1], check=True)
    print('S_xiaopei', art['size'], len(obj), 'objects')
