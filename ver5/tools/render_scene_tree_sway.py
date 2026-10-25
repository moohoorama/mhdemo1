#!/usr/bin/env python3
"""Render four foliage-only poses for every leafy story-scene tree."""

import json
import math
import sys
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "tools/unit3d"))

import base_model as model  # noqa: E402
import scene_props3d as props  # noqa: E402
import structures  # noqa: E402

OUT = ROOT / "tools/spritetool/assets/scene-tree-sway"
COLS = 2
FRAMES = 4
SCALE = 4
WOOD = {props.P.BARK, props.PINE_BARK, props.OLD_BARK}


def model_frame(original, phase, amplitude):
    model.meshes = []
    for vertices, color in original:
        moved = vertices.copy()
        if color not in WOOD:
            for point in moved:
                x, y, _ = point
                outer = min(1.0, max(0.35, math.hypot(x, y) / 1.5))
                offset = 1.1 * x - 0.8 * y
                point[0] += outer * amplitude * (math.sin(phase + offset) - math.sin(offset))
                point[1] += outer * amplitude * 0.35 * (math.cos(phase + offset) - math.cos(offset))
                point[2] += amplitude * 0.15 * (math.sin(phase + 1.8 * x) - math.sin(1.8 * x))
        model.meshes.append((moved, color))
    return structures.draw(model.meshes, cell=(300, 340), pivot=(150, 290))


def render_tree(name):
    props.PROPS[name]()
    original = [(vertices.copy(), color) for vertices, color in model.meshes]
    amplitude = 0.12 if "willow" in name else 0.09 if "bush" in name else 0.10
    renders = [model_frame(original, 2 * math.pi * frame / FRAMES, amplitude)
               for frame in range(FRAMES)]
    boxes = [im.getbbox() for im in renders]
    side = max(max(2 * (150 - b[0] + 12), 2 * (b[2] - 150 + 12), 290 - b[1] + 24)
               for b in boxes)
    side = math.ceil(side / 8) * 8
    x0, y0 = 150 - side // 2, 290 - (side - 12)
    sheet = Image.new("RGB", (COLS * side * SCALE, 2 * side * SCALE), "#ff00ff")
    for frame, render in enumerate(renders):
        cell = render.crop((x0, y0, x0 + side, y0 + side)).resize(
            (side * SCALE, side * SCALE), Image.Resampling.NEAREST)
        background = Image.new("RGBA", cell.size, "#ff00ff")
        background.alpha_composite(cell)
        sheet.paste(background.convert("RGB"), ((frame % COLS) * side * SCALE,
                                                (frame // COLS) * side * SCALE))
    directory = OUT / name
    directory.mkdir(parents=True, exist_ok=True)
    sheet.save(directory / "reference-3d.png")
    print(name, "reference", sheet.size, "cell", side)


def main():
    trees = json.load(open(ROOT / "assets_candidate/tiles/sources/trees.json"))["trees"]
    names = sys.argv[1:] or [name for name in trees if name != "stree_dead"]
    for name in names:
        if name not in trees or name == "stree_dead":
            raise ValueError(f"not a leafy scene tree: {name}")
        render_tree(name)


if __name__ == "__main__":
    main()
