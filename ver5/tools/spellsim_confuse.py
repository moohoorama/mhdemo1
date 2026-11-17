#!/usr/bin/env python3
"""혼란 / 대혼란 on spellsim's world: four question marks spin down from about two cells above the target and
settle into a ring round its head; and the status marks the game turns round the head of a unit under a status.

    uv run --with numpy --with scipy --with pillow python tools/spellsim_confuse.py [names]   # reports/spellsim/

Each mark is a small dot glyph with a dark 1px outline that faces the screen. While falling a question mark turns
about its upright axis (drawn squeezed and mirrored); once on the ring it stands upright, as every status mark
does. A mark is hidden where the target's sprite stands while it is behind the target's centre (x + y < 0).

The status marks (MARKS) bake as one still sprite each, status_mark_<status>, anchored at the glyph's centre; the
game places them on the ring itself (status_visual.go, drawStatusMarks) so marks of several statuses share it.
"""
import argparse
import math
from pathlib import Path

import numpy as np

import spellsim as S
from spellsim import FPS, SIZE, screen
from spellsim_seal import MASK

EDGE = (36, 37, 43)  # the status icons' outline
RED = {"r": (236, 40, 34)}
# glyphs without their outline, one letter a colour; "." is empty. X marks the item barred: a red cross over it.
MARKS = {
    "confusion": ({"y": (255, 222, 74), "w": (255, 252, 214)},
                  [".wwyyy.",
                   "wy...yy",
                   ".....yy",
                   "....yy.",
                   "...yy..",
                   "...yy..",
                   ".......",
                   "...yy..",
                   "...yy.."], False),
    "disarm": ({"w": (220, 226, 223), "l": (147, 158, 171), "g": (211, 175, 107), "h": (120, 74, 44)},
               [".......ww",
                "......wwl",
                ".....wwl.",
                "....wwl..",
                "...wwl...",
                "g.wwl....",
                ".gwl.....",
                ".hg......",
                "h..g....."], True),
    "seal": ({"y": (246, 223, 155), "o": (196, 150, 84), "h": (171, 82, 86)},
             ["...yyyyy...",
              ".yyoyoyoyy.",
              "yoyoyoyoyoy",
              "yyoyoyoyoyy",
              ".yyoyoyoyy.",
              "...yoyoy...",
              "....yoy....",
              ".....h.....",
              ".....h....."], True),
    "root": ({"b": (150, 100, 60), "L": (206, 160, 98), "s": (84, 56, 38)},
             ["bbbb.....",
              "bLLb.....",
              "bLLb.....",
              "bLLb.....",
              "bLLbbb...",
              "bLLLLLbb.",
              "bLLLLLLLb",
              "bLLLLLLLb",
              "sssssssss"], True),
    "poison": ({"o": (125, 168, 79), "w": (211, 231, 144)},  # the poison status icon's drop, dot for dot
               ["...o...",
                "..wo...",
                "..woo..",
                ".ooooo.",
                ".ooooo.",
                "..ooo.."], False),
}

TOP = 28.0          # the test unit's height above its feet (light_infantry, as spellsim.compose draws it)
HEAD = TOP          # the ring turns at the top of the unit: its front half crosses the face, its back half over the head
DROP = 64.0         # the marks start this far above the ring: about two cells
R = 13.0            # the ring's radius in world units (18 px out each side on screen); statusMarkRadius in the game
TURN = 1.2          # seconds a mark takes round the ring; statusMarkTurn in the game
N = 4


def mark_image(kind):
    """The mark as an RGBA array with its 1px outline, the red cross laid over the item when it is barred."""
    pal, rows, barred = MARKS[kind]
    h, w = len(rows) + 2, len(rows[0]) + 2
    img = np.zeros((h, w, 4), np.uint8)
    for r, row in enumerate(rows):
        for c, ch in enumerate(row):
            if ch != ".":
                img[r + 1, c + 1] = (*pal[ch], 255)
    if barred:  # a thin cross, inset a dot, so the item still reads through it
        n = min(h, w) - 4
        y0, x0 = (h - n) // 2, (w - n) // 2
        for k in range(n):
            for x, y in ((x0 + k, y0 + k), (x0 + n - 1 - k, y0 + k)):
                img[y, x] = (*RED["r"], 255)
    a = img[:, :, 3] > 0
    img[S.ring(a) & ~a] = (*EDGE, 255)
    return img


