"""The top of a city wall for the scenes, drawn as pixel art on the small-tile grid (the castle kit's renderer,
gen_pixel_castle.render, with its own stone, bricks and shapes): the people stand on the wall walk, raised the wall's
height (scene.json "heights" "3"); its open front drops as the wall's brick face with the low inner parapet on its edge,
its back (the outer side) carries the crenellated battlements, and the gate tower stands on it.

  rdwall_<mask>_<k>  the same faces dry, without the flood's mark (a wall no flood has reached; 영천 장사성).
  rwall_<mask>_<k>   the face of a raised small tile, mask 4 (+u side drops) / 8 (+v) / 12 (both), k = 0..2: the
                     wall's face from the ground (z = 0) to the walk (LIFT), pivot at the tile's centre on the ground.
  rrim_<side>_<k>    what stands on a walk tile's edge over lower ground (scene.json heights "rim"), k = 0..3, pivot at
                     the tile's centre on the walk: on the front edges sw (+v) and se (+u) the low inner parapet, its
                     bricks running on from the face; on the back edges ne (-v) and nw (-u) the battlements.
  floor_rampart      the walk's paving, one game cell (glyph Q).
  baimen_tower       the gate tower: a two-storey hall on a stone plinth, standing on the walk.
  baimen_gate_<i>    the gate's passage where it opens in the wall's inner face, below the tower, in four slices one
                     small tile wide (i = 0 the leftmost): laid over the face as objects on the ground at the face's
                     foot, each anchored at its tile's middle on the face's line.

The bricks follow the wall's own heights and the tile's own dots, so tiles meet without a seam whatever variant stands
beside them; the variants differ only in wear kept off the tile's sides (cracks, moss, stains, a chipped merlon).

  python3 gen_pixel_rampart.py --preview DIR   # every piece at 4x, a run of faces, and a laid-out wall top, into DIR
  python3 gen_pixel_rampart.py --write         # save them into ../tiles and their entries of tiles.json
"""
import json
import math
import os
import sys

import numpy as np
from PIL import Image

import gen_pixel_castle as pc

LIFT = 120                 # the walk's height over the ground below, pixels
PARAPET = (15, 3, .2)      # inner parapet: plaster height, tiled cap height, thickness (tiles)
CORNICE = (4, .07)         # the stone band round the face's top: height, how far it stands out (tiles)
INSET = .06                # how far the parapet stands back from the face
BATTLE = (11, 10, .3)      # battlements: wall height, merlon height above it, thickness (tiles)
DAMP = 30                  # the flood's mark on the face, pixels up
HOLE = "wX"                # paint key of the walk's top in the face pictures: cut out after rendering
FACES, DRY_FACES, RIMS = "rwall", "rdwall", "rrim"

pc.PAL.update({HOLE: (1, 2, 3), "wH": (34, 32, 30), "wK": (58, 62, 54), "wM": (92, 108, 66), "wm": (74, 88, 54),
               "wT": (196, 192, 180), "wt": (150, 146, 136), "wE": (82, 84, 88), "we": (124, 126, 128)})


def shades(prefix, rgbs, joint):
    for i, c in enumerate(rgbs):
        pc.PAL[f"{prefix}{i}"] = c
    pc.PAL[prefix + "j"] = joint


# brick tones: lit (+v) and shaded (+u) faces, dry and below the flood's mark
shades("wL", [(164, 160, 150), (152, 148, 138), (176, 172, 160)], (108, 104, 96))
shades("wS", [(118, 114, 108), (108, 104, 98), (128, 124, 116)], (76, 72, 68))
shades("wD", [(116, 120, 106), (106, 110, 98), (126, 130, 114)], (76, 80, 70))
shades("wd", [(84, 88, 78), (76, 80, 72), (92, 96, 84)], (54, 58, 52))
pc.PAL.update({"wI": (190, 188, 174), "wi": (138, 136, 126)})   # the tide line, lit and shaded


