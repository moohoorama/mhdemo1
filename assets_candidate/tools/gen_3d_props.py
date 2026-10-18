"""Scenario props modelled in 3D and drawn with the game's own cel renderer (tools/unit3d): the same camera, palette and 1px outline as
the existing props (peach tree, tavern). Models here: a castle hall (관청), a plum tree, a Taihu garden rock and a garden pavilion.
Needs numpy and Pillow; imports ../tools/unit3d read-only."""
import math, os, sys
sys.path.insert(0, os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))), "tools", "unit3d"))
import numpy as np
from PIL import Image
import base_model as m
import render as R
import props as P
import structures as S

V = S.V
TILE_R, TILE_D = '#4a4d5a', '#30323c'
EAVE, EAVE_D = '#7a4a2a', '#4e2e1c'
PLUM_W, PLUM_P, PLUM_R = '#f4e6e2', '#e8b6c0', '#c4283c'
ROCK, ROCK_D, ROCK_L = '#9aa6ae', '#6f7b84', '#c4ccd0'
TRUNK, TRUNK_L = '#3a2a22', '#5c4234'
HOLE = '#1c1a1e'
FLOOR_W, FLOOR_D = '#a06a3c', '#6e4126'
R.add_ramps({TILE_R: '2345', TILE_D: '1233', EAVE: 'fggh', EAVE_D: '0fff', PLUM_W: 'SS6D', PLUM_P: 'BCDD', PLUM_R: 'bcde',
             ROCK: '3455', ROCK_D: '2344', ROCK_L: '4566', TRUNK: '0fff', TRUNK_L: 'fggh', HOLE: '0000', FLOOR_W: 'ghhi', FLOOR_D: 'fggh'})


def finish(size, pivot, k=1.0):
    if k != 1.0:
        m.meshes = [(np.asarray(v) * k, c) for v, c in m.meshes]
    return P.draw(size, pivot)


def swap_xy():
    """mirror a model across the diagonal (x, y) -> (-y, -x), which swaps the two faces the camera sees: the u and v variants of a building"""
    m.meshes = [(np.asarray(v)[:, [1, 0, 2]] * [-1, -1, 1], c) for v, c in m.meshes]


def hip_roof(x0, x1, y0, y1, z0, rise, main, dark, curl=.07):
    """a hip roof over [x0,x1] x [y0,y1] at height z0, ridge along x; the four eave corners curl up a little"""
    yc = (y0 + y1) / 2
    rl = (x1 - x0) * .22
    ra, rb = x0 + rl, x1 - rl
    zt = z0 + rise
    up = lambda z: z + curl
    m.poly([V(x0, y0, up(z0)), V(x1, y0, up(z0)), V(rb, yc, zt), V(ra, yc, zt)], main)       # the near slope (toward -y)
    m.poly([V(x0, y1, up(z0)), V(ra, yc, zt), V(rb, yc, zt), V(x1, y1, up(z0))], dark)       # the far slope
    m.poly([V(x0, y0, up(z0)), V(ra, yc, zt), V(x0, y1, up(z0))], dark)                       # the hip at -x
    m.poly([V(x1, y0, up(z0)), V(x1, y1, up(z0)), V(rb, yc, zt)], main)                       # the hip at +x
    m.rod(V(ra, yc, zt + .01), V(rb, yc, zt + .01), .03, TILE_D)
    for x, y in ((x0, y0), (x1, y0), (x0, y1), (x1, y1)):
        m.rod(V(x, y, z0), V(x, y, up(z0) + .03), .02, EAVE)
    for y in (y0, y1):
        m.rod(V(x0, y, z0 - .005), V(x1, y, z0 - .005), .035, EAVE)
    for x in (x0, x1):
        m.rod(V(x, y0, z0 - .005), V(x, y1, z0 - .005), .035, EAVE_D)
    # tile courses
    for k in range(1, 5):
        t = k / 5
        for y, c in ((y0 + (yc - y0) * t, TILE_D), (y1 - (y1 - yc) * t, TILE_D)):
            m.rod(V(x0 + (ra - x0) * t, y, z0 + rise * t + .01), V(x1 - (x1 - rb) * t, y, z0 + rise * t + .01), .014, c)


