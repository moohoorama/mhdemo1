"""친위대 장수가 말에서 떨어져 엎어진 모습 (2026-11-16, v2): 19장 연진, 관우에게 베여 떨어진 문추 (sheet fallen_royal_guard).

Face down and spread like an X, the head toward the facing side (he fell forward off the charging horse): the right
arm reaching out past the head, the left arm fallen along his side, the left knee drawn up; the helmet knocked off
and rolled away ahead of him, the spear fallen crosswise apart from the body. The first version (on his back, helmet
on, cloak spread under him) was rejected: with 문추's palette (iron and faction both gold/yellow) helmet, face,
armour and cloak were one yellow lump. So the materials are chosen for contrast under any palette: iron only for the
back plate, pauldrons, vambraces and the helmet; a red robe (themeB) on the sleeves, the skirt and the trousers;
dark hair and boots (hair, horseMane tags, which no character recolours); no cloak. The head is modelled with the
pose (no 2D head; a 2D head turned on its side was rejected as grotesque, a 3D blob without a face as unreadable):
lying on its cheek with the face turned to +x, the camera side in the W-side views, shut eye, brow and moustache,
topknot with a red tie. Every row is the one still pose.

  grid.py new royal_guard fallen_royal_guard "쓰러진 친위대" && grid.py export fallen_royal_guard --force
  grid3d_mounted.py resize fallen_royal_guard 0 0 0 14
  grid3d_mounted.py fallen_royal_guard --unit royal_guard_fallen --base none --origin --anims idle --dirs N,S,W,NW,SW --mirror
  python3 mounted_kits/royal_guard_fallen.py     # RETOUCH, then copy idle to the other rows
  grid.py import fallen_royal_guard && python3 ../spritebuild/build.py units/fallen_royal_guard.yaml
"""
import math
import sys
from pathlib import Path

if __name__ != '__main__':
    import grid3d_mounted as M

    U, Hd, m, V, R = M.U, M.Hd, M.G.m, M.G.m.V, M.R
    NO_HEAD = Hd.heads({d: ['.'] for d in ('N', 'NW', 'W', 'SW', 'S')})
    ARMOUR, TRIM, SKIN = U.STEEL, U.GOLD, U.SKIN
    ROBE, HAIR, BOOT = '#c01c2c', '#2a2220', '#3a2a22'
    SHAFT, TIP, EYE = '#966131', '#d6e7ec', '#211f22'
    # private render keys: robe -> themeB (q p), hair -> hair (i h), boots -> horseMane (A z)
    R.add_ramps({HAIR: '7788', BOOT: 'xxyy', ROBE: '9zzz'})
    R.COLOR.update({'7': (42, 34, 32), '8': (70, 52, 44), 'x': (58, 42, 34), 'y': (92, 66, 50), '9': (171, 22, 51),
                    'z': (237, 35, 66)})
    M.KEY2TAG.update({'7': 'i', '8': 'h', 'x': 'A', 'y': 'z', '9': 'q', 'z': 'p'})


def limb(a, b, r, c):
    m.rod(V(a), V(b), r, c)


def arm(sh, el, ha):
    limb(sh, el, .14, ROBE)
    limb(el, V(el)*.15 + V(ha)*.85, .13, ARMOUR)  # vambrace
    m.ell(V(ha), [.13]*3, SKIN)


def leg(hip, knee, foot, toe):
    limb(hip, knee, .17, ROBE)
    limb(knee, foot, .14, BOOT)
    m.ell(V(foot)*.5 + V(toe)*.5, [.13, .13, .13], BOOT)


def spear(a, b):
    a, b = V(a), V(b)
    d = (b - a)/math.dist(a, b)
    limb(a, b - d*.35, .045, SHAFT)
    side = V([-d[1], d[0], 0])*.08
    m.poly([b - d*.35 - side, b, b - d*.35 + side, b - d*.42], TIP)
    m.ell(b - d*.38, [.08, .08, .06], ROBE)  # red tassel


def helmet(c, axis):
    """The iron helmet knocked off, lying on its side: bowl, dark opening toward axis, red plume away from it."""
    c, axis = V(c), V(axis)
    m.ell(c, [.3, .3, .26], ARMOUR)
    m.ell(c + axis*.2, [.2, .2, .17], HAIR)  # the dark opening
    m.ell(c - axis*.05 + V([0, 0, .02]), [.33, .33, .06], TRIM)  # brim band
    m.rod(c - axis*.25, c - axis*.6 + V([0, 0, .05]), .07, ROBE)  # plume
    m.ell(c - axis*.62 + V([0, 0, .05]), [.12, .12, .1], ROBE)


