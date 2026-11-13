"""Ground after a flood has gone down (16장, 하비 성안 after the battle), drawn as pixel art on the small-tile grid:
flat pictures laid on the ground (layer "ground"), and the scene houses with the flood's mud mark on their walls.

Flat pieces are painted dot by dot on the ground plane, one dot per pixel at i = floor(16u), j = floor(16v), so their
edges and streaks step one pixel down per two across like the tile edges. Shapes are smooth noise fields in (u, v)
tiles. The damp is a translucent dark brown laid in steps (dithered at the edges), so it darkens whatever ground it
lies on — soil, grass or paving — instead of being one ground's colour; standing water is opaque.

  wet_patch_0..3    damp ground left where the water stood, darker in the hollows with a glint of water in the
                    lowest dots. Pivot at the picture's back corner (u = v = 0), so a patch sorts before everything
                    that lies on it.
  puddle_0..5       standing water: a dark bank along its back edge, a sky-grey sheet with light streaks, a ring of wet
                    mud round it. puddle_4 runs along u, puddle_5 along v. Pivot at the front corner (u = A, v = B).
  rut_v             two wheel ruts along v full of water, mud churned between and beside them (pivot front).
  wrack_u, wrack_v  the drift line the water left: straw, twigs and silt in a band along u or v (pivot front).
  house_*_mud       the four scene houses with walls stained brown up to the flood's mark and a dark line along it.

  python3 gen_pixel_wet.py --preview DIR   # the pieces on soil, grass and paving, and the houses, at 4x into DIR
  python3 gen_pixel_wet.py --write         # save them into ../tiles and their entries of tiles.json
"""
import json
import os
import sys

import numpy as np
from PIL import Image

HERE = os.path.dirname(os.path.abspath(__file__))
TILES = os.path.join(HERE, "..", "tiles")
MADE = "after the flood (16장 하비 성안), pixel art on the small-tile grid (gen_pixel_wet.py)"

DAMP = (64, 44, 22)              # the damp's colour, laid at the alphas below
DAMP_A = (52, 92, 130, 165)      # damp edge, damp, wet, mud
BANK = (58, 42, 26, 220)         # the back edge of standing water
WATER = {                        # deep (under the bank), sheet, streak, glint
    "sky": [(98, 108, 106), (124, 134, 130), (156, 166, 160), (204, 210, 200)],
    "mud": [(88, 78, 56), (108, 98, 72), (136, 128, 100), (188, 184, 160)],
}
STRAW = [(182, 162, 104), (146, 124, 76)]
TWIG = [(98, 72, 44), (70, 50, 32)]
BAYER = np.array([[0, 8, 2, 10], [12, 4, 14, 6], [3, 11, 1, 9], [15, 7, 13, 5]]) / 16


def noise(seed, u, v, scale):
    """smooth value noise in [0, 1) over (u, v) tiles, features about `scale` tiles across"""
    rng = np.random.default_rng(seed)
    grid = rng.random((64, 64))
    x, y = u / scale + 7, v / scale + 7
    ix, iy = np.floor(x).astype(int), np.floor(y).astype(int)
    fx, fy = x - ix, y - iy
    fx, fy = fx * fx * (3 - 2 * fx), fy * fy * (3 - 2 * fy)
    at = lambda dx, dy: grid[(iy + dy) % 64, (ix + dx) % 64]
    return (at(0, 0) * (1 - fx) + at(1, 0) * fx) * (1 - fy) + (at(0, 1) * (1 - fx) + at(1, 1) * fx) * fy


def ground(A, B, pivot_front):
    """the pixel grid of a flat picture over u in [0, A], v in [0, B]: its (u, v) at each pixel centre, the dot
    indices, and the pivot (the back corner, or the front one)"""
    w, h = int(16 * (A + B)), int(8 * (A + B))
    ox = int(16 * B)                                  # the back corner's x in the picture
    y, x = np.mgrid[0:h, 0:w]
    px, py = x + .5 - ox, y + .5
    u, v = (px / 16 + py / 8) / 2, (py / 8 - px / 16) / 2
    pivot = (int(ox + 16 * (A - B)), h) if pivot_front else (ox, 0)
    return u, v, np.floor(16 * u).astype(int), np.floor(16 * v).astype(int), x, y, pivot


def oval(u, v, A, B, seed, rough=.35, scale=.6):
    """1 at the middle of the [0, A] x [0, B] ellipse falling through 0 at its rim, its rim wobbled by noise"""
    a, b = A / 2, B / 2
    r = np.sqrt(((u - a) / a) ** 2 + ((v - b) / b) ** 2)
    return 1 - r + rough * (noise(seed, u, v, scale) - .5)


