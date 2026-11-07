#!/usr/bin/env python3
"""Make S_xuzhou_town, the streets inside 서주성 (하비) that 유비 walks as the new 서주목 after 도겸's funeral (chapter
14, explore14): the provincial capital on the 사수, so bigger, richer and busier than 소패 and 평원.

  python3 ver5/tools/scene_maps/xuzhou_town.py

Format 2 (internal/scenemap): small tiles, the playing area from (0, 0) to (W-1, H-1) with a margin of M drawn round
it; u runs down to the right on screen, v down to the left. Ground: d packed earth, i paving, . grass, ~ water.
A house's door is on its +v face (house_tile, tavern, hall_u) or +u face (house_tile_x, hall_v).

Plan: the walls run along the two back sides, the gate in the northern (-v) wall, its road paved down to the main
avenue. The paved avenue runs across the town (along u) from the 관청 at its back end, a great hall behind a paved
court with banners, lanterns, an old pine and a garden rock, to the 사수 on the near-right side, where it ends at a
boat landing with moored boats and stalls on the towpath. North of the avenue, between the 관청 and the gate road,
the storehouse yard of the 미 family's trading house (granaries, sacks, carts); east of the gate road the tavern and
the general store face the avenue, the smithy further on. South of the avenue the paved market square with rows of
stalls and the well in the middle; south-west of it a garden with a pavilion, plum trees and willows; the houses of
the lanes fill the south and east. Willows and lanterns line the avenue.
"""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
ID = 'S_xuzhou_town'
W, H, M = 48, 40, 12
AVE = (16, 19)     # the paved avenue across the town, v from, to
GATE = 25          # the gate in the northern wall (v = -1); its road runs down u 24-26
ROAD = (24, 26)
RIVER = (44, 47)   # the 사수 along the near-right side, u from, to

g = [['.'] * (W + 2 * M) for _ in range(H + 2 * M)]
obj = []


def paint(u0, v0, u1, v1, ch):
    for v in range(max(-M, v0), min(H + M - 1, v1) + 1):
        for u in range(max(-M, u0), min(W + M - 1, u1) + 1):
            g[v + M][u + M] = ch


def put(name, u, v):
    obj.append([name, u, v])


# outside the walls: trodden fields; the north road through the gate; the river and its towpath
paint(-M, -M, W + M, -2, 'd')
paint(-M, -M, -2, H + M, 'd')
paint(ROAD[0], -M, ROAD[1], -2, 'd')
paint(ROAD[0], 0, ROAD[1], AVE[0] - 1, 'i')
paint(RIVER[0], -M, RIVER[1], H + M, '~')
paint(RIVER[0] - 2, 0, RIVER[0] - 1, H + M, 'd')
paint(0, AVE[0], RIVER[0] - 1, AVE[1], 'i')

# 관청: the great hall at the back end of the avenue behind a paved court, its door to the avenue
# (the main hall, and a wing hall either side of the court: the north one opens onto it, the south one onto the avenue)
paint(0, 8, 13, 27, 'i')
put('hall_v', 2, 17), put('hall_u', 7, 10), put('hall_v', 7, 26)
put('banner', 11, 12), put('banner', 11, 23), put('command_flag', 5, 14), put('command_flag', 5, 21)
put('lantern', 10, 15), put('lantern', 10, 20), put('brazier', 6, 15), put('brazier', 6, 20)
put('stree_pine_0', 1, 6), put('taihu_rock', 12, 8), put('stree_old', 1, 29)
put('house_tile_x', 2, 2), put('house_tile', 8, 4), put('stree_pine_1', 11, 1)

# the 미 family's storehouse yard north of the avenue, between the 관청 and the gate road
paint(13, 1, 22, 13, 'd')
put('granary_raised', 15, 3), put('granary_round', 19, 2), put('granary_raised', 15, 8), put('granary_round', 20, 7)
put('grain_sack', 17, 11), put('grain_sack', 18, 12), put('grain_sack', 21, 11), put('grain_cart', 13, 12)
put('grain_cart_v', 22, 4), put('banner', 13, 1), put('water_jar', 22, 9)

