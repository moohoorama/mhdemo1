"""Candidates drawn the way gen_pixel_castle.py draws the scene castle: dot patterns laid on the faces of boxes on the
small-tile grid (x = 16(u - v), y = 8(u + v) - z), so every course and joint steps one pixel up per two across.

  battle walls  wall_clean_00..15: one small tile, the open sides (nw -u, ne -v, se +u, sw +v as bits 0..3) crenellated
  battle gate   gatehouse_{closed,open}_{u,v}_{0..3}: one game cell (2 x 2 small tiles) in a two-tile-thick wall line,
                a walled courtyard with a wide arch in each face under a slab on red posts, the leaves shut or swung
                back (the open one also stands for a pass), cut by owning small tile (index du + 2 dv)
  room walls    wall_{plaster,lacquer,canvas}_{nw,ne}: one game-cell edge, anchored at the cell's back corner
  room floors   floor_{wood,carpet,palace,tent}: one game cell, a 64 x 32 diamond, one dot per pixel
  battle sites  fence_00..15 (mask bits as the walls'), village, camp, burnt_village, burnt_camp: one game cell at
                the map's scale, pivot at its centre; village and camp take the place of sites.png's pictures
  scene bridge  scene_bridge_u: the 계교 bridge of S_gongsun_camp, 8 small tiles along u (six of water, one onto each
                bank) by 4 across, one picture anchored at the centre of its tile (3, 1) from the back corner

  python3 gen_pixel_kit.py --compare DIR   # old and new pieces, a battle castle corner, four rooms and a patch of
                                           # fences, villages and camps, into DIR
  python3 gen_pixel_kit.py --write         # save the kit into ../tiles and tiles.json (then ver5/tools/build_extra.py)
"""
import json
import os
import sys

import numpy as np
from PIL import Image, ImageDraw

import gen_pixel_castle as pc
from gen_pixel_castle import Body, box, brick, render, piece
from common import prop, site, sheet, INDEX, GAME

pc.PAL.update({
    # battle stone: the battle map's blue-grey, lit +v face, shaded +u face, top
    "bS": (184, 198, 207), "bs": (132, 147, 159), "bR": (124, 139, 152), "br": (86, 99, 112), "bU": (160, 174, 184),
    # gate: door leaves and their studs, lacquered posts, the slab over the arch
    "bD": (150, 98, 56), "bd": (104, 64, 36), "bi": (176, 182, 194),
    "bQ": (168, 44, 36), "bq": (118, 30, 28), "bK": (52, 58, 70), "bk": (40, 44, 54),
    # room walls
    "pL": (226, 212, 180), "pl": (182, 168, 138), "pt": (210, 196, 164), "pT": (168, 154, 126),
    "wW": (128, 82, 46), "ww": (92, 58, 34), "wC": (150, 104, 64),
    "qR": (164, 40, 34), "qr": (120, 28, 26), "qD": (104, 24, 22), "qd": (76, 18, 18), "qG": (214, 170, 72), "qg": (160, 120, 48),
    "cL": (214, 198, 160), "cl": (170, 154, 120), "cS": (190, 174, 136), "cs": (144, 130, 100), "cr": (120, 96, 64),
})

BH = 16                  # battle wall face height (the old wall cells' face)
BM = (2, 5)              # battle parapet strip and merlon heights
BEDGE = .16
RH = 34                  # room wall height (the old panels')
RT = .12                 # room wall thickness, small tiles


# ---------------------------------------------------------------- battle walls

def bstone(f, u, v, z, s, h):
    if f == "+z":   # the walkway paved in slabs half a tile square, every other row shifted
        i, j = np.floor(16 * u + 1e-6).astype(int), np.floor(16 * v + 1e-6).astype(int)
        return np.where((np.mod(j, 8) == 7) | (np.mod(i + 4 * (np.floor_divide(j, 8) % 2), 8) == 7), "bs", "bU")
    return brick(s, h, "bS", "bs", w=8, rows=3) if pc.lit(f) else brick(s, h, "bR", "br", w=8, rows=3)


def bmerlon(f, u, v, z, s, h):
    return np.full(u.shape, "bU" if f == "+z" else "bS" if pc.lit(f) else "bR")


def bblock(u=0, v=0, tile=None, h=BH):
    return Body(box(u, u + 1, v, v + 1, 0, h), bstone, tile)


def bedge(side, u=0, v=0, tile=None):
    """the battle parapet on one side of tile (u, v): a strip and two merlons, the first at the side's low corner"""
    if side[1] == "u":
        a0, a1 = (u + 1 - BEDGE, u + 1) if side[0] == "+" else (u, u + BEDGE)
        spans = lambda k0, k1, z1: box(a0, a1, v + k0, v + k1, BH, z1)
    else:
        a0, a1 = (v + 1 - BEDGE, v + 1) if side[0] == "+" else (v, v + BEDGE)
        spans = lambda k0, k1, z1: box(u + k0, u + k1, a0, a1, BH, z1)
    out = [Body(spans(0, 1, BH + BM[0]), bmerlon, tile)]
    out += [Body(spans(k, k + .25, BH + BM[1]), bmerlon, tile) for k in (0, .5)]
    return out


SIDES = (("-u", (-1, 0)), ("-v", (0, -1)), ("+u", (1, 0)), ("+v", (0, 1)))   # the runtime's mask bits 0..3
BB = (-40, -48, 40, 24)


def battle_wall(mask):
    """the wall tile whose sides in mask are open: parapets there, and the wall tiles beside it on the closed sides
    drawn as context so the stone runs on without an outline"""
    bodies = [bblock(tile=0)]
    for i, (side, (du, dv)) in enumerate(SIDES):
        if mask >> i & 1:
            bodies += bedge(side, tile=0)
        else:
            bodies.append(bblock(du, dv, -1))
    for (i, (du, _)), (k, (_, dv)) in (((0, SIDES[0][1]), (1, SIDES[1][1])), ((0, SIDES[0][1]), (3, SIDES[3][1])),
                                       ((2, SIDES[2][1]), (1, SIDES[1][1])), ((2, SIDES[2][1]), (3, SIDES[3][1]))):
        if not (mask >> i & 1 or mask >> k & 1):   # the diagonal tile between two closed sides, so no dot of ink at the corner
            bodies.append(bblock(du, dv, -1))
    a, own, o = render(bodies, bounds=BB)
    return piece(a, own, o, 0, (0, 8))


def bdoor(f, u, v, z, s, h):
    c = np.where(s % 4 == 3, "bd", "bD")
    return np.where((h % 5 == 2) & (s % 4 == 1), "bi", c)


