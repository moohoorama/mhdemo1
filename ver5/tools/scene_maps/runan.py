#!/usr/bin/env python3
"""Make S_runan, the small walled town of 여남 (chapter 21): 신도성's plan turned half a turn.

  python3 ver5/tools/scene_maps/runan.py

The positions of xindu.py mirrored through the middle of the ground (u → W-1-u, v → H-1-v) and then written out
here in 여남's own coordinates; the buildings keep facing the viewer, so it reads as another town built from the
same houses. The walls stand on the north-east (back right, with the north gate) and south-east (near right)
sides and level grass lies all round. What the half turn left wrong was put right by hand: the 관아 stands on the
north-east side of its paved yard, facing it, with the main street coming down past its side into the yard; the
willow no longer hides a house; the tiled house that touched the wall stands by the well.
"""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
W, H, M = 32, 28, 12
g = [['.'] * (W + 2 * M) for _ in range(H + 2 * M)]
obj = []


def paint(u0, v0, u1, v1, ch):
    for v in range(max(-M, v0), min(H + M - 1, v1) + 1):
        for u in range(max(-M, u0), min(W + M - 1, u1) + 1):
            g[v + M][u + M] = ch


def put(name, u, v):
    obj.append([name, u, v])


paint(15, -M, 18, 21, 'd')             # the main street: in by the north gate, down into the 관아 yard
paint(15, 27, 18, H + M, 'd')          # and on from the yard to the south
paint(-M, 11, W - 1, 13, 'd')          # the cross street
paint(11, 22, 26, 26, 'i')             # the 관아 yard

put('hall_u', 21, 20), put('banner', 17, 22), put('banner', 26, 22), put('lantern', 14, 26), put('lantern', 20, 26)
put('house_tile_x', 6, 8)              # by the well
put('stree_old', 4, 26)
put('tavern', 25, 15), put('wine_jar', 28, 14), put('bench', 22, 14)
put('house_tile', 9, 15), put('market_stall_jar', 12, 14), put('grain_sack', 6, 14)
put('house_tile_x', 24, 7), put('hut', 28, 6)
put('house_thatch', -3, 5)
put('well', 11, 7), put('water_bucket', 10, 6), put('market_stall', 21, 9), put('stree_willow', 22, 2)
for u, v, name in ((26, 36, 'stree_pine_0'), (11, 36, 'stree_broad_1'), (-6, 24, 'stree_broad_0'), (-8, 10, 'stree_pine_1'),
                   (-5, -3, 'stree_broad_1'), (38, 21, 'stree_pine_1'), (37, 5, 'stree_broad_0'), (23, -7, 'stree_pine_0'),
                   (5, -8, 'stree_old')):
    put(name, u, v)
put('sgate_open', 17, -1)              # the north gate, in the back-right wall

art = {'id': 'S_runan', 'name': '여남성 안', 'format': 2, 'size': [W // 2, H // 2], 'origin': [-M, -M],
       'grass': {'hue': -6, 'sat': 0.8, 'val': 0.94},
       'ground': [''.join(r) for r in g], 'objects': obj,
       'walls': [['swall', [-M, -1], [W, -1]], ['swall', [W, 0], [W, H + M - 1]]]}

if __name__ == '__main__':
    json.dump(art, open(GFX / 'maps/S_runan.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if 'maps/S_runan.json' not in index['maps']:
        index['maps'].append('maps/S_runan.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', 'S_runan'], cwd=GFX.parents[1], check=True)
    print('S_runan', art['size'], len(obj), 'objects')
