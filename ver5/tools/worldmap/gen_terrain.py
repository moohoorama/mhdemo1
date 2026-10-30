#!/usr/bin/env python3
"""Draw the terrain pieces of the battle maps and the world map as pixel art on the battle map's 2:1 grid.

  python3 tools/worldmap/gen_terrain.py            # write assets/graphics/map/terrain.png and terrain.json
  python3 tools/worldmap/gen_terrain.py --sheet F  # also a 4x contact sheet of every piece

Pieces are drawn at the characters' scale like the battle map's forests and villages: one game cell is a 64 x 32
diamond, x = 32(u - v), y = 16(u + v) - z with u, v in cells from the cell's centre and z in pixels up.

- Mountains and hills are height fields ray-marched from the top down (the first height that reaches the ray is the
  pixel), lit from the +v side like the pixel kit's boxes, quantized to a few tones per material and inked with the
  battle map's 1 px outline (not along the foot, where they meet the ground). Forested slopes carry little pines.
- Roads are flat: a beaten earth band from the cell's centre to the middle of each joined side (mask bit i joins
  neighbour (-1,0), (0,-1), (1,0), (0,1): nw, ne, se, sw), laid on the ground under everything that stands.

terrain.json lists each piece's rect on the sheet, pivot (the cell centre on the ground) and layer (ground/object),
the way assets/graphics/index.json lists "extra"; index.json's "terrain" names it and the game reads its pieces as
props. The battle maps use the rolling hills (bt_knoll_*, ground u) and the mountains (wm_mtn_rock_*, wm_mtn_big_*,
ground n); the rest (other mountains, hills, roads, the flat-topped bt_hill_*) were drawn for a world map trial and
are not used yet."""
import argparse
import json
import math
from pathlib import Path

import numpy as np
from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
OUT = ROOT / 'assets/graphics/map'
INK = (11, 16, 27)
S = 45.0  # ground pixels per cell along u or v, to weigh slopes against z


def hexes(*cs):
    return [tuple(int(c[i:i + 2], 16) for i in (1, 3, 5)) for c in cs]


# tones dark to light; matched to the battle map (rock piles, flagstones, grass, pines, snow)
PAL = {
    'rock': hexes('#3d3530', '#5e5143', '#7d6d55', '#a08c69', '#c8b48a'),
    'stone': hexes('#363f4b', '#4f5b69', '#6d7b8a', '#93a3af', '#bccad3'),
    'grass': hexes('#3b5434', '#4a6741', '#5f7c45', '#708f4d', '#86a457'),
    'pine': hexes('#16200f', '#22301a', '#405826', '#688630'),
    'snow': hexes('#7d8aa0', '#aab6c6', '#d4dde6', '#f1f4f6'),
    'earth': hexes('#5c4630', '#7d6242', '#a08458', '#bb9e6a', '#d2b884'),
}
LIGHT = np.array([-.35, .75, .9])
LIGHT /= np.linalg.norm(LIGHT)


def hash2(i, j, seed):
    h = (i * 374761393 + j * 668265263 + seed * 1442695041) & 0xffffffff
    h = ((h ^ (h >> 13)) * 1274126177) & 0xffffffff
    return (h ^ (h >> 16)) / 0xffffffff


def vnoise(u, v, f, seed):
    """Smooth value noise in [0, 1] at frequency f (cells)."""
    x, y = u * f, v * f
    i, j = np.floor(x).astype(np.int64), np.floor(y).astype(np.int64)
    fx, fy = x - i, y - j
    fx, fy = fx * fx * (3 - 2 * fx), fy * fy * (3 - 2 * fy)
    a, b = hash2(i, j, seed), hash2(i + 1, j, seed)
    c, d = hash2(i, j + 1, seed), hash2(i + 1, j + 1, seed)
    return (a * (1 - fx) + b * fx) * (1 - fy) + (c * (1 - fx) + d * fx) * fy


def fbm(u, v, f, seed, octaves=3):
    n, amp, tot = 0, 1, 0
    for k in range(octaves):
        n = n + amp * vnoise(u, v, f * 2 ** k, seed + 17 * k)
        tot += amp
        amp *= .5
    return n / tot


