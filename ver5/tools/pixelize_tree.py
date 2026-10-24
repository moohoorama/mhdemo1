#!/usr/bin/env python3
"""Turn the painted scene trees into their dot sprites: python3 tools/pixelize_tree.py [name ...]

The scene trees (pines, broadleaves, willow, old tree, burnt tree, bushes, plums, peaches) were first modelled in
3D (tools/unit3d/scene_props3d.py); those renders, upscaled, were the references codex painted each tree from
(the item-icon route), and this converts the paintings back into dots on the 3D sprite's exact canvas, so the size,
the pivot, the planting (scene_maps/trees.py) and the engine's width-based ground shadow stay as they were.

assets_candidate/tiles/sources/trees.json lists, per tree, its painting (512 px on flat magenta) and its palette
(the 3D render's colours, kept there so the render is no longer needed; the darkest is the outline). For each tree
the magenta is keyed out, the painting cropped to the tree and stretched onto the canvas less a 1px ring, every
source pixel snapped to the palette in CIE Lab, and each dot takes the colour most of its BxB block has. Knobs:
  solid    share of a dot's block that must be covered (default .5)
  dark     share of the block the painting's dark lines need to win a dot inside the tree (default .25; lower keeps
           the gaps between willow fronds or blossom masses)
  penalty  {colour: Lab distance} added when snapping to that colour, to use it more sparingly
  bark, inner, speck, foot
           bark colours; bark specks under speck dots (default 12), or with foot every bark area not joined to the
           foot, are not bark (a painted dark-red rim round blossom): on the rim they are left to the outline ring,
           inside the tree they take inner
Lone specks are cleaned and the silhouette ringed with the outline. Writes assets_candidate/tiles/<name>.png; then
python3 tools/build_extra.py packs them. Requires Pillow.
"""
import json
import sys
from collections import Counter
from pathlib import Path

from PIL import Image, ImageFilter

CAND = Path(__file__).resolve().parents[2] / 'assets_candidate'
B = 6  # source px per dot when snapping and voting
FOUR = [(1, 0), (-1, 0), (0, 1), (0, -1)]
EIGHT = [(dx, dy) for dy in (-1, 0, 1) for dx in (-1, 0, 1) if dx or dy]


def keyed(path):
    im = Image.open(path).convert('RGBA')
    px = im.load()
    for y in range(im.height):
        for x in range(im.width):
            r, g, b, _ = px[x, y]
            if r > 120 and b > 120 and g < min(r, b) - 60:  # magenta and its fringe
                px[x, y] = (0, 0, 0, 0)
    return im.crop(im.getbbox())


def lab(rgb):
    def lin(v):
        v /= 255
        return v / 12.92 if v <= .04045 else ((v + .055) / 1.055) ** 2.4

    def f(t):
        return t ** (1 / 3) if t > .008856 else 7.787 * t + 16 / 116

    r, g, b = (lin(v) for v in rgb)
    X = (.4124 * r + .3576 * g + .1805 * b) / .9505
    Y = .2126 * r + .7152 * g + .0722 * b
    Z = (.0193 * r + .1192 * g + .9505 * b) / 1.089
    return 116 * f(Y) - 16, 500 * (f(X) - f(Y)), 200 * (f(Y) - f(Z))


def rgb(h):
    return tuple(int(h[i:i + 2], 16) for i in (1, 3, 5))


def pixelize(t, W, H):
    pal = {rgb(c): lab(rgb(c)) for c in t['palette']}
    k = min(pal, key=lambda c: pal[c][0])
    penalty = {rgb(c): v for c, v in t.get('penalty', {}).items()}
    solid, dark = t.get('solid', .5), t.get('dark', .25)
    cache = {}

    def nearest(c):
        if c not in cache:
            L = lab(c)
            cache[c] = min(pal, key=lambda p: sum((a - b) ** 2 for a, b in zip(L, pal[p])) ** .5 + penalty.get(p, 0))
        return cache[c]

    im = keyed(CAND / 'tiles' / t['source'])
    w, h = W - 2, H - 2
    big = im.resize((w * B, h * B), Image.LANCZOS)
    src = big.convert('RGB').filter(ImageFilter.MedianFilter(3)).load()
    alpha = big.getchannel('A').load()
    grid = [[None] * W for _ in range(H)]
    for y in range(h):
        for x in range(w):
            votes, clear = Counter(), 0
            for sy in range(y * B, y * B + B):
                for sx in range(x * B, x * B + B):
                    if alpha[sx, sy] < 128:
                        clear += 1
                    else:
                        votes[nearest(src[sx, sy])] += 1
            if B * B - clear < solid * B * B:
                continue
            paint = {c: v for c, v in votes.items() if c != k}
            best = max(paint, key=paint.get) if paint else k
            if paint and (paint[best] < B * B / 4 and votes[k] > paint[best] or dark < .25 and votes[k] >= dark * B * B):
                best = k
            if best != k or clear * 2 < B * B:  # thin dark lines alone on the rim stay clear; the ring redraws them
                grid[y + 1][x + 1] = best

    def around(x, y, cells):
        return [grid[y + dy][x + dx] for dx, dy in cells if 0 <= x + dx < W and 0 <= y + dy < H]

    for y in range(H):
        for x in range(W):
            c = grid[y][x]
            if c is None:
                continue
            ring = around(x, y, EIGHT)
            if all(v is None for v in ring):
                grid[y][x] = None
            elif c not in around(x, y, FOUR):
                solid_ = [v for v in ring if v is not None]
                best = max(set(solid_), key=solid_.count)
                if solid_.count(best) >= 4:
                    grid[y][x] = best
    bark = {rgb(c) for c in t.get('bark', [])}
    inner = rgb(t['inner']) if 'inner' in t else k
    speck, foot = t.get('speck', 12), t.get('foot', False)
    seen = set()
    for y0 in range(H):
        for x0 in range(W):
            if grid[y0][x0] not in bark or (x0, y0) in seen:
                continue
            part, todo = [], [(x0, y0)]
            while todo:
                x, y = todo.pop()
                if (x, y) in seen or not (0 <= x < W and 0 <= y < H) or grid[y][x] not in bark:
                    continue
                seen.add((x, y))
                part.append((x, y))
                todo += [(x + dx, y + dy) for dx, dy in FOUR]
            if len(part) < speck or foot and max(y for _, y in part) < H - 4:
                for x, y in part:
                    grid[y][x] = None if None in around(x, y, FOUR) or len(around(x, y, FOUR)) < 4 else inner
    edge = [(x, y) for y in range(H) for x in range(W)
            if grid[y][x] is None and any(v is not None and v != k for v in around(x, y, FOUR))]
    for x, y in edge:
        grid[y][x] = k
    out = Image.new('RGBA', (W, H))
    for y in range(H):
        for x in range(W):
            if grid[y][x] is not None:
                out.putpixel((x, y), (*grid[y][x], 255))
    return out


def main():
    trees = json.load(open(CAND / 'tiles/sources/trees.json'))['trees']
    tiles = json.load(open(CAND / 'tiles/tiles.json'))['sprites']
    for name in sys.argv[1:] or trees:
        W, H = tiles[name]['size']
        pixelize(trees[name], W, H).save(CAND / tiles[name]['file'])
        print(name, (W, H))


if __name__ == '__main__':
    main()
