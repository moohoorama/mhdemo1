#!/usr/bin/env python3
"""Assemble ver4/assets/graphics: one image per unit (+ drop shadow), the map tileset, the
castle/village structures and the draw data of every campaign battle map, all described by
index.json for the Go GUI. Requires Pillow, PyYAML (ver3 shadow cache) and Go.

Sources (read only):
  tools/spritetool/assets/ver2-units/        unit sheets (python3 tools/unit3d/build.py --full)
  tools/spritetool/assets/ver4-civilians/    scene sprites, packed as civ_<key> (tools/unit3d/civilian_build.py)
  ver4/assets/scenes.json                    scenario scene maps (exported like battle maps)
  tools/spritetool/assets/ver4-structures/   village, castle floor and walls (tools/unit3d/structures.py)
  assets/terrain.png, objects.png, catalog.json   terrain tiles and decorations (make assets)
  ver4/assets/content/campaign.json          battle maps (Tiles)

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
import build_assets as v3  # noqa: E402  (ver3 packer: unit sheets, tileset)
import civilian  # noqa: E402
import shadows  # noqa: E402

CIVILIANS = ROOT / 'tools/spritetool/assets/ver4-civilians'

STRUCTURES = ROOT / 'tools/spritetool/assets/ver4-structures'
# Rules tile -> editor ground (0 river, 1 grass, 2 wasteland) and decoration (1 rocks, 2 trees).
EDITOR = {'.': (1, 0), 'd': (2, 0), '~': (0, 0), 'f': (1, 2), 's': (2, 1), 'v': (2, 0), 'c': (2, 0), 'i': (2, 0)}


def build_map(stage):
    """Draw data for one battle: ground layers and decorations from the editor's terrain rules,
    plus the rules tiles (the GUI places villages, castle floor and walls from them)."""
    w, h = stage['Width'], stage['Height']
    cells = [dict(ground=EDITOR[c][0], decoration=EDITOR[c][1]) for row in stage['Tiles'] for c in row]
    with tempfile.TemporaryDirectory() as tmp:
        src = Path(tmp) / f"{stage['ID']}.json"
        src.write_text(json.dumps(dict(version=5, width=w, height=h, cells=cells)))
        data = json.loads(subprocess.run(['go', 'run', './ver3/tools/exportmap', str(src)], cwd=ROOT,
                                         check=True, capture_output=True, text=True).stdout)
    out = dict(id=stage['ID'], name=stage['Name'], size=[w, h], tiles=stage['Tiles'], bounds=data['Bounds'],
               ground=[[g['X'], g['Y'], g['Layers']] for g in data['Ground']],
               decorations=[[d['X'], d['Y'], d['Animation'], d['Phase']] for d in data['Decorations']])
    path = OUT / 'maps' / f"{stage['ID']}.json"
    path.write_text(json.dumps(out, ensure_ascii=False, separators=(',', ':')) + '\n')
    print('map', path.relative_to(ROOT))
    return f"maps/{stage['ID']}.json"


def main():
    for d in ('units', 'map', 'maps'):
        (OUT / d).mkdir(parents=True, exist_ok=True)
    v3.ASSET = OUT
    content = json.loads((ROOT / 'ver4/assets/content/campaign.json').read_text())
    scenes = json.loads((ROOT / 'ver4/assets/scenes.json').read_text())
    units = v3.build_units()
    units.update(v3.build_units([(k, t) for k, t, _, _ in civilian.chosen()], CIVILIANS, prefix='civ_'))
    index = dict(version=1, units=units, tileset=v3.build_tileset(),
                 factions=json.loads((v3.UNITS_SRC / 'factions.json').read_text()),
                 maps=[build_map(s) for s in content['Stages'] + scenes['maps']])
    shutil.copy2(STRUCTURES / 'structures.png', OUT / 'map/structures.png')
    structures = json.loads((STRUCTURES / 'structures.json').read_text())
    structures['image'] = 'map/structures.png'
    index['structures'] = structures
    index['shadows'] = shadows.refresh(index, OUT)  # also caches hashes in shadows.yaml
    index['shadow_rgb'] = [18, 26, 14]  # same dark olive as ver3/assets.py SHADOW_RGB
    (OUT / 'index.json').write_text(json.dumps(index, ensure_ascii=False, indent=1) + '\n')
    print('wrote', (OUT / 'index.json').relative_to(ROOT))


if __name__ == '__main__':
    main()
