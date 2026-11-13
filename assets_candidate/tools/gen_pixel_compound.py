"""The compound wall (담장) and its roofed main gate (솟을대문) for the story scene maps, drawn as pixel art on the
small-tile grid with the castle kit's renderer (gen_pixel_castle.render): a low wall about a person's height, white
plaster on a stone footing under a tiled coping, and a gate whose raised middle roof stands over a passage two small
tiles wide between two lower wings.

Every colour is a grey, so the game tints a picture by multiplying it with a colour (format 2 object and wall-line
"tint", docs/presentation-scene.md): white plaster and black tiles as they are, an ochre earth wall, a red palace or
shrine wall, a blue-grey office wall. Each piece comes as separate layers that do not overlap, so each takes its own
colour: <name> the body (plaster, footing), <name>_roof the tiles, and for the gate <name>_wood (posts, beams,
leaves). The outline goes with the layer next to it.

  cwall_<mask>          one small tile of a wall line (scene.json walls "cwall": piece), mask the sides another wall
                        or gate tile closes: 1 -u, 2 -v, 4 +u, 8 +v (05 a straight run along u, 10 along v, 12 03 06
                        09 corners, 01 04 02 08 ends, 07 11 13 14 junctions, 15 a cross, 00 a post). The wall stands on
                        the tile's middle line, 0.3 tile thick; arms run from the middle out to the closed sides, an end
                        reaches the open side flush, its coping hipped down onto it. Pivot at the tile's centre.
  cgate_{open,closed}_{u,v}_<k>   the gate along u / along v, four small tiles k = 0..3 (gate_offset (k - 2) / 2
                        game cells from the gate's point), the passage on k = 1, 2 (scene.json gate_passage). Its
                        leaves stand at the gate's front (+v for a gate along u, +u along v) and open inward.

  python3 gen_pixel_compound.py --preview DIR   # the pieces at 4x, a laid-out compound in the base greys and tinted
  python3 gen_pixel_compound.py --write         # save them into ../tiles and their entries of tiles.json
"""
import json
import os
import sys

import numpy as np
from PIL import Image

import gen_pixel_castle as pc

# greys only, every value used once so a pixel's layer can be read back from its colour
BODY = {
    "CWl": 236, "CWs": 194, "CWt": 214,            # plaster: lit (+v) face, shaded (+u) face, top
    "CBl": 172, "CBs": 142,                        # the shadow band under the coping
    "CFl": 166, "CFj": 134, "CFs": 128, "CFk": 102, "CFt": 150,   # stone footing: lit/joint, shaded/joint, top
}
ROOF = {
    "CRl": 112, "CRf": 88, "CRs": 84, "CRg": 66,   # tile rows and furrows, lit / shaded slopes
    "CEl": 146, "CEs": 116, "CEk": 44,             # the eave's round tile ends lit / shaded, the gaps between
    "CK": 54, "CKt": 80,                           # ridge sides, ridge top
}
WOOD = {
    "CPl": 158, "CPs": 120, "CPt": 140,            # posts and beams
    "CDl": 178, "CDs": 138, "CDj": 108, "CDk": 222,  # leaves, plank seams, iron fittings
}
LAYERS = (("", BODY), ("_wood", WOOD), ("_roof", ROOF))
INK = (22, 22, 22)
for keys in (BODY, ROOF, WOOD):
    pc.PAL.update({k: (g, g, g) for k, g in keys.items()})
GREY_LAYER = {g: i for i, (_, keys) in enumerate(LAYERS) for g in keys.values()}
assert len(GREY_LAYER) == sum(len(k) for _, k in LAYERS), "every grey once"

FOOT, TOP = 5, 22          # footing and plaster top, pixels
T = .15                    # the wall's half thickness, tiles
OVER = .27                 # the coping's overhang past the face
SLOPE = 22                 # coping pitch, pixels per tile