def damp_line(s):
    """the flood's mark: a wavy line, periodic over a tile's 16 dots so it runs on across tiles"""
    m = np.mod(s, 16)
    return DAMP + np.round(2 * np.sin(2 * math.pi * m / 16)).astype(int) + (np.mod(m * 5, 7) == 0)


def bricks(s, h, lit_face, k=None, wet=True):
    """the wall's bricks: courses 4 px (3 dots and a joint), bricks 8 dots with a joint, every other course shifted
    half a brick; each brick's tone from its course and its place in the tile's pair, so a brick across two tiles is
    one tone. Below the flood's mark the bricks are darker, the mark itself a pale line (wet; a dry wall has none).
    k adds a variant's wear."""
    course = np.floor_divide(h, 4)
    off = np.mod(course, 2) * 4
    b = np.floor_divide(s + off, 8)
    joint = (np.mod(h, 4) == 3) | (np.mod(s + off, 8) == 7)
    t = np.mod(course * 7 + np.mod(b, 2) * 5 + np.floor_divide(course, 3) * 3, 6)
    tone = np.where(t < 3, 0, np.where(t < 5, 1, 2)).astype(str)
    dl = damp_line(s) if wet else np.full(np.shape(s), -99)
    dry, damp, tide = ("wL", "wD", "wI") if lit_face else ("wS", "wd", "wi")
    pre = np.where(h < dl, damp, dry)
    keys = np.where(joint, np.char.add(pre, "j"), np.char.add(pre, tone))
    keys = np.where(h == dl, tide, keys)
    if k is not None:
        keys = wear(keys, np.mod(s, 16), h, dl, k, np.char.add(pre, "j"), wet)
    return keys


def wear(keys, m, h, dl, k, joint, wet=True):
    """a variant's marks in the joints' tone, inside the tile's dots 3..12 so its sides stay plain"""
    if k == 0:      # a crack across a few courses
        crack = (h > 74) & (h < 88) & (m == 6 + np.mod(np.floor_divide(h, 4), 3))
        return np.where(crack, joint, keys)
    if k == 1:      # moss over the flood's mark (wet), a spalled brick higher up
        if wet:
            moss = ((m - 8) ** 2 / 9 + (h - dl - 3) ** 2 / 6) < 1
            keys = np.where(moss, np.where(np.mod(m + h, 3) == 0, "wm", "wM"), keys)
        else:
            keys = np.where((m >= 8) & (m <= 11) & (h >= 30) & (h <= 32), joint, keys)
        spall = (m >= 4) & (m <= 7) & (h >= 76) & (h <= 78)
        return np.where(spall, joint, keys)
    if k == 2:      # a stain weeping down from a drain hole under the parapet
        hole = (m >= 9) & (m <= 10) & (h >= LIFT - 8) & (h <= LIFT - 6)
        streak = (m == 9) & (h > LIFT - 22) & (h < LIFT - 8) & (np.mod(h * 7, 11) < 8)
        return np.where(hole, "wH", np.where(streak, joint, keys))
    return keys


# ---------------------------------------------------------------- the face


def face_paint(k, wet=True):
    def paint(f, u, v, z, s, h):
        if f == "+z":
            return np.full(u.shape, HOLE)
        return bricks(s, h, pc.lit(f), k, wet)
    return paint


def wall_block(u, v, tile, k=None, wet=True):
    return pc.Body(pc.box(u, u + 1, v, v + 1, 0, LIFT), face_paint(k, wet), tile)


FACE_BOUNDS = (-48, -LIFT - 40, 48, 40)


pc.PAL.update({"wC": (184, 180, 168), "wc": (134, 130, 120), "wW": (212, 206, 190), "ww": (160, 154, 140),
               "wB": (96, 92, 86)})


def cornice_paint(f, u, v, z, s, h):
    return np.full(u.shape, "wT" if f == "+z" else "wC" if pc.lit(f) else "wc")


