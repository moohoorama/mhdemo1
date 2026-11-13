"""The palace front of 허도 for the scenes, drawn as pixel art on the small-tile grid (the castle kit's renderer,
gen_pixel_castle.render, with its own stone, plaster and roof tiles): the palace wall and the great gate in it.

  pwall_u        one small tile of the palace wall running along u: a granite footing, vermilion plaster, a dark band
                 and a grey tiled coping. Pivot at the tile's centre on the ground; the wall stands on the tile's
                 middle line (v), 0.6 tile thick. Drawn beside its neighbours so a run meets without a seam.
  mansion_cao    조조's house: a walled compound 13 x 9 small tiles (white walls, a tall gate with studded leaves, a
                 two-storey main hall and two side wings). scene.json footprint [-6.5, -4.5, 13, 9]; its gate on the
                 +v side, in the middle.
  mansion_liu    the house given to 유비: 9 x 7 (ochre walls, a plain gate, one hall). Footprint [-4.5, -3.5, 9, 7].
  palace_gate    the gate hall (궁문): a granite terrace 9 × 5 small tiles, a three-bay hall of vermilion columns and
                 walls with three studded doors (the middle one wider), a balcony storey and a two-tier hipped roof,
                 steps down the front. Pivot at the middle tile's centre on the ground; scene.json footprint
                 [-4.5, -2.5, 9, 5]. Meant to stand on the wall's line with the walls running out from its sides.

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


def wall_bodies(u0, u1, tile):
    foot, top, t = WALL
    B = pc.Body
    return [B(pc.box(u0, u1, .5 - t - .04, .5 + t + .04, 0, foot), granite, tile),
            B(pc.box(u0, u1, .5 - t, .5 + t, foot, top), plaster(top), tile),
            B(hip(u0 - 3, u1 + 3, .5 - t - .22, .5 + t + .22, top - 2, 22, top=top + 9), roof(top - 2), tile),
            B(pc.box(u0 - 3, u1 + 3, .44, .56, top + 7, top + 11), ridge, tile)]


def wall():
    """the tile u in [0, 1], v in [0, 1]; a long wall is drawn and the pixels of tile 0 kept (along = floor(u))"""
    bodies = wall_bodies(-3, 4, 0)
    a, own, o = pc.render(bodies, along=lambda u, v: np.floor(u - 1e-6), bounds=(-60, -70, 60, 40))
    return pc.piece(a, own, o, 0, (16 * (.5 - .5), 8 * (.5 + .5)))


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


def gate():
    a, own, o = pc.render(gate_bodies(), along=lambda u, v: np.zeros(u.shape), bounds=(-170, -200, 170, 110))
    return pc.piece(a, own, o, 0, (0, 0))


# ---------------------------------------------------------------- the great houses

pc.PAL.update({
    # white plaster (조조) and pale ochre plaster (유비) walls; black lacquer posts
    "HW": (214, 208, 194), "Hw": (164, 158, 146), "HO": (198, 178, 140), "Ho": (150, 132, 102),
    "HL": (52, 44, 40), "Hl": (36, 30, 28),
})


def walls_paint(lit_key, shade_key, top):
    def paint(f, u, v, z, s, h):
        lit = pc.lit(f) or f == "+z"
        foot = z < 4
        return np.where(foot, "PG" if lit else "Pg", np.where(z >= top - 2, "PB" if lit else "Pb", lit_key if lit else shade_key))
    return paint


def post(f, u, v, z, s, h):
    return np.full(u.shape, "HL" if pc.lit(f) or f == "+z" else "Hl")


def coping(line, a0, a1, c, t, z, tile=None):
    """a tiled coping along a wall: a low hip over the wall's top and a dark ridge"""
    B = pc.Body
    return [B(line.hip(a0 - .2, a1 + .2, c - t - .2, c + t + .2, z - 2, 18, top=z + 5), roof(z - 2), tile),
            B(line.box(a0 - .2, a1 + .2, c - .06, c + .06, z + 3, z + 7), ridge, tile)]


