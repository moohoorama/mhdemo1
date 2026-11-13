"""원술's hoard for the story scenes (17장, after 광릉), drawn as pixel art on the small-tile grid with the castle kit's
renderer (gen_pixel_castle.render): small props, each on one small tile, pivot at the tile's centre on the ground.

  hoard_coin_chest  an open lacquered chest heaped with strung bronze coins, the lid propped open behind, a string of
                    coins spilled in front.
  hoard_chests      two closed lacquered chests stacked, a small one beside them, iron bands and bronze locks.
  hoard_tools       farm tools on a wooden rack: two hoes and a spade leaning on the bar, a sickle hung from it.
  hoard_silk        bolts of cloth rolled and stacked, three below and two above, ends showing the wound layers.

Rods, lids and rolls are convex bodies too: oriented boxes (`obox`) and octagonal prisms (`roll`), their planes in the
renderer's units (u, v in tiles, z in pixels; 16 pixels up count as one tile for their shape).

  python3 gen_pixel_hoard.py --preview DIR   # the props at 4x beside grain_sack and water_jar, into DIR
  python3 gen_pixel_hoard.py --write         # save them into ../tiles and their entries of tiles.json
"""
import json
import math
import os
import sys

import numpy as np
from PIL import Image

import gen_pixel_castle as pc

ZK = 16.0                  # pixels up per tile, for the shape of tilted bodies

pc.PAL.update({
    # lacquer: body lit/shaded, lid top, red interior lit/shaded, iron bands lit/shaded, bronze lock
    "QA": (128, 46, 34), "Qa": (88, 32, 26), "QT": (150, 60, 42), "QI": (184, 66, 46), "Qi": (140, 48, 36),
    "QN": (60, 54, 52), "Qn": (42, 38, 38), "QY": (214, 176, 84), "Qy": (160, 124, 58),
    # black lacquer with a red rim (the small chest)
    "QB": (58, 46, 44), "Qb": (40, 32, 32), "QU": (76, 62, 58),
    # bronze coins: mid, bright, highlight, dark hole, patina; red cord
    "QC": (156, 110, 54), "QD": (186, 138, 70), "QH": (226, 190, 116), "QK": (84, 56, 30), "QP": (104, 132, 108),
    "QR": (176, 44, 36), "Qr": (124, 30, 26),
    # wood: lit, shaded, top, grain; iron blades lit/shaded/edge
    "QW": (150, 100, 58), "Qw": (108, 70, 40), "QX": (170, 120, 72), "QG": (92, 58, 34),
    "QS": (176, 182, 188), "Qs": (120, 126, 134), "QE": (220, 224, 228),
    # cloth bolts: (lit, shaded, top) per colour, and the end's wound rings
    "S1": (70, 82, 140), "s1": (48, 56, 102), "t1": (92, 104, 164),
    "S2": (214, 200, 164), "s2": (166, 152, 120), "t2": (232, 222, 192),
    "S3": (170, 52, 44), "s3": (124, 36, 32), "t3": (196, 72, 58),
    "S4": (196, 156, 60), "s4": (148, 114, 40), "t4": (220, 184, 86),
    "S5": (78, 122, 92), "s5": (54, 88, 66), "t5": (100, 146, 112),
})


# ---------------------------------------------------------------- shapes

def face_name(m):
    """the face a plane's normal (u, v, z per tile) most looks along"""
    n = np.array(m, float) / np.linalg.norm(m)
    if abs(n[2]) > .7:
        return "+z" if n[2] > 0 else "-z"
    if abs(n[0]) > abs(n[1]):
        return "+u" if n[0] > 0 else "-u"
    return "+v" if n[1] > 0 else "-v"


def plane(m, p):
    """the half-space m . q <= m . p, m in tile units, p = (u, v, z px), as the renderer's plane"""
    mu, mv, mz = m
    return ((mu, mv, mz / ZK), mu * p[0] + mv * p[1] + mz * p[2] / ZK, face_name(m))


def obox(c, axes, half):
    """a box centred on c (u, v, z px) along three orthonormal axes (tile units), half sizes in tiles"""
    out = []
    for a, h in zip(axes, half):
        a = np.array(a, float)
        a /= np.linalg.norm(a)
        for sg in (1, -1):
            p = (c[0] + sg * h * a[0], c[1] + sg * h * a[1], c[2] + sg * h * a[2] * ZK)
            out.append(plane(sg * a, p))
    return out


