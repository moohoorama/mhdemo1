#!/usr/bin/env python3
"""Make S_xinye_town, 신야 as chapter 25's night of fire shows it: the 소패 layout (xiaopei.py — the town the explore
chapters walk as 신야성) walled round on all four sides, the way xiapi_town.py walls 하비, with a gate in every wall.

  python3 ver5/tools/scene_maps/xinye_town.py

Format 2 (internal/scenemap), 14×12 game cells like S_xiaopei, with these changes:
  - the two near walls (south v = NEAR_V, east u = NEAR_U) a few tiles past the playing area, so the town reads as
    a walled whole; what stood beyond them in xiaopei.py is left out and the ground there is trodden fields.
  - four gates: the north one is xiaopei's (the lane through it runs on south), the west one opens on a street
    north of the 관아 yard, the south one ends the lane, the east one ends the main street. The west, north and
    south gates are shut (the scene lights them); the east gate stands open.
"""
import json
import subprocess

import xiaopei as base

GFX = base.GFX
ID = 'S_xinye_town'
W, H, M = base.W, base.H, base.M
NEAR_V, NEAR_U = H + 3, W + 3
WEST_GATE_V = 4
SOUTH_GATE_U = base.GATE
EAST_GATE_V = 11

g = [list(r) for r in base.g]


def paint(u0, v0, u1, v1, ch):
    for v in range(max(-M, v0), min(H + M - 1, v1) + 1):
        for u in range(max(-M, u0), min(W + M - 1, u1) + 1):
            g[v + M][u + M] = ch


obj = []
for o in base.obj:
    name, u, v = o[0], o[1], o[2]
    if name == 'stree_old' and (u, v) == (1, 3):
        continue                                                     # on the west gate's street
    if (u >= NEAR_U - 1 or v >= NEAR_V - 1) and u >= 0 and v >= 0:
        continue                                                     # outside the near walls
    obj.append(o)

paint(-M, NEAR_V, W + M, H + M, 'd')                                 # fields outside the near walls
paint(NEAR_U, -M, W + M, H + M, 'd')
paint(-M, WEST_GATE_V - 1, base.LANE[0], WEST_GATE_V, 'd')           # the street from the west gate to the lane
obj.append(['sgate_closed', -1, WEST_GATE_V])
obj.append(['sgate_closed', SOUTH_GATE_U, NEAR_V])
obj.append(['sgate_open', NEAR_U, EAST_GATE_V])
obj = [o for o in obj if o[0] != 'sgate_open' or o[1] == NEAR_U]      # xiaopei's north gate becomes a shut one
obj.append(['sgate_closed', base.GATE, -1])

walls = [['swall', [-1, -1], [NEAR_U, -1]], ['swall', [-1, 0], [-1, NEAR_V]],
         ['swall', [0, NEAR_V], [NEAR_U, NEAR_V]], ['swall', [NEAR_U, 0], [NEAR_U, NEAR_V - 1]]]

art = {'id': ID, 'name': '신야', 'format': 2, 'size': [W // 2, H // 2], 'origin': [-M, -M],
       'ground': [''.join(r) for r in g], 'objects': obj, 'walls': walls}

if __name__ == '__main__':
    json.dump(art, open(GFX / f'maps/{ID}.json', 'w'), ensure_ascii=False, indent=1)
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', ID], cwd=GFX.parents[1], check=True)
    print(ID, art['size'], len(obj), 'objects')
