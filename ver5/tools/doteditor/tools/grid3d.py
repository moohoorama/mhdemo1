#!/usr/bin/env python3
"""3D cel-shaded render -> tag grid working copy (heroes/<id>/grid), for class sprites derived from an existing sheet.

  python3 grid3d.py <id> [--base infantry] [--anims idle,walk,...] [--dirs S,SW,...]

Pipeline (tools/unit3d does the rendering, this file only converts):
  1. render every frame of the unit with the 3D body kit in KITS (no head),
  2. map the renderer's palette keys to the dot tags (KEY2TAG; the keys are unique per material),
  3. paste the head box (12 rows, found by the plume pattern) and the attack/hit effect pixels
     verbatim from the base sheet's grid (`grid.py export <base>` first),
  4. apply the hand-checked FIXES (frames where the 3D pose has no readable blade).
Needs `grid.py new <base> <id> <name>` and `grid.py export <id>` first; writes grid files only, never the yaml
(run `grid.py import <id>` afterwards). Grid files for other frames are left alone.
"""
import argparse
import re
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
REPO = HERE.parents[3]
sys.path.insert(0, str(REPO / 'tools/unit3d'))
import base_model as m  # noqa: E402
import units as U  # noqa: E402
import attacks as A  # noqa: E402
import blocks as Bl  # noqa: E402
import heads as Hd  # noqa: E402
import build as B  # noqa: E402

V = m.V
LEATHER = U.BROWN
W, H = 56, 46
EFFECT = set('EFG123456')
# renderer palette key -> grid tag (iron a b c, skin d e, faction j k l, themeA m n o, themeC s t u, outline #).
# Each material uses its own keys, so the mapping is exact; unknown keys abort the run.
KEY2TAG = {'0': '#', '1': 'c', '2': 'c', '3': 'b', '4': 'b', '5': 'a', '6': 'a', 'i': 'd', 'j': 'e',
           'f': 'o', 'g': 'n', 'h': 'm', 'k': 'u', 'l': 't', 'm': 's', 'W': 'l', 'X': 'l', 'Y': 'k', 'Z': 'j'}


def decorate_heavy(p):
    """중보병: silver cuirass and pauldron caps, leather belt/skirt, faction pauldrons, shield and long cloak."""
    T = U.transform(p)
    m.ell(T([0, -.05, 1.05]), [.31, .24, .5], U.STEEL)
    m.box(T([0, -.06, .78]), [.62, .5, .09], LEATHER)
    for x in (-.18, 0, .18):
        m.box(T([x, -.29, .6]), [.14, .05, .34], LEATHER)
    for s in (-1, 1):
        sh = T([s*.4, 0, 1.42])
        m.ell(sh + V([s*.01, 0, -.02]), [.22, .24, .2], U.BLUE)
        m.ell(sh + V([s*.01, 0, .09]), [.2, .21, .09], U.STEEL)
    c = T(p['hands'][1]) + V([0, -.09, 0])
    m.ell(c + V([0, .02, 0]), [.31, .07, .4], LEATHER)
    m.ell(c, [.27, .08, .36], U.LIGHT_BLUE)
    m.ell(c + V([0, -.07, 0]), [.09, .055, .1], U.STEEL)
    for z, w, y, h in [(1.52, .82, .29, .2), (1.30, .84, .31, .26), (1.02, .88, .33, .32),
                       (.72, .92, .35, .32), (.42, .96, .37, .32), (.14, 1.0, .39, .3)]:
        m.box(T([0, y, z]), [w, .08, h], U.BLUE)
    m.ell(T([0, -.02, 1.5]), [.3, .22, .1], U.BLUE)


# kit name -> (kit dict for units.KITS, decoration drawn after the base soldier)
KITS = {'heavy': (dict(cloth=U.BLUE, tabard=LEATHER, guard=U.STEEL, shield=False, weapon='sword'), decorate_heavy)}
# (anim, dir, frame) -> [(row0, row1, col0, col1, replace_all)]: paste these base-sheet pixels (blade the 3D pose hides)
FIXES = {('heavy', 'attack', 'S', 1): [(9, 17, 13, 21, True), (17, 18, 13, 21, False)], ('heavy', 'attack', 'SW', 1): [(5, 14, 23, 30, True)]}


