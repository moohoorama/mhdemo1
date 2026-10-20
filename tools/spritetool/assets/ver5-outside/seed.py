#!/usr/bin/env python3
"""First pass of the outside scenery's ASCII grids: python3 seed.py pine stone

Writes the named sprites' frames into pixels.json; the grids are then retouched by hand there.
"""
import json
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parents[3] / "ver5/tools"))
from pixelgrid import Grid  # noqa: E402


def pine(s):
    """A shadow disc, the dark cone and its lit left half, at canvas scale; frame = size."""
    w, h = s["cell"]
    ax, ay = s["anchor"]
    frames = []
    for size in (7, 8, 9, 10, 11):
        shadow, tree = Grid(w, h), Grid(w, h)
        shadow.disc(ax + size * .4, ay + size * .1, size * .7, "s")
        tree.polygon([(ax, ay - size * 2), (ax + size * .7, ay), (ax - size * .7, ay)], "d")
        tree.polygon([(ax, ay - size * 2), (ax + size * .15, ay - size * .2), (ax - size * .7, ay)], "l")
        frames.append({"shadow": shadow.lines(), "tree": tree.lines()})
    return frames


def stone(s):
    return [{"stone": ["lld", "ddd"]}]


if __name__ == "__main__":
    settings = json.loads((HERE / "settings.json").read_text())
    path = HERE / "pixels.json"
    pixels = json.loads(path.read_text()) if path.exists() else {}
    for name in sys.argv[1:]:
        pixels[name] = globals()[name](settings["sprites"][name])
    path.write_text(json.dumps(pixels, indent=1) + "\n")
