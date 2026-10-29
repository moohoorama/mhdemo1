#!/usr/bin/env python3
"""Pack the candidate pictures the game uses (assets_candidate/) into assets/graphics.

  python3 ver5/tools/build_extra.py

Every sprite of assets_candidate/tiles/tiles.json except the old pillar gate (gate_*) and the forest-cover idea
(cover_bush_*), and the hay hut of structures.png (as "hut", for the scenario scenes), goes into one sheet, assets/graphics/map/extra.png, and is listed in assets/graphics/index.json under
"extra" the way "props" is (rect, pivot, layer). A sprite marked "frame": 128 is drawn in a 128 x 128 frame centred on
its small tile (docs/plan-tile-floors.md): it goes in trimmed to what it paints, its pivot moved to match. The flame and smoke strips of assets_candidate/effects are copied to
assets/graphics/effects/, and the looping strips of assets_candidate/anims/anims.json (ASCII frames laid over a still
tile, e.g. the smoke over the incense burner) are drawn there; both are listed under "anims", the game's one table of
looping strips (docs/presentation-scene.md). The game reads "extra" into its props and "anims" into Assets.Anims.
Requires Pillow."""
import json
import shutil
from pathlib import Path

from PIL import Image, ImageColor

ROOT = Path(__file__).resolve().parents[2]
CAND = ROOT / 'assets_candidate'
GFX = ROOT / 'ver5/assets/graphics'
SKIP = ('gate_', 'cover_bush_')
WIDTH = 1024


def main():
    tiles = json.load(open(CAND / 'tiles/tiles.json'))['sprites']
    names = sorted(n for n in tiles if not n.startswith(SKIP))
    ims = {n: Image.open(CAND / tiles[n]['file']).convert('RGBA') for n in names}
    pivots = {n: list(tiles[n]['pivot']) for n in names}
    for n in names:  # a piece drawn in a 128 frame goes in trimmed, its pivot moved by what was cut away
        if tiles[n].get('frame'):
            box = ims[n].getbbox() or (0, 0, 1, 1)
            ims[n] = ims[n].crop(box)
            pivots[n] = [pivots[n][0] - box[0], pivots[n][1] - box[1]]
    structures = json.load(open(GFX / 'index.json'))['structures']
    w, h = structures['cell']
    ims['hut'] = Image.open(GFX / structures['image']).convert('RGBA').crop((0, 0, w, h))
    tiles['hut'] = {'pivot': structures['pivot'], 'layer': structures['layer']['village']}
    pivots['hut'] = structures['pivot']
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
        sprites[n] = {'rect': [*place[n], w, h], 'pivot': pivots[n], 'layer': tiles[n]['layer']}
        if tiles[n].get('shadow') is False:  # no ground shadow under it in a story scene (shadows, overlays, cliffs)
            sprites[n]['shadow'] = False
        if 'gate_offset' in tiles[n]:  # a scene gatehouse slice: game cells along its wall from the gate's point
            sprites[n]['gate_offset'] = tiles[n]['gate_offset']
    sheet.save(GFX / 'map/extra.png', optimize=True)

    effects = json.load(open(CAND / 'effects/effects.json'))
    out = {}
    for n, e in effects.items():
        shutil.copyfile(CAND / e['file'], GFX / 'effects' / f'{n}.png')
        out[n] = {'image': f'effects/{n}.png', 'rect': [0, 0, *e['frame']], 'frames': e['frames'], 'ms': e.get('ms', 0),
                  'pivot': e['pivot']}
    for n, a in json.load(open(CAND / 'anims/anims.json')).items():
        if n == 'note':
            continue
        strip = drawn_strip(a, Image.open(CAND / tiles[a['base']]['file']).convert('RGBA'))
        strip.save(GFX / 'effects' / f'{n}.png', optimize=True)
        px, py = tiles[a['base']]['pivot']
        out[n] = {'image': f'effects/{n}.png', 'rect': [0, 0, strip.width // len(a['frames']), strip.height],
                  'frames': len(a['frames']), 'ms': a['ms'], 'pivot': [px, py + a['lift']]}

    path = GFX / 'index.json'
    index = json.load(open(path))
    index['extra'] = {'version': 1, 'image': 'map/extra.png', 'sprites': sprites}
    anims = {n: e for n, e in index.get('anims', {}).items() if not e['image'].startswith('effects/')}  # others' strips
    index['anims'] = dict(sorted({**anims, **out}.items()))
    json.dump(index, open(path, 'w'), ensure_ascii=False, indent=1)
    open(path, 'a').write('\n')
    print('extra', len(sprites), 'sprites', sheet.size, 'anims', len(out))


def drawn_strip(a, base):
    """The frames of an anims.json entry side by side: the still tile lifted by a['lift'] rows, its ASCII frame over it
    (palette colour at the character's alpha)."""
    w, h = base.width, base.height + a['lift']
    strip = Image.new('RGBA', (w * len(a['frames']), h))
    for i, rows in enumerate(a['frames']):
        assert len(rows) <= h and all(len(r) == w for r in rows), f'frame {i}: rows must be {w} wide, at most {h}'
        frame = Image.new('RGBA', (w, h))
        frame.paste(base, (0, a['lift']))
        over = Image.new('RGBA', (w, h))
        for y, row in enumerate(rows):
            for x, c in enumerate(row):
                if c != '.':
                    over.putpixel((x, y), ImageColor.getrgb(a['palette'][c]) + (a['alpha'][c],))
        strip.paste(Image.alpha_composite(frame, over), (i * w, 0))
    return strip


if __name__ == '__main__':
    main()
