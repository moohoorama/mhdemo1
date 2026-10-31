"""중기병: the 경기병 rider with the 중보병's body armour (steel cuirass, belt and tassets, faction pauldrons with a steel
cap, steel vambraces, a short faction cloak) and steel barding on the horse (breast plate, neck guard, face plate, a
steel crupper plate over the rump; the faction shows only in the chest tassel). Head: iron helmet with plume (switch
to the hood: Hd.HOOD, Hd.HOOD_CLOTH)."""
import grid3d_mounted as M

U, A, m = M.U, M.A, M.G.m
V = m.V
STEEL, BLUE = U.STEEL, U.BLUE
KIT = 'rider_heavy'


def rider_armour(p):
    """The 중보병 body armour without a shield, cloak shortened to the saddle (it is drawn in the soldier's space,
    so units.cavalry scales it with the rider)."""
    M.G.short_armour(p)


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
    m.ell(H([0, .5, 1.15]), [.5, .46, .47], STEEL)  # crupper plate over the rump


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
