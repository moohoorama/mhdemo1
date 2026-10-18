"""A scenario-scale castle kit modelled in 3D (unit3d cel renderer): the battle map's walls are deformed and low, so the scenes get their own:
  swall_{u,v}        a tall plain wall with a crenellated top, one small cell long (seamless along its run)
  sgate_{closed,open}_{first,second}_{u,v}  a gate tower four cells wide with a two-cell passage a column can walk through, doors, an upper
                     storey with a tiled roof; cut in two across its run: the first half (up the screen) is drawn before the units, the second (with
                     the roof) after them, so a unit in the passage stands behind the second half's pier
_u: the piece runs along u (the wall's long face looks SW); _v: along v (looks SE). Pivot: the ground centre of the piece."""
import math, os, sys
sys.path.insert(0, os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))), "tools", "unit3d"))
import numpy as np
from PIL import Image
import base_model as m
import render as R
import props as P
import structures as S
import gen_3d_props as G3

V = S.V
BRICK, BRICK_D, BRICK_L = '#b4bcc2', '#767f87', '#d2d8dc'
GATE_STONE = '#9aa3aa'
R.add_ramps({BRICK: '3455', BRICK_D: '2344', BRICK_L: '4566', GATE_STONE: '3445'})
H, T = 3.2, 1.0    # wall height and thickness (a whole cell, flush with the gate), in model units (one unit = one small cell edge; an officer is 1.5 high)


def courses(x0, x1, y, z0, z1, along="x"):
    """mortar lines on a wall face: horizontal courses with staggered vertical joints, as thin protruding boxes (a rod this thin vanishes
    at one pixel in some views)"""
    k = 0
    z = z0 + .3
    while z < z1 - .05:
        m.box(V((x0 + x1) / 2, y, z), V(x1 - x0, .05, .05), BRICK_D)
        zz = z - .3
        joints = [0.0] if k % 2 == 0 else [-.25, .25]
        for jx in joints:
            m.box(V(jx, y, (zz + z) / 2), V(.05, .05, .3), BRICK_D)
        z += .3
        k += 1


def merlons(y, z0, thick=.2, base=.26, h=.6):
    """one parapet along a wall: a base slab and one merlon per cell, whole inside the cell with a dip at both cell edges, an arrow slit and a cap"""
    m.box(V(0, y, z0 + base / 2), V(1.0, thick, base), BRICK)
    m.box(V(0, y, z0 + base + h / 2), V(.56, thick, h), BRICK_L)
    m.box(V(0, y, z0 + base + h + .03), V(.56, thick, .06), BRICK_D)
    m.box(V(0, y - thick / 2 - .005, z0 + base + h * .55), V(.07, .02, .28), P.INK)


def wall_body(tall=H):
    m.box(V(0, 0, tall / 2), V(1.0, T, tall), BRICK)
    m.box(V(0, 0, .12), V(1.0, T + .08, .24), BRICK_D)              # plinth
    courses(-.5, .5, -T / 2 - .025, .24, tall - .2)
    m.box(V(0, 0, tall - .08), V(1.0, T + .14, .16), BRICK_D)        # the corbel course under the parapets


def strip():
    """a wall piece is rendered as three pieces in a row and only the middle one is kept (cropped to 16 px, one cell's run), so its ends
    carry no outline and neighbours join seamlessly"""
    base = list(m.meshes)
    m.meshes = [(np.asarray(v) + [dx, 0, 0], c) for dx in (-1.0, 0.0, 1.0) for v, c in base]


def wall():
    m.meshes = []
    wall_body()
    merlons(-T / 2 + .1, H)                                         # the high outer parapet, toward the camera
    merlons(T / 2 - .08, H, thick=.14, base=.22, h=0)               # a low inner kerb behind the walkway
    strip()
    return G3.finish((80, 130), (40, 112), 1.0)


W2, D, PIER, PH = 4.0, 1.0, 1.0, 2.5                                # gate: width, depth, pier width, passage height
OPEN = W2 - 2 * PIER
HALF = {}


def gb(cx, y0, y1, z0, z1, wx, color):
    """a box over y in [y0, y1], clipped to the half being drawn along x"""
    x0, x1 = max(cx - wx / 2, HALF["x"][0]), min(cx + wx / 2, HALF["x"][1])
    if x1 - x0 > 1e-6:
        m.box(V((x0 + x1) / 2, (y0 + y1) / 2, (z0 + z1) / 2), V(x1 - x0, y1 - y0, z1 - z0), color)


