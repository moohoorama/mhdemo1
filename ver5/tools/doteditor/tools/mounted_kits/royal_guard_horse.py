"""친위대 말 (2026-11-16): the royal_guard horse with its barding and saddle and no rider, for a story scene where the
rider has fallen (19장 연진, 문추's horse running on after 관우 cuts him down). Same renders, palette keys and cell as
royal_guard, so a character's horseSkin, saddle and iron colours recolour it; no head is stamped, the reins hang
loose from the bit to the neck instead of running up to the rider's hands, and there is no weapon.

  grid.py new royal_guard royal_guard_horse "친위대 말" && grid.py export royal_guard_horse --force
  grid3d_mounted.py royal_guard_horse --unit royal_guard_horse --base none --anims idle,walk --dirs N,S,W,NW,SW --mirror
  (the other rows are copies of idle) then grid.py import royal_guard_horse
"""
import grid3d_mounted as M
from mounted_kits import royal_guard as RG

U, Hd, m, V = M.U, M.Hd, M.G.m, M.G.m.V
REINS = '#38261d'
NO_HEAD = Hd.heads({d: ['.'] for d in ('N', 'NW', 'W', 'SW', 'S')})


def make():
    RG.make()

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
            return U.cavalry(row, f, None, kit=RG.KIT, coat='dot', weapon='none')
        finally:
            U.soldier, m.rod = soldier, rod
    return build, NO_HEAD


M.register('royal_guard_horse', make)
