#!/usr/bin/env python3
"""Ways to make one scene tree's leaves sway, all made from the existing dot picture.

Every frame keeps the original eight colours and the trunk pixels; the canopy pixels are
moved or recoloured and the 1px outline is redrawn around the result (4-neighbour, the same
rule that reproduces the original outline exactly). The 3D variants take only motion and
leaf tilt from the tree's 3D model (tools/unit3d/scene_props3d.py) and apply them to the dot.
"""

import json
import math
import sys
import warnings
from functools import cache, cached_property
from pathlib import Path

import numpy as np
from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "tools/unit3d"))

NAME = "stree_broad_0"
SRC = ROOT / "assets_candidate/tiles" / f"{NAME}.png"
OUT = ROOT / "tools/spritetool/assets/tree-sway-variants" / NAME
FRAMES = 8
GUST_FRAMES = 12
MARGIN = 3
TRUNK_Y = 77
SS = 2

OUTLINE = (11, 16, 27)
DARK, SHADE, MID, LIGHT, SHINE = (34, 48, 26), (64, 88, 38), (104, 134, 48), (152, 178, 66), (204, 214, 104)
STEP_UP = {DARK: SHADE, SHADE: MID, MID: LIGHT, LIGHT: SHINE, SHINE: SHINE}
STEP_DOWN = {SHINE: LIGHT, LIGHT: MID, MID: SHADE, SHADE: DARK, DARK: DARK}

# Lobes of the drawn crown: (centre x, centre y, phase, amplitude).
CLUMPS = [
    (40, 10, 0.0, 1.4), (22, 26, 1.3, 1.0), (60, 28, 2.4, 1.0), (42, 36, 0.7, 0.8),
    (12, 48, 3.6, 1.2), (70, 52, 4.4, 1.2), (36, 56, 5.2, 1.0), (52, 62, 2.0, 0.8),
]


def load():
    a = np.array(Image.open(SRC).convert("RGBA"))
    a = np.pad(a, ((0, 0), (MARGIN, MARGIN), (0, 0)))
    colour = a[..., :3]
    body = (a[..., 3] > 0) & ~np.all(colour == OUTLINE, axis=2)
    ys = np.arange(a.shape[0])[:, None]
    trunk = body & ((ys >= TRUNK_Y) | ((ys == TRUNK_Y - 1) & (colour[..., 0] > colour[..., 1])))
    return colour, trunk, body & ~trunk


def compose(colour, trunk, leaf, leaf_colour):
    h, w = trunk.shape
    rgba = np.zeros((h, w, 4), np.uint8)
    rgba[trunk, :3] = colour[trunk]
    rgba[leaf, :3] = leaf_colour[leaf]
    body = trunk | leaf
    rgba[body, 3] = 255
    pad = np.pad(body, 1)
    near = pad[:-2, 1:-1] | pad[2:, 1:-1] | pad[1:-1, :-2] | pad[1:-1, 2:]
    edge = near & ~body
    rgba[edge, :3] = OUTLINE
    rgba[edge, 3] = 255
    return rgba


def warp(colour, leaf, dx, dy):
    """Inverse-map the canopy: each output pixel takes the leaf pixel at (x-dx, y-dy)."""
    h, w = leaf.shape
    ys, xs = np.mgrid[0:h, 0:w]
    sx = np.clip(xs - np.rint(dx).astype(int), 0, w - 1)
    sy = np.clip(ys - np.rint(dy).astype(int), 0, h - 1)
    return leaf[sy, sx], colour[sy, sx]


def restep(colour, mask, table):
    out = colour.copy()
    for src, dst in table.items():
        out[mask & np.all(colour == src, axis=2)] = dst
    return out


def height_weight(h, power):
    y = np.arange(h, dtype=float)[:, None]
    return np.clip((TRUNK_Y - y) / TRUNK_Y, 0, 1) ** power


def phase(t, n=FRAMES):
    return 2 * math.pi * t / n