def plaster(f, u, v, z, s, h):
    lit = pc.lit(f)
    if f == "+z":
        return np.full(u.shape, "CWt")
    band = z >= TOP - 2
    body = np.where(band, "CBl" if lit else "CBs", "CWl" if lit else "CWs")
    return body


def footing(f, u, v, z, s, h):
    if f == "+z":
        return np.full(u.shape, "CFt")
    return pc.brick(s, h, "CFl", "CFj", w=6, rows=2) if pc.lit(f) else pc.brick(s, h, "CFs", "CFk", w=6, rows=2)


def roof(eave_z):
    def paint(f, u, v, z, s, h):
        if f == "+z":
            return np.full(u.shape, "CKt")
        lit = f.endswith("v")
        a, b, end = ("CRl", "CRf", "CEl") if lit else ("CRs", "CRg", "CEs")
        rows = np.where(np.mod(s, 4) == 3, b, a)
        return np.where(z < eave_z + 2, np.where(np.mod(s, 2) == 0, end, "CEk"), rows)
    return paint


def ridge(f, u, v, z, s, h):
    return np.full(u.shape, "CKt" if f == "+z" else "CK")


def post(f, u, v, z, s, h):
    return np.full(u.shape, "CPt" if f == "+z" else "CPl" if pc.lit(f) else "CPs")


def leaf(L, a0, a1):
    """a door leaf seen from the front: planks along its height, two iron bands and a ring of studs"""
    def paint(f, u, v, z, s, h):
        lit = pc.lit(f)
        if f == "+z":
            return np.full(u.shape, "CPt")
        a = u if L.o == "u" else v
        seam = np.mod(s, 4) == 0
        band = (np.abs(z - 6) < 1) | (np.abs(z - 21) < 1)
        stud = band & (np.mod(s, 3) == 1)
        out = np.where(seam, "CDj", "CDl" if lit else "CDs")
        out = np.where(band, "CDj", out)
        return np.where(stud, "CDk", out)
    return paint


def wall_line(L, a0, a1, c=.5, end0=False, end1=False, tile=None, ridge_to=None):
    """the wall along L from a0 to a1 on the line c. A run's coping goes on past both ends (the pieces cut it by tile,
    or a corner's other arm covers it); an end stops flush with its end face, the coping hipped down onto it"""
    B = pc.Body
    w = T + OVER
    r0 = a0 + (w if end0 else 0)
    r1 = a1 - (w if end1 else 0)
    if ridge_to is not None:   # an arm of a corner: its ridge stops at the middle, where the other arm's meets it
        r0, r1 = (max(r0, ridge_to), r1) if a0 > ridge_to - 1 else (r0, min(r1, ridge_to))
    return [B(L.box(a0, a1, c - T - .03, c + T + .03, 0, FOOT), footing, tile),
            B(L.box(a0, a1, c - T, c + T, FOOT, TOP), plaster, tile),
            B(L.hip(a0, a1, c - w, c + w, TOP - 1, SLOPE, top=TOP + 6), roof(TOP - 1), tile),
            B(L.box(r0, r1, c - .07, c + .07, TOP + 4, TOP + 8), ridge, tile)]


def mask_bodies(mask):
    """the wall bodies of small tile [0, 1] x [0, 1] whose closed sides are mask, running on into the closed
    neighbours (owned by them)"""
    c = .5
    U, V = pc.Line("u"), pc.Line("v")
    arms = {"u": (mask & 1, mask & 4), "v": (mask & 2, mask & 8)}
    out = []
    for L, axis in ((U, "u"), (V, "v")):
        lo, hi = arms[axis]
        if not (lo or hi):
            continue
        other = arms["v" if axis == "u" else "u"]
        bend = other[0] or other[1]     # the line turns or branches here: the arm stops at the middle
        if lo and hi:
            out += wall_line(L, -3, 4)
        elif lo:
            out += wall_line(L, -3, c + T + OVER if bend else 1, end1=not bend, ridge_to=c if bend else None)
        else:
            out += wall_line(L, c - T - OVER if bend else 0, 4, end0=not bend, ridge_to=c if bend else None)
    if mask == 0:           # a lone post: a square of wall under a hipped cap
        B = pc.Body
        w = T + OVER
        out = [B(pc.box(c - T - .03, c + T + .03, c - T - .03, c + T + .03, 0, FOOT), footing),
               B(pc.box(c - T, c + T, c - T, c + T, FOOT, TOP), plaster),
               B(U.hip(c - w, c + w, c - w, c + w, TOP - 1, SLOPE), roof(TOP - 1))]
    return out


