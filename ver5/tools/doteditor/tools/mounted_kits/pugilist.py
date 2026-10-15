"""권사: the 무도가 body (bald, bare muscular arms, the martial uniform of mounted_kits/martial.py: faction-colour top
with a chest V, dark belt with a side knot, hemp trousers) with faction-colour gauntlets (steel knuckle plates) over both fists and forearms and
spiked steel pauldrons on both shoulders.

The base soldier is units.soldier with bare skin arms and no weapon; decorate() adds the uniform, the thicker arm
muscles, the gauntlets and the pauldrons. Skin gets a third, darker shade (key 'x' -> skin3) so the muscles read.

Poses: fists held low and a little forward in idle/walk (so the near gauntlet shows in W/E), a left-right punch
combo for the attack (frame 1 right jab, frame 2 left straight, each with a pierce burst at the fist), crossed
forearms in front of the face for the block. Everything that changes shared state happens in make().
"""
import math

import numpy as np

import grid3d_mounted as M
from mounted_kits.martial import head, kit, setup, uniform

G, U, A, B, Bl, Hd, R = M.G, M.U, M.A, M.B, M.Bl, M.Hd, M.R
m, V = U.m, U.V

KIT = 'pugilist'
GAUNTLET = '#29496f'   # faction-colour glove and forearm cover; own colour so it is drawn over the head (block)
SPIKE = '#bacbd6'      # light steel (ramp 4556): spikes and knuckle plates catch the light

SHOULDERS = (V([-.4, 0, 1.42]), V([.4, 0, 1.42]))
# fists low at the sides, a little forward (hand 0 = right / weapon side, x < 0)
REST = dict(hands=[[-.6, -.24, .98], [.6, -.24, .98]], elbows=[[-.64, -.02, 1.17], [.64, -.02, 1.17]])
GUARD = dict(hands=[[-.38, -.42, 1.32], [.36, -.36, 1.22]], elbows=[[-.6, -.1, 1.08], [.6, -.08, 1.04]])
# left-right combo: chamber, right jab, left straight, back to guard
ATTACK = [
    dict(hands=[[-.42, -.3, 1.3], [.4, -.25, 1.22]], elbows=[[-.62, .05, 1.1], [.62, .05, 1.05]], lean=.12, bob=-.05, crouch=.6),
    dict(hands=[[-.22, -1.0, 1.2], [.42, -.22, 1.22]], elbows=[[-.36, -.52, 1.28], [.62, .05, 1.05]], lean=-.3, bob=-.05,
         lunge=.8, hit=0),
    dict(hands=[[-.45, -.2, 1.25], [.16, -1.08, 1.52]], elbows=[[-.64, .08, 1.08], [.3, -.58, 1.46]], lean=-.42, bob=-.07,
         lunge=1.1, hit=1),
    dict(hands=[[-.4, -.38, 1.3], [.38, -.5, 1.28]], elbows=[[-.6, -.06, 1.08], [.58, -.18, 1.08]], lean=-.15, bob=-.03,
         lunge=.6),
]
# crossed forearms before the face: ready, brace, the blow (pushed back), recover
BLOCK = [
    dict(hands=[[-.25, -.45, 1.5], [.25, -.42, 1.42]], elbows=[[-.55, -.25, 1.15], [.55, -.25, 1.1]], lean=.05),
    dict(hands=[[.08, -.5, 1.68], [-.06, -.46, 1.5]], elbows=[[-.42, -.4, 1.28], [.42, -.4, 1.2]], lean=.1, bob=-.04, crouch=.5),
    dict(hands=[[.1, -.42, 1.62], [-.04, -.4, 1.45]], elbows=[[-.4, -.32, 1.22], [.4, -.32, 1.15]], lean=.22, bob=-.08,
         crouch=.9, back=.12),
    dict(hands=[[-.2, -.45, 1.5], [.2, -.42, 1.4]], elbows=[[-.55, -.25, 1.15], [.55, -.25, 1.1]], lean=.08, crouch=.3),
]


def cone(base, tip, r, c, n=6):
    ax = A.unit(tip - base)
    u = np.cross(ax, [0, 0, 1.])
    if np.linalg.norm(u) < .01:
        u = np.cross(ax, [0, 1., 0])
    u /= np.linalg.norm(u)
    v = np.cross(ax, u)
    ring = [base + r*(u*math.cos(t) + v*math.sin(t)) for t in np.linspace(0, math.tau, n + 1)]
    for a, b in zip(ring, ring[1:]):
        m.poly([a, b, tip], c)


