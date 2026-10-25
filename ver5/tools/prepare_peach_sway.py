#!/usr/bin/env python3
"""Align eight painted poses, apply the scene-tree palette, and hold the wood still."""

import json
import statistics
import sys
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "ver5/tools"))

import pixelize_tree  # noqa: E402

SOURCE = ROOT / "tools/spritetool/assets/scene-peach-sway"
FRAMES = 8
COLS = 4
PAD = 40
SIZE = (93, 96)
PAINT_SIZE = (86, 96)
PAINT_OFFSET = (7, 0)
BARK = {(61, 33, 29), (102, 55, 32)}
OUTLINE = (11, 16, 27)


def root_point(image):
    alpha = image.getchannel("A")
    box = alpha.getbbox()
    if box is None:
        raise ValueError("painted frame has no tree")
    xs = [x for y in range(box[3] - 24, box[3]) for x in range(box[0], box[2])
          if alpha.getpixel((x, y)) > 127]
    return round(statistics.median(xs)), box[3] - 1


def wood_pixels(image):
    pixels = image.load()
    wood = {(x, y) for y in range(image.height) for x in range(image.width)
            if pixels[x, y][3] and pixels[x, y][:3] in BARK}
    edge = {(x + dx, y + dy) for x, y in wood for dx in (-1, 0, 1) for dy in (-1, 0, 1)
            if 0 <= x + dx < image.width and 0 <= y + dy < image.height}
    wood.update((x, y) for x, y in edge if pixels[x, y][3] and pixels[x, y][:3] == OUTLINE)
    return wood


def fixed_wood(frames):
    base = frames[0]
    base_wood = wood_pixels(base)
    for image in frames[1:]:
        current = wood_pixels(image)
        pixels = image.load()
        for x, y in current:
            # Replace a displaced branch with its surrounding blossom colour when possible.
            nearby = [pixels[nx, ny] for nx, ny in ((x - 1, y), (x + 1, y), (x, y - 1), (x, y + 1))
                      if 0 <= nx < image.width and 0 <= ny < image.height and (nx, ny) not in current
                      and pixels[nx, ny][3] and pixels[nx, ny][:3] not in BARK]
            pixels[x, y] = max(sorted(set(nearby)), key=nearby.count) if nearby and y < 70 else (0, 0, 0, 0)
        for x, y in base_wood:
            pixels[x, y] = base.getpixel((x, y))
        image.paste(base.crop((0, 73, SIZE[0], SIZE[1])), (0, 73))


def main():
    sheet = Image.open(SOURCE / "painted-sheet.png").convert("RGBA")
    cells = []
    for n in range(FRAMES):
        col, row = n % COLS, n // COLS
        box = (round(col * sheet.width / COLS), round(row * sheet.height / 2),
               round((col + 1) * sheet.width / COLS), round((row + 1) * sheet.height / 2))
        cells.append(pixelize_tree.keyed(sheet.crop(box), trim=False))

    roots = [root_point(cell) for cell in cells]
    reference = roots[0]
    aligned = []
    width, height = max(cell.width for cell in cells) + 2 * PAD, max(cell.height for cell in cells) + 2 * PAD
    for cell, root in zip(cells, roots):
        canvas = Image.new("RGBA", (width, height))
        canvas.alpha_composite(cell, (PAD + reference[0] - root[0], PAD + reference[1] - root[1]))
        aligned.append(canvas)

    boxes = [cell.getbbox() for cell in aligned]
    union = (min(b[0] for b in boxes) - 3, min(b[1] for b in boxes) - 3,
             max(b[2] for b in boxes) + 3, max(b[3] for b in boxes) + 3)
    settings = json.load(open(ROOT / "assets_candidate/tiles/sources/trees.json"))["trees"]["peach_bloom_0"]
    dots = []
    for source in aligned:
        painted = pixelize_tree.pixelize(settings, *PAINT_SIZE, source=source.crop(union), trim=False)
        cell = Image.new("RGBA", SIZE)
        cell.alpha_composite(painted, PAINT_OFFSET)
        dots.append(cell)
    fixed_wood(dots)

    output = Image.new("RGBA", (COLS * SIZE[0], 2 * SIZE[1]))
    for n, cell in enumerate(dots):
        output.alpha_composite(cell, ((n % COLS) * SIZE[0], (n // COLS) * SIZE[1]))
    output.save(SOURCE / "dot-sheet.png")
    print(SOURCE / "dot-sheet.png", output.size)


if __name__ == "__main__":
    main()
