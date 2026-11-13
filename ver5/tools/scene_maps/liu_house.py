#!/usr/bin/env python3
"""Make S_liu_house, the house 조조 gave 유비 in 허도 (chapter 17), from the hall of S_mansion2 with the same room and
furniture places (so the scenes and the walk keep their spots) but plain: plank floor with a reed mat, white plaster
walls, bare wooden pillars, a paper screen instead of the gold one and no dais, curtains or incense — a borrowed house
of a guest, not a lord's hall.

  python3 ver5/tools/scene_maps/liu_house.py
"""
import json
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
ID = 'S_liu_house'

src = json.load(open(GFX / 'maps/S_mansion2.json'))
swap = {'wall_lacquer_nw': 'wall_plaster_nw', 'wall_lacquer_ne': 'wall_plaster_ne', 'pillar_red': 'pillar_wood',
        'screen_gold': 'screen_paper'}
drop = {'curtain_nw', 'curtain_ne', 'dais', 'incense'}
objs = []
candles = 0
for o in src['objects']:
    name = o[0]
    if name in drop:
        continue
    if name == 'candle_stand':
        candles += 1
        if candles > 1:
            continue
    objs.append([swap.get(name, name)] + o[1:])
# plank floor, a reed mat down the middle where the carpet was
ground = [''.join('M' if c == 'C' else 'W' for c in row) for row in src['ground']]
art = dict(src, id=ID, name='유비의 저택', ground=ground, objects=objs)

if __name__ == '__main__':
    json.dump(art, open(GFX / f'maps/{ID}.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if f'maps/{ID}.json' not in index['maps']:
        index['maps'].append(f'maps/{ID}.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', ID], cwd=GFX.parents[1], check=True)
    print(ID, art['size'], len(objs), 'objects')
