"""The palace front of 허도 for the scenes, drawn as pixel art on the small-tile grid (the castle kit's renderer,
gen_pixel_castle.render, with its own stone, plaster and roof tiles): the palace wall and the great gate in it.

  pwall_u        one small tile of the palace wall running along u: a granite footing, vermilion plaster, a dark band
                 and a grey tiled coping. Pivot at the tile's centre on the ground; the wall stands on the tile's
                 middle line (v), 0.6 tile thick. Drawn beside its neighbours so a run meets without a seam.
  pwall_u_end_m, pwall_u_end_p   the tile where the wall stops at its -u / +u side: the end face flush with the
                 tile's side, the coping hipped down onto it (nothing reaches past the tile).
  pwall_u_gate_m, pwall_u_gate_p the wall tiles against the gate hall's terrace (the gate's tile -5 / +5 along u):
                 drawn together with the hall, each keeps only what is nearer than the hall, so the hall and the wall
                 meet exactly whichever the game draws first.
  palace_gate    the gate hall (궁문): a granite terrace 9 × 5 small tiles, a three-bay hall of vermilion columns and
                 walls with three studded doors (the middle one wider), a balcony storey and a two-tier hipped roof,
                 steps down the front. Pivot at the middle tile's centre on the ground; scene.json footprint
                 [-4.5, -2.5, 9, 5]. Stands on the wall's line: drawn with the wall running across its terrace into
                 the hall's side walls (that bit of wall is part of the gate's picture), pwall_u_gate_* either side.

  python3 gen_pixel_palace.py --preview DIR   # the pieces at 4x and a gate with walls laid out, into DIR
  python3 gen_pixel_palace.py --write         # save them into ../tiles and their entries of tiles.json
"""
import json
import os
import sys

import numpy as np
from PIL import Image

import gen_pixel_castle as pc

pc.PAL.update({
    # granite: lit (+v), shaded (+u), joints, top
    "PG": (170, 166, 156), "Pg": (128, 124, 118), "PJ": (120, 116, 108), "Pj": (90, 88, 84), "PU": (188, 184, 174),
    # vermilion plaster and columns
    "PR": (164, 58, 44), "Pr": (122, 40, 32), "PC": (150, 44, 34), "Pc": (108, 30, 26),
    "PB": (58, 40, 34), "Pb": (44, 30, 26),                  # the dark band under the eaves
    # roof tiles: lit / shaded rows, the eave's end tiles, ridge
    "PT": (84, 92, 96), "PF": (70, 78, 82), "Pt": (60, 66, 70), "Pf": (50, 56, 60), "PE": (104, 112, 116),
    "Pe": (78, 84, 88), "PK": (36, 38, 42),
    # doors: red leaves, gold studs, dark frames; lattice windows
    "PD": (138, 34, 28), "Pd": (104, 26, 22), "PY": (214, 176, 84), "PX": (40, 26, 22), "PW": (196, 180, 140),
})


def granite(f, u, v, z, s, h):
    if f == "+z":
        return np.where((np.mod(np.floor(16 * u), 12) == 11) | (np.mod(np.floor(16 * v), 12) == 11), "PJ", "PU")
    return pc.brick(s, h, "PG", "PJ", w=12, rows=3) if pc.lit(f) else pc.brick(s, h, "Pg", "Pj", w=12, rows=3)


def plaster(top, band=3):
    def paint(f, u, v, z, s, h):
        lit = pc.lit(f) or f == "+z"
        return np.where(z >= top - band, "PB" if lit else "Pb", "PR" if lit else "Pr")
    return paint


def column(f, u, v, z, s, h):
    return np.full(u.shape, "PC" if pc.lit(f) or f == "+z" else "Pc")


def roof(eave_z):
    def paint(f, u, v, z, s, h):
        if f == "+z":
            return np.where(np.mod(np.floor(16 * u), 4) == 3, "PX", "PB")
        lit = f.endswith("v")
        a, b, end = ("PT", "PF", "PE") if lit else ("Pt", "Pf", "Pe")
        rows = np.where(np.mod(s, 4) == 3, b, a)
        return np.where(z < eave_z + 2, np.where(np.mod(s, 2) == 0, end, "PK"), rows)
    return paint


def ridge(f, u, v, z, s, h):
    return np.full(u.shape, "PK" if f != "+z" else "Pe")


def hip(u0, u1, v0, v1, zb, slope, top=None):
    return pc.Line("u").hip(u0, u1, v0, v1, zb, slope, top)


# ---------------------------------------------------------------- the wall

WALL = (8, 40, .3)         # footing height, plaster top, half thickness (tiles)


