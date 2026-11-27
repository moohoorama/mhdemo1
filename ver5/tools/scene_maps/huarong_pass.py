#!/usr/bin/env python3
"""Make S_huarong_pass: 화용도 for the title reel — a muddy road squeezed between two ridges, running from the south-east
(+u) to the north-west (-u), where 조조's beaten men come up the pass and find 관우 and his soldiers waiting on a
flat where the valley opens.

  python3 ver5/tools/scene_maps/huarong_pass.py          write the map, register it, format it
  python3 ver5/tools/scene_maps/huarong_pass.py --show   also print the cells: ^ high, # blocked, o hidden by a ridge

Format 2, outdoors, game cells (u grows toward the lower right of the screen, v toward the lower left):

  road      v 5-8 along the whole picture, dirt with mud
  pass      u 16-40: the north-east ridge (height 2, its face toward the road) stands right at the road (v <= 4); the
            south-west ridge stands back at v >= 14 from u 20, so its lifted top lies below the road on screen and
            hides no one on it — the strip v 9-13 under it is scree nobody walks
  flat      u 2-15, v 0-13: where the valley opens; a ridge face on its north-east (v <= -1) and north-west (u <= 1,
            but for the road), the south-west ridge far back (v >= 19)
  north-west of the flat the road runs on between the ridges (v <= 4, v >= 14) and the valley widens past u -12;
  south-east of the pass (u >= 41) the valley widens the same way.

The picture runs 30 cells past the playing area on every side so its edges stay off a zoom-1 screen. Pines and dead
trees stand on the ridge tops only where their lifted pictures cannot cover the road or the flat.
"""
import json
import math
import random
import subprocess
import sys
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
ID = 'S_huarong_pass'
W, H = 44, 14          # the playing area: the flat and the pass
M = 30
U0, U1 = -M, W + M
V0, V1 = -M, H + M
ROAD = (5, 8)
FLAT = (2, 15, 0, 13)  # u from, to, v from, to (inclusive)
PASS = (16, 40)
LIFT = 146             # height 2's lift in pixels

rnd = random.Random(208)
cols, rows = 2 * (U1 - U0), 2 * (V1 - V0)
ground = [['.'] * cols for _ in range(rows)]
height = [['0'] * cols for _ in range(rows)]


def wob(x, seed):
    """0, .5 or 1 cells, so a ridge's edge steps in and out by small tiles instead of running ruler-straight."""
    return .5 * round(1 + .9 * math.sin(x * .21 + seed) + .3 * math.sin(x * .53 + 2 * seed))


def high(u, v):
    """Whether the small tile centred on game point (u, v) is ridge. The edges only step away from where people walk."""
    if u < -12 or u > PASS[1] + 1:                  # the valley widens beyond both ends
        far = min(1., (abs(u - (-12 if u < 0 else PASS[1] + 1))) / 6)
        ne, sw = 5 - 7 * far, 14 + 6 * far
        return v < ne - wob(u, 1) or v >= sw + wob(u, 2)
    if u < FLAT[0]:                                  # north-west of the flat: the road between the ridges
        return v < ROAD[0] - wob(u, 1) or v >= 14 + wob(u, 2)
    if u < FLAT[1] + 1:                               # the flat
        return v < FLAT[2] - wob(u, 3) or v >= 19 + wob(u, 4)
    # the pass
    return v < ROAD[0] - wob(u, 1) or v >= (14 if u >= 20 else 19) + wob(u, 2)


for R in range(rows):
    for C in range(cols):
        u, v = (C + .5) / 2 + U0, (R + .5) / 2 + V0
        n = math.sin(u * .37 + v * .21) + math.sin(u * .11 - v * .33 + 1.3) + rnd.uniform(-.4, .4)
        if high(u, v):
            height[R][C] = '2'
            ground[R][C] = 'd' if n > .7 else '.'   # bare rock and dry grass on the tops
            continue
        if ROAD[0] <= v < ROAD[1] + 1:
            ground[R][C] = 'd'                      # the road
        elif FLAT[0] <= u <= FLAT[1] + 1 and FLAT[2] <= v < FLAT[3] + 1:
            ground[R][C] = 'd' if n > .9 else '.'   # the trodden flat
        elif ROAD[1] < v < ROAD[1] + 3:
            ground[R][C] = 'd' if n > .3 else '.'   # mud thrown off the road
        else:
            ground[R][C] = 'd' if n > 1.1 else '.'

obj = []


def put(name, u, v):
    """An object on game point (u, v): u and v may end in .5 to sit on the second small tile."""
    obj.append([name, int(math.floor(2 * u)), int(math.floor(2 * v))])


def high_cell(u, v):
    return high(u + .25, v + .25) and high(u + .75, v + .75) and high(u + .25, v + .75) and high(u + .75, v + .25)


def walk_cell(u, v):
    return FLAT[0] <= u <= FLAT[1] and FLAT[2] <= v <= FLAT[3] or ROAD[0] <= v <= ROAD[1]


