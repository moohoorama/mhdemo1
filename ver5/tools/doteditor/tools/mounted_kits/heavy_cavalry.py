"""중기병: the 경기병 rider with the 중보병's body armour (steel cuirass, belt and tassets, faction pauldrons with a steel
cap, steel vambraces, a short faction cloak) and steel barding on the horse (breast plate, neck guard, face plate, a
faction caparison over the rump). Head: iron helmet with plume (switch to the hood: Hd.HOOD, Hd.HOOD_CLOTH)."""
import grid3d_mounted as M

U, A, m = M.U, M.A, M.G.m
V = m.V
STEEL, BLUE, LIGHT_BLUE, LEATHER = U.STEEL, U.BLUE, U.LIGHT_BLUE, U.BROWN
KIT = 'rider_heavy'


def rider_armour(p):
    """The 중보병 body armour without a shield, cloak shortened to the saddle (it is drawn in the soldier's space,
    so units.cavalry scales it with the rider)."""
    T = U.transform(p)
    m.ell(T([0, -.04, 1.12]), [.42, .3, .42], STEEL)  # cuirass, wide enough to show from the side
    m.box(T([0, -.06, .78]), [.62, .5, .09], LEATHER)  # belt
    for x in (-.18, 0, .18):  # tassets over the saddle
        m.box(T([x, -.29, .6]), [.14, .05, .34], LEATHER)
    for s in (-1, 1):
        sh = T([s*.4, 0, 1.42])
        m.ell(sh + V([s*.01, 0, -.02]), [.22, .24, .2], BLUE)  # faction pauldron
        m.ell(sh + V([s*.01, 0, .09]), [.2, .21, .09], STEEL)  # with a steel cap
        el, ha = T(p['elbows'][0 if s < 0 else 1]), T(p['hands'][0 if s < 0 else 1])
        m.rod(el*.45 + ha*.55, ha*.9 + el*.1, .15, STEEL)  # vambrace on the forearm
    for z, w, y, h in [(1.52, .82, .29, .2), (1.30, .84, .31, .26), (1.02, .88, .33, .32)]:
        m.box(T([0, y, z]), [w, .08, h], BLUE)  # short cloak, ends at the waist
    m.ell(T([0, -.02, 1.5]), [.3, .22, .1], BLUE)  # cloak collar


def horse_armour(row, f):
    """Barding in horse space: the same transform units.cavalry gives the horse (walk bounce, nodding head,
    the lowered forehand when exhausted); the caller then applies the attack's rear/shift."""
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

    m.ell(H([0, -.56, 1.12]), [.44, .22, .4], STEEL)  # breast plate, flush with the chest
    m.ell(H([0, -.5, 1.5]), [.12, .08, .1], BLUE)  # faction chest tassel
    m.rod(H([0, -.66, 1.34]), H([0, -.9, 1.8]), .28, STEEL)  # crinet over the neck
    m.ell(H([0, -1.12, 1.9]), [.15, .27, .15], STEEL)  # face plate
    m.ell(H([0, .5, 1.15]), [.5, .46, .47], BLUE)  # caparison over the rump
    m.ell(H([0, .5, .98]), [.52, .48, .16], LIGHT_BLUE)  # light trim at its hem


def make():
    U.KITS[KIT] = dict(U.KITS['rider'], shoulder=BLUE)
    soldier, cavalry = U.soldier, U.cavalry

    def armoured_soldier(p, kit, weapon=True):
        out = soldier(p, kit, weapon)
        if kit == KIT:
            rider_armour(p)
        return out

    def heavy(row, f, style=None, **kw):
        res = cavalry(row, f, style, **kw)
        if kw.get('kit') != KIT:
            return res
        style = style or A.SELECTED['cavalry']
        k = A.SPEAR[style]['frames'][f] if row == U.ATTACK else None
        m.meshes = []
        horse_armour(row, f)
        extra = m.meshes
        if k:  # the mount pitches and slides with the attack (the soldier's own meshes were moved by cavalry)
            move = U.horse_move(k.get('rear', 0), k.get('shift', 0))
            extra = [(move(v), c) for v, c in extra]
        res['faces'] = list(res['faces']) + extra
        return res

    U.soldier, U.cavalry = armoured_soldier, heavy
    return M.rider_build(KIT), M.Hd.heads(M.Hd.IRON, M.Hd.PLUME)


M.register('heavy_cavalry', make, vertical_spear=True)