def house_bodies(hw, hd, wall_z, wall_keys, gate_w, halls, grand):
    """a walled compound hw x hd half sizes round the anchor: the enclosure (gap in the front for the gate), the gate
    (posts, red leaves, its own roof), and the halls inside: (u0, u1, v0, v1, eave, keys, two_tier)"""
    B = pc.Body
    t = .2
    U, V = pc.Line("u"), pc.Line("v")
    wp = walls_paint(*wall_keys, wall_z)
    out = []
    # enclosure: back (-v), sides (-u, +u), front (+v) either side of the gate
    out += [B(pc.box(-hw, hw, -hd, -hd + 2 * t, 0, wall_z), wp)] + coping(U, -hw, hw, -hd + t, t, wall_z)
    for a in (-hw, hw - 2 * t):
        out += [B(pc.box(a, a + 2 * t, -hd, hd, 0, wall_z), wp)] + coping(V, -hd, hd, a + t, t, wall_z)
    g = gate_w / 2
    for a0, a1 in ((-hw, -g), (g, hw)):
        out += [B(pc.box(a0, a1, hd - 2 * t, hd, 0, wall_z), wp)] + coping(U, a0, a1, hd - t, t, wall_z)
    # the gate: two posts, the leaves, a lintel, its roof standing proud of the walls
    gz = wall_z + (16 if grand else 10)
    for a in (-g, g - .3):
        out.append(B(pc.box(a, a + .3, hd - .5, hd + .1, 0, gz - 4), post))
    out.append(B(pc.box(-g + .3, g - .3, hd - .3, hd - .1, 0, gz - 8), front(gz - 4, [(-g + .3, g - .3)])))
    out.append(B(pc.box(-g, g, hd - .5, hd + .1, gz - 8, gz - 4), post))
    out.append(B(hip(-g - .5, g + .5, hd - 1.0, hd + .6, gz - 4, 20, top=gz + 8), roof(gz - 4)))
    out.append(B(pc.box(-g - .2, g + .2, hd - .26, hd - .14, gz + 6, gz + 11), ridge))
    # the halls inside
    for u0, u1, v0, v1, ez, keys, two in halls:
        plain = walls_paint(*keys, ez)
        door = front(ez, [((u0 + u1) / 2 - .7, (u0 + u1) / 2 + .7)], plain)

        def hall_paint(f, u, v, z, s, h, plain=plain, door=door, mid=(u0 + u1) / 2, ez=ez):
            return door(f, u, v, z, s, h) if f == "+v" else plain(f, u, v, z, s, h)
        out.append(B(pc.box(u0, u1, v0, v1, 0, ez), hall_paint))
        for a in (u0 + .1, u1 - .4):
            out.append(B(pc.box(a, a + .3, v1 + .05, v1 + .35, 0, ez), post))
        slope = 26 if two else min(30, 16 / max(.5, (v1 - v0) / 2 + .5) * 3)
        out.append(B(hip(u0 - .5, u1 + .5, v0 - .5, v1 + .5, ez - 3, slope, top=ez + 10 if two else None), roof(ez - 3)))
        if two:
            out.append(B(pc.box(u0 + .8, u1 - .8, v0 + .6, v1 - .6, ez + 10, ez + 26), lattice(ez + 26)))
            out.append(B(hip(u0 + .3, u1 - .3, v0 + .1, v1 - .1, ez + 23, 34), roof(ez + 23)))
    return out


def house(name):
    if name == "mansion_cao":    # 13 x 9: white walls, a tall gate, a two-storey main hall and two side wings
        hw, hd = 6.5, 4.5
        bodies = house_bodies(hw, hd, 30, ("HW", "Hw"), 3.0, [
            (-3.2, 3.2, -3.6, -.6, 52, ("PR", "Pr"), True),
            (-5.8, -4.0, -3.4, 2.6, 36, ("PR", "Pr"), False),
            (4.0, 5.8, -3.4, 2.6, 36, ("PR", "Pr"), False)], grand=True)
    else:                        # 9 x 7: ochre walls, a plain gate, one hall
        hw, hd = 4.5, 3.5
        bodies = house_bodies(hw, hd, 24, ("HO", "Ho"), 2.0, [
            (-2.6, 2.6, -2.7, -.2, 40, ("HO", "Ho"), False)], grand=False)
    a, own, o = pc.render(bodies, along=lambda u, v: np.zeros(u.shape), bounds=(-260, -200, 260, 180))
    return pc.piece(a, own, o, 0, (0, 0))


# ---------------------------------------------------------------- the kit

def kit():
    return {"pwall_u": wall(), "palace_gate": gate(), "mansion_cao": house("mansion_cao"), "mansion_liu": house("mansion_liu")}


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
    # the wall's line along v = 0 from u = -12 to 12, the gate on u = 0; drawn back to front
    items = [("pwall_u", u, 0) for u in range(-12, 13) if abs(u) > 4] + [("palace_gate", 0, 0)]
    for name, u, v in sorted(items, key=lambda t: (t[1] + t[2] + (7 if t[0] == "palace_gate" else 1))):
        lay(name, u, v)
    cv.resize((cv.width * 2, cv.height * 2), Image.NEAREST).save(os.path.join(out, "layout.png"))


if __name__ == "__main__":
    if sys.argv[1:2] == ["--write"]:
        write()
    elif sys.argv[1:2] == ["--preview"]:
        preview(sys.argv[2])
    else:
        print(__doc__)
