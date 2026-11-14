#!/usr/bin/env python3
"""Make S_pond_deck, 조조's back garden in 허도 for the 논영회 (chapter 17, c17-garden): a 정자 without roof or pillars —
a plank deck raised one step on a granite plinth (scene.json heights "5", pictures pdeck/prail of
assets_candidate/tools/gen_pixel_pavilion.py), a lacquered railing round its outer edges, half of it standing in a pond,
with a low wine table in the middle where 조조 and 유비 sit face to face. A dirt path comes in from the south-west to a
flight of two stone steps (heights "6", pstep) where the railing is open; plum trees, 태호석 rocks and two lanterns stand round it, and
the woods of S_garden (the trees well outside its old core) close the picture. User pick 11-14: proposal B of
https://claude.ai/artifact/6nvJHi25Cd2NJzfJavYEhq.

Game cells: deck u 2..6 × v 1..3, pond u 4..8 × v 0..3, table (4, 2), steps in front of (3, 3) (small row 8), path u 3
from v 4 south.

  python3 ver5/tools/scene_maps/pond_deck.py
"""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
ID = 'S_pond_deck'

src = json.load(open(GFX / 'maps/S_garden.json'))
ou, ov = src['origin']
rows = [list('.' * len(src['ground'][0])) for _ in src['ground']]


def put(u0, u1, v0, v1, c):
    for v in range(v0, v1 + 1):
        for u in range(u0, u1 + 1):
            rows[v - ov][u - ou] = c


put(8, 17, 0, 7, '~')      # pond, game u 4..8, v 0..3
put(4, 13, 2, 7, 'W')      # deck, game u 2..6, v 1..3
put(6, 7, 9, len(rows) + ov - 1, 'd')  # path, game u 3, from the steps south
put(6, 7, 8, 8, 'i')       # the steps' tread: stone paving
height = [list('0' * len(r)) for r in rows]
for v in range(2, 8):
    for u in range(4, 14):
        height[v - ov][u - ou] = '5'
for u in (6, 7):
    height[8 - ov][u - ou] = '6'

core = lambda o: -6 <= o[1] <= 22 and -6 <= o[2] <= 18  # the deck's tops must not be reached by trees in front of it (any turn)
objs = [o for o in src['objects'] if o[0].startswith('stree_') and not o[0].startswith('stree_plum') and not core(o)]
objs += [
    ['table_wine', 9, 5, -8, -4],
    ['wine_jar', 12, 7, 0, 0],
    ['lantern', 4, 6, 0, 0],
    ['lantern', 13, 2, 0, 0],
    ['brazier', 2, 9, 0, 0],
    ['taihu_rock', 17, 9, 0, 0],
    ['taihu_rock', 1, 0, 0, 5],
    ['taihu_rock', 18, 1, -4, 0],
    ['stree_plum_0', -3, 1, 5, -2],
    ['stree_plum_1', -2, 13, -4, 0],
    ['stree_plum_0', 14, 16, 0, 0],
    ['stree_plum_1', 16, 14, 0, 0],
]
art = {'id': ID, 'name': '조조의 후원 · 못가 마루', 'format': 2, 'size': [9, 8], 'origin': [ou, ov],
       'ground': [''.join(r) for r in rows], 'height': [''.join(r) for r in height], 'objects': objs}

if __name__ == '__main__':
    json.dump(art, open(GFX / f'maps/{ID}.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if f'maps/{ID}.json' not in index['maps']:
        index['maps'].append(f'maps/{ID}.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', ID], cwd=GFX.parents[1], check=True)
    print(ID, art['size'], len(objs), 'objects')
