"""선술사: the 방술사 body (mounted_kits/sorcerer.py: 화양건, wide-sleeved faction robe) one tier up — a person-height
wooden staff with a curled, knotted top and the gourd hanging from the hook, white cloud trim on the hem and cuffs,
and two long white ribbons hanging from the cap band past the shoulders.

Attack: raise the staff overhead; light descends onto its top and flares (frame 2), the flare spreads and fades (3).
Tags: staff themeC (wood), gourd themeB, trim / ribbons / jade iron (white), the rest as the 방술사.
"""
import math

import numpy as np

import grid3d_mounted as M
from mounted_kits import sorcerer as S

U, A, B, Bl, Hd, R = M.U, M.A, M.B, M.Bl, M.Hd, M.R
m, V = U.m, U.V

WOOD, KNOT, TRIM, CORD = '#8a5a34', '#6e4527', '#ede1c0', '#5e3b22'
LENGTH = 2.7
IDLE_HANDS = [[-.7, -.34, 1.22], [.14, -.42, 1.06]]  # staff hand held out so the staff clears the face in E/SE
IDLE_ELBOWS = [[-.7, -.06, 1.04], [.52, -.14, 1.04]]
OUT = S.unit([-1, -.8, 0])  # the hook bends out (away from the body) and forward
# attack: staff hand, staff direction, grip (distance from the staff's foot to the hand), body keys
ATTACK = [dict(hand=[-.55, -.4, 1.35], dir=[0, -.12, 1], grip=.8, lean=.02),
          dict(hand=[-.45, -.2, 1.8], dir=[0, -.05, 1], grip=1.1, lean=.1, crouch=.2),
          dict(hand=[-.5, -.5, 1.62], dir=[0, -.5, 1], grip=1.1, lean=-.12, bob=-.03, lunge=.4),
          dict(hand=[-.55, -.42, 1.4], dir=[0, -.3, 1], grip=.95, lean=-.06, lunge=.3)]


def ribbons(d):
    """(y, x) pixels of the two white ribbons, from the cap band (row 6) down past the shoulders."""
    def strip(x, y0, y1):
        return [(y, x) for y in range(y0, y1)]
    return {'S': strip(1, 6, 17) + strip(2, 6, 16) + strip(14, 6, 16) + strip(15, 6, 17),
            'SW': strip(1, 6, 15) + strip(2, 6, 16) + strip(15, 6, 17) + strip(16, 6, 16),
            'W': strip(15, 6, 17) + strip(16, 6, 16),
            'NW': strip(9, 7, 18) + strip(12, 7, 17),
            'N': strip(6, 7, 18) + strip(10, 7, 18)}[d]


def heads():
    hs = S.heads()
    for d, rows in hs['grids'].items():
        g = [list(r) for r in rows]
        for y, x in ribbons(d):
            while y >= len(g):
                g.append(['.']*17)
            g[y][x] = '6'
        hs['grids'][d] = [''.join(r) for r in g]
    return hs


def staff(hand, d, grip):
    """Staff along d through the hand; the hook at the top curls out, the gourd hangs from its tip on a cord."""
    d = S.unit(d)
    foot, top = hand - d*grip, hand + d*(LENGTH - grip)
    m.rod(foot, top, .065, WOOD)
    for z in (.55, 1.6):
        m.ell(foot + d*z, [.09]*3, KNOT)
    out = OUT - d*np.dot(OUT, d)
    out = S.unit(out) if np.linalg.norm(out) > .1 else V([-1, 0, 0])
    c = top + out*.22
    pts = [c - out*.22*math.cos(a) + d*.22*math.sin(a) for a in np.linspace(0, 1.25*math.pi, 8)]
    for a, b in zip(pts, pts[1:]):
        m.rod(a, b, .075, WOOD)
    m.ell(pts[3], [.1]*3, KNOT)
    tip = pts[-1]
    m.rod(tip, tip + V([0, 0, -.12]), .025, CORD)
    g = tip + V([0, 0, -.3])
    m.ell(g, [.16, .16, .17], S.GOURD)
    m.ell(g + V([0, 0, .2]), [.1]*3, S.GOURD)
    return top + d*.2


def trim(p, T, seated):
    """White cloud trim: scalloped bumps round the hem, a white ring at each cuff."""
    if seated:
        f = (V(p['feet'][0]) + V(p['feet'][1]))/2
        c, (rx, ry) = V([f[0], f[1] + .1, .1]), (.44, .16)
    else:
        fl, fr = V(p['feet'][0]), V(p['feet'][1])
        spread = abs(fl[1] - fr[1])
        c = V([0, (fl[1] + fr[1])/2 + .02, hem_height(spread) + .03])
        rx, ry = .5, .37 + .4*spread
    for a in np.linspace(0, 2*math.pi, 13)[:-1]:
        m.ell(c + V([rx*math.cos(a), ry*math.sin(a), 0]), [.11, .11, .07], TRIM)
    m.ell(c, [rx - .01, ry - .01, .04], TRIM)
    for i in range(2):
        el, ha = V(T(p['elbows'][i])), V(T(p['hands'][i]))
        k = S.unit(ha - el)
        S.cone(ha - k*.01, ha + k*.07, .235, .245, TRIM)


def hem_height(spread):
    return .44 if spread > .1 else .24  # walking: the hem rises well clear so the striding legs show under the trim