def hall():
    """a castle hall: two-step stone base, plastered wall with a dark doorway between red pillars, a hip roof with curled eaves.
    The front long wall faces the viewer (toward -y)."""
    m.meshes = []
    w, d, h = 1.5, 1.0, .62
    m.box(V(0, 0, .05), V(w + .3, d + .3, .1), S.STONE_DARK)
    m.box(V(0, 0, .15), V(w + .16, d + .16, .1), S.STONE)
    z = .2
    m.box(V(0, 0, z + h / 2), V(w, d, h), S.PLASTER)
    m.box(V(0, -d / 2 - .001, z + .27), V(.34, .02, .5), P.INK)                   # the doorway
    m.box(V(0, -d / 2 - .002, z + .56), V(.42, .02, .05), P.GOLD)               # lintel plaque
    for x in (-w / 2, -w / 4, w / 4, w / 2):                                      # the front colonnade
        m.rod(V(x, -d / 2 - .06, z), V(x, -d / 2 - .06, z + h + .02), .035, P.LACQUER)
    for x in (-w / 2, w / 2):
        m.rod(V(x, d / 2 + .0, z), V(x, d / 2, z + h + .02), .035, P.LACQUER_DARK)
    m.box(V(0, -d / 2 - .06, z + h), V(w + .06, .06, .05), P.GOLD)                 # the beam over the columns
    for x in (-w / 2 + .1, w / 2 - .1):                                           # window slits on the side walls
        m.box(V(x, -d / 2 - .002, z + .38), V(.12, .02, .16), P.INK)
    # steps to the door
    m.box(V(0, -d / 2 - .2, .1), V(.5, .14, .06), S.STONE)
    m.box(V(0, -d / 2 - .3, .07), V(.6, .12, .06), S.STONE_DARK)
    hip_roof(-w / 2 - .22, w / 2 + .22, -d / 2 - .26, d / 2 + .2, z + h + .04, .5, TILE_R, TILE_D)
    return finish((190, 170), (95, 128), 1.7)


def plum_tree():
    """a plum in blossom: a short leaning trunk, limbs spreading wide and crooked, and a loose canopy of many small blossom clusters"""
    m.meshes = []
    m.rod(V(0, 0, 0), V(-.05, .03, .3), .08, TRUNK)
    m.rod(V(-.05, .03, .26), V(.1, -.03, .55), .065, TRUNK)
    limbs = [((.1, -.03, .52), (.42, .1, .8), (.62, .08, 1.0)), ((.1, -.03, .52), (-.3, -.1, .75), (-.55, -.02, .95)),
             ((.05, 0, .5), (.0, .38, .78), (.12, .55, .92)), ((.05, 0, .5), (.06, -.4, .8), (-.1, -.55, .9)),
             ((.08, 0, .55), (.05, .05, .85), (.0, 0, 1.1))]
    for a0, a1, a2 in limbs:
        m.rod(V(*a0), V(*a1), .04, TRUNK_L)
        m.rod(V(*a1), V(*a2), .028, TRUNK)
    rnd = np.random.default_rng(11)
    for a0, a1, a2 in limbs:
        for t in (.35, .7, 1.0, 1.0, 1.1):
            p = np.array(a1) * (1 - t) + np.array(a2) * t if t <= 1 else np.array(a2) + (np.array(a2) - np.array(a1)) * .2
            for _ in range(3):
                d = rnd.normal(0, .09, 3) * [1, 1, .6]
                r = float(rnd.uniform(.045, .075))
                col = PLUM_W if rnd.random() < .62 else (PLUM_P if rnd.random() < .6 else PLUM_R)
                m.ell(V(*(p + d)), [r, r, r * .85], col)
    return finish((120, 120), (60, 104), 1.9)