class Relief:
    """A height field: peaks (u, v, height px, radius cells, sharpness), roughened by ridged noise."""

    def __init__(self, peaks, seed, rough=.22, kind='rock', reach=.56):
        self.peaks, self.seed, self.rough, self.kind, self.reach = peaks, seed, rough, kind, reach
        self.top = max(p[2] for p in peaks)

    def h(self, u, v):
        z = np.zeros_like(u)
        for pu, pv, ph, r, k in self.peaks:
            d = np.hypot(u - pu, v - pv) / r
            z = np.maximum(z, ph * np.clip(1 - d, 0, 1) ** k)
        ridge = 1 - np.abs(2 * fbm(u + .3, v - .7, 3.2, self.seed) - 1)
        z = z * (1 - self.rough + self.rough * 1.6 * ridge)
        foot = np.clip((self.reach - np.maximum(np.abs(u), np.abs(v))) / .12, 0, 1)  # stay inside the cell
        return z * foot


def march(rel, w, h, px, py):
    """Ray-march the relief over a w x h canvas whose pivot (cell centre on the ground) is (px, py): per pixel the
    height hit (-1 where the ray misses) and its (u, v)."""
    ys, xs = np.mgrid[0:h, 0:w].astype(float)
    x, y = xs - px + .5, ys - py + .5
    hit = np.full((h, w), -1.0)
    hu, hv = np.zeros((h, w)), np.zeros((h, w))
    for z in np.arange(rel.top + 2, -.01, -.5):
        s, d = (y + z) / 16, x / 32
        u, v = (s + d) / 2, (s - d) / 2
        zz = rel.h(u, v)
        new = (hit < 0) & (zz >= z) & (zz > .4)
        hit[new], hu[new], hv[new] = z, u[new], v[new]
    return hit, hu, hv


def normals(rel, u, v):
    e = .01
    dzu = (rel.h(u + e, v) - rel.h(u - e, v)) / (2 * e * S)
    dzv = (rel.h(u, v + e) - rel.h(u, v - e)) / (2 * e * S)
    n = np.stack([-dzu, -dzv, np.ones_like(u)], -1)
    return n / np.linalg.norm(n, axis=-1, keepdims=True)


def tone(pal, b):
    return pal[int(np.clip(b, 0, .999) * len(pal))]


def paint_relief(rel, w, h, px, py, trees=0, snow=None):
    hit, hu, hv = march(rel, w, h, px, py)
    n = normals(rel, hu, hv)
    lam = np.clip(n @ LIGHT, 0, 1)
    slope = 1 - n[..., 2]
    grain = fbm(hu, hv, 9, rel.seed + 5, 2)
    img = np.zeros((h, w, 4), np.uint8)
    for j in range(h):
        for i in range(w):
            z = hit[j, i]
            if z < 0:
                continue
            b = .15 + .95 * lam[j, i] + .25 * (grain[j, i] - .5)
            zr = z / rel.top
            if snow is not None and zr > snow + .08 * (grain[j, i] - .5) and slope[j, i] < .75:
                pal = PAL['snow']
            elif rel.kind == 'hill' or (rel.kind == 'green' and slope[j, i] < .5 and zr < .78):
                pal = PAL['grass']
            else:
                pal = PAL[rel.kind if rel.kind in PAL else 'rock']
                if rel.kind in ('green', 'hill'):
                    pal = PAL['rock']
            if z < 2.5 and pal is not PAL['snow']:
                pal = PAL['grass']  # the foot sinks into the grass round it
            img[j, i] = (*tone(pal, b), 255)
    # a crease where a nearer ridge stands over a farther slope: the slope's darkest tone
    for j in range(1, h):
        for i in range(w):
            if hit[j, i] >= 0 and hit[j - 1, i] >= 0 and (hu[j - 1, i] + hv[j - 1, i]) - (hu[j, i] + hv[j, i]) > .18:
                img[j - 1, i, :3] = PAL['rock'][0] if rel.kind != 'hill' else PAL['grass'][0]
    if trees:
        plant_pines(img, rel, hit, hu, hv, slope, px, py, trees)
    return outline(img, hit)


