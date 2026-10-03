#!/usr/bin/env python3
"""Assemble ver3/asset: one image per character, one tileset image for the map,
the map draw data, and assets.yaml describing every image. Requires Pillow, PyYAML, Go.

Sources (not modified):
  tools/spritetool/assets/ver2-units/   unit sheets per direction + frames.json + factions.json
  assets/terrain.png, assets/objects.png, assets/catalog.json   terrain tiles and decorations
  ver3/asset/maps/*.json                 maps in the editor format (go run . -map FILE)

  python3 ver3/tools/build_assets.py
"""
import json
import subprocess
import sys
from pathlib import Path

import yaml
from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
ASSET = ROOT / 'ver3/asset'
UNITS_SRC = ROOT / 'tools/spritetool/assets/ver2-units'
sys.path.insert(0, str(ROOT / 'tools/unit3d'))
import build as unit_build  # noqa: E402  (unit keys, titles and order)

CELL_LEGEND = {'.': 'grass', ':': 'wasteland', '~': 'river', 'R': 'rocks', 'T': 'trees'}


class Dumper(yaml.SafeDumper):
    """Short lists and small records on one line; grids and long tables one item per line."""


def _flat(v):
    return v is None or isinstance(v, (str, int, float, bool))


def _shallow(v):
    return _flat(v) or (isinstance(v, list) and all(_flat(x) for x in v))


def _list(dumper, data):
    grid = len(data) > 4 and all(isinstance(x, str) and len(x) > 12 for x in data)  # map rows
    flow = all(_shallow(x) for x in data) and not grid
    return dumper.represent_sequence('tag:yaml.org,2002:seq', data, flow_style=flow)


def _dict(dumper, data):
    flow = len(data) <= 6 and all(_shallow(v) for v in data.values())
    return dumper.represent_mapping('tag:yaml.org,2002:map', data.items(), flow_style=flow)


Dumper.add_representer(list, _list)
Dumper.add_representer(dict, _dict)


def dump(data):
    return yaml.dump(data, Dumper=Dumper, allow_unicode=True, sort_keys=False, width=140)


def build_units():
    """Each character becomes one image: a row per facing, 5 animations x 4 frames per row."""
    entries = {}
    for key, title, _, _ in unit_build.UNITS:
        meta = json.loads((UNITS_SRC / key / 'frames.json').read_text())
        w, h = meta['cell']
        directions, animations = meta['directions'], meta['animations']
        counts = {a: sum(1 for f in meta['frames'] if f['direction'] == directions[0] and f['animation'] == a)
                  for a in animations}
        columns = sum(counts.values())
        sheet = Image.new('RGBA', (w*columns, h*len(directions)))
        first = {}
        col = 0
        for a in animations:
            first[a] = col
            col += counts[a]
        for row, d in enumerate(directions):
            src = Image.open(UNITS_SRC / key / f'{d}-pixel.png')
            for f in meta['frames']:
                if f['direction'] != d:
                    continue
                x, y, cw, ch = f['crop']
                sheet.paste(src.crop((x, y, x + cw, y + ch)), ((first[f['animation']] + f['frame'])*w, row*h))
        path = f'units/{key}.png'
        sheet.save(ASSET / path)
        anims = {}
        for a in animations:
            ms = meta['frame_durations_ms'][a]
            anims[a] = dict(first_column=first[a], frames=counts[a],
                            ms=ms if isinstance(ms, list) else [ms]*counts[a],
                            loop=a in ('idle', 'walk', 'exhausted'))
        entries[key] = dict(name=title, image=path, cell=[w, h], pivot=meta['pivot'], rows=directions,
                            animations=anims)
        print('unit', key, sheet.size)
    return entries


