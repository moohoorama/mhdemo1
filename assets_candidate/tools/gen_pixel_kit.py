"""Candidates drawn the way gen_pixel_castle.py draws the scene castle: dot patterns laid on the faces of boxes on the
small-tile grid (x = 16(u - v), y = 8(u + v) - z), so every course and joint steps one pixel up per two across.

  battle walls  wall_clean_00..15: one small tile, the open sides (nw -u, ne -v, se +u, sw +v as bits 0..3) crenellated
  battle gate   gatehouse_{closed,open}_{u,v}_{0..3}: one game cell (2 x 2 small tiles) in a two-tile-thick wall line,
                a walled courtyard with a wide arch in each face under a slab on red posts, the leaves shut or swung
                back (the open one also stands for a pass), cut by owning small tile (index du + 2 dv)
  room walls    wall_{plaster,lacquer,canvas}_{nw,ne}: one game-cell edge, anchored at the cell's back corner
  room floors   floor_{wood,carpet,palace,tent}: one game cell, a 64 x 32 diamond, one dot per pixel

  python3 gen_pixel_kit.py --compare DIR   # old and new pieces, a battle castle corner and four rooms, into DIR
  python3 gen_pixel_kit.py --write         # save the kit into ../tiles and tiles.json (then ver5/tools/build_extra.py)
"""
import json
import os
import sys

import numpy as np
from PIL import Image, ImageDraw

import gen_pixel_castle as pc
from gen_pixel_castle import Body, box, brick, render, piece
from common import prop, sheet, INDEX

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


def compare(out):
    os.makedirs(out, exist_ok=True)
    zoom = lambda im, n: im.resize((im.width * n, im.height * n), Image.NEAREST)
    for new in (False, True):
        tag = "new" if new else "old"
        zoom(battle_scene(new), 2).save(os.path.join(out, "battle_%s.png" % tag))
        for name, fs, ws, pil in ROOMS:
            zoom(room_scene(fs, ws, pil, new), 3).save(os.path.join(out, "room_%s_%s.png" % (name, tag)))

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
    return out


OLD = ("gatehouse_closed_u", "gatehouse_closed_v", "gatehouse_floor_u", "gatehouse_floor_v")


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
