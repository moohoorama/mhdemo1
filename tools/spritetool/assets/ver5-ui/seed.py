#!/usr/bin/env python3
"""First pass of the UI pictures' ASCII grids: python3 seed.py window_rim icon ...

Writes the named sprites' frames into pixels.json; the grids are then retouched by hand there.
UI pictures are drawn at four art pixels per UI pixel (A below), except the window rim (one, for
crisp edges) and the unit bar (game pixels).
"""
import json
import math
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parents[3] / "ver5/tools"))
from pixelgrid import Grid  # noqa: E402

A = 4  # art pixels per UI pixel


def window_rim(s):
    """The shared window frame on a 17x17 UI px window at 1 art px per UI px, corners 8 (the game slices it 9 ways):
    a black edge, a gold double rim and gold corner studs with lacquer centres."""
    w, h = s["cell"]
    W = w  # one art px per UI px: hard edges are kept crisp by nearest sampling
    g = Grid(w, h)

    def box(x, y, bw, bh, c):  # filled, in UI px
        for py in range(y, y + bh):
            for px in range(x, x + bw):
                g.set(px, py, c)

    def frame(x, y, bw, bh, t, c):  # a t-wide outline
        box(x, y, bw, t, c)
        box(x, y + bh - t, bw, t, c)
        box(x, y, t, bh, c)
        box(x + bw - t, y, t, bh, c)

    frame(0, 0, W, W, 1, "k")
    frame(1, 1, W - 2, W - 2, 2, "g")
    frame(6, 6, W - 12, W - 12, 1, "d")
    for x, y in ((2, 2), (W - 8, 2), (2, W - 8), (W - 8, W - 8)):
        box(x, y, 6, 6, "g")
        box(x + 2, y + 2, 2, 2, "l")
    return [{"rim": g.lines()}]


def _triangle(s, half_w, half_h):
    w, h = s["cell"]
    cx, cy = s["anchor"]
    frames = []
    for up in (True, False):
        g = Grid(w, h)
        d = 1 if up else -1
        g.polygon([(cx - half_w * A, cy + d * half_h * A), (cx + half_w * A, cy + d * half_h * A), (cx, cy - d * half_h * A)], "w")
        frames.append({"mark": g.lines()})
    return frames


def sort_mark(s):
    """The list's sort direction (and the HUD's fold mark): up, down."""
    return _triangle(s, 4, 3)


def scroll_mark(s):
    """A scroll button's mark: up, down."""
    return _triangle(s, 7, 4)


# the command menu's icons on a 12x12 grid, as strokes (lines, and quadratic curves as (ctrl, end))
ICONS = {
    "attack": [[(2, 10), (9, 3)], [(8, 2), (10, 4)], [(3, 7), (5, 9)]],  # sword
    "skill": [[(2, 2), (10, 2), (10, 10), (2, 10), (2, 2)], [(4, 5), (8, 5)], [(4, 7), (7, 7)]],  # book
    "item": [[(3, 5), ((6, 2), (9, 5)), (9, 10), (3, 10), (3, 5)], [(5, 3), (7, 3)]],  # pouch
    "wait": [[(3, 2), (9, 2)], [(3, 10), (9, 10)], [(4, 2), ((4, 6), (6, 6)), ((8, 6), (8, 2))],
             [(4, 10), ((4, 6), (6, 6)), ((8, 6), (8, 10))]],  # hourglass
}
ICON_ORDER = ["attack", "skill", "item", "wait"]


def icon(s):
    w, h = s["cell"]
    ox, oy = s["anchor"]  # the 12x12 grid's corner, inside the clear margin
    frames = []
    for name in ICON_ORDER:
        g = Grid(w, h)
        for stroke in ICONS[name]:
            pts = [stroke[0]]
            for p in stroke[1:]:
                if isinstance(p[0], tuple):  # quadratic
                    (cx, cy), (ex, ey) = p
                    sx, sy = pts[-1]
                    for j in range(1, 9):
                        t = j / 8
                        pts.append(((1 - t) ** 2 * sx + 2 * (1 - t) * t * cx + t * t * ex, (1 - t) ** 2 * sy + 2 * (1 - t) * t * cy + t * t * ey))
                else:
                    pts.append(p)
            for (ax, ay), (bx, by) in zip(pts, pts[1:]):
                g.line(ox + ax * A, oy + ay * A, ox + bx * A, oy + by * A, 1.4 * A, "w")
        frames.append({"icon": g.lines()})
    return frames


def bar_frame(s):
    """The unit bar's case at game pixels: black edge, dark body, a lit top row, the HP trough and the MP track."""
    w, h = s["cell"]
    rows = []
    for y in range(h):
        row = ""
        for x in range(w):
            if x in (0, w - 1) or y in (0, h - 1):
                row += "k"
            elif y == 1:
                row += "h"
            elif 2 <= y <= 4 and 2 <= x <= w - 3:
                row += "t"
            elif y == 6 and 2 <= x <= w - 3:
                row += "m"
            else:
                row += "b"
        rows.append(row)
    return [{"case": rows}]


def bar_fill(s):
    """HP fill, enemy then ally: the lit top half, its edge on the middle row, the darker bottom."""
    w, h = s["cell"]
    return [{"fill": ["h" * w, "m" * w, "l" * w]}, {"fill": ["H" * w, "M" * w, "L" * w]}]


def bar_trail(s):
    w, h = s["cell"]
    return [{"trail": ["w" * w] * h}]


def bar_ticks(s):
    """Dark ticks splitting the HP into quarters."""
    w, h = s["cell"]
    cols = {round(w * i / 4 - .5) for i in (1, 2, 3)}
    return [{"ticks": ["".join("k" if x in cols else "." for x in range(w))] * h}]


def bar_mp(s):
    w, h = s["cell"]
    return [{"mp": ["m" * w] * h}]


def scroll_grain(s):
    return [{"grain": ["g"] * 4}]


if __name__ == "__main__":
    settings = json.loads((HERE / "settings.json").read_text())
    path = HERE / "pixels.json"
    pixels = json.loads(path.read_text()) if path.exists() else {}
    for name in sys.argv[1:]:
        pixels[name] = globals()[name](settings["sprites"][name])
    path.write_text(json.dumps(pixels, indent=1) + "\n")
