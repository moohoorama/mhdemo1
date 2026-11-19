#!/usr/bin/env python3
"""Make S_vil_peace, the 융중 the 23rd and 24th chapters walk (explore23, explore23b, explore24 and the scenes
c23-visit1/2, c24-cottage3): a hermit's thatched cottage inside a low fence, seen in autumn, snow and spring.

  python3 ver5/tools/scene_maps/longzhong.py

Format 2, small tiles, origin -24, -24 (the cottage stays where the scenes were staged, u 1..12, v 1..8). The way in
from 신야 (the exit at 24, 8) is a packed-earth road that bends west to the cottage door; the corridor u 8..24,
v 0..9 stays free of anything but the road (the scenes' guests walk in along it). Round the cottage: pines and
broad trees thickening to a wood on the north and west, broad trees and bushes at the fence, small plots of furrowed
field south of the road with a well. One map serves three seasons, so the plots are bare furrows and sprouts
(no ripe grain under snow) and the grass tint comes from the scene or area's Grass value.
"""
import json
import random
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
OX = OY = -24
W, H = 58, 56

g = [['.'] * W for _ in range(H)]
obj = [["sfence_12", 1, 1], ["sfence_05", 2, 1], ["sfence_10", 1, 2], ["sfence_05", 3, 1], ["sfence_10", 1, 3],
       ["sfence_05", 4, 1], ["sfence_10", 1, 4], ["sfence_09", 5, 1], ["sfence_06", 1, 5], ["sfence_10", 5, 2],
       ["sfence_01", 2, 5], ["sfence_10", 5, 3], ["tavern", 7, 1, 0, 2], ["sfence_10", 5, 4],
       ["notice_board", 3, 6, -10, 0], ["sfence_02", 5, 5], ["stree_pine_1", -1, 0, -2, 6], ["stree_broad_0", 0, 7],
       ["stree_bush_1", 11, 1, 2, -6], ["house_thatch", 3, 2, 8, -2], ["house_thatch_x", 2, 4, -2, -6]]
keep = []   # (u0, v0, u1, v1) boxes trees stay out of


def paint(u, v, ch):
    if 0 <= u - OX < W and 0 <= v - OY < H:
        g[v - OY][u - OX] = ch


def road(points, width=2):
    """Packed earth along the polyline, width tiles wide."""
    for (u0, v0), (u1, v1) in zip(points, points[1:]):
        n = max(abs(u1 - u0), abs(v1 - v0))
        for i in range(n + 1):
            u, v = round(u0 + (u1 - u0) * i / n), round(v0 + (v1 - v0) * i / n)
            for du in range(width):
                for dv in range(width):
                    paint(u + du, v + dv, 'd')


# the old patch before the cottage door, kept
for u, v in [(3, 4), (3, 5), (3, 6), (3, 7), (3, 8), (4, 6), (4, 7), (4, 8), (5, 7), (5, 8), (6, 7), (6, 8), (6, 9), (7, 8),
             (7, 9), (7, 10), (8, 8), (8, 9)]:
    paint(u, v, 'd')

# the road from 신야: in at the exit (24, 8), bending west along the corridor to the door and the notice board
ROAD = [(31, 11), (27, 9), (24, 8), (20, 7), (16, 7), (12, 8), (9, 8), (6, 8)]
road(ROAD)
keep += [(8, 0, 25, 10), (-2, -1, 14, 11)]

# small furrowed plots south of the road, a well and its bucket, a stone pile
for u, v, name in [(2, 13, 'field_bare'), (6, 13, 'field_sprout'), (10, 14, 'field_bare'), (14, 17, 'field_bare'),
                   (18, 17, 'field_sprout'), (22, 15, 'field_bare'), (2, 17, 'field_bare'), (6, 17, 'field_bare')]:
    obj.append([name, u, v])
    keep.append((u - 3, v - 3, u + 3, v + 3))
