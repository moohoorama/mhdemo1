"""권성: the 권사 (mounted_kits/pugilist.py) in lion armour (사자 갑주), the third tier of 무도가 → 권사 → 권성.

Same body, bald head, poses (idle, walk, hit, exhausted, crossed-gauntlet block) and martial uniform as the 권사, with:
gold lion-head pauldrons (a dark mane ring round a gold face with open dark jaws and eyes, so the gold stays apart
from the bald head), a small gold-rimmed steel chest plate, gold cuff bands on the faction gauntlets, and a long
darker-faction sash knotted in front instead of the dark belt.

Attack: a palm strike (장풍): chamber, gather, step in and thrust the open palm, an energy orb flies out (frame 2 at the
palm, frame 3 further out and fading). Gold maps to the horseSkin1-3 tags (free on a foot unit); the patches that do
this live in make().
"""
import math

import numpy as np

import grid3d_mounted as M
from mounted_kits import pugilist as P
from mounted_kits.martial import head, kit, setup, uniform

U, A, B, Bl, Hd, R = M.U, M.A, M.B, M.Bl, M.Hd, M.R
E = B.E
m, V = U.m, U.V

KIT = 'fist_saint'
GOLD = '#d6a224'       # ramp l m n n -> horseSkin2 / horseSkin1 (mostly the bright yellow)
MANE = '#9a6a1c'       # darker gold mane (ramp k k l l -> horseSkin3 / horseSkin2): the dark edge between the lion and the head
MOUTH = '#2a1a16'      # lion jaws and eyes (ramp 0 0 1 1 -> outline / iron3)
SASH = '#29496e'       # long sash, full faction ramp W X Y Z (darker than the light faction top)
TAGS = dict(horseSkin1='#f7d65a', horseSkin2='#d9a62c', horseSkin3='#7a5418')  # grid.py colors after import

OFF = dict(hand=[.36, -.32, 1.25], elbow=[.6, -.06, 1.05])
# palm strike: chamber at the hip, gather, step-in thrust (orb leaves the palm), follow-through (orb flies on)
ATTACK = [
    dict(hand=[-.45, .15, 1.15], elbow=[-.62, .1, 1.08], lean=.08, bob=-.04, crouch=.6),
    dict(hand=[-.5, .32, 1.25], elbow=[-.66, .2, 1.15], lean=.22, bob=-.1, crouch=1),
    dict(hand=[-.2, -1.0, 1.36], elbow=[-.32, -.52, 1.38], lean=-.4, bob=-.07, lunge=1.2, orb=(.5, .42, 1)),
    dict(hand=[-.24, -.9, 1.34], elbow=[-.36, -.48, 1.36], lean=-.26, bob=-.05, lunge=1.0, orb=(.9, .48, .8)),
]


def ball(layer, fx, to_px):
    """Energy orb: three concentric discs (outer glow, body, bright core) in the effect colours."""
    c = np.array(to_px(fx['center']))
    r = np.linalg.norm(np.array(to_px(fx['center'] + V([fx['radius'], 0, 0]))) - c)
    r = max(r, np.linalg.norm(np.array(to_px(fx['center'] + V([0, 0, fx['radius']]))) - c))
    for scale, key, alpha in ((1, 'p', 120), (.7, 'q', 175), (.38, 'r', 235)):
        rr = r*scale

        def paint(dr, s, rr=rr):
            dr.ellipse([(c[0] - rr)*s, (c[1] - rr)*s, (c[0] + rr)*s, (c[1] + rr)*s], fill=255)
        layer.fill(layer.mask(paint), key, alpha*fx['alpha'])


def lions(T):
    for i in range(2):
        s = -1 if i == 0 else 1
        c = T([s*.47, -.02, 1.5])
        look = A.unit([s*.5, -.86, .05])
        up = V([0, 0, 1.])
        side = A.unit(np.cross(look, up))
        m.ell(c, [.26, .27, .22], MANE)
        for a in np.linspace(0, math.tau, 9)[:-1]:
            m.ell(c + look*.1 + (side*math.cos(a) + up*math.sin(a))*.23, [.08]*3, MANE)
        face = c + look*.19
        m.ell(face, [.23, .23, .2], GOLD)
        m.ell(face + look*.13 + V([0, 0, -.05]), [.17, .12, .11], MOUTH)   # open jaws
        m.ell(face + look*.16 + V([0, 0, -.15]), [.12, .08, .04], GOLD)    # lower jaw
        for e in (-1, 1):
            m.ell(face + look*.14 + side*e*.1 + V([0, 0, .1]), [.065]*3, MOUTH)  # eyes
        m.ell(face + V([0, 0, .16]), [.12, .12, .05], GOLD)                # brow


def chest_plate(T):
    c = T([0, -.3, 1.22])
    m.box(c, [.36, .06, .2], U.STEEL)
    for z in (-.11, .11):
        m.box(c + V([0, -.01, z]), [.4, .07, .04], GOLD)
    for x in (-.19, .19):
        m.box(c + V([x, -.01, 0]), [.04, .07, .24], GOLD)


