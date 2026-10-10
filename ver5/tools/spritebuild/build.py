#!/usr/bin/env python3
"""Bake the dot editor's units/*.yaml into assets/sprites/<id>.spr (format: docs/sprite-binary.md).

  python3 build.py                  # all units/*.yaml
  python3 build.py units/liubei.yaml
Requires PyYAML only.
"""
import struct
import sys
import zlib
from pathlib import Path

import yaml

HERE = Path(__file__).resolve().parent
UNITS = HERE.parents[1] / 'tools/doteditor/units'
OUT = HERE.parents[1] / 'assets/sprites'
MAGIC, VERSION = b'MHSP', 1
LOADER = getattr(yaml, 'CSafeLoader', yaml.SafeLoader)


def expand_noncombat(doc):
    """<hero>_noncombat holds NW·SW idle, two walk steps and one action. NE·SE are mirrored rows (cell widened
    to be symmetric about the pivot) and walk becomes idle, left, idle, right. Tags are mirrored with the pixels."""
    u = doc['unit']
    (w, h), (px, py) = u['cell'], u['pivot']
    half = max(px, w - px)
    pad = lambda row: [0]*(half - px) + row + [0]*(half - (w - px))
    src = {(f['direction'], f['animation'], f['frame']): ([pad(r) for r in f['pixels']], [pad(r) for r in f['tags']])
           for f in doc['frames']}
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
                px_, tg = src[(mirror or d, a, n)]
                flip = lambda rows: [r[::-1] for r in rows] if mirror else rows
                frames.append(dict(direction=d, animation=name, frame=k, pixels=flip(px_), tags=flip(tg)))
    doc['frames'] = frames
    u.update(cell=[2*half, h], pivot=[half, py], rows=['NW', 'SW', 'NE', 'SE'], animations=anims)
    return doc


def s(text):
    b = text.encode()
    return struct.pack('<H', len(b)) + b


def rgba(c):
    return bytes.fromhex(c[1:9])


def tag_group(tid):
    g = tid.rstrip('0123456789')
    return g, int(tid[len(g):] or 1)


def bake(path):
    doc = yaml.load(path.read_text(), Loader=LOADER)
    if doc.get('format') != 'mhdemo-unit-pixels':
        raise SystemExit(f'{path}: not a dot editor unit file')
    if doc['unit']['id'].endswith('_noncombat'):
        doc = expand_noncombat(doc)
    u = doc['unit']
    (w, h), (px, py), rows, anims = u['cell'], u['pivot'], u['rows'], u['animations']
    pal, tags = doc['palette'], doc['tag_definitions']
    if len(pal) > 256 or len(tags) > 255:
        raise SystemExit(f'{path}: palette or tag table too large')
    cols = max(a['first_column'] + a['frames'] for a in anims.values())
    b = bytearray(s(u['id']) + s(u['name']) + struct.pack('<HHHHB', w, h, px, py, len(rows)))
    for r in rows:
        b += s(r)
    b += struct.pack('<B', len(anims))
    for name, a in anims.items():
        b += s(name) + struct.pack('<HHBB', a['first_column'], a['frames'], a['loop'], len(a['ms'])) + struct.pack(f"<{len(a['ms'])}I", *a['ms'])
    b += struct.pack('<B', len(tags))
    for t, c in zip(tags, doc['tag_colors']):
        g, shade = tag_group(t['id'])
        b += struct.pack('<B', t['index']) + s(g) + struct.pack('<B', shade) + rgba(c)
    b += struct.pack('<H', len(pal)) + b''.join(rgba(c) for c in pal)
    b += struct.pack('<H', cols)
    cells = {(rows.index(f['direction']), anims[f['animation']]['first_column'] + f['frame']): f for f in doc['frames']}
    blank = bytes(w*h)
    for r in range(len(rows)):
        for c in range(cols):
            f = cells.get((r, c))
            for key in ('pixels', 'tags'):
                if f is None:
                    b += blank
                    continue
                plane = bytes(v for row in f.get(key) or [[0]*w]*h for v in row)
                if len(plane) != w*h:
                    raise SystemExit(f"{path}: frame {f['direction']}/{f['animation']}/{f['frame']} {key} has {len(plane)} cells, want {w*h}")
                b += plane
    return u['id'], MAGIC + struct.pack('<H', VERSION) + zlib.compress(bytes(b), 9)


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    for path in [Path(p).resolve() for p in sys.argv[1:]] or sorted(UNITS.glob('*.yaml')):
        key, data = bake(path)
        (OUT/f'{key}.spr').write_bytes(data)
        print(f'{path.name} {path.stat().st_size} -> {key}.spr {len(data)}')


if __name__ == '__main__':
    main()
