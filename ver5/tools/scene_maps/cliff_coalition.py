#!/usr/bin/env python3
"""The cliff over the coalition camp (S_cliff_coalition), for chapter 8's dusk talk of Cao Cao and Liu Bei.

  python3 ver5/tools/scene_maps/cliff_coalition.py

A bare cliff top runs along the north-east (v < LIP) and drops on its south-west side, the side the camera sees,
so the rock face stands between the people on the lip and the camp spread below it. The camp fills the low ground:
a fenced square with a gate on the cliff road, a central avenue of braziers to the command tent, rows of tents,
fires and flags. Coordinates are small tiles; game cell (u, v) is small tile (2u, 2v) to (2u+1, 2v+1).
"""
import json
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
U0, U1, V0, V1 = -24, 71, -30, 53   # the picture, small tiles
LIP = 10                            # small rows v < LIP are the cliff top
ROAD = (11, 12)                     # the road along the cliff foot
FENCE = (-14, 54, 16, 46)           # the camp fence: u from, u to, v from, v to
GATE = (19, 22)                     # the gap in the north fence, u from, to (exclusive)
AVENUE = (19, 22)                   # the dirt avenue from the gate south through the camp


def fence(objs):
    """Pieces by their links: 1 -u, 2 -v, 4 +u, 8 +v."""
    ua, ub, va, vb = FENCE
    links = {}
    for u in range(ua, ub + 1):
        for v in (va, vb):
            if v == va and GATE[0] <= u < GATE[1]:
                continue
            links[u, v] = links.get((u, v), 0) | (1 if u > ua and not (v == va and u == GATE[1]) else 0) \
                | (4 if u < ub and not (v == va and u == GATE[0] - 1) else 0)
    for v in range(va, vb + 1):
        for u in (ua, ub):
            links[u, v] = links.get((u, v), 0) | (2 if v > va else 0) | (8 if v < vb else 0)
    objs += [[f'sfence_{m:02d}', u, v] for (u, v), m in sorted(links.items())]


def main():
    W, H = U1 - U0 + 1, V1 - V0 + 1
    ground, height = [], []
    for v in range(V0, V1 + 1):
        g, h = [], []
        for u in range(U0, U1 + 1):
            high = v < LIP
            soil = high or v in ROAD or (AVENUE[0] <= u < AVENUE[1] and ROAD[1] < v <= FENCE[3])   # the avenue meets the cliff road
            g.append('d' if soil else '.')
            h.append('2' if high else '0')
        ground.append(''.join(g))
        height.append(''.join(h))

    objs = []
    fence(objs)
    ua, ub, va, vb = FENCE
    mid = (AVENUE[0] + AVENUE[1]) // 2
    objs += [['brazier', AVENUE[0] - 1, v] for v in range(va + 3, vb - 8, 5)]
    objs += [['brazier', AVENUE[1], v] for v in range(va + 3, vb - 8, 5)]
    objs += [['command_tent', mid, vb - 7], ['command_flag', mid - 3, vb - 7], ['command_flag', mid + 3, vb - 7],
             ['banner', AVENUE[0] - 1, va + 1], ['banner', AVENUE[1], va + 1]]
    # rows of tents on both sides of the avenue, a fire and a flag here and there, in the rhythm of the camp map
    kinds = ['tent', 'tent_round', 'tent', 'tent', 'tent_round']
    k = 0
    for v in range(va + 4, vb - 2, 5):
        for u in list(range(ua + 4, AVENUE[0] - 3, 5)) + list(range(AVENUE[1] + 4, ub - 2, 5)):
            if abs(u - mid) < 8 and v > vb - 11:
                continue  # the command tent's yard
            k += 1
            objs.append([kinds[k % len(kinds)], u, v])
            if k % 4 == 0:
                objs.append(['campfire', u + 2, v + 2])
            elif k % 7 == 0:
                objs.append(['command_flag', u + 2, v + 1])
            elif k % 9 == 0:
                objs.append(['grain_sack', u + 2, v + 2])
    objs += [['grain_cart', ua + 6, vb - 3], ['grain_cart', ub - 6, va + 2], ['weapon_rack', mid + 4, va + 2]]
    # the bare top: a beacon tower behind the lip, a few bushes and pines; the lip where the people stand and the
    # paths they walk (rows 0 to 3 east of u 18, the columns down to their spots) stay clear
    objs += [['beacon', 16, 4], ['stree_bush_0', 27, 7], ['stree_bush_1', 39, 8],
             ['stree_pine_1', 54, -12], ['stree_pine_0', -6, -8]]
    # trees outside the fence on the low ground
    objs += [['stree_broad_1', -20, 30], ['stree_pine_1', -19, 44], ['stree_broad_0', 62, 24], ['stree_pine_0', 64, 40]]

    art = {'id': 'S_cliff_coalition', 'name': '연합 진영 위 벼랑', 'format': 2, 'size': [24, 16], 'origin': [U0, V0],
           'grass': {'hue': -4, 'sat': 0.95},
           'ground': ground, 'height': height, 'objects': objs}
    assert len(ground) == H and len(ground[0]) == W
    json.dump(art, open(GFX / 'maps/S_cliff_coalition.json', 'w'), ensure_ascii=False, separators=(',', ':'))
    index = json.load(open(GFX / 'index.json'))
    if 'maps/S_cliff_coalition.json' not in index['maps']:
        index['maps'].insert(index['maps'].index('maps/S_cliff_camp.json') + 1, 'maps/S_cliff_coalition.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')


if __name__ == '__main__':
    main()
