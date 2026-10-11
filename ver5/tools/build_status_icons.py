#!/usr/bin/env python3
"""Compile hand-authored status pixels without resampling or palette changes."""
import json
from pathlib import Path

from PIL import Image, ImageColor

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT.parent / "tools/spritetool/assets/ver5-status"
settings = json.loads((SOURCE / "settings.json").read_text())
pixels = json.loads((SOURCE / "pixels.json").read_text())
frames = []
for name in settings["frame_order"]:
    directions = ("",) if name in pixels else ("up", "down")
    for direction in directions:
        frames.append((name + ("-" + direction if direction else ""), name, direction))
w, h = settings["cell"]
sheet = Image.new("RGBA", (w * len(frames), h))
index = {}
for i, (key, name, direction) in enumerate(frames):
    colors = settings["palette"][name]
    palette = dict(zip("mhr", colors))
    palette["o"] = settings["outline"]
    rows = pixels[direction or name]
    dx, dy = settings["arrow_offset"] if direction else (0, 0)
    for y, row in enumerate(rows):
        for x, pixel in enumerate(row):
            if pixel != ".":
                sheet.putpixel((i * w + dx + x, dy + y), ImageColor.getrgb(palette[pixel]) + (255,))
    index[key] = [i * w, 0, w, h]
output = ROOT / "assets/graphics/status"
output.mkdir(parents=True, exist_ok=True)
sheet.save(output / "icons.png")
(output / "icons.json").write_text(json.dumps(index, indent=2) + "\n")
