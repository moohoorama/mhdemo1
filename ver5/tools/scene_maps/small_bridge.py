"""The scene's small bridge decks, one small tile each: the battle map's bridge_u / bridge_v at half size, as the old
scene maps drew them over their `b` tiles (internal/gui nativeProp, scale 0.5, nearest). The scene map rules lay
them as objects (tools/scene_maps/scene_v2.py puts one on each `b` tile).

  python3 ver5/tools/scene_maps/small_bridge.py    # sbridge_u, sbridge_v into assets_candidate/tiles and tiles.json
then `python3 ver5/tools/build_extra.py`.
"""
import json
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[3]
GFX = ROOT / 'ver5/assets/graphics'
TILES = ROOT / 'assets_candidate/tiles'


def main():
    index = json.load(open(GFX / 'index.json'))
    sheet = Image.open(GFX / index['props']['image']).convert('RGBA')
    man = json.load(open(TILES / 'tiles.json'))
    for axis in ('u', 'v'):
        sp = index['props']['sprites'][f'bridge_{axis}']
        x, y, w, h = sp['rect']
        im = sheet.crop((x, y, x + w, y + h))
        small = im.resize(((w + 1) // 2, (h + 1) // 2), Image.NEAREST)
        name = f'sbridge_{axis}'
        small.save(TILES / f'{name}.png')
        man['sprites'][name] = dict(file=f'tiles/{name}.png', size=list(small.size),
                                    pivot=[sp['pivot'][0] // 2, sp['pivot'][1] // 2], layer='ground', scale='scene',
                                    made='bridge_' + axis + ' at half size (tools/scene_maps/small_bridge.py)', shadow=False)
    json.dump(man, open(TILES / 'tiles.json', 'w'), ensure_ascii=False, indent=1)
    open(TILES / 'tiles.json', 'a').write('\n')


if __name__ == '__main__':
    main()
