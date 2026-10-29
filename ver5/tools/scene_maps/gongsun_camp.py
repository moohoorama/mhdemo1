#!/usr/bin/env python3
"""Make S_gongsun_camp, 공손찬's camp on the west bank of the 계수 at 계교 (chapter 10), walked before the battle.

  python3 ver5/tools/scene_maps/gongsun_camp.py

Format 2 (internal/scenemap): small tiles, the playing area from (0, 0) to (W-1, H-1) with a margin of M drawn round
it; u runs down to the right on screen, v down to the left. Ground: d packed earth, . grass, ~ water.

Plan: a fenced square camp on flat ground. Its gate opens on the near-right (+u) side onto the river road; past the
road the 계수 runs along the right edge with the 계교 bridge beyond the frame. The command tent stands at the back
behind a dirt yard with the command flags; rows of tents either side of the avenue from the gate; by the gate the
wounded of the white horse guard lie on straw mats after the day's defeat; grain carts and racks at the back left.
"""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
W, H, M = 36, 28, 12
FENCE = (1, 31, 1, 26)    # u from, to, v from, to
GATE = (12, 16)           # gap in the +u fence, v from, to (exclusive)
RIVER = (38, 42)          # u from, to

g = [['.'] * (W + 2 * M) for _ in range(H + 2 * M)]
obj = []


def paint(u0, v0, u1, v1, ch):
    for v in range(max(-M, v0), min(H + M - 1, v1) + 1):
        for u in range(max(-M, u0), min(W + M - 1, u1) + 1):
            g[v + M][u + M] = ch


def put(name, u, v):
    obj.append([name, u, v])


def fence():
    ua, ub, va, vb = FENCE
    links = {}
    for u in range(ua, ub + 1):
        for v in (va, vb):
            links[u, v] = links.get((u, v), 0) | (1 if u > ua else 0) | (4 if u < ub else 0)
    for v in range(va, vb + 1):
        for u in (ua, ub):
            if u == ub and GATE[0] <= v < GATE[1]:
                continue
            links[u, v] = links.get((u, v), 0) | (2 if v > va and not (u == ub and v == GATE[1]) else 0) \
                | (8 if v < vb and not (u == ub and v == GATE[0] - 1) else 0)
    obj.extend([f'sfence_{m:02d}', u, v] for (u, v), m in sorted(links.items()))


ua, ub, va, vb = FENCE
paint(ua + 1, GATE[0], W + M, GATE[1] - 1, 'd')           # the avenue from the gate to the command yard, and the road
paint(6, 3, 18, 9, 'd')                                     # the command yard
paint(RIVER[0] - 3, -M, RIVER[0] - 1, H + M, 'd')           # the river road
paint(RIVER[0], -M, RIVER[1], H + M, '~')
fence()

put('command_tent', 11, 4), put('command_flag', 7, 6), put('command_flag', 15, 6), put('banner', 9, 9), put('banner', 13, 9)
put('brazier', 6, 9), put('brazier', 17, 9), put('map_table', 12, 8)
kinds = ['tent', 'tent_round', 'tent']
k = 0
for v in (4, 9):
    for u in (22, 27):
        put(kinds[k % 3], u, v); k += 1
for v in (19, 23):
    for u in (6, 11, 16, 21):
        put(kinds[k % 3], u, v); k += 1
put('campfire', 24, 7), put('campfire', 9, 21), put('campfire', 19, 21)
put('grain_cart', 3, 4), put('grain_sack', 3, 8), put('grain_sack', 4, 9), put('weapon_rack', 3, 11), put('weapon_rack', 3, 16)
# the wounded of the white horse guard by the gate
for u, v in ((25, 17), (27, 17), (29, 17), (25, 20), (27, 20)):
    put('straw_mat', u, v)
put('medicine_pot', 29, 20), put('water_bucket', 30, 21), put('bloody_cloth', 26, 22)
put('banner', 30, 11), put('banner', 30, 16)
# willows along the far bank, trees behind the camp
for v in (-4, 6, 18, 30):
    put('stree_willow', 45, v)
for u, v, name in ((-6, 4, 'stree_broad_1'), (-7, 18, 'stree_pine_0'), (8, -7, 'stree_pine_1'), (22, -8, 'stree_broad_0'),
                   (-5, 30, 'stree_old'), (12, 35, 'stree_broad_0'), (30, 36, 'stree_pine_1')):
    put(name, u, v)

art = {'id': 'S_gongsun_camp', 'name': '계교 · 공손찬의 진영', 'format': 2, 'size': [W // 2, H // 2], 'origin': [-M, -M],
       'ground': [''.join(r) for r in g], 'objects': obj}

if __name__ == '__main__':
    json.dump(art, open(GFX / 'maps/S_gongsun_camp.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if 'maps/S_gongsun_camp.json' not in index['maps']:
        index['maps'].append('maps/S_gongsun_camp.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', 'S_gongsun_camp'], cwd=GFX.parents[1], check=True)
    print('S_gongsun_camp', art['size'], len(obj), 'objects')