def own_tile(u, v):
    return np.where((np.floor(u - 1e-6) == 0) & (np.floor(v - 1e-6) == 0), 0, -1)


BOUNDS = (-40, -48, 40, 28)


def wall_piece(mask):
    a, own, o = pc.render(mask_bodies(mask), along=own_tile, bounds=BOUNDS)
    return layered(a, own, o, 0, (0, 8))


# ---------------------------------------------------------------- the gate

GZ = 38                    # the middle roof's eave
WZ = 24                    # the wings' eave


def gate_bodies(closed, L):
    """the gate along L over a in [0, 4] (tiles 0..3), the passage a in [1, 3], across c in [0, 1]; the front at +c"""
    B = pc.Body
    out = wall_line(L, -3, .3, tile=None) + wall_line(L, 3.7, 7, tile=None)
    # the wings: a low plastered gatehouse either side of the passage, on the full depth, under its own roof
    for a0, a1 in ((.04, .9), (3.1, 3.96)):
        out += [B(L.box(a0, a1, .1, .9, 0, FOOT), footing), B(L.box(a0, a1, .12, .88, FOOT, WZ), plaster)]
    out.append(B(L.hip(.03, 1.02, .04, .96, WZ - 2, 24, top=WZ + 9), roof(WZ - 2)))
    out.append(B(L.hip(2.98, 3.97, .04, .96, WZ - 2, 24, top=WZ + 9), roof(WZ - 2)))
    for a0, a1 in ((.3, .75), (3.25, 3.7)):
        out.append(B(L.box(a0, a1, .45, .55, WZ + 7, WZ + 11), ridge))
    # posts at the passage's four corners, beams over its front and back
    for a0 in (.86, 2.96):
        for c0 in (.04, .76):
            out.append(B(L.box(a0, a0 + .18, c0, c0 + .2, 0, GZ), post))
    for c0 in (.06, .78):
        out.append(B(L.box(.86, 3.14, c0, c0 + .16, GZ - 5, GZ), post))
    out.append(B(L.box(1.04, 2.96, .82, .9, 0, 2), post))            # the sill
    if closed:
        out.append(B(L.box(1.04, 2.96, .82, .88, 2, GZ - 5), leaf(L, 1.04, 2.96)))
    else:                       # each leaf swung back against its post, inside the passage
        for a0 in (1.04, 2.9):
            out.append(B(L.box(a0, a0 + .06, .08, .82, 2, GZ - 5), leaf(L, a0, a0 + .06)))
    # the raised middle roof, its ridge with lifted ends
    out.append(B(L.hip(.7, 3.3, .02, .98, GZ - 2, 30, top=GZ + 13), roof(GZ - 2)))
    out.append(B(L.box(1.0, 3.0, .44, .56, GZ + 11, GZ + 15), ridge))
    for a0 in (1.0, 2.86):
        out.append(B(L.box(a0, a0 + .14, .43, .57, GZ + 11, GZ + 19), ridge))
    return out


def gate_pieces(closed, orient):
    L = pc.Line(orient)
    a, own, o = pc.render(gate_bodies(closed, L), along=L.along(), bounds=(-110, -80, 110, 70))
    return [layered(a, own, o, k, L.tile(k)) for k in range(4)]


# ---------------------------------------------------------------- layers