def cornice(side, u, v, tile):
    """the stone band along side ('+v' or '+u') of tile (u, v), at the top of the face, standing out from it"""
    hc, out = CORNICE
    if side == "+v":
        box = pc.box(u, u + 1, v + 1 - .02, v + 1 + out, LIFT - hc, LIFT)
    else:
        box = pc.box(u + 1 - .02, u + 1 + out, v, v + 1, LIFT - hc, LIFT)
    return pc.Body(box, cornice_paint, tile)


def face(mask, k, wet=True):
    """tile (0, 0) whose sides in mask drop, beside the walk's tiles it meets so lines run on without a seam; the
    stone band along its dropping sides' tops, the neighbours' band beside it"""
    du, dv = mask & 4, mask & 8
    near = [(0, -1), (-1, -1)] + ([] if dv else [(-1, 1), (0, 1)]) + ([] if du else [(1, -1), (1, 0)])
    near += [(-1, 0)] if dv or not du else []
    bodies = [wall_block(0, 0, 0, k, wet)] + [wall_block(a, b, -1, k, wet) for a, b in set(near)]
    if dv:
        bodies += [cornice("+v", 0, 0, 0), cornice("+v", -1, 0, -1)] + ([] if du else [cornice("+v", 1, 0, -1)])
    if du:
        bodies += [cornice("+u", 0, 0, 0), cornice("+u", 0, -1, -1)] + ([] if dv else [cornice("+u", 0, 1, -1)])
    a, own, o = pc.render(bodies, bounds=FACE_BOUNDS)
    hole = np.all(a[..., :3] == pc.PAL[HOLE], axis=-1) & (a[..., 3] > 0)
    a[hole] = 0
    own = np.where(hole, -1, own)
    return pc.piece(a, own, o, 0, (0, 8))


# ---------------------------------------------------------------- the rims: inner parapet in front, battlements behind


def cap_paint(f, u, v, z, s, h):
    """the parapet's cap of dark tiles: rows along its top, round tile ends every two dots along its edges"""
    if f == "+z":
        return np.where(np.mod(np.floor(16 * (u + v)).astype(int), 3) == 0, "wK", "wE")
    lit_face = pc.lit(f)
    return np.where(np.mod(s, 2) == 0, "we" if lit_face else "wE", "wE" if lit_face else "wK")


def plaster_paint(f, u, v, z, s, h):
    """the parapet's whitewash over brick, a dark plinth line at its foot, grime streaks here and there"""
    lit_face = pc.lit(f)
    if f == "+z":
        return np.full(u.shape, "wW")
    base = np.where(np.mod(s * 7 + np.floor_divide(h, 3) * 3, 23) == 0, "wt", "wW" if lit_face else "ww")
    return np.where(h <= LIFT + 1, "wB", base)


def bat_paint(k, front):
    """bricks running on from the wall's courses; on the face toward the walk an arrow loop through each merlon"""
    hb, hm, _ = BATTLE

    def paint(f, u, v, z, s, h):
        if f == "+z":
            return np.full(u.shape, "wT")
        keys = bricks(s, h, pc.lit(f))
        m, r = np.mod(s, 16), h - LIFT
        if f == front:
            keys = np.where((m >= 7) & (m <= 8) & (r >= hb + 3) & (r <= hb + 6), "wH", keys)
            if k == 1:
                keys = np.where((m >= 4) & (m <= 5) & (r >= 3) & (r <= 5), "wK", keys)
            if k == 3:
                keys = np.where(((m - 11) ** 2 / 4 + (r - 2) ** 2 / 3) < 1, "wM", keys)
        return keys
    return paint


def strip(side, u, v, a0, a1, c0, c1, z0, z1):
    """a box along side's edge of tile (u, v): a0..a1 along the edge, c0..c1 in from it"""
    if side == "sw":
        return pc.box(u + a0, u + a1, v + 1 - c1, v + 1 - c0, z0, z1)
    if side == "se":
        return pc.box(u + 1 - c1, u + 1 - c0, v + a0, v + a1, z0, z1)
    if side == "ne":
        return pc.box(u + a0, u + a1, v + c0, v + c1, z0, z1)
    return pc.box(u + c0, u + c1, v + a0, v + a1, z0, z1)