def decorate(p):
    T = U.transform(p)
    uniform(p, T)
    for i in range(2):
        s = -1 if i == 0 else 1
        sh, el, ha = T([s*.4, 0, 1.42]), T(p['elbows'][i]), T(p['hands'][i])
        m.ell(sh + V([s*.03, 0, -.02]), [.21, .22, .2], U.SKIN)              # deltoid
        m.rod(sh + (el - sh)*.25, sh + (el - sh)*.75, .175, U.SKIN)          # biceps / triceps
        fore = A.unit(ha - el)
        m.rod(el + (ha - el)*.3, ha, .2, GAUNTLET)                          # gauntlet over the forearm
        m.ell(ha + fore*.04, [.21, .21, .2], GAUNTLET)                       # gloved fist
        m.ell(ha + fore*.16, [.13, .13, .12], SPIKE)                         # bright knuckle plate
        m.rod(el + (ha - el)*.28, el + (ha - el)*.36, .215, U.BROWN)         # leather cuff band
        cap = T([s*.44, 0, 1.49])
        m.ell(cap, [.25, .27, .17], U.STEEL)                                 # pauldron
        for dx, dy, dz in [(.55, 0, .75), (.85, -.35, .2), (.85, .35, .2), (.2, -.8, .45), (.2, .8, .45)]:
            d = A.unit([s*dx, dy, dz])
            base = cap + d*V([.2, .22, .14])
            cone(base, base + d*.34, .07, SPIKE)


def posed(row, f, arms, k=None):
    p = U.pose(row, f)
    p['hands'], p['elbows'] = [list(h) for h in arms['hands']], [list(e) for e in arms['elbows']]
    if k:
        p['lean'], p['bob'] = k.get('lean', 0), k.get('bob', 0)
        A.stance(p, k)
    return p


def body(p, row, f, effects=()):
    _, head = U.soldier(p, KIT, weapon=False)
    decorate(p)
    return U.result(m.meshes, head, row, f, effects)


def punch(k, p):
    i = k['hit']
    hand, elbow = U.transform(p)(p['hands'][i]), U.transform(p)(p['elbows'][i])
    d = A.unit(hand - elbow)
    out = A.pierce(hand + d*.2, d, hand, burst=True, reach=.45)
    t = U.VIEW['t']
    along = float(np.dot(d, t)) if t is not None else 0
    # the converter keeps effect pixels only on empty cells: punching toward or away from the camera the fist sits over
    # the body, so the ring is widened to show around it; away from the camera it stays behind the head
    wide = abs(along) > .8
    for e in out:
        if e['kind'] == 'cone':
            e.update(length=.7, width=.9 if wide else .5)
        elif e['kind'] == 'ring':
            e.update(center=e['center'] - e['axis']*.1, radius=.62 if wide else .26)
        if along < -.5:
            e['layer'] = 'under'
    return out


def fighter(row, f):
    if row == U.ATTACK:
        k = ATTACK[f]
        p = posed(0, 0, k, k)
        p['row'] = U.ATTACK
        return body(p, row, f, punch(k, p) if 'hit' in k else ())
    if row == U.HIT:
        return body(U.hit_pose(f, posed(0, 0, GUARD)), row, f)
    if row == 4:  # exhausted: the 적병's seated pose, fists on the knees
        return body(U.pose(4, f), row, f)
    p = posed(row, f, REST)
    if row == 1:  # walk: the fists swing against the stride
        sw = [-.12, 0, .12, 0][f]
        p['hands'][0][1] += sw
        p['hands'][1][1] -= sw
    if row == 0:  # breathing: the fists rise with the chest
        for h in p['hands']:
            h[2] += [0, .02, 0, -.02][f]
    return body(p, row, f)


def block(f):
    k = BLOCK[f]
    return Bl.displace(body(posed(0, 0, k, k), 0, f), 0, k.get('back', 0))


def make():
    U.KITS[KIT] = kit(guard=U.HEMP_DARK)
    setup()
    U.WEAPON_COLORS.update({GAUNTLET, SPIKE})  # gauntlets cross the face in the block: draw them over the head
    R.add_ramps({GAUNTLET: 'WXYZ'})

    def build(row, f, style=None):
        return block(f) if row == B.BLOCK else fighter(row, f)
    return build, head()


M.register(KIT, make)
