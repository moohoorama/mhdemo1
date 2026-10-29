#!/usr/bin/env python3
"""Make S_pingyuan, 평원현 where Liu Bei is made magistrate in chapter 8 and which he walks between the Hebei battles.

  python3 ver5/tools/scene_maps/pingyuan.py

Format 2 (internal/scenemap): small tiles, the playing area from (0, 0) to (W-1, H-1) with a margin of M drawn round
it; u runs down to the right on screen, v down to the left. Ground: d packed earth, i paving, . grass, ~ water.
A house's door is on its +v face (house_tile, tavern, hall_u) or +u face (house_tile_x).

Plan: a county seat on the flat Hebei plain. Unlike 탁현 (gate on the west, the hall straight up the main street)
the gate is in the north-east wall, the road north to 신도 and 계교 runs from it straight through the town, and a
canal with moored boats runs along the near-right side. Back left the 현청 behind a paved yard with the drill
ground beside it; the cross street carries the tavern (near left), the general store (by the road) and the smithy
(east of the road); the houses and the well lie south of the cross street, the boat landing and its stalls by the
canal.
"""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
W, H, M = 36, 28, 12
ROAD = (22, 26)       # the north road, u from, to
CROSS = (14, 17)      # the cross street, v from, to
CANAL = (38, 41)      # the canal, u from, to

g = [['.'] * (W + 2 * M) for _ in range(H + 2 * M)]
obj = []


def paint(u0, v0, u1, v1, ch):
    for v in range(max(-M, v0), min(H + M - 1, v1) + 1):
        for u in range(max(-M, u0), min(W + M - 1, u1) + 1):
            g[v + M][u + M] = ch


def put(name, u, v):
    obj.append([name, u, v])


# outside the walls (the margin behind them): fields of packed earth
paint(-M, -M, W + M, -2, 'd')
paint(-M, -M, -2, H + M, 'd')
# the road from the north gate, the cross street, the canal and its towpath
paint(ROAD[0], -M, ROAD[1], H + M, 'd')
paint(0, CROSS[0], CANAL[0] - 1, CROSS[1], 'd')
paint(CANAL[0], 0, CANAL[1], H + M, '~')
paint(CANAL[0] - 2, 0, CANAL[0] - 1, H + M, 'd')

# 현청: the hall behind a paved yard, banners, the paved way down to the cross street; the drill ground beside it
paint(5, 6, 17, 10, 'i')
paint(10, 11, 12, CROSS[0] - 1, 'i')
put('hall_u', 11, 4), put('house_tile_x', 6, 3), put('house_tile', 16, 2), put('banner', 7, 7), put('banner', 15, 7), put('lantern', 9, 10), put('lantern', 13, 10)
paint(0, 1, 3, 11, 'd')
put('weapon_rack', 1, 2), put('weapon_rack', 1, 5), put('armor_stand', 1, 8), put('command_flag', 3, 1)

# the cross street: the tavern (near left), the general store by the road, the smithy east of the road
put('tavern', 5, 12), put('wine_jar', 2, 13), put('bench', 8, 13), put('table_wine', 8, 14)
put('house_tile', 18, 11), put('market_stall_jar', 15, 13), put('grain_sack', 20, 13)
put('house_tile', 31, 11), put('weapon_rack', 29, 13), put('armor_stand', 33, 13), put('brazier', 34, 12)
put('notice_board', 21, 2), put('house_thatch', 30, 4), put('house_thatch_x', 34, 6), put('stree_broad_0', 18, 2)

# south of the cross street: houses turned to the road and the lanes, the well
put('house_tile_x', 19, 21), put('house_tile', 13, 20), put('house_thatch', 4, 20), put('house_thatch_x', 8, 25)
put('well', 15, 24), put('water_bucket', 16, 25), put('stree_willow', 11, 23), put('stree_bush_0', 1, 24), put('house_tile', 3, 28), put('stree_broad_1', 13, 30)
put('house_tile_x', 30, 21), put('straw_mat', 28, 25)

# the boat landing on the canal: stalls on the towpath, sacks waiting, boats moored, willows on the far bank
for v, name in ((19, 'market_stall'), (23, 'market_stall_cloth'), (27, 'market_stall_jar')):
    put(name, 35, v)
put('grain_sack', 36, 21), put('grain_cart_v', 34, 25), put('water_jar', 36, 26)
put('boat_v', 39, 21), put('boat_v', 40, 29)
for v in (2, 12, 33):
    put('stree_willow', 46, v)

# trees in the fields behind the walls
for u, v, name in ((-8, 4, 'stree_broad_1'), (-6, 18, 'stree_pine_0'), (-9, 30, 'stree_broad_0'),
                   (6, -8, 'stree_pine_1'), (16, -7, 'stree_broad_0'), (32, -8, 'stree_old'), (44, -6, 'stree_pine_0')):
    put(name, u, v)
put('sgate_open', 24, -1)

art = {'id': 'S_pingyuan', 'name': '평원현', 'format': 2, 'size': [W // 2, H // 2], 'origin': [-M, -M],
       'ground': [''.join(r) for r in g], 'objects': obj,
       'walls': [['swall', [-1, -1], [W + M - 1, -1]], ['swall', [-1, 0], [-1, H + M - 1]]]}

if __name__ == '__main__':
    json.dump(art, open(GFX / 'maps/S_pingyuan.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if 'maps/S_pingyuan.json' not in index['maps']:
        index['maps'].append('maps/S_pingyuan.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', 'S_pingyuan'], cwd=GFX.parents[1], check=True)
    print('S_pingyuan', art['size'], len(obj), 'objects')
