#!/usr/bin/env python3
"""Turn a generated four-tree sway sheet into an aligned transparent source strip.

The magenta chroma key is removed, the four trees are found by column gaps,
frames are aligned on their roots, and the trunk rows below the lowest leaf are
copied from frame 0 so the ground contact never moves.

  python3 tools/prepare_tree_sway.py tools/spritetool/assets/tree-sway/tree1-generated.png \
      tools/spritetool/assets/tree-sway/tree1.png
"""
import sys

from PIL import Image


def key_out(im):
    px = im.load()
    for y in range(im.height):
        for x in range(im.width):
            r, g, b, _ = px[x, y]
            if r > g + 30 and b > g + 30:
                px[x, y] = (0, 0, 0, 0)


def frame_columns(im):
    px = im.load()
    counts = [sum(px[x, y][3] > 0 for y in range(im.height)) for x in range(im.width)]
    runs, start = [], None
    for x, c in enumerate(counts + [0]):
        if c and start is None:
            start = x
        elif not c and start is not None:
            if x - start > 40:
                runs.append((start, x))
            start = None
    while len(runs) < 4:  # neighbouring canopies touch: split the widest run at its thinnest column
        i = max(range(len(runs)), key=lambda i: runs[i][1] - runs[i][0])
        a, b = runs[i]
        m = min(range(a + (b - a)//3, b - (b - a)//3), key=lambda x: counts[x])
        runs[i:i + 1] = [(a, m), (m + 1, b)]
    if len(runs) != 4:
        sys.exit(f'expected 4 trees, found {len(runs)}')
    return runs


def is_leaf(p):
    r, g, _, a = p
    return a and g > r + 5 and g > 60


def main(src, dst):
    im = Image.open(src).convert('RGBA')
    key_out(im)
    frames = []
    for x0, x1 in frame_columns(im):
        f = im.crop((x0, 0, x1, im.height))
        f = f.crop(f.getbbox())
        fp = f.load()
        xs = [x for y in range(int(f.height*.94), f.height) for x in range(f.width) if fp[x, y][3]]
        frames.append((f, sum(xs)/len(xs)))
    left = max(c for _, c in frames)
    right = max(f.width - c for f, c in frames)
    cw, ch = int(left + right) + 2, max(f.height for f, _ in frames) + 2
    cells = []
    for f, c in frames:
        cell = Image.new('RGBA', (cw, ch))
        cell.paste(f, (round(left - c) + 1, ch - 1 - f.height))
        cells.append(cell)
    leaf_bottom = max(y for c in cells for y in range(ch) for x in range(cw) if is_leaf(c.getpixel((x, y))))
    trunk = cells[0].crop((0, leaf_bottom + 1, cw, ch))
    strip = Image.new('RGBA', (cw*4, ch))
    for i, c in enumerate(cells):
        c.paste(trunk, (0, leaf_bottom + 1))
        strip.paste(c, (i*cw, 0))
    strip.save(dst)


if __name__ == '__main__':
    main(*sys.argv[1:])
