#!/usr/bin/env python3
"""Make S_gyeqiao_gully: the battle of 계교 for the title reel — 공손찬 flees on horseback from 문추 across an open field
into a gully that ends against a cliff, and 조운 rides down a side valley into the gully between them.

  python3 ver5/tools/scene_maps/gyeqiao_gully.py          write the map, register it, format it
  python3 ver5/tools/scene_maps/gyeqiao_gully.py --show   also print the cells: ^ high, # blocked, ~ water, = bridge,
                                                          o hidden by a ridge

Format 2, outdoors, game cells (u grows toward the lower right of the screen, v toward the lower left):

  field     u 21-40, v 2-21: open grass; a dirt track along v 11-12 from the south-east edge to the gully's mouth
  stream    v 19 from the foot of the south-west ridge (u 21) out to the south-east edge, a small wooden bridge on
            u 29-30 with a path crossing it along those u; nothing of it lies on the chase
  gully     u 6-20, v 10-13, floor of gravel and dirt between a ridge (height 2) on its north-east (v <= 9), whose
            face shows, and one on its south-west standing back at v >= 19 so its lifted top hides no one on the
            floor (the scree v 14-18 under it is not walked); it ends against the cliff at u <= 5 (dead end u 6)
  side      a valley four cells wide running up the screen (toward -u and -v together: cells with u - v in 0..3,
            v <= 9) into the gully's north-east side at u 9-12, v 9: from its mouth it climbs to the picture's edge, so a
            rider coming down it is never behind a ridge; its walls show their faces on both sides

The picture runs 30 cells past the playing area on every side.
"""
import json
import math
import random
import subprocess
import sys
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
ID = 'S_gyeqiao_gully'
W, H = 41, 22          # the playing area: the gully and the field
M = 30
U0, U1 = -M, W + M
V0, V1 = -M, H + M
FIELD = (21, 40, 2, 21)   # u from, to, v from, to (inclusive)
GULLY = (6, 20, 10, 13)
SIDE = (0, 3)             # the side valley: u - v in this range, v < GULLY[2]
TRACK = (11, 12)
STREAM = 19
BRIDGE = (29, 30)
SW_RIDGE = 19
LIFT = 146

rnd = random.Random(191)
cols, rows = 2 * (U1 - U0), 2 * (V1 - V0)
ground = [['.'] * cols for _ in range(rows)]
height = [['0'] * cols for _ in range(rows)]


def wob(x, seed):
    """0, .5 or 1 cells, so a ridge's edge steps in and out by small tiles."""
    return .5 * round(1 + .9 * math.sin(x * .21 + seed) + .3 * math.sin(x * .53 + 2 * seed))


def side(u, v):
    return v < GULLY[2] and SIDE[0] <= u - v < SIDE[1] + 1


def high(u, v):
    """Whether the small tile centred on game point (u, v) is ridge. The edges only step away from where people walk."""
    if u >= FIELD[0]:
        return False
    if v < GULLY[2] - wob(u, 1):
        # the north-east ridge, but for the side valley, its walls stepping away from it
        d = u - v
        return not (SIDE[0] - wob(v, 3) <= d < SIDE[1] + 1 + wob(v, 4))
    # the dead end and the mass behind it, the south-west ridge
    return u < GULLY[0] - wob(v, 5) or v >= SW_RIDGE + wob(u, 2)


def stream(u, v):
    return u >= FIELD[0] and STREAM <= v < STREAM + 1


for R in range(rows):
    for C in range(cols):
        u, v = (C + .5) / 2 + U0, (R + .5) / 2 + V0
        n = math.sin(u * .37 + v * .21) + math.sin(u * .11 - v * .33 + 1.3) + rnd.uniform(-.4, .4)
        if high(u, v):
            height[R][C] = '2'
            ground[R][C] = 'd' if n > .8 else '.'
        elif stream(u, v):
            ground[R][C] = 'b' if BRIDGE[0] <= u < BRIDGE[1] + 1 else '~'
        elif BRIDGE[0] <= u < BRIDGE[1] + 1 and v >= TRACK[1] + 1:
            ground[R][C] = 'd'                            # the path over the bridge
        elif u < FIELD[0] and (side(u, v) or v < SW_RIDGE):
            ground[R][C] = 'd' if n > -.9 else '.'        # the gully's gravel and the side valley
        elif TRACK[0] <= v < TRACK[1] + 1 or (TRACK[0] - 1 <= v < TRACK[1] + 2 and n > .2):
            ground[R][C] = 'd'                            # the track the riders churn
        else:
            ground[R][C] = 'd' if n > 1.1 else '.'

obj = []


def put(name, u, v):
    """An object on game point (u, v): u and v may end in .5 to sit on the second small tile."""
    obj.append([name, int(math.floor(2 * u)), int(math.floor(2 * v))])


# the bridge's deck: one small plank deck on each small tile of the stream under it
for R in range(rows):
    for C in range(cols):
        if ground[R][C] == 'b':
            obj.append(['sbridge_v', C + 2 * U0, R + 2 * V0])


def high_cell(u, v):
    return all(high(u + a, v + b) for a in (.25, .75) for b in (.25, .75))


def walk_cell(u, v):
    return (FIELD[0] <= u <= FIELD[1] and FIELD[2] <= v <= FIELD[3] and v != STREAM or
            GULLY[0] - 1 <= u <= GULLY[1] and GULLY[2] <= v <= GULLY[3] or side(u + .5, v + .5))


