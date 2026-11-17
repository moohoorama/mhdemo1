#!/usr/bin/env python3
"""Make S_yanjin: the south bank of the Yellow River at 연진 (chapter 19), where 문추 leads 원소's van against 장료,
falls on the baggage 조조 left as bait, and is cut down by 관우 on 적토마 while 유비 watches from the rear guard.

  python3 ver5/tools/scene_maps/yanjin.py

Format 2, an outdoor field like S_long_field: 14 cells across (u) and 62 long (v), the picture running past the sides
and ends so a camera following a rider never sees the edge. The head of the map (v < BLUFF) is a high bluff (height 2)
where 유비 parts from 문추 and stays with the rear guard; below it the plain runs on along the river (u >= 11, willows
on its bank) with a dirt road down the middle (u = 5-6). The clash is far down the plain (v 36-58): 장료's line, and
behind it the abandoned baggage (carts, chests, silk, sacks) strewn either side of the road. The plain between is
open grass with a few bushes away from the riders' lanes (u 3-10), so the chase back up the field shows how far it
runs and ends well short of the bluff.
"""
import json
import math
import random
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
ID = 'S_yanjin'
W, H = 14, 62      # game cells
BLUFF = 8          # the bluff's top runs up to this v (its face looks down the plain)
MU, MV = 14, 8     # cells of picture past the edges across (u) and along (v)
ROAD = (5, 6)
RIVER = 11         # water from this u outwards (south-east)

rnd = random.Random(19)
cols, rows = 2 * (W + 2 * MU), 2 * (H + 2 * MV)
ground = [['.'] * cols for _ in range(rows)]
height = [['0'] * cols for _ in range(rows)]


def cell(U, V):
    return (U - .5) / 2 - MU, (V - .5) / 2 - MV  # small tile centres in game cells (smallTileAt: 2u + 1)


def bank(v):
    return RIVER + .6 * (1 + math.sin(v / 5.3)) - .3


def edge(u):
    """The bluff's face: a small tile further out or in here and there, so it does not run ruler-straight."""
    return BLUFF + .5 * round(math.sin(u * .9) + .4 * math.sin(u * 2.3))


for V in range(rows):
    for U in range(cols):
        u, v = cell(U, V)
        if u >= bank(v):
            ground[V][U] = '~'
            continue
        if v < edge(u):
            height[V][U] = '2'
        if ROAD[0] <= u < ROAD[1] + 1:
            ground[V][U] = 'd'
        elif 34 <= v <= 52 and 1 <= u <= 10 and rnd.random() < .16:  # the trampled ground of the clash
            ground[V][U] = 'd'
        elif 0 <= v < BLUFF and 1 <= u <= 9 and rnd.random() < .18:  # where the rear guard waits on the bluff
            ground[V][U] = 'd'

obj = []


def put(name, u, v):
    obj.append([name, int(round(2 * u)) + 1, int(round(2 * v)) + 1])


# willows along the bank below the bluff, never in the water's edge lane the riders use (u 9-10)
v = BLUFF + 3
while v < H + MV:
    put('stree_willow', RIVER - .5 + rnd.random() * .3, v + rnd.random())
    v += rnd.randint(4, 6)
# the far side (north-west) of the field: trees and bushes beyond the lanes, on the bluff and below it
v = -6
while v < H + MV:
    u = rnd.uniform(-MU + 2, -10 if v < BLUFF + 2 else -1.5)  # the van rides off north-west along the bluff
    if abs(v - edge(u)) > 1.5:
        put(rnd.choice(['stree_broad_0', 'stree_broad_1', 'stree_pine_0', 'stree_bush_0', 'stree_bush_1']), u, v)
    v += rnd.randint(2, 4)
# pines on the bluff behind the rear guard
for u, v in ((0, -9), (3, -10), (8, -9), (11, -8)):
    put(rnd.choice(['stree_pine_0', 'stree_pine_1']), u, v)
# a few bushes in the open plain, clear of the lanes (u 2-10) so the chase never runs through them
for u, v in ((1, 15), (1, 23), (1.5, 58), (0.5, 30)):
    put('stree_bush_1' if rnd.random() < .5 else 'stree_bush_0', u, v)

# the bait: carts, chests, silk and sacks strewn either side of the road behind 장료's line
for name, u, v in (('grain_cart', 3, 48), ('grain_cart_v', 8, 49), ('hoard_chests', 4, 50.5), ('hoard_coin_chest', 7.5, 47.5),
                   ('hoard_silk', 2.5, 49.8), ('hoard_silk', 8.6, 51), ('grain_sack', 3.4, 52), ('grain_sack', 7.6, 52.6),
                   ('wine_jar', 9, 47.6), ('grain_cart', 2.6, 54), ('hoard_coin_chest', 8.4, 54.4), ('grain_cart_v', 3, 57),
                   ('hoard_silk', 8.2, 56.5)):
    put(name, u, v)
for u, v in ((4.2, 37), (8.3, 39.4), (2.4, 40.6)):
    put('broken_spear', u, v)
put('fallen_flag', 9.2, 37.6)

art = {'id': ID, 'name': '연진 강가', 'format': 2, 'size': [W, H], 'origin': [-2 * MU, -2 * MV],
       'height': [''.join(r) for r in height],
       'ground': [''.join(r) for r in ground], 'objects': obj}

if __name__ == '__main__':
    json.dump(art, open(GFX / f'maps/{ID}.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if f'maps/{ID}.json' not in index['maps']:
        index['maps'].append(f'maps/{ID}.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', ID], cwd=GFX.parents[1], check=True)
    print(ID, art['size'], len(obj), 'objects')