def envelope(t, n):
    """Calm at frame 0, the gust at its height half way through the loop."""
    return .5 - .5 * math.cos(phase(t, n))


class Tree:
    def __init__(self):
        self.colour, self.trunk, self.leaf = load()
        self.h, self.w = self.leaf.shape
        self.ys, self.xs = np.mgrid[0:self.h, 0:self.w]
        self.across = self.xs + .45 * self.ys
        self.speck = np.random.default_rng(7).random(
            (self.h // 2 + 1, self.w // 2 + 1)).repeat(2, 0).repeat(2, 1)[:self.h, :self.w]

    def frame(self, leaf_mask, leaf_colour):
        return compose(self.colour, self.trunk, leaf_mask, leaf_colour)

    @cached_property
    def rim(self):
        """Distance in pixels from the crown's edge (1..4), 99 further in."""
        dist = np.where(self.leaf, 99, 0)
        for d in range(1, 5):
            pad = np.pad(dist < d, 1)
            grow = pad[:-2, 1:-1] | pad[2:, 1:-1] | pad[1:-1, :-2] | pad[1:-1, 2:]
            dist = np.where((dist == 99) & grow, d, dist)
        return dist

    @cache
    def cells(self, spacing, seed):
        """Small jittered Voronoi cells, a few pixels across: one leaf or a small spray each."""
        rnd = np.random.default_rng(seed)
        gy, gx = np.mgrid[0:self.h + spacing:spacing, 0:self.w + spacing:spacing]
        pts = np.stack([gx.ravel(), gy.ravel()], 1) + rnd.uniform(-.4, .4, (gx.size, 2)) * spacing
        d = (self.xs[..., None] - pts[:, 0]) ** 2 + (self.ys[..., None] - pts[:, 1]) ** 2
        return d.argmin(axis=2), len(pts)

    @cached_property
    def tufts(self):
        """Lit leaf sprays: connected light pixels cut into pieces at most ~5px across."""
        lit = self.leaf & (np.all(self.colour == LIGHT, axis=2) | np.all(self.colour == SHINE, axis=2))
        label = np.full(lit.shape, -1)
        count = 0
        for y, x in zip(*np.nonzero(lit)):
            if label[y, x] >= 0:
                continue
            stack = [(y, x)]
            label[y, x] = count
            while stack:
                cy, cx = stack.pop()
                for ny, nx in ((cy - 1, cx), (cy + 1, cx), (cy, cx - 1), (cy, cx + 1)):
                    if 0 <= ny < self.h and 0 <= nx < self.w and lit[ny, nx] and label[ny, nx] < 0:
                        label[ny, nx] = count
                        stack.append((ny, nx))
            count += 1
        cell, ncell = self.cells(5, 11)
        key = np.where(lit, label * ncell + cell, -1)
        return [np.nonzero(key == k) for k in np.unique(key[lit]) if (key == k).sum() >= 2]


# Motion pieces. Each takes the canopy colours so far and returns new canopy colours (or a mask too).

def flutter(tree, colour, t, n=FRAMES, amp=1.0, bias=0.0):
    """Tufts along the rim (3px deep) bob in and out; the inside keeps `colour`."""
    ang = np.arctan2(tree.ys - 40, tree.xs - (40 + MARGIN))
    tuft = np.sin(ang * 7 + 1.7 * np.sin(ang * 3))
    wobble = ((tree.rim <= 3) | ~tree.leaf) * height_weight(tree.h, .3)
    dx = wobble * (amp * 1.2 * np.sin(phase(t, n) + tuft * 2.2) + bias)
    dy = wobble * amp * .9 * np.sin(phase(t, n) * 2 + tuft * 3.1 + 1)
    m, c = warp(colour, tree.leaf, dx, dy)
    keep = tree.leaf & (tree.rim > 3)
    return m | keep, np.where(keep[..., None], colour, c)


def sparkle(tree, colour, t, n=FRAMES, density=.14):
    """Small leaf cells all over the crown turn to the light and away, each on its own beat."""
    cell, count = tree.cells(3, 5)
    rnd = np.random.default_rng(3)
    ph, k = rnd.uniform(0, 2 * math.pi, count), rnd.integers(1, 3, count)
    wave = .5 + .5 * np.sin(phase(t, n) - tree.across * 2 * math.pi / 70)
    v = np.sin(k[cell] * phase(t, n) + ph[cell])
    up = tree.leaf & (v > 1 - 1.3 * density * wave)
    down = tree.leaf & (v < -1 + .7 * density * wave)
    return restep(restep(colour, up, STEP_UP), down, STEP_DOWN)


def waves(tree, colour, t, n=FRAMES, strength=1.0):
    """Wide wind waves cross the whole crown: leaves on a crest flip pale and lean, the trough behind darkens."""
    crest = np.clip(np.sin(phase(t, n) - tree.across * 2 * math.pi / 60), 0, 1) ** 2
    trough = np.clip(np.sin(phase(t, n) - tree.across * 2 * math.pi / 60 + .9), 0, 1) ** 3
    out = restep(colour, tree.leaf & (tree.speck < .5 * strength * crest), STEP_UP)
    out = restep(out, tree.leaf & (tree.speck > 1 - .3 * strength * trough), STEP_DOWN)
    lean = .9 * strength * height_weight(tree.h, 1.2) * crest
    _, c = warp(out, tree.leaf, lean, 0)
    return np.where(tree.leaf[..., None], c, out)


def sprays(tree, colour, t, n=FRAMES, amp=.9, bias=0.0):
    """Each lit spray (a few pixels) slides a pixel on its own beat; the shade under it shows."""
    rnd = np.random.default_rng(9)
    out = colour.copy()
    moves = []
    for ys, xs in tree.tufts:
        ph = rnd.uniform(0, 2 * math.pi)
        dx = int(np.rint(amp * math.sin(phase(t, n) + ph) + bias))
        dy = int(np.rint(.6 * amp * math.sin(2 * phase(t, n) + 1.7 * ph)))
        if dx or dy:
            moves.append((ys, xs, dx, dy))
            out[ys, xs] = [STEP_DOWN[tuple(p)] for p in colour[ys, xs]]
    for ys, xs, dx, dy in moves:
        ny, nx = ys + dy, xs + dx
        ok = (ny >= 0) & (ny < tree.h) & (nx >= 0) & (nx < tree.w)
        ok[ok] = tree.leaf[ny[ok], nx[ok]]
        out[ny[ok], nx[ok]] = colour[ys[ok], xs[ok]]
    return out


# First round: A-E.

def shear(tree, t):
    m, c = warp(tree.colour, tree.leaf, 2.2 * height_weight(tree.h, 1.4) * math.sin(phase(t)), 0)
    return tree.frame(m, c)


def ripple(tree, t):
    y = np.arange(tree.h, dtype=float)[:, None]
    dx = 1.3 * height_weight(tree.h, .6) * np.sin(phase(t) + y / 9) + np.zeros((1, tree.w))
    return tree.frame(*warp(tree.colour, tree.leaf, dx, 0))


def lobes(tree, t):
    dx, dy, total = (np.zeros((tree.h, tree.w)) for _ in range(3))
    for cx, cy, ph, amp in CLUMPS:
        wgt = np.exp(-((tree.xs - cx - MARGIN) ** 2 + (tree.ys - cy) ** 2) / (2 * 9 ** 2))
        dx += wgt * amp * 1.3 * math.sin(phase(t) + ph)
        dy += wgt * amp * .8 * math.sin(2 * phase(t) + ph)
        total += wgt
    return tree.frame(*warp(tree.colour, tree.leaf, dx / total, dy / total * height_weight(tree.h, .5)))


def fringe(tree, t):
    return tree.frame(*flutter(tree, tree.colour, t))


def gust(tree, t):
    front = -20 + (tree.w + 40) * t / FRAMES
    across = tree.across - front - 18
    out = restep(tree.colour, tree.leaf & (tree.speck < .55 * np.exp(-(across / 7) ** 2)), STEP_UP)
    out = restep(out, tree.leaf & (tree.speck > 1 - .35 * np.exp(-((across + 9) / 5) ** 2)), STEP_DOWN)
    lean = 1.1 * height_weight(tree.h, 1.2) * np.exp(-(across / 20) ** 2)
    return tree.frame(*warp(out, tree.leaf, lean, 0))


# Second round: C, D and E mixed, small pieces moving everywhere instead of big lobes.

def rim_sparkle(tree, t):
    return tree.frame(*flutter(tree, sparkle(tree, tree.colour, t), t))


def rim_waves(tree, t):
    return tree.frame(*flutter(tree, waves(tree, tree.colour, t), t))


def rim_sprays(tree, t):
    return tree.frame(*flutter(tree, sprays(tree, tree.colour, t), t))


def rim_sprays_waves(tree, t):
    c = waves(tree, sprays(tree, tree.colour, t, amp=.8), t, strength=.7)
    return tree.frame(*flutter(tree, c, t))


def gust_cycle(tree, t):
    n, e = GUST_FRAMES, envelope(t, GUST_FRAMES)
    c = sprays(tree, tree.colour, t, n, amp=.5 + .5 * e, bias=.55 * e)
    c = sparkle(tree, c, t, n, density=.08)
    c = waves(tree, c, t, n, strength=.8 * e)
    return tree.frame(*flutter(tree, c, t, n, amp=.6 + .5 * e, bias=.6 * e))


# 3D: the tree's model gives which clump of leaves is in front at each pixel, how it moves, and how its faces tilt.

class Model3D:
    def __init__(self, tree):
        import base_model as model
        import render as renderer
        import scene_props3d as props
        import structures

        self.tree = tree
        self.R = renderer
        renderer.ELEVATION = math.radians(30)
        self.view = renderer.view(renderer.AZIMUTH[structures.VIEW])
        self.scale = structures.SCALE
        props.PROPS[NAME]()
        leaf_colours = {props.BROAD, props.BROAD_LIGHT, props.BROAD_DARK}
        self.polys = [(v.astype(float), c) for v, c in model.meshes]
        self.clump = np.full(len(self.polys), -1)
        k = 0
        for i, (_, c) in enumerate(self.polys):
            if c in leaf_colours:
                self.clump[i] = k // 96  # base_model.ell makes 8x12 quads
                k += 1
        self.nclump = self.clump.max() + 1

        rest, normals = self.raster(self.polys)
        hit = np.nonzero(rest >= 0)
        self.rbox = (hit[1].min() / SS, hit[0].min() / SS, (hit[1].max() + 1) / SS, (hit[0].max() + 1) / SS)
        body = np.nonzero(tree.leaf | tree.trunk)
        self.dbox = (body[1].min(), body[0].min(), body[1].max() + 1, body[0].max() + 1)
        self.kx = (self.rbox[2] - self.rbox[0]) / (self.dbox[2] - self.dbox[0])
        self.ky = (self.rbox[3] - self.rbox[1]) / (self.dbox[3] - self.dbox[1])
        rx = self.rbox[0] + (tree.xs + .5 - self.dbox[0]) * self.kx
        ry = self.rbox[1] + (tree.ys + .5 - self.dbox[1]) * self.ky
        self.ix = np.clip((rx * SS).astype(int), 0, rest.shape[1] - 1)
        self.iy = np.clip((ry * SS).astype(int), 0, rest.shape[0] - 1)
        self.rest = rest[self.iy, self.ix]
        self.normals = normals

        _, r, _ = self.view
        self.centre = np.zeros((self.nclump, 3))
        for i, (v, _) in enumerate(self.polys):
            if self.clump[i] >= 0:
                self.centre[self.clump[i]] += v.mean(0) / 96
        cx = self.dbox[0] + (150 + .5 + self.centre @ r * self.scale - self.rbox[0]) / self.kx
        z = self.centre[:, 2]
        self.clump_x = cx
        self.clump_lift = (z - z.min()) / (z.max() - z.min())
        rnd = np.random.default_rng(21)
        self.clump_phase = rnd.uniform(0, 2 * math.pi, self.nclump)
        _, ncell = tree.cells(3, 13)
        self.cell_phase = rnd.uniform(0, 2 * math.pi, ncell)
        self.cell_rate = rnd.integers(1, 3, ncell)
        axis = rnd.normal(size=(ncell, 3))
        self.cell_axis = axis / np.linalg.norm(axis, axis=1, keepdims=True)
        light = np.array(renderer.LIGHT)
        self.light = light / np.linalg.norm(light)

    def raster(self, polys):
        faces, index, normals = [], [], np.zeros((len(polys), 3))
        t, r, u = self.view
        for i, (v, c) in enumerate(polys):
            n = np.cross(v[1] - v[0], v[2] - v[0])
            ln = np.linalg.norm(n)
            if ln > 1e-9:
                n = n / ln
                faces.append((v, n, c))
                index.append(i)
                n = n if n @ t >= 0 else -n
                normals[i] = (n @ r, n @ u, n @ t)
        _, ids, _, _ = self.R.rasterize(faces, t, r, u, self.scale * SS, ((150 + .5) * SS, (290 + 1) * SS),
                                        (300 * SS, 340 * SS))
        index = np.array(index)
        return np.where(ids >= 0, index[np.maximum(ids, 0)], -1), normals

    def shifts(self, t, n, amp):
        """Each clump's move in dot pixels: higher clumps more, a wind wave crossing left to right."""
        reach = (.6 + .6 * self.clump_lift) * amp
        ph = phase(t, n) + self.clump_phase - self.clump_x * 2 * math.pi / 90
        return reach * np.sin(ph), .5 * reach * np.sin(2 * phase(t, n) + self.clump_phase)

    def flow(self, t, n, amp, bias=0.0):
        dx, dy = self.shifts(t, n, amp)
        dx = dx + bias * (.6 + .6 * self.clump_lift)
        _, r, u = self.view
        moved = []
        for i, (v, c) in enumerate(self.polys):
            k = self.clump[i]
            if k >= 0:
                v = v + r * (dx[k] * self.kx / self.scale) - u * (dy[k] * self.ky / self.scale)
            moved.append((v, c))
        ids, _ = self.raster(moved)
        ids = ids[self.iy, self.ix]
        k = np.where(ids >= 0, self.clump[np.maximum(ids, 0)], -2)
        fx = np.where(k >= 0, dx[np.maximum(k, 0)], np.where(k == -1, 0, np.nan))
        fy = np.where(k >= 0, dy[np.maximum(k, 0)], np.where(k == -1, 0, np.nan))
        for _ in range(6):  # pixels the model misses take their neighbours' motion
            for f in (fx, fy):
                pad = np.pad(f, 1, constant_values=np.nan)
                near = np.stack([pad[:-2, 1:-1], pad[2:, 1:-1], pad[1:-1, :-2], pad[1:-1, 2:]])
                with np.errstate(invalid="ignore"), warnings.catch_warnings():
                    warnings.simplefilter("ignore", RuntimeWarning)
                    fill = np.nanmean(near, axis=0)
                gap = np.isnan(f)
                f[gap] = fill[gap]
        return np.nan_to_num(fx), np.nan_to_num(fy)

    def tilt(self, colour, t, n, amp):
        """Small sprays on each face of the leaf clumps rock a little; those turning to the light go a step lighter."""
        tree = self.tree
        cell, _ = tree.cells(3, 13)
        poly = self.rest
        leafy = tree.leaf & (poly >= 0) & (self.clump[np.maximum(poly, 0)] >= 0)
        p = np.maximum(poly, 0)
        wave = .4 + .6 * (.5 + .5 * np.sin(phase(t, n) - tree.across * 2 * math.pi / 70))
        theta = amp * wave * np.sin(self.cell_rate[cell] * phase(t, n) + self.cell_phase[cell])
        base = self.normals[p]
        tilted = base + theta[..., None] * self.cell_axis[cell]
        tilted /= np.linalg.norm(tilted, axis=2, keepdims=True)
        delta = (tilted - base) @ self.light
        out = restep(colour, leafy & (delta > .1), STEP_UP)
        return restep(out, leafy & (delta < -.12), STEP_DOWN)


def model_clumps(tree, t):
    fx, fy = tree.model.flow(t, FRAMES, 1.4)
    return tree.frame(*warp(tree.colour, tree.leaf, fx, fy))


def model_tilt(tree, t):
    c = tree.model.tilt(tree.colour, t, FRAMES, .55)
    return tree.frame(tree.leaf, c)


def model_both(tree, t):
    c = tree.model.tilt(tree.colour, t, FRAMES, .45)
    fx, fy = tree.model.flow(t, FRAMES, 1.2)
    return tree.frame(*warp(c, tree.leaf, fx, fy))


def model_gust(tree, t):
    n, e = GUST_FRAMES, envelope(t, GUST_FRAMES)
    c = tree.model.tilt(tree.colour, t, n, .25 + .35 * e)
    fx, fy = tree.model.flow(t, n, .5 + .5 * e, bias=.6 * e)
    m, c = warp(c, tree.leaf, fx, fy)
    m2, c2 = flutter(tree, np.where(m[..., None], c, tree.colour), t, n, amp=.5 + .4 * e, bias=.4 * e)
    return tree.frame(m2 | m, np.where(m2[..., None], c2, c))


VARIANTS = [
    ("rim_sparkle", "F1 · 가장자리 살랑 + 잎 반짝임", rim_sparkle, FRAMES),
    ("rim_waves", "F2 · 가장자리 살랑 + 넓은 바람 물결", rim_waves, FRAMES),
    ("rim_sprays", "F3 · 가장자리 살랑 + 잎 조각 들썩", rim_sprays, FRAMES),
    ("rim_sprays_waves", "F4 · F3 + 바람 물결", rim_sprays_waves, FRAMES),
    ("gust_cycle", "F5 · 고요 → 돌풍 → 고요", gust_cycle, GUST_FRAMES),
    ("model_clumps", "G1 · 3D 잎덩이 45개 이동", model_clumps, FRAMES),
    ("model_tilt", "G2 · 3D 잎 면 기울기 명암", model_tilt, FRAMES),
    ("model_both", "G3 · G1 + G2", model_both, FRAMES),
    ("model_gust", "G4 · G3 + 가장자리, 돌풍 주기", model_gust, GUST_FRAMES),
    ("shear", "A · 수관째 휘청", shear, FRAMES),
    ("ripple", "B · 아래에서 위로 물결", ripple, FRAMES),
    ("clumps", "C · 잎덩이마다 따로", lobes, FRAMES),
    ("fringe", "D · 가장자리 잎만 살랑", fringe, FRAMES),
    ("gust", "E · 바람 띠(잎 뒤집힘)", gust, FRAMES),
]


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    tree = Tree()
    tree.model = Model3D(tree)
    Image.fromarray(tree.frame(tree.leaf, tree.colour)).save(OUT / "original.png")
    meta = {"cell": [tree.w, tree.h], "pivot_offset": [MARGIN, 0], "variants": []}
    only = set(sys.argv[1:])
    for key, title, fn, n in VARIANTS:
        if only and key not in only:
            continue
        sheet = np.concatenate([fn(tree, t) for t in range(n)], axis=1)
        Image.fromarray(sheet).save(OUT / f"{key}.png")
        print(key, n)
    meta["variants"] = [{"key": k, "title": title, "frames": n} for k, title, _, n in VARIANTS]
    (OUT / "variants.json").write_text(json.dumps(meta, ensure_ascii=False, indent=1) + "\n")


if __name__ == "__main__":
    main()
