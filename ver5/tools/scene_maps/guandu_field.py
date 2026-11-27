#!/usr/bin/env python3
"""Make S_guandu_field: the battle of 관도 for the title reel — 원소's cavalry galloping across an open plain from the
north-east into 조조's palisaded camp, pikemen holding its gates and archers shooting from the trampled ground
behind it.

  python3 ver5/tools/scene_maps/guandu_field.py          write the map, register it, format it
  python3 ver5/tools/scene_maps/guandu_field.py --show   also print the map one game cell per character

Format 2, outdoors, game cells (u grows toward the lower right of the screen, v toward the lower left):

  v -30 .. 39   the plain: grass with trodden dirt, nothing standing in u -8..48 (bushes, rocks and a few trees
                only further out on the flanks and beyond v -22), so riders coming from the north-east (small v)
                never pass through anything
  v 40          a row of antler barriers (녹각) along u, broken in front of each gate
  v 42          the palisade: one line of stakes (sfence) along u across the whole picture, its outer face toward the
                plain (north-east); four gates 4 cells wide, u 4-7, 14-17, 24-27 and 34-37
  v 43 .. 46    the strip behind the palisade (pikemen in reserve); watchtowers between the gates at v 44
  v 47 .. 57    the archers' ground: wide trampled earth, empty over u -6..45
  v 58 ..       tents, banners, the command tent, carts and fires, out to the end of the picture; tents also fill the
                camp on both flanks (u < -6, u > 45) from v 44 down

The picture runs 30 cells past the playing area on every side so its edges stay off a zoom-1 screen.
"""
import json
import math
import random
import subprocess
import sys
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
ID = 'S_guandu_field'
W, H = 40, 64          # game cells of the playing area
M = 30                 # cells of picture past each side
U0, U1 = -M, W + M     # game cells covered by the picture: u in [U0, U1), v in [V0, V1)
V0, V1 = -M, H + M
PAL = 42               # the palisade's row (game cell v)
ANTLER = 40            # the antler barriers' row
GATES = [(4, 7), (14, 17), (24, 27), (34, 37)]   # gate gaps, game cells u (inclusive)
TOWERS = [-1, 11, 21, 31, 41]                     # watchtowers behind the palisade, between the gates
CLEAR_U = (-8, 48)     # the riding plain and the archers' ground stay clear across these u
ARCHERS = (47, 57)

rnd = random.Random(200)
cols, rows = 2 * (U1 - U0), 2 * (V1 - V0)
ground = [['.'] * cols for _ in range(rows)]


def noise(u, v):
    return (math.sin(u * .31 + v * .17) + math.sin(u * .13 - v * .29 + 1.7) + .6 * math.sin(u * .71 + v * .53 + .4))


for R in range(rows):
    for C in range(cols):
        u, v = (C + .5) / 2 + U0, (R + .5) / 2 + V0
        n = noise(u, v) + rnd.uniform(-.35, .35)
        if v < PAL:
            # the plain: dirt patches, more of them close to the palisade where the riders churn the ground
            churn = max(0., (v - 24) / 18)
            dirt = n > 1.15 - 1.3 * churn
        else:
            # inside the camp: trodden earth everywhere people stand, some grass left between the tents
            dirt = v < ARCHERS[1] + 1.5 or n > -.6
        if dirt:
            ground[R][C] = 'd'

obj = []


def put(name, u, v):
    """An object on game cell (u, v): u and v may end in .5 to sit on the second small tile."""
    obj.append([name, int(round(2 * u)), int(round(2 * v))])


def in_gate(u):
    return any(a <= u <= b for a, b in GATES)


# the palisade: one stake piece per small tile along u at V = 2 * PAL, with posts at the gates' sides
for U in range(2 * U0, 2 * U1):
    u = U / 2
    if in_gate(math.floor(u)):
        continue
    left, right = in_gate(math.floor((U - 1) / 2)), in_gate(math.floor((U + 1) / 2))
    obj.append(['sfence_04' if left else 'sfence_01' if right else 'sfence_05', U, 2 * PAL])

# antler barriers in front, one per game cell, broken a cell wider than each gate on both sides
for u in range(U0, U1):
    if not any(a - 1 <= u <= b + 1 for a, b in GATES):
        put('antler_barrier', u, ANTLER + .5)

for u in TOWERS:
    put('watchtower', u + .5, 44)
