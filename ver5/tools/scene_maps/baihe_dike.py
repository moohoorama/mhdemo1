#!/usr/bin/env python3
"""Make S_baihe_dike: the dike works on the 백하 above 신야 (chapter 25), where 관우 and 미축 talk while the army piles
earth and sacks into the gorge, before and after the flashbacks to 하비.

  python3 ver5/tools/scene_maps/baihe_dike.py

Format 2, 48 x 44 cells, the battle map's own size (S_bridge, which the scene used before, was 12 x 10): the ford of the
show's second shot lies at v 30 and below, the river running on shallow over sand (scene.json flood kind 'S') where
the army wades over. It is the place the flood show of
internal/gui/uidemo_suigong.go (assets/graphics/effects/flood/suigong, tools/floodsim.py) opens up the dam of, laid out
by the same cells: the river six cells wide down the v axis (u 12-17), the gorge's cliffs above row 10 (u 9-11 and
u 18-22, scene.json height 7: the battle map's own 26 px hill, the lift the show stands 관우 on) and the dam of
earth and sacks across the river at v 8; 관우 and 미축 stand on the right cliff over the dam at (19, 7) and (20, 7),
the cells the show puts them on. The ground round the gorge is the battle map's (tools/mapgen/layouts/hero/
map25flood.py): the hills' tops are bare earth all over, grass beyond, and the pine groves (the battle's own
forest_N pictures) stand on the same cells as there.
The work is a few sacks and carts at the dam and its road and two braziers on the cliff; nothing stands on the
ground that the battle's map leaves bare.
"""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
ID = 'S_baihe_dike'
W, H = 48, 44
MU, MV = 10, 8
RIVER, WIDTH, DAM, CLIFF_END, FORD = 12, 6, 8, 10, 30

cols, rows = 2 * (W + 2 * MU), 2 * (H + 2 * MV)
ground = [['.'] * cols for _ in range(rows)]
height = [['0'] * cols for _ in range(rows)]


def cell(U, V):
    """Where small tile (U, V) lies in game cells, as the cell index that contains it (cell k is [k, k + 1))."""
    return (U - .5) / 2 - MU + .5, (V - .5) / 2 - MV + .5


for V in range(rows):
    for U in range(cols):
        u, v = cell(U, V)
        in_river = RIVER <= u < RIVER + WIDTH
        if in_river:
            ground[V][U] = 'd' if DAM <= v < DAM + 1 else 'S' if v >= FORD else '~'   # the ford: shallow water over sand
        elif v < CLIFF_END and (RIVER - 3 <= u < RIVER or RIVER + WIDTH <= u < RIVER + WIDTH + 5):
            height[V][U] = '7'
            ground[V][U] = 'd'  # a hill's top is bare earth (the battle map's 'h')
        elif RIVER + WIDTH <= u < RIVER + WIDTH + 4 and DAM - 1 <= v < DAM + 4 and v >= CLIFF_END - 1:
            ground[V][U] = 'd'  # the road up to the dam's end
        elif RIVER - 3 <= u < RIVER and DAM <= v < DAM + 2:
            ground[V][U] = 'd'

obj = []


def put(name, u, v, *shift):
    obj.append([name, int(round(2 * u)) + 1, int(round(2 * v)) + 1, *shift])


# the dam: a row of sacks along the top of its earth
for i in range(12):
    put('grain_sack', RIVER + .25 + i * .5, DAM + .75)
for u, v in ((RIVER - 2.5, DAM - 1.0), (RIVER + WIDTH + 1.5, DAM + 2.4), (RIVER + WIDTH + 3, DAM + 1.2)):
    put('straw_mat', u, v)

# the road and the heaps of sacks beside it
for name, u, v in (('grain_cart', 21.4, 12.5), ('grain_cart_v', 23.8, 10.8), ('grain_sack', 20.2, 11.2), ('grain_sack', 20.8, 11.6),
                   ('grain_sack', 22.6, 12.2), ('grain_sack', 22.2, 11.6), ('brazier', 18.6, 8.6), ('brazier', 19.0, 10.4)):
    put(name, u, v)


def tree_cell(u, v):
    """The battle map's trees (map25flood.py): the same cells, as far as this board reaches."""
    west = u < RIVER - 1 and v > 11
    east = 19 <= u <= 37 and 12 < v < 39
    return ((west or east) and (u * 11 + v * 7) % 17 < 2 and (v < 21 or v > 37 or u > 5)
            and not (2 <= u <= 6 and 31 <= v <= 36) and not (u < 12 and 10 <= v <= 20))


for v in range(H):
    for u in range(W):
        if tree_cell(u, v) and not (v < CLIFF_END and RIVER - 3 <= u < RIVER + WIDTH + 5):
            put(f'forest_{(u * 5 + v * 3) % 4}', u + .5, v + .5)

art = {'id': ID, 'name': '백하 둑', 'format': 2, 'size': [W, H], 'origin': [-2 * MU, -2 * MV],
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
