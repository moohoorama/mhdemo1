#!/usr/bin/env python3
"""3D cel-shaded render -> tag grid working copy for MOUNTED class sprites (rider on a horse, chariot).

  python3 grid3d_mounted.py <id> --unit light_cavalry [--base cavalry] [--anims ..] [--dirs ..]
  python3 grid3d_mounted.py <id> --unit K --denoise          # also clean the shading noise (denoise.py), head kept
  python3 grid3d_mounted.py <id> --unit K --denoise-only     # clean the existing grid files in place (no re-convert)
  python3 grid3d_mounted.py <id> --unit K --origin          # model origin on the pivot (lying figures), not the hooves
  python3 grid3d_mounted.py resize <id> LEFT RIGHT TOP BOTTOM     # grow the cell, pivot follows (negative shrinks)

Same idea as grid3d.py (tools/unit3d renders, this file converts), but for units whose head is a real 3D-stamped
head (no head box pasted from a base sheet) and whose horse has its own palette keys, so every material maps to
exactly one tag. Needs `grid.py new <base> <id> <name>` first; writes grid files only (run `grid.py import <id>`).

A unit is a module in mounted_kits/ that calls register(name, make). make() applies its monkeypatches and returns
(build, heads): build(row, f, style=None) for all six animation rows (the block row included) and the head set.
A scene pose row (--anims prone, added first with `grid.py poses <id> prone`) calls build(name, f) and stamps
heads['poses'][name](direction) when the kit gives one (e.g. the head turned on its side).
Attack effects come from the 3D render (stamped as effect tags); other animations paste the base sheet's effects.
A kit whose sheet was hand-cleaned after its conversion (관우) can add rows to it: heads['tags'] overrides KEY2TAG
for its own materials, and heads['sheet_head'] puts the sheet's own head (idle frame 0 of the direction, old weapon
pixels dropped) where the 2D head stamp lands, moved with the head, instead of the stamp.
"""
import argparse
import copy
import importlib
import re
import subprocess
import sys
from pathlib import Path

import grid3d as G
import denoise
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
# The face expression (heads.expression) draws the mouth with 'b' (red cloth shadow) and the tired sweat drop with
# 'q' over 'p' (horse muzzle / coat keys here): re-key those pixels to private keys so they get their own tags.
MOUTH, SWEAT = '<', {'q': '>', 'p': '^'}
R.COLOR.update({MOUTH: (101, 13, 36), '>': (140, 213, 255), '^': (73, 173, 240)})
# keys -> tags: horseSkin v w x, horseMane y z A, saddle B C D; mouth themeB3 (the 2D sheets' dark red; a kit whose
# themeB is a material, e.g. a buckler or a gourd, maps MOUTH to hair2 instead), sweat effect2 over effect3
KEY2TAG = dict(G.KEY2TAG, n='x', o='w', p='v', q='w', r='v', s='A', t='z', u='D', v='C', w='B', b='h',
               **{MOUTH: 'r', '>': 'F', '^': 'G'})


def _face_keys(expression):
    face = {'b': MOUTH, **SWEAT}

    def keyed(head, kind, skin, lid):
        out = expression(head, kind, skin, lid)
        return [''.join(face.get(c, c) if c != h else c for c, h in zip(row, src)) for row, src in zip(out, head)]
    return keyed


Hd.expression = _face_keys(Hd.expression)


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


class Tag(str):
    """A sheet tag pasted into a render grid (kept apart from the render keys, which share letters with tags)."""


EFFECT_NAMES = {tag: name for _, tag, name in G.EFFECT_RGB.values()}
WEAPON_TAGS = set('abcpqrt')  # the 관우 sheet's glaive: blade iron, shaft and tassel themeB, collar themeC2