def rim(side, u, v, tile, k):
    """what stands on side's edge of tile (u, v): on the front edges (sw, se) the low inner parapet, its bricks
    running on from the face below and a cap on top; on the back edges (ne, nw) the battlements, a wall with one
    merlon in the middle of the tile so the gaps fall on its sides (k = 2 chips the merlon's top corner)"""
    if side in ("sw", "se"):
        hb, hc, t = PARAPET
        return [pc.Body(strip(side, u, v, 0, 1, INSET, INSET + t, LIFT, LIFT + hb), plaster_paint, tile),
                pc.Body(strip(side, u, v, 0, 1, INSET - .04, INSET + t + .04, LIFT + hb, LIFT + hb + hc), cap_paint, tile)]
    hb, hm, t = BATTLE
    paint = bat_paint(k, "+v" if side == "ne" else "+u")
    out = [pc.Body(strip(side, u, v, 0, 1, 0, t, LIFT, LIFT + hb), paint, tile)]
    if k == 2 and tile == 0:
        out += [pc.Body(strip(side, u, v, .25, .62, 0, t, LIFT + hb, LIFT + hb + hm), paint, tile),
                pc.Body(strip(side, u, v, .62, .75, 0, t, LIFT + hb, LIFT + hb + hm - 3), paint, tile)]
    else:
        out.append(pc.Body(strip(side, u, v, .25, .75, 0, t, LIFT + hb, LIFT + hb + hm), paint, tile))
    return out


def rim_piece(side, k):
    """tile (0, 0)'s rim beside the same rim on the tiles before and after it along the edge, all on the walk"""
    along = [(-1, 0), (1, 0)] if side in ("sw", "ne") else [(0, -1), (0, 1)]
    bodies = rim(side, 0, 0, 0, k) + [wall_block(0, 0, -1)]
    for a, b in along:
        bodies += rim(side, a, b, -1, 0) + [wall_block(a, b, -1)]
    a, own, o = pc.render(bodies, bounds=FACE_BOUNDS)
    return pc.piece(a, own, o, 0, (0, 8 - LIFT))


# ---------------------------------------------------------------- the walk's paving

pc.PAL.update({"wP0": (146, 138, 122), "wP1": (138, 130, 114), "wP2": (154, 146, 130), "wP3": (132, 124, 108),
               "wPj": (104, 98, 86), "wPg": (120, 112, 98)})


def paving(i, j):
    """slabs 8 dots square in rows along u, every other row shifted half a slab; tones by the slab's place in the cell"""
    off = (np.floor_divide(j, 8) % 2) * 4
    a = np.floor_divide(i + off, 8) % 4
    b = np.floor_divide(j, 8)
    joint = (np.mod(i + off, 8) == 7) | (np.mod(j, 8) == 7)
    tone = np.mod(a * 3 + b * 5 + a * b, 4).astype(str)
    keys = np.where(joint, "wPj", np.char.add("wP", tone))
    grime = np.mod(i * 7 + j * 13, 29) == 0
    return np.where(grime & ~joint, "wPg", keys)


def floor():
    X, Y = np.meshgrid(np.arange(-32, 32) + .5, np.arange(0, 32) + .5)
    u, v = X / 32 + Y / 16, Y / 16 - X / 32
    inside = (u >= 0) & (u < 2) & (v >= 0) & (v < 2)
    i, j = np.floor(16 * u).astype(int).clip(0, 31), np.floor(16 * v).astype(int).clip(0, 31)
    keys = paving(i, j)
    a = np.zeros(X.shape + (4,), np.uint8)
    for key in set(keys[inside]):
        m = inside & (keys == key)
        a[m, :3] = pc.PAL[key]
        a[m, 3] = 255
    return Image.fromarray(a), (32, 16)


# ---------------------------------------------------------------- the gate tower