def head(c):
    """Face down on its cheek, crown toward -y, the face turned to +x: hair over the back, the crown and the -x side,
    the face with a shut eye, a brow and a moustache, the topknot with its red tie."""
    c = V(c)
    m.ell(c + V([-.1, -.06, .04]), [.42, .46, .4], HAIR)
    m.ell(c + V([.1, .06, -.02]), [.36, .42, .36], SKIN)
    m.ell(c + V([-.02, -.5, .1]), [.18, .18, .17], HAIR)  # topknot
    m.ell(c + V([-.02, -.36, .12]), [.15, .07, .13], ROBE)  # its tie
    m.box(c + V([.43, -.06, .06]), [.05, .16, .05], EYE)  # shut eye
    m.box(c + V([.42, -.14, .12]), [.05, .18, .06], HAIR)  # brow
    m.box(c + V([.42, .24, .02]), [.05, .2, .06], HAIR)  # moustache
    m.ell(c + V([.02, -.08, -.2]), [.3, .2, .12], HAIR)  # hair under the ear


def body():
    """The figure in model space: the facing side is -y, the body's middle on the origin."""
    m.ell([0, -.45, .3], [.44, .55, .26], ARMOUR)  # back plate
    m.ell([0, -.47, .5], [.08, .4, .06], TRIM)  # spine ridge
    m.ell([0, .37, .22], [.4, .32, .2], ROBE)  # robe skirt
    m.box([0, .17, .3], [.86, .12, .4], BOOT)  # belt
    for s in (-1, 1):
        m.ell([s*.44, -.87, .34], [.22, .24, .2], ARMOUR)  # pauldrons
    m.ell([0, -1.0, .32], [.32, .2, .2], ROBE)  # collar
    head([0, -1.41, .38])
    arm([.44, -.91, .3], [1.0, -1.35, .14], [1.2, -1.9, .12])  # reaching out past the head
    arm([-.44, -.91, .3], [-.88, -.3, .14], [-.85, .45, .12])  # fallen along his side
    leg([.22, .45, .2], [.5, 1.05, .15], [.62, 1.65, .14], [.65, 1.75, .05])
    leg([-.22, .45, .2], [-.9, .7, .15], [-.95, 1.3, .14], [-1.0, 1.4, .05])  # knee drawn up
    helmet([-1.95, -2.0, .26], [.7, .3, .2])
    spear([1.7, 1.55, .05], [1.9, -1.15, .05])


def make():
    def build(row, f, style=None):
        m.meshes = []
        body()
        return U.result(m.meshes, V([0, 0, .5]), 0, f)
    return build, NO_HEAD


# Hand fixes after conversion, per rendered view (NE, E, SE are mirrored from NW, W, SW before this runs, so the
# fixes are mirrored too): (x, y, tag).
RETOUCH = {
    'SW': [(46, 55, '#'), (46, 56, '#'), (42, 55, 'i'), (42, 56, 'i')],  # the face apart from the back plate; shut eye
}


def retouch():
    """Apply RETOUCH to the idle grids (mirroring the W-side fixes to the E side), then copy idle to every row."""
    grid = Path(__file__).resolve().parents[2] / 'heroes/fallen_royal_guard/grid'
    mirror = {'SW': 'SE', 'W': 'E', 'NW': 'NE'}
    pivot_x = 51
    fixes = {}
    for d, px in RETOUCH.items():
        fixes.setdefault(d, []).extend(px)
        if d in mirror:
            fixes.setdefault(mirror[d], []).extend((2*pivot_x - x, y, t) for x, y, t in px)
    for d, px in fixes.items():
        path = grid / 'idle' / f'{d}-0.txt'
        lines = path.read_text().split('\n')
        for x, y, t in px:
            i = next(n for n, l in enumerate(lines) if l.startswith(f'{y:02d}|'))
            row = lines[i]
            lines[i] = row[:3 + x] + t + row[4 + x:]
        path.write_text('\n'.join(lines))
    for f in (grid / 'idle').glob('*-0.txt'):
        d = f.stem.split('-')[0]
        for anim in ('idle', 'walk', 'attack', 'hit', 'exhausted', 'block'):
            for n in range(4):
                if (anim, n) != ('idle', 0):
                    (grid / anim / f'{d}-{n}.txt').write_text(f.read_text())


if __name__ == '__main__':
    retouch()
    sys.exit()

M.register('royal_guard_fallen', make)
