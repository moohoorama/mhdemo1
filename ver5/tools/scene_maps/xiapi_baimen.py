#!/usr/bin/env python3
"""Make S_baimen, the top of 하비성's wall at its 백문루 gate, where 조조 judges 여포 and his officers (chapter 16, S#3).

  python3 ver5/tools/scene_maps/xiapi_baimen.py

Format 2 (internal/scenemap): small tiles, the playing area from (0, 0), the picture round it. Objects are placed in
game cells (a person's cell is a whole number; a prop's anchor may sit between cells) and turned into small-tile anchors
with canvas offsets. The wall's pieces are drawn by assets_candidate/tools/gen_pixel_rampart.py.

Plan: the wall runs along u across the picture, its walk (height 3, paving Q) the whole playing area, 6 cells deep.
Behind it (-v, the picture's upper right) the battlements, and below them, outside the town, the wet ground the flood
left: the moat a muddy ford along the wall's foot, puddles in the fields, a road from the gate. In front (+v, lower
left) the walk ends at the low inner parapet and the wall's face drops to the town. The gate tower stands at the back of
the walk in the middle (u = 5), the gate's passage opening in the inner face below it; 조조 sits on a dais before it, a mat runs from him to the front, where the prisoners are
brought along the walk from the right. 유비's party on the left of the mat, 허저 and the guards on the right.
"""
import json
import math
import random
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
ID = 'S_baimen'
W, H = 12, 6               # game cells of walk
MU, BACK, FRONT = 16, 36, 12   # small tiles of picture: either end, behind (outside the town), in front (inside)
AISLE = 5
rnd = random.Random(16)
obj = []


def put(name, u, v):
    """A prop at game-cell point (u, v) (a whole number is a cell's centre) as a small-tile anchor and offset."""
    U, V = math.floor(2 * u + 1), math.floor(2 * v + 1)
    x, y = math.floor((u - v) * 32 + .5), math.floor((u + v + 1) * 16 + .5)
    dx, dy = x - 16 * (U - V), y - 8 * (U + V + 1)
    obj.append([name, U, V] + ([dx, dy] if dx or dy else []))


cols, rows = 2 * W + 2 * MU, BACK + 2 * H + FRONT
ground = [['.'] * cols for _ in range(rows)]
height = [['0'] * cols for _ in range(rows)]


def cell(u, v, g):
    """small tile (u, v) of the playing area's coordinates"""
    if 0 <= u + MU < cols and 0 <= v + BACK < rows:
        ground[v + BACK][u + MU] = g


# the walk, the whole length of the picture
for v in range(2 * H):
    for u in range(-MU, 2 * W + MU):
        cell(u, v, 'Q')
        height[v + BACK][u + MU] = '3'
for v in range(6, 2 * H):
    cell(2 * AISLE, v, 'M'), cell(2 * AISLE + 1, v, 'M')

# outside: wet fields of grass and bare soil, the moat a ford along the foot, a road out from the gate, puddles
for v in range(-BACK, 0):
    for u in range(-MU, 2 * W + MU):
        n = math.sin(u * .31 + v * .17) + math.sin(u * .13 - v * .29) + rnd.random() * .8
        cell(u, v, 'd' if n > .7 else '.')
for v in range(-5, 0):
    for u in range(-MU, 2 * W + MU):
        cell(u, v, 'R' if v < -1 and (u * 3 + v * 5) % 11 else 'd')
for v in range(-BACK, -5):
    for u in range(2 * AISLE - 1, 2 * AISLE + 3):
        cell(u + (-v) // 6, v, 'd')
for _ in range(30):                 # puddles: blobs grown a tile at a time
    pu, pv = rnd.randrange(-MU, 2 * W + MU), rnd.randrange(-BACK + 2, -8)
    for _ in range(rnd.randrange(2, 9)):
        cell(pu, pv, '~')
        du, dv = rnd.choice(((1, 0), (-1, 0), (0, 1), (0, -1)))
        pu, pv = pu + du, max(-BACK + 1, min(-7, pv + dv))

# inside: the town's bare earth along the wall
for v in range(2 * H, 2 * H + FRONT):
    for u in range(-MU, 2 * W + MU):
        cell(u, v, 'd')

# the gate tower at the back of the walk, 조조's dais before its porch, banners along the battlements
put('baimen_tower', AISLE, .9)
put('dais', AISLE, 2.9)
for i in range(4):                      # the passage under the tower, opening in the inner face, a slice a tile
    put(f'baimen_gate_{i}', AISLE + (i - 1.5) / 2, H - .5)
put('command_flag', AISLE - 1.5, 2.7), put('command_flag', AISLE + 1.5, 2.7)
for u in (0, 1.5, 9, 10.5):
    put('banner', u, .1)
put('brazier', .5, 2.5), put('brazier', 10.5, 2.5), put('brazier', 1.5, 4.4), put('brazier', 9.5, 3.6)
put('weapon_rack', 0, 4.2), put('armor_stand', 10.8, 4.2)
# trees out in the fields, clear of the puddles; those just past the wall's foot show only their crowns over the
# battlements, which tells how far below the fields lie
for u, v in ((-6, -15), (-2, -13), (3, -16), (8.5, -12), (13, -15), (17, -13), (19, -16), (-4, -9), (16, -9.5),
             (7, -3.5), (10, -3.5), (12.5, -4), (15, -3.5), (2, -3.5), (18, -4)):
    if ground[math.floor(2 * v + 1) + BACK][math.floor(2 * u + 1) + MU] in '.d':
        put(rnd.choice(['stree_broad_0', 'stree_broad_1', 'stree_willow']), u, v)

art = {'id': ID, 'name': '하비성 백문루', 'format': 2, 'size': [W, H], 'origin': [-MU, -BACK],
       'grass': {'hue': 6, 'sat': .72, 'val': .8},      # the fields still sodden: duller, darker
       'ground': [''.join(r) for r in ground], 'height': [''.join(r) for r in height], 'objects': obj}

if __name__ == '__main__':
    json.dump(art, open(GFX / f'maps/{ID}.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if f'maps/{ID}.json' not in index['maps']:
        index['maps'].append(f'maps/{ID}.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', ID], cwd=GFX.parents[1], check=True)
    print(ID, art['size'], len(obj), 'objects')
