"""서생: young civil officer. Bare topknot with a dark 망건 and a small faction hair tie, knee-length faction robe over
hemp trousers, brown cloth sash, a rolled bamboo-slip scroll (죽간) in the right hand instead of the 참모's fan.

Body: a slimmer, unarmoured copy of units.soldier (no steel guard, no tabard). The scroll follows the weapon hand
along the pose's blade direction, tipped outward so it stands clear of the robe. Attack (stratagem casting): both
hands take the rolled scroll (0), pull it open (1), thrust the open sheet at the enemy with a burst (2), hold (3).
The open sheet turns a little toward the camera in side views (like units.facing for blades) so it never renders
edge-on. Block: blocks.SWORD 'guard' poses, the scroll held across like the fan.

Tags: hair 7 8 9 -> hair1-3, robe and tie -> faction, trousers (hemp S T U) -> themeC, sash and shoes (leather)
-> themeA, scroll (grid3d.BUCKLER keys A B C) -> themeB, 망건 and scroll cords -> iron3/iron2, collar -> iron1.
"""
import numpy as np

import grid3d_mounted as M

G, U, A, B, Bl, Hd, R = M.G, M.U, M.A, M.B, M.Bl, M.Hd, M.R
m, V = U.m, U.V

ROBE, BROWN, SKIN = U.BLUE, U.BROWN, U.SKIN
TROUSERS = U.HEMP          # ramp UTSS
SCROLL = G.BUCKLER         # ramp ABCC
CORD = '#2a2d36'           # units.BLACK, ramp 1234
COLLAR = '#ede1bd'         # ramp 5566
UP = V([0, 0, 1])
OUT = V([-.3, 0, 0])       # tips the held scroll away from the body


