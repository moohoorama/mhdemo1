#!/usr/bin/env python3
"""Tall scene cliffs made from the battle map's own cliff blocks: python3 ver5/tools/scene_maps/tall_cliff.py

A battle cliff block (assets/graphics/map/structures.png, cliff_<mask>_<variant>, one small tile, its top 26 px above
the ground) is cut at its plain face rows (where the silhouette's sides run straight down) and those rows are stacked
again and again, each tier taken from the same mask's next variant, so the top and the foot stay the original picture
and only the rock face grows. Writes assets_candidate/tiles/tcliff_<mask>_<variant>.png and lcliff_<mask>_<variant>
(the block unchanged, sorted later) and their tiles.json entries (only these); ver5/tools/build_extra.py packs them. compose.tall_cliffs() turns a map's cliff tiles into them.
"""
import json
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[3]
CAND = ROOT / 'assets_candidate'
GFX = ROOT / 'ver5/assets/graphics'
BAND = (21, 31)  # the block's plain face rows: below its top diamond, above its foot's slant
TIERS = 12       # extra bands: the top stands 26 + 120 px up, a little above the scene gate's roof
LATE = 16        # lcliff: px the pivot sits below the block's own (its anchor half a cell further along u and v)


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
        # the same block at its own height, sorted half a cell later: the lip of a low cliff in front that must draw
        # over what stands right behind it (a gatehouse post), placed by compose.tall_cliffs(..., kind='lcliff')
        n = f'lcliff_{mask}_{var}'
        im.save(CAND / 'tiles' / f'{n}.png')
        made[n] = dict(file=f'tiles/{n}.png', size=list(im.size), pivot=[pivot[0], pivot[1] + LATE], layer='object',
                       scale='scene', made='battle cliff block, sorted later (tall_cliff.py)', shadow=False)
    # the scene wall pieces raised onto a cliff block's top (the wall running on over a low cliff), pivot a cliff rise low
    rise = json.load(open(GFX / 'index.json'))['structures']['cliff_rise']
    for axis in 'uv':
        src = json.load(open(path))['sprites'][f'swall_{axis}']
        n = f'swall_{axis}_up'
        Image.open(CAND / src['file']).save(CAND / 'tiles' / f'{n}.png')
        made[n] = dict(file=f'tiles/{n}.png', size=src['size'], pivot=[src['pivot'][0], src['pivot'][1] + rise], layer='object',
                       scale='scene', made='scene wall piece standing on a cliff top (tall_cliff.py)', shadow=False)
    man = json.load(open(path))  # read just before writing: other tools write this file too
    man['sprites'].update(made)
    json.dump(man, open(path, 'w'), ensure_ascii=False, indent=1)
    open(path, 'a').write('\n')
    print(len(made), 'tall cliffs', made[next(iter(made))]['size'])


if __name__ == '__main__':
    main()