# east of the gate road: the general store and the tavern face the avenue, the smithy further on
put('house_tile', 29, 13), put('market_stall_jar', 27, 14), put('grain_sack', 31, 14)
put('tavern', 35, 13), put('wine_jar', 33, 14), put('table_wine', 38, 15), put('bench', 37, 15)
put('house_tile_x', 40, 6), put('weapon_rack', 42, 9), put('armor_stand', 42, 11), put('brazier', 41, 12)
put('house_tile', 29, 4), put('house_thatch', 34, 3), put('notice_board', 27, 1), put('stree_broad_0', 37, 1)
put('stree_willow', 23, 12), put('stree_willow', 41, 14)
for u in (18, 26, 34):
    put('lantern', u, AVE[0] - 1), put('lantern', u, AVE[1] + 1)

# the market square south of the avenue: two rows of stalls, the well in the middle, carts and mats
paint(14, AVE[1] + 1, 40, 30, 'i')
for i, u in enumerate((16, 20, 24, 32, 36)):
    put(('market_stall', 'market_stall_cloth', 'market_stall_jar')[i % 3], u, 22)
for i, u in enumerate((18, 22, 30, 34, 38)):
    put(('market_stall_cloth', 'market_stall_jar', 'market_stall')[i % 3], u, 28)
put('well', 27, 25), put('water_bucket', 28, 26), put('grain_cart_v', 40, 24), put('straw_mat', 15, 26)
put('grain_sack', 16, 27), put('water_jar', 39, 27)

# the boat landing where the avenue meets the river: stalls on the towpath, boats moored
put('boat_v', 45, 14), put('boat_v', 46, 22), put('boat_v', 45, 31)
put('market_stall_jar', 42, 21), put('grain_sack', 43, 24), put('grain_sack', 42, 14)
for v in (2, 24, 46):
    put('stree_willow', 49, v)

# the garden south-west of the market: a pavilion among plum trees and willows
paint(1, 30, 11, 38, '.')
put('pavilion', 6, 33), put('stree_plum_1', 2, 31), put('stree_plum_1', 10, 36), put('stree_willow', 1, 37)
put('taihu_rock', 10, 31)

# the lanes of houses, south and east
for u, v, name in [(15, 35, 'house_tile'), (21, 36, 'house_tile_x'), (27, 35, 'house_tile'), (33, 36, 'house_tile'),
                   (39, 35, 'house_tile_x'), (17, 40, 'house_thatch'), (25, 41, 'house_tile'), (35, 41, 'house_thatch_x')]:
    put(name, u, v)
put('stree_bush_0', 12, 33), put('stree_broad_1', 42, 39)
for u, v, name in [(13, 44, 'house_tile_x'), (30, 45, 'house_tile_x'), (20, 46, 'house_thatch'), (40, 44, 'house_tile'),
                   (6, 42, 'house_thatch_x'), (28, 50, 'house_thatch'), (36, 49, 'house_tile_x')]:
    put(name, u, v)
put('well', 23, 43), put('water_bucket', 24, 44), put('stree_broad_0', 10, 48), put('stree_willow', 33, 47)
# the far bank of the 사수: fields and trees
for u, v, name in ((52, 6, 'field_ripe'), (52, 18, 'field_sprout'), (54, 30, 'field_ripe'), (52, 42, 'field_bare'),
                   (56, 12, 'stree_broad_1'), (57, 36, 'stree_pine_0'), (55, 48, 'stree_broad_0')):
    put(name, u, v)

# trees in the fields behind the walls
for u, v, name in ((-8, 4, 'stree_broad_1'), (-6, 18, 'stree_pine_0'), (-9, 32, 'stree_broad_0'),
                   (6, -8, 'stree_pine_1'), (16, -7, 'stree_broad_0'), (34, -8, 'stree_old'), (40, -6, 'stree_pine_0')):
    put(name, u, v)
put('sgate_open', GATE, -1)

art = {'id': ID, 'name': '서주성', 'format': 2, 'size': [W // 2, H // 2], 'origin': [-M, -M],
       'ground': [''.join(r) for r in g], 'objects': obj,
       'walls': [['swall', [-1, -1], [RIVER[0] - 3, -1]], ['swall', [-1, 0], [-1, H + M - 1]]]}

if __name__ == '__main__':
    json.dump(art, open(GFX / f'maps/{ID}.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if f'maps/{ID}.json' not in index['maps']:
        index['maps'].append(f'maps/{ID}.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', ID], cwd=GFX.parents[1], check=True)
    print(ID, art['size'], len(obj), 'objects')