def plant_pines(img, rel, hit, hu, hv, slope, px, py, step):
    """Little pines on the gentler, lower slopes: a 3 x 5 cone each, far to near so nearer ones overlap."""
    h, w = hit.shape
    spots = []
    for a in np.arange(-.6, .6, step):
        for b in np.arange(-.6, .6, step):
            u = a + (hash2(int(a * 97), int(b * 89), rel.seed) - .5) * step
            v = b + (hash2(int(b * 61), int(a * 71), rel.seed + 3) - .5) * step
            z = float(rel.h(np.array([u]), np.array([v]))[0])
            if z < 1 or z > .72 * rel.top:
                continue
            x, y = int(round(px + 32 * (u - v))), int(round(py + 16 * (u + v) - z))
            if not (1 <= x < w - 1 and 4 <= y < h) or hit[y, x] < 0 or abs(hit[y, x] - z) > 2.5 or slope[y, x] > .6:
                continue
            spots.append((u + v, x, y))
    pine = PAL['pine']
    shape = [(0, -4, 3), (-1, -3, 2), (0, -3, 3), (1, -3, 2), (-1, -2, 1), (0, -2, 2), (1, -2, 2),
             (-1, -1, 1), (0, -1, 1), (1, -1, 2), (0, 0, 0)]
    for _, x, y in sorted(spots):
        for dx, dy, t in shape:
            if 0 <= y + dy < h and 0 <= x + dx < w:
                img[y + dy, x + dx] = (*pine[t], 255)
                hit[y + dy, x + dx] = max(hit[y + dy, x + dx], 3)


def outline(img, hit):
    """Ink the empty pixels beside a filled one, except under the foot (filled pixels lower than 3 px)."""
    a = img[..., 3] > 0
    h, w = a.shape
    out = img.copy()
    for j in range(h):
        for i in range(w):
            if a[j, i]:
                continue
            for di, dj in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                jj, ii = j + dj, i + di
                if 0 <= jj < h and 0 <= ii < w and a[jj, ii] and (hit[jj, ii] >= 3 or dj > 0):
                    out[j, i] = (*INK, 255)
                    break
    return out


def trim(img, px, py):
    a = np.argwhere(img[..., 3] > 0)
    (y0, x0), (y1, x1) = a.min(0), a.max(0) + 1
    return img[y0:y1, x0:x1], (px - x0, py - y0)