TOWER = (10, 4.4)          # length along u, depth along v, small tiles; centred on its anchor
pc.PAL.update({
    "tS": (170, 166, 154), "ts": (124, 120, 112), "tU": (186, 182, 170),          # plinth stone
    "tC": (156, 44, 34), "tc": (108, 30, 26),                                     # lacquered columns
    "tL": (222, 214, 194), "tl": (170, 162, 146),                                 # plaster
    "tD": (120, 78, 46), "td": (86, 54, 32), "tG": (66, 42, 26),                  # lattice doors and windows
    "tR": (96, 100, 108), "tr": (66, 70, 78), "tF": (80, 84, 92), "tf": (54, 58, 66),   # roof tiles lit/shaded, furrows
    "tE": (148, 146, 140), "te": (112, 110, 106), "tK": (42, 44, 48),             # tile ends, ridge
    "tB": (54, 40, 30), "tb": (198, 160, 70),                                     # the name board and its gilt frame
    "tA": (132, 52, 38), "ta": (96, 38, 28),                                      # balcony rail
})
Z1, Z2 = 40, 80            # the first storey's eave, the second storey's eave


def plinth_paint(f, u, v, z, s, h):
    if f == "+z":
        return np.full(u.shape, "tU")
    return pc.brick(s, h, "tS", "ts", w=12, rows=5) if pc.lit(f) else pc.brick(s, h, "ts", "wSj", w=12, rows=5)


def column_paint(f, u, v, z, s, h):
    return np.full(u.shape, "tC" if pc.lit(f) or f == "+z" else "tc")


def hall_paint(top):
    """plaster walls with lattice doors on the front and windows on the sides, a dark band under the eave"""
    def paint(f, u, v, z, s, h):
        lit_face = pc.lit(f)
        base = np.full(u.shape, "tL" if lit_face else "tl")
        if f == "+z":
            return base
        m = np.mod(s, 16)
        band = h >= top - 3
        if f == "+v":
            door = (m >= 3) & (m <= 12) & (h <= top - 7)
            lattice = door & ((np.mod(m, 3) == 0) | (np.mod(h, 4) == 0))
            frame = door & ((m == 3) | (m == 12) | (h == top - 7))
            keys = np.where(door, np.where(frame | lattice, "td", "tD"), base)
        else:
            win = (m >= 5) & (m <= 10) & (h >= top - 18) & (h <= top - 8)
            keys = np.where(win, np.where((np.mod(m, 2) == 0) | (np.mod(h, 3) == 0), "tG", "td"), base)
        return np.where(band, "tG", keys)
    return paint


def roof_paint(f, u, v, z, s, h, eave_z=0):
    if f == "+z":                    # an eave's flat top: the floor of the storey above, in planks
        return np.where(np.mod(np.floor(16 * u).astype(int), 4) == 3, "tG", "td")
    lit_face = f.endswith("v")
    a, b, end = ("tR", "tF", "tE") if lit_face else ("tr", "tf", "te")
    rows = np.where(np.mod(s, 4) == 3, b, a)
    return np.where(z < eave_z + 2, np.where(np.mod(s, 2) == 0, end, "tK"), rows)


def eave(zb):
    return lambda f, u, v, z, s, h: roof_paint(f, u, v, z, s, h, zb)


def ridge_paint(f, u, v, z, s, h):
    return np.full(u.shape, "tK" if f != "+z" else "te")


def board_paint(f, u, v, z, s, h):
    edge = (np.mod(s, 16) == 0) | (h <= Z2 - 15) | (h >= Z2 - 8)
    return np.where(edge, "tb", "tB")


def rail_paint(f, u, v, z, s, h):
    return np.where((np.mod(s, 3) == 0) | (h >= Z1 + 10), "tA" if pc.lit(f) else "ta", "tL" if pc.lit(f) else "tl")


def hip(u0, u1, v0, v1, zb, slope, top=None):
    """a hipped roof over [u0,u1]x[v0,v1] rising slope px per tile from zb at its edges, cut flat at top"""
    return pc.Line("u").hip(u0, u1, v0, v1, zb, slope, top)