def wall_bodies(u0, u1, c=.5, tile=None, end0=False, end1=False):
    """the wall from u0 to u1 on the line v = c. A run's coping reaches past both ends (its pieces cut it by tile); an
    end (end0 at u0, end1 at u1) stops flush with the wall's end face, the coping hipped down onto it"""
    foot, top, t = WALL
    B = pc.Body
    a0, a1 = (u0 if end0 else u0 - 3), (u1 if end1 else u1 + 3)
    r0, r1 = (u0 + .45 if end0 else u0 - 3), (u1 - .45 if end1 else u1 + 3)
    return [B(pc.box(u0, u1, c - t - .04, c + t + .04, 0, foot), granite, tile),
            B(pc.box(u0, u1, c - t, c + t, foot, top), plaster(top), tile),
            B(hip(a0, a1, c - t - .22, c + t + .22, top - 2, 22, top=top + 9), roof(top - 2), tile),
            B(pc.box(r0, r1, c - .06, c + .06, top + 7, top + 11), ridge, tile)]


def wall(end=None):
    """the tile u in [0, 1], v in [0, 1] of a run (cut out of a long wall: along = floor(u)); end "m" or "p": the
    tile where the wall stops at its -u or +u side"""
    u0, u1 = {None: (-3, 4), "m": (0, 4), "p": (-3, 1)}[end]
    bodies = wall_bodies(u0, u1, end0=end == "m", end1=end == "p")
    a, own, o = pc.render(bodies, along=lambda u, v: np.floor(u - 1e-6), bounds=(-60, -70, 60, 40))
    return pc.piece(a, own, o, 0, (0, 8))


# ---------------------------------------------------------------- the gate hall

HW, HD = 4.5, 2.5          # the terrace's half width (u) and half depth (v)
Z0, Z1, Z2 = 10, 56, 98    # terrace top, first eave, second eave


def front(top, doors, base=None):
    """the hall's walls: on the front (+v) face the doors at the given (u0, u1) spans, studded, framed; the sides
    plain plaster"""
    base = base or plaster(top, band=4)

    def paint(f, u, v, z, s, h):
        keys = base(f, u, v, z, s, h)
        if f != "+v":
            return keys
        for a0, a1 in doors:
            inside = (u >= a0) & (u <= a1) & (z < top - 10)
            frame = inside & ((u - a0 < .09) | (a1 - u < .09) | (z >= top - 13))
            mid = inside & (np.abs(u - (a0 + a1) / 2) < .04)
            stud = inside & (np.mod(np.floor(16 * (u - a0)), 3) == 1) & (np.mod(np.floor(z), 5) == 2)
            leaf = np.where(stud, "PY", np.where(np.mod(np.floor(16 * u), 4) == 0, "Pd", "PD"))
            keys = np.where(inside, np.where(frame | mid, "PX", leaf), keys)
        return keys
    return paint


def lattice(top):
    base = plaster(top, band=4)

    def paint(f, u, v, z, s, h):
        keys = base(f, u, v, z, s, h)
        if f not in ("+v", "+u"):
            return keys
        a = u if f == "+v" else v
        win = (np.mod(a + 10, 1.4) > .25) & (z > top - 20) & (z < top - 7)
        grid = (np.mod(np.floor(16 * a), 3) == 0) | (np.mod(np.floor(z), 3) == 0)
        return np.where(win, np.where(grid, "PX", "PW"), keys)
    return paint


def gate_bodies():
    B = pc.Body
    out = [B(pc.box(-HW, HW, -HD, HD, 0, Z0), granite)]
    # steps down the front, the middle bay's width
    for i, (d, z) in enumerate(((.35, 7), (.7, 4))):
        out.append(B(pc.box(-1.6, 1.6, HD, HD + d, 0, z), granite))
    # first storey: the hall set back, its three doors, columns along the front
    out.append(B(pc.box(-HW + .8, HW - .8, -HD + .9, HD - 1.0, Z0, Z1), front(Z1, [(-1.1, 1.1), (-3.0, -1.9), (1.9, 3.0)])))
    for a in (-HW + .55, -1.5, 1.15, HW - .9):
        out.append(B(pc.box(a, a + .35, HD - .8, HD - .45, Z0, Z1), column))
    out.append(B(hip(-HW - .4, HW + .4, -HD - .4, HD + .4, Z1 - 3, 14, top=Z1 + 12), roof(Z1 - 3)))
    # second storey: a narrower hall with lattice windows
    out.append(B(pc.box(-HW + 1.9, HW - 1.9, -HD + 1.5, HD - 1.5, Z1 + 6, Z2), lattice(Z2)))
    for a in (-HW + 1.7, HW - 2.05):
        out.append(B(pc.box(a, a + .3, HD - 1.55, HD - 1.25, Z1 + 6, Z2), column))
    # the top roof: a skirt, a steep hip, the ridge with raised ends
    out.append(B(hip(-HW + 1.2, HW - 1.2, -HD + .8, HD - .8, Z2 - 3, 16, top=Z2 + 5), roof(Z2 - 3)))
    out.append(B(hip(-HW + 1.5, HW - 1.5, -HD + 1.05, HD - 1.05, Z2 + 3, 40), roof(-99)))
    zr = Z2 + 3 + 40 * (HD - 1.05)
    ra = HW - 1.5 - (HD - 1.05)
    out.append(B(pc.box(-ra, ra, -.12, .12, zr - 3, zr + 2), ridge))
    for a in (-ra, ra - .3):
        out.append(B(pc.box(a, a + .3, -.16, .16, zr - 3, zr + 9), ridge))
    return out