def cut(L, out):
    """the gate cell's bodies rendered with the line's wall tiles on both ends as context and cut in two at the cell's
    centre line (u + v = 2): what lies behind a unit standing in the gate goes to the back small tile (0), what lies
    in front to the front one (3), so the game's depth sort draws the unit between them. The side tiles (1, 2) sort
    level with the unit and stay empty."""
    for b in out:
        b.tile = None
    ctx = [Body(L.box(a, a + 1, c, c + 1, 0, BH), bstone, -1) for a in (-1, 2) for c in (0, 1)]
    a, own, o = render(out + ctx, along=lambda u, v: np.where(u + v < 2, 0, 3), bounds=(-72, -64, 72, 48))
    pieces = [piece(a, own, o, k, (16 * ((k % 2) - (k // 2)), 8 * ((k % 2) + (k // 2) + 1))) for k in range(4)]
    return [p or (Image.new("RGBA", (1, 1)), (0, 0)) for p in pieces]


GW = .36                 # the gate's side walls, along the line
GT = .28                 # its gate walls, across the line


def crenels(L, a0, a1, c0, c1):
    """a parapet strip on top of the wall over [a0,a1]x[c0,c1] with merlons every half tile along its long side"""
    out = [Body(L.box(a0, a1, c0, c1, BH, BH + BM[0]), bmerlon)]
    if a1 - a0 >= c1 - c0:
        out += [Body(L.box(k, min(k + .25, a1), c0, c1, BH, BH + BM[1]), bmerlon) for k in np.arange(a0, a1 - .1, .5)]
    else:
        out += [Body(L.box(a0, a1, k, min(k + .25, c1), BH, BH + BM[1]), bmerlon) for k in np.arange(c0, c1 - .1, .5)]
    return out


def slab(f, u, v, z, s, h):
    return np.full(u.shape, "bK" if f == "+z" or pc.lit(f) else "bk")


def lacquer_post(f, u, v, z, s, h):
    return np.full(u.shape, "bQ" if pc.lit(f) else "bq")


GPASS = 13               # the gate arch's height


def gate_wall(L, c0, c1, closed):
    """a wall across the passage over [GW, 2 - GW] x [c0, c1], an arch nearly as wide as the courtyard through it, under
    a dark slab on four red lacquered posts standing on the wall. Shut, the two leaves meet in the arch; open, they
    are swung back flat against the side walls' courtyard faces."""
    o0, o1 = GW + .06, 2 - GW - .06
    out = [Body(L.box(GW, o0, c0, c1, 0, BH), bstone), Body(L.box(o1, 2 - GW, c0, c1, 0, BH), bstone),
           Body(L.box(o0, o1, c0, c1, GPASS, BH), bstone)]
    for w, dz in ((.1, 3), (.05, 6)):
        out += [Body(L.box(o0, o0 + w, c0, c1, GPASS - dz, GPASS), bstone), Body(L.box(o1 - w, o1, c0, c1, GPASS - dz, GPASS), bstone)]
    half = (o1 - o0) / 2
    if closed:
        d = (c0 + c1) / 2
        out += [Body(L.box(o0, o0 + half, d - .04, d + .04, 0, GPASS - 1), bdoor), Body(L.box(o0 + half, o1, d - .04, d + .04, 0, GPASS - 1), bdoor)]
    else:
        d0, d1 = (c1, c1 + half) if c0 == 0 else (c0 - half, c0)
        out += [Body(L.box(GW, GW + .05, d0, d1, 0, GPASS - 1), bdoor), Body(L.box(2 - GW - .05, 2 - GW, d0, d1, 0, GPASS - 1), bdoor)]
    for a in (o0 - .02, o1 - .06):
        for cp in (c0 + .02, c1 - .08):
            out.append(Body(L.box(a, a + .08, cp, cp + .06, BH, BH + 7), lacquer_post))
    out.append(Body(L.box(o0 - .1, o1 + .1, c0 - .08, c1 + .08, BH + 7, BH + 9), slab))
    return out


def battle_gate(orient, closed):
    """a game cell of gate in a wall line along `orient` (the line two tiles thick across it), built as a walled
    courtyard: side walls along the line's ends, an outer and an inner gate wall across the passage with their arches
    face to face, the courtyard open to the sky over the castle floor the game draws"""
    L = pc.Line(orient)
    out = [Body(L.box(0, GW, 0, 2, 0, BH), bstone), Body(L.box(2 - GW, 2, 0, 2, 0, BH), bstone)]
    out += crenels(L, GW - BEDGE, GW, 0, 2) + crenels(L, 2 - GW, 2 - GW + BEDGE, 0, 2)
    for c0, c1 in ((0, GT), (2 - GT, 2)):
        out += gate_wall(L, c0, c1, closed)
    return cut(L, out)


# ---------------------------------------------------------------- room walls

def plaster(f, u, v, z, s, h):
    """whitewashed wattle between timber: posts at the ends and the middle, a sill and a head beam, a few dots of
    texture"""
    if f == "+z":
        return np.full(u.shape, "wC")
    lit = pc.lit(f)
    wood, face, dot = ("wW", "pL", "pt") if lit else ("ww", "pl", "pT")
    sm = np.mod(s, 32)
    c = np.where(((s * 7 + h * 3) % 23 == 0), dot, face)
    c = np.where((h < 3) | (h >= RH - 4) | (sm < 2) | ((sm >= 15) & (sm < 17)) | (h == RH - 13) & (sm > 1), wood, c)
    return c


def lacquer(f, u, v, z, s, h):
    """red lacquer boards, dark posts, a gold band under the head beam and over the sill"""
    if f == "+z":
        return np.full(u.shape, "qD")
    lit = pc.lit(f)
    red, seam, post, gold = ("qR", "qr", "qD", "qG") if lit else ("qr", "qd", "qd", "qg")
    sm = np.mod(s, 32)
    c = np.where(np.mod(s, 4) == 3, seam, red)
    c = np.where((h == RH - 7) | (h == 4) | (h == 5), gold, c)
    c = np.where((h < 3) | (h >= RH - 4) | (sm < 2) | ((sm >= 15) & (sm < 17)), post, c)
    return c


def canvas(f, u, v, z, s, h):
    """tent canvas in widths sewn together, a rope down every width and the hem dark"""
    if f == "+z":
        return np.full(u.shape, "cr")
    lit = pc.lit(f)
    cloth, seam = ("cL", "cS") if lit else ("cl", "cs")
    sm = np.mod(s, 11)
    c = np.where(sm == 10, seam, cloth)
    c = np.where((sm == 5) & (h > 2), "cr", c)
    return np.where(h < 2, "cr", c)


ROOM = {"plaster": plaster, "lacquer": lacquer, "canvas": canvas}


def room_wall(style, side):
    """the wall on one back edge of a game cell, outside the cell, anchored at the cell's back corner (canvas 0, 0)"""
    body = box(-RT, 0, 0, 2, 0, RH) if side == "nw" else box(0, 2, -RT, 0, 0, RH)
    a, own, o = render([Body(body, ROOM[style], 0)], bounds=(-40, -44, 40, 24))
    return piece(a, own, o, 0, (0, 0))


# ---------------------------------------------------------------- room floors

FLOOR = {
    # a palette key per dot, from (i, j): dots along u and along v from the cell's back corner, 32 each
    "wood": lambda i, j: np.where((j % 8 == 7) | ((i + 11 * (j // 8)) % 24 == 23), "fJ",
                                  np.where(j % 8 == 0, "fH", np.where((j // 8) % 2, "fA", "fB"))),
    "carpet": lambda i, j: (lambda e: np.where(e < 2, "kB", np.where(e < 4, "kG", np.where(e < 5, "kB",
                                  np.where(((i // 4) + (j // 4)) % 2, "kR", "kr")))))(np.minimum(np.minimum(i, 31 - i), np.minimum(j, 31 - j))),
    "palace": lambda i, j: np.where((i % 16 == 15) | (j % 16 == 15), "tJ", np.where((i % 16 == 0) | (j % 16 == 0), "tH",
                                  np.where(((i // 16) + (j // 16)) % 2, "tA", "tB"))),
    "tent": lambda i, j: (lambda e: np.where(e < 5, np.where((i * 5 + j * 3) % 13 == 0, "eD", "eA"),
                                  np.where((i + j) % 4 < 2, "mA", "mB")))(np.minimum(np.minimum(i, 31 - i), np.minimum(j, 31 - j))),
}
pc.PAL.update({
    "fA": (150, 92, 54), "fB": (136, 82, 48), "fH": (168, 108, 64), "fJ": (82, 48, 30),
    "kR": (150, 30, 40), "kr": (128, 24, 34), "kB": (96, 14, 26), "kG": (206, 160, 70),
    "tA": (70, 72, 80), "tB": (60, 62, 70), "tH": (92, 94, 104), "tJ": (34, 36, 44),
    "eA": (138, 112, 78), "eD": (116, 92, 62), "mA": (196, 170, 112), "mB": (170, 144, 92),
})


def floor(style):
    """one game cell: canvas x = 16(u - v), y = 8(u + v) with the cell's back corner at (0, 0); each pixel's centre
    names the dot under it"""
    X, Y = np.meshgrid(np.arange(-32, 32) + .5, np.arange(0, 32) + .5)
    u, v = X / 32 + Y / 16, Y / 16 - X / 32
    inside = (u >= 0) & (u < 2) & (v >= 0) & (v < 2)
    i, j = np.floor(16 * u).astype(int).clip(0, 31), np.floor(16 * v).astype(int).clip(0, 31)
    keys = FLOOR[style](i, j)
    a = np.zeros(X.shape + (4,), np.uint8)
    for k in set(keys[inside]):
        m = inside & (keys == k)
        a[m, :3] = pc.PAL[k]
        a[m, 3] = 255
    return Image.fromarray(a), (32, 16)


# ---------------------------------------------------------------- battle sites: fence, village, camp
#
# All three cover one game cell (2 x 2 small tiles) with its back corner at canvas (0, 0) and the pivot at the cell
# centre (0, 16), drawn at the map's own scale (a fence no longer doubled).

pc.PAL.update({
    # weathered wood: posts and rails lit/shaded, end grain, the dark knots
    "zW": (150, 104, 62), "zw": (104, 70, 42), "zE": (184, 146, 98), "zk": (76, 50, 32),
    # the packed earth of a yard, and a camp's trampled ground
    "yA": (170, 140, 96), "yB": (156, 126, 84), "yD": (138, 110, 72), "ys": (112, 88, 58),
    "yC": (160, 146, 110), "yc": (144, 130, 96),
    # village: plaster walls, timber, thatch rows lit/shaded/furrow, the ridge's bundle, grey tiled roof
    "vL": (224, 208, 172), "vl": (176, 160, 128), "vT": (110, 72, 42), "vt": (80, 52, 32), "vH": (52, 40, 32),
    "hA": (206, 170, 92), "hB": (178, 142, 70), "ha": (160, 126, 62), "hb": (132, 102, 50), "hR": (120, 92, 46),
    "gA": (112, 120, 134), "gB": (88, 96, 110), "ga": (82, 88, 102), "gb": (62, 68, 80), "gE": (150, 158, 170),
    # stone of the well, and its dark water
    "sW": (170, 168, 158), "sw": (124, 122, 114), "sU": (150, 148, 140), "sK": (36, 46, 56),
    # camp: tent canvas (reusing cL/cl/cS/cs/cr), the red band of a commander's tent, stake points, bundles
    "nR": (172, 48, 40), "nr": (124, 34, 30), "nD": (44, 34, 30),
    "bX": (186, 160, 112), "bx": (140, 118, 80),
    # village extras: sauce jars, the plot's crop, earth walls, washing
    "jA": (118, 70, 40), "ja": (84, 48, 30), "jT": (60, 36, 24), "jR": (150, 100, 62), "gV": (104, 140, 64), "gv": (82, 116, 50),
    "eW": (186, 150, 100), "ew": (150, 118, 76), "lB": (92, 120, 168), "lb": (66, 88, 128),
})
SITE_BB = (-50, -60, 50, 50)
SITE_PIVOT = (0, 16)


def wood(f, u, v, z, s, h):
    if f == "+z":
        return np.full(u.shape, "zE")
    a, b = ("zW", "zw") if pc.lit(f) else ("zw", "zk")
    return np.where((s * 5 + h * 3) % 17 == 0, b, a)


def gable(L, a0, a1, c0, c1, zb, s, z0=0):
    """a ridge roof (or a tent) over [a0,a1]x[c0,c1]: two slopes across the line rising s pixels a tile from zb, the
    ends standing upright"""
    return [L.plane(0, s, 1, zb + s * c1, "r+c"), L.plane(0, -s, 1, zb - s * c0, "r-c")] + L.box(a0, a1, c0, c1, z0, 1e9)


# ---- fence: a post at the cell's centre, a rail fence out to the middle of each joined side

FH, FRAIL = 15, ((5, 7), (10, 12))
FP, FR = .07, .035       # post and rail half thickness, small tiles


def fence_arm(i, tile, du=0, dv=0):
    """the arm toward side i (the runtime's mask bit: -u, -v, +u, +v) from the cell centre: two rails and a post at
    the side's middle (shared with the next cell's arm), shifted by (du, dv) cells for context"""
    side, _ = SIDES[i]
    a0, a1 = (0, 1) if side[0] == "-" else (1, 2)
    e = 0 if side[0] == "-" else 2
    def b(a0, a1, c0, c1, z0, z1):
        return box(a0 + 2 * du, a1 + 2 * du, c0 + 2 * dv, c1 + 2 * dv, z0, z1) if side[1] == "u" else \
            box(c0 + 2 * du, c1 + 2 * du, a0 + 2 * dv, a1 + 2 * dv, z0, z1)
    out = [Body(b(a0, a1, 1 - FR, 1 + FR, z0, z1), wood, tile) for z0, z1 in FRAIL]
    out.append(Body(b(e - FP, e + FP, 1 - FP, 1 + FP, 0, FH), wood, tile))
    return out


def fence(mask):
    bodies = [Body(box(1 - FP, 1 + FP, 1 - FP, 1 + FP, 0, FH + 1), wood, 0)]
    for i, (_, (du, dv)) in enumerate(SIDES):
        if mask >> i & 1:
            bodies += fence_arm(i, 0)
            bodies += fence_arm((i + 2) % 4, -1, du, dv)   # the next cell's arm back toward this one
    a, own, o = render(bodies, bounds=SITE_BB)
    return piece(a, own, o, 0, SITE_PIVOT)


# ---- village: two houses at the back, a haystack and a well at the sides, a wattle fence along the front, all on
# a yard of packed earth; the cell's centre stays open for the unit standing there

def yard(f, u, v, z, s, h):
    if f == "+z":
        i, j = np.floor(16 * u).astype(int), np.floor(16 * v).astype(int)
        return np.where((i * 7 + j * 13) % 29 == 0, "yD", np.where((i * 3 + j * 5) % 11 == 0, "yB", "yA"))
    return np.full(u.shape, "ys")


def camp_ground(f, u, v, z, s, h):
    if f == "+z":
        i, j = np.floor(16 * u).astype(int), np.floor(16 * v).astype(int)
        return np.where((i * 7 + j * 13) % 23 == 0, "yc", np.where((i + 2 * j) % 31 == 0, "yD", "yC"))
    return np.full(u.shape, "ys")


VH = 12                  # village wall height


def thatch(ze):
    """straw laid in rows: a darker furrow every third pixel up the slope, the eave's cut ends (below ze + 2) dark"""
    def p(f, u, v, z, s, h):
        if f == "+z":
            return np.full(u.shape, "hR")
        a, b = ("hA", "hB") if pc.lit(f) else ("ha", "hb")
        c = np.where(np.mod(h, 3) == 0, b, a)
        c = np.where((s * 7 + h * 5) % 19 == 0, b, c)
        return np.where(z < ze + 2, "hR", c)
    return p


def tiles_roof(ze):
    """grey tile rows down the slope, a furrow every fourth dot, round tile ends along the eave"""
    def p(f, u, v, z, s, h):
        if f == "+z":
            return np.full(u.shape, "gb")
        a, b, eave = ("gA", "gB", "gE") if pc.lit(f) else ("ga", "gb", "gB")
        c = np.where(np.mod(s, 4) == 3, b, a)
        return np.where(z < ze + 2, np.where(np.mod(s, 4) == 3, "gb", eave), c)
    return p


def ridge_bundle(f, u, v, z, s, h):
    return np.full(u.shape, "hR" if f == "+z" or pc.lit(f) else "vt")


def well_stone(f, u, v, z, s, h):
    if f == "+z":
        return np.full(u.shape, "sU")
    return brick(s, h, "sW", "sw", w=4, rows=2) if pc.lit(f) else brick(s, h, "sw", "ys", w=4, rows=2)


EAVE = .08               # how far a roof overhangs its walls


def house(L, a0, a1, c0, c1, roof, burnt, wall=VH, rise=26, hip=False, door=(), window=()):
    """a house over [a0,a1]x[c0,c1] along L: plaster between timber posts every eight dots, a sill and a beam under
    the eave, a roof of thatch or tiles (gabled across the line, or hipped). Openings are (face, t0, t1) with t the
    place along the face: u on a +v face, v on a +u face. Burnt, the walls stand half high and the roof is gone."""
    def walls(f, u, v, z, s, h):
        if f == "+z":
            return np.full(u.shape, "vT")
        face, timber = ("vL", "vT") if pc.lit(f) else ("vl", "vt")
        c = np.where((np.mod(s, 8) == 0) | (h <= 1) | (h >= wall - 1), timber, face)
        at = u if f.endswith("v") else v
        for fc, t0, t1 in door:
            c = np.where((f == fc) & (at >= t0) & (at < t1) & (h >= 2) & (h < 9), "vH", c)
        for fc, t0, t1 in window:
            c = np.where((f == fc) & (at >= t0) & (at < t1) & (h >= 5) & (h < 8), np.where(np.mod(s, 2) == 0, "vt", "vH"), c)
        return c
    if burnt:
        return ruin(L, a0, a1, c0, c1, wall, walls)
    out = [Body(L.box(a0, a1, c0, c1, 0, wall), walls)]
    paint = (thatch if roof == "thatch" else tiles_roof)(wall - 1)
    r = (a0 - EAVE, a1 + EAVE, c0 - EAVE, c1 + EAVE)
    if hip:
        return out + [Body(L.hip(*r, wall - 1, rise), paint)]
    out.append(Body(gable(L, *r, wall - 1, rise, z0=wall - 1), paint))
    if roof == "thatch":   # the straw bundle along the ridge
        cm, zr = (c0 + c1) / 2, wall - 1 + rise * ((c1 - c0) / 2 + EAVE)
        out.append(Body(L.box(r[0], r[1], cm - .03, cm + .03, zr - 1, zr + 2), ridge_bundle))
    return out


def ruin(L, a0, a1, c0, c1, wall, paint, t=.07):
    """what is left of a burnt house: its four walls a thin shell broken to uneven heights round a floor of ash"""
    rnd = np.random.default_rng(int(1000 * (a0 + 3 * c0)))
    out = [Body(L.box(a0, a1, c0, c1, 0, 1), ash)]
    for (p0, p1, q0, q1) in ((a0, a1, c0, c0 + t), (a0, a1, c1 - t, c1), (a0, a0 + t, c0, c1), (a1 - t, a1, c0, c1)):
        along = p1 - p0 > q1 - q0
        n = max(1, int(round(((p1 - p0) if along else (q1 - q0)) / .2)))
        for k in range(n):
            hgt = int(rnd.integers(2, max(3, int(wall * .7))))
            if along:
                out.append(Body(L.box(p0 + (p1 - p0) * k / n, p0 + (p1 - p0) * (k + 1) / n, q0, q1, 0, hgt), paint))
            else:
                out.append(Body(L.box(p0, p1, q0 + (q1 - q0) * k / n, q0 + (q1 - q0) * (k + 1) / n, 0, hgt), paint))
    return out


def ash(f, u, v, z, s, h):
    return np.full(u.shape, "yD" if f == "+z" else "ys")


def chimneys(points, burnt):
    return [] if burnt else [Body(box(u - .05, u + .05, v - .05, v + .05, VH, zt), well_stone) for (u, v), zt in points]


def village_0(burnt):
    """a thatched house at the back left, a tiled one at the back right, a haystack, a well, a rail fence in front"""
    out = house(pc.Line("v"), .2, 1.15, .18, .78, "thatch", burnt, door=[("+v", .375, .625)],
                window=[("+u", .75, 1), ("+u", .3125, .5)])
    out += house(pc.Line("u"), 1.05, 1.85, .18, .72, "tiles", burnt, rise=30, door=[("+u", .3125, .5625)],
                 window=[("+v", 1.25, 1.4375), ("+v", 1.5625, 1.75)])
    out += chimneys(SMOKE[0], burnt)
    if not burnt:
        out.append(Body(box(.3, .66, 1.38, 1.74, 0, 5), hay))
        out.append(Body(pc.Line("u").hip(.3, .66, 1.38, 1.74, 4, 44), hay))
    out.append(Body(box(1.5, 1.82, .88, 1.2, 0, 5), well_top))
    for a in (1.1, 1.5, 1.9):
        out.append(Body(box(a - .05, a + .05, 1.85, 1.95, 0, 8), wood))
    for z0 in (3, 6):
        out.append(Body(box(1.1, 1.9, 1.88, 1.92, z0, z0 + 1), wood))
    return out


def village_1(burnt):
    """two thatched houses meeting in an L at the back, a terrace of sauce jars at the right, a vegetable plot in
    front at the left"""
    out = house(pc.Line("v"), .2, 1.2, .15, .7, "thatch", burnt, door=[("+v", .35, .5)],
                window=[("+u", .4, .6), ("+u", .85, 1.02)])
    out += house(pc.Line("u"), .9, 1.8, .15, .7, "thatch", burnt, door=[("+u", .35, .5)],
                 window=[("+v", 1.05, 1.2), ("+v", 1.45, 1.6)])
    out += chimneys(SMOKE[1], burnt)
    out.append(Body(box(1.4, 1.88, .92, 1.34, 0, 2), well_stone))
    if not burnt:
        for (u, v), hgt in (((1.52, 1.02), 6), ((1.76, 1.04), 5), ((1.62, 1.24), 6)):
            out.append(Body(box(u - .09, u + .09, v - .09, v + .09, 2, 2 + hgt), jar))
    out.append(Body(box(.22, .86, 1.3, 1.86, 0, 2), plot(burnt)))
    return out


def village_2(burnt):
    """a rich man's tiled house across the back, a tiled storehouse at the left, low earth walls with a tiled cap
    along the front sides"""
    out = house(pc.Line("u"), .2, 1.55, .15, .8, "tiles", burnt, wall=14, rise=30, door=[("+v", .82, 1.0)],
                window=[("+v", .4, .55), ("+v", 1.2, 1.35), ("+u", .4, .55)])
    out += house(pc.Line("v"), 1.15, 1.7, .2, .7, "tiles", burnt, wall=9, rise=30, door=[("+u", 1.32, 1.5)])
    out += chimneys(SMOKE[2], burnt)
    for b in (box(.95, 1.5, 1.86, 1.94, 0, 7), box(1.86, 1.94, .25, 1.0, 0, 7)):
        out.append(Body(b, earth_wall))
    if not burnt:
        out += [Body(box(.93, 1.52, 1.84, 1.96, 7, 9), cap), Body(box(1.84, 1.96, .23, 1.02, 7, 9), cap)]
    return out


def village_3(burnt):
    """three small round-roofed huts, a woodpile, a line of washing"""
    out = []
    for (u0, u1, v0, v1), door in (((.15, .6, .15, .6), ("+v", .3, .42)), ((.15, .6, 1.1, 1.55), ("+u", 1.26, 1.38)),
                                   ((1.1, 1.55, .15, .6), ("+v", 1.25, 1.37))):
        out += house(pc.Line("u"), u0, u1, v0, v1, "thatch", burnt, wall=9, rise=34, hip=True, door=[door])
    out.append(Body(box(1.45, 1.85, 1.15, 1.4, 0, 6 if not burnt else 3), logs))
    if not burnt:
        out += [Body(box(1.1, 1.16, 1.7, 1.76, 0, 13), wood), Body(box(1.74, 1.8, 1.7, 1.76, 0, 13), wood),
                Body(box(1.16, 1.74, 1.72, 1.74, 12, 13), wood),
                Body(box(1.22, 1.38, 1.725, 1.735, 6, 12), cloth("cL", "cl")),
                Body(box(1.46, 1.62, 1.725, 1.735, 7, 12), cloth("lB", "lb"))]
    return out


VILLAGES = (village_0, village_1, village_2, village_3)


def village(n):
    """village n on its yard of packed earth"""
    return lambda burnt=False: [Body(box(.04, 1.96, .04, 1.96, 0, 1), yard)] + VILLAGES[n](burnt)
SMOKE = (   # (u, v) and height of each smoke source, per village: chimney tops, or a hut roof's smoke hole
    [((.45, 1.05), 23), ((1.55, .25), 22)],
    [((.42, .95), 23), ((1.6, .4), 23)],
    [((1.35, .3), 26)],
    [((.375, .375), 19), ((1.325, .375), 19)],
)


def jar(f, u, v, z, s, h):
    """a glazed sauce jar: a pale rim on top, the glaze darker toward the foot"""
    if f == "+z":
        return np.where((np.floor(16 * u) + np.floor(16 * v)) % 3 == 0, "jT", "jR")
    return np.where(h <= 3, "ja" if pc.lit(f) else "jT", "jA" if pc.lit(f) else "ja")


def plot(burnt):
    """furrows along u, green where the crop stands (bare when burnt)"""
    def p(f, u, v, z, s, h):
        if f != "+z":
            return np.full(u.shape, "ys")
        j = np.floor(16 * v).astype(int)
        return np.where(j % 3 == 0, "yD" if burnt else np.where(np.floor(16 * u).astype(int) % 2 == 0, "gV", "gv"), "yB")
    return p


def earth_wall(f, u, v, z, s, h):
    if f == "+z":
        return np.full(u.shape, "eW")
    a, b = ("eW", "ew") if pc.lit(f) else ("ew", "ys")
    return np.where((s * 5 + h * 7) % 13 == 0, b, a)


def cap(f, u, v, z, s, h):
    return np.full(u.shape, "gA" if f == "+z" else "gB" if pc.lit(f) else "gb")


def logs(f, u, v, z, s, h):
    """a woodpile: bark along it, the cut ends ringed on its +u end"""
    if f == "+z":
        return np.full(u.shape, "zw")
    if f == "+u":
        return np.where((np.mod(s, 3) == 0) | (h % 3 == 0), "zk", "zE")
    return np.where(h % 3 == 0, "zk", "zw" if pc.lit(f) else "zk")


def cloth(a, b):
    return lambda f, u, v, z, s, h: np.full(u.shape, a if pc.lit(f) or f == "+z" else b)


def hay(f, u, v, z, s, h):
    """a haystack: straw in wisps, the low skirt shaded"""
    if f == "+z":
        return np.full(u.shape, "hA")
    a, b = ("hA", "hB") if pc.lit(f[-2:]) else ("ha", "hb")
    return np.where((s * 3 + h * 7) % 5 == 0, b, a)


def well_top(f, u, v, z, s, h):
    if f == "+z":
        return np.where((u > 1.58) & (u < 1.74) & (v > .96) & (v < 1.12), "sK", "sU")
    return brick(s, h, "sW", "sw", w=4, rows=2) if pc.lit(f) else brick(s, h, "sw", "ys", w=4, rows=2)


# ---- camp: a palisade of sharpened stakes round the cell with an opening on the front right, a commander's tent at
# the back, a smaller one at the right, bundles of supplies at the left, the flagpole at the back corner (the game
# hangs the flag on it)

STAKE = .11
PH = 8                  # palisade height to the points' base


def stake(f, u, v, z, s, h):
    if f.startswith("r"):
        return np.full(u.shape, "zE" if pc.lit(f[1:]) else "zw")
    if f == "+z":
        return np.full(u.shape, "zE")
    a, b = ("zW", "zw") if pc.lit(f) else ("zw", "zk")
    return np.where((h == 4) | (h == 10), "zk", np.where(np.mod(s, 2) == 0, a, b))


def canvas_tent(band):
    """tent canvas with sewn widths, the gable ends dark at the flap, a red band low on a commander's tent"""
    def p(f, u, v, z, s, h):
        if f.startswith("r"):
            lit = pc.lit(f[1:])
            cloth, seam = ("cL", "cS") if lit else ("cl", "cs")
            c = np.where(np.mod(s, 6) == 5, seam, cloth)
            if band:
                c = np.where((h >= 3) & (h < 5), "nR" if lit else "nr", c)
            return np.where(h < 2, "cr", c)
        lit = pc.lit(f)
        c = np.full(u.shape, "cL" if lit else "cl")
        return c
    return p


def flap(f, u, v, z, s, h):
    return np.full(u.shape, "nD")


def bundle(f, u, v, z, s, h):
    if f == "+z":
        return np.where(np.mod(np.floor(16 * u).astype(int), 4) == 0, "bx", "bX")
    a, b = ("bX", "bx") if pc.lit(f) else ("bx", "zw")
    return np.where((np.mod(s, 5) == 2) | (h % 4 == 3), "zk" if not pc.lit(f) else b, a)


def palisade(burnt=False):
    """stakes STAKE wide along the cell's four sides, inset a little, a gap on the front right (+u) side"""
    out = []
    rnd = np.random.default_rng(7)
    e0, e1 = .06, 1.94
    n = int(round((e1 - e0) / STAKE))
    for side in ("-u", "-v", "+u", "+v"):
        for k in range(n):
            a = e0 + k * STAKE
            if side == "+u" and .7 < a < 1.2:
                continue
            top = PH if not burnt else int(rnd.integers(2, 7))
            c = (e0, e0 + STAKE) if side[0] == "-" else (e1 - STAKE, e1)
            L = pc.Line("v" if side[1] == "u" else "u")
            out.append(Body(L.box(a, a + STAKE, *c, 0, top), stake))
            if not burnt:
                out.append(Body(L.hip(a, a + STAKE, *c, top, 70), stake))
    return out


def camp(burnt=False):
    out = [Body(box(.04, 1.96, .04, 1.96, 0, 1), camp_ground)] + palisade(burnt)
    U, V = pc.Line("u"), pc.Line("v")
    if not burnt:
        # the commander's tent: ridge along u, at the back left of centre
        out.append(Body(gable(U, .45, 1.25, .3, .9, 1, 52), canvas_tent(True)))
        out.append(Body(U.box(1.25, 1.26, .48, .72, 1, 12), flap))
        # a soldiers' tent at the right, ridge along v
        out.append(Body(gable(V, .35, .95, 1.3, 1.75, 1, 50), canvas_tent(False)))
        out.append(Body(V.box(.95, .96, 1.43, 1.62, 1, 8), flap))
        out.append(Body(box(.2, .25, .2, .25, 0, 30), wood))   # the flagpole
    # supplies at the left: sacks and a bundle of spears laid flat
    out += [Body(box(.3, .55, 1.35, 1.6, 1, 6), bundle), Body(box(.3, .5, 1.62, 1.82, 1, 5), bundle)]
    if not burnt:
        out.append(Body(box(.32, .52, 1.4, 1.55, 6, 9), bundle))
    return out


FLAG_TOP = (0, -37)      # where the game hangs the camp flag, from the pivot (scenery.json camp overlay)


def site_piece(bodies):
    for b in bodies:
        b.tile = 0 if b.tile is None else b.tile
    a, own, o = render(bodies, bounds=SITE_BB)
    return piece(a, own, o, 0, SITE_PIVOT)


def smoke_points(n):
    """village n's smoke sources from the pivot, for the smoke overlays (gen_scenery.py)"""
    return [(int(16 * (u - v)) - SITE_PIVOT[0], int(8 * (u + v)) - zt - SITE_PIVOT[1]) for (u, v), zt in SMOKE[n]]


CHAR = [(34, 30, 32), (58, 50, 48), (84, 72, 62), (110, 92, 76)]


def charred(im):
    """a burnt site: fills to charcoal by brightness (warm where they were wood), ink kept, a few embers"""
    a = np.array(im).astype(int)
    l = (a[..., 0] * 30 + a[..., 1] * 59 + a[..., 2] * 11) / 100
    fill = (a[..., 3] > 0) & (l >= 40)
    c = np.array(CHAR)[np.digitize(l, (70, 110, 150))]
    c[..., 0] += np.where(a[..., 0] > a[..., 1] + 25, 22, 0)
    c[..., 1] += np.where(a[..., 0] > a[..., 1] + 25, 6, 0)
    a[fill, :3] = c[fill]
    ys, xs = np.nonzero(fill)
    for k in np.random.default_rng(3).choice(len(xs), 10, replace=False):
        a[ys[k], xs[k], :3] = ((255, 150, 40), (230, 80, 30), (255, 210, 90))[k % 3]
    return Image.fromarray(a.astype(np.uint8))


def sites():
    """name -> (image, pivot): fence_00..15, village, camp and their burnt forms"""
    out = {"fence_%02d" % m: fence(m) for m in range(16)}
    for n, f in [("village_%d" % i, village(i)) for i in range(len(VILLAGES))] + [("camp", camp)]:
        out[n] = site_piece(f(False))
        im, pv = site_piece(f(True))
        out["burnt_" + n] = (charred(im), pv)
    return out


# ---------------------------------------------------------------- scene bridge
#
# A plank deck on a stone abutment at each end, raised over the water on rows of piles with a cap beam across each
# row, a rail of posts along both sides. The water stays flat under it (the scene maps' water is drawn level with
# the ground), so the deck stands BZ pixels over it and ramps down onto each bank.

pc.PAL.update({
    "kA": (164, 124, 80), "kB": (150, 112, 70), "ka": (104, 74, 46),     # deck planks, two tones, the seams
})
BL, BW, BZ = 8, 4, 10    # length along u, width along v (small tiles), deck top (pixels)
BDECK = 3                # deck thickness
BRAMP = 7                # pixels the ramps rise a tile
BRAIL = (10, 11, 15, 17) # mid rail and top rail, pixels above the deck
BANCHOR = (3.5, 1.5)     # the anchor tile's centre, from the back corner


def planks(f, u, v, z, s, h):
    if f.startswith("+z") or f.startswith("r"):
        i = np.floor(16 * u + 1e-6).astype(int)
        j = np.floor(16 * v + 1e-6).astype(int)
        seam = (i % 5 == 4) | ((j + 7 * (i // 5)) % 23 == 0)
        return np.where(seam, "ka", np.where((i // 5) % 2 == 0, "kA", "kB"))
    return wood(f, u, v, z, s, h)


def abutment(f, u, v, z, s, h):
    if f == "+z" or f.startswith("r"):
        return np.full(u.shape, "sU")
    return brick(s, h, "sW", "sw", w=6, rows=2) if pc.lit(f) else brick(s, h, "sw", "zk", w=6, rows=2)


def scene_bridge():
    up = ((-BRAMP, 0, 1), 1, "r-u")                 # z <= 1 + BRAMP u: the near ramp
    down = ((BRAMP, 0, 1), 1 + BRAMP * BL, "r+u")   # z <= 1 + BRAMP (BL - u): the far ramp
    bodies = [Body(box(0, BL, .15, BW - .15, BZ - BDECK, BZ) + [up, down], planks, 0)]
    for a0, a1, ramp in ((0, 1.4, ((-BRAMP, 0, 1), 0, "r-u")), (BL - 1.4, BL, ((BRAMP, 0, 1), BRAMP * BL, "r+u"))):
        bodies.append(Body(box(a0, a1, .05, BW - .05, 0, BZ - BDECK) + [ramp], abutment, 0))
    for a in (2.6, 4, 5.4):                        # rows of piles, a cap beam across each
        bodies.append(Body(box(a - .09, a + .09, .1, BW - .1, BZ - BDECK - 1.5, BZ - BDECK), wood, 0))
        for c in (.3, 1.4, 2.6, BW - .3):
            bodies.append(Body(box(a - .07, a + .07, c - .07, c + .07, 0, BZ - BDECK), wood, 0))
    for c in (.22, BW - .22):                      # the rails: posts every 1.4 tiles, a mid and a top rail
        for a in (1.2, 2.6, 4, 5.4, BL - 1.2):
            bodies.append(Body(box(a - .07, a + .07, c - .07, c + .07, BZ, BZ + BRAIL[3] + 1), wood, 0))
        for z0, z1 in ((BRAIL[0], BRAIL[1]), (BRAIL[2], BRAIL[3])):
            bodies.append(Body(box(1.2, BL - 1.2, c - .04, c + .04, BZ + z0, BZ + z1), wood, 0))
    a, own, o = render(bodies)
    return piece(a, own, o, 0, (16 * (BANCHOR[0] - BANCHOR[1]), 8 * (BANCHOR[0] + BANCHOR[1])))


# ---------------------------------------------------------------- comparison

def old_prop(name):
    """a picture the game draws now (compare against it before --write and build_extra.py put the kit in the game), from props.png or extra.png (where the sheets overlap, extra.png wins as it does
    in the game)"""
    s = INDEX["extra"]["sprites"].get(name)
    if s is None:
        return prop(name)
    x, y, w, h = s["rect"]
    return sheet("extra").crop((x, y, x + w, y + h)), tuple(s["pivot"])


GRASS, GRID = (118, 136, 86, 255), (104, 120, 76)


def ground(s, ox, oy, cells):
    d = ImageDraw.Draw(s)
    for (U, V) in cells:
        cx, cy = ox + 16 * (U - V), oy + 8 * (U + V)
        d.polygon([(cx, cy), (cx + 16, cy + 8), (cx, cy + 16), (cx - 16, cy + 8)], GRASS, GRID)


def battle_scene(new):
    """a castle corner on the half-cell lattice: two-tile-thick lines along u and v, a gate cell in each, drawn like
    the runtime does (small tiles by u + v, each its wall picture by open sides or its gate piece)"""
    walls = {(u, v) for u in range(0, 14) for v in (0, 1)} | {(u, v) for u in (0, 1) for v in range(0, 12)}
    gates = {}
    for du in (0, 1):
        for dv in (0, 1):
            gates[(6 + du, dv)] = ("u", du + 2 * dv)
            gates[(du, 6 + dv)] = ("v", du + 2 * dv)
    lo, hi = -2, 16
    W, H = 32 * (hi - lo) + 80, 16 * (hi - lo) + 120
    s = Image.new("RGBA", (W, H), (40, 44, 52, 255))
    ox, oy = W // 2 + 16 * (0), 80 - 8 * (2 * lo)
    ox = W // 2 - 16 * ((hi + lo) / 2 - (hi + lo) / 2)
    ground(s, ox, oy, [(u, v) for u in range(lo, hi) for v in range(lo, hi)])
    new_walls = {m: battle_wall(m) for m in range(16)}
    new_gates = {o: battle_gate(o, True) for o in ("u", "v")}
    for U, V in sorted(walls, key=lambda p: (p[0] + p[1], p[0] - p[1])):
        mask = sum(1 << i for i, (_, (du, dv)) in enumerate(SIDES) if (U + du, V + dv) not in walls)
        if (U, V) in gates:
            o, k = gates[(U, V)]
            im, pv = new_gates[o][k] if new else old_prop("gatehouse_closed_%s_%d" % (o, k))
        else:
            im, pv = new_walls[mask] if new else old_prop("wall_clean_%02d" % mask)
        s.alpha_composite(im, (int(ox + 16 * (U - V) - pv[0]), int(oy + 8 * (U + V + 1) - pv[1])))
    return s


ROOMS = (("house", "wood", "plaster", "pillar_wood"), ("palace", "palace", "lacquer", "pillar_red"),
         ("tent", "tent", "canvas", "pillar_red"), ("hall", "carpet", "lacquer", "pillar_red"))


def room_scene(floor_style, wall_style, pillar, new, n=3):
    """an n x n room of game cells, its two back edges walled, pillars at the corners (unchanged)"""
    W, H = 64 * n + 64, 32 * n + 96
    s = Image.new("RGBA", (W, H), (40, 44, 52, 255))
    ox, oy = W // 2, 56
    at = lambda U, V: (ox + 32 * (U - V), oy + 16 * (U + V))   # game-cell coordinates, cell (0, 0)'s back corner at 0, 0
    fl = floor(floor_style) if new else None
    for U in range(n):
        for V in range(n):
            im, pv = fl if new else old_prop("floor_" + floor_style)
            x, y = at(U + .5, V + .5)
            s.alpha_composite(im, (int(x - pv[0]), int(y - pv[1])))
    walls = [("nw", 0, V) for V in range(n)] + [("ne", U, 0) for U in range(n)]
    for side, U, V in sorted(walls, key=lambda w: w[1] + w[2]):
        im, pv = room_wall(wall_style, side) if new else old_prop("wall_%s_%s" % (wall_style, side))
        x, y = at(U, V)
        s.alpha_composite(im, (int(x - pv[0]), int(y - pv[1])))
    for U, V in ((0, 0), (n, 0), (0, n)):
        im, pv = old_prop(pillar)
        x, y = at(U, V)
        s.alpha_composite(im, (int(x - pv[0]), int(y - pv[1])))
    return s


def site_scene(new, burnt=False):
    """a patch of battle map, game cells 64 x 32: a fence line with a corner, a T and a crossing, a village and a camp
    beside it, drawn as the runtime does (sites at the map scale with their flag and smoke, old fences doubled)"""
    sc = json.load(open(os.path.join(GAME, "map", "scenery.json")))
    scs = sheet("scenery")
    def sp(name):
        x, y, w, h = sc["sprites"][name]["rect"]
        return scs.crop((x, y, x + w, y + h)), tuple(sc["sprites"][name]["pivot"])
    fences = {(x, 0) for x in range(0, 6)} | {(0, y) for y in range(0, 6)} | {(3, y) for y in range(-1, 3)} | {(x, 4) for x in range(0, 3)}
    sites_at = {(2, 2): "village", (4, 2): "camp", (2, 5): "camp"}
    sites_at.update({(u, v): "village" for u in (5, 6) for v in (3, 4, 5)})
    lo, hi = -2, 8
    W, H = 64 * (hi - lo) + 40, 32 * (hi - lo) + 110
    s = Image.new("RGBA", (W, H), (40, 44, 52, 255))
    ox, oy = W // 2, 90 - 32 * lo
    d = ImageDraw.Draw(s)
    for U in range(lo, hi):
        for V in range(lo, hi):
            cx, cy = ox + 32 * (U - V), oy + 16 * (U + V)
            d.polygon([(cx, cy - 16), (cx + 32, cy), (cx, cy + 16), (cx - 32, cy)], GRASS, GRID)
    kit_ = sites() if new else None
    cells = sorted(fences | set(sites_at), key=lambda p: (p[0] + p[1], p[0] - p[1]))
    for U, V in cells:
        px, py = ox + 32 * (U - V), oy + 16 * (U + V)
        if (U, V) in sites_at:
            name = sites_at[(U, V)]
            n = (U + 2 * V) % 4   # the runtime's pick of a village picture
            pic = "village_%d" % n if name == "village" and new else name
            if burnt:
                im, pv = kit_["burnt_" + pic] if new else old_prop("burnt_" + name)
            elif new:
                im, pv = kit_[pic]
            else:
                im, pv = site(name)
            s.alpha_composite(im, (px - pv[0], py - pv[1]))
            if burnt:
                continue
            if name == "camp":
                fl, _ = sp("camp_flag_1")
                s.alpha_composite(fl, (px + FLAG_TOP[0], py + FLAG_TOP[1]))
            else:
                pts = smoke_points(n) if new else [(13, -12), (-22, -13)]
                for i, (dx, dy) in enumerate(pts):
                    sm, spv = sp("smoke_%d" % (2 + 3 * i))
                    s.alpha_composite(sm, (px + dx - spv[0], py + dy - spv[1]))
            continue
        mask = sum(1 << i for i, (_, (du, dv)) in enumerate(SIDES) if (U + du, V + dv) in fences)
        if new:
            im, pv = kit_["fence_%02d" % mask]
            s.alpha_composite(im, (px - pv[0], py - pv[1]))
        else:
            im, pv = old_prop("fence_%02d" % mask)
            im = im.resize((im.width * 2, im.height * 2), Image.NEAREST)
            s.alpha_composite(im, (px - 2 * pv[0], py - 2 * pv[1]))
    return s


def compare(out):
    os.makedirs(out, exist_ok=True)
    zoom = lambda im, n: im.resize((im.width * n, im.height * n), Image.NEAREST)
    for new in (False, True):
        tag = "new" if new else "old"
        zoom(battle_scene(new), 2).save(os.path.join(out, "battle_%s.png" % tag))
        for name, fs, ws, pil in ROOMS:
            zoom(room_scene(fs, ws, pil, new), 3).save(os.path.join(out, "room_%s_%s.png" % (name, tag)))
        zoom(site_scene(new), 2).save(os.path.join(out, "sites_%s.png" % tag))
    zoom(site_scene(True, burnt=True), 2).save(os.path.join(out, "sites_burnt_new.png"))

    def strip(items, gap=6):
        w = sum(im.width for im in items) + gap * (len(items) + 1)
        h = max(im.height for im in items) + 2 * gap
        s = Image.new("RGBA", (w, h), (40, 44, 52, 255))
        x = gap
        for im in items:
            s.alpha_composite(im, (x, h - gap - im.height))
            x += im.width + gap
        return s

    zoom(strip([old_prop("wall_clean_%02d" % m)[0] for m in range(16)]), 4).save(os.path.join(out, "walls_old.png"))
    zoom(strip([battle_wall(m)[0] for m in range(16)]), 4).save(os.path.join(out, "walls_new.png"))
    zoom(strip([old_prop("gatehouse_closed_%s_%d" % (o, k))[0] for o in "uv" for k in range(4)]), 4).save(os.path.join(out, "gate_old.png"))
    zoom(strip([im for o in "uv" for im, _ in battle_gate(o, True) + battle_gate(o, False)]), 4).save(os.path.join(out, "gate_new.png"))
    zoom(strip([old_prop(f"wall_{s}_{d}")[0] for s in ROOM for d in ("nw", "ne")] + [old_prop("floor_" + f)[0] for f in FLOOR]), 4).save(os.path.join(out, "room_parts_old.png"))
    zoom(strip([room_wall(s, d)[0] for s in ROOM for d in ("nw", "ne")] + [floor(f)[0] for f in FLOOR]), 4).save(os.path.join(out, "room_parts_new.png"))


def kit():
    """name -> (image, pivot, tiles.json fields)"""
    out = {}
    for m in range(16):
        out["wall_clean_%02d" % m] = (*battle_wall(m), dict(layer="object", scale="battle", mask=m))
    for o in "uv":
        for k, (im, pv) in enumerate(battle_gate(o, True)):
            out["gatehouse_closed_%s_%d" % (o, k)] = (im, pv, dict(layer="object", scale="battle", blocks=True))
        for k, (im, pv) in enumerate(battle_gate(o, False)):
            out["gatehouse_open_%s_%d" % (o, k)] = (im, pv, dict(layer="object", scale="battle"))
    for style in ROOM:
        for side in ("nw", "ne"):
            out["wall_%s_%s" % (style, side)] = (*room_wall(style, side), dict(layer="object", scale="scene", shadow=False))
    for style in FLOOR:
        out["floor_" + style] = (*floor(style), dict(layer="ground", scale="scene"))
    for name, (im, pv) in sites().items():
        fields = dict(mask=int(name[-2:]), sides_bits=["nw", "ne", "se", "sw"]) if name.startswith("fence_") else {}
        out[name] = (im, pv, dict(layer="object", scale="battle", **fields))
    out["scene_bridge_u"] = (*scene_bridge(), dict(layer="object", scale="scene"))
    return out


OLD = ("gatehouse_closed_u", "gatehouse_closed_v", "gatehouse_floor_u", "gatehouse_floor_v", "village", "burnt_village")


def write():
    """save the kit into ../tiles and its entries of tiles.json, dropping the one-tile gatehouse it replaces. The room
    walls and floors take the names of the props.png pictures, which extra.png overrides in the game."""
    tiles = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "tiles")
    path = os.path.join(tiles, "tiles.json")
    man = json.load(open(path))
    for old in OLD:
        man["sprites"].pop(old, None)
        if os.path.exists(os.path.join(tiles, old + ".png")):
            os.remove(os.path.join(tiles, old + ".png"))
    for name, (im, pv, fields) in kit().items():
        im.save(os.path.join(tiles, name + ".png"))
        man["sprites"][name] = dict(file="tiles/%s.png" % name, size=list(im.size), pivot=list(pv), **fields,
                                    made="pixel art drawn on the small-tile grid (gen_pixel_kit.py)")
    json.dump(man, open(path, "w"), ensure_ascii=False, indent=1)
    open(path, "a").write("\n")


if __name__ == "__main__":
    if sys.argv[1:] == ["--write"]:
        write()
    elif sys.argv[1:2] == ["--compare"]:
        compare(sys.argv[2])
    else:
        print(__doc__)