def layered(a, owner, origin, k, center):
    """the pixels owned by k split by layer: [(suffix, image, pivot)] for each layer that has any (the body always,
    a 1 x 1 clear picture when empty). Ink goes with the layer of a solid neighbour, the roof's first."""
    b = a.copy()
    b[owner != k] = 0
    rgb, alpha = b[..., :3], b[..., 3] > 0
    ink = alpha & np.all(rgb == pc.INK, axis=-1)
    layer = np.full(alpha.shape, -1)
    for g, i in GREY_LAYER.items():
        layer[alpha & ~ink & np.all(rgb == g, axis=-1)] = i
    assert not (alpha & ~ink & (layer < 0)).any(), "a colour off the palette"
    H, W = alpha.shape
    near = np.full(alpha.shape, -1)
    for pick in (2, 1, 0):          # roof, then wood, then body
        hit = np.zeros(alpha.shape, bool)
        for dy, dx in ((0, 1), (0, -1), (1, 0), (-1, 0), (1, 1), (-1, -1), (1, -1), (-1, 1)):
            sh = np.full(alpha.shape, -1)
            ys, yd = slice(max(dy, 0), H + min(dy, 0)), slice(max(-dy, 0), H + min(-dy, 0))
            xs, xd = slice(max(dx, 0), W + min(dx, 0)), slice(max(-dx, 0), W + min(-dx, 0))
            sh[yd, xd] = layer[ys, xs]
            hit |= sh == pick
        near = np.where((near < 0) & hit, pick, near)
    layer = np.where(ink, np.where(near < 0, 0, near), layer)
    b[ink, :3] = INK
    out = []
    for i, (suffix, _) in enumerate(LAYERS):
        m = layer == i
        ys, xs = np.nonzero(m)
        if len(xs) == 0:
            if suffix == "":
                out.append((suffix, Image.new("RGBA", (1, 1)), (0, 0)))
            continue
        c = np.zeros_like(b)
        c[m] = b[m]
        x0, x1, y0, y1 = xs.min(), xs.max() + 1, ys.min(), ys.max() + 1
        out.append((suffix, Image.fromarray(c[y0:y1, x0:x1]),
                    (int(center[0] - origin[0] - x0), int(center[1] - origin[1] - y0))))
    return out


MASKS = range(16)


def kit():
    """name -> (image, pivot, gate_offset or None)"""
    out = {}
    for m in MASKS:
        for suffix, im, pv in wall_piece(m):
            out["cwall_%02d%s" % (m, suffix)] = (im, pv, None)
    for state, closed in (("open", False), ("closed", True)):
        for o in ("u", "v"):
            for k, layers in enumerate(gate_pieces(closed, o)):
                for suffix, im, pv in layers:
                    out["cgate_%s_%s_%d%s" % (state, o, k, suffix)] = (im, pv, (k - 2) / 2 if suffix == "" else None)
    return out


MADE = "compound wall and gate, greys to be tinted, pixel art on the small-tile grid (gen_pixel_compound.py)"


def write():
    here = os.path.dirname(os.path.abspath(__file__))
    tiles_dir = os.path.join(here, "..", "tiles")
    path = os.path.join(tiles_dir, "tiles.json")
    made = {}
    for name, (img, pivot, off) in kit().items():
        img.save(os.path.join(tiles_dir, name + ".png"))
        made[name] = {"file": f"tiles/{name}.png", "size": list(img.size), "pivot": list(pivot),
                      "layer": "object", "scale": "scene", "made": MADE, "shadow": False}
        if off is not None:
            made[name]["gate_offset"] = off
    tiles = json.load(open(path))
    tiles["sprites"].update(made)
    json.dump(tiles, open(path, "w"), ensure_ascii=False, indent=1)
    open(path, "a").write("\n")


# ---------------------------------------------------------------- preview

def tinted(im, rgb):
    a = np.array(im).astype(float)
    a[..., :3] *= np.array(rgb) / 255
    return Image.fromarray(a.round().astype(np.uint8))


