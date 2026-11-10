#!/usr/bin/env python3
"""Make a map document (assets/scenario/<scenario>/maps/mapNN.json) from a text layout.

    python3 tools/mapgen/mapgen.py layout.txt assets/scenario/hero/maps/map01.json

The layout is plain text: one row per line, one cell per three characters, cells separated by
spaces: ground, structure, state (see Grounds, Structures and States in assets/data/game.json).
    ground     g grass, d wasteland, w water, c wall, m gate wall, i castle floor, h cliff, u rolling hill,
               n mountain
    structure  - none, t tree, v village, k camp, o castle, p gate, b bridge, z fence
    state      - none, e embers, F flames, s soot, d damaged bridge, x broken bridge
A structure stands only on the grounds its Structures entry lists (a gate on castle floor, a bridge
on water). `g-- gt- dv-` is grass, a tree on grass and a village on wasteland. Blank lines and lines
starting with # are ignored. All rows must have the same number of cells (the map's width).

A row without spaces is the older one-character form (. d ~ f v c i b g k h o); each character is
read as the cell the layouts of chapters 1-9 meant by it (LEGACY below).

Outside the play area the picture runs on with the edge cells (as the map editor's import does) — a camp, castle,
village or gate on the edge is not repeated there, only its ground — and
the document has no decorations or props. Decorations can be added later in the map editor.
"""
import json
import sys

LEGACY = {".": "g--", "d": "d--", "~": "w--", "f": "gt-", "v": "dv-", "k": "dk-", "c": "c--", "i": "i--",
          "b": "wb-", "g": "ip-", "o": "go-", "h": "h--"}
SINGLE = "kovp"  # camps, castles, villages and gates stand once; the edge copies outside keep only the ground


def make(rows, pad=4):
    h, w = len(rows), len(rows[0])
    assert all(len(r) == w for r in rows), "rows differ in length"
    assert 4 <= w <= 64 and 4 <= h <= 64, "a map is 4..64 cells each way"

    def cell(x, y):
        c = rows[max(0, min(h - 1, y))][max(0, min(w - 1, x))]
        if (0 <= x < w and 0 <= y < h) or c[1] not in SINGLE:
            return c
        return c[0] + "-" + c[2]

    cells = {}
    for y in range(-pad, h + pad):
        for x in range(-pad, w + pad):
            cells[f"{x},{y}"] = cell(x, y)
    return {
        "Decorations": [],
        "Props": [],
        "Version": 2,
        "Play": {"X": 0, "Y": 0, "W": w, "H": h},
        "Visual": {"X": -pad, "Y": -pad, "W": w + 2 * pad, "H": h + 2 * pad},
        "Cells": cells,
        "Default": rows[0][0],
    }


def read(path):
    rows = []
    for line in open(path, encoding="utf-8").read().splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        tokens = line.split() if " " in line.strip() else [LEGACY[c] for c in line.strip()]
        assert all(len(t) == 3 for t in tokens), f"a cell is three characters: {line}"
        rows.append(tokens)
    return rows


if __name__ == "__main__":
    if len(sys.argv) != 3:
        sys.exit(__doc__)
    doc = make(read(sys.argv[1]))
    with open(sys.argv[2], "w", encoding="utf-8") as f:
        f.write(json.dumps(doc, ensure_ascii=False, indent=1) + "\n")
