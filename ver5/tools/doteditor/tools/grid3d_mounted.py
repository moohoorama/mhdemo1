#!/usr/bin/env python3
"""3D cel-shaded render -> tag grid working copy for MOUNTED class sprites (rider on a horse, chariot).

  python3 grid3d_mounted.py <id> --unit light_cavalry [--base cavalry] [--anims ..] [--dirs ..]
  python3 grid3d_mounted.py resize <id> LEFT RIGHT TOP BOTTOM     # grow the cell, pivot follows (negative shrinks)

Same idea as grid3d.py (tools/unit3d renders, this file converts), but for units whose head is a real 3D-stamped
head (no head box pasted from a base sheet) and whose horse has its own palette keys, so every material maps to
exactly one tag. Needs `grid.py new <base> <id> <name>` first; writes grid files only (run `grid.py import <id>`).

A unit is a module in mounted_kits/ that calls register(name, make). make() applies its monkeypatches and returns
(build, heads): build(row, f, style=None) for all six animation rows (the block row included) and the head set.
Attack effects come from the 3D render (stamped as effect tags); other animations paste the base sheet's effects.
"""
import argparse
import copy
import importlib
import re
import subprocess
import sys
from pathlib import Path

import grid3d as G
import dots

if __name__ == '__main__':
    sys.modules['grid3d_mounted'] = sys.modules['__main__']  # mounted_kits modules import this file by name

U, A, B, Bl, Hd, R = G.U, G.A, G.B, G.Bl, G.Hd, G.B.R
HERE = Path(__file__).resolve().parent
UNITS = {}

# Horse materials get their own render keys (units.cavalry draws them with colours that share ramps with leather).
COAT = ('#825033', '#9a6846', '#352821')  # coat, muzzle, mane and hooves
R.add_ramps({COAT[0]: 'nopp', COAT[1]: 'qrrr', COAT[2]: '0sst', '#a91e2c': 'uuvw'})
U.COATS['dot'] = COAT
# the knight palette stops at 'r': placeholder render colours for the new keys (the dot tags decide the final colours)
R.COLOR.update({'s': (61, 34, 29), 't': (102, 56, 32), 'u': (27, 51, 88), 'v': (53, 103, 166), 'w': (90, 143, 208)})
# keys -> tags: horseSkin v w x, horseMane y z A, saddle B C D; 'b' is the face's mouth pixel (hair2)
KEY2TAG = dict(G.KEY2TAG, n='x', o='w', p='v', q='w', r='v', s='A', t='z', u='D', v='C', w='B', b='h')


def register(name, make, vertical_spear=False):
    """vertical_spear: the unit's build takes attacks.SPEAR style names; N and S attacks then use the pitched charge."""
    UNITS[name] = make
    if vertical_spear:
        VERTICAL_SPEAR.add(name)


# Facing the camera or away from it the spear points along the view axis and shrinks to a stub: pitch the thrust
# down (S) or up (N) so the spear stays vertical on screen and shows its length (same idea as the pikeman's).
VERTICAL_SPEAR = set()
FACING_STYLE = {'N': 'charge_n', 'S': 'charge_s'}
SHORT_BURST = {'on': False}  # set per direction: in N and S the burst and speed lines would leave the cell


def _short_pierce(pierce):
    def short(*args, **kw):
        if not SHORT_BURST['on']:
            return pierce(*args, **kw)
        out = pierce(*args, **dict(kw, reach=.6))
        for e in out:
            if e['kind'] == 'cone':
                e.update(length=.55, width=.45)
            elif e['kind'] == 'ring':
                e.update(center=e['center'] - e['axis']*.1, radius=.28)
        return out
    return short


A.pierce = _short_pierce(A.pierce)
for _name, _z in (('charge_s', -.45), ('charge_n', .45)):
    A.SPEAR[_name] = copy.deepcopy(A.SPEAR['charge'])
    for _frame in A.SPEAR[_name]['frames'][2:]:
        _frame['dir'] = [0, -.9, _z]


def load_units():
    sys.path.insert(0, str(HERE))
    for path in sorted((HERE / 'mounted_kits').glob('*.py')):
        importlib.import_module(f'mounted_kits.{path.stem}')


def rider_build(kit, coat='dot', weapon='spear', cavalry=None):
    """build(row, f, style=None) for a rider unit: units.cavalry for every row, blocks.rider for the block row."""
    cavalry = cavalry or U.cavalry
    block = Bl.rider(kit=kit, coat=coat, weapon=weapon)

    def build(row, f, style=None):
        if row == B.BLOCK:
            return block(f, Bl.SELECTED)
        return cavalry(row, f, style, kit=kit, coat=coat, weapon=weapon)
    return build