def rod(p0, p1, w, d=None):
    """a square rod from p0 to p1 (u, v, z px), w tiles across; d tiles across the other way"""
    a = np.array([p1[0] - p0[0], p1[1] - p0[1], (p1[2] - p0[2]) / ZK])
    ln = np.linalg.norm(a)
    a /= ln
    e1 = np.cross(a, (0, 0, 1))
    e1 = e1 / np.linalg.norm(e1) if np.linalg.norm(e1) > 1e-6 else np.array([1., 0, 0])
    e2 = np.cross(e1, a)
    c = ((p0[0] + p1[0]) / 2, (p0[1] + p1[1]) / 2, (p0[2] + p1[2]) / 2)
    return obox(c, (a, e1, e2), (ln / 2, w / 2, (d or w) / 2))


def roll(u0, u1, vc, zc, rv, rz):
    """an octagonal prism along u: a bolt of cloth, rv tiles and rz pixels round"""
    out = [((1, 0, 0), u1, "+u"), ((-1, 0, 0), -u0, "-u")]
    for k in range(8):
        t = k * math.pi / 4
        cv, sz = math.cos(t), math.sin(t)
        out.append(((0, cv / rv, sz / rz), cv * vc / rv + sz * zc / rz + 1, "b%d" % k))
    return out


def tone(f):
    return "top" if f in ("+z", "-z") else "lit" if f.endswith("v") else "shade"


# ---------------------------------------------------------------- the coin chest

CU, CV, CZ = .36, .22, 9   # the chest's half width (u), half depth (v), rim height


def lacquer_chest(u0, u1, v0, v1, z0, z1, body=("QA", "Qa", "QT"), bands=True, lock=True, lid=3):
    """a closed chest: lacquer body, a lid line `lid` pixels under the top, two iron bands over it, a lock on the front"""
    lk, sk, tk = body
    w = u1 - u0

    def paint(f, u, v, z, s, h):
        t = tone(f)
        if t == "top":
            band = bands & (np.abs(np.abs(u - (u0 + u1) / 2) - w * .28) < .035)
            return np.where(band, "QN", tk)
        a = u - u0 if f.endswith("v") else v - v0
        keys = np.full(u.shape, lk if t == "lit" else sk)
        keys = np.where(z >= z1 - lid, tk if t == "lit" else lk, keys)       # the lid's lip
        if bands and f == "+v":
            keys = np.where(np.abs(np.abs(u - (u0 + u1) / 2) - w * .28) < .035, "QN", keys)
        if lock and f == "+v":   # a bronze hasp over the lid line, a dark keyhole dot
            m = np.abs(u - (u0 + u1) / 2) < .06
            keys = np.where(m & (z > z1 - lid - 2.5) & (z < z1 - lid + .5), "QY", keys)
            keys = np.where(m & (np.floor(16 * (u - u0)) == np.floor(16 * w / 2)) & (np.floor(z) == np.floor(z1 - lid - 2)), "QK", keys)
        if f == "+u":
            keys = np.where((np.abs(a - (v1 - v0) / 2) < .04) & (z > z1 - lid - 3) & (z < z1 - lid), "Qy", keys)
        return keys
    o = .025
    return [pc.Body(pc.box(u0, u1, v0, v1, z0, z1 - lid), paint), pc.Body(pc.box(u0 - o, u1 + o, v0 - o, v1 + o, z1 - lid, z1), paint)]


def coins(f, u, v, z, s, h):
    """a heap of strung coins: bronze dots, a few holes and glints, a red cord through some strings"""
    X = np.floor(16 * (u - v) + 1e-6).astype(int) + 64
    Y = np.floor(8 * (u + v) - z + 1e-6).astype(int) + 64
    cy = Y // 2
    cx, lx, ly = (X + 2 * (cy % 2)) // 3, (X + 2 * (cy % 2)) % 3, Y % 2
    k = (cx * 7 + cy * 5) % 9
    shade = tone(f) == "shade"
    body = np.where(shade | (k < 3), "QC", "QD")
    keys = np.where(lx == 2, np.where(ly == 0, "QK", "QC"), np.where((lx == 0) & (ly == 0) & ~shade, "QH", body))
    cord = (k == 4) & (ly == 1)
    return np.where(cord, "Qr" if shade else "QR", keys)