def glyph_pixels(squeeze, mirror, kind="confusion"):
    """The mark's (dx, dy, colour) about its centre, squeezed to |squeeze| of its width, mirrored when turned away."""
    g = mark_image(kind)
    h, w = g.shape[:2]
    cols = max(3, round(w * max(.35, abs(squeeze))))
    out = []
    for c in range(cols):
        src = min(w - 1, int((c + .5) / cols * w))
        if mirror:
            src = w - 1 - src
        for r in range(h):
            if g[r, src, 3]:
                out.append((c - cols // 2, r - h // 2, tuple(int(v) for v in g[r, src, :3])))
    return out


def draw_mark(img, p, spin, alpha=255):
    """One mark at world point p, turned spin radians about its upright axis."""
    x, y = screen(np.array([p], float))
    x, y = int(round(x[0])), int(round(y[0]))
    behind = p[0] + p[1] < 0
    lay = np.zeros((SIZE[1], SIZE[0], 4), np.uint8)
    for dx, dy, col in glyph_pixels(math.cos(spin), math.cos(spin) < 0):
        if 0 <= x + dx < SIZE[0] and 0 <= y + dy < SIZE[1]:
            lay[y + dy, x + dx] = (*col, 255)
    m = lay[:, :, 3] > 0
    if behind:
        m &= ~MASK
    lay[m, 3] = alpha
    img[m] = lay[m]


class Marks:
    """Marks over the cells at, each following path(i, t) -> (point, spin, alpha) or None."""

    def __init__(self, cells, path, secs, hit):
        self.cells, self.path, self.secs, self.hit = cells, path, secs, hit
        self.t = 0.0

    def fade(self, k):
        pass

    def step(self, dt):
        self.t += dt

    def render(self):
        img = np.zeros((SIZE[1], SIZE[0], 4), np.uint8)
        got = []
        for c in self.cells:
            for i in range(N):
                q = self.path(i, self.t)
                if q is not None:
                    p, spin, a = q
                    got.append(((p[0] + c[0], p[1] + c[1], p[2]), spin, a))
        for p, spin, a in sorted(got, key=lambda g: g[0][0] + g[0][1]):  # far first
            draw_mark(img, p, spin, a)
        return img


def ring_point(i, t):
    """Where mark i is on the settled ring at time t, standing upright (spin 0): one TURN round."""
    a = 2 * math.pi * (t / TURN + i / N)
    return (math.cos(a) * R, math.sin(a) * R, HEAD), 0.0


def descent(settle=.75, stagger=.06):
    """Marks spiral down from DROP above the ring, wide and fast, narrowing onto it by settle; then they keep turning
    round it. Each starts stagger seconds after the one before."""
    def path(i, t):
        t0 = i * stagger
        if t < t0:
            return None
        u = min(1.0, (t - t0) / (settle - t0))
        e = 1 - (1 - u) ** 2                   # eases onto the ring
        (x, y, h), spin = ring_point(i, t)
        extra = (1 - e) * 2.5 * math.pi       # a faster wind while falling
        a = math.atan2(y, x) + extra
        r = R * (1 + .7 * (1 - e))
        p = (math.cos(a) * r, math.sin(a) * r, h + DROP * (1 - e))
        a_in = min(1.0, (t - t0) / .12)
        return p, spin + extra * 1.5, int(255 * a_in)
    return path


def honran_q():
    """혼란: four question marks spin down onto the target's head and turn round it."""
    return Marks([(0, 0)], descent(), 1.5, .75), "single"


def daehonran_q():
    """대혼란: the same on each of the five cells, the centre first and the arms a moment later."""
    lag = {c: .1 * (c != (0, 0)) for c in S.CROSS}
    base = descent()
    m = Marks(S.CROSS, None, 1.6, .85)

    def render():
        img = np.zeros((SIZE[1], SIZE[0], 4), np.uint8)
        got = []
        for c in S.CROSS:
            for i in range(N):
                q = base(i, m.t - lag[c]) if m.t >= lag[c] else None
                if q is not None:
                    p, spin, a = q
                    got.append(((p[0] + c[0], p[1] + c[1], p[2]), spin, a))
        for p, spin, a in sorted(got, key=lambda g: g[0][0] + g[0][1]):
            draw_mark(img, p, spin, a)
        return img
    m.render = render
    return m, "cross"


def status_marks():
    """{sprite: frames} for the status marks the game turns round a unit's head: one still frame each, its centre
    at the canvas origin (the bake's anchor)."""
    out = {}
    ox, oy = S.ORIGIN
    for kind in MARKS:
        g = mark_image(kind)
        h, w = g.shape[:2]
        frame = np.zeros((SIZE[1], SIZE[0], 4), np.uint8)
        frame[oy - h // 2:oy - h // 2 + h, ox - w // 2:ox - w // 2 + w] = g
        out["status_mark_" + kind] = [frame]
    return out


SPELLS = {f.__name__: f for f in (honran_q, daehonran_q)}


def run(name):
    sim, layout = SPELLS[name]()
    frames = []
    n = int(sim.secs * FPS)
    for _ in range(n):
        sim.step(1 / FPS)
        frames.append(sim.render())
    run.hit = sim.hit
    return frames, layout


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("names", nargs="*")
    ap.add_argument("--out", default=str(S.ROOT / "reports/spellsim"))
    args = ap.parse_args()
    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    from PIL import Image
    sheet = np.concatenate([np.pad(mark_image(k), ((0, 13 - mark_image(k).shape[0]), (1, 1), (0, 0)))
                            for k in MARKS], axis=1)
    Image.fromarray(sheet, "RGBA").resize((sheet.shape[1] * 8, sheet.shape[0] * 8), Image.NEAREST).save(out / "status_marks.png")
    print(out / "status_marks.png")
    for name in args.names or SPELLS:
        frames, layout = run(name)
        S.compose(name, frames, layout, out)
        print(out / f"{name}.gif")


if __name__ == "__main__":
    main()
