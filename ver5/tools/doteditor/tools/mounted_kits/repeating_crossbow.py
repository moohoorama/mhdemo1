"""연노병: the 노병 (crossbow.py) with a 제갈노 repeating crossbow, a steel skullcap over the hood and one steel pauldron.

The 제갈노 is the 노병's crossbow (prod, string, stock) with a tall box magazine standing on the fore-end, three
racked bolts showing their tips above it, and a cocking lever pivoting at the magazine's rear foot. Magazine and lever
are separate parts so the attack can move them: fire, pump the lever (the magazine feeds the next bolt), fire again,
so three bolts are in the air at once (three streaks).

The weapon rides lower than the 노병's (hip/chest height): at the 노병's height the magazine stands behind the stamped
2D head in S and over the face in profile.
"""
import itertools

import numpy as np

import grid3d_mounted as M
from mounted_kits import crossbow as CB

G, U, A, B, Bl, Hd = M.G, M.U, M.A, M.B, M.Bl, M.Hd
m, V = U.m, U.V

KIT = 'repeating_crossbow'
MAG = '#704022'        # magazine: the bolt case's dark wood (themeA)
LEVER = CB.WOOD        # pale wood like the stock (themeB)
SHAFT = CB.WOOD
TIP = '#c9cdd0'        # light steel (iron); not in WEAPON_COLORS, so the 2D head stays over the tips
PLATE = '#c9cdd0'      # light steel pauldron: stands out from the archer kit's dark steel shoulder caps

IDLE_REAR, IDLE_FRONT = [0, -.3, 1.02], [0, -.8, 1.02]
LEVER_UP, LEVER_BACK = (.7, -.05), (.3, -.62)          # lever end = pivot + top*a + aim*b
# attack: aim, fire 1, pump lever and fire 2 + 3 (three bolts in flight), lever back up while lowering
ATTACK = [dict(rear=[0, -.32, 1.12], front=[0, -.82, 1.12], lean=.12, bob=-.03, crouch=.4),
          dict(rear=[0, -.24, 1.14], front=[0, -.74, 1.14], lean=.04, bob=-.03, crouch=.4, shots=1),
          dict(rear=[0, -.3, 1.12], front=[0, -.8, 1.12], lean=.14, bob=-.04, crouch=.5, shots=3, lever=LEVER_BACK,
               feed=.07),
          dict(rear=[0, -.3, 1.04], front=[0, -.66, .92], lean=.06, bob=-.02, lever=(.5, -.3))]
SHOTS = {1: [(.9, 0)], 3: [(1.4, .24), (.95, -.24), (.45, 0)]}  # (bolt tail past the front hand, raise), oldest first


def obox(c, axes, size, color):
    """Box at c with half-extents size along the three unit axes (m.box is axis-aligned only)."""
    pts = [c + sum(sg*hs*ax for sg, hs, ax in zip(signs, size, axes)) for signs in itertools.product((-1, 1), repeat=3)]
    for ids in [[0, 1, 3, 2], [4, 6, 7, 5], [0, 4, 5, 1], [2, 3, 7, 6], [0, 2, 6, 4], [1, 5, 7, 3]]:
        m.poly([pts[i] for i in ids], color)


def magazine(aim, side, top, base, feed=0, h=.44, l=.48, w=.2):
    """Box magazine standing on the stock (base = stock top under its centre), racked bolt tips above it.
    feed: how far the stack has dropped (a bolt fed onto the stock)."""
    c = base + top*(h/2)
    obox(c, (aim, side, top), (l/2, w/2, h/2), MAG)
    obox(c - top*(h/2 - .05), (aim, side, top), (l/2 + .03, w/2 + .03, .05), U.STEEL)   # foot band on the stock
    for k, rise in ((-.4, 0), (0, .09), (.4, 0)):   # middle bolt higher: the tips stay apart in the diagonals
        p = c + aim*k*l + top*(h/2 + rise - feed)
        m.rod(p - top*.04, p + top*.12, .032, SHAFT)
        m.rod(p + top*.1, p + top*.26, .055, TIP)


def lever(aim, side, top, pivot, pose=LEVER_UP):
    a, b = pose
    end = pivot + top*a + aim*b
    m.rod(pivot, end, .07, LEVER)
    m.ell(end, [.09]*3, LEVER)
    m.ell(pivot, [.075]*3, U.STEEL)


