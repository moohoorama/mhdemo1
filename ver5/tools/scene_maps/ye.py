#!/usr/bin/env python3
"""Make S_ye, the 업 the 19th chapter walks (assets/scenario/hero/explore19_ye.json): 원소's seat, the greatest city
north of the river, on the eve of 백마 — larger than 낙양's street (S_luoyang) and 허도 (S_xudu), every road paved,
the army mustering inside the walls.

  python3 ver5/tools/scene_maps/ye.py

Writes format 2 (internal/scenemap) straight: small tiles from origin 0, 0; u runs down to the right on screen,
v down to the left. Ground: d packed earth, i paving, . grass. A hall_u's door is on its +v face, a house_tile's
and a tavern's on its +v face.

Plan: 원소's 관부 behind a vermilion wall along the back (v = 5) with the great gate hall (palace_gate) on the avenue,
a paved square before it lined with banners. The paved avenue (u 29..35) runs down the middle under willows and
lanterns, the paved cross street along v 27..30.
  back left   the drill ground: troops mustering for 백마 — weapon racks, armour stands, command flags, braziers.
  back right  a general's house: a hall inside a white compound wall with an open gate (cwall, cgate_open).
  near left   the house given to 유비: one hall in a plum garden, flagstones down to the street, a well.
  near right  the market, grain carts and round granaries (the army's stores).
The tavern and the general store face the cross street either side of the avenue; the smithy down the near lanes.
"""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
W, H = 64, 56
WALL_V = 5
GATE_U = 32

g = [['d'] * W for _ in range(H)]
obj = []
walls = []


def paint(u0, v0, u1, v1, ch):
    for v in range(max(0, v0), min(H, v1 + 1)):
        for u in range(max(0, u0), min(W, u1 + 1)):
            g[v][u] = ch


def put(name, u, v):
    obj.append([name, u, v])


# behind the wall: paved courts and the roofs of 원소's halls
paint(0, 0, W - 1, WALL_V + 2, 'i')
put('hall_u', 12, 1), put('pavilion', 22, 1), put('hall_u', 43, 1), put('hall_u', 55, 1)
for u in range(W):
    d = u - GATE_U
    if abs(d) > 5:
        put({0: 'pwall_u_end_m', W - 1: 'pwall_u_end_p'}.get(u, 'pwall_u'), u, WALL_V)
    elif abs(d) == 5:
        put('pwall_u_gate_m' if d < 0 else 'pwall_u_gate_p', u, WALL_V)
put('palace_gate', GATE_U, WALL_V)

# the square before the gate, the avenue, the cross street
paint(16, 8, 48, 13, 'i')
paint(29, 8, 35, H - 1, 'i')
paint(0, 27, W - 1, 30, 'i')
for u in (18, 24, 40, 46):
    put('banner', u, 9)
put('lantern', 25, 12), put('lantern', 39, 12), put('command_flag', 20, 12), put('command_flag', 44, 12)
for v in (17, 23, 34, 41, 48):
    put('stree_willow', 27, v), put('stree_willow', 37, v)
for v in (20, 37, 45, 52):
    put('lantern', 28, v), put('lantern', 36, v)

# the drill ground, back left: the army mustering for 백마
paint(2, 15, 22, 24, 'd')
for u in (4, 9, 14, 19):
    put('banner', u, 15)
put('weapon_rack', 5, 18), put('weapon_rack', 8, 18), put('armor_stand', 11, 18), put('armor_stand', 13, 18)
put('command_flag', 17, 19), put('brazier', 4, 22), put('brazier', 20, 22), put('grain_sack', 15, 23)
put('weapon_rack', 7, 23), put('straw_mat', 11, 22)

# a general's house, back right: a hall in a white compound wall, the open gate on its front wall
GEN = (42, 61, 15, 24)       # u0, u1, v0, v1 of the wall lines
GGATE = 52
paint(43, 16, 60, 23, 'i')
paint(GGATE - 2, GEN[3], GGATE + 1, GEN[3] + 2, 'i')
walls.extend([['cwall', [GEN[0], GEN[2]], [GEN[1], GEN[2]]], ['cwall', [GEN[0], GEN[3]], [GEN[1], GEN[3]]],
              ['cwall', [GEN[0], GEN[2]], [GEN[0], GEN[3]]], ['cwall', [GEN[1], GEN[2]], [GEN[1], GEN[3]]]])
obj.append(['cgate_open', GGATE, GEN[3], {'wood': '#6a3a2a'}])
obj.extend([['hall_u', 51, 19, 8, 4], ['flagstones_0', 51, 22, 8, 4]])
put('stree_pine_0', 44, 17), put('stree_pine_1', 59, 17), put('taihu_rock', 45, 22), put('lantern', 49, 22)
put('lantern', 55, 22), put('banner', 49, 25), put('banner', 56, 25)

# the tavern and the general store on the cross street
put('tavern', 23, 25), put('wine_jar', 20, 26), put('table_wine', 19, 31), put('bench', 18, 31)
put('house_tile', 41, 25), put('market_stall_jar', 44, 26), put('grain_sack', 39, 26)

# 유비's house, near left: one hall in a plum garden, flagstones down to the street, a well
paint(6, 34, 20, 41, '.')
put('hall_u', 13, 38)
put('stree_plum_0', 9, 35), put('stree_plum_1', 17, 35), put('stree_plum_0', 8, 40)
put('lantern', 11, 40), put('lantern', 16, 40)
put('flagstones_0', 13, 40), put('flagstones_1', 13, 41)
put('well', 20, 43), put('water_bucket', 21, 44)
paint(0, 46, 27, 47, 'i')
for u, v, name in [(3, 33, 'house_tile_x'), (23, 36, 'house_tile_x'), (4, 51, 'house_tile'), (10, 52, 'house_tile_x'),
                   (17, 51, 'house_tile'), (23, 52, 'house_tile')]:
    put(name, u, v)
put('stree_broad_0', 1, 42)

# the market, near right, and the army's stores beside it
paint(38, 36, W - 1, 38, 'i')
for i, u in enumerate((40, 44, 48, 52, 56)):
    put(('market_stall', 'market_stall_cloth', 'market_stall_jar')[i % 3], u, 34)
for i, u in enumerate((42, 46, 50, 54)):
    put(('market_stall_cloth', 'market_stall_jar', 'market_stall')[i % 3], u, 41)
put('granary_round', 60, 33), put('granary_round', 60, 41), put('grain_cart', 58, 37), put('grain_cart_v', 39, 44)
put('grain_sack', 57, 44), put('water_jar', 38, 33), put('wine_jar', 55, 45)
paint(38, 50, 60, 51, 'i')
for u, v, name in [(43, 48, 'house_tile'), (50, 49, 'house_tile_x'), (57, 48, 'house_tile'), (46, 54, 'house_tile_x')]:
    put(name, u, v)   # (43, 48) the 대장간

art = {'id': 'S_ye', 'name': '업 거리', 'format': 2, 'size': [W // 2, H // 2], 'origin': [0, 0],
       'ground': [''.join(r) for r in g], 'objects': obj, 'walls': walls}

if __name__ == '__main__':
    json.dump(art, open(GFX / 'maps/S_ye.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if 'maps/S_ye.json' not in index['maps']:
        index['maps'].append('maps/S_ye.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', 'S_ye'], cwd=GFX.parents[1], check=True)
    print('S_ye', art['size'], len(obj), 'objects')
