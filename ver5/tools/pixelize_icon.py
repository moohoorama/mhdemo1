#!/usr/bin/env python3
"""Turn a painted UI icon into the UI sheet's dot grid: uv run --with pillow python tools/pixelize_icon.py item_leather ...

For each named sprite of tools/spritetool/assets/ver5-ui whose settings have a "source" (a painting
on flat magenta, e.g. drawn with the codex-image skill from the current icon), this keys out the
magenta, crops to the object and fits it in "fit" dots (default 30) of the sprite's dot grid; a dot
is solid when at least "solid" of its block is (default .5, lower for thin blades and shafts). Every
source pixel is first snapped to the sprite's palette (CIE Lab distance; only the letters in "use"
when given) or to the outline "k" for the painting's own dark lines, then each dot takes the colour
most of its block has, so a thin cord over a plate never averages into mud; a colour that covers a
quarter of the block beats the outline, so thin bright strings and blades survive. Lone specks are cleaned and the silhouette is ringed
with a 1px "k" outline. The grid is written to pixels.json; build the sheet with build_effects.py.
"""
import json
import sys
from pathlib import Path

from PIL import Image, ImageFilter

UI = Path(__file__).resolve().parents[2] / "tools/spritetool/assets/ver5-ui"
B = 6  # source px per dot when snapping and voting


def keyed(path):
    im = Image.open(path).convert("RGBA")
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


def pixelize(s, n):
    palette = s["layers"]["icon"]["palette"]
    use = s.get("use")
    pal = {c: lab(tuple(int(v[i:i + 2], 16) for i in (1, 3, 5))) for c, v in palette.items()
           if c == "k" or use is None or c in use}
    cache = {}

    def nearest(rgb):
        if rgb not in cache:
            L = lab(rgb)
            cache[rgb] = min(pal, key=lambda c: sum((a - b) ** 2 for a, b in zip(L, pal[c])))
        return cache[rgb]

    im = keyed(UI / s["source"])
    k = s.get("fit", 30) / max(im.width, im.height)
    w, h = max(1, round(im.width * k)), max(1, round(im.height * k))
    big = im.resize((w * B, h * B), Image.LANCZOS)
    rgb = big.convert("RGB").filter(ImageFilter.MedianFilter(3)).load()
    alpha = big.getchannel("A").load()
    grid = [["."] * n for _ in range(n)]
    ox, oy = (n - w) // 2, (n - h) // 2
    for y in range(h):
        for x in range(w):
            votes, clear = {}, 0
            for sy in range(y * B, y * B + B):
                for sx in range(x * B, x * B + B):
                    if alpha[sx, sy] < 128:
                        clear += 1
                    else:
                        c = nearest(rgb[sx, sy])
                        votes[c] = votes.get(c, 0) + 1
            if B * B - clear >= s.get("solid", .5) * B * B:
                paint = {c: v for c, v in votes.items() if c != "k"}
                best = max(paint, key=paint.get) if paint else "k"
                if paint and paint[best] < B * B / 4 and votes.get("k", 0) > paint[best]:
                    best = "k"
                if best != "k" or clear * 2 < B * B:  # thin dark lines alone stay clear; the ring redraws them
                    grid[oy + y][ox + x] = best

    def around(x, y, cells):
        return [grid[y + dy][x + dx] for dx, dy in cells if 0 <= x + dx < n and 0 <= y + dy < n]

    eight = [(dx, dy) for dy in (-1, 0, 1) for dx in (-1, 0, 1) if dx or dy]
    four = [(1, 0), (-1, 0), (0, 1), (0, -1)]
    for y in range(n):
        for x in range(n):
            c = grid[y][x]
            if c == ".":
                continue
            ring = around(x, y, eight)
            if all(v == "." for v in ring):
                grid[y][x] = "."
            elif c not in around(x, y, four):
                solid = [v for v in ring if v != "."]
                best = max(set(solid), key=solid.count)
                if solid.count(best) >= 4:
                    grid[y][x] = best
    edge = [(x, y) for y in range(n) for x in range(n)
            if grid[y][x] == "." and any(v != "." for v in around(x, y, four))]
    for x, y in edge:
        grid[y][x] = "k"
    margin = s["anchor"][0] // s["dot"]
    for y in range(n):
        for x in range(n):
            assert grid[y][x] == "." or margin <= x < n - margin and margin <= y < n - margin, f"pixel in the clear margin at {x},{y}"
    return ["".join(r) for r in grid]


settings = json.loads((UI / "settings.json").read_text())
pixels = json.loads((UI / "pixels.json").read_text())
for name in sys.argv[1:]:
    s = settings["sprites"][name]
    pixels[name] = [{"icon": pixelize(s, s["cell"][0] // s["dot"])}]
    print(name)
(UI / "pixels.json").write_text(json.dumps(pixels, indent=1) + "\n")
