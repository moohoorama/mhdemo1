#!/usr/bin/env python3
"""Make S_xindu, the inside of 신도성 the day after the siege (chapter 9), walked before the Julu battle.

  python3 ver5/tools/scene_maps/xindu.py

Format 2 (internal/scenemap): small tiles, the playing area from (0, 0) to (W-1, H-1) with a margin of M drawn round
it; u runs down to the right on screen, v down to the left. Ground: d packed earth, i paving, . grass.

Plan: unlike 탁현 and 평원 (walls on the two back sides), 신도성's walls run along its north-west and south-west sides,
as on the B08 battle map where the town is the north-east corner and its gate faces south-west; the gate is in the
near-left wall and the fields lie open behind the town. The main street runs from the gate up to the 관아 at the back,
the cross street carries the tavern and the general store; inside the gate the rubble, a broken spear and a fallen
flag of the siege are still lying.
"""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
W, H, M = 32, 28, 12
MAIN = (13, 16)    # the main street from the gate to the 관아, u from, to
CROSS = (14, 16)   # the cross street, v from, to
GATE = 14          # the gate in the south-west wall (v = H)

g = [['.'] * (W + 2 * M) for _ in range(H + 2 * M)]
obj = []


def paint(u0, v0, u1, v1, ch):
    for v in range(max(-M, v0), min(H + M - 1, v1) + 1):
        for u in range(max(-M, u0), min(W + M - 1, u1) + 1):
            g[v + M][u + M] = ch


def put(name, u, v):
    obj.append([name, u, v])


paint(-M, -M, -2, H + M, 'd')          # fields outside the north-west wall
paint(-M, H + 1, W + M, H + M, 'd')    # and outside the south-west wall
paint(MAIN[0], -M, MAIN[1], H + M, 'd')   # the main street runs on past the 관아 to both ends of the ground
paint(0, CROSS[0], W + M, CROSS[1], 'd')
paint(9, 4, 20, 8, 'i')                # the 관아 yard

put('hall_u', 14, 3), put('banner', 10, 5), put('banner', 19, 5), put('lantern', 12, 8), put('lantern', 17, 8)
put('house_tile_x', 5, 4), put('stree_old', 27, 1)
put('tavern', 6, 12), put('wine_jar', 3, 13), put('bench', 9, 13)
put('house_tile', 22, 12), put('market_stall_jar', 19, 13), put('grain_sack', 25, 13)
put('house_tile_x', 7, 20), put('hut', 3, 21)
put('house_thatch', 34, 22)   # a lone farmhouse a way off from the town's houses, in the open east
put('well', 20, 20), put('water_bucket', 21, 21), put('market_stall', 10, 18), put('stree_willow', 4, 18)
# the siege inside the gate
put('rubble_1', 11, 25), put('rubble_3', 18, 26), put('broken_spear', 17, 23), put('fallen_flag', 10, 22)
put('bloody_cloth', 15, 25)
# trees in the open fields behind the town and outside the walls
for u, v, name in ((6, -7, 'stree_pine_0'), (20, -9, 'stree_broad_1'), (37, 3, 'stree_broad_0'), (39, 17, 'stree_pine_1'),
                   (36, 30, 'stree_broad_1'), (-7, 6, 'stree_pine_1'), (-6, 22, 'stree_broad_0'), (8, 34, 'stree_pine_0'),
                   (26, 35, 'stree_old')):
    put(name, u, v)
put('sgate_open', GATE, H)

art = {'id': 'S_xindu', 'name': '신도성 안', 'format': 2, 'size': [W // 2, H // 2], 'origin': [-M, -M],
       'grass': {'hue': 10, 'sat': 0.72, 'val': 0.96},   # the map's grass tint (docs: 맵별 초원 색조)
       'ground': [''.join(r) for r in g], 'objects': obj,
       'walls': [['swall', [-1, -M], [-1, H]], ['swall', [0, H], [W + M - 1, H]]]}

if __name__ == '__main__':
    json.dump(art, open(GFX / 'maps/S_xindu.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if 'maps/S_xindu.json' not in index['maps']:
        index['maps'].append('maps/S_xindu.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', 'S_xindu'], cwd=GFX.parents[1], check=True)
    print('S_xindu', art['size'], len(obj), 'objects')
