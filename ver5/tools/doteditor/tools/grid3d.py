#!/usr/bin/env python3
"""3D cel-shaded render -> tag grid working copy (heroes/<id>/grid), for class sprites derived from an existing sheet.

  python3 grid3d.py <id> [--base heavy_infantry] [--anims idle,walk,...] [--dirs S,SW,...]

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
BUCKLER = '#d7bc7c'  # light wood; its ramp keys A B C map to themeB
U.RAMPS[BUCKLER] = 'ABCC'
B.R.add_ramps(U.RAMPS)
W, H = 56, 46  # the base sheets' cell
PAD = {'pike': 12}  # kit -> columns added on each side of the cell (the couched spear reaches past the 56 columns)
EXTRA_ROWS = {'pike': 8}  # kit -> rows added below the cell (the spear lying toward the camera when exhausted, thrusting toward it)
UP_ROWS = {'pike': 2}  # kit -> rows added above the cell (the spear thrust away from the camera); the pivot row moves down by this
EFFECT = set('EFG123456')
# renderer palette key -> grid tag (iron a b c, skin d e, faction j k l, themeA m n o, themeC s t u, outline #).
# Each material uses its own keys, so the mapping is exact; unknown keys abort the run.
KEY2TAG = {'0': '#', '1': 'c', '2': 'c', '3': 'b', '4': 'b', '5': 'a', '6': 'a', 'i': 'd', 'j': 'e',
           'f': 'o', 'g': 'n', 'h': 'm', 'k': 'u', 'l': 't', 'm': 's', 'W': 'l', 'X': 'l', 'Y': 'k', 'Z': 'j',
           'A': 'r', 'B': 'q', 'C': 'p',
           'b': 'r', 'c': 'q', 'd': 'p'}  # themeB: the 경보병 buckler, the 장창병 red spear tassel


def round_shield(p, T):
    c = T(p['hands'][1]) + V([0, -.09, 0])
    m.ell(c + V([0, .02, 0]), [.31, .07, .4], LEATHER)
    m.ell(c, [.27, .08, .36], U.LIGHT_BLUE)
    m.ell(c + V([0, -.07, 0]), [.09, .055, .1], U.STEEL)


def tower_shield(p, T):
    """Tall rectangular shield: leather frame, faction face, steel rim bands and boss."""
    c = T(p['hands'][1]) + V([0, -.09, .12])
    m.box(c + V([0, .02, 0]), [.46, .07, 1.0], LEATHER)
    m.box(c, [.38, .08, .92], U.LIGHT_BLUE)
    for z in (-.46, .46):
        m.box(c + V([0, -.02, z]), [.46, .09, .07], U.STEEL)
    m.box(c + V([0, -.05, 0]), [.08, .06, .8], U.STEEL)
    m.ell(c + V([0, -.07, .06]), [.09, .055, .1], U.STEEL)


def decorate_body(p, shield):
    """중보병 body: silver cuirass and pauldron caps, leather belt/skirt, faction pauldrons, shield and long cloak."""
    T = U.transform(p)
    m.ell(T([0, -.05, 1.05]), [.31, .24, .5], U.STEEL)
    m.box(T([0, -.06, .78]), [.62, .5, .09], LEATHER)
    for x in (-.18, 0, .18):
        m.box(T([x, -.29, .6]), [.14, .05, .34], LEATHER)
    for s in (-1, 1):
        sh = T([s*.4, 0, 1.42])
        m.ell(sh + V([s*.01, 0, -.02]), [.22, .24, .2], U.BLUE)
        m.ell(sh + V([s*.01, 0, .09]), [.2, .21, .09], U.STEEL)
    shield(p, T)
    for z, w, y, h in [(1.52, .82, .29, .2), (1.30, .84, .31, .26), (1.02, .88, .33, .32),
                       (.72, .92, .35, .32), (.42, .96, .37, .32), (.14, 1.0, .39, .3)]:
        m.box(T([0, y, z]), [w, .08, h], U.BLUE)
    m.ell(T([0, -.02, 1.5]), [.3, .22, .1], U.BLUE)


def decorate_heavy(p):
    decorate_body(p, round_shield)


def decorate_guards(p):
    decorate_body(p, tower_shield)


def decorate_light(p, buckler=True):
    """경보병: cloth body, leather cross straps / chest plate / belt / shin wraps, small buckler with iron studs."""
    T = U.transform(p)
    for i in range(2):
        foot, knee = V(p['feet'][i]), V(p['knees'][i])
        m.rod(foot + V([0, 0, .05]), foot + (knee - foot)*.42, .16, LEATHER)
    for s in (-1, 1):
        for y in (-.235, .235):
            m.rod(T([s*.3, y*.78, 1.5]), T([-s*.22, y, .98]), .04, LEATHER)
    m.ell(T([0, -.265, 1.24]), [.2, .06, .17], LEATHER)
    m.box(T([0, 0, .78]), [.62, .5, .09], LEATHER)
    if not buckler:
        return
    c = T(p['hands'][1]) + V([0, -.09, 0])
    m.ell(c, [.23, .07, .29], BUCKLER)
    m.ell(c + V([0, -.065, 0]), [.1, .055, .1], U.STEEL)
    for sx, sz in [(-.15, .17), (.15, .17), (-.15, -.17), (.15, -.17), (0, .21), (0, -.21)]:
        m.ell(c + V([sx, -.065, sz]), [.05, .045, .05], U.STEEL)


# kit name -> (kit dict for units.KITS, decoration drawn after the base soldier)
HEAVY_KIT = dict(cloth=U.BLUE, tabard=LEATHER, guard=U.STEEL, shield=False, weapon='sword')
LIGHT_KIT = dict(cloth=U.BLUE, tabard=U.LIGHT_BLUE, guard=U.BLUE, shoulder=U.BLUE, shield=False, weapon='sword')
PIKE_KIT = dict(LIGHT_KIT, weapon='spear')
KITS = {'heavy': (HEAVY_KIT, decorate_heavy), 'guards': (HEAVY_KIT, decorate_guards), 'light': (LIGHT_KIT, decorate_light),
        'pike': (PIKE_KIT, lambda p: decorate_light(p, buckler=False))}
# 장창병: the cavalry's couched thrust (hand and spear direction of attacks.SPEAR['charge']) on the infantry thrust stance
A.SWORD['pike'] = dict(name='창 찌르기', effect='pierce', frames=[
    dict(hand=[-.48, .3, 1.2], blade=[0, -1, .12], lean=.1),
    dict(hand=[-.45, .6, 1.3], blade=[0, -1, .2], lean=.3, bob=-.12, crouch=1),
    dict(hand=[-.48, -1.1, 1.15], blade=[0, -1, 0], lean=-.42, bob=-.08, lunge=1.2),
    dict(hand=[-.48, -.8, 1.15], blade=[0, -1, -.05], lean=-.24, bob=-.06, lunge=1)])
# facing the camera or away from it the spear points along the view axis and shrinks to a stub: pitch it down (S) or up (N)
# so it stays vertical on screen and shows its length
A.SWORD['pike_s'] = dict(name='창 찌르기 (정면)', effect='pierce', frames=[
    dict(hand=[-.48, .3, 1.2], blade=[0, -1, .12], lean=.1),
    dict(hand=[-.45, .6, 1.3], blade=[0, -1, .2], lean=.3, bob=-.12, crouch=1),
    dict(hand=[-.48, -1.1, 1.15], blade=[0, -.9, -.45], lean=-.42, bob=-.08, lunge=1.2),
    dict(hand=[-.48, -.8, 1.15], blade=[0, -.9, -.45], lean=-.24, bob=-.06, lunge=1)])
A.SWORD['pike_n'] = dict(name='창 찌르기 (후면)', effect='pierce', frames=[
    dict(hand=[-.48, .3, 1.2], blade=[0, -1, .12], lean=.1),
    dict(hand=[-.45, .6, 1.3], blade=[0, -1, .2], lean=.3, bob=-.12, crouch=1),
    dict(hand=[-.48, -1.1, 1.15], blade=[0, -.9, .45], lean=-.42, bob=-.08, lunge=1.2),
    dict(hand=[-.48, -.8, 1.15], blade=[0, -.9, .45], lean=-.24, bob=-.06, lunge=1)])
# effects.py band colours -> (opaque grid char, tag id, legend name); translucent pixels take one of ALPHA_LEVELS
EFFECT_RGB = {(212, 243, 255): ('E', 'effect1', '공격 효과1'), (140, 213, 255): ('F', 'effect2', '공격 효과2'),
              (73, 173, 240): ('G', 'effect3', '공격 효과3')}
ALPHA_LEVELS = (0x60, 0xa0, 0xd7)
TRANSLUCENT_CHARS = '123456789*+=%&'
ATTACK_STYLE = {'pike': 'pike'}  # kit -> style in A.SWORD (default: the infantry sheet's)
# kit -> tag substitution inside the pasted head box (leather cap: iron a b c -> themeA m n o)
HEAD_MAP = {'light': {'a': 'm', 'b': 'n', 'c': 'o', 'r': 'h'}}
HEAD_MAP['pike'] = HEAD_MAP['light']  # r: the base face's dark-red mark (themeB3) -> hair2; themeB stays the buckler's
# (anim, dir, frame) -> [(row0, row1, col0, col1, replace_all)]: paste these base-sheet pixels (blade the 3D pose hides)
BLADE_FIXES = {('attack', 'S', 1): [(9, 17, 13, 21, True), (17, 18, 13, 21, False)], ('attack', 'SW', 1): [(5, 14, 23, 30, True)]}
FIXES = {(kit, *k): v for kit in KITS if kit != 'pike' for k, v in BLADE_FIXES.items()}


def read(path):
    return [l[3:] for l in path.read_text().split('\n') if re.match(r'^\d\d\|', l)]


def stamp(img, out, dx, dy):
    """Lay a 3D effect layer (RGBA, canvas coordinates) onto empty grid cells as effect tags."""
    px = img.load()
    for y in range(img.height):
        for x in range(img.width):
            r, g, b, al = px[x, y]
            if al < 32 or (r, g, b) not in EFFECT_RGB:
                continue
            xx, yy = x + dx, y + dy
            if not (0 <= xx < len(out[0]) and 0 <= yy < len(out)):
                print('clipped effect', xx, yy)
            elif out[yy][xx] == '.':
                ch, tag, name = EFFECT_RGB[(r, g, b)]
                level = min(ALPHA_LEVELS, key=lambda a: abs(a - al))
                out[yy][xx] = ch if al > 0xe8 else ('t', tag, f'{level:02x}', name)


def write_grid(path, out):
    """Replace the pixel rows of a grid file; translucent effect cells get legend lines appended as needed."""
    lines = path.read_text().split('\n')
    table, used, last = {}, set(), -1
    for i, l in enumerate(lines):
        if re.match(r'; \S = ', l):
            last, _ = i, used.add(l[2])
            m = re.match(r'; \S = (effect\d) .*alpha ([0-9a-f]{2})$', l)
            if m:
                table[m.groups()] = l[2]
    new = []

    def char(cell):
        if isinstance(cell, str):
            return cell
        if cell[1:3] not in table:
            ch = next(c for c in TRANSLUCENT_CHARS if c not in used)
            used.add(ch)
            table[cell[1:3]] = ch
            new.append(f'; {ch} = {cell[1]} {cell[3]} 반투명 alpha {cell[2]}')
        return table[cell[1:3]]

    for i, l in enumerate(lines):
        if re.match(r'^\d\d\|', l):
            lines[i] = l[:3] + ''.join(char(c) for c in out[int(l[:2])])
    lines[last + 1:last + 1] = new
    path.write_text('\n'.join(lines))


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument('id')
    ap.add_argument('--kit', default='heavy', choices=KITS)
    ap.add_argument('--base', default='heavy_infantry', help='base sheet whose exported grid (heroes/<base>/grid) supplies heads and effects')
    ap.add_argument('--anims', default=','.join(B.ANIMATIONS))
    ap.add_argument('--dirs', default=','.join(B.DIRECTIONS))
    a = ap.parse_args()
    kit = a.kit
    pad, up = PAD.get(kit, 0), UP_ROWS.get(kit, 0)
    w, h = W + 2*pad, H + up + EXTRA_ROWS.get(kit, 0)
    U.KITS[kit], decorate = KITS[kit]
    A.SELECTED[kit] = ATTACK_STYLE.get(kit) or A.SELECTED['infantry']  # the 3D attack style of the base sheet (tools/unit3d's own 'infantry' unit)
    orig_soldier = U.soldier

    def soldier(p, k, weapon=True):
        out = orig_soldier(p, k, weapon)
        if k == kit:
            decorate(p)
        return out

    U.soldier = soldier
    foot, block = U.foot_unit(kit), Bl.foot(kit)

    view = {}
    effects = []
    if kit == 'pike':
        pierce = A.pierce

        def short_pierce(*args, **kw):  # the burst must fit the cell: a short cone and a small ring at the spear tip
            out = pierce(*args, **kw)
            for e in out:
                if e['kind'] == 'cone':
                    e.update(length=.55, width=.45)
                elif e['kind'] == 'ring':
                    e.update(center=e['center'] - e['axis']*.1, radius=.28)
            return out

        A.pierce = short_pierce
        draw = B.E.draw
        B.E.draw = lambda *args: effects.append(draw(*args)) or effects[-1]

    def build(row, f, style=None):
        if row == B.BLOCK:
            return block(f, Bl.SELECTED)
        if kit == 'pike' and row == U.ATTACK and view['d'] in ('N', 'S'):
            style = 'pike_' + view['d'].lower()
        return foot(row, f, style=style) if style else foot(row, f)

    captured = []
    to_img = Hd.to_img
    Hd.to_img = lambda g: (captured.append([r[:] for r in g]), to_img(g))[1]
    head = Hd.heads({d: ['.' * 13] * 12 for d in Hd.IRON})  # body only; heads come from the base grid
    out_root, ref_root = HERE.parent / 'heroes' / a.id / 'grid', HERE.parent / 'heroes' / a.base / 'grid'
    plume = re.compile(r'#(?:j|jk|kj|jj)#')
    solid = lambda c: c != '.' and c not in EFFECT
    for d in a.dirs.split(','):
        view['d'] = d
        captured.clear()
        B.frame(build, head, B.SPEC, d, 0, 0, lift=0)
        bottom = max(y for y, row in enumerate(captured[-1]) for c in row if c != '.')
        dx, dy = 28 + pad - B.SPEC['pivot'][0], 38 + up - bottom  # idle frame 0 puts the feet on row 38
        for anim in a.anims.split(','):
            for n in range(4):
                captured.clear()
                effects.clear()
                B.frame(build, head, B.SPEC, d, B.ANIMATIONS.index(anim), n, lift=0)
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
                ref = read(ref_root / anim / f'{d}-{n}.txt')
                top = next(y for y, r in enumerate(ref) if plume.search(r[20:40]))
                px = 20 + plume.search(ref[top][20:40]).start() + 1
                for y in range(top, min(top + 12, H)):
                    for x in range(px - 7, px + 8):
                        if solid(ref[y][x]):
                            out[y + up][x + pad] = HEAD_MAP.get(kit, {}).get(ref[y][x], ref[y][x])
                for r0, r1, c0, c1, replace in FIXES.get((kit, anim, d, n), []):
                    for y in range(r0, r1):
                        for x in range(c0, c1):
                            if solid(ref[y][x]) and (replace or out[y + up][x + pad] == '.'):
                                out[y + up][x + pad] = ref[y][x]
                for y in range(H):
                    for x in range(W):
                        if ref[y][x] in EFFECT and out[y + up][x + pad] == '.' and not (kit == 'pike' and anim == 'attack'):
                            out[y + up][x + pad] = ref[y][x]
                if kit == 'pike' and anim == 'attack':
                    for under, over in effects:
                        for layer in (over, under):
                            stamp(layer, out, dx, dy)
                write_grid(out_root / anim / f'{d}-{n}.txt', out)
        print(d, 'done', flush=True)


if __name__ == '__main__':
    main()
