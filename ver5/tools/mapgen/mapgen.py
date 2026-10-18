#!/usr/bin/env python3
"""Make a map document (assets/scenario/<scenario>/maps/mapNN.json) from a text layout.

    python3 tools/mapgen/mapgen.py layout.txt assets/scenario/hero/maps/map01.json

The layout is plain text: one row per line, one game tile glyph per cell (see `Tiles` in
assets/data/game.json: . grass, d wasteland, ~ water, f forest, v village, c wall, i castle floor,
b bridge, g gate, k camp, h cliff, o castle, F burning ground, W wall of flames). Blank lines and
lines starting with # are ignored. All rows must have the same length (the map's width).

Outside the play area the picture runs on with the edge cells (as the map editor's import does), and
the document has no decorations or props. Decorations can be added later in the map editor.
"""
import json
import sys


def make(rows, pad=4):
    h, w = len(rows), len(rows[0])
    assert all(len(r) == w for r in rows), "rows differ in length"
    assert 4 <= w <= 64 and 4 <= h <= 64, "a map is 4..64 cells each way"

    def tile(x, y):
        return rows[max(0, min(h - 1, y))][max(0, min(w - 1, x))]

    cells, resolved = {}, {}
    for y in range(-pad, h + pad):
        for x in range(-pad, w + pad):
            t = tile(x, y)
            resolved[f"{x},{y}"] = t
            for j in range(2):
                for i in range(2):
                    cells[f"{2 * x + i},{2 * y + j}"] = t
    return {
        "Decorations": [],
        "Props": [],
        "Version": 1,
        "Play": {"X": 0, "Y": 0, "W": w, "H": h},
        "Visual": {"X": -pad, "Y": -pad, "W": w + 2 * pad, "H": h + 2 * pad},
        "Cells": cells,
        "Resolved": resolved,
        "Default": rows[0][0],
    }


def read(path):
    rows = []
    for line in open(path, encoding="utf-8").read().splitlines():
        if line.strip() and not line.lstrip().startswith("#"):
            rows.append(line.rstrip("\n"))
    return rows


if __name__ == "__main__":
    if len(sys.argv) != 3:
        sys.exit(__doc__)
    doc = make(read(sys.argv[1]))
    with open(sys.argv[2], "w", encoding="utf-8") as f:
        f.write(json.dumps(doc, ensure_ascii=False, indent=1) + "\n")