def clear_of_walk(u, v, tall, wide):
    """A thing `tall` pixels high standing at cell (u, v) (lifted when on a ridge) covers no cell people walk."""
    return not any(walk_cell(pu, pv) and covers(u, v, tall, wide, pu, pv)
                   for pu in range(u - 12, u + 3) for pv in range(v - 12, v + 3))


def covers(u, v, tall, wide, pu, pv):
    """Whether a thing `tall` x `wide` px on cell (u, v) is drawn over a person (60 px) on cell (pu, pv)."""
    lift = LIFT if high_cell(u, v) else 0
    dx, dy = 32 * ((u - pu) - (v - pv)), 16 * ((u - pu) + (v - pv))
    return (u, v) != (pu, pv) and abs(dx) < wide / 2 + 20 and dy - lift > -60 and dy - lift - tall < 0 and (dy > 0 or lift)


# ridge tops: pines and rocks, sparse; the south-west ridge only well back from the road
for v in range(V0 + 1, V1 - 1):
    for u in range(U0 + 1, U1 - 1):
        if not high_cell(u, v):
            continue
        r = rnd.random()
        if r < .07:
            name, tall, wide = rnd.choice(['stree_pine_0', 'stree_pine_1']), 130, 92
        elif r < .085:
            name, tall, wide = 'stree_dead', 86, 36
        elif r < .11:
            name, tall, wide = rnd.choice(['rock_pile_0', 'rock_pile_1', 'rock_pile_2']), 30, 50
        else:
            continue
        if clear_of_walk(u, v, tall, wide):
            put(name, u + rnd.choice([0, .5]), v + rnd.choice([0, .5]))

# the scree under the south-west ridge and the flat's rough edge: rocks, dead trees, dry bushes
for v in range(ROAD[1] + 1, 20):
    for u in range(U0 + 1, U1 - 1):
        if high_cell(u, v) or walk_cell(u, v):
            continue
        r = rnd.random()
        if r < .05:
            name, tall, wide = rnd.choice(['rock_pile_0', 'rock_pile_1', 'rock_pile_2']), 30, 50
        elif r < .075:
            name, tall, wide = 'stree_dead', 86, 36
        elif r < .09:
            name, tall, wide = rnd.choice(['stree_bush_0', 'stree_bush_1']), 40, 50
        else:
            continue
        if clear_of_walk(u, v, tall, wide):
            put(name, u + .5, v + .5)

# dead trees and rocks in the open valley beyond both ends, off the road
for _ in range(60):
    u = rnd.choice([rnd.uniform(U0 + 1, -4), rnd.uniform(PASS[1] + 3, U1 - 1)])
    v = rnd.uniform(-4, 22)
    if ROAD[0] - 1.5 <= v <= ROAD[1] + 1.5 or high_cell(int(u), int(v)):
        continue
    put(rnd.choice(['stree_dead', 'stree_dead', 'rock_pile_0', 'rock_pile_1', 'stree_bush_1', 'stree_old']),
        round(u * 2) / 2, round(v * 2) / 2)

# along the edges of the flat: a dead tree or two by the ridge foot, a broken flag of 조조's men by the road
for name, u, v in (('stree_dead', 3, 0), ('rock_pile_1', 7, 0), ('stree_dead', 14, 0), ('rock_pile_2', 2, 13),
                   ('stree_dead', 12, 13), ('broken_spear', 38, 9), ('fallen_flag', 33, 9.5), ('broken_spear', 27, 9)):
    put(name, u, v)

art = {'id': ID, 'name': '화용도', 'format': 2, 'size': [W, H], 'origin': [2 * U0, 2 * V0],
       'grass': {'hue': -24, 'sat': 0.7, 'val': 0.92},
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
    """Whether a person on cell (u, v) has a ridge top (lifted LIFT px) over their feet on screen."""
    for hu in range(u, u + 7):
        for hv in range(v, v + 7):
            dx, dy = 32 * ((hu - u) - (hv - v)), 16 * ((hu - u) + (hv - v))
            if 0 < dy < LIFT + 16 and abs(dx) < 40 and high(hu + .5, hv + .5):
                return True
    return False


def show():
    blocked = {(o[1] // 2, o[2] // 2) for o in obj}
    for v in range(-8, 24):
        line = []
        for u in range(-16, 56):
            if high_cell(u, v):
                c = '^'
            elif (u, v) in blocked:
                c = '#'
            elif hidden(u, v):
                c = 'o'
            else:
                c = ground[2 * (v - V0)][2 * (u - U0)]
            line.append(c)
        print(f'{v:4d} ' + ''.join(line))
    print('     u from -16')


if __name__ == '__main__':
    (GFX / f'maps/{ID}.json').write_text(json.dumps(art, ensure_ascii=False, indent=1) + '\n')
    register(f'maps/{ID}.json')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', ID], cwd=GFX.parents[1], check=True)
    print(ID, art['size'], art['origin'], len(obj), 'objects')
    if '--show' in sys.argv:
        show()