def covers(u, v, tall, wide, pu, pv):
    """Whether a thing `tall` x `wide` px on cell (u, v) is drawn over a person (60 px) on cell (pu, pv)."""
    lift = LIFT if high_cell(u, v) else 0
    dx, dy = 32 * ((u - pu) - (v - pv)), 16 * ((u - pu) + (v - pv))
    return (u, v) != (pu, pv) and abs(dx) < wide / 2 + 20 and dy - lift > -60 and dy - lift - tall < 0 and (dy > 0 or lift)


def clear_of_walk(u, v, tall, wide):
    return not any(walk_cell(pu, pv) and covers(u, v, tall, wide, pu, pv)
                   for pu in range(u - 12, u + 3) for pv in range(v - 12, v + 3))


# ridge tops: pines and rocks, only where their lifted pictures cover no one
for v in range(V0 + 1, V1 - 1):
    for u in range(U0 + 1, FIELD[0]):
        if not high_cell(u, v):
            continue
        r = rnd.random()
        if r < .08:
            name, tall, wide = rnd.choice(['stree_pine_0', 'stree_pine_1']), 130, 92
        elif r < .095:
            name, tall, wide = 'stree_dead', 86, 36
        elif r < .12:
            name, tall, wide = rnd.choice(['rock_pile_0', 'rock_pile_1', 'rock_pile_2']), 30, 50
        else:
            continue
        if clear_of_walk(u, v, tall, wide):
            put(name, u + rnd.choice([0, .5]), v + rnd.choice([0, .5]))

# scree under the south-west ridge
for v in range(GULLY[3] + 1, SW_RIDGE):
    for u in range(GULLY[0], FIELD[0]):
        if not high_cell(u, v) and rnd.random() < .12:
            put(rnd.choice(['rock_pile_0', 'rock_pile_1', 'rock_pile_2', 'stree_bush_1']), u + .5, v + .5)

# the far sides of the field: willows along the stream, bushes and a few trees, none on the chase or the path
for u in range(FIELD[0] + 2, U1 - 1, 3):
    if BRIDGE[0] - 2 <= u <= BRIDGE[1] + 2:
        continue
    if rnd.random() < .6:
        put('stree_willow', u + rnd.choice([0, .5]), STREAM + 1.5 + rnd.choice([0, .5]))
    if rnd.random() < .3:
        put('stree_bush_0', u + .5, STREAM - 1)
for _ in range(90):
    u, v = rnd.uniform(FIELD[0] - 4, U1 - 1), rnd.uniform(V0 + 1, V1 - 1)
    near = FIELD[0] - 1 <= u <= FIELD[1] + 1 and FIELD[2] - 1 <= v <= FIELD[3] + 1
    if near or high_cell(int(u), int(v)) or abs(v - STREAM - .5) < 1.5 or BRIDGE[0] - 1 <= u <= BRIDGE[1] + 2:
        continue
    if v < FIELD[2] and u < FIELD[0] + 2:
        continue
    put(rnd.choice(['stree_broad_0', 'stree_broad_1', 'stree_bush_0', 'stree_bush_1', 'rock_pile_0', 'stree_old']),
        round(u * 2) / 2, round(v * 2) / 2)
# a few things on the field's rim and in the gully's corners, clear of the chase
for name, u, v in (('rock_pile_1', 21, 2), ('stree_bush_1', 40, 3), ('broken_spear', 33, 15),
                   ('fallen_flag', 36, 8), ('broken_spear', 25, 6)):
    put(name, u, v)

art = {'id': ID, 'name': '계교 골짜기', 'format': 2, 'size': [W, H], 'origin': [2 * U0, 2 * V0],
       'grass': {'hue': -16, 'sat': 0.8, 'val': 1.04},
       'height': [''.join(r) for r in height],
       'ground': [''.join(r) for r in ground], 'objects': obj}


def register(relative):
    """Add the map to index.json's maps by text, so other sessions' edits of the file stay as they are."""
    path = GFX / 'index.json'
    text = path.read_text()
    if f'"{relative}"' in text:
        return
    end = text.index('\n ],', text.index(' "maps": ['))
    path.write_text(text[:end] + f',\n  "{relative}"' + text[end:])


def hidden(u, v):
    """Whether a person on cell (u, v) has a ridge (lifted LIFT px) over them on screen."""
    for hu in range(u, u + 7):
        for hv in range(v, v + 7):
            dx, dy = 32 * ((hu - u) - (hv - v)), 16 * ((hu - u) + (hv - v))
            if 0 < dy < LIFT + 16 and abs(dx) < 40 and high(hu + .5, hv + .5):
                return True
    return False


def show():
    blocked = {(o[1] // 2, o[2] // 2) for o in obj if o[0] != 'sbridge_v'}
    for v in range(-14, 26):
        line = []
        for u in range(-4, 46):
            g = ground[2 * (v - V0)][2 * (u - U0)]
            c = ('^' if high_cell(u, v) else '#' if (u, v) in blocked else '=' if g == 'b' else
                 '~' if g == '~' else 'o' if hidden(u, v) else g)
            line.append(c)
        print(f'{v:4d} ' + ''.join(line))
    print('     u from -4')


if __name__ == '__main__':
    (GFX / f'maps/{ID}.json').write_text(json.dumps(art, ensure_ascii=False, indent=1) + '\n')
    register(f'maps/{ID}.json')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', ID], cwd=GFX.parents[1], check=True)
    print(ID, art['size'], art['origin'], len(obj), 'objects')
    if '--show' in sys.argv:
        show()
