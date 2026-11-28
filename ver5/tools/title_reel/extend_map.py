#!/usr/bin/env python3
"""Writes the title reel's wide copies of story-scene maps: S_title_<name> is S_<name> with its ground carried on
`pad` cells past every edge, each border small tile repeated outward (grass stays grass, a road or a river runs on),
so that no edge of the map comes into the title's remembered circle at zoom 1. The story keeps the original maps.

    python3 tools/title_reel/extend_map.py S_field_road [S_xuzhou_ruin ...] [--pad 18]

Each copy is listed in assets/graphics/index.json "maps".
"""
import json
import random
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
MAPS = ROOT / 'assets/graphics/maps'
INDEX = ROOT / 'assets/graphics/index.json'


def pad_rows(rows, n):
    """Rows of a small-tile grid, each border tile repeated n tiles outward on every side."""
    rows = [r[0] * n + r + r[-1] * n for r in rows]
    return [rows[0]] * n + rows + [rows[-1]] * n


# maps whose grass and waste lie in long straight stripes, which on the screen turn into flat zigzags and diagonals:
# the copy lays the two out again in soft patches (smooth_ground), the waste's share kept
SMOOTH = {'S_jiangxia_harbor', 'S_guandu_field'}
# maps of plain grass that the reel shows wide: patches of waste laid over a share of the grass, the roads kept
MOTTLE = {'S_long_field': .3}
BARE = {'S_long_field'}  # the open plain the reel marches whole armies across: its scattered trees go
# game-cell rows (v) the reel uses of a long map: the copy keeps these alone, since a scene draws its whole map
# into one picture and the full 165-row field padded out would make a picture of some 13000 x 7000 px
CROP = {'S_long_field': (-30, 92)}
CROP_U = {'S_long_field': (-50, 62)}  # likewise the columns (u)


def value_noise(w, h, cell, seed):
    """Smooth noise in [0, 1) over a w x h grid, a random value every `cell` tiles eased between."""
    rnd = random.Random(seed)
    gw, gh = w // cell + 2, h // cell + 2
    grid = [[rnd.random() for _ in range(gw)] for _ in range(gh)]
    ease = lambda f: f * f * (3 - 2 * f)
    out = []
    for y in range(h):
        gy, fy = divmod(y / cell, 1)
        gy, fy = int(gy), ease(fy)
        row = []
        for x in range(w):
            gx, fx = divmod(x / cell, 1)
            gx, fx = int(gx), ease(fx)
            top = grid[gy][gx] * (1 - fx) + grid[gy][gx + 1] * fx
            bottom = grid[gy + 1][gx] * (1 - fx) + grid[gy + 1][gx + 1] * fx
            row.append(top * (1 - fy) + bottom * fy)
        out.append(row)
    return out


def smooth_ground(rows, seed=7):
    """Grass ('.') and waste ('d') laid out again in soft rounded patches: noise of three sizes summed and cut at the
    level that keeps the waste's share. Other ground (water, floors, bridges) stays as it is."""
    h, w = len(rows), len(rows[0])
    layers = [(value_noise(w, h, c, seed + c), k) for c, k in ((24, .55), (10, .3), (4, .15))]
    field = [[sum(n[y][x] * k for n, k in layers) for x in range(w)] for y in range(h)]
    open_ = [(y, x) for y in range(h) for x in range(w) if rows[y][x] in '.d']
    share = sum(rows[y][x] == 'd' for y, x in open_) / max(1, len(open_))
    cut = sorted(field[y][x] for y, x in open_)[int(len(open_) * (1 - share))] if open_ else 1
    out = [list(r) for r in rows]
    for y, x in open_:
        out[y][x] = 'd' if field[y][x] >= cut else '.'
    return [''.join(r) for r in out]


def mottle_ground(rows, share, seed=11):
    """Grass ('.') with soft rounded patches of waste over about `share` of it; what is already waste (a road)
    stays, and so does all other ground."""
    h, w = len(rows), len(rows[0])
    layers = [(value_noise(w, h, c, seed + c), k) for c, k in ((22, .55), (9, .3), (4, .15))]
    grass = [(y, x) for y in range(h) for x in range(w) if rows[y][x] == '.']
    if not grass:
        return rows
    field = {(y, x): sum(n[y][x] * k for n, k in layers) for y, x in grass}
    cut = sorted(field.values())[int(len(grass) * (1 - share))]
    out = [list(r) for r in rows]
    for at, f in field.items():
        if f >= cut:
            out[at[0]][at[1]] = 'd'
    return [''.join(r) for r in out]


def extend(base, pad):
    art = json.loads((MAPS / f'{base}.json').read_text())
    if base in BARE:
        art['objects'] = [o for o in art['objects'] if not str(o[0]).startswith(('stree', 'rock'))]
    if art.get('format') != 2:
        raise SystemExit(f'{base}: not a format 2 map')
    n = 2 * pad  # small tiles
    name = 'S_title_' + base.removeprefix('S_')
    art['id'] = name
    art['origin'] = [art['origin'][0] - n, art['origin'][1] - n]
    art['ground'] = pad_rows(art['ground'], n)
    if 'height' in art:
        art['height'] = pad_rows(art['height'], n)
    if base in SMOOTH:
        art['ground'] = smooth_ground(art['ground'])
    if base in CROP:  # keep small-tile rows 2*v0 .. 2*v1+1 and what stands on them
        v0, v1 = CROP[base]
        top = 2 * v0 - art['origin'][1]
        rows = slice(max(0, top), 2 * v1 + 2 - art['origin'][1])
        art['ground'] = art['ground'][rows]
        if 'height' in art:
            art['height'] = art['height'][rows]
        art['origin'][1] = max(art['origin'][1], 2 * v0)
        art['objects'] = [o for o in art['objects'] if 2 * v0 <= o[2] <= 2 * v1 + 1]
        art['size'][1] = min(art['size'][1], v1)
    if base in CROP_U:
        u0, u1 = CROP_U[base]
        left = 2 * u0 - art['origin'][0]
        cols = slice(max(0, left), 2 * u1 + 2 - art['origin'][0])
        art['ground'] = [r[cols] for r in art['ground']]
        if 'height' in art:
            art['height'] = [r[cols] for r in art['height']]
        art['origin'][0] = max(art['origin'][0], 2 * u0)
        art['objects'] = [o for o in art['objects'] if 2 * u0 <= o[1] <= 2 * u1 + 1]
    if base in MOTTLE:
        art['ground'] = mottle_ground(art['ground'], MOTTLE[base])
    (MAPS / f'{name}.json').write_text(json.dumps(art, ensure_ascii=False, separators=(',', ':')) + '\n')
    return name


def register(names):
    text = INDEX.read_text()
    index = json.loads(text)
    have = set(index['maps'])
    new = [f'maps/{n}.json' for n in names if f'maps/{n}.json' not in have]
    if not new:
        return
    # add the lines after the last map entry, keeping the file's own layout
    last = index['maps'][-1]
    at = text.index(f'"{last}"') + len(last) + 2
    text = text[:at] + ''.join(f',\n  "{p}"' for p in new) + text[at:]
    json.loads(text)
    INDEX.write_text(text)


def main(args):
    pad = 18
    if '--pad' in args:
        i = args.index('--pad')
        pad = int(args[i + 1])
        args = args[:i] + args[i + 2:]
    names = [extend(base, pad) for base in args]
    register(names)
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', *names], cwd=ROOT, check=True)  # the map files' own layout
    for n in names:
        print(n)


if __name__ == '__main__':
    main(sys.argv[1:])