# banners on the gate posts' inner side
for a, b in GATES:
    put('banner', a - 1, 43)
    put('banner', b + 1.5, 43)
# flanks of the archers' ground
for v in (48, 52, 56):
    put('banner', -7, v)
    put('banner', 46.5, v)
put('weapon_rack', -8, 50)
put('weapon_rack', 47.5, 54)
put('grain_sack', -7.5, 54)

# the back of the camp: rows of tents with lanes between them, the command tent in the middle
put('command_tent', 20, 64)
put('command_flag', 16.5, 61.5)
put('command_flag', 23.5, 61.5)
put('brazier', 17.5, 60)
put('brazier', 22.5, 60)
put('banner', 20, 60)
for v in range(60, V1 - 1, 5):
    for u in range(U0 + 2, U1 - 1, 4):
        uu, vv = u + (2 if (v // 5) % 2 else 0), v
        if 13 <= uu <= 27 and 59 <= vv <= 68:
            continue                       # the command tent's yard
        if rnd.random() < .12:
            continue                       # a gap in the rows now and then
        put(rnd.choice(['tent', 'tent', 'tent', 'tent_round']), uu, vv)
    for u in range(U0 + 4, U1 - 1, 12):
        uu = u + rnd.randint(0, 3)
        if 13 <= uu <= 27 and v <= 68:
            continue
        put(rnd.choice(['campfire', 'grain_sack', 'water_jar', 'banner', 'grain_cart', 'weapon_rack']), uu, v + 2.5)
# tents on the flanks of the camp, from the palisade down
for v in range(45, 58, 4):
    for u in list(range(U0 + 2, CLEAR_U[0] - 3, 4)) + list(range(CLEAR_U[1] + 3, U1 - 1, 4)):
        if rnd.random() < .85:
            put(rnd.choice(['tent', 'tent', 'tent_round']), u + (2 if (v // 4) % 2 else 0), v)
for u in (-12, -24, 54, 64):
    put('watchtower', u + .5, 44)

# the plain's flanks and the far side: bushes, rocks, a few trees, never in the riders' lanes
for _ in range(70):
    side = rnd.random()
    if side < .35:
        u, v = rnd.uniform(U0 + 1, CLEAR_U[0] - 2), rnd.uniform(V0 + 1, ANTLER - 3)
    elif side < .7:
        u, v = rnd.uniform(CLEAR_U[1] + 2, U1 - 1), rnd.uniform(V0 + 1, ANTLER - 3)
    else:
        u, v = rnd.uniform(U0 + 1, U1 - 1), rnd.uniform(V0 + 1, -22)
    kind = rnd.random()
    name = (rnd.choice(['stree_bush_0', 'stree_bush_1']) if kind < .6 else
            rnd.choice(['rock_pile_0', 'rock_pile_1', 'rock_pile_2']) if kind < .8 else
            rnd.choice(['stree_broad_0', 'stree_broad_1', 'stree_old', 'stree_dead']))
    put(name, round(u * 2) / 2, round(v * 2) / 2)

art = {'id': ID, 'name': '관도 들판', 'format': 2, 'size': [W, H], 'origin': [2 * U0, 2 * V0],
       'grass': {'hue': -18, 'sat': 0.78, 'val': 1.0},
       'ground': [''.join(r) for r in ground], 'objects': obj}


def register(relative):
    """Add the map to index.json's maps by text, so other sessions' edits of the file stay as they are."""
    path = GFX / 'index.json'
    text = path.read_text()
    if f'"{relative}"' in text:
        return
    end = text.index('\n ],', text.index(' "maps": ['))
    path.write_text(text[:end] + f',\n  "{relative}"' + text[end:])


def show():
    blocked = set()
    for name, U, V, *_ in obj:
        blocked.add((U // 2, V // 2))
    for v in range(V0, V1):
        line = []
        for u in range(U0, U1):
            g = ground[2 * (v - V0)][2 * (u - U0)]
            line.append('#' if (u, v) in blocked else g)
        print(f'{v:4d} ' + ''.join(line))


if __name__ == '__main__':
    (GFX / f'maps/{ID}.json').write_text(json.dumps(art, ensure_ascii=False, indent=1) + '\n')
    register(f'maps/{ID}.json')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', ID], cwd=GFX.parents[1], check=True)
    print(ID, art['size'], art['origin'], len(obj), 'objects')
    if '--show' in sys.argv:
        show()