# ---- head ----------------------------------------------------------------------------------
def head_rows(rows):
    """Topknot head: hair -> hair keys 7/8/9, lowest band row -> dark 망건, other band rows -> hair, band tails
    removed, two faction pixels (a narrow tie) in the bun's middle row."""
    rows = [list(r) for r in rows]
    band = [y for y, r in enumerate(rows) if any(c in 'mlk' for c in r)]
    for y in band:
        r = rows[y]
        last = max(i for i, c in enumerate(r) if c == '0')
        for x in range(last + 1, len(r)):
            if r[x] in 'mlk':
                r[x] = '.'
    low = max(y for y in band if any(c in 'mlk' for c in rows[y]))
    for y, r in enumerate(rows):
        for x, c in enumerate(r):
            if c in 'mlk':
                r[x] = {'m': '1', 'l': '1', 'k': '0'}[c] if y == low else {'m': '7', 'l': '8', 'k': '9'}[c]
            elif c in 'fg':
                r[x] = {'f': '8', 'g': '7'}[c]
    r = rows[2]
    xs = [x for x, c in enumerate(r) if c in '78']
    mid = xs[len(xs)//2]
    r[mid] = 'Y'
    if mid + 1 in xs:
        r[mid + 1] = 'X'
    return [''.join(r) for r in rows]


HEADS = Hd.heads({d: head_rows(r) for d, r in Hd.TOPKNOT.items()})


# ---- body ----------------------------------------------------------------------------------
def body(p):
    m.meshes = []
    T = U.transform(p)
    for i in range(2):
        hip = T([(-.22 if i == 0 else .22), 0, .77])
        knee, foot = V(p['knees'][i]), V(p['feet'][i])
        m.rod(hip, knee, .15, TROUSERS)
        m.rod(knee, foot, .12, TROUSERS)
        m.ell(foot + [0, -.035, 0], [.13, .2, .1], BROWN)
    m.ell(T([0, -.01, .74]), [.4, .3, .36], ROBE)       # robe skirt down to the knee
    m.ell(T([0, 0, .94]), [.4, .26, .3], ROBE)
    m.ell(T([0, 0, 1.26]), [.37, .25, .36], ROBE)
    m.box(T([0, -.235, 1.36]), [.16, .04, .2], COLLAR)  # white inner collar at the neck opening
    m.box(T([0, 0, 1.01]), [.75, .53, .09], BROWN)      # plain cloth sash with a short tie end
    m.box(T([-.12, -.27, .9]), [.09, .04, .2], BROWN)
    m.rod(T([0, 0, 1.48]), T([0, 0, 1.68]), .13, SKIN)
    for i in range(2):
        sh, el, ha = T([(-.38 if i == 0 else .38), 0, 1.4]), T(p['elbows'][i]), T(p['hands'][i])
        m.ell(sh, [.17, .2, .17], ROBE)
        m.rod(sh, el, .13, ROBE)
        m.rod(el, ha, .13, ROBE)
        m.ell(ha, [.115]*3, SKIN)
    return T


def rolled(c, d, length=.7):
    """Rolled bundle centred on c along d: pale slips, two dark cords near the ends."""
    a, b = c - d*length/2, c + d*length/2
    m.rod(a, b, .12, SCROLL)
    for t in (.2, .8):
        q = a + (b - a)*t
        m.rod(q - d*.03, q + d*.03, .13, CORD)


def obox(c, x, y, z, color):
    """Box with edge vectors x, y, z (full lengths) centred on c."""
    pts = [c + (sx*x + sy*y + sz*z)/2 for sx in (-1, 1) for sy in (-1, 1) for sz in (-1, 1)]
    for ids in [[0, 1, 3, 2], [4, 6, 7, 5], [0, 4, 5, 1], [2, 3, 7, 6], [0, 2, 6, 4], [1, 5, 7, 3]]:
        m.poly([pts[i] for i in ids], color)


SHEET_H = .56


def sheet(r, l, normal):
    """Open scroll between the hands r (right) and l (left): flat slips, cords across, a roll at each end."""
    across = l - r
    c = (r + l)/2
    n = normal*.04
    obox(c, across, n, UP*SHEET_H, SCROLL)
    for z in (-.17, .17):
        obox(c + UP*z, across, n*1.6, UP*.05, CORD)
    for e in (r, l):
        m.rod(e - UP*(SHEET_H/2 + .03), e + UP*(SHEET_H/2 + .03), .075, SCROLL)


# attack frames: c = sheet centre, w = hand spread (0: rolled bundle held in both hands)
ATTACK = [dict(c=[0, -.44, 1.18], w=.22, lean=.04),
          dict(c=[0, -.52, 1.24], w=.5, lean=.08, crouch=.3, bob=-.02),
          dict(c=[0, -.86, 1.28], w=.8, lean=.22, lunge=.5, bob=-.04),
          dict(c=[0, -.74, 1.24], w=.8, lean=.14, lunge=.35, bob=-.02)]
SHOULDERS = (V([-.38, 0, 1.4]), V([.38, 0, 1.4]))


def facing_normal():
    """Forward (-y), turned toward the camera by its sideways component so the sheet shows in E/W views."""
    t = U.VIEW['t']
    n = V([0, -1, 0]) + (V([t[0], 0, 0])*.8 if t is not None else 0)
    return n/np.linalg.norm(n)


def casting(f):
    k = ATTACK[f]
    p = U.pose(0, 0)
    p['row'] = U.ATTACK
    p['lean'], p['bob'] = k['lean'], k.get('bob', 0)
    A.stance(p, k)
    n = facing_normal()
    side = V([-n[1], n[0], 0])
    c = V(k['c'])
    hands = [c - side*k['w']/2, c + side*k['w']/2]
    p['hands'] = [h.tolist() for h in hands]
    p['elbows'] = [((SHOULDERS[0] + hands[0])/2 + [-.16, .06, -.1]).tolist(),
                   ((SHOULDERS[1] + hands[1])/2 + [.16, .06, -.1]).tolist()]
    T = body(p)
    r, l = T(hands[0]), T(hands[1])
    effects = []
    if f == 0:
        rolled((r + l)/2 + V([0, -.06, 0]), UP)
    else:
        sheet(r, l, n)
    if f >= 2:
        d = A.unit(V([0, -1, -.08]))
        tip = (r + l)/2 + d*.1
        t = U.VIEW['t']
        if t is not None and abs(t[1]) > .9*np.hypot(t[0], t[1]):
            # facing the camera (S) or away (N) the burst would sit on the body, where effect tags are never
            # stamped: a wide shock ring around the sheet instead, reaching past the shoulders
            effects = [dict(kind='ring', center=tip + d*.3, axis=d, radius=.8 if f == 2 else .95,
                            alpha=1 if f == 2 else .5, layer='over')]
        else:
            effects = [e for e in A.pierce(tip, d, tip, alpha=1 if f == 2 else .6, burst=f == 2)
                       if e['kind'] != 'lines']
            for e in effects:
                if e['kind'] == 'cone':
                    e.update(length=.95, width=.6)
                elif e['kind'] == 'ring':
                    e.update(radius=.34 if f == 2 else .44)
    return U.result(m.meshes, T(U.HEAD), U.ATTACK, f, effects)


def holding(p, block=False):
    T = body(p)
    d = A.unit(V(p['blade']) if block else V(p['blade']) + OUT)
    rolled(T(p['hands'][0]) + d*.08, d)
    return T


def scholar(row, f):
    if row == U.ATTACK:
        return casting(f)
    p = U.hit_pose(f, U.pose(0, 0)) if row == U.HIT else U.pose(row, f)
    T = holding(p)
    return U.result(m.meshes, T(U.HEAD), row, f)


def block(f):
    k = Bl.SWORD[Bl.SELECTED]['frames'][f]
    p = Bl.foot_pose(k)
    T = holding(p, block=True)
    return Bl.displace(U.result(m.meshes, T(U.HEAD), 0, f), k.get('sway', 0), k.get('back', 0))


def make():
    R.add_ramps({TROUSERS: 'UTSS', SCROLL: 'ABCC', CORD: '1234', COLLAR: '5566'})
    M.KEY2TAG.update({'7': 'g', '8': 'h', '9': 'i', 'S': 's', 'T': 't', 'U': 'u',
                      'p': 'F', 'q': 'E'})  # sweat drops of the tired face: effect colours, not horse tags
    U.WEAPON_COLORS.add(SCROLL)  # the scroll is drawn over the stamped head where it passes in front of it

    def build(row, f, style=None):
        return block(f) if row == B.BLOCK else scholar(row, f)
    return build, HEADS


M.register('scholar', make)