def damp_layer(rgba, field, x, y, cuts):
    """lay the damp in steps: where field passes cuts[k] (dithered across the last .06 below it), alpha DAMP_A[k]"""
    d = BAYER[y % 4, x % 4]
    for k, c in enumerate(cuts):
        on = field > c - .06 * d
        rgba[on] = DAMP + (DAMP_A[k],)


def water_layer(rgba, wet, i, j, x, y, seed, tone):
    """standing water where `wet`: the bank along its back edge, deep under it, a sheet with streaks and a glint"""
    deep, sheet, streak, glint = WATER[tone]
    above = np.zeros_like(wet)
    above[1:] = wet[:-1]
    above2 = np.zeros_like(wet)
    above2[2:] = wet[:-2] & wet[1:-1]
    rgba[wet] = sheet + (255,)
    rgba[wet & ~above2] = deep + (255,)
    rgba[wet & ~above] = BANK
    rng = np.random.default_rng(seed)
    h, w = wet.shape
    for _ in range(max(1, wet.sum() // 40)):
        ys, xs = np.nonzero(wet & above2)
        if len(ys) == 0:
            break
        k = rng.integers(len(ys))
        yy, xx, n = ys[k], xs[k], int(rng.integers(2, 7))
        for dx in range(n):
            if xx + dx < w and wet[yy, xx + dx] and above2[yy, xx + dx]:
                rgba[yy, xx + dx] = streak + (255,)
    ys, xs = np.nonzero(wet & above2)
    if len(ys):
        k = rng.integers(len(ys))
        rgba[ys[k], xs[k]] = glint + (255,)


def wet_patch(A, B, seed):
    u, v, i, j, x, y, pv = ground(A, B, False)
    f = oval(u, v, A, B, seed, rough=.9, scale=.8)
    rgba = np.zeros(u.shape + (4,), np.uint8)
    damp_layer(rgba, f, x, y, (0, .22, .5, .72))
    # a few dots of water in the lowest hollows
    hollow = (f > .62) & (noise(seed + 5, u, v, .25) > .7)
    water_layer(rgba, hollow, i, j, x, y, seed, "sky")
    return Image.fromarray(rgba), pv


def puddle(A, B, seed, tone="sky"):
    u, v, i, j, x, y, pv = ground(A, B, True)
    f = oval(u, v, A, B, seed, rough=.6, scale=.45)
    rgba = np.zeros(u.shape + (4,), np.uint8)
    damp_layer(rgba, f, x, y, (-.04, .06, .16, .24))
    water_layer(rgba, f > .3, i, j, x, y, seed, tone)
    return Image.fromarray(rgba), pv


def rut_v(B, seed):
    A = 1.25
    u, v, i, j, x, y, pv = ground(A, B, True)
    rgba = np.zeros(u.shape + (4,), np.uint8)
    n = noise(seed, u, v, .5) - .5
    along = np.minimum(v, B - v) / .5                 # fades out near both ends
    churn = 1 - np.abs(u - A / 2) / (A / 2) + .4 * n
    damp_layer(rgba, np.minimum(churn, along - .3), x, y, (0, .2, .45, .7))
    wet = np.zeros(u.shape, bool)
    for c in (.36, .89):                              # the two wheels, a cart's track apart
        groove = np.abs(u - c - .04 * np.sin(v * 2.1 + seed)) < .085 + .03 * n
        wet |= groove & (along > .35 + .5 * (noise(seed + int(c * 10), u, v, .7) - .5))
    water_layer(rgba, wet, i, j, x, y, seed, "mud")
    return Image.fromarray(rgba), pv


def wrack(axis, L, seed):
    A, B = (L, .6) if axis == "u" else (.6, L)
    u, v, i, j, x, y, pv = ground(A, B, True)
    t, s = (u, v) if axis == "u" else (v, u)          # along, across
    rng = np.random.default_rng(seed)
    rgba = np.zeros(u.shape + (4,), np.uint8)
    line = .3 + .07 * np.sin(t * 1.7 + seed) + .05 * (noise(seed, u, v, .4) - .5)
    ends = np.minimum(t, L - t)
    band = 1 - np.abs(s - line) / .16 + .3 * (ends < .4) * (ends / .4 - 1)
    damp_layer(rgba, band, x, y, (0, .35, .6, .9))
    # straw and twigs: short strokes along the band, each laid along u or v so it runs at the tiles' slope
    for _ in range(int(L * 34)):
        tt = rng.uniform(.15, L - .15)
        ss = .3 + .07 * np.sin(tt * 1.7 + seed) + rng.normal(0, .05)
        uu, vv = (tt, ss) if axis == "u" else (ss, tt)
        twig = rng.random() < .25
        col = TWIG if twig else STRAW
        n = int(rng.integers(3, 7 if twig else 5))
        du, dv = (1, 0) if rng.random() < .5 else (0, 1)
        for k in range(n):
            pu, pv_ = uu + du * k / 16, vv + dv * k / 16
            xx, yy = int(16 * (pu - pv_) + 16 * B), int(8 * (pu + pv_))
            if 0 <= yy < rgba.shape[0] and 0 <= xx < rgba.shape[1]:
                rgba[yy, xx] = col[(k + (yy & 1)) % 2] + (255,)
    return Image.fromarray(rgba), pv


MARK = 15                     # the flood's mark, pixels above the ground line of a house's walls
MUD = np.array((112, 88, 56))


def mud_house(name):
    """a scene house with its walls stained up to MARK pixels above their foot, a dark line along the mark"""
    im = Image.open(os.path.join(TILES, name + ".png")).convert("RGBA")
    a = np.array(im).astype(float)
    op = a[..., 3] > 0
    h, w = op.shape
    lum = a[..., :3].mean(axis=2)
    for x in range(w):
        ys = np.nonzero(op[:, x])[0]
        if len(ys) == 0:
            continue
        foot = ys[-1]
        for y in range(max(0, foot - MARK), foot + 1):
            if not op[y, x] or lum[y, x] < 45:        # leave the outline
                continue
            up = foot - y
            k = .72 if up == MARK else .58 - .2 * up / MARK
            if (up == MARK - 1) and (x + y) % 3 == 0:  # a ragged edge under the line
                k = .68
            a[y, x, :3] = a[y, x, :3] * (1 - k) + MUD * k
    return Image.fromarray(a.clip(0, 255).astype(np.uint8)), None


def kit():
    out = {
        "wet_patch_0": wet_patch(4, 3, 11), "wet_patch_1": wet_patch(5, 4, 12),
        "wet_patch_2": wet_patch(3, 2, 13), "wet_patch_3": wet_patch(6, 3, 14),
        "puddle_0": puddle(1.5, 1, 21), "puddle_1": puddle(2, 1.5, 22), "puddle_2": puddle(3, 2, 23, "mud"),
        "puddle_3": puddle(1, 1, 24), "puddle_4": puddle(2.5, 1, 25, "mud"), "puddle_5": puddle(1, 2.5, 26),
        "rut_v": rut_v(4, 31), "wrack_u": wrack("u", 4, 41), "wrack_v": wrack("v", 4, 42),
    }
    for n in ("house_thatch", "house_thatch_x", "house_tile", "house_tile_x"):
        out[n + "_mud"] = mud_house(n)
    return out


def write():
    path = os.path.join(TILES, "tiles.json")
    tiles = json.load(open(path))
    for name, (img, pivot) in kit().items():
        img.save(os.path.join(TILES, name + ".png"))
        if name.endswith("_mud"):
            base = dict(tiles["sprites"][name[:-4]])
            base.update(file=f"tiles/{name}.png", made=MADE)
            tiles["sprites"][name] = base
            continue
        tiles["sprites"][name] = {"file": f"tiles/{name}.png", "size": list(img.size), "pivot": list(pivot),
                                  "layer": "ground", "scale": "scene", "shadow": False, "made": MADE}
    json.dump(tiles, open(path, "w"), ensure_ascii=False, indent=1)
    open(path, "a").write("\n")


def preview(out):
    os.makedirs(out, exist_ok=True)
    k = kit()
    grounds = {"soil": (187, 158, 106), "grass": (113, 122, 95), "paving": (150, 160, 172)}
    flat = [n for n in k if not n.endswith("_mud")]
    for gname, col in grounds.items():
        cv = Image.new("RGBA", (1100, 200), col + (255,))
        x = 10
        for n in flat:
            img, (px, py) = k[n]
            if x + img.width > cv.width:
                break
            cv.alpha_composite(img, (x, 100 - img.height // 2))
            x += img.width + 8
        cv.resize((cv.width * 3, cv.height * 3), Image.NEAREST).save(os.path.join(out, f"flat_{gname}.png"))
    cv = Image.new("RGBA", (4 * 150, 90), grounds["soil"] + (255,))
    for n, name in enumerate(("house_thatch", "house_thatch_x", "house_tile", "house_tile_x")):
        old = Image.open(os.path.join(TILES, name + ".png")).convert("RGBA")
        cv.alpha_composite(old, (n * 150 + 4, 6))
        cv.alpha_composite(k[name + "_mud"][0], (n * 150 + 76, 6))
    cv.resize((cv.width * 4, cv.height * 4), Image.NEAREST).save(os.path.join(out, "houses.png"))


if __name__ == "__main__":
    if sys.argv[1:2] == ["--write"]:
        write()
    elif sys.argv[1:2] == ["--preview"]:
        preview(sys.argv[2])
    else:
        print(__doc__)