def own_head(path, stamped, dx, dy):
    """The sheet's head pixels where idle frame 0's 2D head stamp landed, relative to the stamp centre; the old
    weapon (and outline touching only it) is dropped, so the new frame's own weapon or body shows there."""
    ref = read(path)
    sx, sy = stamped['at']
    out = {}
    for y, (r0, r1) in enumerate(zip(stamped['body'], stamped['head'])):
        for x, (c0, c1) in enumerate(zip(r0, r1)):
            if c0 == c1 or not (0 <= y + dy < len(ref) and 0 <= x + dx < len(ref[0])):
                continue
            tag = ref[y + dy][x + dx]
            if tag in WEAPON_TAGS or tag == '.':
                continue
            if tag == '#':
                near = [ref[y + dy + j][x + dx + i] for i, j in ((1, 0), (-1, 0), (0, 1), (0, -1))
                        if 0 <= y + dy + j < len(ref) and 0 <= x + dx + i < len(ref[0])]
                if not any(c not in WEAPON_TAGS and c not in '.#' for c in near):
                    continue
            out[x - sx, y - sy] = Tag(tag)
    return out


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
    head_px = []  # pixels the 2D head stamp changed (render coordinates): never denoised
    stamp, stamped = Hd.stamp, {}

    def tracked_stamp(g, head_set, d, face, cx, cy):
        before = [r[:] for r in g]
        stamp(g, head_set, d, face, cx, cy)
        head_px[:] = [(x, y) for y, (r0, r1) in enumerate(zip(before, g)) for x, (c0, c1) in enumerate(zip(r0, r1)) if c0 != c1]
        stamped.update(body=before, head=[r[:] for r in g], at=(round(cx), round(cy)))
    Hd.stamp = tracked_stamp
    key2tag = dict(KEY2TAG, **heads.get('tags', {}))
    out_root, ref_root = dots.HEROES / a.id / 'grid', dots.HEROES / a.base / 'grid'
    bw, bh = base['cell']
    for d in a.dirs.split(','):
        SHORT_BURST['on'] = vertical and d in FACING_STYLE
        captured.clear()
        B.frame(build, heads, B.SPEC, d, 0, 0, lift=0)
        bottom = max(y for y, row in enumerate(captured[-1]) for c in row if c != '.')
        dx, dy = px - B.SPEC['pivot'][0], py - bottom  # idle frame 0 puts the hooves on the pivot row
        if a.origin:  # a lying figure: the model origin (the body's middle) on the pivot
            dy = py - B.SPEC['pivot'][1]
        sheet_head = heads.get('sheet_head') and own_head(out_root / 'idle' / f'{d}-0.txt', stamped, dx, dy)
        for anim in a.anims.split(','):
            row = B.ANIMATIONS.index(anim) if anim in B.ANIMATIONS else anim
            head_set = heads['poses'][anim](d) if anim in heads.get('poses', {}) else heads
            for n in range(unit['animations'][anim]['frames']):
                captured.clear()
                effects.clear()
                B.frame(build, head_set, B.SPEC, d, row, n, lift=0,
                        style=FACING_STYLE.get(d) if vertical and anim == 'attack' else None)
                path = out_root / anim / f'{d}-{n}.txt'
                protect = [[False] * w for _ in range(h)]
                for x, y in head_px:
                    if 0 <= x + dx < w and 0 <= y + dy < h:
                        protect[y + dy][x + dx] = True
                if a.denoise_only:  # keep the grid file's pixels (hand fixes included), only clean them
                    G.write_grid(path, denoise.clean(read(path), protect)[0])
                    continue
                out = [['.'] * w for _ in range(h)]
                final = captured[-1]
                if sheet_head:
                    final = [r[:] for r in final]
                    for y, line in enumerate(final):
                        for x, c in enumerate(line):
                            if stamped['head'][y][x] != stamped['body'][y][x] and c == stamped['head'][y][x]:
                                line[x] = stamped['body'][y][x]  # the stamp's pixel (not a weapon in front of it)
                    sx, sy = stamped['at']
                    for (hx, hy), tag in sheet_head.items():
                        x, y = sx + hx, sy + hy
                        if captured[-1][y][x] == stamped['head'][y][x]:  # no weapon drawn in front of the head here
                            final[y][x] = tag
                for y, line in enumerate(final):
                    for x, c in enumerate(line):
                        if c == '.':
                            continue
                        if c not in key2tag and not (sheet_head and c in dots.TAG_CHARS):
                            sys.exit(f'unknown palette key {c!r} in {anim} {d} {n}')
                        if 0 <= x + dx < w and 0 <= y + dy < h:
                            out[y + dy][x + dx] = c if isinstance(c, Tag) else key2tag[c]
                        else:
                            print('clipped', anim, d, n, x + dx, y + dy)
                if anim == 'attack' or a.base == 'none':
                    for under, over in effects:
                        for layer in (over, under):
                            G.stamp(layer, out, dx, dy)
                    trail = heads.get('trail', {})  # effect tag -> (tag, alpha): a hand-cleaned sheet's own trail shades
                    for line in out:
                        for x, c in enumerate(line):
                            if isinstance(c, tuple) and c[1] in trail:
                                tag, alpha = trail[c[1]]
                                line[x] = ('t', tag, alpha, EFFECT_NAMES[tag])
                if anim != 'attack' and a.base != 'none' and anim in B.ANIMATIONS:
                    ref = read(ref_root / anim / f'{d}-{n}.txt')
                    for y in range(bh):
                        for x in range(bw):
                            if ref[y][x] in G.EFFECT and 0 <= x + ox < w and 0 <= y + oy < h and out[y + oy][x + ox] == '.':
                                out[y + oy][x + ox] = ref[y][x]
                if a.denoise:
                    out = denoise.clean(out, protect)[0]
                G.write_grid(path, out)
        print(d, 'done', flush=True)
    if a.mirror:
        for anim in a.anims.split(','):
            for n in range(unit['animations'][anim]['frames']):
                for d, src in Hd.MIRROR.items():
                    rows = read(out_root / anim / f'{src}-{n}.txt')
                    G.write_grid(out_root / anim / f'{d}-{n}.txt',
                                 [[r[2*px - x] if 0 <= 2*px - x < w else '.' for x in range(w)] for r in rows])
                    print(d, 'mirrored from', src, flush=True)


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
    ap.add_argument('--denoise', action='store_true', help='clean the converted frames with denoise.clean (head kept)')
    ap.add_argument('--mirror', action='store_true',
                    help='write NE, E and SE as the NW, W and SW grids flipped about the pivot (render only the W side)')
    ap.add_argument('--origin', action='store_true',
                    help='put the model origin on the pivot instead of the lowest idle pixel (a figure lying down)')
    ap.add_argument('--denoise-only', action='store_true',
                    help='render only to locate the head, then clean the existing grid files in place')
    convert(ap.parse_args())


if __name__ == '__main__':
    main()