def coin_chest():
    B = pc.Body
    out = []

    def box_paint(f, u, v, z, s, h):
        t = tone(f)
        if t == "top":   # the rim: the lacquer's edge, the red inside wall just within it
            e = np.minimum(np.minimum(u + CU, CU - u), np.minimum(v + CV, CV - v))
            return np.where(e < .05, "QT", np.where(e < .1, "QI", "QC"))
        keys = np.full(u.shape, "QA" if t == "lit" else "Qa")
        keys = np.where(z > CZ - 1.5, "QT" if t == "lit" else "QA", keys)   # the rim's moulding
        keys = np.where(z < 1.5, "QK" if t == "lit" else "Qn", keys)       # the plinth
        if f == "+v":
            keys = np.where((np.abs(np.abs(u) - .2) < .035) & (z >= 1.5), "QN", keys)
            keys = np.where((np.abs(u) < .05) & (z > CZ - 5) & (z < CZ - 1), "QY", keys)
        if f == "+u":
            keys = np.where((np.abs(v) < .04) & (z > CZ - 5) & (z < CZ - 2), "Qy", keys)
        return keys
    out.append(B(pc.box(-CU, CU, -CV, CV, 0, CZ), box_paint))
    # the coins: a low mound over the inside, rising past the rim
    out.append(B(pc.Line("u").hip(-CU + .09, CU - .09, -CV + .08, CV - .08, CZ - 1, 44, top=CZ + 5), coins))
    # the lid: hinged on the back edge of the rim, swung back past upright, its inside (red) toward us
    phi = math.radians(14)
    up = (0, -math.sin(phi), math.cos(phi))
    inner = (0, math.cos(phi), math.sin(phi))
    L, T = 2 * CV + .02, .09
    c = (0, -CV + up[1] * L / 2 - inner[1] * T / 2, CZ + (up[2] * L / 2 - inner[2] * T / 2) * ZK)

    def lid_paint(f, u, v, z, s, h):
        if f == "+v":    # the inside: red, framed by the lacquer's edge
            edge = (np.abs(u) > CU - .06) | (z > CZ + L * ZK * up[2] - 2)
            return np.where(edge, "QT", "QI")
        return np.full(u.shape, "Qa" if tone(f) == "shade" else "QT")
    out.append(B(obox(c, ((1, 0, 0), up, inner), (CU + .01, L / 2, T / 2)), lid_paint))
    # a string of coins spilled on the ground in front
    out.append(B(rod((-.26, CV + .14, 1), (.12, CV + .2, 1), .06, .1), coins))
    return out


# ---------------------------------------------------------------- the stacked chests

def chests():
    return (lacquer_chest(-.46, -.02, -.3, .26, 0, 9) + lacquer_chest(.04, .44, -.26, .2, 0, 8, body=("QB", "Qb", "QU"), lid=2)
            + lacquer_chest(-.3, .26, -.26, .14, 9, 16, lid=2))
    return out


# ---------------------------------------------------------------- the tools

def wood(f, u, v, z, s, h):
    t = tone(f)
    if t == "top":
        return np.full(u.shape, "QX")
    return np.where(np.mod(h, 7) == 3, "QG", "QW" if t == "lit" else "Qw")


def iron(f, u, v, z, s, h):
    t = tone(f)
    return np.full(u.shape, "QE" if t == "top" else "QS" if t == "lit" else "Qs")


def tools():
    B = pc.Body
    out = []
    # the rack: two posts on feet, a bar near the top
    for a in (-.42, .34):
        out.append(B(pc.box(a, a + .08, -.32, -.24, 0, 25), wood))
        out.append(B(pc.box(a - .02, a + .1, -.42, -.14, 0, 2), wood))
    out.append(B(pc.box(-.46, .46, -.32, -.24, 20, 23), wood))
    # leaning on the bar, feet out in front: two hoes, blade up and bent forward; a spade, blade down
    for a in (-.3, .2):
        out.append(B(rod((a, .2, 0), (a, -.2, 29), .08), wood))
        out.append(B(rod((a, -.22, 29), (a, -.02, 24), .18, .04), iron))
    out.append(B(rod((-.05, .16, 7), (-.05, -.2, 30), .08), wood))
    out.append(B(rod((-.05, .24, 0), (-.05, .15, 9), .22, .04), iron))
    out.append(B(rod((-.13, -.2, 30), (.03, -.2, 30), .06), wood))       # the spade's grip
    # a sickle hung on the bar by the +u post: a short handle down, the blade curving forward and up
    a = .32
    out.append(B(rod((a, -.2, 21), (a, -.2, 15), .07), wood))
    out.append(B(rod((a, -.19, 15), (a - .04, -.06, 11), .04, .04), iron))
    out.append(B(rod((a - .04, -.06, 11), (a - .1, .04, 14), .04, .04), iron))
    return out


