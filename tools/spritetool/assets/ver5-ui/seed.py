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


# The scrollbar options (10x10 UI px pieces, one art px per UI px, nearest sampled): the game slices each nine ways
# with 4 px corners and stretches the middle rows and columns, so rows 4-5 of every piece are the same row.
def _row(*parts):
    s = "".join(c * n for c, n in parts)
    assert len(s) == 10, s
    return s


def scroll_a_track(s):
    """A: a gold-framed lacquer groove, the window rim's double gold line."""
    edge, body = _row(("k", 1), ("g", 8), ("k", 1)), _row(("k", 1), ("d", 1), ("l", 6), ("g", 1), ("k", 1))
    return [{"track": [_row(("k", 10)), edge, body, body, body, body, body, body, edge, _row(("k", 10))]}]


def scroll_a_thumb(s):
    """A: a solid gold block with a light top edge and a dark foot, lit when hovered (the second frame)."""
    rows = [_row(("k", 10)), _row(("k", 1), ("w", 8), ("k", 1))]
    rows += [_row(("k", 1), ("g", 7), ("d", 1), ("k", 1))] * 5
    rows += [_row(("k", 1), ("d", 8), ("k", 1))] * 2 + [_row(("k", 10))]
    lit = [r.translate(str.maketrans("wgdk", "WGDk")) for r in rows]
    return [{"thumb": rows}, {"thumb": lit}]


def scroll_b_track(s):
    """B: a beaded groove, a dark capsule with a gold line down its middle."""
    r0 = _row((".", 3), ("k", 4), (".", 3))
    r1 = _row((".", 2), ("k", 1), ("l", 4), ("k", 1), (".", 2))
    r2 = _row((".", 1), ("k", 1), ("l", 6), ("k", 1), (".", 1))
    mid = _row((".", 1), ("k", 1), ("l", 2), ("g", 2), ("l", 2), ("k", 1), (".", 1))
    return [{"track": [r0, r1, r2, mid, mid, r2, r1, r1, r1, r0]}]


def scroll_b_thumb(s):
    """B: a round-ended bead, gold with a white glint and a dark foot, lit when hovered."""
    rows = [
        _row((".", 3), ("k", 4), (".", 3)),
        _row((".", 2), ("k", 1), ("w", 4), ("k", 1), (".", 2)),
        _row((".", 1), ("k", 1), ("w", 1), ("g", 4), ("k", 1), (".", 2)),
        _row((".", 1), ("k", 1), ("w", 1), ("g", 4), ("d", 1), ("k", 1), (".", 1)),
        _row(("k", 1), ("w", 1), ("g", 6), ("d", 1), ("k", 1)),
        _row(("k", 1), ("w", 1), ("g", 6), ("d", 1), ("k", 1)),
        _row(("k", 1), ("g", 6), ("d", 2), ("k", 1)),
        _row((".", 1), ("k", 1), ("g", 4), ("d", 2), ("k", 1), (".", 1)),
        _row((".", 2), ("k", 1), ("d", 4), ("k", 1), (".", 2)),
        _row((".", 3), ("k", 4), (".", 3)),
    ]
    lit = [r.translate(str.maketrans("wgdk", "WGDk")) for r in rows]
    return [{"thumb": rows}, {"thumb": lit}]


def scroll_c_track(s):
    """C: a wooden roller, dark wood with light grain and a dark edge."""
    v1 = _row(("k", 1), ("b", 8), ("k", 1))
    v2 = _row(("k", 1), ("b", 3), ("l", 1), ("b", 4), ("k", 1))
    v3 = _row(("k", 1), ("b", 1), ("l", 2), ("b", 5), ("k", 1))
    dark = _row(("k", 1), ("d", 8), ("k", 1))
    return [{"track": [_row(("k", 10)), v1, v2, v3, v2, v2, v3, v1, dark, _row(("k", 10))]}]


def scroll_c_thumb(s):
    """C: a paper slip with gold ends and a dark edge, lit when hovered."""
    paper = _row(("k", 1), ("p", 8), ("k", 1))
    slip = _row(("k", 1), ("p", 7), ("d", 1), ("k", 1))
    rows = [_row(("k", 10)), _row(("k", 1), ("g", 8), ("k", 1)), slip, slip, paper, paper, paper, slip,
            _row(("k", 1), ("d", 8), ("k", 1)), _row(("k", 10))]
    lit = [r.translate(str.maketrans("pgdk", "PGDk")) for r in rows]
    return [{"thumb": rows}, {"thumb": lit}]


