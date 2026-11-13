#!/usr/bin/env python3
"""Make S_xiapi_flood, the outskirts of 하비성 drowned by 조조's dykes, seen from a height (chapter 16, 하비; also for
later talk of the flooding).

  python3 ver5/tools/scene_maps/xiapi_flood.py

Format 2 (internal/scenemap): small tiles, the playing area from (0, 0) to (W-1, H-1) with a margin of M drawn round
it; u runs down to the right on screen, v down to the left. Ground: . grass, d soil, F grass and R road under the
flood (scene.json "flood": muddy water over the ground, what stands in it sunk to its waterline). Height '2' is the
tall cliff of the 영천 cliff scenes.

Plan: a bare cliff top runs along the south-west (v >= LIP) and drops on its north-east side, away from the camera,
so only its lip shows; the scene's camera puts it in the lower left of the view. The party stands on the lip and
looks north-east over the water. Everything
below is the flooded edge of the city: the market by the south gate (stalls up to their awnings), rows of tiled and
thatched houses up to their eaves, the road under water, willows, fence tops and a milestone. The wall itself is
out of the picture.
"""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
ID = 'S_xiapi_flood'
W, H, M = 28, 24, 20
LIP = 16           # the cliff top: small rows v >= LIP

g = [['F'] * (W + 2 * M) for _ in range(H + 2 * M)]
h = [['0'] * (W + 2 * M) for _ in range(H + 2 * M)]
obj = []


def paint(u0, v0, u1, v1, ch, grid=g):
    for v in range(max(-M, v0), min(H + M - 1, v1) + 1):
        for u in range(max(-M, u0), min(W + M - 1, u1) + 1):
            grid[v + M][u + M] = ch


def put(name, u, v):
    obj.append([name, u, v])


# the bare cliff top along the south-west, a few pines and bushes clinging to it
paint(-M, LIP, W + M, H + M, 'd')
paint(-M, LIP, W + M, H + M, '2', h)
for u, v, name in ((1, 19, 'stree_pine_0'), (4, 25, 'stree_pine_1'), (19, 21, 'stree_bush_0'), (-4, 22, 'stree_bush_1'),
                   (24, 26, 'stree_pine_0')):
    put(name, u, v)
put('banner', 17, 17)

# the road from the south gate (off the top of the picture) under water, the milestone at its bend
paint(10, -M, 13, 6, 'R')
paint(4, 4, 13, 7, 'R')
put('milestone', 9, 8)

# the market by the gate: stalls up to their awnings along the road, a cart left among them
for u, v, name in ((8, 1, 'market_stall_cloth'), (8, -3, 'market_stall'), (15, -1, 'market_stall_jar'), (15, 3, 'market_stall'),
                   (20, 7, 'market_stall_jar')):
    put(name, u, v)
put('grain_cart', 17, 6)
# rows of tiled houses toward the gate, thatched ones further out, up to their eaves
for u, v in ((3, -6), (17, -7), (22, -3), (3, -1), (27, 1), (-3, -4), (14, -14), (30, -9), (-6, -10)):
    put('house_tile', u, v)
for u, v in ((20, 2), (-2, 2), (26, -6), (8, -12), (24, -12)):
    put('house_tile_x', u, v)
for u, v, name in ((24, 9, 'house_thatch'), (-6, 8, 'house_thatch_x'), (31, 7, 'house_thatch'), (10, -8, 'hut'),
                   (-12, -4, 'house_thatch'), (0, -16, 'house_thatch_x')):
    put(name, u, v)
# willows along the road and the fields, fence tops of a vegetable plot
for u, v, name in ((18, 10, 'stree_willow'), (2, 5, 'stree_willow'), (32, 12, 'stree_broad_0'), (-9, 1, 'stree_willow'),
                   (32, -3, 'stree_broad_1'), (20, -16, 'stree_willow'), (36, 3, 'stree_willow')):
    put(name, u, v)
for u in range(24, 29):
    put('sfence_05', u, 12)

art = {'id': ID, 'name': '물에 잠긴 하비성', 'format': 2, 'size': [W // 2, H // 2], 'origin': [-M, -M],
       'ground': [''.join(r) for r in g], 'height': [''.join(r) for r in h], 'objects': obj}

if __name__ == '__main__':
    json.dump(art, open(GFX / f'maps/{ID}.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if f'maps/{ID}.json' not in index['maps']:
        index['maps'].append(f'maps/{ID}.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', ID], cwd=GFX.parents[1], check=True)
    print(ID, art['size'], len(obj), 'objects')
