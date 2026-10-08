#!/usr/bin/env python3
"""Assemble ver4/assets/graphics: one image per unit (+ drop shadow), the map tileset, the
castle/village structures and the draw data of every campaign battle map, all described by
index.json for the Go GUI. Requires Pillow, PyYAML (ver3 shadow cache) and Go.

Sources (read only):
  tools/spritetool/assets/ver2-units/        unit sheets (python3 tools/unit3d/build.py --full)
  tools/spritetool/assets/ver4-civilians/    scene sprites, packed as civ_<key> (tools/unit3d/civilian_build.py)
  ver4/assets/scenes.json                    scenario scene maps (exported like battle maps)
  tools/spritetool/assets/ver4-structures/   village, castle floor and walls (tools/unit3d/structures.py)
  tools/spritetool/assets/ver4-props/        scene props and tiles at character scale (tools/unit3d/props.py)
  assets/terrain.png, objects.png, catalog.json   terrain tiles and decorations (make assets)
  ver4/assets/content/campaign.json          battle maps (Tiles)
  ver4/assets/content/simulator-maps.json    battle simulator terrain maps (Tiles, Art)
  tools/illustrations/portraits/             ink-wash portraits, packed as portraits/<key>.png
  ver4/tools/doteditor/units/*.yaml          dot editor sheets, replacing the unit of the same id

  python3 ver4/tools/build_assets.py
"""
import json
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
OUT = ROOT / 'ver4/assets/graphics'
sys.path.insert(0, str(ROOT / 'ver3/tools'))
sys.path.insert(0, str(ROOT / 'ver3'))
sys.path.insert(0, str(ROOT / 'tools/unit3d'))
sys.path.insert(0, str(ROOT / 'ver4/tools/doteditor'))
import apply as doteditor  # noqa: E402
import build_assets as v3  # noqa: E402  (ver3 packer: unit sheets, tileset)
import civilian  # noqa: E402
from PIL import Image  # noqa: E402
import shadows  # noqa: E402

CIVILIANS = ROOT / 'tools/spritetool/assets/ver4-civilians'
PORTRAITS = ROOT / 'tools/illustrations/portraits'
LIUBEI_PORTRAIT = ROOT / 'tools/illustrations/style-samples/4-ink-wash.png'  # the style reference is his portrait
PORTRAIT_SIZE = 256

STRUCTURES = ROOT / 'tools/spritetool/assets/ver4-structures'
PROPS = ROOT / 'tools/spritetool/assets/ver4-props'
# Rules tile -> editor ground (0 river, 1 grass, 2 wasteland) and decoration (1 rocks, 2 trees).
EDITOR = {'.': (1, 0), 'd': (2, 0), '~': (0, 0), 'f': (1, 2), 's': (2, 1), 'v': (2, 0), 'c': (2, 0), 'i': (2, 0),
          'b': (0, 0), 'g': (2, 0), 'k': (1, 0)}  # bridge: water under the deck; gate: like a wall cell
# Battle maps draw forests and mountains as deformed cells (structures forest, mountain), like villages;
# the big trees and rocks stay for scenario stages.
BATTLE = dict(EDITOR, f=(1, 0), s=(2, 0))


def build_map(stage, battle):
    """Draw data for one map: ground layers and decorations from the editor's terrain rules, plus the
    rules tiles (the GUI places villages, castle floor and walls from them, and on battle maps the
    deformed forest and mountain cells)."""
    w, h = stage['Width'], stage['Height']
    rules = BATTLE if battle else EDITOR
    # Art, when given, is drawn in place of Tiles (the simulator's forest keeps its units visible)
    cells = [dict(ground=rules[c][0], decoration=rules[c][1]) for row in stage.get('Art', stage['Tiles']) for c in row]
    with tempfile.TemporaryDirectory() as tmp:
        src = Path(tmp) / f"{stage['ID']}.json"
        src.write_text(json.dumps(dict(version=5, width=w, height=h, cells=cells)))
        data = json.loads(subprocess.run(['go', 'run', './ver3/tools/exportmap', str(src)], cwd=ROOT,
                                         check=True, capture_output=True, text=True).stdout)
    out = dict(id=stage['ID'], name=stage['Name'], size=[w, h], tiles=stage['Tiles'], bounds=data['Bounds'],
               ground=[[g['X'], g['Y'], g['Layers']] for g in data['Ground']],
               decorations=[[d['X'], d['Y'], d['Animation'], d['Phase']] for d in data['Decorations'] or []],
               props=stage.get('Props', []), deformed=battle)
    path = OUT / 'maps' / f"{stage['ID']}.json"
    path.write_text(json.dumps(out, ensure_ascii=False, separators=(',', ':')) + '\n')
    print('map', path.relative_to(ROOT))
    return f"maps/{stage['ID']}.json"