def build_tileset():
    """Terrain tiles (16x8) on top, decorations (40x48) below, in one image."""
    catalog = json.loads((ROOT / 'assets/catalog.json').read_text())
    sheets = [Image.open(ROOT / 'assets' / s['file']).convert('RGBA') for s in catalog['sheets']]
    offsets, height = [], 0
    for im in sheets:
        offsets.append(height)
        height += im.height
    tileset = Image.new('RGBA', (max(im.width for im in sheets), height))
    for im, oy in zip(sheets, offsets):
        tileset.paste(im, (0, oy))
    tileset.save(ASSET / 'map/tileset.png')
    sprites = {}
    for i, s in enumerate(catalog['sprites']):
        sh = catalog['sheets'][s['sheet']]
        cw, ch, cols = sh['cell_width'], sh['cell_height'], sh['columns']
        x, y = s['cell'] % cols*cw, offsets[s['sheet']] + s['cell']//cols*ch
        sprites[i] = [x, y, cw, ch, s['pivot']['X'], s['pivot']['Y']]
    tick = 125  # ms per editor tick at 1x
    animations = {name: dict(frames=a['frames'], ms=a['frame_ticks']*tick, loop=a['loop'])
                  for name, a in sorted(catalog['animations'].items())}
    print('tileset', tileset.size)
    return dict(image='map/tileset.png', sprites=sprites, animations=animations,
                ground=dict(water='0-7 (animation water)', wasteland='8-22 (8 = full, 8+mask = edge)',
                            grass='23-37 (23 = full, 23+mask = edge)'),
                decorations='38-60 (see animations: rock_*, grass_*, tree_*)')


def build_map(source):
    data = json.loads(subprocess.run(['go', 'run', './ver3/tools/exportmap', str(source)], cwd=ROOT,
                                     check=True, capture_output=True, text=True).stdout)
    w, h = data['Width'], data['Height']
    rows = []
    for y in range(h):
        row = ''
        for x in range(w):
            ground, decoration = data['Cells'][y*w + x]
            row += {1: 'R', 2: 'T'}.get(decoration) or {0: '~', 1: '.', 2: ':'}[ground]
        rows.append(row)
    out = dict(
        name=source.stem, source=source.name, size=[w, h],
        projection='screen_x = (u - v) * 16, screen_y = (u + v) * 8; a cell (u, v) is a 32x16 diamond, '
                   'its centre is at (u + .5, v + .5)',
        bounds=data['Bounds'],
        legend=CELL_LEGEND, walkable='grass, wasteland, rocks', cells=rows,
        ground_note='ground: [x, y, [tileset sprite ids drawn in order]]; id 0 = current water frame',
        ground=[[g['X'], g['Y'], g['Layers']] for g in data['Ground']],
        decoration_note='decorations: [x, y, tileset animation, phase in editor ticks], drawn depth-sorted with units',
        decorations=[[d['X'], d['Y'], d['Animation'], d['Phase']] for d in data['Decorations']])
    path = source.with_suffix('.yaml')
    path.write_text(dump(out))
    print('map', path.relative_to(ROOT))
    return str(path.relative_to(ASSET))


HEADER = """\
# ver3 asset index. Every image under ver3/asset is described here.
#
# units.<key>: one image per character.
#   row  = facing, in the order of `rows` (screen directions; SW = facing down-left)
#   column = animations[a].first_column + frame
#   frame rect = (column * cell[0], row * cell[1], cell[0], cell[1])
#   draw so that `pivot` (the feet, ground contact) lands on the unit's map position.
#   Team-colored pixels use factions.team_keys; replace them with a faction's ramp.
# tileset: one image for the map. sprites[id] = [x, y, w, h, pivot_x, pivot_y].
# maps: map draw data (ground layers, decorations, cell grid) built from the editor maps.
"""


def main():
    for d in ('units', 'map', 'maps'):
        (ASSET / d).mkdir(parents=True, exist_ok=True)
    index = dict(version=1, units=build_units(), tileset=build_tileset(),
                 factions=json.loads((UNITS_SRC / 'factions.json').read_text()),
                 maps=[build_map(p) for p in sorted((ASSET / 'maps').glob('*.json'))])
    (ASSET / 'assets.yaml').write_text(
        HEADER + dump(index))
    print('wrote', (ASSET / 'assets.yaml').relative_to(ROOT))


if __name__ == '__main__':
    main()
