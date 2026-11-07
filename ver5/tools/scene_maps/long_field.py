#!/usr/bin/env python3
"""Make S_long_field: the long open field of 광릉 (chapter 14) where 관우·장비 and 기령 face off, 여포's halberd lands
between them, and the three leaders ride off south-west talking until 여포 shoots back at the halberd 150 paces away.

  python3 ver5/tools/scene_maps/long_field.py

Format 2, an outdoor field like S_field_road: 12 cells across (u) and 165 long (v), the picture running 16 cells past
the sides (north-west, south-east: an arrow flying high still sees grass) and 6 past the ends. A dirt road runs down the middle (u = 5-6) and ends at v = 159 in a clearing of bare earth with a milestone where
it opens out, an old pavilion and old trees: the end of the ride, where 여포 shoots back along the road.
The head of the field (v < 16) is the open ground where the armies stand, the 관우·장비 side at u 0-3, 기령's at
u 9-12, the halberd's mark (6, 9) between them; the rest is grass with trees, bushes and rocks along both sides of
the road, scattered but never on it, so the ground shows how far the riders and the arrow have gone.

The canvas is one picture of the whole map (about 6400 x 3200 px at the game's scale): 150 cells is about as long as a
scene map can be (a 500-cell field would pass the 16384 px texture limit).
"""
import json
import random
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
ID = 'S_long_field'
W, H = 12, 165     # game cells
MU, MV = 16, 6     # cells of picture past the edges across (u: north-west, south-east) and along (v)
ROAD = (5, 6)      # the road's two columns (u)
END = 159          # where the ride ends and 여포 shoots: 150 cells from the halberd's mark (v = 9)
CLEAR = 4.5        # the clearing's radius there, cells

rnd = random.Random(14)
cols, rows = 2 * (W + 2 * MU), 2 * (H + 2 * MV)
ground = [['.'] * cols for _ in range(rows)]
for V in range(min(rows, 2 * (END + MV))):
    for u in ROAD:
        for k in (0, 1):
            ground[V][2 * (u + MU) + k] = 'd'
# the trampled ground at the head of the field where the two armies stand
for V in range(2 * MV + 2, 2 * MV + 30):
    for U in range(2 * MU - 2, 2 * (W + MU) + 2):
        if rnd.random() < .35:
            ground[V][U] = 'd'

# the road ends in a trodden clearing: a ring of bare earth, ragged at its rim
for V in range(rows):
    for U in range(cols):
        u, v = (U + .5) / 2 - MU, (V + .5) / 2 - MV
        d = ((u - 6) ** 2 + (v - END) ** 2) ** .5
        if d < CLEAR - .6 or d < CLEAR + .4 and rnd.random() < .5:
            ground[V][U] = 'd'

obj = []
TREES = ['stree_willow', 'stree_broad_0', 'stree_broad_1', 'stree_pine_0', 'stree_pine_1', 'stree_old']
BUSHES = ['stree_bush_0', 'stree_bush_1']


def free(u, v):
    near_end = ((u - 6) ** 2 + (v - END) ** 2) ** .5 < CLEAR + 3
    return not (ROAD[0] - 1 <= u <= ROAD[1] + 1) and not (v < 16 and -1 <= u <= W) and not near_end


def put(name, u, v):
    obj.append([name, int(2 * u + 1), int(2 * v + 1)])


# beyond the army ground: a few trees framing it, clear of the rows (v 4-14) the armies leave by
for u, v in ((-4, 1), (-7, 2), (W + 3, 1), (W + 6, 2), (-5, 16), (W + 4, 16)):
    put(rnd.choice(TREES), u, v)
v = 18
while v < H + MV:
    for side in (-1, 1, -1, 1):
        if rnd.random() < .75:
            u = rnd.randint(-MU + 1, 2) if side < 0 else rnd.randint(9, W + MU - 2)
            if free(u, v):
                put(rnd.choice(TREES if rnd.random() < .45 else BUSHES), u, v + rnd.random() * .8)
    v += rnd.randint(2, 4)

# the landmarks of the end: a milestone by the road where it opens out, an old pavilion and old trees about the
# clearing, a heap of stones
put('milestone', 10, END - 6)  # beside the riders' lane (u 4-8), not in it
put('pavilion', -1, END + 1)
put('stree_old', 12, END + 3)
put('stree_old', 1, END + 5)
put('rock_pile_1', 11, END - 3)
put('stree_willow', -2, END - 4)

art = {'id': ID, 'name': '광릉 들판', 'format': 2, 'size': [W, H], 'origin': [-2 * MU, -2 * MV],
       'height': ['0' * cols for _ in range(rows)],
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