# ---------------------------------------------------------------- the silk

def bolt(colour):
    lk, sk, tk = "S" + colour, "s" + colour, "t" + colour

    def paint(f, u, v, z, s, h):
        if f in ("+u", "-u"):  # the end: rings of the wound cloth
            r = np.sqrt(((v - paint.vc) / paint.rv) ** 2 + ((z - paint.zc) / paint.rz) ** 2)
            return np.where(np.mod(np.floor(r * 4), 2) == 1, sk, lk) if f == "+u" else np.full(u.shape, sk)
        side = {"b0": lk, "b1": tk, "b2": tk, "b3": lk}.get(f, sk)       # b0 faces +v, b2 up, b7 down-front
        band = np.abs(np.abs(u - paint.uc) - .21) < .035
        return np.where(band, "QG", side)   # hemp ties
    return paint


def silk():
    out = []
    rv, rz = .1, 3.4
    lay = [(-.21, rz, "1", -.4, .36), (-.01, rz, "2", -.38, .38), (.19, rz, "3", -.42, .34),
           (-.11, 3 * rz - .3, "4", -.36, .4), (.09, 3 * rz - .3, "5", -.4, .36)]
    for vc, zc, colour, u0, u1 in lay:
        p = bolt(colour)
        p.vc, p.zc, p.rv, p.rz, p.uc = vc, zc, rv, rz, (u0 + u1) / 2
        out.append(pc.Body(roll(u0, u1, vc, zc, rv, rz), p))
    return out


# ---------------------------------------------------------------- the kit

def one(bodies):
    bodies = bodies + [pc.Body(pc.box(1, 0, 1, 0, 1, 0), None)]   # empty: render reads its extent from box planes
    a, own, o = pc.render(bodies, along=lambda u, v: np.zeros(u.shape), bounds=(-40, -50, 40, 30))
    return pc.piece(a, own, o, 0, (0, 0))


def kit():
    return {"hoard_coin_chest": one(coin_chest()), "hoard_chests": one(chests()),
            "hoard_tools": one(tools()), "hoard_silk": one(silk())}


MADE = "원술's hoard (17장), pixel art on the small-tile grid (gen_pixel_hoard.py)"


def write():
    here = os.path.dirname(os.path.abspath(__file__))
    tiles_dir = os.path.join(here, "..", "tiles")
    path = os.path.join(tiles_dir, "tiles.json")
    tiles = json.load(open(path))
    for name, (img, pivot) in kit().items():
        img.save(os.path.join(tiles_dir, name + ".png"))
        tiles["sprites"][name] = {"file": f"tiles/{name}.png", "size": list(img.size), "pivot": list(pivot),
                                  "layer": "object", "scale": "scene", "made": MADE}
    json.dump(tiles, open(path, "w"), ensure_ascii=False, indent=1)
    open(path, "a").write("\n")


def preview(out):
    os.makedirs(out, exist_ok=True)
    tiles_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "tiles")
    man = json.load(open(os.path.join(tiles_dir, "tiles.json")))["sprites"]
    k = kit()
    for name in ("grain_sack", "water_jar"):
        k[name] = (Image.open(os.path.join(tiles_dir, man[name]["file"].split("/")[-1])).convert("RGBA"), tuple(man[name]["pivot"]))
    from PIL import ImageDraw
    names = ["grain_sack", "water_jar", "hoard_coin_chest", "hoard_chests", "hoard_tools", "hoard_silk"]
    cv = Image.new("RGBA", (56 * len(names) + 16, 80), (118, 128, 96, 255))
    d = ImageDraw.Draw(cv)
    for n, name in enumerate(names):
        img, (px, py) = k[name]
        x, y = 36 + 56 * n, 60
        d.polygon([(x - 16, y), (x, y - 8), (x + 16, y), (x, y + 8)], outline=(96, 106, 78))
        cv.alpha_composite(img, (x - px, y - py))
    cv.resize((cv.width * 4, cv.height * 4), Image.NEAREST).save(os.path.join(out, "hoard.png"))


if __name__ == "__main__":
    if sys.argv[1:2] == ["--write"]:
        write()
    elif sys.argv[1:2] == ["--preview"]:
        preview(sys.argv[2])
    else:
        print(__doc__)
