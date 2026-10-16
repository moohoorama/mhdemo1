"""의적: the 흉적 (mounted_kits/ruffian.py) as a bandit chief: a faction cape from the shoulders instead of the tiger
pelt, a brown leather cuirass with front straps over the hemp top, a neat short beard; faction headband and the
heavy dadao (with the 흉적's two-handed overhead chop and slanted guard) are kept.

The cape is an upper back panel fixed to the torso plus two lower halves, split down the middle, that hang to just
below the knees and swing back with each leg's stride (the legs show under and between them in N/NW walk). Seated (exhausted)
the halves spread on the ground behind instead of standing up.
Everything that touches shared modules is patched inside make() (via ruffian.make() for the 흉적 parts).
"""
import numpy as np

import grid3d_mounted as M
from mounted_kits import ruffian as RK

G, U, A, B, Bl, Hd = M.G, M.U, M.A, M.B, M.Bl, M.Hd
m, V = U.m, U.V

LEATHER, LEATHER_DARK = U.BROWN, '#5a3018'   # cuirass (ramp fghh) and its rim / straps (ffgg): themeA
CAPE = U.BLUE                                 # faction ramp W X Y Z
KIT = dict(RK.KIT)


def slab(quad, thick, color):
    """A flat quad given thickness (a hexahedron), so a cape panel still shows edge-on in E/W views."""
    q = [V(c) for c in quad]
    n = np.cross(q[1] - q[0], q[3] - q[0])
    n = n/np.linalg.norm(n)*thick/2
    a, b = [c + n for c in q], [c - n for c in q]
    m.poly(a, color)
    m.poly(b[::-1], color)
    for i in range(4):
        j = (i + 1) % 4
        m.poly([a[i], a[j], b[j], b[i]], color)


def armour(T):
    m.ell(T([0, -.02, 1.2]), [.4, .3, .3], LEATHER)
    m.box(T([0, -.02, 1.0]), [.7, .54, .07], LEATHER_DARK)
    for s in (-1, 1):
        m.rod(T([s*.24, -.2, 1.42]), T([s*.3, .2, 1.42]), .07, LEATHER_DARK)    # shoulder straps
        m.rod(T([s*.2, -.31, 1.36]), T([s*.15, -.33, 1.06]), .04, LEATHER_DARK)  # front straps down the chest


def cape(p, T):
    m.ell(T([0, .05, 1.47]), [.5, .3, .12], CAPE)
    for s in (-1, 1):  # drape over the shoulder tops: the faction colour shows from the front
        m.ell(T([s*.38, .04, 1.5]), [.25, .28, .15], CAPE)
    m.box(T([0, .31, 1.12]), [.96, .08, .62], CAPE)
    seated = p.get('seated') or p['row'] == 4
    for i, s in enumerate((-1, 1)):
        xs = (s*.08, s*.52)
        tops = [T([x, .34, .86]) for x in xs]
        if seated:
            bottoms = [V([t[0]*1.15, t[1] + .55, .05]) for t in tops]
        else:
            swing = .1 + .9*max(0., float(p['feet'][i][1]))  # the back leg kicks its half out and up
            bottoms = [T([x, .36 + swing, .55 + .4*swing]) for x in xs]
            for b in bottoms:
                b[2] = max(b[2], .05)
        slab([tops[0], tops[1], bottoms[1], bottoms[0]], .07, CAPE)


def short_beard(rows, skin='ij'):
    """Neat short beard: jaw and chin below the mouth line, short sideburns, one row under the chin."""
    g = [list(r) for r in rows]
    eyes = Hd.find_eyes(g, skin)
    if not eyes:
        return rows
    w = len(g[0])
    ey = eyes[0][0]
    eye_x = {x for _, x in eyes}
    profile = len(eyes) == 1
    chin = max(y for y, row in enumerate(g) if any(c in skin for c in row))
    for y in range(ey + 1, ey + 3):
        xs = [x for x, c in enumerate(g[y]) if c in skin]
        if xs:
            for x in ([xs[-1]] if profile else [xs[0], xs[-1]]):
                if x - 1 not in eye_x and x + 1 not in eye_x:
                    g[y][x] = 'B'
    for y in range(ey + 3, chin + 1):
        for x in range(w):
            if g[y][x] in skin:
                g[y][x] = 'B'
    cols = [x for x, c in enumerate(g[chin]) if c == 'B']
    left, right = min(cols), max(cols)
    g += [['.'] * w for _ in range(max(0, chin + 2 - len(g)))]
    for x in range(left + 1, right):
        g[chin + 1][x] = 'B'
    Hd.outline_beard(g, chin + 1)
    for y in range(len(g)):
        for x in range(w):
            if g[y][x] == 'B':
                g[y][x] = '1' if y == chin + 1 else 'f'
    return [''.join(r) for r in g]


def heads():
    grids = {d: Hd.recolor(short_beard(r), RK.HAIR_KEYS) for d, r in Hd.TOPKNOT.items()}
    return Hd.heads(grids, Hd.BAND)


def make():
    RK.make()  # dadao, chop styles, two-hand grip, beard mouth hint, key -> tag mapping
    U.RAMPS[LEATHER_DARK] = 'ffgg'
    B.R.add_ramps({LEATHER_DARK: 'ffgg'})
    U.KITS['outlaw'] = KIT
    A.SELECTED['outlaw'] = 'ruffian_chop'

    soldier = U.soldier

    def dressed(p, k, weapon=True):
        out = soldier(p, k, weapon)
        if k == 'outlaw':
            T = U.transform(p)
            RK.open_chest(T)
            armour(T)
            cape(p, T)
        return m.meshes, out[1]

    U.soldier = dressed
    foot = U.foot_unit('outlaw')

    def block(f):
        k = RK.GUARD[f]
        faces, head = U.soldier(Bl.foot_pose(k), 'outlaw')
        return Bl.displace(U.result(faces, head, 0, f), k.get('sway', 0), k.get('back', 0))

    def build(row, f, style=None):
        if row == B.BLOCK:
            return block(f)
        t = U.VIEW['t']
        if row == U.ATTACK and style is None and t is not None and abs(t[0]) < .1:  # S or N view
            style = 'ruffian_chop_diagonal'
        return foot(row, f, style=style) if style else foot(row, f)
    return build, heads()


M.register('outlaw', make)
