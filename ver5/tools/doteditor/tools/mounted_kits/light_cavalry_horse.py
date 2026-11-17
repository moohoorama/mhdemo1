"""경기병 말 (2026-11-17): the light_cavalry horse with its saddle and no rider, for a story scene where the riders
have got down (19장 연진, 문추's van loading the baggage 조조 left as bait). Same renders, palette keys and cell as
light_cavalry, so the faction colours recolour it; no head is stamped, the reins hang loose from the bit to the neck,
and there is no weapon.

  grid.py new light_cavalry light_cavalry_horse "경기병 말" && grid.py export light_cavalry_horse --force
  grid3d_mounted.py light_cavalry_horse --unit light_cavalry_horse --base none --anims idle,walk --dirs N,S,W,NW,SW --mirror
  (the other rows are copies of idle) then grid.py import light_cavalry_horse
"""
import grid3d_mounted as M

U, Hd, m, V = M.U, M.Hd, M.G.m, M.G.m.V
REINS = '#38261d'
NO_HEAD = Hd.heads({d: ['.'] for d in ('N', 'NW', 'W', 'SW', 'S')})


def make():
    def riderless(p, kit, weapon=True):
        m.meshes = []
        return [], U.transform(p)(U.HEAD)

    def loose_rod(a, b, r, color):
        if color == REINS:  # from the bit back along the side of the neck, slack
            b = V([a[0]*1.9, -.62, a[2] - .12])
        return rod(a, b, r, color)

    rod = m.rod

    def build(row, f, style=None):
        soldier = U.soldier
        U.soldier, m.rod = riderless, loose_rod
        try:
            return U.cavalry(row, f, None, kit='rider', coat='dot', weapon='none')
        finally:
            U.soldier, m.rod = soldier, rod
    return build, NO_HEAD


M.register('light_cavalry_horse', make)