obj += [['well', 13, 12], ['water_bucket', 14, 13], ['rock_pile_1', 24, 19], ['grain_sack', 12, 13]]
keep += [(11, 10, 16, 14), (13, 9, 19, 16)]

# broad trees and bushes at the fence, bushes along the road
obj += [['stree_broad_1', -3, 3], ['stree_old', -2, 10], ['stree_broad_0', 10, 10]]
for u, v in [(14, 10), (19, 10), (22, 10), (18, 0), (10, 11), (-4, 6), (8, 12), (26, 12)]:
    obj.append(['stree_bush_0' if (u + v) % 2 else 'stree_bush_1', u, v])

# the wood: pines mostly, broad trees and old pines mixed, thick to the north and west, loose to the south and east
rng = random.Random(7)


def near_keep(u, v, pad=2):
    return any(a - pad <= u <= c + pad and b - pad <= v <= d + pad for a, b, c, d in keep)


def near_road(u, v):
    return any(abs(u - a) <= 3 and abs(v - b) <= 3 for a, b in [(p[0] + t, p[1] + t) for p in ROAD for t in (0, 1)])


placed = []
wood_from = len(obj)
for _ in range(2600):
    u, v = rng.randint(-23, 33), rng.randint(-23, 31)
    north, west = v < -3, u < -5
    p = 0.2 if (north or west) else 0.04 if (v > 22 or u > 30) else 0.0
    if v < -3 and u < -5:
        p = 0.35
    if rng.random() > p or near_keep(u, v) or near_road(u, v):
        continue
    if any(abs(u - a) < 4 and abs(v - b) < 4 for a, b in placed):
        continue
    placed.append((u, v))
    name = rng.choice(['stree_pine_0', 'stree_pine_1', 'stree_pine_0', 'stree_pine_1', 'stree_broad_0', 'stree_broad_1',
                       'stree_old'] if not (north and west) else
                      ['stree_pine_0', 'stree_pine_1', 'stree_broad_0', 'stree_pine_1', 'stree_broad_1'])
    obj.append([name, u, v])
for u, v in placed[::5]:
    if not near_keep(u + 1, v + 1, 1):
        obj.append(['stree_bush_1', u + 1, v + 1])

# bamboo, the hermit's own grove: some of the wood on the north and west near the cottage turns to it, a thick
# stand stands right behind and beside the cottage, slim stems line the road and dwarf bamboo marks its end
brng = random.Random(23)
BAMBOO = ['stree_bamboo_0', 'stree_bamboo_1']
for o in obj[wood_from:]:
    u, v = o[1], o[2]
    if o[0].startswith(('stree_pine', 'stree_broad')) and u < 14 and v < 13 and (v < -3 or u < -5) and brng.random() < .45:
        o[0] = brng.choice(BAMBOO)
GROVES = [(2, -2), (5, -3), (8, -2), (11, -3), (-5, 1), (-6, 4), (-5, 8), (-3, -4), (14, -5)]
SLIM = [(17, 11), (22, 13), (-2, 13)]
SASA = [(9, 10), (4, 10), (20, 11)]
fixed = obj[:wood_from]
wood = [o for o in obj[wood_from:] if not any(abs(o[1] - u) < 3 and abs(o[2] - v) < 3 for u, v in GROVES + SLIM + SASA)]
obj = fixed + wood
obj += [[BAMBOO[i % 2], u, v] for i, (u, v) in enumerate(GROVES)]
obj += [['stree_bamboo_thin', u, v] for u, v in SLIM] + [['stree_sasa', u, v] for u, v in SASA]

art = {'id': 'S_vil_peace', 'name': '마을(울타리)', 'format': 2, 'size': [6, 5], 'origin': [OX, OY],
       'ground': [''.join(r) for r in g], 'objects': obj}

if __name__ == '__main__':
    json.dump(art, open(GFX / 'maps/S_vil_peace.json', 'w'), ensure_ascii=False, indent=1)
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', 'S_vil_peace'], cwd=GFX.parents[1], check=True)
    print('S_vil_peace', len(obj), 'objects')
