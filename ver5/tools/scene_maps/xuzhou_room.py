#!/usr/bin/env python3
"""Make S_xuzhou_room, the bedchamber of 도겸's house in 서주 (his quarters behind the 관청)
where 도겸 lies dying and hands 서주 to 유비 (chapter 14).

  python3 ver5/tools/scene_maps/xuzhou_room.py

Format 2, a room like xuzhou_hall.py (same walls and floor, so it reads as the same house): 7 x 6 cells. The bed
stands (drawn with 도겸 in his sheet's bed pose, not a map object) against the back (-v) wall on the left (2, 1),
a paper screen behind it, a candle stand and incense beside it;
a small desk with a medicine pot by the right wall, a lantern; the doorway at the front (3, 5) between two pillars.
"""
import json
import math
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
ID = 'S_xuzhou_room'
W, H = 7, 6
DOOR = 3
obj = []


def put(name, u, v):
    """A prop at game-cell point (u, v) as a small-tile anchor and offset."""
    U, V = math.floor(2 * u + 1), math.floor(2 * v + 1)
    x, y = math.floor((u - v) * 32 + .5), math.floor((u + v + 1) * 16 + .5)
    dx, dy = x - 16 * (U - V), y - 8 * (U + V + 1)
    obj.append([name, U, V] + ([dx, dy] if dx or dy else []))


ground = [['P'] * (2 * W) for _ in range(2 * H)]
for V in range(0, 2 * H, 2):
    obj.extend([['wall_umber_nw', 0, V, 0, -8], ['curtain_slate_nw', 0, V, 0, -8]])
for U in range(0, 2 * W, 2):
    obj.extend([['wall_umber_ne', U, 0, 0, -8], ['curtain_slate_ne', U, 0, 0, -8]])
obj.extend([['pillar_umber', 0, 0, 0, -8], ['pillar_umber', 2 * W, 0, 0, -8], ['pillar_umber', 0, 2 * H, 0, -8]])
put('screen_paper', 2, .25)
put('candle_stand', .6, .4), put('incense', 3.6, .6), put('candle_stand', 3.9, .3)
put('desk', 6, 1.5), put('medicine_pot', 6.1, 2.5), put('lantern', 6.4, -.1)
put('pillar_umber', DOOR - 1, H - .25), put('pillar_umber', DOOR + 1, H - .25)

art = {'id': ID, 'name': '도겸의 집 침소', 'format': 2, 'size': [W, H], 'origin': [0, 0],
       'ground': [''.join(r) for r in ground], 'objects': obj}

if __name__ == '__main__':
    json.dump(art, open(GFX / f'maps/{ID}.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if f'maps/{ID}.json' not in index['maps']:
        index['maps'].append(f'maps/{ID}.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', ID], cwd=GFX.parents[1], check=True)
    print(ID, art['size'], len(obj), 'objects')
