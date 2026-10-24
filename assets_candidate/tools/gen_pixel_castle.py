"""The scene castle kit drawn as pixel art straight onto the small-tile grid, no 3D renderer: the wall block (one small
tile, both orientations) and the ten-tile gate, open and closed, cut by owning tile.

Every solid is a convex body (an intersection of half-spaces) in tile units along u and v and pixels up z, seen
through the scene's projection x = 16(u - v), y = 8(u + v) - z. A pixel's colour comes from the nearest body its
centre's ray meets and a dot pattern laid on that body's face in face coordinates (s across in pixels, h up in
pixels), so a course of bricks steps up one pixel every two pixels across, as the tile edges do.

  python3 gen_pixel_castle.py --preview DIR   # pieces whole and cut, and sample scenes, into DIR
  python3 gen_pixel_castle.py --write         # save the kit into ../tiles and its entries of tiles.json
"""
import json
import os
import sys

import numpy as np
from PIL import Image

INK = (11, 16, 27)
WALL_H = 66               # wall face height, pixels (the 3D kit's 3.2 units); the gate's tower is as high
MERLON = (3, 8)           # parapet strip and merlon heights above the wall top
EDGE = .14                # parapet thickness, tiles
PASS_H = 46               # passage height
PAV = 18                  # pavilion wall height
GATE_TILES = 10
PASSAGE_VEIL = (14, 12, 10, 170)   # black laid over what shows through an open gate's passage

PAL = {
    # stone: lit face S/s, shaded face R/r as block/joint, top U
    "S": (214, 202, 178), "s": (156, 144, 124), "R": (150, 136, 114), "r": (104, 92, 76), "U": (190, 180, 160),
    "P": (74, 70, 66),                                    # passage inside
    "D": (126, 74, 42), "d": (88, 50, 30), "k": (52, 44, 40),   # door planks, plank seams, studs
    "C": (172, 52, 40), "c": (122, 36, 30),               # pavilion columns lit/shaded
    "L": (232, 224, 206), "l": (186, 178, 162),           # pavilion plaster lit/shaded
    # roof: tile rows lit/shaded, the furrows between them, the eave's round tile ends, the ridge
    "F": (92, 100, 116), "f": (62, 68, 82), "G": (70, 76, 90), "g": (48, 52, 64),
    "E": (150, 156, 166), "e": (110, 116, 128), "K": (54, 58, 70), "k2": (84, 90, 104),
}


