#!/usr/bin/env python3
"""Make S_mangdang, the mountain village on 망탕산 where 장비 waits after 서주 falls (chapters 19–20).

It is S_danyang_village (the hillside village of chapter 4) in peace and deeper in the hills: the burnt houses are
whole again, the debris of the fight is gone, the trampled fields are kept. No cliff behind (the user found the hill
odd): the ground is flat, the village fenced on all four sides, and a wood of pines and broadleaves all round it.

  python3 ver5/tools/scene_maps/mangdang.py
"""
import json
import random
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
ID = 'S_mangdang'

art = json.load(open(GFX / 'maps/S_danyang_village.json'))
ground = [list(r) for r in art['ground']]
for r in range(36, len(ground)):   # the path down the front and the way east run on to the edge of the ground
    ground[r][32] = ground[r][33] = 'd'
for r in (20, 21):
    for c in range(44, len(ground[r])):
        ground[r][c] = 'd'
art['ground'] = [''.join(r) for r in ground]
GONE = {'broken_spear', 'fallen_flag', 'rubble_1'}
WHOLE = {'house_burnt': 'house_thatch', 'burnt_hut': 'hut'}
obj = [[WHOLE.get(n, n), u, v] for n, u, v in art['objects'] if n not in GONE]
obj += [['stree_pine_0', 34, 21], ['stree_pine_1', 30, 17],
        ['grain_sack', 7, 9], ['water_jar', 9, 12], ['wine_jar', 6, 15], ['wine_jar', 2, 15]]   # the 주막 at (3, 12)
# no hill behind: the ground is flat all round, the village fenced on all four sides (the back two had the cliff)
art.pop('height', None)
FENCE = {(0, 0): 'sfence_12', (0, 18): 'sfence_06', (18, 0): 'sfence_09'}
obj = [o for o in obj if not (o[0].startswith('sfence') and (o[1], o[2]) in FENCE)]
obj += [[n, u, v] for (u, v), n in FENCE.items()]
obj += [['sfence_05', u, 0] for u in range(1, 18)] + [['sfence_10', 0, v] for v in range(1, 18)]

# a wood all round outside the playing area (u, v 0..31, 0..23), thick behind, thin in front, off the roads
rnd = random.Random(7)
KINDS = ['stree_pine_0', 'stree_pine_1', 'stree_broad_0', 'stree_broad_1', 'stree_pine_1']
taken = {(u, v) for _, u, v in obj}
road = {(u, v) for v, row in enumerate(art['ground']) for u, ch in enumerate(row) if ch == 'd'}
ou, ov = art['origin']
for v in range(ov, ov + len(art['ground']), 3):
    for u in range(ou, ou + len(art['ground'][0]), 3):
        if 0 <= u <= 31 and 0 <= v <= 23 or 28 <= u <= 38 and 4 <= v <= 20:   # the board; the way in from the east
            continue
        du, dv = u + rnd.randint(-1, 1), v + rnd.randint(-1, 1)
        near = any((du + a, dv + b) in taken for a in (-1, 0, 1) for b in (-1, 0, 1))
        by_road = any((du - ou + a, dv - ov + b) in road for a in range(-6, 3) for b in range(-6, 3))   # nor hiding it
        front = u > 31 or v > 23   # the near side thinly, so the wood does not hide the board's front
        if near or by_road or rnd.random() < (.6 if front else .15):
            continue
        obj.append([rnd.choice(KINDS), du, dv])
        taken.add((du, dv))
obj = [o for o in obj if not (o[0].startswith('stree') and o[0] != 'stree_old' and o[0] != 'stree_willow'
                                and any((o[1] - ou + a, o[2] - ov + b) in road for a in range(-2, 3) for b in range(-2, 3)))]
art.update(id=ID, name='망탕산 산속 마을', objects=obj, grass={'hue': 18, 'sat': 1.2, 'val': 0.8})

if __name__ == '__main__':
    json.dump(art, open(GFX / f'maps/{ID}.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if f'maps/{ID}.json' not in index['maps']:
        index['maps'].append(f'maps/{ID}.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', ID], cwd=GFX.parents[1], check=True)
    print(ID, art['size'], len(obj), 'objects')