def scroll_d_track(s):
    """D: dark lacquer with gold studs down the groove."""
    edge = _row(("k", 1), ("d", 8), ("k", 1))
    plain = _row(("k", 1), ("l", 8), ("k", 1))
    frame = _row(("k", 1), ("d", 1), ("l", 6), ("d", 1), ("k", 1))
    stud = _row(("k", 1), ("d", 1), ("l", 2), ("g", 2), ("l", 2), ("d", 1), ("k", 1))
    return [{"track": [_row(("k", 10)), edge, frame, stud, plain, plain, stud, frame, edge, _row(("k", 10))]}]


def scroll_d_thumb(s):
    """D: a dark lacquer block with a gold frame and a gold stud at its middle, lit when hovered."""
    gold = _row(("k", 1), ("g", 8), ("k", 1))
    frame = _row(("k", 1), ("g", 1), ("l", 6), ("g", 1), ("k", 1))
    mid = _row(("k", 1), ("g", 1), ("l", 2), ("g", 2), ("l", 2), ("g", 1), ("k", 1))
    rows = [_row(("k", 10)), gold, frame, frame, mid, mid, frame, frame, gold, _row(("k", 10))]
    lit = [r.translate(str.maketrans("lgk", "LGk")) for r in rows]
    return [{"thumb": rows}, {"thumb": lit}]


def scroll_e_track(s):
    """E: a dotted fabric, dark pixels with holes where the table shows through (a dither)."""
    odd = _row(("k", 1), ("a", 1), (".", 1), ("a", 1), (".", 1), ("a", 1), (".", 1), ("a", 1), (".", 1), ("k", 1))
    even = _row(("k", 1), (".", 1), ("a", 1), (".", 1), ("a", 1), (".", 1), ("a", 1), (".", 1), ("a", 1), ("k", 1))
    edge = _row(("k", 1), ("a", 8), ("k", 1))
    return [{"track": [_row(("k", 10)), edge, even, odd, odd, odd, even, odd, edge, _row(("k", 10))]}]


def scroll_e_thumb(s):
    """E: a gold checker, gold and dark gold alternating (a dither), lit when hovered."""
    a = _row(("k", 1), ("g", 1), ("d", 1), ("g", 1), ("d", 1), ("g", 1), ("d", 1), ("g", 1), ("d", 1), ("k", 1))
    b = _row(("k", 1), ("d", 1), ("g", 1), ("d", 1), ("g", 1), ("d", 1), ("g", 1), ("d", 1), ("g", 1), ("k", 1))
    edge = _row(("k", 1), ("d", 8), ("k", 1))
    rows = [_row(("k", 10)), _row(("k", 1), ("g", 8), ("k", 1)), a, b, a, a, b, a, edge, _row(("k", 10))]
    lit = [r.translate(str.maketrans("gdk", "GDk")) for r in rows]
    return [{"thumb": rows}, {"thumb": lit}]


# Shaded options: the same colours and outlines as the flat A-E pieces above (the same masks), with light from the
# upper left as in the window rim: the top and left bevels lit, the bottom and right in shadow, and colour steps down
# the body. Each piece is 10x10 and is nine-sliced, so rows 4-5 are the stretched middle row (identical), rows 0-3 and
# 6-9 are the fixed top and foot.
def _check(rows):
    assert len(rows) == 10 and all(len(r) == 10 for r in rows), rows
    assert rows[4] == rows[5], rows
    return rows


def _lit(rows, table):
    return [r.translate(str.maketrans(table)) for r in rows]


def scroll_a_shade_track(s):
    """A shaded: the gold-framed groove, lit top and left, the lacquer darkest under the top frame and lit at its foot."""
    return [{"track": _check([
        "k" * 10, "kGGGGGGGGk", "kGxxxxxxdk", "kGlllllldk", "kGlllllldk", "kGlllllldk", "kGlllllldk",
        "kGyyyyyydk", "kddddddddk", "k" * 10])}]


def scroll_a_shade_thumb(s):
    """A shaded: the gold block, a light top-left bevel, the colour stepping darker down to a dark foot; lit frame bright."""
    rows = _check([
        "k" * 10, "khhhhhhhwk", "khwwwwwwdk", "khwgggggdk", "khggggggdk", "khggggggdk", "khgmmmmgdk",
        "khmmmmmmdk", "k" + "d" * 8 + "k", "k" * 10])
    lit = _lit(rows, {"h": "H", "w": "W", "g": "G", "m": "M", "d": "D"})
    return [{"thumb": rows}, {"thumb": lit}]


