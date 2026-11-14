#!/usr/bin/env python3
"""Turn old scene maps into the scene map format 2 (internal/scenemap, docs/presentation-scene.md).

  python3 ver5/tools/scene_maps/scene_v2.py S_tent_mid S_castle_town                    # the map files as they are
  python3 ver5/tools/scene_maps/scene_v2.py --rev scene-render-base S_checkpoint        # the maps of a git revision
  python3 ver5/tools/scene_maps/scene_v2.py --copy 2 S_zhuo                               # a format 2 copy S_zhuo beside the old map

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
    # the battle map's glyphs the old maps drew themselves: a bridge deck on each `b` tile (an object, the deck at the
    # old size), the castle wall blocks `c` as wall lines on grass, the cliff blocks `h` as the tall cliff (height 2, the scene cliff the old maps' highland became)
    height = [list(r.replace('1', '2')) for r in old['highland']] if 'highland' in old else None
    at = lambda U, V: ground[V - vy][U - vx] if 0 <= V - vy < len(ground) and 0 <= U - vx < len(ground[0]) else ''
    for y, row in enumerate(ground):
        for x, g in enumerate(row):
            U, V = x + vx, y + vy
            if g == 'b':
                objects.append(['sbridge_v' if 'b' in (at(U, V - 1), at(U, V + 1)) else 'sbridge_u', U, V])
            elif g == 'c':
                if 'c' in (at(U - 1, V), at(U + 1, V)):
                    walls.setdefault(('u', V), set()).add(U)
                if 'c' in (at(U, V - 1), at(U, V + 1)) or 'c' not in (at(U - 1, V), at(U + 1, V)):
                    walls.setdefault(('v', U), set()).add(V)
            elif g == 'h':
                if height is None:
                    height = [['0'] * len(r) for r in ground]
                height[y][x] = '2'
    ground = [['.' if g == 'c' else g for g in row] for row in ground]
    lines = []
    for (axis, line), ts in sorted(walls.items()):
        ts = sorted(ts)
        runs = [[ts[0]]]
        for t in ts[1:]:   # a line with gaps is several lines
            runs[-1].append(t) if t == runs[-1][-1] + 1 else runs.append([t])
        for run in runs:
            a, b = run[0], run[-1]
            lines.append(['swall', [a, line], [b, line]] if axis == 'u' else ['swall', [line, a], [line, b]])
    out = {'id': old['id'], 'name': old['name'], 'format': 2, 'size': old['size'], 'origin': [vx, vy],
           'ground': [''.join(r) for r in ground]}
    if height:
        out['height'] = [''.join(r) for r in height]
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
    rev, copy = None, ''
    while args[:1] in (['--rev'], ['--copy']):
        if args[0] == '--rev':
            rev = args[1]
        else:
            copy = args[1]
        args = args[2:]
    index = json.load(open(GFX / 'index.json'))
    sprites = {**index['props']['sprites'], **index['extra']['sprites']}
    floors = {f['sprite']: g for g, f in json.load(open(GFX / 'scene.json'))['floors'].items()}
    for map_id in args:
        art = convert(load(map_id, rev), sprites, floors)
        art['id'] = map_id + copy
        if copy:
            art['name'] += ' (형식 2 사본)'
        path = GFX / 'maps' / f'{art["id"]}.json'
        write(art, path)
        rel = f'maps/{art["id"]}.json'
        if rel not in index['maps']:  # the game loads the maps listed here
            index['maps'].append(rel)
            json.dump(index, open(GFX / 'index.json', 'w'), ensure_ascii=False, indent=1)
            open(GFX / 'index.json', 'a').write('\n')
        print(map_id, path.stat().st_size, 'bytes,', len(art['objects']), 'objects,', len(art.get('walls', [])), 'walls')


if __name__ == '__main__':
    main()
