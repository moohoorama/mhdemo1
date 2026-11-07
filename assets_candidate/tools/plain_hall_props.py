#!/usr/bin/env python3
"""Plain copies of the palace hall pieces for a provincial governor's hall (S_xuzhou_hall, chapter 13): the same
pictures with their palette swapped — the red lacquer, red curtains, red pillars, gold screen and red carpet of the
throne room turned to dark wood, slate-grey curtains, umber pillars, a paper screen, a faded brown mat and a wooden dais.

  python3 assets_candidate/tools/plain_hall_props.py     # writes tiles/<new>.png and their tiles.json entries

Then python3 ver5/tools/build_extra.py packs them into the game's extra sheet.
"""
import colorsys
import json
from pathlib import Path

from PIL import Image

CAND = Path(__file__).resolve().parents[1]
PROPS = CAND.parent / 'ver5/assets/graphics/map/props.png'

# new name: (source file or props.png rect and pivot, {hue band: (new hue in degrees, saturation factor, value factor)})
RED, GOLD = 'red', 'gold'
SWAPS = {
    'curtain_slate_ne': ('tiles/curtain_ne.png', {RED: (210, .3, .72), GOLD: (40, .3, .75)}),
    'curtain_slate_nw': ('tiles/curtain_nw.png', {RED: (210, .3, .72), GOLD: (40, .3, .75)}),
    'wall_umber_ne': ('tiles/wall_lacquer_ne.png', {RED: (24, .5, .75), GOLD: (35, .35, .8)}),
    'wall_umber_nw': ('tiles/wall_lacquer_nw.png', {RED: (24, .5, .75), GOLD: (35, .35, .8)}),
    'pillar_umber': ((608, 0, 15, 50, 8, 46), {RED: (24, .5, .75), GOLD: (35, .35, .8)}),
    'screen_paper': ('tiles/screen_gold.png', {RED: (20, .4, 1), GOLD: (42, .22, .95)}),
    'floor_mat': ('tiles/floor_carpet.png', {RED: (30, .38, .85), GOLD: (40, .35, .9)}),
    'dais_plain': ((640, 0, 53, 54, 27, 40), {RED: (24, .45, .7), GOLD: (32, .4, .75)}),
}
FROM = {'curtain_slate_ne': 'curtain_ne', 'curtain_slate_nw': 'curtain_nw', 'wall_umber_ne': 'wall_lacquer_ne',
        'wall_umber_nw': 'wall_lacquer_nw', 'screen_paper': 'screen_gold', 'floor_mat': 'floor_carpet'}


def band(h, s):
    if s < .25:
        return None
    deg = h * 360
    if deg <= 18 or deg >= 330:
        return RED
    if 22 <= deg <= 65:
        return GOLD
    return None


def swap(im, table):
    px = im.load()
    for y in range(im.height):
        for x in range(im.width):
            r, g, b, a = px[x, y]
            if a == 0:
                continue
            h, s, v = colorsys.rgb_to_hsv(r / 255, g / 255, b / 255)
            t = table.get(band(h, s))
            if t:
                nh, sf, vf = t
                r, g, b = (round(c * 255) for c in colorsys.hsv_to_rgb(nh / 360, s * sf, min(1, v * vf)))
                px[x, y] = (r, g, b, a)
    return im


def main():
    path = CAND / 'tiles/tiles.json'
    doc = json.load(open(path))
    tiles = doc['sprites']
    for name, (src, table) in SWAPS.items():
        if isinstance(src, tuple):
            x, y, w, h, px, py = src
            im = Image.open(PROPS).convert('RGBA').crop((x, y, x + w, y + h))
            entry = {'size': [w, h], 'pivot': [px, py], 'layer': 'object', 'scale': 'scene'}
        else:
            im = Image.open(CAND / src).convert('RGBA')
            entry = {k: v for k, v in tiles[FROM[name]].items() if k != 'made'}
        swap(im, table).save(CAND / f'tiles/{name}.png')
        entry.update(file=f'tiles/{name}.png', made='palette swap of the palace piece (plain_hall_props.py)')
        tiles[name] = entry
        print(name)
    json.dump(doc, open(path, 'w'), ensure_ascii=False, indent=1)
    open(path, 'a').write('\n')


if __name__ == '__main__':
    main()