def scroll_b_shade_track(s):
    """B shaded: the dark bead capsule, its top in shadow, the gold line down the middle, the lower lip lit."""
    return [{"track": _check([
        "...kkkk...", "..kxxxxk..", ".kxxllxxk.", ".kxlggllk.", ".kylggllk.", ".kylggllk.", "..kllllk..",
        "..kyyyyk..", "..kddddk..", "...kkkk..."])}]


def scroll_b_shade_thumb(s):
    """B shaded: the round gold bead with a white glint at the top-left, shading toward the lower right."""
    rows = _check([
        "...kkkk...", "..kwhhwk..", ".kwhgggk..", ".kwggggdk.", "kwggggmmdk", "kwggggmmdk",
        "kgggggmddk", ".kggggdek.", "..keddek..", "...kkkk..."])
    lit = _lit(rows, {"h": "H", "w": "W", "g": "G", "m": "M", "d": "D", "e": "E"})
    return [{"thumb": rows}, {"thumb": lit}]


def scroll_c_shade_track(s):
    """C shaded: the wooden roller, lit on its left and turning dark on the right, the grain kept as before."""
    rows = _check([
        "k" * 10, "khlbbbbdsk", "khbblbbdsk", "kbllbbbdsk", "khbblbbdsk", "khbblbbdsk", "kbllbbbdsk",
        "khlbbbbdsk", "k" + "d" * 8 + "k", "k" * 10])
    return [{"track": rows}]


def scroll_c_shade_thumb(s):
    """C shaded: the paper slip with a lit gold end and a shaded paper that darkens toward its right and foot."""
    rows = _check([
        "k" * 10, "k" + "g" * 8 + "k", "kpppppppdk", "kpppppppdk", "kpppoqqqdk", "kpppoqqqdk", "kppoqqqqdk",
        "kqqqqqqqdk", "k" + "d" * 8 + "k", "k" * 10])
    lit = _lit(rows, {"g": "G", "p": "P", "o": "O", "q": "Q", "d": "D"})
    return [{"thumb": rows}, {"thumb": lit}]


def scroll_d_shade_track(s):
    """D shaded: the lacquer groove with its gold studs, lit on the top and left, the foot in deep shadow."""
    return [{"track": _check([
        "k" * 10, "k" + "d" * 8 + "k", "kdLLLLLLdk", "kdlgGGglxk", "kdllllllxk", "kdllllllxk", "kdlggGGlxk",
        "kxllllllxk", "k" + "x" * 8 + "k", "k" * 10])}]


def scroll_d_shade_thumb(s):
    """D shaded: the lacquer block in a gold frame lit from the upper left, the centre lighter and the foot dark gold."""
    rows = _check([
        "k" * 10, "k" + "h" * 8 + "k", "khLLLLLLdk", "khlllllldk", "khllggllxk", "khllggllxk", "khlllllldk",
        "kdllllllxk", "k" + "d" * 8 + "k", "k" * 10])
    lit = _lit(rows, {"h": "H", "g": "G", "L": "T", "l": "M", "d": "D", "x": "X"})
    return [{"thumb": rows}, {"thumb": lit}]


def scroll_e_shade_track(s):
    """E shaded: the dotted fabric keeps its holes and dark dots; lit dots at the top and left, dark dots at the foot."""
    return [{"track": _check([
        "k" * 10, "k" + "b" * 8 + "k", "k.b.b.a.zk",
        "kb.b.a.a.k", "kb.a.a.z.k", "kb.a.a.z.k", "k.a.a.z.zk", "ka.z.z.z.k",
        "k" + "z" * 8 + "k", "k" * 10])}]


def scroll_e_shade_thumb(s):
    """E shaded: the gold checker with lit gold at the top, stepping to dark gold and a dark foot (the dither kept)."""
    rows = _check([
        "k" * 10, "k" + "h" * 8 + "k", "khghghghdk", "kghghghddk", "kgdgdgdgxk", "kgdgdgdgxk", "kdgdgdgdxk",
        "kdxdxdxxxk", "k" + "x" * 8 + "k", "k" * 10])
    lit = _lit(rows, {"h": "W", "g": "G", "d": "D", "x": "X"})
    return [{"thumb": rows}, {"thumb": lit}]


if __name__ == "__main__":
    settings = json.loads((HERE / "settings.json").read_text())
    path = HERE / "pixels.json"
    pixels = json.loads(path.read_text()) if path.exists() else {}
    for name in sys.argv[1:]:
        pixels[name] = globals()[name](settings["sprites"][name])
    path.write_text(json.dumps(pixels, indent=1) + "\n")