def tower_bodies():
    L, D = TOWER[0] / 2, TOWER[1] / 2
    B = pc.Body
    out = [B(pc.box(-L, L, -D, D, 0, 5), plinth_paint)]
    # first storey: a hall set back behind a row of columns, the porch open in front
    out.append(B(pc.box(-L + .7, L - .7, -D + .6, D - .9, 5, Z1), hall_paint(Z1)))
    for a in np.arange(-L + .3, L - .2, 2.35):
        out.append(B(pc.box(a, a + .35, D - .55, D - .2, 5, Z1), column_paint))
    for a in (-L + .3, L - .65):
        out.append(B(pc.box(a, a + .35, -D + .2, -D + .55, 5, Z1), column_paint))
        out.append(B(pc.box(a, a + .35, -.2, .15, 5, Z1), column_paint))
    # the first eave all round, flat on top as the second storey's floor, a balcony rail along its front
    out.append(B(hip(-L - .5, L + .5, -D - .5, D + .5, Z1 - 3, 14, top=Z1 + 5), eave(Z1 - 3)))
    out.append(B(pc.box(-L + 1.2, L - 1.2, D - .3, D - .2, Z1 + 5, Z1 + 12), rail_paint))
    # second storey, its name board under the eave
    out.append(B(pc.box(-L + 1.9, L - 1.9, -D + 1.2, D - 1.2, Z1 + 5, Z2), hall_paint(Z2)))
    for a in (-L + 1.7, L - 2.05):
        out.append(B(pc.box(a, a + .35, D - 1.25, D - .95, Z1 + 5, Z2), column_paint))
    out.append(B(pc.box(-1.3, 1.3, D - 1.2, D - 1.12, Z2 - 16, Z2 - 7), board_paint))
    # the top roof: a skirt, then a steep hip and its ridge with raised ends
    out.append(B(hip(-L + 1.3, L - 1.3, -D + .8, D - .75, Z2 - 3, 16, top=Z2 + 5), eave(Z2 - 3)))
    out.append(B(hip(-L + 1.6, L - 1.6, -D + 1.05, D - 1.05, Z2 + 3, 40), eave(-99)))
    zr = Z2 + 3 + 40 * (D - 1.05)
    ra = L - 1.6 - (D - 1.05)
    out.append(B(pc.box(-ra, ra, -.12, .12, zr - 3, zr + 2), ridge_paint))
    for a in (-ra, ra - .3):
        out.append(B(pc.box(a, a + .3, -.16, .16, zr - 3, zr + 8), ridge_paint))
    return out


def tower():
    a, own, o = pc.render(tower_bodies(), along=lambda u, v: np.zeros(u.shape), bounds=(-160, -220, 160, 80))
    return pc.piece(a, own, o, 0, (0, 0))


# ---------------------------------------------------------------- the gate's passage on the inner face

GATE = (3.0, 64, .5)       # the passage: width (small tiles), height to the arch's crown (px), depth shown (tiles)
pc.PAL.update({"gP": (40, 36, 34), "gp": (58, 52, 48), "gF": (70, 64, 56), "gV": (196, 190, 176), "gv": (150, 144, 132),
               "gD": (96, 60, 36), "gd": (70, 44, 28), "gS": (42, 30, 22)})


def dark_paint(f, u, v, z, s, h):
    return np.full(u.shape, "gF" if f == "+z" else "gp" if f.endswith("u") else "gP")


def voussoir_paint(f, u, v, z, s, h):
    """the arch's ring of dressed stones, a joint every three dots"""
    if f.endswith("u"):
        return np.full(u.shape, "gv")
    return np.where(np.mod(s, 3) == 2, "gv", "gV")


def leaf_paint(f, u, v, z, s, h):
    """a gate leaf swung back against the passage's side: planks, an iron band near the top and foot"""
    c = np.where(np.mod(s, 4) == 3, "gd", "gD")
    return np.where((np.abs(h - 8) <= 1) | (np.abs(h - 34) <= 1), "gS", c)


