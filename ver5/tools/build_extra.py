#!/usr/bin/env python3
"""Pack the candidate pictures the game uses (assets_candidate/) into assets/graphics.

  python3 ver5/tools/build_extra.py

Every sprite of assets_candidate/tiles/tiles.json except the old pillar gate (gate_*) and the forest-cover idea
(cover_bush_*), and the hay hut of structures.png (as "hut", for the scenario scenes), goes into one sheet, assets/graphics/map/extra.png, and is listed in assets/graphics/index.json under
"extra" the way "props" is (rect, pivot, layer). The flame and smoke strips of assets_candidate/effects are copied to
assets/graphics/effects/ and listed under "effects". The game reads both lists into its props and flame tables.
Requires Pillow."""
import json
import shutil
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
CAND = ROOT / 'assets_candidate'
GFX = ROOT / 'ver5/assets/graphics'
SKIP = ('gate_', 'cover_bush_')
WIDTH = 1024


def main():
    tiles = json.load(open(CAND / 'tiles/tiles.json'))['sprites']
    names = sorted(n for n in tiles if not n.startswith(SKIP))
    ims = {n: Image.open(CAND / tiles[n]['file']).convert('RGBA') for n in names}
    structures = json.load(open(GFX / 'index.json'))['structures']
    w, h = structures['cell']
    ims['hut'] = Image.open(GFX / structures['image']).convert('RGBA').crop((0, 0, w, h))
    tiles['hut'] = {'pivot': structures['pivot'], 'layer': structures['layer']['village']}
    names.append('hut')
    order = sorted(names, key=lambda n: (-ims[n].height, n))
    x = y = row = 0
    place = {}
    for n in order:
        w, h = ims[n].size
        if x + w > WIDTH:
            x, y, row = 0, y + row + 1, 0
        place[n] = (x, y)
        x += w + 1
        row = max(row, h)
    sheet = Image.new('RGBA', (WIDTH, y + row + 1), (0, 0, 0, 0))
    sprites = {}
    for n in names:
        sheet.paste(ims[n], place[n])
        w, h = ims[n].size
        sprites[n] = {'rect': [*place[n], w, h], 'pivot': tiles[n]['pivot'], 'layer': tiles[n]['layer']}
    sheet.save(GFX / 'map/extra.png', optimize=True)

    effects = json.load(open(CAND / 'effects/effects.json'))
    out = {}
    for n, e in effects.items():
        shutil.copyfile(CAND / e['file'], GFX / 'effects' / f'{n}.png')
        out[n] = {'image': f'effects/{n}.png', 'frame': e['frame'], 'frames': e['frames'], 'ms': e.get('ms', 0), 'pivot': e['pivot']}

    path = GFX / 'index.json'
    index = json.load(open(path))
    index['extra'] = {'version': 1, 'image': 'map/extra.png', 'sprites': sprites}
    index['effects'] = out
    json.dump(index, open(path, 'w'), ensure_ascii=False, indent=1)
    open(path, 'a').write('\n')
    print('extra', len(sprites), 'sprites', sheet.size, 'effects', len(out))


if __name__ == '__main__':
    main()