def brick(s, h, a, b, w=8, rows=4):
    """courses of `rows` dot rows and a joint row, blocks w wide with one joint dot, every other course shifted half
    a block"""
    period = rows + 1
    r = h % (2 * period)
    course = r // period
    joint = (r % period == rows) | ((s + course * (w // 2)) % w == w - 1)
    return np.where(joint, b, a)


def box(u0, u1, v0, v1, z0, z1):
    return [((1, 0, 0), u1, "+u"), ((-1, 0, 0), -u0, "-u"), ((0, 1, 0), v1, "+v"), ((0, -1, 0), -v0, "-v"),
            ((0, 0, 1), z1, "+z"), ((0, 0, -1), -z0, "-z")]


class Body:
    def __init__(self, planes, paint, tile=None):
        self.planes, self.paint, self.tile = planes, paint, tile


def lit(f):
    """+v faces take the light, +u faces the shade, whichever way a line runs"""
    return f.endswith("v")


def stone_paint(f, u, v, z, s, h):
    if f == "+z":
        return np.full(u.shape, "U")
    return brick(s, h, "S", "s") if lit(f) else brick(s, h, "R", "r")


def merlon_paint(f, u, v, z, s, h):
    return np.full(u.shape, "U" if f == "+z" else "S" if lit(f) else "R")


def passage_paint(f, u, v, z, s, h):
    return np.full(u.shape, "P")


def door_paint(f, u, v, z, s, h):
    c = np.where(s % 4 == 3, "d", "D")
    return np.where((h % 10 == 6) & (s % 4 == 1), "k", c)


def pavilion_paint(f, u, v, z, s, h):
    col = (s % 8) < 2
    return np.where(col, "C", "L") if lit(f) else np.where(col, "c", "l")


def roof_paint(f, u, v, z, s, h):
    """tile rows running down each slope, a furrow every fourth dot, the row of round tile ends along the eave"""
    if f == "+z":
        return np.full(u.shape, "K")
    a, b, eave = ("F", "f", "E") if lit(f) else ("G", "g", "e")
    row = np.where(s % 4 == 3, b, a)
    return np.where(z < WALL_H + PAV, np.where(s % 4 == 3, "K", eave), row)


def ridge_paint(f, u, v, z, s, h):
    return np.full(u.shape, "k2" if f == "+z" else "K")


class Line:
    """a run along u or along v: boxes and planes given in (along, across) are laid the right way"""
    def __init__(self, orient):
        self.o = orient

    def box(self, a0, a1, c0, c1, z0, z1):
        return box(a0, a1, c0, c1, z0, z1) if self.o == "u" else box(c0, c1, a0, a1, z0, z1)

    def plane(self, na, nc, nz, d, name):
        """name ends with a or c, the plane's facing along or across, turned to u or v"""
        nu, nv = (na, nc) if self.o == "u" else (nc, na)
        axis = name[-1]
        return ((nu, nv, nz), d, name[:-1] + (("u" if self.o == "u" else "v") if axis == "a" else ("v" if self.o == "u" else "u")))

    def tile(self, j):
        """the canvas centre of the line's tile j"""
        return (16 * j, 8 * (j + 1)) if self.o == "u" else (-16 * j, 8 * (j + 1))

    def along(self):
        return (lambda u, v: np.floor(u - 1e-6)) if self.o == "u" else (lambda u, v: np.floor(v - 1e-6))

    def hip(self, a0, a1, c0, c1, zb, s, top=None):
        """a hipped roof over [a0,a1]x[c0,c1] rising s pixels per tile from zb on every side, cut flat at top"""
        out = [self.plane(0, s, 1, zb + s * c1, "r+c"), self.plane(0, -s, 1, zb - s * c0, "r-c"),
               self.plane(s, 0, 1, zb + s * a1, "r+a"), self.plane(-s, 0, 1, zb - s * a0, "r-a")]
        return out + self.box(a0, a1, c0, c1, zb, top if top is not None else 1e9)


def edge(side, u=0, v=0, tile=None):
    """the parapet along one side (+u, -u, +v, -v) of tile (u, v): a low strip and two merlons, the first at the
    side's low corner, so sides meeting at a corner share that merlon"""
    if side[1] == "u":
        a0, a1 = (u + 1 - EDGE, u + 1) if side[0] == "+" else (u, u + EDGE)
        spans = lambda k0, k1, z1: box(a0, a1, v + k0, v + k1, WALL_H, z1)
    else:
        a0, a1 = (v + 1 - EDGE, v + 1) if side[0] == "+" else (v, v + EDGE)
        spans = lambda k0, k1, z1: box(u + k0, u + k1, a0, a1, WALL_H, z1)
    out = [Body(spans(0, 1, WALL_H + MERLON[0]), merlon_paint, tile)]
    out += [Body(spans(k, k + .25, WALL_H + MERLON[1]), merlon_paint, tile) for k in (0, .5)]
    return out


def block(u=0, v=0, tile=None):
    return Body(box(u, u + 1, v, v + 1, 0, WALL_H), stone_paint, tile)


def gate(closed, L):
    """the ten-tile gate along the line from 0: wall all along with parapets on both long sides, the passage through
    tiles 4 and 5 with a stepped arch, and a pavilion over tiles 3 to 6"""
    t0, t1 = 3, 7
    out = [Body(L.box(0, t0 + 1, 0, 1, 0, WALL_H), stone_paint), Body(L.box(t1 - 1, GATE_TILES, 0, 1, 0, WALL_H), stone_paint),
           Body(L.box(t0 + 1, t1 - 1, 0, 1, PASS_H, WALL_H), stone_paint)]
    def shoulder(f, u, v, z, s, h):
        """stone on the wall's face, the passage's dark on the faces looking along the line (inside the arch)"""
        return passage_paint(f, u, v, z, s, h) if f.endswith(L.o) else stone_paint(f, u, v, z, s, h)

    for w, dz in ((.25, 6), (.125, 12)):   # the arch's shoulders, a step at a time
        out += [Body(L.box(t0 + 1, t0 + 1 + w, 0, 1, PASS_H - dz, PASS_H), shoulder),
                Body(L.box(t1 - 1 - w, t1 - 1, 0, 1, PASS_H - dz, PASS_H), shoulder)]
    out.append(Body(L.box(t0 + 1, t0 + 1 + 1e-3, 0, 1, 0, PASS_H), passage_paint))
    if closed:
        out.append(Body(L.box(t0 + 1, t1 - 1, .45, .55, 0, PASS_H - 1), door_paint))
    for j in range(GATE_TILES):
        for side in ("+", "-"):
            uu, vv = (j, 0) if L.o == "u" else (0, j)
            out += edge(side + ("v" if L.o == "u" else "u"), uu, vv)
    zp = WALL_H + PAV
    out.append(Body(L.box(t0 + .3, t1 - .3, .28, .72, WALL_H, zp), pavilion_paint))
    # the roof: a low skirt out to the wall's faces, a steeper hip above it, so the slope sags like a tiled roof
    out.append(Body(L.hip(t0, t1, 0, 1, zp - 2, 18, top=zp + 4), roof_paint))
    out.append(Body(L.hip(t0 + .22, t1 - .22, .22, .78, zp + 3, 44), roof_paint))
    zr = zp + 3 + 44 * .28
    out.append(Body(L.box(t0 + .48, t1 - .48, .45, .55, zr - 2, zr + 2), ridge_paint))
    for a in (t0 + .48, t1 - .48 - .12):   # the ridge's raised ends
        out.append(Body(L.box(a, a + .12, .44, .56, zr - 2, zr + 6), ridge_paint))
    return out


def render(bodies, along=lambda u, v: np.floor(u - 1e-6), bounds=None, veil=None):
    """the bodies drawn: RGBA, each pixel's owning tile (along the line; -1 for none), the canvas point of pixel (0, 0).
    veil (planes, rgba) tints the empty pixels whose ray passes through it, owned by the tile where the ray enters."""
    planes = [p for b in bodies for p in b.planes]
    us = [d for n, d, f in planes if n == (1, 0, 0)]
    vs = [d for n, d, f in planes if n == (0, 1, 0)]
    zs = [d for n, d, f in planes if n == (0, 0, 1) and abs(d) < 1e8] + [WALL_H + MERLON[1]]
    u0 = min(-d for b in bodies for (n, d, f) in b.planes if n == (-1, 0, 0))
    v0 = min(-d for b in bodies for (n, d, f) in b.planes if n == (0, -1, 0))
    x0, x1 = int(16 * (u0 - max(vs))) - 2, int(16 * (max(us) - v0)) + 2
    y0, y1 = int(8 * (u0 + v0) - max(zs)) - 2, int(8 * (max(us) + max(vs))) + 2
    if bounds:
        x0, y0, x1, y1 = bounds
    X, Y = np.meshgrid(np.arange(x0, x1) + .5, np.arange(y0, y1) + .5)
    au, av = X / 32 + Y / 16, Y / 16 - X / 32
    best = np.full(X.shape, -np.inf)
    rgb = np.zeros(X.shape + (3,), np.uint8)
    who = np.full(X.shape, -1)
    depth = np.full(X.shape, -np.inf)
    for bi, b in enumerate(bodies):
        lo, hi = np.full(X.shape, -np.inf), np.full(X.shape, np.inf)
        face = np.full(X.shape, "", object)
        ok = np.ones(X.shape, bool)
        for (nu, nv, nz), d, f in b.planes:
            c = nu / 16 + nv / 16 + nz
            rhs = d - nu * au - nv * av
            if c > 0:
                bound = rhs / c
                face = np.where(bound < hi, f, face)
                hi = np.minimum(hi, bound)
            elif c < 0:
                lo = np.maximum(lo, rhs / c)
            else:
                ok &= rhs >= 0
        hit = ok & (lo <= hi + 1e-9) & (hi > best + 1e-9)
        if not hit.any():
            continue
        z = hi
        u, v = au + z / 16, av + z / 16
        for f in set(face[hit]):
            m = hit & (face == f)
            s = np.floor(16 * u[m] + 1e-6).astype(int) if f.endswith("v") else np.floor(-16 * v[m] + 1e-6).astype(int)
            h = np.floor(z[m] + 1e-6).astype(int)
            keys = b.paint(f, u[m], v[m], z[m], s, h)
            rgb[m] = np.array([PAL[k] for k in np.atleast_1d(keys)], np.uint8).reshape(-1, 3)
        best = np.where(hit, z, best)
        depth = np.where(hit, u + v, depth)
        who = np.where(hit, along(u, v).astype(int) if b.tile is None else b.tile, who)
    solid = np.isfinite(best)
    # ink: a pixel beside a nearer one (by more than a quarter tile), or empty beside a solid one; the ink belongs
    # with the nearer pixel's tile
    ink = np.zeros(X.shape, bool)
    inkwho = np.full(X.shape, -1)
    inkdepth = np.full(X.shape, -np.inf)
    H, W = X.shape
    for dy, dx in ((0, 1), (0, -1), (1, 0), (-1, 0)):
        nd = np.full(X.shape, -np.inf)
        nw = np.full(X.shape, -1)
        ys, yd = slice(max(dy, 0), H + min(dy, 0)), slice(max(-dy, 0), H + min(-dy, 0))
        xs, xd = slice(max(dx, 0), W + min(dx, 0)), slice(max(-dx, 0), W + min(-dx, 0))
        nd[yd, xd] = depth[ys, xs]
        nw[yd, xd] = who[ys, xs]
        nearer = np.isfinite(nd) & (~solid | (nd > depth + .25))
        take = nearer & (nd > inkdepth)
        ink |= nearer
        inkwho = np.where(take, nw, inkwho)
        inkdepth = np.where(take, nd, inkdepth)
    a = np.zeros(X.shape + (4,), np.uint8)
    a[..., :3] = rgb
    a[solid, 3] = 255
    a[ink] = INK + (255,)
    owner = np.where(ink, inkwho, np.where(solid, who, -1))
    if veil:
        lo, hi = np.full(X.shape, -np.inf), np.full(X.shape, np.inf)
        for (nu, nv, nz), d, f in veil[0]:
            c = nu / 16 + nv / 16 + nz
            bound = (d - nu * au - nv * av) / c
            hi, lo = (np.minimum(hi, bound), lo) if c > 0 else (hi, np.maximum(lo, bound))
        through = (lo <= hi) & ~solid & ~ink
        a[through] = veil[1]
        owner = np.where(through, along(au + hi / 16, av + hi / 16).astype(int), owner)
    return a, owner, (x0, y0)


def piece(a, owner, origin, k, center):
    """the pixels owned by k, trimmed, and the pivot: canvas point `center` (the owning tile's centre)"""
    b = a.copy()
    b[owner != k] = 0
    ys, xs = np.nonzero(b[..., 3])
    if len(xs) == 0:
        return None
    x0, x1, y0, y1 = xs.min(), xs.max() + 1, ys.min(), ys.max() + 1
    return Image.fromarray(b[y0:y1, x0:x1]), (int(center[0] - origin[0] - x0), int(center[1] - origin[1] - y0))


BOUNDS = (-48, -110, 48, 40)


def block_kit():
    """the wall tile and its four parapet sides. The tile is drawn beside the tiles behind it on both sides, (-1, 0)
    and (0, -1), so a line meets unbroken either way; a parapet side takes, besides its own dots, the tile's outline
    that the missing tile on its side would have hidden. Each side is drawn beside the same side of the tile behind
    it along that side."""
    behind = {"-u": (-1, 0), "-v": (0, -1)}
    a, own, o = render([block(tile=0), block(-1, 0, -1), block(0, -1, -1)], bounds=BOUNDS)
    out = {"swall2": piece(a, own, o, 0, (0, 8))}
    base = own == 0
    for side in ("+u", "-u", "+v", "-v"):
        ctx = [block(*behind[k], tile=-1) for k in behind if k != side]
        prev = (0, -1) if side[1] == "u" else (-1, 0)
        bodies = [block(tile=0)] + ctx + edge(side, tile=1) + edge(side, *prev, tile=-1)
        b, own2, _ = render(bodies, bounds=BOUNDS)
        mine = (own2 == 1) | ((own2 == 0) & ~base & np.all(b[..., :3] == INK, axis=-1))
        own2 = np.where(mine, 1, -1)
        out["sparapet_" + side.replace("+", "p").replace("-", "m")] = piece(b, own2, o, 1, (0, 8))
    return out


def gate_kit(closed, orient):
    """the gate's ten tiles, each with its pivot at its own tile's centre, drawn beside the wall tile behind it"""
    L = Line(orient)
    back = block(-1, 0, -1) if orient == "u" else block(0, -1, -1)
    veil = None if closed else (L.box(4, 6, 0, 1, 0, PASS_H), PASSAGE_VEIL)   # the dark of the open passage
    a, owner, o = render([back] + gate(closed, L), along=L.along(), veil=veil)
    return [piece(a, owner, o, j, L.tile(j)) for j in range(GATE_TILES)]


SHADE = ((.16, 128), (.38, 84), (.62, 42))   # the ground shadow at a wall's foot: bands out from the face (tiles) and their alpha
SHADE_RGB = (18, 22, 14)


def shade_kit():
    """the shadow a wall tile lays on the ground in front of it: a band out from its +u side, one out from its +v side,
    and the square between them past its front corner, for when both sides are open. Darkest at the foot, in steps."""
    x0, y0, x1, y1 = -40, -4, 40, 40
    X, Y = np.meshgrid(np.arange(x0, x1) + .5, np.arange(y0, y1) + .5)
    u, v = X / 32 + Y / 16, Y / 16 - X / 32
    out = {}
    for name, du, dv in (("pu", u - 1, None), ("pv", None, v - 1), ("pc", u - 1, v - 1)):
        if name == "pu":
            d, inside = du, (v >= 0) & (v < 1) & (du >= 0)
        elif name == "pv":
            d, inside = dv, (u >= 0) & (u < 1) & (dv >= 0)
        else:
            d, inside = np.maximum(du, dv), (du >= 0) & (dv >= 0)
        a = np.zeros(X.shape, np.uint8)
        for w, alpha in reversed(SHADE):
            a[inside & (d < w)] = alpha
        img = np.zeros(X.shape + (4,), np.uint8)
        img[..., :3] = SHADE_RGB
        img[..., 3] = a
        out["swall2_shade_" + name] = piece(img, np.where(a > 0, 0, -1), (x0, y0), 0, (0, 8))
    return out


def kit():
    out = block_kit()
    out.update(shade_kit())
    for state, closed in (("closed", True), ("open", False)):
        for o in ("u", "v"):
            for j, (im, pv) in enumerate(gate_kit(closed, o)):
                out["sgate2_%s_%s_%d" % (state, o, j)] = (im, pv, (j - 5) / 2)
    return out


def write():
    """save the kit into ../tiles and its entries of tiles.json, dropping the 3D kit's wall pictures it replaces"""
    tiles = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "tiles")
    made = {}
    for old in ("swall2_u", "swall2_v"):
        if os.path.exists(os.path.join(tiles, old + ".png")):
            os.remove(os.path.join(tiles, old + ".png"))
    for name, (im, pv, *off) in kit().items():
        im.save(os.path.join(tiles, name + ".png"))
        made[name] = dict(file="tiles/%s.png" % name, size=list(im.size), pivot=list(pv), layer="object", scale="scene",
                          made="pixel art drawn on the small-tile grid (gen_pixel_castle.py)")
        if off:
            made[name]["gate_offset"] = off[0]
    path = os.path.join(tiles, "tiles.json")
    man = json.load(open(path))
    for old in ("swall2_u", "swall2_v"):
        man["sprites"].pop(old, None)
    man["sprites"].update(made)
    json.dump(man, open(path, "w"), ensure_ascii=False, indent=1)
    open(path, "a").write("\n")


def preview(out):
    os.makedirs(out, exist_ok=True)
    k = kit()
    zoom = lambda im, n=4: im.resize((im.width * n, im.height * n), Image.NEAREST)

    def sheet(items, gap=8, bg=(40, 44, 52, 255)):
        w = sum(im.width for im in items) + gap * (len(items) + 1)
        h = max(im.height for im in items) + 2 * gap
        s = Image.new("RGBA", (w, h), bg)
        x = gap
        for im in items:
            s.alpha_composite(im, (x, h - gap - im.height))
            x += im.width + gap
        return s

    def scene(walls, gates, lo=-2, hi=14, bg=(118, 128, 96, 255)):
        """wall tiles (block, then parapets on the sides no wall tile touches, back sides first) and gate slices at
        tile centres over a small-tile grid, drawn by u + v then x"""
        from PIL import ImageDraw
        x0, x1 = 16 * (lo - hi) - 40, 16 * (hi - lo) + 40
        y0, y1 = 16 * lo - 150, 16 * hi + 36
        s = Image.new("RGBA", (x1 - x0, y1 - y0), bg)
        d = ImageDraw.Draw(s)
        for t in range(lo, hi + 1):
            d.line([(16 * (t - lo) - x0, 8 * (t + lo) - y0), (16 * (t - hi) - x0, 8 * (t + hi) - y0)], (96, 106, 78))
            d.line([(16 * (lo - t) - x0, 8 * (lo + t) - y0), (16 * (hi - t) - x0, 8 * (hi + t) - y0)], (96, 106, 78))
        solid = set(walls) | set(gates)
        for U, V in sorted(solid, key=lambda p: (p[0] + p[1], p[0] - p[1])):
            names = ["swall2_shade_" + side for side, opens in (
                ("pu", [(1, 0)]), ("pv", [(0, 1)]), ("pc", [(1, 0), (0, 1)])) if all((U + du, V + dv) not in solid for du, dv in opens)]
            names += [gates[(U, V)]] if (U, V) in gates else ["swall2"] + [
                "sparapet_" + side for side, (du, dv) in (("mu", (-1, 0)), ("mv", (0, -1)), ("pu", (1, 0)), ("pv", (0, 1)))
                if (U + du, V + dv) not in solid]
            for name in names:
                im, pv = k[name][:2]
                s.alpha_composite(im, (16 * (U - V) - x0 - pv[0], 8 * (U + V + 1) - y0 - pv[1]))
        return s

    parts = [k[n][0] for n in ("swall2", "sparapet_mu", "sparapet_mv", "sparapet_pu", "sparapet_pv")]
    zoom(sheet(parts)).save(os.path.join(out, "block.png"))
    for state in ("closed", "open"):
        for o in ("u", "v"):
            names = ["sgate2_%s_%s_%d" % (state, o, j) for j in range(GATE_TILES)]
            zoom(sheet([k[n][0] for n in names]), 3).save(os.path.join(out, "cut_%s_%s.png" % (state, o)))
            tiles = {((j, 0) if o == "u" else (0, j)): n for j, n in enumerate(names)}
            zoom(scene({}, tiles, -1, 11), 3).save(os.path.join(out, "whole_%s_%s.png" % (state, o)))
    # a closed square corner: a line along u with a closed gate, one along v with an open gate, a short spur
    walls = [(u, 0) for u in range(0, 13) if not 2 <= u < 12] + [(0, v) for v in range(1, 14) if not 3 <= v < 13]
    walls += [(u, 10) for u in range(1, 5)]
    gates = {(u, 0): "sgate2_closed_u_%d" % (u - 2) for u in range(2, 12)}
    gates.update({(0, v): "sgate2_open_v_%d" % (v - 3) for v in range(3, 13)})
    zoom(scene(walls, gates), 2).save(os.path.join(out, "scene.png"))
    zoom(scene([(u, 0) for u in range(0, 5)] + [(0, v) for v in range(1, 5)] + [(4, v) for v in range(1, 5)] + [(u, 4) for u in range(1, 4)],
               {}, -1, 6), 4).save(os.path.join(out, "square.png"))


if __name__ == "__main__":
    if sys.argv[1:] == ["--write"]:
        write()
    elif sys.argv[1:2] == ["--preview"]:
        preview(sys.argv[2])
    else:
        print(__doc__)
