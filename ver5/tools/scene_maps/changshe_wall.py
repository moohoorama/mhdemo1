#!/usr/bin/env python3
"""Make S_changshe_wall, the top of 장사성's wall in 영천 at night, 황보숭 looking out over the 황건 fires (chapter 2,
c02-castle).

  python3 ver5/tools/scene_maps/changshe_wall.py

Format 2 (internal/scenemap), the same wall top as 백문루 (S_baimen, xiapi_baimen.py) without the flood: the walk is
height 4 (scene.json: the dry wall faces rdwall, the inner parapet and battlements rrim), paved Q, running along u the
whole length of the picture, 5 cells deep. Behind it (-v, the upper right) the battlements and, below them, the dry
fields where the 황건 have camped: tents, yellow banners and campfires spread back into the dark; the nearest banners
stand close under the wall so only their tops show over the battlements. In front (+v, lower left) the inner parapet,
the face dropping to the town's bare ground and a few roofs. On the walk: lanterns along the battlements, the
defenders' grain sacks and braziers, the wounded laid on a mat with a medicine pot. No tower.
"""
import json
import math
import random
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
ID = 'S_changshe_wall'
W, H = 14, 5               # game cells of walk
MU, BACK, FRONT = 16, 40, 12
rnd = random.Random(2)
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
    if 0 <= u + MU < cols and 0 <= v + BACK < rows:
        ground[v + BACK][u + MU] = g


for v in range(2 * H):
    for u in range(-MU, 2 * W + MU):
        cell(u, v, 'Q')
        height[v + BACK][u + MU] = '4'

# outside: dry fields of grass and trodden soil, worn bare round the camp
for v in range(-BACK, 0):
    for u in range(-MU, 2 * W + MU):
        n = math.sin(u * .27 + v * .19) + math.sin(u * .11 - v * .33) + rnd.random() * .9 - (.6 if v < -10 else 0)
        cell(u, v, 'd' if n > .5 else '.')
# inside: the town's bare ground
for v in range(2 * H, 2 * H + FRONT):
    for u in range(-MU, 2 * W + MU):
        cell(u, v, 'd')

# the walk: lanterns along the battlements, banners and a command flag, the defenders' stores and braziers, the wounded
for u in (.6, 2.4, 4.1, 5.9, 7.8, 10.1, 12.4):
    put('lantern', u, .4)
put('banner', 3.4, .3), put('banner', 6.8, .3), put('banner', 8.4, .3), put('command_flag', 9.2, .3)
put('incense', 8, 2)
put('grain_sack', 4.2, 2.3), put('grain_sack', 4.6, 2.6), put('grain_sack', 3.9, 2.7)
put('brazier', 5.1, 1.0), put('brazier', 8.6, 1.0)
put('straw_mat', 5.4, 2.7), put('bloody_cloth', 5.9, 3.2), put('medicine_pot', 3.4, 3.6)

# outside: the 황건 camp spread back into the dark; the nearest yellow banners close under the wall, their tops over
# the battlements (2.5 to 4.4 cells out from a 120 px walk with its 21 px battlements)
for u, v in ((1, -3.2), (5.5, -3.6), (9, -3.3), (12.5, -3.8), (16, -3.4)):
    put('banner_yellow', u, v)
for u, v in ((-2, -8), (3, -9), (8, -8.5), (13, -9), (18, -8), (0, -13), (6, -12.5), (11, -13.5), (16, -12),
             (-4, -15), (4, -16), (9, -17), (14, -16)):
    put(rnd.choice(['tent', 'tent_round']), u, v)
put('command_flag_yellow', 7, -10.5), put('command_flag_yellow', 15, -10)
for u, v in ((1, -6.5), (5, -6), (10, -6.5), (14.5, -6), (2.5, -11), (9, -11), (13, -11.5), (-1, -11), (7, -14.5),
             (12, -15), (17, -14.5)):
    put('campfire', u, v)

# inside: a few roofs of the town under the wall
for u, v in ((-1, 8.5), (3, 9), (7.5, 8.5), (12, 9), (16, 8.5)):
    put(rnd.choice(['house_tile', 'house_tile_x']), u, v)

art = {'id': ID, 'name': '장사성 성벽', 'format': 2, 'size': [W, H], 'origin': [-MU, -BACK],
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