def gate_bodies(tile=None):
    """the face's section round the passage, centred on u = 0 with the face at v = 0 (the passage runs into -v):
    piers and a wall over it in the face's bricks, a ring of stones stepping in to a round crown, the dark inside with
    the leaves swung back against its sides; around it, as context, the face running on"""
    w, crown, d = GATE
    hw = w / 2
    B = pc.Body
    out = []
    spring = crown - 14
    # the ring and the wall it carries, in steps that round off the opening
    steps = [(0, spring), (.18, spring + 5), (.42, spring + 9), (.7, spring + 12), (1.0, crown)]
    for i, (inset, z0) in enumerate(steps):
        z1 = steps[i + 1][1] if i + 1 < len(steps) else crown
        if z1 > z0:
            for a0, a1 in ((-hw, -hw + inset + .001), (hw - inset - .001, hw)):
                out.append(B(pc.box(a0, a1, -.12, 0, z0, z1 + 6), voussoir_paint, tile))
    out.append(B(pc.box(-hw, hw, -.12, 0, crown, crown + 6), voussoir_paint, tile))
    out.append(B(pc.box(-GATE_SPAN, -hw, -d, 0, 0, crown + 8), face_paint(None), tile))
    out.append(B(pc.box(hw, GATE_SPAN, -d, 0, 0, crown + 8), face_paint(None), tile))
    out.append(B(pc.box(-hw, hw, -d, -.12, spring, crown + 8), face_paint(None), tile))
    out.append(B(pc.box(-hw, hw, -.12, 0, crown + 6, crown + 8), face_paint(None), tile))
    # inside: the passage's far dark, its sides, the leaves
    out.append(B(pc.box(-hw, hw, -d - .3, -d, 0, crown + 8), dark_paint, tile))
    out.append(B(pc.box(-hw - .2, -hw, -d, -.12, 0, spring), dark_paint, tile))
    out.append(B(pc.box(hw, hw + .2, -d, -.12, 0, spring), dark_paint, tile))
    out.append(B(pc.box(-hw, -hw + .12, -d + .02, -.2, 0, spring - 2), leaf_paint, tile))
    out.append(B(pc.box(hw - .12, hw, -d + .02, -.2, 0, spring - 2), leaf_paint, tile))
    return out


GATE_SPAN = 2              # the section reaches this many small tiles either side of the passage's middle


def gate():
    """the section cut into one slice per small tile along the face (k = 0 the leftmost), each with its pivot at its
    tile's middle on the face's line, so each stands only over its own tile (the walk's tops behind stay clear)"""
    other = 99                  # the context's owner: not a slice (they are -2..1, so not -1 either)
    ctx = [pc.Body(pc.box(-GATE_SPAN - 3, -GATE_SPAN, -.5, 0, 0, LIFT), face_paint(None), other),
           pc.Body(pc.box(GATE_SPAN, GATE_SPAN + 3, -.5, 0, 0, LIFT), face_paint(None), other),
           pc.Body(pc.box(-GATE_SPAN - 3, GATE_SPAN + 3, -.5, 0, GATE[1] + 8, LIFT), face_paint(None), other)]
    # a pixel goes with the tile of the face where its ray crosses the face's plane (v = 0): what lies deeper in the
    # passage shows over that tile, not over the one beside it
    a, own, o = pc.render(gate_bodies() + ctx, along=lambda u, v: np.floor(u - v - 1e-6), bounds=(-120, -LIFT - 20, 120, 80))
    return {f"baimen_gate_{i}": pc.piece(a, own, o, j, (16 * (j + .5), 8 * (j + .5)))
            for i, j in enumerate(range(-GATE_SPAN, GATE_SPAN))}


# ---------------------------------------------------------------- the kit