def robe(p, T, seated):
    if seated:
        return S.robe(p, T, True)
    fl, fr = V(p['feet'][0]), V(p['feet'][1])
    spread = abs(fl[1] - fr[1])
    S.frustum(T([0, 0, .98]), (.42, .3), V([0, (fl[1] + fr[1])/2 + .02, hem_height(spread)]), (.5, .36 + .4*spread),
              U.BLUE)


def body(p, row, f, d, grip, effects=()):
    seated = row == 4
    _, head = U.soldier(p, 'mystic', weapon=False)
    T = U.transform(p)
    for i in range(2):
        m.rod(V(p['knees'][i]), V(p['feet'][i]), .135, U.BROWN)
    robe(p, T, seated)
    S.sleeves(p, T)
    trim(p, T, seated)
    staff(V(T(p['hands'][0])), d, grip)
    return U.result(m.meshes, head, row, f, effects)


def attack_pose(k):
    p = U.pose(0, 0)
    p['row'] = U.ATTACK
    p['hands'] = [list(k['hand']), list(IDLE_HANDS[1])]
    p['elbows'] = [S.elbow_for(k['hand']), list(IDLE_ELBOWS[1])]
    p['lean'], p['bob'] = k.get('lean', 0), k.get('bob', 0)
    A.stance(p, k)
    return p


def staff_top(k):
    p = attack_pose(k)
    return U.transform(p)(p['hands'][0]) + S.unit(k['dir'])*(LENGTH - k['grip'] + .1)


def attack_effects(f):
    """Frame 2: beams of light falling onto the staff's top and a flare there; frame 3: the flare spreads, fading."""
    if f < 2:
        return []
    top = staff_top(ATTACK[f])
    if f == 2:
        beams = [(top + V([x, y, .62]), top + V([x*.3, y*.3, .12])) for x, y in
                 ((-.3, 0), (.3, 0), (0, -.25), (0, .25), (-.18, -.18), (.18, .18))]
        return [dict(kind='lines', segments=beams, alpha=1, layer='under'),
                dict(kind='ring', center=top, axis=V([0, 0, 1]), radius=.3, alpha=1, layer='over'),
                dict(kind='ring', center=top, axis=V([0, -1, 0]), radius=.2, alpha=1, layer='over')]
    rays = [(top + V([math.cos(a), 0, math.sin(a)])*.25, top + V([math.cos(a), 0, math.sin(a)])*.6)
            for a in np.linspace(0, 2*math.pi, 9)[:-1]]
    return [dict(kind='lines', segments=rays, alpha=.45, layer='under'),
            dict(kind='ring', center=top, axis=V([0, 0, 1]), radius=.5, alpha=.45, layer='over')]


def idle_pose(row, f):
    p = U.pose(row, f)
    p['hands'], p['elbows'] = [list(h) for h in IDLE_HANDS], [list(e) for e in IDLE_ELBOWS]
    return p


def planted(p):
    """(direction, grip) of the staff standing on the ground beside the right foot."""
    hand = V(U.transform(p)(p['hands'][0]))
    foot = V([hand[0] - .08, hand[1] + .02, .02])
    t = U.VIEW['t']
    if t is not None:  # the staff's foot may not sit lower on screen than the feet (they set the pivot row)
        up = S.unit(V([0, 0, 1]) - t*t[2])
        lowest = max(-np.dot(V(f) + [0, -.2, -.1], up) for f in p['feet'])
        over = -np.dot(foot, up) - lowest
        if over > 0:
            foot[2] += over/up[2]
    return hand - foot, np.linalg.norm(hand - foot)


def mystic(row, f):
    if row == U.ATTACK:
        k = ATTACK[f]
        return body(attack_pose(k), row, f, k['dir'], k['grip'], attack_effects(f))
    if row == B.BLOCK:  # guard: the staff held level across the body, both ends clear of the face
        k = Bl.SWORD[Bl.SELECTED]['frames'][f]
        p = Bl.foot_pose(k)
        res = body(p, 0, f, k['blade'], 1.25)
        return Bl.displace(res, k.get('sway', 0), k.get('back', 0))
    if row in (0, 1):
        p = idle_pose(row, f)
        return body(p, row, f, *planted(p))
    if row == U.HIT:
        p = U.hit_pose(f, idle_pose(0, 0))
        return body(p, row, f, [-.1, .25, 1], 1.0)
    p = U.pose(row, f)  # exhausted: seated, the staff leaning on the shoulder, its foot on the ground
    p['hands'][0], p['elbows'][0] = [-.58, -.3, 1.02], [-.66, -.05, 1.12]
    return body(p, row, f, [-.15, .15, 1], .5)


def make():
    U.KITS['mystic'] = dict(cloth=U.BLUE, tabard=U.LIGHT_BLUE, guard=U.BLUE, shoulder=U.BLUE, shield=False,
                            weapon=None, robe=U.BLUE)
    R.add_ramps({WOOD: 'kllm', KNOT: 'kkll', CORD: 'kkkl', TRIM: '5566', S.GOURD: 'ABCC'})
    R.COLOR.update({'x': (42, 30, 25), 'y': (71, 52, 42)})
    M.KEY2TAG.update(x='i', y='h', q='j', p='k')  # hair keys; the tired face's sweat drop -> faction light

    def build(row, f, style=None):
        return mystic(row, f)
    return build, heads()


M.register('mystic', make)
