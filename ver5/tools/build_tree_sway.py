#!/usr/bin/env python3
"""Leaf sway frames for the story-scene trees, packed for the game.

  uv run --with numpy --with pillow python ver5/tools/build_tree_sway.py

Each leafy tree of assets_candidate/tiles (stree_*, peach_bloom_*) gets FRAMES frames made from its own dot picture
(variant F3 of docs/tree-sway-variants.html): lit sprays of leaves or blossom, cut into pieces at most ~5px across,
slide a pixel each on their own beat and show the shade under them, and the tufts within 3px of the crown's rim bob in
and out. Bark, trunk and branch colours never move; only the picture's own colours are used and the 1px outline is
redrawn round every frame (4-neighbour, the rule that reproduces the original outline).

Writes assets_candidate/tiles/sway/<name>.png (a strip of frames) and packs them into assets/graphics/map/sway.png,
listed in assets/graphics/index.json under "anims" (the game's table of looping strips) with each tree's frame-0
rect, pivot, frame count and time, and "shadows" so every frame casts its own ground shadow. A tree of that name
placed on any scene map plays it."""
import json
import math
from pathlib import Path

import numpy as np
from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
CAND = ROOT / 'assets_candidate/tiles'
GFX = ROOT / 'ver5/assets/graphics'
FRAMES, MS = 8, 160
SIDE, TOP = 3, 2  # room for the rim to move into
WIDTH = 1024

OUTLINE = (11, 16, 27)
WOOD = {(61, 33, 29), (102, 55, 32), (142, 51, 38), (108, 88, 60), (168, 146, 102), (217, 198, 152)}
RAMPS = [  # dark to light
    [(34, 48, 26), (64, 88, 38), (104, 134, 48), (152, 178, 66), (204, 214, 104)],
    [(150, 62, 92), (215, 119, 144), (242, 167, 184), (255, 214, 222)],
    [(196, 210, 217), (241, 246, 245)],
    [(63, 138, 74)],
]
BAMBOO = {'stree_bamboo_0', 'stree_bamboo_1', 'stree_bamboo_thin', 'stree_sasa'}  # yellow-green stems stay, blue-green leaves sway
BAMBOO_WOOD = {(64, 88, 38), (104, 134, 48), (152, 178, 66), (204, 214, 104)}
BAMBOO_RAMP = [(42, 94, 53), (63, 138, 74), (121, 184, 102)]
BAMBOO_AMP = 1.7  # a bamboo spray bows further than a crown of leaves
HANGING = {'stree_willow'}  # strands hang to the ground, so the low leaves move as much as the high ones


def leafy_trees():
    tiles = json.load(open(CAND / 'tiles.json'))['sprites']
    return {n: t for n, t in sorted(tiles.items()) if n.startswith(('stree_', 'peach_bloom_')) and n != 'stree_dead'}


