#!/usr/bin/env python3
"""Make S_xiapi, 하비 as 유비 finds it on his way back from 광릉 (chapter 18, explore18): the same streets as the
서주성 of chapter 14 (xuzhou_town.py), a year after 조조's flood — walled round now, the lanes fenced off yard by yard
as the houses went back up, a few walls still stained or lying in rubble, and the 미 family's house in its own
walled compound where the pavilion garden was.

  python3 ver5/tools/scene_maps/xiapi_town.py

Format 2 (internal/scenemap), the layout of xuzhou_town.py with these changes:
  - the town wall on all four sides: the two back walls as before, the near ones past the playing area
    (v = NEAR_V, u = NEAR_U); what stood beyond them is taken away and the ground there is trodden fields.
  - the 미 house: a white compound wall (cwall) round a hall in the south-west, its open gate (cgate_open) on the
    near (+v) wall, where the pavilion garden was.
  - wattle fences (sfence) between the yards of the south lanes and round the market square's back.
  - the flood's last traces: two houses with the stain still on their walls, rubble by them.
"""
import json
import subprocess

import xuzhou_town as base

GFX = base.GFX
ID = 'S_xiapi'
W, H, M = base.W, base.H, base.M
NEAR_V, NEAR_U = H + 4, W + 4
MI = (0, 13, 29, 38)     # the 미 house compound: u0, u1, v0, v1 of its wall lines
MI_GATE = 7

g = [list(r) for r in base.g]


def paint(u0, v0, u1, v1, ch):
    for v in range(max(-M, v0), min(H + M - 1, v1) + 1):
        for u in range(max(-M, u0), min(W + M - 1, u1) + 1):
            g[v + M][u + M] = ch


def buildingish(name):
    return name.startswith(('house', 'tavern', 'hall', 'market', 'grain', 'wine', 'water', 'well', 'field', 'straw'))


obj = []
for o in base.obj:
    name, u, v = o[0], o[1], o[2]
    if (u >= NEAR_U - 1 or v >= NEAR_V - 1) and buildingish(name):
        continue                                                     # outside the near walls
    if MI[0] <= u <= MI[1] + 1 and MI[2] - 1 <= v <= MI[3] + 1:
        continue                                                     # the garden gives way to the 미 house
    if (name, u, v) in {('house_tile', 15, 35), ('house_thatch', 17, 40)}:
        continue                                                     # the lot next to the 미 house's wall
    obj.append(o)


def put(name, u, v, *rest):
    obj.append([name, u, v, *rest])


paint(-M, NEAR_V, W + M, H + M, 'd')     # fields outside the near walls
paint(NEAR_U, -M, W + M, H + M, 'd')

# the 미 house: the hall at the back of a paved court, a side hall, plum trees; the open gate on the near wall
paint(MI[0] + 1, MI[2] + 1, MI[1] - 1, MI[3] - 1, '.')
paint(MI[0] + 2, MI[2] + 4, MI[1] - 2, MI[3] - 1, 'i')
paint(MI_GATE - 2, MI[3], MI_GATE + 1, MI[3] + 2, 'i')
obj.append(['cgate_open', MI_GATE, MI[3], {'wood': '#6a4a36'}])
obj.extend([['hall_u', 7, 32, 8, 4], ['hall_v', 3, 35, 8, 4], ['flagstones_0', 7, 35, 8, 4]])
put('stree_plum_0', 2, 30), put('stree_plum_1', 12, 30), put('stree_pine_1', 12, 36)
put('lantern', 5, 34), put('lantern', 10, 34), put('grain_sack', 11, 37), put('water_jar', 2, 37)
put('banner', 4, 40), put('banner', 11, 40)

# the flood's last traces: two houses still stained to the waterline, rubble beside them
for i, o in enumerate(obj):
    if (o[0], o[1], o[2]) in {('house_tile', 27, 35), ('house_thatch_x', 6, 42)}:
        obj[i] = [o[0] + '_mud', *o[1:]]
put('rubble_1', 29, 38), put('rubble_3', 9, 41)

# wattle fences: between the yards of the south lanes, and behind the market square
fence = set()
for u in (18, 24, 30, 36):
    for v in range(32, 39):
        fence.add((u, v))
for u in range(16, 41):
    if u % 6 not in (0, 1):           # a gap into every yard
        fence.add((u, 32))
for v in range(41, NEAR_V - 1):
    fence.add((22, v)), fence.add((31, v))
for (u, v) in sorted(fence):
    m = (1 if (u - 1, v) in fence else 0) | (4 if (u + 1, v) in fence else 0) | \
        (2 if (u, v - 1) in fence else 0) | (8 if (u, v + 1) in fence else 0)
    put(f'sfence_{m:02d}', u, v)

walls = [['swall', [-1, -1], [NEAR_U, -1]], ['swall', [-1, 0], [-1, NEAR_V]],
         ['swall', [0, NEAR_V], [NEAR_U, NEAR_V]], ['swall', [NEAR_U, 0], [NEAR_U, NEAR_V - 1]],
         ['cwall', [MI[0], MI[2]], [MI[1], MI[2]]], ['cwall', [MI[0], MI[3]], [MI[1], MI[3]]],
         ['cwall', [MI[0], MI[2]], [MI[0], MI[3]]], ['cwall', [MI[1], MI[2]], [MI[1], MI[3]]]]

art = {'id': ID, 'name': '하비', 'format': 2, 'size': [W // 2, H // 2], 'origin': [-M, -M],
       'ground': [''.join(r) for r in g], 'objects': obj, 'walls': walls}

if __name__ == '__main__':
    json.dump(art, open(GFX / f'maps/{ID}.json', 'w'), ensure_ascii=False, indent=1)
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', ID], cwd=GFX.parents[1], check=True)
    print(ID, art['size'], len(obj), 'objects')
