#!/usr/bin/env python3
"""Make S_xuzhou_hall, the great hall of 서주 관청 where 도겸 receives his guests (chapter 13, 소패의 손님).

  python3 ver5/tools/scene_maps/xuzhou_hall.py

Format 2 (internal/scenemap), a room: small tiles from (0, 0), no margin; the walls stand on the two back sides
(-u, -v) and the front two are open to the camera. Objects are placed here in game cells (a person's cell is a whole
number; a prop's anchor may sit between cells) and turned into small-tile anchors with canvas offsets, the way
scene_v2.py turns the old maps' props. Ground: P palace floor, M faded mat (per game cell).

Colours are plain, not a palace's (plain_hall_props.py swaps the red and gold pieces to wood, slate,
paper and a faded mat).

Plan (9 x 8 cells, larger than the county offices): the governor's seat on a dais against the back (-v) wall in the
middle (u = 4) behind his desk, a gold screen and candle stands; the red carpet runs from the dais to the open front
(v = 7), the way in between two red pillars. The officials stand in two rows either side of the carpet (u = 2 and
u = 6), a pair of red pillars by each row; guests stand on and beside the carpet facing the dais. Banners of the
Han on the back wall, lanterns, a weapon rack and armour by the side wall.
"""
import json
import math
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
W, H = 9, 8          # game cells
DOOR = 4             # the carpet and the way in, u
obj = []


def put(name, u, v):
    """A prop at game-cell point (u, v) (a whole number is a cell's centre) as a small-tile anchor and offset."""
    U, V = math.floor(2 * u + 1), math.floor(2 * v + 1)
    x, y = math.floor((u - v) * 32 + .5), math.floor((u + v + 1) * 16 + .5)
    dx, dy = x - 16 * (U - V), y - 8 * (U + V + 1)
    obj.append([name, U, V] + ([dx, dy] if dx or dy else []))


ground = [['P'] * (2 * W) for _ in range(2 * H)]
for v in range(6, 2 * H):
    ground[v][2 * DOOR] = ground[v][2 * DOOR + 1] = 'M'

for V in range(0, 2 * H, 2):
    obj.extend([['wall_umber_nw', 0, V, 0, -8], ['curtain_slate_nw', 0, V, 0, -8]])
for U in range(0, 2 * W, 2):
    obj.extend([['wall_umber_ne', U, 0, 0, -8], ['curtain_slate_ne', U, 0, 0, -8]])
obj.extend([['pillar_umber', 0, 0, 0, -8], ['pillar_umber', 2 * W, 0, 0, -8], ['pillar_umber', 0, 2 * H, 0, -8]])

# the seat: gold screen, dais, desk, candle stands and incense
put('screen_paper', DOOR, .35), put('dais_plain', DOOR, .95), put('desk', DOOR, 2.0)
put('candle_stand', DOOR - 1.4, .4), put('candle_stand', DOOR + 1.4, .4), put('incense', DOOR + 1.4, 1.6)
put('banner', 1.6, -.2), put('banner', 6.4, -.2), put('lantern', 8.3, -.1), put('lantern', -.1, 7.3)
# the pillars by the two rows of officials
for v in (2.5, 5.5):
    put('pillar_umber', .9, v), put('pillar_umber', 7.1, v)
put('pillar_umber', DOOR - 1, 7.75), put('pillar_umber', DOOR + 1, 7.75)  # the doorway at the front
put('weapon_rack', -.1, 1.2), put('armor_stand', -.05, 4.0)
put('wine_jar', 8.2, .3), put('wine_jar', 8.5, .2)

art = {'id': 'S_xuzhou_hall', 'name': '서주 관청 대청', 'format': 2, 'size': [W, H], 'origin': [0, 0],
       'ground': [''.join(r) for r in ground], 'objects': obj}

if __name__ == '__main__':
    json.dump(art, open(GFX / 'maps/S_xuzhou_hall.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if 'maps/S_xuzhou_hall.json' not in index['maps']:
        index['maps'].append('maps/S_xuzhou_hall.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', 'S_xuzhou_hall'], cwd=GFX.parents[1], check=True)
    print('S_xuzhou_hall', art['size'], len(obj), 'objects')
