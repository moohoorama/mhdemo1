"""주창 (적병 -> 흉적 -> 의적): the bandit line in shaggy furs, a mountain bandit from 와우산.

Every tier keeps the 흉적's thick shaggy beard, faction headband and heavy dadao (two-handed overhead chop, slanted
guard); the hemp top is covered by an open fur vest (bare chest down the front, ragged hem, fur on both shoulders).
흉적 adds a wide fur mantle round the neck and a bear hide down the back; 의적 the leather cuirass straps over the
vest and the 의적's faction cape under the fur mantle. Sheet ids: zhoucang (적병), zhoucang_ruffian,
zhoucang_outlaw (Look.Sprites).
Fur uses the 흉적's pelt colours (render keys k l m -> themeB, tufts -> themeA); grid.py colors sets the browns.
"""
import grid3d_mounted as M
from mounted_kits import outlaw as OK, ruffian as RK

G, U, A, B, Bl, Hd = M.G, M.U, M.A, M.B, M.Bl, M.Hd
m, V = U.m, U.V

FUR, FUR_LIGHT, TUFT = RK.PELT, RK.PELT_LIGHT, RK.STRIPE


def vest(T):
    """Open fur vest: a bulky shaggy body over the hemp top, the chest bare down the front, a fur collar and fur
    shoulders standing out past the arms, a ragged hem of uneven tufts."""
    m.ell(T([0, .04, 1.2]), [.45, .36, .38], FUR)
    m.ell(T([0, .04, 1.5]), [.36, .29, .1], FUR_LIGHT)
    for s in (-1, 1):
        m.ell(T([s*.39, .02, 1.52]), [.25, .29, .16], FUR_LIGHT)
    for i, x in enumerate((-.36, -.24, -.12, 0, .12, .24, .36)):
        drop = .1 if i % 2 else 0
        m.box(T([x, .03, .9 - drop/2]), [.14, .62, .12 + drop], FUR)
    for x, y, z in ((-.24, .33, 1.34), (.2, .34, 1.12), (-.06, .35, .98), (.32, .24, 1.0), (-.36, .12, 1.18),
                    (.38, .1, 1.3), (-.3, -.2, 1.06), (.3, -.22, 1.1)):
        m.box(T([x, y, z]), [.08, .05, .13], TUFT)
    m.ell(T([0, -.3, 1.3]), [.11, .06, .17], U.SKIN)


def mantle(T):
    """Wide fur mantle round the neck and over both shoulders, and a bear hide hanging down the back."""
    m.ell(T([0, .06, 1.52]), [.52, .36, .14], FUR_LIGHT)
    for z, w in ((1.36, .62), (1.12, .66), (.9, .58), (.7, .46)):
        m.box(T([0, .34, z]), [w, .08, .25], FUR)
    for x, z in ((-.2, 1.24), (.16, 1.02), (-.08, .8)):
        m.box(T([x, .39, z]), [.1, .04, .12], TUFT)


def tier(name, base, extra):
    """make() for one tier: base.make() (the class body, dadao and motions) with the fur laid over its kit."""
    def make():
        base.make()
        U.KITS[name] = dict(base.KIT)
        A.SELECTED[name] = 'ruffian_chop'
        soldier = U.soldier

        def dressed(p, k, weapon=True):
            out = soldier(p, k, weapon)
            if k != name:
                return out
            T = U.transform(p)
            vest(T)
            extra(p, T)
            return m.meshes, out[1]

        U.soldier = dressed
        foot = U.foot_unit(name)

        def block(f):
            k = RK.GUARD[f]
            faces, head = U.soldier(Bl.foot_pose(k), name)
            return Bl.displace(U.result(faces, head, 0, f), k.get('sway', 0), k.get('back', 0))

        def build(row, f, style=None):
            if row == B.BLOCK:
                return block(f)
            t = U.VIEW['t']
            if row == U.ATTACK and style is None and t is not None and abs(t[0]) < .1:
                style = 'ruffian_chop_diagonal'
            return foot(row, f, style=style) if style else foot(row, f)
        return build, RK.heads()
    return make


M.register('zhoucang', tier('zhoucang', RK, lambda p, T: None))
M.register('zhoucang_ruffian', tier('zhoucang_ruffian', RK, lambda p, T: mantle(T)))
M.register('zhoucang_outlaw', tier('zhoucang_outlaw', OK, lambda p, T: (OK.armour(T), OK.cape(p, T), mantle(T))))