class Tree:
    def __init__(self, name):
        a = np.array(Image.open(CAND / f'{name}.png').convert('RGBA'))
        a = np.pad(a, ((TOP, 0), (SIDE, SIDE), (0, 0)))
        self.colour = a[..., :3].copy()
        self.h, self.w = a.shape[:2]
        solid = a[..., 3] > 0
        body = solid & ~self.match(OUTLINE)
        pad = np.pad(body, 1)
        rim = (pad[:-2, 1:-1] | pad[2:, 1:-1] | pad[1:-1, :-2] | pad[1:-1, 2:]) & ~body
        line = solid & self.match(OUTLINE) & ~rim  # dark lines the outline rule does not draw (plum twigs) stay put
        self.line = line
        self.amp = BAMBOO_AMP if name in BAMBOO else 1.0
        ramps = RAMPS + [BAMBOO_RAMP] if name in BAMBOO else RAMPS
        self.wood = body & np.any([self.match(c) for c in WOOD | (BAMBOO_WOOD if name in BAMBOO else set())], axis=0)
        self.wood |= self.bark_shade(body & ~self.wood)
        self.leaf = body & ~self.wood
        self.ys, self.xs = np.mgrid[0:self.h, 0:self.w]
        self.down, lit = {}, []
        for ramp in ramps:
            here = [c for c in ramp if self.match(c).any()]
            if not here:
                continue
            for lo, hi in zip([here[0]] + here, here):
                self.down[hi] = lo
            lit += here[-2:] if len(here) >= 4 else here[-1:]
        self.lit = self.leaf & np.any([self.match(c) for c in lit], axis=0)
        ys = np.nonzero(self.leaf)[0]
        base = ys.max() + 1
        rows = np.arange(self.h, dtype=float)[:, None]
        top = ys.min()
        if name in HANGING:  # strand tips swing more than the crown they hang from
            self.weight = .35 + .65 * np.clip((rows - top) / (base - top), 0, 1)
        else:
            self.weight = np.clip((base - rows) / base, 0, 1) ** .3
        self.centre = (self.xs[self.leaf].mean(), self.ys[self.leaf].mean())

    def bark_shade(self, leaf):
        """Runs of at most two leaf-coloured pixels in a row beside bark: the trunk's shaded side, not leaves."""
        shade = np.zeros_like(leaf)
        for y in range(self.h):
            x = 0
            while x < self.w:
                if not leaf[y, x]:
                    x += 1
                    continue
                end = x
                while end < self.w and leaf[y, end]:
                    end += 1
                beside = (x > 0 and self.wood[y, x - 1]) or (end < self.w and self.wood[y, end])
                if end - x <= 2 and beside:
                    shade[y, x:end] = True
                x = end
        return shade

    def match(self, c):
        return np.all(self.colour == c, axis=2)

    def rim(self):
        """Distance in pixels from the crown's edge (1..4), 99 further in."""
        dist = np.where(self.leaf, 99, 0)
        for d in range(1, 5):
            pad = np.pad(dist < d, 1)
            grow = pad[:-2, 1:-1] | pad[2:, 1:-1] | pad[1:-1, :-2] | pad[1:-1, 2:]
            dist = np.where((dist == 99) & grow, d, dist)
        return dist

    def sprays(self):
        """Connected lit pixels cut by jittered cells ~5px across; pieces of 2 pixels or more."""
        rnd = np.random.default_rng(11)
        gy, gx = np.mgrid[0:self.h + 5:5, 0:self.w + 5:5]
        pts = np.stack([gx.ravel(), gy.ravel()], 1) + rnd.uniform(-2, 2, (gx.size, 2))
        cell = ((self.xs[..., None] - pts[:, 0]) ** 2 + (self.ys[..., None] - pts[:, 1]) ** 2).argmin(axis=2)
        label = np.full(self.lit.shape, -1)
        count = 0
        for y, x in zip(*np.nonzero(self.lit)):
            if label[y, x] >= 0:
                continue
            stack, label[y, x] = [(y, x)], count
            while stack:
                cy, cx = stack.pop()
                for ny, nx in ((cy - 1, cx), (cy + 1, cx), (cy, cx - 1), (cy, cx + 1)):
                    if 0 <= ny < self.h and 0 <= nx < self.w and self.lit[ny, nx] and label[ny, nx] < 0:
                        label[ny, nx] = count
                        stack.append((ny, nx))
            count += 1
        key = np.where(self.lit, label * len(pts) + cell, -1)
        return [np.nonzero(key == k) for k in np.unique(key[self.lit]) if (key == k).sum() >= 2]

    def frames(self):
        rim, pieces = self.rim(), self.sprays()
        rnd = np.random.default_rng(9)
        budget, kept = .25 * self.leaf.sum(), []  # a crown mostly in its lightest colour (plum) moves no more than others
        for i in rnd.permutation(len(pieces)):
            if budget >= len(pieces[i][0]):
                kept.append(pieces[i])
                budget -= len(pieces[i][0])
        pieces = kept
        phases = rnd.uniform(0, 2 * math.pi, len(pieces))
        ang = np.arctan2(self.ys - self.centre[1], self.xs - self.centre[0])
        tuft = np.sin(ang * 7 + 1.7 * np.sin(ang * 3))
        wobble = ((rim <= 3) | ~self.leaf) * self.weight
        out = []
        for t in range(FRAMES):
            ph = 2 * math.pi * t / FRAMES
            c = self.colour.copy()
            moves = []
            for (ys, xs), p in zip(pieces, phases):
                dx, dy = int(np.rint(.9 * self.amp * math.sin(ph + p))), int(np.rint(.54 * self.amp * math.sin(2 * ph + 1.7 * p)))
                if dx or dy:
                    moves.append((ys, xs, dx, dy))
                    c[ys, xs] = [self.down[tuple(v)] for v in self.colour[ys, xs]]
            for ys, xs, dx, dy in moves:
                ny, nx = ys + dy, xs + dx
                ok = (ny >= 0) & (ny < self.h) & (nx >= 0) & (nx < self.w)
                ok[ok] = self.leaf[ny[ok], nx[ok]]
                c[ny[ok], nx[ok]] = self.colour[ys[ok], xs[ok]]
            dx = 1.2 * self.amp * wobble * np.sin(ph + tuft * 2.2)
            dy = .9 * self.amp * wobble * np.sin(2 * ph + tuft * 3.1 + 1)
            sx = np.clip(self.xs - np.rint(dx).astype(int), 0, self.w - 1)
            sy = np.clip(self.ys - np.rint(dy).astype(int), 0, self.h - 1)
            inner = self.leaf & (rim > 3)
            leaf = self.leaf[sy, sx] | inner
            c = np.where(inner[..., None], c, c[sy, sx])
            out.append(self.compose(leaf, c))
        return out

    def compose(self, leaf, leaf_colour):
        rgba = np.zeros((self.h, self.w, 4), np.uint8)
        body = self.wood | leaf
        pad = np.pad(body, 1)
        edge = ((pad[:-2, 1:-1] | pad[2:, 1:-1] | pad[1:-1, :-2] | pad[1:-1, 2:]) & ~body) | (self.line & ~leaf)
        rgba[edge] = (*OUTLINE, 255)
        rgba[self.wood, :3] = self.colour[self.wood]
        rgba[leaf, :3] = leaf_colour[leaf]
        rgba[body, 3] = 255
        return rgba


