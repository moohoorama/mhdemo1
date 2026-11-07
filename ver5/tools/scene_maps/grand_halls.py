#!/usr/bin/env python3
"""Make the two great halls where a whole court stands: S_shouchun_hall (원술's hall at 수춘, chapter 14) and
S_luoyang_hall (the throne hall at 낙양 where 동탁 stands by the boy emperor, chapter 5).

  python3 ver5/tools/scene_maps/grand_halls.py

Format 2 (internal/scenemap), a room like xuzhou_hall.py: walls on the two back sides (-u, -v), the front open to the
camera. Both are 13 x 12 cells, larger than 서주 관청 (9 x 8), so the court fills them: the seat on a dais against the
back (-v) wall in the middle (u = 6), the red carpet from the dais to the open front (v = 11), the way in between two
red pillars; officials stand three files deep on each side of the carpet (u = 2-4 and 8-10, v = 3-8), red pillars
outside the files (u = 1 and 11).

The two differ in their furnishing:
- 수춘: 원술 apes an emperor's hall: a gold screen behind the gilded seat (the dais), yellow banners all along the back wall,
  and the wine tables and jars of a lord who lives well.
- 낙양: the real throne hall: the gold screen and the dais (the emperor's sheet brings its own throne), curtains
  over the lacquer walls, candle stands, incense and lanterns; no wine.
"""
import json
import math
import subprocess
from pathlib import Path

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
W, H = 13, 12        # game cells
DOOR = 6             # the carpet and the way in, u


def hall(map_id, name, furnish):
    obj = []

    def put(prop, u, v):
        """A prop at game-cell point (u, v) (a whole number is a cell's centre) as a small-tile anchor and offset."""
        U, V = math.floor(2 * u + 1), math.floor(2 * v + 1)
        x, y = math.floor((u - v) * 32 + .5), math.floor((u + v + 1) * 16 + .5)
        dx, dy = x - 16 * (U - V), y - 8 * (U + V + 1)
        obj.append([prop, U, V] + ([dx, dy] if dx or dy else []))

    ground = [['P'] * (2 * W) for _ in range(2 * H)]
    for V in range(4, 2 * H):
        ground[V][2 * DOOR] = ground[V][2 * DOOR + 1] = 'C'
    for V in range(0, 2 * H, 2):
        obj.append(['wall_lacquer_nw', 0, V, 0, -8])
    for U in range(0, 2 * W, 2):
        obj.append(['wall_lacquer_ne', U, 0, 0, -8])
    obj.extend([['pillar_red', 0, 0, 0, -8], ['pillar_red', 2 * W, 0, 0, -8], ['pillar_red', 0, 2 * H, 0, -8]])
    put('screen_gold', DOOR, .35), put('dais', DOOR, 1.0)
    for v in (2.5, 5.5, 8.5):
        put('pillar_red', 1, v), put('pillar_red', W - 2, v)
    put('pillar_red', DOOR - 1, H - .25), put('pillar_red', DOOR + 1, H - .25)
    furnish(put, obj)
    art = {'id': map_id, 'name': name, 'format': 2, 'size': [W, H], 'origin': [0, 0],
           'ground': [''.join(r) for r in ground], 'objects': obj}
    json.dump(art, open(GFX / f'maps/{map_id}.json', 'w'), ensure_ascii=False, indent=1)
    index = json.load(open(GFX / 'index.json'))
    if f'maps/{map_id}.json' not in index['maps']:
        index['maps'].append(f'maps/{map_id}.json')
        json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
        open(GFX / 'index.json', 'a').write('\n')
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', map_id], cwd=GFX.parents[1], check=True)
    print(map_id, art['size'], len(obj), 'objects')


def shouchun(put, obj):
    for u in (1.6, 3.6, 8.4, 10.4):
        put('banner_yellow', u, -.2)
    for v in (2.0, 6.0, 10.0):
        put('banner_yellow', -.2, v)
    put('candle_stand', DOOR - 1.5, .4), put('candle_stand', DOOR + 1.5, .4)
    put('incense', DOOR - 1.3, 1.7), put('incense', DOOR + 1.3, 1.7)
    for u in (2.0, 3.5, 8.5, 10.0):
        put('table_wine', u, 10.2)
    put('wine_jar', W - .6, .3), put('wine_jar', W - .3, .5), put('wine_jar', -.1, H - 1.4), put('wine_jar', .2, H - 1.1)
    put('brazier', 3.0, 1.0), put('brazier', 9.0, 1.0)


def luoyang(put, obj):
    for V in range(0, 2 * H, 2):
        obj.append(['curtain_nw', 0, V, 0, -8])
    for U in range(0, 2 * W, 2):
        obj.append(['curtain_ne', U, 0, 0, -8])
    put('candle_stand', DOOR - 1.5, .4), put('candle_stand', DOOR + 1.5, .4)
    put('incense', DOOR - 1.3, 1.7), put('incense', DOOR + 1.3, 1.7)
    for u in (2.5, 9.5):
        put('lantern', u, -.1)
    for v in (3.0, 7.0):
        put('lantern', -.1, v)
    put('candle_stand', 2.0, 1.0), put('candle_stand', 10.0, 1.0)


if __name__ == '__main__':
    hall('S_shouchun_hall', '수춘 대전', shouchun)
    hall('S_luoyang_hall', '낙양 궁궐 대전', luoyang)
