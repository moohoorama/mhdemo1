#!/usr/bin/env python3
"""Render eight poses of the existing scene peach model on fixed, root-aligned cells."""

import math
import sys
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "tools/unit3d"))

import base_model as model  # noqa: E402
import scene_props3d as props  # noqa: E402
import structures  # noqa: E402

OUT = ROOT / "tools/spritetool/assets/scene-peach-sway/reference-3d.png"
CELL = 128
SCALE = 4
FRAMES = 8
FOLIAGE = {props.P.BLOSSOM, props.P.BLOSSOM_DARK, props.PEACH_LEAF}


def main():
    props.peach_tree(0)
    original = [(vertices.copy(), color) for vertices, color in model.meshes]
    sheet = Image.new("RGB", (4 * CELL * SCALE, 2 * CELL * SCALE), "#ff00ff")

    for frame in range(FRAMES):
        phase = 2 * math.pi * frame / FRAMES
        model.meshes = []
        for vertices, color in original:
            moved = vertices.copy()
            if color in FOLIAGE:
                for point in moved:
                    x, y, _ = point
                    outer = min(1.0, max(0.35, math.hypot(x, y) / 1.5))
                    offset = 1.1 * x - 0.8 * y
                    point[0] += outer * 0.10 * (math.sin(phase + offset) - math.sin(offset))
                    point[1] += outer * 0.035 * (math.cos(phase + offset) - math.cos(offset))
                    point[2] += 0.015 * (math.sin(phase + 1.8 * x) - math.sin(1.8 * x))
            model.meshes.append((moved, color))

        render = structures.draw(model.meshes, cell=(300, 340), pivot=(150, 290))
        cell = render.crop((86, 178, 86 + CELL, 178 + CELL)).resize(
            (CELL * SCALE, CELL * SCALE), Image.Resampling.NEAREST)
        background = Image.new("RGBA", cell.size, "#ff00ff")
        background.alpha_composite(cell)
        sheet.paste(background.convert("RGB"), ((frame % 4) * CELL * SCALE,
                                                (frame // 4) * CELL * SCALE))

    OUT.parent.mkdir(parents=True, exist_ok=True)
    sheet.save(OUT)
    print(OUT)


if __name__ == "__main__":
    main()
