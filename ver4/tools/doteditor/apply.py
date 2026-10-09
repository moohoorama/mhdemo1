#!/usr/bin/env python3
"""Apply the dot editor's units/*.yaml to ver4/assets/graphics: each file replaces the packed sheet
of the unit with the same id, then index.json and the drop shadows are updated.
build_assets.py applies them too, so a full rebuild keeps the edits. A <hero>_noncombat file (NW·SW only)
is expanded to four directions and a four-step walk (expand_noncombat). Requires Pillow and PyYAML.

  make apply                  # all units/*.yaml
  python3 apply.py units/infantry.yaml
"""
import json
import sys
from pathlib import Path

import yaml
from PIL import Image

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]
OUT = ROOT / 'ver4/assets/graphics'
UNITS = HERE / 'units'


def expand_noncombat(doc):
    """A <hero>_noncombat file holds NW·SW idle, two walk steps and one action. The sheet gets NE·SE as
    mirrored rows (cell widened to be symmetric about the pivot) and walk as idle, left, idle, right."""
    u = doc['unit']
    (w, h), (px, py) = u['cell'], u['pivot']
    half = max(px, w - px)
    pad = lambda row: [0]*(half - px) + row + [0]*(half - (w - px))
    src = {(f['direction'], f['animation'], f['frame']): [pad(r) for r in f['pixels']] for f in doc['frames']}
    walk = u['animations']['walk']['ms']
    plan = {'idle': [('idle', 0)], 'walk': [('idle', 0), ('walk', 0), ('idle', 0), ('walk', 1)], 'action': [('action', 0)]}
    ms = {'idle': u['animations']['idle']['ms'], 'walk': [walk[0], walk[0], walk[1], walk[1]],
          'action': u['animations']['action']['ms']}
    frames, anims, col = [], {}, 0
    for name, cells in plan.items():
        anims[name] = dict(first_column=col, frames=len(cells), ms=ms[name], loop=u['animations'][name]['loop'])
        col += len(cells)
        for d, mirror in (('NW', None), ('SW', None), ('NE', 'NW'), ('SE', 'SW')):
            for k, (a, n) in enumerate(cells):
                pixels = src[(mirror or d, a, n)]
                frames.append(dict(direction=d, animation=name, frame=k,
                                   pixels=[r[::-1] for r in pixels] if mirror else pixels))
    doc['frames'] = frames
    u.update(cell=[2*half, h], pivot=[half, py], rows=['NW', 'SW', 'NE', 'SE'], animations=anims)
    return doc


def render(path, out=OUT):
    """Draw one editor YAML as units/<id>.png under out; returns (id, index entry)."""
    doc = yaml.safe_load(path.read_text())
    if doc.get('format') != 'mhdemo-unit-pixels':
        raise SystemExit(f'{path}: not a dot editor unit file')
    if doc['unit']['id'].endswith('_noncombat'):
        doc = expand_noncombat(doc)
    u = doc['unit']
    (w, h), rows, anims = u['cell'], u['rows'], u['animations']
    palette = [bytes.fromhex(c[1:]) for c in doc['palette']]
    columns = max(a['first_column'] + a['frames'] for a in anims.values())
    sheet = Image.new('RGBA', (w*columns, h*len(rows)))
    for f in doc['frames']:
        cell = Image.frombytes('RGBA', (w, h), b''.join(palette[i] for row in f['pixels'] for i in row))
        sheet.paste(cell, ((anims[f['animation']]['first_column'] + f['frame'])*w, rows.index(f['direction'])*h))
    image = f"units/{u['id']}.png"
    sheet.save(out / image)
    print('dot', path.relative_to(ROOT), '->', image)
    return u['id'], dict(name=u['name'], image=image, cell=[w, h], pivot=u['pivot'], rows=rows, animations=anims)


def apply_all(units, out=OUT):
    """Overlay every units/*.yaml onto the index's units."""
    for path in sorted(UNITS.glob('*.yaml')):
        key, entry = render(path, out)
        units[key] = entry


def main():
    sys.path.insert(0, str(ROOT / 'ver3'))
    import shadows
    index_path = OUT / 'index.json'
    index = json.loads(index_path.read_text())
    for path in [Path(p).resolve() for p in sys.argv[1:]] or sorted(UNITS.glob('*.yaml')):
        key, entry = render(path)
        index['units'][key] = entry
    tiles = index['tileset']
    tiles['sprites'] = {int(k): v for k, v in tiles['sprites'].items()}  # JSON keys come back as strings
    index['shadows'] = shadows.refresh(index, OUT)
    index_path.write_text(json.dumps(index, ensure_ascii=False, indent=1) + '\n')
    print('wrote', index_path.relative_to(ROOT))


if __name__ == '__main__':
    main()