def spire(x, y, r, h, lean, col=None):
    """a four-sided jagged spire: a square base r wide, the tip leaning by `lean` (dx, dy)"""
    col = col or ROCK
    b = [V(x - r, y - r, 0), V(x + r, y - r * .8, 0), V(x + r * .9, y + r, 0), V(x - r * 1.1, y + r * .9, 0)]
    tip = V(x + lean[0], y + lean[1], h)
    mid = [V(p[0] * .8 + (x + lean[0] * .4) * .2, p[1] * .8 + (y + lean[1] * .4) * .2, h * .55) for p in b]
    for i in range(4):
        j = (i + 1) % 4
        m.poly([b[i], b[j], mid[j], mid[i]], col)
        m.poly([mid[i], mid[j], tip], col)


def taihu_rock():
    """a Taihu garden rock: craggy and twisting, five spires of different heights and leans rising from one wide base, with holes through it"""
    m.meshes = []
    m.ell(V(0, 0, .07), [.46, .34, .11], ROCK_D)
    spire(-.05, .05, .2, 1.35, (.1, -.05))
    spire(-.3, -.02, .16, .85, (-.1, .06), ROCK_L)
    spire(.22, .0, .16, .95, (.08, .05))
    spire(.08, -.2, .13, .55, (.0, -.05), ROCK_L)
    spire(-.12, .22, .12, .62, (-.04, .08))
    for x, y, z, rx, rz in ((-.05, -.17, .6, .07, .12), (.03, -.12, 1.0, .05, .09), (-.3, -.16, .4, .05, .08), (.22, -.15, .5, .05, .09), (.0, -.1, .25, .08, .06)):
        m.ell(V(x, y, z), [rx, .06, rz], HOLE)
    return finish((90, 110), (45, 96), 1.5)


def pavilion():
    """a garden pavilion over 2 x 2 cells: a wooden deck with a low rail, four red pillars, beams, a hip roof with curled eaves"""
    m.meshes = []
    w = 1.0
    m.box(V(0, 0, .06), V(2 * w + .2, 2 * w + .2, .12), S.STONE_DARK)
    m.box(V(0, 0, .14), V(2 * w + .1, 2 * w + .1, .05), FLOOR_W)
    pil = w * .86
    for x, y in ((-pil, -pil), (pil, -pil), (-pil, pil), (pil, pil)):
        m.rod(V(x, y, .16), V(x, y, 1.15), .045, P.LACQUER)
        m.ell(V(x, y, .17), [.07, .07, .035], S.STONE)
    for (a, b) in (((-pil, -pil), (pil, -pil)), ((-pil, pil), (pil, pil)), ((-pil, -pil), (-pil, pil)), ((pil, -pil), (pil, pil))):
        m.rod(V(a[0], a[1], 1.12), V(b[0], b[1], 1.12), .035, P.GOLD)
        m.rod(V(a[0], a[1], .5), V(b[0], b[1], .5), .02, P.LACQUER_DARK)   # the rail
    hip_roof(-w - .3, w + .3, -w - .3, w + .3, 1.14, .7, TILE_R, TILE_D, curl=.1)
    m.rod(V(0, 0, 1.84), V(0, 0, 2.0), .02, P.GOLD)
    return finish((200, 190), (100, 150), 1.1)


def models():
    out = {}
    for name, fn in (("hall_u", hall), ("plum_tree", plum_tree), ("taihu_rock", taihu_rock), ("pavilion", pavilion)):
        im = fn()
        out[name] = (im, im.pivot)
    m.meshes = []
    im = hall()
    swap_xy()
    out["hall_v"] = (P.draw((190, 170), (95, 128)), (96, 129))
    return out


def trimmed(im, pivot):
    x0, y0, x1, y1 = im.getbbox()
    return im.crop((x0, y0, x1, y1)), (pivot[0] - x0, pivot[1] - y0)


def run():
    import gen_all
    for name, (im, pv) in models().items():
        t, p = trimmed(im, pv)
        gen_all.save("tiles", name, t, p, "object", scale="scene", made="3D model + unit3d cel renderer")


if __name__ == "__main__":
    # preview: python3 gen_3d_props.py OUT_DIR writes every model's trimmed picture there
    out = sys.argv[1]
    for name, (im, pv) in models().items():
        t, p = trimmed(im, pv)
        print(name, t.size, p)
        t.save(os.path.join(out, "3d_" + name + ".png"))