def gate(closed, part):
    """a gate tower along x: a passage along y with a stepped arch, a plastered upper storey with dark windows and a tiled hip roof.
    part: "first" (x < 0, up the screen), "second" (x >= 0, with the roof) or "all"; the halves only tell which pixels of the whole belong
    where (see models), so the cut leaves no outline"""
    m.meshes = []
    HALF["x"] = {"first": (-9.0, 0.0), "second": (0.0, 9.0), "all": (-9.0, 9.0)}[part]
    for sx in (-1, 1):
        cx = sx * (W2 / 2 - PIER / 2)
        gb(cx, -D / 2, D / 2, 0, H, PIER, GATE_STONE)
        gb(cx, -D / 2 - .04, D / 2 + .04, 0, .24, PIER + .06, BRICK_D)
        for z in (.5, .8, 1.1, 1.4, 1.7, 2.0, 2.3, 2.6, 2.9):
            gb(cx, -D / 2 - .05, -D / 2, z - .025, z + .025, PIER, BRICK_D)
    gb(0, -D / 2, D / 2, PH, H, OPEN + .02, GATE_STONE)             # the lintel block over the passage
    for i, w in enumerate((.45, .3, .15)):                          # the arch steps down into the corners
        for sx in (-1, 1):
            gb(sx * (OPEN / 2 - w / 2), -D / 2, D / 2, PH - .15 * (i + 1), PH - .15 * i, w, GATE_STONE)
    gb(0, -D / 2 - .03, -D / 2, PH - .06, PH + .06, OPEN - .2, P.GOLD)   # a gilded band along the arch
    if closed:
        for sx in (-1, 1):
            gb(sx * OPEN / 4, -D / 2 + .06, -D / 2 + .22, 0, PH - .3, OPEN / 2 - .03, P.PLANK)
            for z in (.45, 1.0, 1.55, 2.0):
                gb(sx * OPEN / 4, -D / 2 + .02, -D / 2 + .06, z - .04, z + .04, OPEN / 2 - .03, BRICK_D)
        gb(0, -D / 2 + .02, -D / 2 + .06, 0, PH - .3, .05, P.INK)
    else:
        for sx in (-1, 1):                                          # opened inward, folded flat against the piers
            gb(sx * (OPEN / 2 - .06), -.42, .42, 0, PH - .35, .08, P.PLANK)
    sz = H + .02                                                    # upper storey
    yf = -(D - .15) / 2
    gb(0, yf, -yf, sz, sz + 1.3, W2 - .2, S.PLASTER)
    for sx in (-1.2, 0, 1.2):
        gb(sx, yf - .015, yf, sz + .35, sz + .95, .55, P.INK)
    for sx in (-2, -1, 0, 1, 2):
        gb(sx * .9, yf - .03, yf, sz, sz + 1.3, .09, P.LACQUER)
    gb(0, -D / 2, D / 2, sz - .03, sz + .03, W2, BRICK_D)
    if part != "first":
        G3.hip_roof(-W2 / 2 - .25, W2 / 2 + .25, -D / 2 - .3, D / 2 + .25, sz + 1.32, .85, G3.TILE_R, G3.TILE_D)
    return list(m.meshes)


def mirror():
    m.meshes = [(np.asarray(v)[:, [1, 0, 2]] * [-1, -1, 1], c) for v, c in m.meshes]


def crop_strip(im, pivot, side):
    """keep only the middle cell's 16 px run of a three-piece strip, centred on the cell's front face (it sits half the wall's thickness toward
    the camera: 8 px left of the pivot for _u, right for _v), so a tile owns its whole face and the merlon above it"""
    x0 = pivot[0] - 16 if side == "u" else pivot[0]
    return im.crop((x0, 0, x0 + 16, im.height)), (pivot[0] - x0, pivot[1])


def models():
    out = {}
    im = wall()
    out["swall_u"] = crop_strip(im, im.pivot, "u")
    mirror()
    im = P.draw((80, 130), (40, 112))
    out["swall_v"] = crop_strip(im, im.pivot, "v")
    for state, closed in (("closed", True), ("open", False)):
        parts = {p: gate(closed, p) for p in ("all", "first", "second")}
        for orient in ("u", "v"):
            ims = {}
            for k, meshes in parts.items():
                m.meshes = meshes
                if orient == "v":
                    mirror()
                ims[k] = P.draw((170, 210), (85, 170))
            for k, im in split_halves(ims).items():
                out["sgate_%s_%s_%s" % (state, k, orient)] = (im, ims["all"].pivot)
    return out


def split_halves(ims):
    """cut the whole gate's picture into the two halves' pixels: where the second half (nearer the camera) has a pixel it owns it, the
    first half owns the rest of its own; so the halves join without an outline at the cut"""
    whole = ims["all"]
    A = np.asarray(ims["first"])[:, :, 3] > 0
    B = np.asarray(ims["second"])[:, :, 3] > 0
    res = {}
    for k, own in (("first", A & ~B), ("second", B)):
        arr = np.asarray(whole).copy()
        arr[~own] = 0
        im = Image.fromarray(arr)
        res[k] = im
    return res


def run():
    import gen_all
    for name, (im, pv) in models().items():
        t, p = G3.trimmed(im, pv)
        gen_all.save("tiles", name, t, p, "object", scale="scene", made="3D model + unit3d cel renderer")


if __name__ == "__main__":
    # preview: python3 gen_3d_castle.py OUT_DIR writes every model's trimmed picture there
    out = sys.argv[1]
    for name, (im, pv) in models().items():
        t, p = G3.trimmed(im, pv)
        print(name, t.size, p)
        t.save(os.path.join(out, "cs_" + name + ".png"))