def repeater(rear, front, loaded=True, lever_pose=LEVER_UP, feed=0):
    aim = CB.crossbow(rear, front, loaded)
    side, top = CB.frame_of(aim)
    base = front + top*.22 + aim*.08          # stock top (lift .07 + half the stock and the bolt on it)
    magazine(aim, side, top, base, feed)
    lever(aim, side, top, base - aim*.31 + top*.05, lever_pose)
    return aim


def armour(T):
    CB.bolt_case(T)
    sh = T([.4, 0, 1.42])
    m.ell(sh + V([.03, 0, .06]), [.27, .29, .17], PLATE)
    m.ell(T([.46, 0, 1.29]), [.25, .27, .1], U.STEEL)


def volley(fr, aim, shots):
    """Flying bolts and their streaks: the newest leaves the muzzle, older ones are further out and staggered."""
    side = A.unit(np.cross(aim, [0, 0, 1]))
    up = np.cross(side, aim)
    effects = []
    for flight, rise in SHOTS[shots]:
        tail = fr + aim*flight + up*rise
        CB.flying_bolt(tail, aim)
        start = tail - aim*.75
        effects.append(dict(kind='wedge', start=start, end=tail, width=.26, alpha=1, layer='under'))
    return effects


def soldier(row, f):
    p = U.pose(0 if row == U.HIT else row, 0 if row == U.HIT else f)
    rear, front = IDLE_REAR, IDLE_FRONT
    k = ATTACK[f] if row == U.ATTACK else None
    if row == 1:
        p['bob'] = [-.025, .035, -.025, .035][f]
    if k:
        rear, front = k['rear'], k['front']
        p['lean'], p['bob'] = k.get('lean', 0), k.get('bob', 0)
        A.stance(p, k)
    if row == 4:
        rear, front = [-.4, -.25, 1.0], [.35, -.55, .9]
    p['hands'], p['elbows'] = [list(rear), list(front)], CB.elbows(rear, front)
    if row == U.HIT:
        U.hit_lean(p, f)
    _, head = U.soldier(p, KIT, weapon=False)
    T = U.transform(p)
    r, fr = T(rear), T(front)
    k = k or {}
    aim = repeater(r, fr, lever_pose=k.get('lever', LEVER_UP), feed=k.get('feed', 0))
    effects = volley(fr, aim, k['shots']) if k.get('shots') else []
    armour(T)
    return U.result(m.meshes, head, row, f, effects)


def block(f):
    k = Bl.BOW[Bl.SELECTED]['frames'][f]
    p = U.pose(0, 0)
    p['hands'], p['elbows'] = [list(k['hand']), list(k['off'])], CB.elbows(k['hand'], k['off'])
    p['lean'], p['bob'] = k.get('lean', 0), k.get('bob', 0)
    A.stance(p, k)
    _, head = U.soldier(p, KIT, weapon=False)
    T = U.transform(p)
    repeater(T(p['hands'][0]), T(p['hands'][1]))
    armour(T)
    return Bl.displace(U.result(m.meshes, head, 0, f), k.get('sway', 0), k.get('back', 0))


# steel skullcap (iron keys) over the hood's top rows; the hood cloth still shows on the sides and the back
HELM_TOPS = {
    'S': ['.....000.....', '...0345530...', '..034555430..', '.02344444320.'],
    'SW': ['......000....', '....0345530..', '...034555430.', '..0234444320.'],
    'W': ['.......000...', '.....0345530.', '....034555430', '...0234444320'],
    'NW': ['......000....', '....0345530..', '...034555430.', '..0234444320.'],
    'N': ['.....000.....', '...0345530...', '..034555430..', '.02344444320.'],
}


def make():
    U.KITS[KIT] = dict(U.KITS['archer'])
    M.KEY2TAG[M.MOUTH] = 'h'  # themeB is the pale wood: the mouth takes hair2
    A.BOW[KIT] = dict(name='연노', effect='streak', frames=[])

    def build(row, f, style=None):
        return block(f) if row == B.BLOCK else soldier(row, f)
    helm = {d: tops + Hd.HOOD[d][len(tops):] for d, tops in HELM_TOPS.items()}
    return build, Hd.heads(helm, Hd.HOOD_CLOTH)


M.register(KIT, make)
