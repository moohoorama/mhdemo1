"""무도가: a young bare-handed martial artist (profile/brawler.txt). Same body family as 권사 (mounted_kits/pugilist.py):
bald head, sturdy build with thick bare arms, but no steel anywhere.

Clothes: the martial uniform of mounted_kits/martial.py (a sleeveless faction-colour top closed across the chest so the
skin shows in a V, a dark belt with a short knot at the side, hemp trousers), brown cloth shin wraps and sandals.
Faction-colour cloth wraps over the fists (back of the hand to the wrist), where the 권사 wears gauntlets.

Poses: a martial guard with both fists up at chest height and the knees a little bent (idle, walk), one step-in
straight punch with an impact burst (attack), both forearms raised before the face (block); hit and exhausted are the
standard poses. Everything that changes shared state happens in make().
"""
import numpy as np

import grid3d_mounted as M
from mounted_kits.martial import head, kit, setup, uniform

G, U, A, B, Bl, Hd, R = M.G, M.U, M.A, M.B, M.Bl, M.Hd, M.R
m, V = U.m, U.V

KIT = 'brawler'
ARM = '#efb15d'    # skin of the forearms: its own colour so raised forearms are drawn over the stamped head
WRAP = '#3a76a2'   # faction-colour cloth hand wraps (ramp Y Z Z Z -> faction2 / faction1)

# fists up at chest height, lead (right, x < 0) fist forward
GUARD = dict(hands=[[-.33, -.5, 1.36], [.37, -.36, 1.26]], elbows=[[-.56, -.16, 1.08], [.58, -.06, 1.02]])
STANCE = dict(crouch=.45, bob=-.04)
# step-in straight punch: chamber, gather, strike, follow-through
ATTACK = [
    dict(hand=[-.42, .1, 1.18], elbow=[-.62, .12, 1.08], lean=.06, bob=-.05, crouch=.6),
    dict(hand=[-.46, .3, 1.15], elbow=[-.64, .22, 1.14], lean=.22, bob=-.1, crouch=1),
    dict(hand=[-.22, -1.08, 1.38], elbow=[-.34, -.56, 1.4], lean=-.42, bob=-.07, lunge=1.2, hit=1.0),
    dict(hand=[-.28, -.84, 1.34], elbow=[-.4, -.46, 1.34], lean=-.24, bob=-.05, lunge=1.0, hit=.5),
]
OFF_GUARD = dict(hand=[.3, -.3, 1.3], elbow=[.58, -.04, 1.04])
# both forearms upright before the face: raise, brace, the blow (pushed back), recover
BLOCK = [
    dict(hands=[[-.2, -.5, 1.62], [.2, -.48, 1.58]], elbows=[[-.42, -.36, 1.2], [.42, -.34, 1.18]], lean=.04, crouch=.4),
    dict(hands=[[-.15, -.52, 1.9], [.17, -.5, 1.86]], elbows=[[-.27, -.44, 1.36], [.29, -.42, 1.34]], lean=.08, bob=-.04,
         crouch=.6),
    dict(hands=[[-.15, -.46, 1.86], [.17, -.44, 1.82]], elbows=[[-.27, -.38, 1.32], [.29, -.36, 1.3]], lean=.22, bob=-.08,
         crouch=.9, back=.12),
    dict(hands=[[-.2, -.5, 1.66], [.2, -.48, 1.6]], elbows=[[-.42, -.36, 1.22], [.42, -.34, 1.2]], lean=.08, crouch=.5),
]


def decorate(p):
    T = U.transform(p)
    uniform(p, T)
    for i in range(2):
        s = -1 if i == 0 else 1
        sh, el, ha = T([s*.4, 0, 1.42]), T(p['elbows'][i]), T(p['hands'][i])
        m.ell(sh + V([s*.03, 0, -.02]), [.21, .22, .2], U.SKIN)              # deltoid
        m.rod(sh + (el - sh)*.25, sh + (el - sh)*.75, .175, U.SKIN)          # biceps / triceps
        m.rod(el, el + (ha - el)*.6, .15, ARM)                               # forearm
        fore = A.unit(ha - el)
        m.rod(el + (ha - el)*.62, ha, .15, WRAP)                             # wrapped wrist
        m.ell(ha + fore*.03, [.16, .16, .15], WRAP)                          # wrapped fist


def posed(row, f, hands, elbows, k):
    p = U.pose(row, f)
    p['hands'], p['elbows'] = [list(h) for h in hands], [list(e) for e in elbows]
    p['lean'] = k.get('lean', 0)
    p['bob'] = p['bob'] + k.get('bob', 0)
    A.stance(p, k)
    return p


def body(p, row, f, effects=()):
    _, head = U.soldier(p, KIT, weapon=False)
    decorate(p)
    return U.result(m.meshes, head, row, f, effects)


def punch(p, k):
    T = U.transform(p)
    hand, elbow = T(p['hands'][0]), T(p['elbows'][0])
    d = A.unit(hand - elbow)
    out = A.pierce(hand + d*.2, d, hand, alpha=k['hit'], burst=k['hit'] == 1, reach=.45)
    t = U.VIEW['t']
    along = float(np.dot(d, t)) if t is not None else 0
    # the converter keeps effect pixels only on empty cells: toward or away from the camera the fist sits over the body,
    # so the ring is widened to show around it; away from the camera it stays behind the head
    wide = abs(along) > .8
    for e in out:
        if e['kind'] == 'cone':
            e.update(length=.7, width=.9 if wide else .5)
        elif e['kind'] == 'ring':
            e.update(center=e['center'] - e['axis']*.1, radius=(.62 if wide else .26) * (1 if k["hit"] == 1 else .9))
        if along < -.5:
            e['layer'] = 'under'
    return out


def fighter(row, f):
    if row == U.ATTACK:
        k = ATTACK[f]
        p = posed(0, 0, [k['hand'], OFF_GUARD['hand']], [k['elbow'], OFF_GUARD['elbow']], k)
        p['row'] = U.ATTACK
        return body(p, row, f, punch(p, k) if 'hit' in k else ())
    if row == U.HIT:
        return body(U.hit_pose(f, posed(0, 0, GUARD['hands'], GUARD['elbows'], STANCE)), row, f)
    if row == 4:
        return body(U.pose(4, f), row, f)
    p = posed(row, f, GUARD['hands'], GUARD['elbows'], STANCE if row == 0 else dict(bob=-.02))
    if row == 0:  # breathing: the fists rise with the chest
        for h in p['hands']:
            h[2] += [0, .02, 0, -.02][f]
    return body(p, row, f)


def block(f):
    k = BLOCK[f]
    return Bl.displace(body(posed(0, 0, k['hands'], k['elbows'], k), 0, f), 0, k.get('back', 0))


def make():
    U.KITS[KIT] = kit(guard=U.BROWN)  # guard: the brown shin wraps
    setup()
    U.WEAPON_COLORS.update({ARM, WRAP})  # raised forearms (block) are drawn over the head
    R.add_ramps({ARM: 'xijj', WRAP: 'YZZZ'})

    def build(row, f, style=None):
        return block(f) if row == B.BLOCK else fighter(row, f)
    return build, head()


M.register(KIT, make)
