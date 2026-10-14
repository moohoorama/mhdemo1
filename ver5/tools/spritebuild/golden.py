#!/usr/bin/env python3
"""Regenerate internal/sprite/testdata/units/<id>.png (the sheet as the dot editor draws it) and heavy_infantry.shadow.png.
Needs Pillow and PyYAML."""
from pathlib import Path

import yaml
from PIL import Image

import build
import shadows

HERE = Path(__file__).resolve().parent
OUT = HERE.parents[1]/'internal/sprite/testdata/units'


def render(path):
    doc = yaml.load(path.read_text(), build.LOADER)
    if doc['unit']['id'].endswith('_noncombat'):
        doc = build.expand_noncombat(doc)
    u = doc['unit']
    (w, h), rows, anims = u['cell'], u['rows'], u['animations']
    palette = [bytes.fromhex(c[1:]) for c in doc['palette']]
    columns = max(a['first_column'] + a['frames'] for a in anims.values())
    sheet = Image.new('RGBA', (w*columns, h*len(rows)))
    for f in doc['frames']:
        cell = Image.frombytes('RGBA', (w, h), b''.join(palette[i] for row in f['pixels'] for i in row))
        sheet.paste(cell, ((anims[f['animation']]['first_column'] + f['frame'])*w, rows.index(f['direction'])*h))
    sheet.save(OUT/f"{u['id']}.png")
    print('dot', path.name, '->', f"{u['id']}.png")


OUT.mkdir(parents=True, exist_ok=True)
for p in sorted(build.UNITS.glob('*.yaml')):
    render(p)
sheet, *_ = shadows.unit_sheet(Image.open(OUT/'heavy_infantry.png').convert('RGBA'), (56, 46), (28, 38))
sheet.save(OUT/'heavy_infantry.shadow.png')