def mountain(seed, kind='rock', big=False, snow=None, trees=0):
    rnd = lambda k: hash2(seed, k, 11)
    if big:
        top = 58 + 10 * rnd(0)
        peaks = [(.04, .04, top, .62, 1.05), (-.22 + .1 * rnd(1), .18, top * .62, .4, 1.2),
                 (.2, -.2 + .1 * rnd(2), top * .55, .38, 1.2)]
        reach = .62
    else:
        top = 34 + 12 * rnd(0)
        peaks = [(.06 * (rnd(1) - .5), .06 * (rnd(2) - .5), top, .5, 1.1),
                 (-.2, .16, top * (.45 + .2 * rnd(3)), .32, 1.25), (.18, -.16, top * (.4 + .2 * rnd(4)), .3, 1.25)]
        reach = .55
    rel = Relief(peaks, seed, kind=kind, reach=reach)
    w, h = 96, int(top) + 60
    return trim(paint_relief(rel, w, h, w // 2, h - 30, trees=trees, snow=snow), w // 2, h - 30)


def hill(seed):
    rnd = lambda k: hash2(seed, k, 23)
    top = 11 + 5 * rnd(0)
    peaks = [(.08 * (rnd(1) - .5), .08 * (rnd(2) - .5), top, .55, .8), (-.2, .2, top * .7, .32, .9),
             (.22, -.15, top * .6, .3, .9)]
    rel = Relief(peaks, seed, rough=.12, kind='hill', reach=.56)
    w, h = 96, int(top) + 50
    return trim(paint_relief(rel, w, h, w // 2, h - 30, trees=.34), w // 2, h - 30)


def road(mask, seed=7):
    """A beaten earth band to the middle of each joined side; mask 0 is a small round yard."""
    w, h = 64, 32
    px, py = 32, 16
    ys, xs = np.mgrid[0:h, 0:w].astype(float)
    x, y = xs - px + .5, ys - py + .5
    s, d = y / 16, x / 32
    u, v = (s + d) / 2, (s - d) / 2
    inside = (np.abs(u) <= .5) & (np.abs(v) <= .5)
    dist = np.hypot(u, v)
    for bit, (du, dv) in enumerate(((-1, 0), (0, -1), (1, 0), (0, 1))):
        if mask & (1 << bit):
            t = np.clip(u * du + v * dv, 0, .5)
            dist = np.minimum(dist, np.hypot(u - du * t, v - dv * t))
    width = .13 if mask else .2
    wob = .025 * (fbm(u, v, 6, seed) - .5)
    band = (dist + wob < width) & inside
    edge = band & (dist + wob > width - .045)
    grain = fbm(u, v, 14, seed + 1, 2)
    img = np.zeros((h, w, 4), np.uint8)
    earth = PAL['earth']
    for j in range(h):
        for i in range(w):
            if not band[j, i]:
                continue
            g = grain[j, i]
            t = earth[1] if edge[j, i] else earth[3] if g < .62 else earth[4]
            if not edge[j, i] and g < .3:
                t = earth[2]
            img[j, i] = (*t, 255)
    return img, (px, py)


MESA = 8  # px: the battle hill's flat top, the height a unit standing on it is lifted by


def mesa(mask, seed=31):
    """The battle hill of one game cell: a flat top MESA px high over the whole cell (so the lifted cell marks meet
    it), running on into the joined neighbours (mask bit i: (-1,0), (0,-1), (1,0), (0,1)) and on the open sides
    falling in a short bank just outside the cell. The neighbours are drawn as context so the top meets them without
    a seam or an ink line; a pixel belongs to the cell whose hill lies nearest the ground point it shows."""
    from scipy import ndimage
    R = 48  # raster cells per game cell
    joined = [bool(mask & (1 << b)) for b in range(4)]
    hill = np.zeros((3 * R, 3 * R), bool)  # cells (-1..1, -1..1): index [v+1, u+1]
    hill[R:2 * R, R:2 * R] = True
    for b, (du, dv) in enumerate(((-1, 0), (0, -1), (1, 0), (0, 1))):
        if joined[b]:
            hill[(dv + 1) * R:(dv + 2) * R, (du + 1) * R:(du + 2) * R] = True
    for (a, b_), (du, dv) in (((0, 1), (-1, -1)), ((1, 2), (1, -1)), ((2, 3), (1, 1)), ((3, 0), (-1, 1))):
        if joined[a] and joined[b_]:  # two joined sides: the corner between them is hill too
            hill[(dv + 1) * R:(dv + 2) * R, (du + 1) * R:(du + 2) * R] = True
    dist, near = ndimage.distance_transform_edt(~hill, return_indices=True)
    bank = .2
    t = np.clip(1 - dist / R / bank, 0, 1)
    field = MESA * t * t * (3 - 2 * t)
    mine = (near[0] // R == 1) & (near[1] // R == 1)  # the nearest hill is this cell's

    def at(grid, u, v):
        x, y = np.clip(((u + 1.5) * R).astype(int), 0, 3 * R - 1), np.clip(((v + 1.5) * R).astype(int), 0, 3 * R - 1)
        return grid[y, x]

    def h(u, v):
        return at(field, u, v)

    w, hh = 64 + 2 * 16, 32 + MESA + 2 * 10
    px, py = w // 2, 16 + MESA + 10
    ys, xs = np.mgrid[0:hh, 0:w].astype(float)
    x, y = xs - px + .5, ys - py + .5
    hit = np.full((hh, w), -1.0)
    hu, hv = np.zeros((hh, w)), np.zeros((hh, w))
    for z in np.arange(MESA + 1, -.01, -.25):
        s, d = (y + z) / 16, x / 32
        u, v = (s + d) / 2, (s - d) / 2
        zz = h(u, v)
        new = (hit < 0) & (zz >= z) & (zz > .3) & (np.abs(u) <= 1.5) & (np.abs(v) <= 1.5)
        hit[new], hu[new], hv[new] = z, u[new], v[new]
    own = (hit >= 0) & at(mine, hu, hv)
    e = .02
    dzu = (h(hu + e, hv) - h(hu - e, hv)) / (2 * e * S)
    dzv = (h(hu, hv + e) - h(hu, hv - e)) / (2 * e * S)
    n = np.stack([-dzu, -dzv, np.ones_like(hu)], -1)
    n /= np.linalg.norm(n, axis=-1, keepdims=True)
    lam = np.clip(n @ LIGHT, 0, 1)
    grain = fbm(hu, hv, 11, seed, 2)
    tuft = fbm(hu, hv, 23, seed + 9, 1)
    img = np.zeros((hh, w, 4), np.uint8)
    grass, earth = PAL['grass'], PAL['earth']
    for j in range(hh):
        for i in range(w):
            if not own[j, i]:
                continue
            z, steep = hit[j, i], 1 - n[j, i, 2]
            if z >= MESA - .3:  # the top: the battle grass, a shade lighter, with tufts
                c = grass[4] if tuft[j, i] > .72 else grass[3] if grain[j, i] > .35 else grass[2]
            else:
                b = .1 + 1.05 * lam[j, i] + .2 * (grain[j, i] - .5)
                pal = earth[:4] if steep > .18 and grain[j, i] > .45 else grass
                c = tone(pal, b)
                if z < 1.2:
                    c = grass[2]  # the foot sinks into the grass round it
            img[j, i] = (*c, 255)
    # ink the open edges: empty (in the context too) beside the cell's own pixels, above the foot
    out = img.copy()
    for j in range(hh):
        for i in range(w):
            if hit[j, i] >= 0:
                continue
            for di, dj in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                jj, ii = j + dj, i + di
                if 0 <= jj < hh and 0 <= ii < w and own[jj, ii] and hit[jj, ii] >= 2:
                    out[j, i] = (*INK, 255)
                    break
    return out, (px, py)


KNOLL = 7  # px: the rolling hill's crest, the height a unit standing on it is lifted by (the cell marks stay on the ground)


def knoll(mask, seed=47):
    """The rolling hill of one game cell: a low round rise KNOLL px high that runs on into the joined neighbours
    (mask bit i: (-1,0), (0,-1), (1,0), (0,1)) as one swell and falls to the ground at the open sides, inside the
    cell. Only the picture is high: the cell is chosen and marked on the flat ground, as over the sunk water."""
    from scipy import ndimage
    R = 48
    joined = [bool(mask & (1 << b)) for b in range(4)]
    hill = np.zeros((3 * R, 3 * R), bool)
    hill[R:2 * R, R:2 * R] = True
    for b, (du, dv) in enumerate(((-1, 0), (0, -1), (1, 0), (0, 1))):
        if joined[b]:
            hill[(dv + 1) * R:(dv + 2) * R, (du + 1) * R:(du + 2) * R] = True
    for (a, b_), (du, dv) in (((0, 1), (-1, -1)), ((1, 2), (1, -1)), ((2, 3), (1, 1)), ((3, 0), (-1, 1))):
        if joined[a] and joined[b_]:
            hill[(dv + 1) * R:(dv + 2) * R, (du + 1) * R:(du + 2) * R] = True
    # the hill shrunk a fifth of a cell from its open sides and blurred: round, back to the ground at the cell's edge
    core = ndimage.binary_erosion(hill, iterations=int(.2 * R), border_value=0)
    blur = ndimage.gaussian_filter(core.astype(float), .15 * R)
    ys_, xs_ = np.mgrid[0:3 * R, 0:3 * R] / R - 1.5
    swell = 1 + .45 * (fbm(xs_ + 3.1, ys_ + 1.7, 1.4, seed, 2) - .5)  # the crest rises and falls along a ridge
    field = KNOLL * np.clip(blur / .9, 0, 1) ** .8 * swell

    def h(u, v):
        x, y = np.clip(((u + 1.5) * R).astype(int), 0, 3 * R - 1), np.clip(((v + 1.5) * R).astype(int), 0, 3 * R - 1)
        return field[y, x]

    w, hh = 64, 32 + KNOLL + 6
    px, py = 32, 16 + KNOLL + 3
    ys, xs = np.mgrid[0:hh, 0:w].astype(float)
    x, y = xs - px + .5, ys - py + .5
    hit = np.full((hh, w), -1.0)
    hu, hv = np.zeros((hh, w)), np.zeros((hh, w))
    for z in np.arange(KNOLL + 2, -.01, -.25):
        s, d = (y + z) / 16, x / 32
        u, v = (s + d) / 2, (s - d) / 2
        zz = h(u, v)
        new = (hit < 0) & (zz >= z) & (zz > .2) & (np.abs(u) <= 1.5) & (np.abs(v) <= 1.5)
        hit[new], hu[new], hv[new] = z, u[new], v[new]
    own = (hit >= 0) & (np.abs(hu) <= .5) & (np.abs(hv) <= .5)
    e = .02
    dzu = (h(hu + e, hv) - h(hu - e, hv)) / (2 * e * S)
    dzv = (h(hu, hv + e) - h(hu, hv - e)) / (2 * e * S)
    n = np.stack([-dzu, -dzv, np.ones_like(hu)], -1)
    n /= np.linalg.norm(n, axis=-1, keepdims=True)
    lam = np.clip(n @ LIGHT, 0, 1)
    grain = fbm(hu, hv, 11, seed, 2)
    tuft = fbm(hu, hv, 23, seed + 9, 1)
    img = np.zeros((hh, w, 4), np.uint8)
    grass = PAL['grass']
    lit = n[..., 0] * LIGHT[0] + n[..., 1] * LIGHT[1]  # how far a slope turns to the light, ground level 0
    for j in range(hh):
        for i in range(w):
            if not own[j, i] or hit[j, i] < .5:
                continue  # the lowest rim is the ground itself
            # the battle grass (tone 3) on the flat, lighter on the slopes to the light, darker away from it; the
            # rim fades into the ground round it
            k = min(1, hit[j, i] / 2.5)
            b = max(.42, .7 + k * (4 * lit[j, i] + .12 * (grain[j, i] - .5)) + .22 * (hit[j, i] / KNOLL - .5))
            c = tone(grass, b)
            if tuft[j, i] > .8 and b > .62:
                c = grass[4]
            img[j, i] = (*c, 255)
    out = img.copy()  # ink the crest where it stands clear of the ground behind it
    for j in range(hh - 1):
        for i in range(w):
            if img[j, i, 3] == 0 and img[j + 1, i, 3] and hit[j + 1, i] >= 2 and hit[j, i] < 0:
                out[j, i] = (*INK, 255)
    return out, (px, py)


def pieces():
    out = {}
    for k in range(3):
        out[f'wm_mtn_rock_{k}'] = (*mountain(100 + k, 'rock'), 'object')
        out[f'wm_mtn_stone_{k}'] = (*mountain(200 + k, 'stone'), 'object')
        out[f'wm_mtn_green_{k}'] = (*mountain(300 + k, 'green', trees=.2), 'object')
        out[f'wm_hill_{k}'] = (*hill(400 + k), 'object')
    for k in range(2):
        out[f'wm_peak_{k}'] = (*mountain(500 + k, 'stone', big=True, snow=.62), 'object')
        out[f'wm_mtn_big_{k}'] = (*mountain(600 + k, 'rock', big=True), 'object')
        out[f'wm_mtn_snow_{k}'] = (*mountain(700 + k, 'stone', snow=.5), 'object')
    for m in range(16):
        out[f'wm_road_{m:02d}'] = (*road(m), 'ground')
        out[f'bt_hill_{m:02d}'] = (*mesa(m), 'ground')
        out[f'bt_knoll_{m:02d}'] = (*knoll(m), 'ground')
    return out


def pack(ps, width=1024):
    order = sorted(ps, key=lambda n: (-ps[n][0].shape[0], n))
    x = y = row = 0
    place = {}
    for n in order:
        hh, ww = ps[n][0].shape[:2]
        if x + ww > width:
            x, y, row = 0, y + row + 1, 0
        place[n] = (x, y)
        x += ww + 1
        row = max(row, hh)
    sheet = Image.new('RGBA', (width, y + row + 1))
    sprites = {}
    for n in sorted(ps):
        img, (px, py), layer = ps[n]
        sheet.paste(Image.fromarray(img), place[n])
        sprites[n] = {'rect': [*place[n], img.shape[1], img.shape[0]], 'pivot': [int(px), int(py)], 'layer': layer}
    return sheet, sprites


def contact(ps, path):
    """Every piece on a grass diamond, 4x, for looking at."""
    names = sorted(ps)
    cols = 8
    cw, ch = 110, 120
    rows = (len(names) + cols - 1) // cols
    im = Image.new('RGBA', (cols * cw, rows * ch), (40, 44, 52, 255))
    grass = PAL['grass'][3]
    for k, n in enumerate(names):
        img, (px, py), _ = ps[n]
        ox, oy = (k % cols) * cw + cw // 2, (k // cols) * ch + ch - 30
        for j in range(-16, 17):
            half = 32 - abs(j) * 2
            for i in range(-half, half):
                im.putpixel((ox + i, oy + j), (*grass, 255))
        sp = Image.fromarray(img)
        im.alpha_composite(sp, (int(ox - px), int(oy - py)))
    im.resize((im.width * 4, im.height * 4), Image.NEAREST).save(path)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--sheet', help='write a 4x contact sheet here')
    a = ap.parse_args()
    ps = pieces()
    OUT.mkdir(parents=True, exist_ok=True)
    sheet, sprites = pack(ps)
    sheet.save(OUT / 'terrain.png')
    json.dump({'image': 'map/terrain.png', 'sprites': sprites}, open(OUT / 'terrain.json', 'w'), indent=1)
    if a.sheet:
        contact(ps, a.sheet)
    print(f'{len(sprites)} pieces -> {OUT / "terrain.png"}')


if __name__ == '__main__':
    main()