def read(path):
    return [l[3:] for l in path.read_text().split('\n') if re.match(r'^\d\d\|', l)]


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument('id')
    ap.add_argument('--kit', default='heavy', choices=KITS)
    ap.add_argument('--base', default='infantry', help='base sheet whose exported grid (heroes/<base>/grid) supplies heads and effects')
    ap.add_argument('--anims', default=','.join(B.ANIMATIONS))
    ap.add_argument('--dirs', default=','.join(B.DIRECTIONS))
    a = ap.parse_args()
    kit = a.kit
    U.KITS[kit], decorate = KITS[kit]
    A.SELECTED[kit] = A.SELECTED[a.base]
    orig_soldier = U.soldier

    def soldier(p, k, weapon=True):
        out = orig_soldier(p, k, weapon)
        if k == kit:
            decorate(p)
        return out

    U.soldier = soldier
    foot, block = U.foot_unit(kit), Bl.foot(kit)

    def build(row, f, style=None):
        if row == B.BLOCK:
            return block(f, Bl.SELECTED)
        return foot(row, f, style=style) if style else foot(row, f)

    captured = []
    to_img = Hd.to_img
    Hd.to_img = lambda g: (captured.append([r[:] for r in g]), to_img(g))[1]
    head = Hd.heads({d: ['.' * 13] * 12 for d in Hd.IRON})  # body only; heads come from the base grid
    out_root, ref_root = HERE.parent / 'heroes' / a.id / 'grid', HERE.parent / 'heroes' / a.base / 'grid'
    plume = re.compile(r'#(?:j|jk|kj|jj)#')
    solid = lambda c: c != '.' and c not in EFFECT
    for d in a.dirs.split(','):
        captured.clear()
        B.frame(build, head, B.SPEC, d, 0, 0, lift=0)
        bottom = max(y for y, row in enumerate(captured[-1]) for c in row if c != '.')
        dx, dy = 28 - B.SPEC['pivot'][0], 38 - bottom  # idle frame 0 puts the feet on row 38
        for anim in a.anims.split(','):
            for n in range(4):
                captured.clear()
                B.frame(build, head, B.SPEC, d, B.ANIMATIONS.index(anim), n, lift=0)
                out = [['.'] * W for _ in range(H)]
                for y, row in enumerate(captured[-1]):
                    for x, c in enumerate(row):
                        if c == '.':
                            continue
                        if c not in KEY2TAG:
                            sys.exit(f'unknown palette key {c!r} in {anim} {d} {n}')
                        if 0 <= x + dx < W and 0 <= y + dy < H:
                            out[y + dy][x + dx] = KEY2TAG[c]
                        else:
                            print('clipped', anim, d, n, x + dx, y + dy)
                ref = read(ref_root / anim / f'{d}-{n}.txt')
                top = next(y for y, r in enumerate(ref) if plume.search(r[20:40]))
                px = 20 + plume.search(ref[top][20:40]).start() + 1
                for y in range(top, min(top + 12, H)):
                    for x in range(px - 7, px + 8):
                        if solid(ref[y][x]):
                            out[y][x] = ref[y][x]
                for r0, r1, c0, c1, replace in FIXES.get((kit, anim, d, n), []):
                    for y in range(r0, r1):
                        for x in range(c0, c1):
                            if solid(ref[y][x]) and (replace or out[y][x] == '.'):
                                out[y][x] = ref[y][x]
                for y in range(H):
                    for x in range(W):
                        if ref[y][x] in EFFECT and out[y][x] == '.':
                            out[y][x] = ref[y][x]
                path = out_root / anim / f'{d}-{n}.txt'
                lines = path.read_text().split('\n')
                for i, l in enumerate(lines):
                    if re.match(r'^\d\d\|', l):
                        lines[i] = l[:3] + ''.join(out[int(l[:2])])
                path.write_text('\n'.join(lines))
        print(d, 'done', flush=True)


if __name__ == '__main__':
    main()
