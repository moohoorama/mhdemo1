#!/usr/bin/env python3
"""Turn old scene maps into the scene map format 2 (internal/scenemap, docs/presentation-scene.md).

  python3 ver5/tools/scene_maps/scene_v2.py S_tent_mid S_castle_town                    # the map files as they are
  python3 ver5/tools/scene_maps/scene_v2.py --rev scene-render-base S_checkpoint        # the maps of a git revision

An old map stored the picture: every ground piece, cliff face, foot and corner shadow and grass tuft. Format 2 keeps
what was decided by hand and lets the rules make the rest:
  ground   the small tiles' glyphs; a floor prop (floor_tent, …) on a game cell becomes its floor glyph there
  height   the raised small tiles (old "highland": a tall cliff, height step 2)
  objects  [name, U, V, dx, dy]: the prop's anchor dx, dy canvas pixels from the centre of small tile (U, V), rounded
           the way the old viewer rounded its drawing position, so it lands on the same pixel; a gate is one object
           at its middle, set into its wall line by the rules
  walls    [family, [U, V], [U, V]]: each run of scene wall pieces
Left out, because the rules make them: ground, decorations, tcliff_/lcliff_ faces, cfoot_ and jshade_ shadows, wall
pieces and gate slices. Writes assets/graphics/maps/<id>.json.
"""
import json
import math
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
GFX = ROOT / 'ver5/assets/graphics'
RULED = ('tcliff_', 'lcliff_', 'cfoot_', 'jshade_')


def js_round(x):
    return math.floor(x + .5)


def load(map_id, rev):
    rel = f'ver5/assets/graphics/maps/{map_id}.json'
    if rev:
        return json.loads(subprocess.check_output(['git', 'show', f'{rev}:{rel}'], cwd=ROOT))
    return json.load(open(ROOT / rel))


def anchors(old):
    """Every placed picture of an old map as (name, U, V, x, y): its small tile and its drawing point in canvas pixels
    (the old viewer's), dropping the ground and the pictures the rules make."""
    out = []
    for p in old.get('props', []):
        if len(p) != 3:
            raise SystemExit(f'{old["id"]}: fill props are not read: {p}')
        name, u, v = p
        out.append((name, math.floor(2 * u + 1), math.floor(2 * v + 1), (u - v) * 32, (u + v + 1) * 16))
    # the checkpoint's small-grid entries: snapped to their tile's centre (small_snap), faces on their tile's centre
    for name, U, V, *_ in old.get('small_props', []) + old.get('small_faces', []):
        out.append((name, U, V, 16 * (U - V), 8 * (U + V + 1)))
    return [a for a in out if not a[0].startswith(RULED)]


def convert(old, sprites, floors):
    vx, vy, vw, vh = old['visual']
    ground = [list(r) for r in old['small_tiles']]
    objects, walls, gates = [], {}, {}
    for name, U, V, x, y in anchors(old):
        if name in floors:  # a floor on a game cell: its four small tiles
            gu, gv = round((x / 32 + y / 16 - 1) / 2), round((y / 16 - 1 - x / 32) / 2)
            for j in range(2):
                for i in range(2):
                    ground[2 * gv + j - vy][2 * gu + i - vx] = floors[name]
            continue
        if name.startswith('swall_'):
            axis = name[-1]
            walls.setdefault((axis, V if axis == 'u' else U), set()).add(U if axis == 'u' else V)
            continue
        if name.startswith('sgate_'):
            _, state, axis, k = name.split('_')
            t = (U if axis == 'u' else V) - round(2 * sprites[name]['gate_offset'])
            key = (f'sgate_{state}', axis, V if axis == 'u' else U)
            gates.setdefault(key, set()).add(t)
            if sprites[name]['rect'][2] >= 16:  # a slice in place of a wall piece: the line runs through it
                walls.setdefault((axis, key[2]), set()).add(U if axis == 'u' else V)
            continue
        dx, dy = js_round(x) - 16 * (U - V), js_round(y) - 8 * (U + V + 1)
        objects.append([name, U, V] + ([dx, dy] if dx or dy else []))
    for (name, axis, line), ts in gates.items():
        if len(ts) != 1:
            raise SystemExit(f'{old["id"]}: {name} slices disagree on the gate point {ts}')
        t = ts.pop()
        objects.append([name, t, line] if axis == 'u' else [name, line, t])
    lines = []
    for (axis, line), ts in sorted(walls.items()):
        ts = sorted(ts)
        if ts != list(range(ts[0], ts[-1] + 1)):
            raise SystemExit(f'{old["id"]}: wall {axis} {line} has gaps {ts}')
        a, b = (ts[0], line), (ts[-1], line)
        lines.append(['swall', list(a), list(b)] if axis == 'u' else ['swall', [line, ts[0]], [line, ts[-1]]])
    out = {'id': old['id'], 'name': old['name'], 'format': 2, 'size': old['size'], 'origin': [vx, vy],
           'ground': [''.join(r) for r in ground]}
    if 'highland' in old:
        out['height'] = [r.replace('1', '2') for r in old['highland']]
    out['objects'] = objects
    if lines:
        out['walls'] = lines
    return out


def write(art, path):
    """One line per row and per object, so a diff shows what changed."""
    D = lambda x: json.dumps(x, ensure_ascii=False, separators=(',', ':'))
    out = ['{']
    for k, v in art.items():
        if k in ('ground', 'height', 'objects', 'walls'):
            out.append(f' {D(k)}: [')
            out += [f'  {D(x)}' + (',' if i < len(v) - 1 else '') for i, x in enumerate(v)]
            out.append(' ],')
        else:
            out.append(f' {D(k)}: {D(v)},')
    out[-1] = out[-1].rstrip(',')
    out.append('}')
    open(path, 'w').write('\n'.join(out) + '\n')


def main():
    args = sys.argv[1:]
    rev = None
    if args[:1] == ['--rev']:
        rev, args = args[1], args[2:]
    index = json.load(open(GFX / 'index.json'))
    sprites = {**index['props']['sprites'], **index['extra']['sprites']}
    floors = {f['sprite']: g for g, f in json.load(open(GFX / 'scene.json'))['floors'].items()}
    for map_id in args:
        art = convert(load(map_id, rev), sprites, floors)
        path = GFX / 'maps' / f'{map_id}.json'
        write(art, path)
        print(map_id, path.stat().st_size, 'bytes,', len(art['objects']), 'objects,', len(art.get('walls', [])), 'walls')


if __name__ == '__main__':
    main()
