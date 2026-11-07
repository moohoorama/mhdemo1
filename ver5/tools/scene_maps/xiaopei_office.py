#!/usr/bin/env python3
"""Make S_xiaopei_office, the county office of 소패 that 도겸 lends 유비 (chapter 13, 소패의 손님).

  python3 ver5/tools/scene_maps/xiaopei_office.py

Format 2 (internal/scenemap), a room: small tiles from (0, 0), no margin; the walls stand on the two back sides
(-u, -v) and the front two are open to the camera. Objects are placed in game cells and turned into small-tile anchors
with canvas offsets as in xuzhou_hall.py. Ground: W wooden floor.

Plan (8 x 6 cells, a plain county office, no dais, no carpet): the magistrate's plain screen and desk against the
back (-v) wall at the left (u = 2), the way in straight down from it at the open front (v = 5) between two wooden
pillars; in the middle right a low wine table set for a farewell (two tables end to end along u at v = 3, people
sitting on either side at v = 2 and v = 4); the map of 서주 on a table in the back right corner, wine jars beside it;
spears on racks along the left wall.
"""
import json
import math
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
W, H = 8, 6          # game cells
DOOR = 2             # the seat and the way in, u
obj = []


def put(name, u, v):
    """A prop at game-cell point (u, v) (a whole number is a cell's centre) as a small-tile anchor and offset."""
    U, V = math.floor(2 * u + 1), math.floor(2 * v + 1)
    x, y = math.floor((u - v) * 32 + .5), math.floor((u + v + 1) * 16 + .5)
    dx, dy = x - 16 * (U - V), y - 8 * (U + V + 1)
    obj.append([name, U, V] + ([dx, dy] if dx or dy else []))


ground = [['W'] * (2 * W) for _ in range(2 * H)]

for V in range(0, 2 * H, 2):
    obj.append(['wall_plaster_nw', 0, V, 0, -8])
for U in range(0, 2 * W, 2):
    obj.append(['wall_plaster_ne', U, 0, 0, -8])
obj.extend([['pillar_wood', 0, 0, 0, -8], ['pillar_wood', 2 * W, 0, 0, -8], ['pillar_wood', 0, 2 * H, 0, -8]])

# the magistrate's seat: a plain screen and his desk, a candle stand, the banner of the Han
put('screen_u', DOOR, .2), put('desk', DOOR, 1.7), put('candle_stand', DOOR - 1.1, .3), put('banner', DOOR + 1.3, -.2)
# the farewell table and the wine
put('table_wine', 4.5, 3), put('table_wine', 5.5, 3), put('wine_jar', 7.3, 1.6), put('wine_jar', 7.2, 2.1)
# the map table in the back right corner, a lantern, jars
put('map_table', 5.6, .7), put('lantern', 7.3, -.1), put('water_jar', 7.1, .3), put('lantern', -.1, 4.6)
# weapons along the left wall
put('weapon_rack', -.1, 1.6), put('weapon_rack', -.1, 2.7), put('armor_stand', -.05, 3.7)
# the doorway at the front
put('pillar_wood', DOOR - 1, 5.75), put('pillar_wood', DOOR + 1, 5.75)

art = {'id': 'S_xiaopei_office', 'name': '소패 관청', 'format': 2, 'size': [W, H], 'origin': [0, 0],
       'ground': [''.join(r) for r in ground], 'objects': obj}

if __name__ == '__main__':
    json.dump(art, open(GFX / 'maps/S_xiaopei_office.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if 'maps/S_xiaopei_office.json' not in index['maps']:
        index['maps'].append('maps/S_xiaopei_office.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', 'S_xiaopei_office'], cwd=GFX.parents[1], check=True)
    print('S_xiaopei_office', art['size'], len(obj), 'objects')