def kit():
    out = {f"{FACES}_{m:02d}_{k}": face(m, k) for m in (4, 8, 12) for k in range(3)}
    out.update({f"{DRY_FACES}_{m:02d}_{k}": face(m, k, wet=False) for m in (4, 8, 12) for k in range(3)})
    out.update({f"{RIMS}_{side}_{k}": rim_piece(side, k) for side in ("sw", "se", "ne", "nw") for k in range(4)})
    out["floor_rampart"] = floor()
    out["baimen_tower"] = tower()
    out.update(gate())
    return out


MADE = "city wall top, pixel art on the small-tile grid (gen_pixel_rampart.py)"


def write():
    here = os.path.dirname(os.path.abspath(__file__))
    tiles_dir = os.path.join(here, "..", "tiles")
    path = os.path.join(tiles_dir, "tiles.json")
    tiles = json.load(open(path))
    for name, (img, pivot) in kit().items():
        img.save(os.path.join(tiles_dir, name + ".png"))
        e = {"file": f"tiles/{name}.png", "size": list(img.size), "pivot": list(pivot),
             "layer": "ground" if name.startswith("floor") else "object", "scale": "scene", "made": MADE}
        if not name.startswith("floor"):
            e["shadow"] = False
        tiles["sprites"][name] = e
    json.dump(tiles, open(path, "w"), ensure_ascii=False, indent=1)
    open(path, "a").write("\n")


def lay(canvas, img_pivot, x, y):
    img, (px, py) = img_pivot
    canvas.alpha_composite(img, (int(x - px), int(y - py)))


def preview(out):
    """the pieces at 4x, and a wall top laid out the way the engine lays it (floors, battlements, faces, the tower)"""
    os.makedirs(out, exist_ok=True)
    k = kit()
    for name, (img, _) in k.items():
        img.resize((img.width * 4, img.height * 4), Image.NEAREST).save(os.path.join(out, name + ".png"))
    W, D = 16, 8                      # small tiles of walk along u, across v
    cv = Image.new("RGBA", (720, 600), (96, 110, 80, 255))
    ox, oy = 360 - 16 * (W // 2 - D // 2), 440 - 8 * (W // 2 + D // 2)

    def at(u, v, z=0):
        return ox + 16 * (u - v), oy + 8 * (u + v) - z
    for gv in range(D // 2):
        for gu in range(W // 2):
            x, y = at(2 * gu, 2 * gv, LIFT)
            lay(cv, k["floor_rampart"], x, y + 16)
    for u in range(W):
        x, y = at(u, 0, LIFT)
        lay(cv, k[f"{RIMS}_ne_{(u * 7) % 4}"], x, y + 8)
    lay(cv, k["baimen_tower"], *at(W / 2, 2.6, LIFT))
    for u in range(W):
        v = D - 1
        x, y = at(u, v)
        name = f"{FACES}_{12 if u == W - 1 else 8:02d}_{(u * 7 + v * 3) % 3}"
        lay(cv, k[name], x, y + 8)
    for v in range(D - 1):
        x, y = at(W - 1, v)
        lay(cv, k[f"{FACES}_04_{(v * 3 + (W - 1) * 7) % 3}"], x, y + 8)
    for i in range(2 * GATE_SPAN):
        lay(cv, k[f"baimen_gate_{i}"], *at(W / 2 - GATE_SPAN + i + .5, D))
    for u in range(W):
        x, y = at(u, D - 1, LIFT)
        lay(cv, k[f"{RIMS}_sw_{(u * 5) % 4}"], x, y + 8)
    for v in range(D):
        x, y = at(W - 1, v, LIFT)
        lay(cv, k[f"{RIMS}_se_{(v * 5) % 4}"], x, y + 8)
    cv.resize((cv.width * 2, cv.height * 2), Image.NEAREST).save(os.path.join(out, "layout.png"))


if __name__ == "__main__":
    if sys.argv[1:2] == ["--write"]:
        write()
    elif sys.argv[1:2] == ["--preview"]:
        preview(sys.argv[2])
    else:
        print(__doc__)