def main():
    out_dir = CAND / 'sway'
    out_dir.mkdir(exist_ok=True)
    strips = {}
    trees = leafy_trees()
    for name in trees:
        tree = Tree(name)
        assert (tree.compose(tree.leaf, tree.colour)[TOP:, SIDE:-SIDE] ==
                np.array(Image.open(CAND / f'{name}.png').convert('RGBA'))).all(), f'{name}: outline rule'
        strip = Image.fromarray(np.concatenate(tree.frames(), axis=1))
        strip.save(out_dir / f'{name}.png')
        strips[name] = strip

    order = sorted(strips, key=lambda n: (-strips[n].height, n))
    x = y = row = 0
    place = {}
    for n in order:
        w, h = strips[n].size
        if x + w > WIDTH:
            x, y, row = 0, y + row + 1, 0
        place[n] = (x, y)
        x += w + 1
        row = max(row, h)
    sheet = Image.new('RGBA', (WIDTH, y + row + 1), (0, 0, 0, 0))
    sprites = {}
    for n in strips:
        sheet.paste(strips[n], place[n])
        px, py = trees[n]['pivot']
        sprites[n] = {'image': 'map/sway.png', 'rect': [*place[n], strips[n].width // FRAMES, strips[n].height],
                      'frames': FRAMES, 'ms': MS, 'pivot': [px + SIDE, py + TOP], 'shadows': True}
    sheet.save(GFX / 'map/sway.png', optimize=True)

    path = GFX / 'index.json'
    index = json.load(open(path))
    anims = {n: e for n, e in index.get('anims', {}).items() if e['image'] != 'map/sway.png'}  # others' strips
    index['anims'] = dict(sorted({**anims, **sprites}.items()))
    json.dump(index, open(path, 'w'), ensure_ascii=False, indent=1)
    open(path, 'a').write('\n')
    print('sway', len(sprites), 'trees', sheet.size)


if __name__ == '__main__':
    main()