def lay(cv, k, name, u, v, ox, oy, tint):
    """draw the piece at small tile (u, v) with its layers, each multiplied by its colour"""
    for suffix, key in (("", "body"), ("_wood", "wood"), ("_roof", "roof")):
        if name + suffix not in k:
            continue
        im, (px, py), _ = k[name + suffix]
        if key in tint:
            im = tinted(im, tint[key])
        x, y = ox + 16 * (u - v), oy + 8 * (u + v + 1)
        cv.alpha_composite(im, (int(x - px), int(y - py)))


def compound(k, tint, gate="open", w=10, d=7, bg=(150, 146, 132, 255)):
    """a w x d compound, its gate in the middle of the front (+v) wall and a short spur out of its right wall, drawn
    back to front by tile"""
    wall = {(u, 0) for u in range(w)} | {(u, d - 1) for u in range(w)} | {(0, v) for v in range(d)} | {(w - 1, v) for v in range(d)}
    wall |= {(w + i, 2) for i in range(3)}
    gt = w // 2
    gates = {(gt - 2 + j, d - 1): "cgate_%s_u_%d" % (gate, j) for j in range(4)}
    cv = Image.new("RGBA", (16 * (w + d + 4) + 80, 8 * (w + d + 4) + 90), bg)
    ox, oy = 16 * (d + 2) + 40, 70
    for (u, v) in sorted(wall, key=lambda p: (p[0] + p[1], p[0])):
        if (u, v) in gates:
            lay(cv, k, gates[(u, v)], u, v, ox, oy, tint)
            continue
        m = sum(bit for bit, (du, dv) in ((1, (-1, 0)), (2, (0, -1)), (4, (1, 0)), (8, (0, 1))) if (u + du, v + dv) in wall)
        lay(cv, k, "cwall_%02d" % m, u, v, ox, oy, tint)
    return cv


TINTS = [("기본 (흰 회벽·검은 기와)", {"wood": (150, 96, 64)}),
         ("흙담", {"body": (214, 168, 108), "roof": (200, 186, 160), "wood": (150, 110, 74)}),
         ("붉은 담 (궁성·사당)", {"body": (200, 80, 62), "wood": (180, 60, 44)}),
         ("관청 (청회색)", {"body": (164, 178, 196), "roof": (150, 160, 176), "wood": (110, 100, 96)})]


def preview(out):
    os.makedirs(out, exist_ok=True)
    k = kit()
    zoom = lambda im, n: im.resize((im.width * n, im.height * n), Image.NEAREST)
    # the strip of pieces in the base greys
    names = ["cwall_05", "cwall_10", "cwall_12", "cwall_09", "cwall_06", "cwall_03", "cwall_01", "cwall_04",
             "cwall_02", "cwall_08", "cwall_07", "cwall_15", "cwall_00"]
    cv = Image.new("RGBA", (64 * len(names) + 16, 80), (150, 146, 132, 255))
    for i, n in enumerate(names):
        lay(cv, k, n, 0, 0, 40 + 64 * i, 40, {})
    zoom(cv, 4).save(os.path.join(out, "pieces.png"))
    for state in ("open", "closed"):
        cv = Image.new("RGBA", (420, 150), (150, 146, 132, 255))
        for j in range(4):
            lay(cv, k, "cgate_%s_u_%d" % (state, j), j, 0, 70, 50, {"wood": (150, 96, 64)})
        for j in range(4):
            lay(cv, k, "cgate_%s_v_%d" % (state, j), 0, j, 330, 50, {"wood": (150, 96, 64)})
        zoom(cv, 4).save(os.path.join(out, "gate_%s.png" % state))
    for i, (label, t) in enumerate([("회색 원본", {})] + TINTS):
        zoom(compound(k, t, "open" if i % 2 == 0 else "closed"), 3).save(os.path.join(out, "compound_%d.png" % i))


if __name__ == "__main__":
    if sys.argv[1:2] == ["--write"]:
        write()
    elif sys.argv[1:2] == ["--preview"]:
        preview(sys.argv[2])
    else:
        print(__doc__)