def add_mirrored_rows(entry):
    """Scene sprites are drawn facing NW and SW only; NE and SE are those rows flipped. The cells
    are first widened to be symmetric about the pivot so a flipped row keeps the same pivot."""
    if entry['rows'] != ['NW', 'SW']:
        return
    path = OUT / entry['image']
    sheet = Image.open(path).convert('RGBA')
    (w, h), (px, py) = entry['cell'], entry['pivot']
    half = max(px, w - px)
    cols = sheet.width // w
    out = Image.new('RGBA', (2*half*cols, h*4))
    for row in range(2):
        for col in range(cols):
            cell = sheet.crop((col*w, row*h, (col + 1)*w, (row + 1)*h))
            x = col*2*half + half - px
            out.paste(cell, (x, row*h))
            flipped = Image.new('RGBA', (2*half, h))
            flipped.paste(cell, (half - px, 0))
            out.paste(flipped.transpose(Image.FLIP_LEFT_RIGHT), (col*2*half, (row + 2)*h))
    out.save(path)
    entry.update(cell=[2*half, h], pivot=[half, py], rows=['NW', 'SW', 'NE', 'SE'])


def build_portraits():
    """Officer portraits scaled to PORTRAIT_SIZE: {key: path}."""
    sources = {p.stem: p for p in sorted(PORTRAITS.glob('*.png'))}
    sources['liubei'] = LIUBEI_PORTRAIT
    out = {}
    for key, src in sorted(sources.items()):
        path = f'portraits/{key}.png'
        Image.open(src).convert('RGB').resize((PORTRAIT_SIZE, PORTRAIT_SIZE), Image.LANCZOS).save(OUT / path)
        out[key] = path
    print('portraits', len(out))
    return out


def main():
    for d in ('units', 'map', 'maps', 'portraits'):
        (OUT / d).mkdir(parents=True, exist_ok=True)
    v3.ASSET = OUT
    content = json.loads((ROOT / 'ver4/assets/content/campaign.json').read_text())
    scenes = json.loads((ROOT / 'ver4/assets/scenes.json').read_text())
    simulator = json.loads((ROOT / 'ver4/assets/content/simulator-maps.json').read_text())
    units = v3.build_units()
    doteditor.apply_all(units, OUT)
    civilians = v3.build_units([(k, t) for k, t, _, _ in civilian.chosen()], CIVILIANS, prefix='civ_')
    for entry in civilians.values():
        add_mirrored_rows(entry)
    units.update(civilians)
    index = dict(version=1, units=units, tileset=v3.build_tileset(),
                 factions=json.loads((v3.UNITS_SRC / 'factions.json').read_text()),
                 maps=[build_map(s, True) for s in content['Stages']] + [build_map(s, False) for s in scenes['maps']]
                 + [build_map(s, True) for s in simulator], portraits=build_portraits())
    shutil.copy2(STRUCTURES / 'structures.png', OUT / 'map/structures.png')
    structures = json.loads((STRUCTURES / 'structures.json').read_text())
    structures['image'] = 'map/structures.png'
    index['structures'] = structures
    shutil.copy2(PROPS / 'props.png', OUT / 'map/props.png')
    props = json.loads((PROPS / 'props.json').read_text())
    props['image'] = 'map/props.png'
    index['props'] = props
    index['shadows'] = shadows.refresh(index, OUT)  # also caches hashes in shadows.yaml
    index['shadow_rgb'] = [18, 26, 14]  # same dark olive as ver3/assets.py SHADOW_RGB
    (OUT / 'index.json').write_text(json.dumps(index, ensure_ascii=False, indent=1) + '\n')
    print('wrote', (OUT / 'index.json').relative_to(ROOT))


if __name__ == '__main__':
    main()
