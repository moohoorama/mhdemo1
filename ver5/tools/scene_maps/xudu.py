#!/usr/bin/env python3
"""Make S_xudu, the 허도 the 17th chapter walks (assets/scenario/hero/explore17.json): the new capital under 조조,
with two great houses — 조조's and the one given to 유비 — and the palace front.

  python3 ver5/tools/scene_maps/xudu.py

Writes format 2 (internal/scenemap) straight: small tiles from origin 0, 0; u runs down to the right on screen,
v down to the left. Ground: d packed earth, i paving, . grass. A hall_u's door is on its +v face, a hall_v's on its
-u face (the avenue side), a house_tile's on its +v face.

Plan: the palace wall along the back (v = 4) in vermilion plaster (pwall_u, gen_pixel_palace.py) with the great gate
hall (palace_gate) on the avenue; a paved square before it with guards' banners. The avenue (u 25..31) runs down the
middle, the cross street along v 24..27.
  back left   조조's house (mansion_cao, gen_pixel_palace.py): a white-walled compound with a tall gate, a two-storey
              hall and two wings, before it a paved forecourt with banners and lanterns.
  back right  the 관청 by the notice board.
  near left   유비's house (mansion_liu): an ochre-walled compound with a plain gate and one hall, in an earth yard
              with a willow and a well — the house 조조 gave the 황숙, modest beside the other.
  near right  the market and the general store on the cross street.
"""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
W, H = 56, 48
WALL_V = 4
GATE_U = 28

g = [['d'] * W for _ in range(H)]
obj = []


def paint(u0, v0, u1, v1, ch):
    for v in range(max(0, v0), min(H, v1 + 1)):
        for u in range(max(0, u0), min(W, u1 + 1)):
            g[v][u] = ch


def put(name, u, v):
    obj.append([name, u, v])


# behind the palace wall: paved courts and the roofs of the palace halls
paint(0, 0, W - 1, WALL_V + 2, 'i')
put('hall_u', 10, 1), put('hall_u', 46, 1)
# the wall and the gate hall
for u in range(W):
    if abs(u - GATE_U) > 4:
        put('pwall_u', u, WALL_V)
put('palace_gate', GATE_U, WALL_V)

# the square before the gate, the avenue, the cross street
paint(16, 7, 40, 11, 'i')
paint(25, 7, 31, H - 1, 'i')
paint(0, 24, W - 1, 27, 'd')
put('banner', 22, 8), put('banner', 34, 8)
put('lantern', 23, 10), put('lantern', 33, 10)
for v in (14, 20, 31, 37, 44):
    put('stree_willow', 23, v), put('stree_willow', 33, v)
for v in (17, 34, 41):
    put('lantern', 24, v), put('lantern', 32, v)

# 조조의 저택 (mansion_cao, 13 x 9, gate on its +v side at u 11): a paved forecourt with banners, lanterns, a pine
paint(3, 9, 22, 23, 'i')
put('mansion_cao', 11, 15)
put('stree_pine_0', 20, 11), put('taihu_rock', 20, 17), put('stree_plum_1', 3, 21)
put('banner', 8, 21), put('banner', 14, 21), put('lantern', 9, 22), put('lantern', 13, 22)

# 관청 and its notice board, back right
paint(34, 13, 47, 21, 'i')
put('hall_v', 44, 16), put('banner', 41, 13), put('banner', 41, 19), put('notice_board', 36, 15)
put('house_tile', 52, 12), put('stree_broad_0', 54, 18)

# 유비의 저택 (mansion_liu, 9 x 7, gate on its +v side at u 11): a plain earth yard, a willow, a well, a plot
paint(3, 30, 20, 43, 'd')
put('mansion_liu', 11, 36)
put('stree_willow', 4, 32), put('well', 18, 35), put('water_bucket', 19, 36)
put('straw_mat', 5, 41), put('grain_sack', 6, 42), put('stree_broad_1', 19, 41)
put('lantern', 9, 41), put('lantern', 13, 41)
for u, v, name in [(3, 46, 'house_tile'), (9, 46, 'house_tile_x'), (16, 46, 'house_tile'), (21, 33, 'house_tile_x')]:
    put(name, u, v)

# the general store on the cross street
put('house_tile', 37, 22)
put('market_stall_jar', 40, 23), put('grain_sack', 35, 23), put('water_jar', 39, 21)

# the market, near right
paint(33, 33, W - 1, 35, 'i')
for i, u in enumerate((36, 40, 44, 48, 52)):
    put(('market_stall', 'market_stall_cloth', 'market_stall_jar')[i % 3], u, 31)
for i, u in enumerate((38, 42, 46, 50)):
    put(('market_stall_cloth', 'market_stall_jar', 'market_stall')[i % 3], u, 38)
put('grain_cart', 54, 29), put('grain_cart_v', 35, 41), put('wine_jar', 53, 37), put('water_jar', 34, 30)
for u, v, name in [(40, 45, 'house_tile'), (47, 46, 'house_tile'), (53, 44, 'house_tile_x')]:
    put(name, u, v)

art = {'id': 'S_xudu', 'name': '허도 거리', 'format': 2, 'size': [W // 2, H // 2], 'origin': [0, 0],
       'ground': [''.join(r) for r in g], 'objects': obj}

if __name__ == '__main__':
    json.dump(art, open(GFX / 'maps/S_xudu.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if 'maps/S_xudu.json' not in index['maps']:
        index['maps'].append('maps/S_xudu.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', 'S_xudu'], cwd=GFX.parents[1], check=True)
    print('S_xudu', art['size'], len(obj), 'objects')
