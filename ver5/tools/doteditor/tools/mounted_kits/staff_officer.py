"""참모: the 군사 body (units.soldier with a long robe) without the cape, a feather fan (羽扇) in the right hand.

Fan: a short handle, a gold collar and an oval leaf of feathers laid across the view (units.facing) so it never
renders edge-on. Attack (stratagem): fan raised before the face (0), lifted high (1), swept forward at the enemy
with a burst at its tip (2), held (3); in N and S a shock ring around the fan instead (as the 서생's). Raised (0, 1)
and guarding, the feathers use FAN_BACK so they never cover the face. Block: blocks.SWORD 'guard' poses with the fan
held across at the chest.

build(dress, prop, colour, back) makes the unit for any body (dress) and anything held like the fan (prop, e.g. a
sword), so a hero (mounted_kits/story_cast) can take it in its own clothes and with its own weapon. Class sheet tags: robe, sash -> faction, fan (grid3d.BUCKLER keys A B C) -> themeB,
handle (leather) -> themeA, collar (gold) -> themeC, hair 7 8 9 -> hair1-3.
"""
import numpy as np

import grid3d_mounted as M
from mounted_kits import grand_strategist as GS, scholar as SC

G, U, A, B, Bl, R = M.G, M.U, M.A, M.B, M.Bl, M.R
m, V = U.m, U.V

FAN = G.BUCKLER            # ramp ABCC
UP = V([0, 0, 1])
OUT = V([-.35, 0, 0])      # tips the held fan away from the body
KIT = dict(GS.KIT)

# attack: hand, fan direction, body
ATTACK = [dict(hand=[-.45, -.42, 1.4], dir=[-.1, -.3, 1], lean=.02),
          dict(hand=[-.6, -.2, 1.7], dir=[-.2, .15, 1], lean=-.03, crouch=.2, bob=-.02),
          dict(hand=[-.3, -.86, 1.36], dir=[0, -1, .15], lean=.18, lunge=.5, bob=-.04),
          dict(hand=[-.32, -.76, 1.28], dir=[0, -1, .05], lean=.12, lunge=.35, bob=-.02)]


def fan_at(hand, d, feathers, collar=U.GOLD):
    """Handle along d from the hand, a gold collar, then seven feathers spread over a half-round across the view."""
    side = U.facing(d)
    m.rod(hand - d*.08, hand + d*.16, .045, U.BROWN)
    base = hand + d*.17
    m.ell(base, [.065]*3, collar)
    c = base + d*.3
    m.poly([c + d*np.cos(a)*.32 + side*np.sin(a)*.21 for a in np.linspace(0, 2*np.pi, 12, endpoint=False)], feathers)
    return base + d*.55


def robed(kit):
    """dress(p) for a units.KITS robe kit: draws the body, returns the head centre."""
    return lambda p: U.soldier(p, kit, weapon=False)[1]


def build(dress, prop, colour, back):
    """build(row, f, style=None) for the 참모 drawn by dress(p) (returns the head centre) holding prop(hand, d, colour)
    (returns its tip) in colour, back while raised or guarding."""
    def holding(p, block=False):
        head = dress(p)
        T = U.transform(p)
        if block:
            prop(T(p['hands'][0]), A.unit(V(p['blade'])), back)
        elif p.get('seated'):
            prop(T(p['hands'][0]), A.unit(V([-.3, -.55, .6])), colour)
        else:
            prop(T(p['hands'][0]), A.unit(V(p['blade']) + OUT), colour)
        return head

    def casting(f):
        k = ATTACK[f]
        p = U.pose(0, 0)
        p['row'] = U.ATTACK
        p['lean'], p['bob'] = k['lean'], k.get('bob', 0)
        A.stance(p, k)
        p['hands'][0] = list(k['hand'])
        p['elbows'][0] = ((GS.SHOULDER + V(k['hand']))/2 + [-.16, .06, -.08]).tolist()
        head = dress(p)
        T = U.transform(p)
        d = A.unit(V(k['dir']))
        tip = prop(T(k['hand']), d, back if f < 2 else colour)
        effects = []
        if f >= 2:
            t = U.VIEW['t']
            if t is not None and abs(t[1]) > .9*np.hypot(t[0], t[1]):
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
        return U.result(m.meshes, head, U.ATTACK, f, effects)

    def unit(row, f, style=None):
        if row == B.BLOCK:
            k = Bl.SWORD[Bl.SELECTED]['frames'][f]
            head = holding(Bl.foot_pose(dict(k, hand=list(V(k['hand']) + [0, -.08, -.24]))), block=True)
            return Bl.displace(U.result(m.meshes, head, 0, f), k.get('sway', 0), k.get('back', 0))
        if row == U.ATTACK:
            return casting(f)
        p = U.hit_pose(f, U.pose(0, 0)) if row == U.HIT else U.pose(row, f)
        head = holding(p)  # before reading m.meshes: units.soldier rebinds the list
        return U.result(m.meshes, head, row, f)
    return unit


FAN_BACK = '#d7bc7d'  # the same feathers, not a weapon colour: stays behind the stamped head (raised, guarding)


def make():
    U.KITS['staff_officer'] = KIT
    R.add_ramps({FAN: 'ABCC', FAN_BACK: 'ABCC'})
    M.KEY2TAG.update({'7': 'g', '8': 'h', '9': 'i'})
    U.WEAPON_COLORS.add(FAN)
    return build(robed('staff_officer'), fan_at, FAN, FAN_BACK), SC.HEADS  # no 참모 cap yet: the 서생's topknot


M.register('staff_officer', make)
