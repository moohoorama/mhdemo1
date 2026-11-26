#!/usr/bin/env python3
"""Tall scene cliffs made from the battle map's own cliff blocks: python3 ver5/tools/scene_maps/tall_cliff.py

A battle cliff block (assets/graphics/map/structures.png, cliff_<mask>_<variant>, one small tile, its top 26 px above
the ground) is cut at its plain face rows (where the silhouette's sides run straight down) and those rows are stacked
again and again, each tier taken from the same mask's next variant, so the top and the foot stay the original picture
and only the rock face grows. Writes assets_candidate/tiles/tcliff_<mask>_<variant>.png (scene.json height 2's faces)
and their tiles.json entries (only these); ver5/tools/build_extra.py packs them. The same blocks are also written
unstacked as ccliff_<mask>_<variant> (scene.json height 7's faces, a hill of the battle map's own 26 px).
"""
import json
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[3]
CAND = ROOT / 'assets_candidate'
GFX = ROOT / 'ver5/assets/graphics'
BAND = (21, 31)  # the block's plain face rows: below its top diamond, above its foot's slant
TIERS = 12       # extra bands: the top stands 26 + 120 px up, a little above the scene gate's roof


def blocks():
    st = json.load(open(GFX / 'index.json'))['structures']
    sheet = Image.open(GFX / st['image']).convert('RGBA')
    w, h = st['cell']
    cols = sheet.width // w
    out = {}
    for name, i in st['sprites'].items():
        if name.startswith('cliff_'):
            out[name] = sheet.crop(((i % cols) * w, (i // cols) * h, (i % cols) * w + w, (i // cols) * h + h))
    return out, st['pivot']


def tall(base, others, tiers=TIERS):
    a, b = BAND
    band = b - a
    im = Image.new('RGBA', (base.width, base.height + band * tiers))
    im.paste(base.crop((0, 0, base.width, b)), (0, 0))
    for k in range(tiers):
        src = others[k % len(others)]
        im.paste(src.crop((0, a, base.width, b)), (0, b + band * k))
    im.paste(base.crop((0, b, base.width, base.height)), (0, b + band * tiers))  # retain the entire original foot
    return im


def main():
    cells, pivot = blocks()
    path = CAND / 'tiles/tiles.json'
    made = {}
    for name, im in cells.items():
        _, mask, var = name.split('_')
        others = [cells[f'cliff_{mask}_{(int(var) + d) % 3}'] for d in (1, 2, 0)]
        t = tall(im, others)
        n = f'tcliff_{mask}_{var}'
        t.save(CAND / 'tiles' / f'{n}.png')
        made[n] = dict(file=f'tiles/{n}.png', size=list(t.size), pivot=[pivot[0], pivot[1] + (BAND[1] - BAND[0]) * TIERS],
                       layer='object', scale='scene', made='battle cliff block, face rows stacked (tall_cliff.py)', shadow=False)
        n = f'ccliff_{mask}_{var}'
        im.save(CAND / 'tiles' / f'{n}.png')
        made[n] = dict(file=f'tiles/{n}.png', size=list(im.size), pivot=list(pivot), layer='object', scale='scene',
                       made='battle cliff block as it is (tall_cliff.py)', shadow=False)
    man = json.load(open(path))  # read just before writing: other tools write this file too
    man['sprites'].update(made)
    json.dump(man, open(path, 'w'), ensure_ascii=False, indent=1)
    open(path, 'a').write('\n')
    print(len(made), 'tall cliffs', made[next(iter(made))]['size'])


if __name__ == '__main__':
    main()
