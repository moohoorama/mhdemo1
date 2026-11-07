"""세객 and 종횡가: the envoys' line. Both carry a 절(節), the envoy's staff of office: a long bamboo rod held upright
with tufts of dyed yak tail tied on in tiers below a gold knob, instead of the 참모's fan or the 군사's ruyi.

세객 (orator): the 참모 body (ankle-length faction robe, sash) and the 서생's topknot head, a 절 of three tufts.
종횡가 (diplomat): the 군사 body with its white-edged cape, a black official cap (looks.CAP) instead of the 군사's
진현관, and a taller 절 of four tufts under a larger gold finial.

The staff stays upright (tipped a little away from the body) in idle, walk and hit, wherever the pose points the
weapon; seated it leans on the shoulder. Attack (persuasion): staff raised before the face (0), lifted high (1),
swung forward at the enemy with rings of voice spreading from its head (2), held while they fade (3). Raised (0, 1)
and guarding, the tufts use TUFT_BACK. The staff is never drawn over the stamped head (no weapon colours), so
it never covers the face. Block: blocks.SWORD 'guard' poses with the staff
held across.

Tags: tufts (grid3d.BUCKLER keys A B C) -> themeB, rod (leather) -> themeA, knob and bands (gold) -> themeC,
hair 7 8 9 -> hair1-3, robe, cape -> faction.
"""
import numpy as np

import grid3d_mounted as M
from mounted_kits import grand_strategist as GS, scholar as SC

G, U, A, B, Bl, Hd, R = M.G, M.U, M.A, M.B, M.Bl, M.Hd, M.R
m, V = U.m, U.V
looks = B.looks

TUFT = G.BUCKLER           # ramp ABCC
TUFT_BACK = '#d6bb7c'      # the same tufts, not a weapon colour: stays behind the stamped head (raised, guarding)
UP = V([0, 0, 1])
OUT = V([-.1, -.2, 0])     # tips the upright staff forward and a little away from the body
SHOULDER = V([-.4, 0, 1.42])

ATTACK = [dict(hand=[-.42, -.36, 1.2], dir=[-.08, -.12, 1], lean=.02),
          dict(hand=[-.55, -.18, 1.5], dir=[-.15, .12, 1], lean=-.03, crouch=.2, bob=-.02),
          dict(hand=[-.32, -.7, 1.25], dir=[0, -.75, .66], lean=.16, lunge=.45, bob=-.04),
          dict(hand=[-.34, -.64, 1.2], dir=[0, -.7, .7], lean=.1, lunge=.3, bob=-.02)]


def staff(tufts, length, knob):
    """prop(hand, d, colour) drawing a 절 along d from the hand; returns the knob. The tufts hang from the upper
    part of the rod with a gap of bare rod between them, each longer than wide."""
    def prop(hand, d, colour):
        top = hand + d*length
        m.rod(hand - d*.55, top, .032, U.BROWN)
        for i in range(tufts):
            c = top - d*(.16 + .24*i)
            m.ell(c, [.06]*3, U.GOLD)                              # the band the tuft hangs from
            m.ell(c - d*.1, [.085, .085, .11], colour)
        m.ell(top + d*.04, [knob]*3, U.GOLD)
        return top
    return prop


def build(dress, prop, cap=False):
    def holding(p, block=False):
        head = dress(p)
        T = U.transform(p)
        hand = T(p['hands'][0])
        if block:
            prop(hand, A.unit(V(p['blade'])), TUFT_BACK)
        elif p.get('seated'):
            prop(hand, A.unit(V([-.25, .35, 1])), TUFT)
        else:
            prop(hand, A.unit(UP + OUT), TUFT)
        return head

    def casting(f):
        k = ATTACK[f]
        p = U.pose(0, 0)
        p['row'] = U.ATTACK
        p['lean'], p['bob'] = k['lean'], k.get('bob', 0)
        A.stance(p, k)
        p['hands'][0] = list(k['hand'])
        p['elbows'][0] = ((SHOULDER + V(k['hand']))/2 + [-.16, .06, -.08]).tolist()
        head = dress(p, swing=.08) if cap and f >= 2 else dress(p)
        T = U.transform(p)
        d = A.unit(V(k['dir']))
        tip = prop(T(k['hand']), d, TUFT_BACK if f < 2 else TUFT)
        effects = []
        if f >= 2:  # rings of voice round the staff head, turned to the camera so no view sees them edge-on
            t = U.VIEW['t']
            axis = A.unit(V(t)) if t is not None else A.unit(V([0, -1, 0]))
            for r in ((.28, .5) if f == 2 else (.72,)):
                effects.append(dict(kind='ring', center=tip, axis=axis, radius=r, alpha=1 if f == 2 else .5,
                                    layer='over'))
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


def setup():
    R.add_ramps({TUFT: 'ABCC', TUFT_BACK: 'ABCC'})
    M.KEY2TAG.update({'7': 'g', '8': 'h', '9': 'i'})


def make_orator():
    U.KITS['orator'] = dict(GS.KIT)
    setup()
    return build(lambda p, swing=0.: U.soldier(p, 'orator', weapon=False)[1], staff(3, 1.25, .07)), SC.HEADS


def make_diplomat():
    GS.make()
    U.WEAPON_COLORS.discard(U.GOLD)  # the 군사's ruyi colour: here the staff's knob, kept behind the head
    setup()
    dress = lambda p, swing=0.: GS.dressed(p, swing)[1]
    return build(dress, staff(4, 1.4, .09), cap=True), Hd.heads(looks.CAP)


M.register('orator', make_orator)
M.register('diplomat', make_diplomat)