def sash(T):
    m.box(T([0, 0, 1.01]), [.82, .59, .12], SASH)
    m.ell(T([0, -.3, 1.0]), [.1, .06, .08], SASH)
    m.box(T([-.07, -.32, .76]), [.12, .04, .4], SASH)   # two ends hanging in front (mid-thigh)
    m.box(T([.08, -.31, .8]), [.11, .04, .32], SASH)
    m.box(T([-.07, -.33, .57]), [.13, .045, .05], GOLD)
    m.box(T([.08, -.32, .65]), [.12, .045, .05], GOLD)


def decorate(p, palm=False):
    T = U.transform(p)
    uniform(p, T)
    for i in range(2):
        s = -1 if i == 0 else 1
        sh, el, ha = T([s*.4, 0, 1.42]), T(p['elbows'][i]), T(p['hands'][i])
        m.ell(sh + V([s*.03, 0, -.02]), [.21, .22, .2], U.SKIN)
        m.rod(sh + (el - sh)*.25, sh + (el - sh)*.75, .175, U.SKIN)
        fore = A.unit(ha - el)
        m.rod(el + (ha - el)*.3, ha, .2, P.GAUNTLET)
        if palm and i == 0:  # open palm, fingers up, facing the strike
            m.ell(ha + fore*.08, [.18, .08, .22], P.GAUNTLET)
            m.ell(ha + fore*.13 + V([0, 0, .14]), [.12, .05, .1], P.GAUNTLET)
        else:
            m.ell(ha + fore*.04, [.21, .21, .2], P.GAUNTLET)
            m.ell(ha + fore*.16, [.13, .13, .12], P.SPIKE)
        m.rod(el + (ha - el)*.28, el + (ha - el)*.35, .215, GOLD)  # gold cuff band
    lions(T)
    chest_plate(T)
    sash(T)


def body(p, row, f, effects=(), palm=False):
    _, hd = U.soldier(p, KIT, weapon=False)
    decorate(p, palm)
    return U.result(m.meshes, hd, row, f, effects)


def blast(p, k):
    T = U.transform(p)
    hand, elbow = T(p['hands'][0]), T(p['elbows'][0])
    d = A.unit(hand - elbow)
    dist, radius, alpha = k['orb']
    c = hand + d*dist
    t = U.VIEW['t']
    along = float(np.dot(d, t)) if t is not None else 0
    layer = 'under' if along < -.5 else 'over'  # away from the camera: behind the body
    if abs(along) > .8:  # toward or away from the camera the orb sits over the body (effects keep only empty cells)
        radius *= 1.5
    out = [dict(kind='ball', center=c, radius=radius, alpha=alpha, layer=layer)]
    side = A.unit(np.cross(d, [0, 0, 1.])) if abs(d[2]) < .9 else V([1., 0, 0])
    up = np.cross(side, d)
    out.append(dict(kind='lines', alpha=alpha*.9, layer='under',
                    segments=[(c - d*(radius + .5) + o, c - d*radius + o) for o in (side*.18, -side*.18, up*.2)]))
    if alpha < 1:
        out.append(dict(kind='ring', center=c + d*.1, axis=d, radius=radius + .14, alpha=alpha, layer=layer))
    return out


def fighter(row, f):
    if row == U.ATTACK:
        k = ATTACK[f]
        p = P.posed(0, 0, dict(hands=[k['hand'], OFF['hand']], elbows=[k['elbow'], OFF['elbow']]), k)
        p['row'] = U.ATTACK
        return body(p, row, f, blast(p, k) if 'orb' in k else (), palm=f >= 2)
    if row == U.HIT:
        return body(U.hit_pose(f, P.posed(0, 0, P.GUARD)), row, f)
    if row == 4:
        return body(U.pose(4, f), row, f)
    p = P.posed(row, f, P.REST)
    if row == 1:
        sw = [-.12, 0, .12, 0][f]
        p['hands'][0][1] += sw
        p['hands'][1][1] -= sw
    if row == 0:
        for h in p['hands']:
            h[2] += [0, .02, 0, -.02][f]
    return body(p, row, f)


def block(f):
    k = P.BLOCK[f]
    return Bl.displace(body(P.posed(0, 0, k, k), 0, f), 0, k.get('back', 0))


def make():
    U.KITS[KIT] = kit(guard=U.HEMP_DARK)
    setup()
    U.WEAPON_COLORS.update({P.GAUNTLET, P.SPIKE})
    R.add_ramps({P.GAUNTLET: 'WXYZ', GOLD: 'lmnn', MANE: 'kkll', MOUTH: '0011', SASH: 'WXYZ'})
    M.KEY2TAG.update(k='x', l='w', m='v', n='v')  # gold -> horseSkin3 / horseSkin2 / horseSkin1
    E.DRAW['ball'] = ball

    def build(row, f, style=None):
        return block(f) if row == B.BLOCK else fighter(row, f)
    return build, head()


M.register(KIT, make)
