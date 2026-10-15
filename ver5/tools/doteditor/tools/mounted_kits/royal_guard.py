"""친위대: the 중기병 one step further, the way 근위병 goes beyond 중보병: fuller steel (cuirass with a faction breast
badge, steel tassets, big faction pauldrons with a steel cap, gold rim and a spike), gold trim (gorget, belt buckle,
helm crest, tassets, cloak edge), a long faction cloak that drapes over the horse's rump, and heavier barding (steel
chamfron, crinet, breast plate, flank plates, a gold-hemmed crupper plate). Head: iron helmet with plume and a taller
faction crest above it."""
import numpy as np

import grid3d_mounted as M

U, A, m = M.U, M.A, M.G.m
V = m.V
STEEL, BLUE, LIGHT_BLUE, LEATHER, GOLD = U.STEEL, U.BLUE, U.LIGHT_BLUE, U.BROWN, U.GOLD
KIT = 'rider_royal'


def cloak(p, T):
    """Faction cloak from the shoulders over the saddle and the rump (soldier space; the path stays outside the horse)."""
    path = [([0, .3, 1.55], .46), ([0, .38, 1.35], .5), ([0, .5, .9], .56), ([0, .72, .5], .62), ([0, 1.0, .15], .68),
            ([0, 1.12, -.2], .7)]

    def hang(q):
        """Shoulders follow the rider (bob, lean); the lower cloak rests on the rump and stays where the horse is."""
        t = min(1., max(0., (q[2] - .5)/1.0))
        return T(q)*t + V(q)*(1 - t)

    left = [hang([-w, *q[1:]]) for q, w in path]
    right = [hang([w, *q[1:]]) for q, w in path]
    U.strip(left, right, BLUE)
    for side in (left, right):  # gold edge along both sides
        for i in range(len(side) - 1):
            m.rod(side[i], side[i + 1], .035, GOLD)
    m.rod(left[-1], right[-1], .05, GOLD)  # gold hem
    hem_l, hem_r = [hang([-w*.97, y, z + .12]) for (_, y, z), w in path[-1:]], [hang([w*.97, y, z + .12]) for (_, y, z), w in path[-1:]]
    U.strip([left[-2], hem_l[0]], [right[-2], hem_r[0]], LIGHT_BLUE)  # lighter lining band above the hem


def rider_armour(p):
    T = U.transform(p)
    m.ell(T([0, -.04, 1.12]), [.46, .33, .46], STEEL)  # fuller cuirass
    m.ell(T([0, -.02, 1.5]), [.31, .25, .13], STEEL)  # gorget
    m.ell(T([0, -.02, 1.45]), [.33, .27, .05], GOLD)  # gold ring at its foot
    m.ell(T([0, -.31, 1.2]), [.19, .05, .19], GOLD)  # breast badge: gold rim and a faction disc
    m.ell(T([0, -.345, 1.2]), [.13, .04, .13], LIGHT_BLUE)
    m.box(T([0, -.06, .78]), [.64, .52, .09], LEATHER)  # belt
    m.box(T([0, -.32, .79]), [.15, .05, .11], GOLD)  # gold buckle
    for x in (-.2, 0, .2):  # steel tassets with a gold edge
        m.box(T([x, -.31, .58]), [.17, .05, .38], STEEL)
        m.box(T([x, -.325, .4]), [.17, .055, .04], GOLD)
    for s in (-1, 1):
        sh = T([s*.42, 0, 1.42])
        m.ell(sh + V([s*.01, 0, -.02]), [.28, .3, .25], BLUE)  # big faction pauldron
        m.ell(sh + V([s*.01, 0, .08]), [.27, .28, .12], STEEL)  # steel cap
        m.ell(sh + V([s*.01, 0, .02]), [.3, .32, .04], GOLD)  # gold rim between them
        m.rod(sh + V([s*.02, 0, .15]), sh + V([s*.05, 0, .5]), .05, GOLD)  # spike
        m.ell(sh + V([s*.05, 0, .52]), [.07, .07, .09], GOLD)
        el, ha = T(p['elbows'][0 if s < 0 else 1]), T(p['hands'][0 if s < 0 else 1])
        m.rod(el*.45 + ha*.55, ha*.9 + el*.1, .16, STEEL)  # vambrace
    cloak(p, T)
    m.rod(T([0, .02, 2.52]), T([0, .08, 2.95]), .09, BLUE)  # taller faction crest above the 2D helmet's plume
    m.ell(T([0, .12, 3.02]), [.11, .15, .14], LIGHT_BLUE)
    m.ell(T([0, .02, 2.5]), [.11, .12, .06], GOLD)  # gold crest socket


def horse_armour(row, f):
    """Barding in horse space (same transform units.cavalry gives the horse: walk bounce, nodding head, lowered
    forehand when exhausted); the caller applies the attack's rear/shift."""
    bounce = [.08, 0, .14, .03][f] if row == 1 else 0
    nod = [0, -.04, -.08, -.04][f] if row == 0 else 0

    def H(q):
        v = V(q)
        v[2] += bounce
        if v[1] < -.6:
            v[2] += nod*(-v[1] - .6)/.7
        if row == 4 and v[1] < -.7:
            v[2] -= .25
            v[1] -= .08
        return v

    m.ell(H([0, -.56, 1.12]), [.47, .24, .43], STEEL)  # breast plate
    m.ell(H([0, -.6, 1.0]), [.4, .2, .05], GOLD)  # with a gold band below
    m.ell(H([0, -.5, 1.52]), [.14, .09, .12], BLUE)  # faction chest tassel
    m.rod(H([0, -.66, 1.34]), H([0, -.9, 1.8]), .3, STEEL)  # crinet
    m.ell(H([0, -1.12, 1.9]), [.17, .29, .17], STEEL)  # chamfron (face plate)
    m.ell(H([0, -1.02, 2.1]), [.06, .12, .06], GOLD)  # gold plate crest between the ears
    m.rod(H([0, -.95, 2.14]), H([0, -.9, 2.42]), .04, BLUE)  # small faction plume on the forehead
    for s in (-1, 1):  # flank plates
        m.ell(H([s*.42, .05, 1.1]), [.07, .5, .3], STEEL)
        m.ell(H([s*.44, .05, .84]), [.06, .52, .05], GOLD)
    m.ell(H([0, .55, 1.12]), [.5, .44, .45], STEEL)  # crupper plate under the cloak
    m.ell(H([0, .55, .86]), [.52, .46, .06], GOLD)  # with a gold hem


def make():
    U.KITS[KIT] = dict(U.KITS['rider'], shoulder=BLUE)
    soldier, cavalry = U.soldier, U.cavalry

    def armoured_soldier(p, kit, weapon=True):
        out = soldier(p, kit, weapon)
        if kit == KIT:
            rider_armour(p)
        return out

    def royal(row, f, style=None, **kw):
        res = cavalry(row, f, style, **kw)
        if kw.get('kit') != KIT:
            return res
        style = style or A.SELECTED['cavalry']
        k = A.SPEAR[style]['frames'][f] if row == U.ATTACK else None
        m.meshes = []
        horse_armour(row, f)
        extra = m.meshes
        if k:
            move = U.horse_move(k.get('rear', 0), k.get('shift', 0))
            extra = [(move(v), c) for v, c in extra]
        res['faces'] = list(res['faces']) + extra
        return res

    U.soldier, U.cavalry = armoured_soldier, royal
    return M.rider_build(KIT), M.Hd.heads(M.Hd.IRON, M.Hd.PLUME)


M.register('royal_guard', make, vertical_spear=True)
