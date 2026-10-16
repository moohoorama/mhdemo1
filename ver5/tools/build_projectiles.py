#!/usr/bin/env python3
"""Compile hand-authored projectile pixels (tools/spritetool/assets/ver5-projectiles) without resampling."""
import json
from pathlib import Path

from PIL import Image, ImageColor

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT.parent / "tools/spritetool/assets/ver5-projectiles"
settings = json.loads((SOURCE / "settings.json").read_text())
pixels = json.loads((SOURCE / "pixels.json").read_text())
rows = []  # one sheet row per sprite
for name in settings["frame_order"]:
    s = settings["sprites"][name]
    rows.append((name, s, pixels[name]))
width = max(s["cell"][0] * len(frames) for _, s, frames in rows)
height = sum(s["cell"][1] for _, s, _ in rows)
sheet = Image.new("RGBA", (width, height))
index = {}
top = 0
for name, s, frames in rows:
    w, h = s["cell"]
    palette = {"o": settings["outline"], **s["palette"]}
    rects = []
    for i, grid in enumerate(frames):
        assert len(grid) == h and all(len(r) == w for r in grid), f"{name} frame {i}: not {w}x{h}"
        for y, row in enumerate(grid):
            for x, pixel in enumerate(row):
                if pixel != ".":
                    sheet.putpixel((i * w + x, top + y), ImageColor.getrgb(palette[pixel]) + (255,))
        rects.append([i * w, top, w, h])
    index[name] = {"anchor": s["anchor"], "frames": rects}
    if "ms" in s:
        assert len(s["ms"]) == len(frames), f"{name}: ms and frames differ"
        index[name]["ms"] = s["ms"]
    top += h
output = ROOT / "assets/graphics/effects"
output.mkdir(parents=True, exist_ok=True)
sheet.save(output / "projectiles.png")
(output / "projectiles.json").write_text(json.dumps(index, indent=1) + "\n")
