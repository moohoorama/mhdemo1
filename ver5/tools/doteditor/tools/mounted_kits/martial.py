"""Shared parts of the bare-handed line (무도가 brawler, 권사 pugilist): the bald head and the martial uniform.

Not a unit (no register call): mounted_kits/brawler.py and mounted_kits/pugilist.py import it.
Uniform: a sleeveless faction-colour top closed across the chest so the skin shows in a V (dark faction edges on the
collar), a dark leather belt with a short two-tail knot at the right hip, hemp trousers (the kit's cloth).
setup() holds the shared-state patches and is called from each kit's make().
"""
import numpy as np

import grid3d_mounted as M

U, Hd, R = M.U, M.Hd, M.R
m, V = U.m, U.V

TOP = U.LIGHT_BLUE      # faction cloth: ramp X Y Z Z -> faction3 shadow, faction2 base, faction1 light
TRIM = '#28486f'        # collar edges: darkest faction step only (ramp W W W W -> faction3)
BELT = '#3a2418'        # dark leather: ramp f f g g -> themeA3 / themeA2


# Clean shaved head, drawn directly (not a recoloured helmet): a round cranium as wide as the face, ears at the sides (front, 3/4 and side views; the back views stay a plain dome).
# skin keys: j crown highlight (skin1), i skin (skin2), x back of the head and neck side (skin3); eyes as in the
# other heads (2px tall, S/W rows 7-8, SW rows 8-9) so expressions and the anchor work unchanged.
BALD = {
    'S': Hd.grid('.............', '.............', '....00000....', '..00jjjii00..',
                 '.0jjjjjiiii0.', '.0jjjiiiiii0.', '.0iiiiiiiix0.', '.0xi0iii0ix0.',
                 '.0xi0iii0ix0.', '.0iiiiiiiix0.', '..0iiiiiix0..', '...0xxxxx0...'),
    'SW': Hd.grid('.............', '.............', '....00000....', '..00jjjii00..',
                  '.0jjjjiiiix0.', '.0jjjiiiixx0.', '.0iiiiiiixx0.', '.0iiiiiiixx0.',
                  '.0i0ii0iixi0.', '.0i0ii0iixi0.', '..0iiiiiixx0.', '...0iiiixx0..'),
    'W': Hd.grid('.............', '.............', '.....00000...', '...00jjjii00.',
                 '..0jjjjiiixx0', '.0jjjiiiixxx0', '.0iiiiiiixxx0', '.0i0iiiixixx0',
                 '0ii0iiiixixx0', '.0iiiiiiixx0.', '..0iiiiixx0..', '...000000....'),
    'NW': Hd.grid('.............', '.............', '....00000....', '..00jjjii00..',
                  '.0jjjjiiiix0.', '.0jjiiiiixx0.', '.0iiiiiixxx0.', '.0iiiiixxxx0.',
                  '.0iiiixxxxx0.', '.0iiixxxxxx0.', '..0ixxxxxx0..', '...0000000...'),
    'N': Hd.grid('.............', '.............', '....00000....', '..00jjjii00..',
                 '.0jjjjjiiii0.', '.0jjjiiiiix0.', '.0iiiiiiixx0.', '.0iiiiiixxx0.',
                 '.0iiiiiixxx0.', '.0iiiiixxxx0.', '..0iixxxxx0..', '...0000000...'),
}


def head():
    return Hd.heads(BALD, skin='ijx', lid='i')


def kit(**extra):
    """units.KITS entry: hemp trousers (cloth), dark belt (tabard), bare skin arms; the top is drawn by uniform()."""
    return dict(dict(cloth=U.HEMP, tabard=BELT, shoulder=U.SKIN, arms=U.SKIN, shield=False, weapon=None), **extra)


def uniform(p, T):
    """Faction top over the soldier's upper torso, chest V with dark edges, belt knot (the front flap is dropped)."""
    m.meshes[:] = [(v, c) for v, c in m.meshes if not (c == BELT and np.asarray(v)[:, 2].max() < .97)]
    m.ell(T([0, 0, 1.26]), [.41, .28, .385], TOP)
    m.ell(T([0, -.035, 1.3]), [.37, .285, .28], TOP)
    m.box(T([0, 0, 1.01]), [.8, .57, .11], BELT)
    m.ell(T([-.36, -.17, 1.0]), [.09, .08, .08], BELT)
    m.box(T([-.38, -.2, .87]), [.08, .05, .2], BELT)
    m.box(T([-.3, -.22, .89]), [.07, .05, .16], BELT)
    # a narrow V: the top must read as faction cloth at 1x even with the fists raised in front of it
    m.poly([T([-.1, -.29, 1.54]), T([.1, -.29, 1.54]), T([0, -.33, 1.3])], U.SKIN)
    m.rod(T([-.13, -.28, 1.56]), T([.01, -.33, 1.28]), .035, TRIM)
    m.rod(T([.13, -.28, 1.56]), T([-.01, -.33, 1.28]), .035, TRIM)


def setup():
    R.add_ramps({U.SKIN: 'xijj', TRIM: 'WWWW', BELT: 'ffgg'})
    R.COLOR['x'] = (150, 94, 43)  # preview colour of the dark skin key
    M.KEY2TAG.update(S='s', T='t', U='u', j='d', i='e', x='f')  # hemp -> themeC; skin j/i/x -> skin1/2/3
