#!/usr/bin/env python3
"""Make S_xudu, the 허도 the 17th chapter walks (assets/scenario/hero/explore17.json): the new capital under 조조,
with two great houses — 조조's and the one given to 유비 — and the palace front.

  python3 ver5/tools/scene_maps/xudu.py

Writes format 2 (internal/scenemap) straight: small tiles from origin 0, 0; u runs down to the right on screen,
v down to the left. Ground: d packed earth, i paving, . grass. A hall_u's door is on its +v face, a hall_v's on its
+u face, a house_tile's on its +v face.

Plan: the palace wall along the back (v = 4) in vermilion plaster (pwall_u, gen_pixel_palace.py) from the board's
left edge to its right, stopping flush with both (pwall_u_end_m/p); the great gate hall (palace_gate) on the avenue,
the wall running across its terrace into its side walls, the tiles against it pwall_u_gate_m/p (drawn with the hall,
so they meet exactly). A paved square before it with guards' banners. The avenue (u 25..31) runs down the middle,
the cross street along v 24..27.
  back left   조조's house: the hall-like house of 탁현's mansion (hall_u, its door on +v) grown into a compound — two
              side halls (hall_v) facing each other across a paved court, pines and plums in a garden behind, a
              pavilion and a taihu rock — inside a white compound wall (cwall) with an open 솟을대문 (cgate_open)
              in its front wall, banners outside it.
  back right  the 관청 by the notice board.
  near left   유비's house: 탁현's mansion as it is — the one hall in a small plum garden, lanterns either side of the
              door, flagstones down to the street; a vegetable patch beside it (the 황숙 growing greens in 허도).
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
walls = []


def paint(u0, v0, u1, v1, ch):
    for v in range(max(0, v0), min(H, v1 + 1)):
        for u in range(max(0, u0), min(W, u1 + 1)):
            g[v][u] = ch


def put(name, u, v):
    obj.append([name, u, v])


# behind the palace wall: paved courts and the roofs of the palace halls
paint(0, 0, W - 1, WALL_V + 2, 'i')
put('hall_u', 10, 1), put('hall_u', 46, 1)
# the wall and the gate hall: the wall stops flush with the board's edges; the tiles against the gate's terrace and the
# gate's own (u GATE_U - 4 .. GATE_U + 4) are drawn with the hall
for u in range(W):
    d = u - GATE_U
    if abs(d) > 5:
        put({0: 'pwall_u_end_m', W - 1: 'pwall_u_end_p'}.get(u, 'pwall_u'), u, WALL_V)
    elif abs(d) == 5:
        put('pwall_u_gate_m' if d < 0 else 'pwall_u_gate_p', u, WALL_V)
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

# 조조의 저택: the main hall (hall_u, door +v at u 11.5) at the back of a paved court, the side halls (hall_v) either
# side of it, a garden of pines and plums behind with a pavilion and a taihu rock; banners and lanterns at the mouth
# The compound wall (cwall, gen_pixel_compound.py: white plaster, black tiles as drawn) round it from u 2 to 21, v 10
# to 20, the open main gate (cgate_open, its woodwork tinted a dark lacquer) on the front wall at u 12: its passage is
# small tiles (11, 20) and (12, 20), the explore exit's spot; the gatekeepers stand outside it. The halls and the
# flagstones sit half a tile toward +u (dx 8, dy 4) so they stand square behind the passage.
CAO = (2, 21, 10, 20)        # u0, u1, v0, v1 of the wall lines
GATE = 12
paint(3, 11, 20, 13, '.')
paint(3, 14, 20, 19, 'i')
paint(8, 21, 16, 23, 'i')    # the paved apron before the gate, and the paving through it
paint(GATE - 2, CAO[3], GATE + 1, CAO[3], 'i')
walls.extend([['cwall', [CAO[0], CAO[2]], [CAO[1], CAO[2]]], ['cwall', [CAO[0], CAO[3]], [CAO[1], CAO[3]]],
              ['cwall', [CAO[0], CAO[2]], [CAO[0], CAO[3]]], ['cwall', [CAO[1], CAO[2]], [CAO[1], CAO[3]]]])
obj.append(['cgate_open', GATE, CAO[3], {'wood': '#8c5a40'}])
obj.extend([['hall_u', 11, 15, 8, 4], ['hall_v', 6, 17, 8, 4], ['hall_v', 16, 17, 8, 4], ['flagstones_0', 11, 18, 8, 4]])
put('stree_pine_0', 4, 11), put('stree_pine_1', 19, 11), put('stree_plum_0', 8, 12), put('stree_plum_1', 15, 12)
put('pavilion', 18, 13), put('taihu_rock', 19, 18), put('taihu_rock', 3, 15), put('stree_pine_1', 3, 18)
put('banner', 8, 22), put('banner', 15, 22)
put('lantern', 9, 17), put('lantern', 14, 17)

# 관청 and its notice board, back right
paint(34, 13, 47, 21, 'i')
put('hall_v', 44, 16), put('banner', 41, 13), put('banner', 41, 19), put('notice_board', 36, 15)
put('tavern', 52, 12), put('wine_jar', 50, 14), put('stree_broad_0', 54, 18)   # the 주점 by the 관청

# 유비의 저택: 탁현's mansion — the one hall (hall_u, door +v at u 11.5) in a small plum garden, lanterns either side
# of the door, flagstones down to the street; a vegetable patch beside it, a well
paint(6, 32, 16, 39, '.')
put('hall_u', 11, 36)
put('stree_plum_0', 8, 33), put('stree_plum_1', 14, 33), put('stree_plum_0', 7, 38)
put('lantern', 9, 38), put('lantern', 14, 38)
put('flagstones_0', 11, 38), put('flagstones_1', 11, 39)
put('field_sprout', 18, 37), put('well', 16, 41), put('water_bucket', 17, 42)
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
for u, v, name in [(40, 45, 'house_tile'), (47, 46, 'house_tile'), (53, 44, 'house_tile_x')]:   # (40, 45) the 대장간
    put(name, u, v)

art = {'id': 'S_xudu', 'name': '허도 거리', 'format': 2, 'size': [W // 2, H // 2], 'origin': [0, 0],
       'ground': [''.join(r) for r in g], 'objects': obj, 'walls': walls}

if __name__ == '__main__':
    json.dump(art, open(GFX / 'maps/S_xudu.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if 'maps/S_xudu.json' not in index['maps']:
        index['maps'].append('maps/S_xudu.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', 'S_xudu'], cwd=GFX.parents[1], check=True)
    print('S_xudu', art['size'], len(obj), 'objects')