def resize(args):
    """Grow the cell (every frame padded with transparent pixels) and move the pivot with the left/top padding."""
    left, right, top, bottom = args.left, args.right, args.top, args.bottom
    if min(left, right, top, bottom) < 0:
        raise SystemExit('only growing is supported')
    doc = dots.with_alpha(dots.load(args.id))
    for f in doc['frames']:
        for key, empty in (('tags', 0), ('alpha', '00')):
            w = len(f[key][0]) + left + right
            f[key] = ([[empty]*w for _ in range(top)] + [[empty]*left + r + [empty]*right for r in f[key]]
                      + [[empty]*w for _ in range(bottom)])
    u = doc['unit']
    u['cell'] = [u['cell'][0] + left + right, u['cell'][1] + top + bottom]
    u['pivot'] = [u['pivot'][0] + left, u['pivot'][1] + top]
    print('wrote', dots.save(doc), 'cell', u['cell'], 'pivot', u['pivot'])
    subprocess.run([sys.executable, str(HERE / 'grid.py'), 'export', args.id, '--force'], check=True)


def read(path):
    return [l[3:] for l in path.read_text().split('\n') if re.match(r'^\d\d\|', l)]


def convert(a):
    load_units()
    build, heads = UNITS[a.unit]()
    vertical = a.unit in VERTICAL_SPEAR
    unit = dots.load(a.id)['unit']
    base = unit if a.base == 'none' else dots.load(a.base)['unit']  # --base none: no effects pasted from a base sheet
    (w, h), (px, py) = unit['cell'], unit['pivot']
    ox, oy = px - base['pivot'][0], py - base['pivot'][1]  # base-sheet coordinates -> this sheet's
    captured, effects = [], []
    to_img = Hd.to_img
    Hd.to_img = lambda g: (captured.append([r[:] for r in g]), to_img(g))[1]
    draw = B.E.draw
    B.E.draw = lambda *args: effects.append(draw(*args)) or effects[-1]
    out_root, ref_root = dots.HEROES / a.id / 'grid', dots.HEROES / a.base / 'grid'
    bw, bh = base['cell']
    for d in a.dirs.split(','):
        SHORT_BURST['on'] = vertical and d in FACING_STYLE
        captured.clear()
        B.frame(build, heads, B.SPEC, d, 0, 0, lift=0)
        bottom = max(y for y, row in enumerate(captured[-1]) for c in row if c != '.')
        dx, dy = px - B.SPEC['pivot'][0], py - bottom  # idle frame 0 puts the hooves on the pivot row
        for anim in a.anims.split(','):
            for n in range(4):
                captured.clear()
                effects.clear()
                B.frame(build, heads, B.SPEC, d, B.ANIMATIONS.index(anim), n, lift=0,
                        style=FACING_STYLE.get(d) if vertical and anim == 'attack' else None)
                out = [['.'] * w for _ in range(h)]
                for y, row in enumerate(captured[-1]):
                    for x, c in enumerate(row):
                        if c == '.':
                            continue
                        if c not in KEY2TAG:
                            sys.exit(f'unknown palette key {c!r} in {anim} {d} {n}')
                        if 0 <= x + dx < w and 0 <= y + dy < h:
                            out[y + dy][x + dx] = KEY2TAG[c]
                        else:
                            print('clipped', anim, d, n, x + dx, y + dy)
                if anim == 'attack' or a.base == 'none':
                    for under, over in effects:
                        for layer in (over, under):
                            G.stamp(layer, out, dx, dy)
                if anim != 'attack' and a.base != 'none':
                    ref = read(ref_root / anim / f'{d}-{n}.txt')
                    for y in range(bh):
                        for x in range(bw):
                            if ref[y][x] in G.EFFECT and 0 <= x + ox < w and 0 <= y + oy < h and out[y + oy][x + ox] == '.':
                                out[y + oy][x + ox] = ref[y][x]
                G.write_grid(out_root / anim / f'{d}-{n}.txt', out)
        print(d, 'done', flush=True)


def main():
    if len(sys.argv) > 1 and sys.argv[1] == 'resize':
        ap = argparse.ArgumentParser()
        ap.add_argument('cmd'), ap.add_argument('id')
        for k in ('left', 'right', 'top', 'bottom'):
            ap.add_argument(k, type=int)
        return resize(ap.parse_args())
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument('id')
    ap.add_argument('--unit', required=True)
    ap.add_argument('--base', default='cavalry')
    ap.add_argument('--anims', default=','.join(B.ANIMATIONS))
    ap.add_argument('--dirs', default=','.join(B.DIRECTIONS))
    convert(ap.parse_args())


if __name__ == '__main__':
    main()
