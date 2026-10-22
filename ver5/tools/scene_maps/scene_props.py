#!/usr/bin/env python3
"""Draw the scene pictures that are recolours or a few pixels: Yellow Turban flags (the red camp flags in ochre), a
gold screen (the screen in gold leaf), and the scene effects' birds and glints. The modelled props (weapon rack,
armour stand, candle stand, antler barrier, watchtower, shield wall, straw mat, bloody cloth, medicine pot, broken
spear, fallen flag, cauldron, grain sack, wine jar) come from tools/unit3d/scene_props3d.py. Both write
assets_candidate/tiles/<name>.png and their entries in tiles.json; ver5/tools/build_extra.py then packs them into the
game's extra sheet.

  python3 ver5/tools/scene_maps/scene_props.py && python3 tools/unit3d/scene_props3d.py && python3 ver5/tools/build_extra.py
"""
import json
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[3]
CAND = ROOT / 'assets_candidate'
GFX = ROOT / 'ver5/assets/graphics'

# the camp flags' red cloth and gold emblem, and what they become on a Yellow Turban flag
YELLOW = {(171, 22, 51): (204, 158, 38), (101, 13, 36): (138, 96, 22), (237, 35, 66): (240, 202, 74),
          (230, 157, 35): (122, 62, 24), (255, 208, 82): (150, 82, 30), (173, 99, 24): (96, 46, 18)}
# the screen's green panels and pale paper in gold leaf, its frame in red lacquer (원술's tent, the throne)
GILT = {(96, 128, 112): (196, 146, 52), (120, 150, 130): (232, 186, 80), (236, 226, 196): (250, 232, 172),
        (214, 200, 164): (224, 196, 126), (120, 76, 44): (150, 36, 34)}

def birds():
    """A bird in three wing beats (up, level, down), dark against the sky."""
    out = []
    beats = (['o.......o', '.o.....o.', '..o...o..', '...ooo...'],
             ['.........', 'ooo...ooo', '...ooo...', '....o....'],
             ['....o....', '...ooo...', '..o...o..', '.o.....o.'])
    for f, rows in enumerate(beats):
        im = Image.new('RGBA', (9, 4), (0, 0, 0, 0))
        for y, r in enumerate(rows):
            for x, ch in enumerate(r):
                if ch == 'o':
                    im.putpixel((x, y), (32, 28, 30, 255))
        out.append((f'bird_{f}', im, [4, 2]))
    return out


def glints():
    out = []
    shapes = (['.w.', 'www', '.w.'], ['..w..', '..w..', 'wwWww', '..w..', '..w..'], ['w.w', '.W.', 'w.w'])
    for f, rows in enumerate(shapes):
        im = Image.new('RGBA', (len(rows[0]), len(rows)), (0, 0, 0, 0))
        for y, r in enumerate(rows):
            for x, ch in enumerate(r):
                if ch in 'wW':
                    im.putpixel((x, y), (255, 255, 240, 255) if ch == 'W' else (220, 236, 255, 230))
        out.append((f'glint_{f}', im, [len(rows[0]) // 2, len(rows) // 2]))
    return out


def recolor(im, table):
    out = im.copy()
    px = out.load()
    for y in range(out.height):
        for x in range(out.width):
            r, g, b, a = px[x, y]
            if a and (r, g, b) in table:
                px[x, y] = (*table[(r, g, b)], a)
    return out


def main():
    index = json.load(open(GFX / 'index.json'))
    sheet = Image.open(GFX / index['props']['image']).convert('RGBA')
    made = {}
    for name in ('banner', 'command_flag'):
        p = index['props']['sprites'][name]
        x, y, w, h = p['rect']
        made[f'{name}_yellow'] = (recolor(sheet.crop((x, y, x + w, y + h)), YELLOW), p['pivot'])
    extra = index['extra']['sprites']
    x, y, w, h = extra['screen_u']['rect']
    made['screen_gold'] = (recolor(Image.open(GFX / index['extra']['image']).convert('RGBA').crop((x, y, x + w, y + h)), GILT),
                           extra['screen_u']['pivot'])
    for name, im, pivot in birds() + glints():
        made[name] = (im, pivot)
    path = CAND / 'tiles/tiles.json'
    tiles = json.load(open(path))
    for name, (im, pivot) in made.items():
        im.save(CAND / 'tiles' / f'{name}.png')
        tiles['sprites'][name] = {'file': f'tiles/{name}.png', 'size': list(im.size), 'pivot': pivot, 'layer': 'object'}
        print(name, im.size)
    json.dump(tiles, open(path, 'w'), ensure_ascii=False, indent=1)
    open(path, 'a').write('\n')


if __name__ == '__main__':
    main()
