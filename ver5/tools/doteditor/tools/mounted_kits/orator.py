"""세객 and 종횡가: the envoys' line, who sway the enemy with words. Both hold a feather fan like the 참모 and are told
apart from the other civil classes by three things that read at 1:1:

  head  — 세객: a white cloth 복건 (幅巾) wrapped over the topknot with its loose ends hanging behind; 종횡가: the
          black official cap (looks.CAP) with a gold pin on its crown.
  hand  — a letter (書信) in the empty left hand: a roll of yellowed silk tied with a red cord and sealed with clay,
          held upright before the chest in every pose (종횡가: the seal in gold).
  attack — the fan swept forward at the enemy sends out rings of voice, turned to the camera so no view sees them
          edge-on: two then one wide ring (세객), three then one (종횡가), where the 참모's fan throws a burst.

세객 (orator): the 참모 body (ankle-length faction robe, sash). 종횡가 (diplomat): the 군사 body with its white-edged
cape and a larger fan.

Fan, attack poses and block as staff_officer (raised and guarding, the feathers use FAN_BACK so they never cover
the face).

Tags: fan (grid3d.BUCKLER keys A B C) -> themeB (grey), handle (leather) -> themeA, collar and gold seal ->
themeC, letter (horse coat key) -> horseSkin (yellowed silk), cord and red seal (saddle red key) -> saddle, 복건 cloth 5 4 3 -> iron1/iron2, hair 7 8 9 -> hair1-3, robe, cape ->
faction.
"""
import numpy as np

import grid3d_mounted as M
from mounted_kits import grand_strategist as GS, staff_officer as SO

G, U, A, B, Bl, Hd, R = M.G, M.U, M.A, M.B, M.Bl, M.Hd, M.R
m, V = U.m, U.V
looks = B.looks

FAN = G.BUCKLER            # ramp ABCC
FAN_BACK = SO.FAN_BACK
CORD = '#a91e2c'           # the saddle key (ramp uuvw -> saddle): nothing else on a foot officer uses it
OUT = V([-.35, 0, 0])      # tips the held fan away from the body
SILK = M.COAT[0]           # a horse-coat key (ramp nopp -> horseSkin): the letter gets its own tag, apart from the
                           # 복건 and the cape trim (iron1)


# ---- heads ---------------------------------------------------------------------------------
def bokgeon(rows):
    """The topknot head with the bun and its band wrapped in white cloth (light 5, shade 4, fold 3), the band's loose
    tails kept as the cloth's ends hanging behind; the hair below the band stays hair (7/8/9)."""
    band = [y for y, r in enumerate(rows) if any(c in 'mlk' for c in r)]
    last = max(band)
    cloth = {'f': '4', 'g': '5', 'm': '5', 'l': '4', 'k': '3'}
    hair = {'f': '8', 'g': '7'}
    return [''.join((cloth if y <= last else hair).get(c, c) for c in r) for y, r in enumerate(rows)]


def pinned(rows):
    """The official cap with a gold pin (gold keys m over l -> themeC) in the middle of its crown."""
    rows = [list(r) for r in rows]
    xs = [x for x, c in enumerate(rows[1]) if c not in '.0']
    if xs:
        mid = xs[len(xs)//2]
        rows[1][mid], rows[2][mid] = 'm', 'l'
    return [''.join(r) for r in rows]


ORATOR_HEADS = Hd.heads({d: bokgeon(r) for d, r in Hd.TOPKNOT.items()})
DIPLOMAT_HEADS = Hd.heads({d: pinned(r) for d, r in looks.CAP.items()})


# ---- props ---------------------------------------------------------------------------------
def fan(size):
    """staff_officer.fan_at drawn size times as large; returns the tip."""
    def prop(hand, d, feathers):
        side = U.facing(d)
        m.rod(hand - d*.08*size, hand + d*.16*size, .045, U.BROWN)
        base = hand + d*.17*size
        m.ell(base, [.065*size]*3, U.GOLD)
        c = base + d*.3*size
        m.poly([c + d*np.cos(a)*.32*size + side*np.sin(a)*.21*size for a in np.linspace(0, 2*np.pi, 12, endpoint=False)],
               feathers)
        return base + d*.55*size
    return prop


def letter(T, hand, wax):
    """The letter in the left hand: a short thick roll of yellowed silk standing upright, a red cord round its middle and
    a seal of wax (red, or gold) on the cord, turned outward."""
    c = V(hand) + V([.03, -.06, .1])
    m.rod(T(c - V([0, 0, .2])), T(c + V([0, 0, .26])), .095, SILK)
    m.rod(T(c + V([0, 0, .01])), T(c + V([0, 0, .06])), .103, CORD)
    m.ell(T(c + V([.09, -.05, .035])), [.06]*3, wax)


# ---- unit ----------------------------------------------------------------------------------
def build(dress, prop, wax, rings):
    def holding(p, block=False):
        head = dress(p)
        T = U.transform(p)
        letter(T, p['hands'][1], wax)
        hand = T(p['hands'][0])
        if block:
            prop(hand, A.unit(V(p['blade'])), FAN_BACK)
        elif p.get('seated'):
            prop(hand, A.unit(V([-.3, -.55, .6])), FAN)
        else:
            prop(hand, A.unit(V(p['blade']) + OUT), FAN)
        return head

    def casting(f):
        k = SO.ATTACK[f]
        p = U.pose(0, 0)
        p['row'] = U.ATTACK
        p['lean'], p['bob'] = k['lean'], k.get('bob', 0)
        A.stance(p, k)
        p['hands'][0] = list(k['hand'])
        p['elbows'][0] = ((GS.SHOULDER + V(k['hand']))/2 + [-.16, .06, -.08]).tolist()
        head = dress(p)
        T = U.transform(p)
        letter(T, p['hands'][1], wax)
        d = A.unit(V(k['dir']))
        tip = prop(T(k['hand']), d, FAN_BACK if f < 2 else FAN)
        effects = []
        if f >= 2:
            t = U.VIEW['t']
            axis = A.unit(V(t)) if t is not None else A.unit(V([0, -1, 0]))
            for r in (rings if f == 2 else rings[-1:]):
                effects.append(dict(kind='ring', center=tip + d*.2, axis=axis, radius=r if f == 2 else r*1.25,
                                    alpha=1 if f == 2 else .5, layer='over'))
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
    R.add_ramps({FAN: 'ABCC', FAN_BACK: 'ABCC'})
    M.KEY2TAG.update({'7': 'g', '8': 'h', '9': 'i'})
    U.WEAPON_COLORS.add(FAN)  # the fan is drawn over the stamped head where it passes in front


def make_orator():
    U.KITS['orator'] = dict(GS.KIT)
    setup()
    return build(lambda p: U.soldier(p, 'orator', weapon=False)[1], fan(1.0), CORD, (.28, .5)), ORATOR_HEADS


def make_diplomat():
    GS.make()
    setup()
    return build(lambda p: GS.dressed(p)[1], fan(1.25), U.GOLD, (.24, .42, .6)), DIPLOMAT_HEADS


M.register('orator', make_orator)
M.register('diplomat', make_diplomat)
