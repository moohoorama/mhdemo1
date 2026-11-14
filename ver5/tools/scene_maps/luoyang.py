#!/usr/bin/env python3
"""Make S_luoyang, the street of 낙양 the 3rd chapter walks (assets/scenario/hero/explore03.json): the capital, so
unlike the county towns (S_zhuo, S_castle_town) it is paved, wide and crowded, under the palace wall.

  python3 ver5/tools/scene_maps/luoyang.py

Writes format 2 (internal/scenemap) straight: small tiles from origin 0, 0; u runs down to the right on screen,
v down to the left. Ground: d packed earth, i paving, . grass. A house's door is on its +v face (house_tile,
tavern) or +u face (house_tile_x); a hall_u's on its +v face.

Plan: the palace wall along the back (v = 4), vermilion with the great gate hall (palace_gate) on the avenue — the
emperor still sits here, so 낙양 and 허도 (S_xudu) alone get the palace wall; 업 (S_ye), a governor's seat, a stone one; a paved square before it; the
paved avenue (u 25..31) down the middle lined with willows and lanterns; the cross street (v 24..27). Back left
조조's house, a hall behind a paved yard with a pine and a garden rock; back right the 관아 by the notice board;
near left the houses of the lanes; near right the market. The tavern and the general store face the cross street
either side of the avenue.
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


# behind the palace wall: the paved court and the roofs of the palace
paint(0, 0, W - 1, 3, 'i')
put('hall_u', 18, 1), put('pavilion', 38, 1), put('hall_u', 48, 1), put('hall_u', 8, 1)
for u in range(W):
    d = u - GATE_U
    if abs(d) > 5:
        put({0: 'pwall_u_end_m', W - 1: 'pwall_u_end_p'}.get(u, 'pwall_u'), u, WALL_V)
    elif abs(d) == 5:
        put('pwall_u_gate_m' if d < 0 else 'pwall_u_gate_p', u, WALL_V)
put('palace_gate', GATE_U, WALL_V)
# the square before the gate, the avenue, the cross street, the lanes
paint(16, 5, 40, 10, 'i')
paint(25, 5, 31, H - 1, 'i')
paint(0, 24, W - 1, 27, 'd')
for u in (17, 39):
    put('banner', u, 6), put('lantern', u, 9)
put('banner', 23, 5), put('banner', 33, 5)
for v in (12, 18, 31, 37, 44):
    put('stree_willow', 23, v), put('stree_willow', 33, v)
for v in (15, 21, 34, 41):
    put('lantern', 24, v), put('lantern', 32, v)

# 조조의 저택: a hall behind a paved yard, a pine, a garden rock, lanterns at the yard's mouth
paint(4, 11, 20, 13, '.')
paint(8, 15, 19, 19, 'i')
paint(9, 20, 24, 21, 'i')
put('hall_u', 14, 14)
put('stree_pine_0', 6, 13), put('taihu_rock', 9, 16), put('stree_plum_1', 19, 12)
put('lantern', 11, 19), put('lantern', 17, 19)
put('house_tile', 5, 18), put('house_tile', 3, 22)

# the tavern on the cross street, west of the avenue: tables out front
put('tavern', 21, 22)
put('wine_jar', 19, 21), put('table_wine', 18, 23), put('bench', 17, 23)

# the general store on the cross street, east of the avenue
put('house_tile', 37, 22)
put('market_stall_jar', 40, 23), put('grain_sack', 35, 23), put('water_jar', 39, 21)

# the 관아 and its notice board by the avenue
paint(34, 11, 47, 18, 'i')
put('hall_v', 44, 14), put('banner', 41, 11), put('banner', 41, 17), put('notice_board', 35, 14)
put('house_tile', 52, 10), put('house_tile_x', 50, 20), put('stree_broad_0', 54, 15)

# the lanes of houses, near left
paint(0, 39, 24, 40, 'd')
for u, v, name in [(4, 31, 'house_tile'), (9, 31, 'house_tile'), (15, 31, 'house_tile_x'), (21, 31, 'house_tile'),
                   (4, 37, 'house_tile_x'), (10, 37, 'house_tile'), (17, 37, 'house_tile'),
                   (3, 44, 'house_tile'), (9, 45, 'house_tile_x'), (16, 44, 'house_tile'), (21, 45, 'house_tile')]:
    put(name, u, v)
put('well', 13, 34), put('water_bucket', 14, 35), put('stree_broad_1', 1, 28)

# the market, near right: stalls either side of the market lane, carts and mats
paint(33, 33, W - 1, 35, 'i')
for i, u in enumerate((36, 40, 44, 48, 52)):
    put(('market_stall', 'market_stall_cloth', 'market_stall_jar')[i % 3], u, 31)
for i, u in enumerate((38, 42, 46, 50)):
    put(('market_stall_cloth', 'market_stall_jar', 'market_stall')[i % 3], u, 38)
put('grain_cart', 54, 29), put('grain_cart_v', 35, 41), put('straw_mat', 44, 41), put('grain_sack', 45, 42),
put('wine_jar', 53, 37), put('water_jar', 34, 30)
for u, v, name in [(40, 45, 'house_tile'), (47, 46, 'house_tile'), (53, 44, 'house_tile_x')]:
    put(name, u, v)

art = {'id': 'S_luoyang', 'name': '낙양 저잣거리', 'format': 2, 'size': [W // 2, H // 2], 'origin': [0, 0],
       'grass': {'hue': -4, 'sat': 0.95}, 'ground': [''.join(r) for r in g], 'objects': obj, 'walls': []}

if __name__ == '__main__':
    json.dump(art, open(GFX / 'maps/S_luoyang.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if 'maps/S_luoyang.json' not in index['maps']:
        index['maps'].append('maps/S_luoyang.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', 'S_luoyang'], cwd=GFX.parents[1], check=True)
    print('S_luoyang', art['size'], len(obj), 'objects')
