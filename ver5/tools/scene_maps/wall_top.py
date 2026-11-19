#!/usr/bin/env python3
"""Make S_wall_top and S_wall_top_flip, a city wall's walk by day with the field below it: the garrison on the wall,
a crowd on the ground outside (양양·강릉 gate scenes, chapters 25-26).

  python3 ver5/tools/scene_maps/wall_top.py

Format 2 (internal/scenemap), the wall top of S_changshe_wall (changshe_wall.py: height 4, faces rdwall, inner parapet
and battlements rrim, paved Q, 14 x 5 cells running along u) turned to daylight: no 황건 camp, no wounded, banners and a
command flag along the battlements, stores and racks on the inner edge, roofs of the town under the wall. Outside, in
the upper right (-v), a trodden gathering ground and a road running away from the wall, a few trees far back, so a
crowd can stand there clear of the battlements that hide anyone closer than about 4 cells.

S_wall_top_flip is the same map turned left to right: swapping u and v mirrors the screen on a 2:1 iso grid, so the
ground and height grids are transposed, every object goes to (v, u) with its pixel offset's x negated, and the
houses swap their door side (house_tile <-> house_tile_x). The wall then runs along v with its face at the lower
right and the outside in the upper left (-u). A scene staged on one is turned for the other by the same swap: u <-> v
in every cell, SE <-> SW and NE <-> NW in every Face (wall_top_scenes.py does it).
"""
import json
import math
import random
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
W, H = 14, 5               # game cells of walk
MU, BACK, FRONT = 16, 40, 12
rnd = random.Random(7)
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

# outside: grass with a road running away from the wall and a worn gathering ground where the crowd stands
for v in range(-BACK, 0):
    for u in range(-MU, 2 * W + MU):
        cu = 14 + 3 * math.sin(v * .12)
        road = abs(u - cu) < 3.2
        ring = -26 < v < -5 and -2 < u < 2 * W + 3 and \
            math.sin(u * .31 + v * .23) + math.sin(u * .13 - v * .29) + rnd.random() * .8 > -1.1 + abs(u - 14) * .03
        cell(u, v, 'd' if road or ring else '.')
for v in range(2 * H, 2 * H + FRONT):
    for u in range(-MU, 2 * W + MU):
        cell(u, v, 'd')

# on the wall: banners and a command flag by the battlements, stores and racks on the inner edge
for u in (1.5, 4.5, 9.5, 12.5):
    put('banner', u, -.3)
put('command_flag', 7.5, -.45)
for u, v in ((.5, 4.4), (1.5, 4.45), (11.5, 4.4), (13.5, 4.4)):
    put('grain_sack', u, v)
put('weapon_rack', 3, 4.3), put('weapon_rack', 10.5, 4.3)
put('water_jar', 6.5, 4.4), put('water_jar', 7.7, 4.4)

# outside: trees far back and to the sides, a milestone on the road, rubble and rocks at the edge of the gathering
for u, v in ((-6, -16), (-3, -21), (3, -24), (-8, -27), (24, -19), (30, -24), (22, -28), (33, -17), (-1, -32),
             (12, -30), (27, -33)):
    put(rnd.choice(['stree_broad_0', 'stree_broad_1', 'stree_pine_0', 'stree_pine_1']), u, v)
for u, v in ((-4, -9), (-5, -13), (18, -8.5), (17, -14), (-2, -18)):
    put(rnd.choice(['stree_bush_0', 'stree_bush_1']), u, v)
put('milestone', 11.2, -3.2)
put('rock_pile_0', -3, -6.5), put('rock_pile_1', 19.5, -12)

# inside: roofs of the town under the wall
for u, v in ((-1, 8.5), (3, 9), (7.5, 8.5), (12, 9), (16, 8.5)):
    put(rnd.choice(['house_tile', 'house_tile_x']), u, v)

BASE = {'id': 'S_wall_top', 'name': '성벽 위', 'format': 2, 'size': [W, H], 'origin': [-MU, -BACK],
        'ground': [''.join(r) for r in ground], 'height': [''.join(r) for r in height], 'objects': obj}


def flipped(art):
    """The map turned left to right: u and v swapped, the pixel offsets' x negated, the houses' doors swapped."""
    swap = {'house_tile': 'house_tile_x', 'house_tile_x': 'house_tile', 'house_thatch': 'house_thatch_x',
            'house_thatch_x': 'house_thatch'}
    out = []
    for o in art['objects']:
        n = [swap.get(o[0], o[0]), o[2], o[1]]
        if len(o) == 5:
            n += [-o[3], o[4]]
        out.append(n)
    g = art['ground']
    h = art['height']
    t = lambda rowsv: [''.join(rowsv[r][c] for r in range(len(rowsv))) for c in range(len(rowsv[0]))]
    return {'id': 'S_wall_top_flip', 'name': '성벽 위(좌우 반전)', 'format': 2, 'size': [art['size'][1], art['size'][0]],
            'origin': [art['origin'][1], art['origin'][0]], 'ground': t(g), 'height': t(h), 'objects': out}


if __name__ == '__main__':
    index_path = GFX / 'index.json'
    text = index_path.read_text()
    for art in (BASE, flipped(BASE)):
        json.dump(art, open(GFX / f"maps/{art['id']}.json", 'w'), ensure_ascii=False, indent=1)
        entry = f"maps/{art['id']}.json"
        if f'"{entry}"' not in text:
            anchor = '  "maps/S_changshe_wall.json",\n'
            assert anchor in text
            text = text.replace(anchor, anchor + f'  "{entry}",\n')
            index_path.write_text(text)
        subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', art['id']], cwd=GFX.parents[1], check=True)
        print(art['id'], art['size'], len(art['objects']), 'objects')
