#!/usr/bin/env python3
"""Turn painted 2×2 scene-tree sheets into four root-aligned, palette-limited frames."""

import json
import math
import statistics
import sys
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "ver5/tools"))

import pixelize_tree  # noqa: E402

SOURCE = ROOT / "tools/spritetool/assets/scene-tree-sway"
COLS = ROWS = 2
FRAMES = 4
PAD = 48
WOOD_HEX = {"#3d211d", "#663720", "#8e3326", "#6c583c", "#a89266", "#d9c698"}


def rgb(value):
    return tuple(bytes.fromhex(value[1:]))


def root_point(image):
    alpha = image.getchannel("A")
    box = alpha.getbbox()
    if box is None:
        raise ValueError("a frame has no tree")
    band = max(3, round((box[3] - box[1]) * 0.06))
    xs = [x for y in range(max(box[1], box[3] - band), box[3])
          for x in range(box[0], box[2]) if alpha.getpixel((x, y)) > 127]
    return round(statistics.median(xs)), box[3] - 1


def crop_axis(low, high, anchor, target, extent):
    fraction = max(0.05, min(0.95, (target - 1) / (extent - 2)))
    left = anchor - low + 5
    right = high - anchor + 5
    width = math.ceil(max(left / fraction, right / (1 - fraction))) + 2
    start = round(anchor - fraction * width)
    return start, start + width


def wood_mask(image, bark, outline):
    pixels = image.load()
    wood = {(x, y) for y in range(image.height) for x in range(image.width)
            if pixels[x, y][3] and pixels[x, y][:3] in bark}
    edge = {(x + dx, y + dy) for x, y in wood for dx in (-1, 0, 1) for dy in (-1, 0, 1)
            if 0 <= x + dx < image.width and 0 <= y + dy < image.height}
    wood.update((x, y) for x, y in edge if pixels[x, y][3] and pixels[x, y][:3] == outline)
    return wood


def fix_wood(frames, settings, name):
    bark = {rgb(color) for color in settings["palette"] if color.lower() in WOOD_HEX}
    if not bark:
        return
    outline = min((rgb(color) for color in settings["palette"]), key=lambda color: sum(color))
    base = frames[0]
    base_wood = wood_mask(base, bark, outline)
    for image in frames[1:]:
        current = wood_mask(image, bark, outline)
        pixels = image.load()
        for x, y in current:
            nearby = [pixels[nx, ny] for nx, ny in ((x - 1, y), (x + 1, y), (x, y - 1), (x, y + 1))
                      if 0 <= nx < image.width and 0 <= ny < image.height and (nx, ny) not in current
                      and pixels[nx, ny][3] and pixels[nx, ny][:3] not in bark]
            pixels[x, y] = max(sorted(set(nearby)), key=nearby.count) if nearby else (0, 0, 0, 0)
        for x, y in base_wood:
            pixels[x, y] = base.getpixel((x, y))
        if name != "stree_willow":
            foot = round(image.height * 0.82)
            image.paste(base.crop((0, foot, image.width, image.height)), (0, foot))


def prepare(name, settings, tiles):
    size = tuple(tiles[name]["size"])
    directory = SOURCE / name
    sheet = Image.open(directory / "painted-sheet.png").convert("RGBA")
    cells = []
    for frame in range(FRAMES):
        col, row = frame % COLS, frame // COLS
        box = (round(col * sheet.width / COLS), round(row * sheet.height / ROWS),
               round((col + 1) * sheet.width / COLS), round((row + 1) * sheet.height / ROWS))
        cells.append(pixelize_tree.keyed(sheet.crop(box), trim=False))

    roots = [root_point(cell) for cell in cells]
    common = roots[0]
    width, height = max(cell.width for cell in cells) + 2 * PAD, max(cell.height for cell in cells) + 2 * PAD
    aligned = []
    for cell, root in zip(cells, roots):
        canvas = Image.new("RGBA", (width, height))
        canvas.alpha_composite(cell, (PAD + common[0] - root[0], PAD + common[1] - root[1]))
        aligned.append(canvas)
    union = [cell.getbbox() for cell in aligned]
    target = root_point(Image.open(ROOT / "assets_candidate/tiles" / f"{name}.png").convert("RGBA"))
    anchor = (common[0] + PAD, common[1] + PAD)
    x0, x1 = crop_axis(min(b[0] for b in union), max(b[2] for b in union), anchor[0], target[0], size[0])
    y0, y1 = min(b[1] for b in union) - 3, max(b[3] for b in union) + 3
    frames = []
    for cell in aligned:
        image = pixelize_tree.pixelize(settings, *size, source=cell.crop((x0, y0, x1, y1)), trim=False)
        root = root_point(image)
        placed = Image.new("RGBA", size)
        placed.paste(image, (target[0] - root[0], target[1] - root[1]))
        frames.append(placed)
    fix_wood(frames, settings, name)

    output = Image.new("RGBA", (COLS * size[0], ROWS * size[1]))
    for frame, image in enumerate(frames):
        output.alpha_composite(image, ((frame % COLS) * size[0], (frame // COLS) * size[1]))
    output.save(directory / "dot-sheet.png")
    print(name, size, "target root", target, "frame roots", [root_point(image) for image in frames])


def main():
    settings = json.load(open(ROOT / "assets_candidate/tiles/sources/trees.json"))["trees"]
    tiles = json.load(open(ROOT / "assets_candidate/tiles/tiles.json"))["sprites"]
    names = sys.argv[1:] or [name for name in settings if name != "stree_dead"]
    for name in names:
        if name not in settings or name == "stree_dead":
            raise ValueError(f"not a leafy scene tree: {name}")
        prepare(name, settings[name], tiles)


if __name__ == "__main__":
    main()