GATE = 99                  # the gate hall's owner in the joint render; the wall tiles own their tile index


def gate_front():
    """the gate hall and the wall running into its sides, drawn together so every pixel belongs to whichever is
    nearer: the hall (with the bits of wall inside its terrace, tiles -4..4 along u) and the wall tiles either side
    of it (-5, +5), whose pictures leave out what the hall hides. Coordinates round the gate's tile centre; tile k of
    the line spans u in [k - .5, k + .5]."""
    hall = gate_bodies()
    for b in hall:
        b.tile = GATE
    side = HW - .8         # the hall's side walls; the wall runs up to them across the terrace
    bodies = hall + wall_bodies(-9, -side, c=0, end1=True) + wall_bodies(side, 9, c=0, end0=True)
    a, own, o = pc.render(bodies, along=lambda u, v: np.floor(u + .5), bounds=(-200, -240, 200, 110))
    hall_own = np.where((own == GATE) | ((own >= -4) & (own <= 4)), 0, -1)
    out = {"palace_gate": pc.piece(a, hall_own, o, 0, (0, 0))}
    for k, name in ((-5, "pwall_u_gate_m"), (5, "pwall_u_gate_p")):
        out[name] = pc.piece(a, own, o, k, (16 * k, 8 * k))
    return out


# ---------------------------------------------------------------- the kit

def kit():
    out = {"pwall_u": wall(), "pwall_u_end_m": wall("m"), "pwall_u_end_p": wall("p")}
    out.update(gate_front())
    return out


MADE = "palace front of 허도, pixel art on the small-tile grid (gen_pixel_palace.py)"


def write():
    here = os.path.dirname(os.path.abspath(__file__))
    tiles_dir = os.path.join(here, "..", "tiles")
    path = os.path.join(tiles_dir, "tiles.json")
    tiles = json.load(open(path))
    for name, (img, pivot) in kit().items():
        img.save(os.path.join(tiles_dir, name + ".png"))
        tiles["sprites"][name] = {"file": f"tiles/{name}.png", "size": list(img.size), "pivot": list(pivot),
                                  "layer": "object", "scale": "scene", "made": MADE, "shadow": False}
    json.dump(tiles, open(path, "w"), ensure_ascii=False, indent=1)
    open(path, "a").write("\n")


def preview(out):
    os.makedirs(out, exist_ok=True)
    k = kit()
    for name, (img, _) in k.items():
        img.resize((img.width * 4, img.height * 4), Image.NEAREST).save(os.path.join(out, name + ".png"))
    cv = Image.new("RGBA", (640, 360), (150, 146, 132, 255))
    ox, oy = 320, 200

    def lay(name, u, v):           # u, v: the small tile; its centre at (u + .5, v + .5)
        img, (px, py) = k[name]
        x, y = ox + 16 * (u - v), oy + 8 * (u + v + 1)
        cv.alpha_composite(img, (int(x - px), int(y - py)))
    # the wall's line along v = 0 from u = -12 to 12 with its ends, the gate on u = 0; drawn back to front as the
    # game orders them: the tiles behind the gate's footprint (u < -4), the gate, the tiles in front (u > 4)
    def piece_at(u):
        return {-12: "pwall_u_end_m", 12: "pwall_u_end_p", -5: "pwall_u_gate_m", 5: "pwall_u_gate_p"}.get(u, "pwall_u")
    for u in range(-12, -4):
        lay(piece_at(u), u, 0)
    lay("palace_gate", 0, 0)
    for u in range(5, 13):
        lay(piece_at(u), u, 0)
    cv.resize((cv.width * 2, cv.height * 2), Image.NEAREST).save(os.path.join(out, "layout.png"))


if __name__ == "__main__":
    if sys.argv[1:2] == ["--write"]:
        write()
    elif sys.argv[1:2] == ["--preview"]:
        preview(sys.argv[2])
    else:
        print(__doc__)
