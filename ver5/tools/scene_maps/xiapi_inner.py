#!/usr/bin/env python3
"""Make S_xiapi_inner, the inside of 하비성 the day 여포 falls (chapter 16, after the battle): the flood has gone down
— the fighting was done on foot — but the low quarter where 단양 people lived still shows it: damp ground, puddles,
ruts full of water in its lane, the drift line of straw and twigs where the water stood, and the houses' walls stained
to the flood's mark. The party comes in by the street from the south gate and looks round.

  python3 ver5/tools/scene_maps/xiapi_inner.py

Format 2 (internal/scenemap): small tiles, the playing area from (0, 0) to (W-1, H-1) with a margin of M drawn round
it; u runs down to the right on screen, v down to the left. Ground: d soil, i paving. The wet ground is flat
pictures (assets_candidate/tools/gen_pixel_wet.py): wet_patch_* anchored at their back corner, puddle_*, rut_v and
wrack_* at their front corner.

Plan: the 관아 at the back on a paved yard (high ground), a street from the front (the way in from the south gate,
near right) up to it; on the left the low quarter — houses, a willow, a lane, all wet; on the right the granary and
grain spread on mats to dry, carts.
"""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
ID = 'S_xiapi_inner'
W, H, M = 32, 28, 10

g = [['d'] * (W + 2 * M) for _ in range(H + 2 * M)]
obj = []


def paint(u0, v0, u1, v1, ch):
    for v in range(max(-M, v0), min(H + M - 1, v1) + 1):
        for u in range(max(-M, u0), min(W + M - 1, u1) + 1):
            g[v + M][u + M] = ch


def put(name, u, v):
    obj.append([name, u, v])


# the low quarter, left (large v is the left of the screen): its yards and lane, wet where the flood stood
paint(-M, 14, 12, H + M, 'd')     # silt over what was grass; the lane shows by its ruts
for u, v, name in [(-2, 14, 'wet_patch_3'), (6, 15, 'wet_patch_0'), (0, 20, 'wet_patch_1'), (7, 21, 'wet_patch_1'),
                   (-2, 27, 'wet_patch_0'), (7, 27, 'wet_patch_3'), (2, 33, 'wet_patch_2'), (10, 25, 'wet_patch_2'),
                   (-6, 20, 'wet_patch_3'), (-5, 31, 'wet_patch_1'), (9, 32, 'wet_patch_0'), (3, 17, 'wet_patch_2')]:
    put(name, u, v)
for u, v, name in [(6, 20, 'rut_v'), (6, 27, 'rut_v'), (5, 32, 'puddle_5'), (3, 21, 'puddle_1'), (8, 22, 'puddle_2'),
                   (-1, 19, 'puddle_0'), (11, 20, 'puddle_4'), (8, 16, 'puddle_3'), (12, 29, 'puddle_1'),
                   (0, 28, 'puddle_0'), (3, 25, 'puddle_2'), (10, 34, 'puddle_3')]:
    put(name, u, v)
for u, v, name in [(13, 18, 'wrack_v'), (13, 24, 'wrack_v'), (13, 30, 'wrack_v'), (-3, 14, 'wrack_u'),
                   (3, 14, 'wrack_u'), (9, 14, 'wrack_u')]:     # where the water stood highest
    put(name, u, v)
for u, v, name in [(1, 17, 'house_thatch_mud'), (9, 17, 'house_thatch_x_mud'), (1, 23, 'house_thatch_x_mud'),
                   (9, 24, 'house_thatch_mud'), (2, 30, 'house_tile_mud'), (10, 31, 'house_tile_x_mud')]:
    put(name, u, v)
put('stree_willow', 12, 21), put('stree_broad_1', -2, 26)

# the 관아 on its paved yard at the back
paint(10, 0, 24, 8, 'i')
put('hall_u', 17, 4)
put('banner', 13, 7), put('banner', 21, 7)

# the dry street from the south gate (front) up to the yard
paint(15, 9, 19, H + M, 'i')
for u, v, name in [(17, 15, 'puddle_0'), (16, 22, 'puddle_3'), (18, 27, 'puddle_5'), (14, 7, 'puddle_3')]:
    put(name, u, v)

# the granary side, right: sacks, grain spread on mats to dry, carts
put('house_tile_x', 26, 6), put('house_tile', 28, 13)
put('straw_mat', 23, 14), put('straw_mat', 23, 17), put('grain_sack', 25, 15), put('grain_sack', 26, 16),
put('grain_cart', 25, 21), put('puddle_3', 24, 19), put('wet_patch_2', 26, 17), put('water_jar', 22, 11), put('stree_broad_0', 30, 22)

art = {'id': ID, 'name': '하비 성안', 'format': 2, 'size': [W // 2, H // 2], 'origin': [-M, -M],
       'ground': [''.join(r) for r in g], 'objects': obj}

if __name__ == '__main__':
    json.dump(art, open(GFX / f'maps/{ID}.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if f'maps/{ID}.json' not in index['maps']:
        index['maps'].append(f'maps/{ID}.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', ID], cwd=GFX.parents[1], check=True)
    print(ID, art['size'], len(obj), 'objects')
