#!/usr/bin/env python3
"""First pass of an effect's ASCII grids: python3 seed.py heal water ...

Writes the named sprites' frames into pixels.json from the shapes below; the grids are then
retouched by hand there. Running it again for a sprite throws its retouching away.
"""
import json
import math
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parents[3] / "ver5/tools"))
from pixelgrid import Grid, ks  # noqa: E402


def heal(s):
    w, h = s["cell"]
    cx, cy = s["anchor"]
    frames = []
    for k in ks(s["frames"]):
        pillar = Grid(w, h)
        tall = 40 * min(1, k * 3)
        for half, c in ((9, "a"), (5, "b"), (2, "c")):
            pillar.trapezoid(cx, cy, cy - tall, half, half * .6, c)
        ring, inner = Grid(w, h), Grid(w, h)
        rx = 10 + 6 * math.sin(k * math.pi)
        ring.ellipse(cx, cy, rx, 2.2, "r")
        inner.ellipse(cx, cy, rx * .65, 1.6, "w")
        frames.append({"pillar": pillar.lines(), "ring": ring.lines(), "inner": inner.lines()})
    return frames


def heal_mote(s):
    return [{"mote": ["..m..", "..m..", "mmwmm", "..m..", "..m.."]}]


def water(s):
    w, h = s["cell"]
    cx, cy = s["anchor"]
    frames = []
    for k in ks(s["frames"]):
        ripple, ripple2, column = Grid(w, h), Grid(w, h), Grid(w, h)
        ripple.ellipse(cx, cy, 4 + k * 22, 2.4, "r")
        ripple2.ellipse(cx, cy, 2 + k * 14, 1.8, "r")
        if k < .5:
            tall = math.sin(k / .5 * math.pi) * 26
            column.trapezoid(cx, cy, cy - tall, 5, 2, "b")
            column.trapezoid(cx, cy, cy - tall, 3.5, 1.3, "c")
            column.trapezoid(cx, cy - 1, cy - tall + 1, 1, .5, "w")
        frames.append({"ripple": ripple.lines(), "ripple2": ripple2.lines(), "column": column.lines()})
    return frames


def water_drop(s):
    return [{"drop": [".a.", "awa", ".a."]}, {"drop": [".b.", "bab", ".b."]}, {"drop": [".c.", "cbc", ".c."]}]


def fire(s):
    w, h = s["cell"]
    cx, cy = s["anchor"]
    frames = []
    for k in ks(s["frames"]):
        scorch = Grid(w, h)
        scorch.ellipse(cx, cy, 10 + 16 * k, 3, "d")
        scorch.ellipse(cx, cy, 10 + 16 * k, 1.6, "s")
        frames.append({"scorch": scorch.lines()})
    return frames


def ember(s):
    return [{"ember": [c * 2, c * 2]} for c in "abc"]


def buff(s):
    return heal(s)


def buff_arrow(s):
    return [{"arrow": ["..w..", ".www.", "wgwgw", "..w..", "..w..", "..g.."]}]


def buff_dot(s):
    return [{"dot": [".g.", "gwg", ".g."]}]


def debuff(s):
    w, h = s["cell"]
    cx, cy = s["anchor"]
    frames = []
    for k, t in zip(ks(s["frames"]), range(s["frames"])):
        ring, orbit = Grid(w, h), Grid(w, h)
        ring.ellipse(cx, cy, 14 - 6 * k, 1.6, "r")
        r = 12 - 4 * k
        oy = cy - 21  # chest height of an average unit, a little above
        for i in range(6):
            a = k * 9 * 1.1 + i * math.pi / 3
            x, y = cx + math.cos(a) * r, oy + math.sin(a) * r * .4
            for dx, dy in ((0, -1), (-1, 0), (1, 0), (0, 1)):
                orbit.set(x + dx, y + dy, "p")
            orbit.set(x, y, "w")
        frames.append({"ring": ring.lines(), "orbit": orbit.lines()})
    return frames


def debuff_arrow(s):
    return [{"arrow": ["..p..", "..p..", "..w..", "pwpwp", ".ppp.", "..p.."]}]


def slash(s):
    w, h = s["cell"]
    cx, cy = s["anchor"]
    frames = []
    for k in ks(s["frames"]):
        arc = Grid(w, h)
        if k < .45:  # two crossing crescents drawn on, the leading end brightest
            n = int(2 + k / .45 * 14)
            for i, sg in enumerate((1, -1)):
                pts = []
                for j in range(n + 1):
                    t = j / 16 * 2.2 - 1.1
                    pts.append((cx + sg * math.sin(t) * 15 * .9 + sg * i * 2, cy + math.cos(t) * 15 * .35 * sg - t * 15 * .55))
                for j, (a, b) in enumerate(zip(pts, pts[1:])):
                    arc.line(*a, *b, 2.4, "w" if j >= n - 3 else "y")
        frames.append({"arc": arc.lines()})
    return frames


def critical(s):
    w, h = s["cell"]
    cx, cy = s["anchor"]
    frames = []
    for k in ks(s["frames"]):
        rays = Grid(w, h)
        for i in range(8):
            a = i * math.pi / 4 + .3
            r0, r1 = 6 + k * 40, 16 + k * 70
            x0, y0 = cx + math.cos(a) * r0, cy + math.sin(a) * r0 * .7
            x1, y1 = cx + math.cos(a) * r1, cy + math.sin(a) * r1 * .7
            rays.line(x0, y0, x1, y1, 2.2, "w")
            rays.line(x0 + (x1 - x0) * .7, y0 + (y1 - y0) * .7, x1, y1, 2.2, "y")
        frames.append({"rays": rays.lines()})
    return frames


def critical_wave(s):
    w, h = s["cell"]
    cx, cy = s["anchor"]
    frames = []
    for k in ks(s["frames"]):
        outer, inner = Grid(w, h), Grid(w, h)
        outer.ellipse(cx, cy, 6 + k * 44, 3, "y")
        inner.ellipse(cx, cy, 3 + k * 28, 1.6, "w")
        frames.append({"outer": outer.lines(), "inner": inner.lines()})
    return frames


def block_pop(k):
    if k < .12:
        return .6 + k / .12 * .5
    if k < .22:
        return 1.1 - (k - .12) / .1 * .1
    return 1


def block(s):
    """The shield facing right (the game mirrors it for a blow from the left)."""
    w, h = s["cell"]
    ax, ay = s["anchor"]
    frames = []
    for k in ks(s["frames"]):
        pop = block_pop(k)
        cx, rx, ry = ax + 10, 7 * pop, 15 * pop
        pane, rim, hexes, r1, r2 = Grid(w, h), Grid(w, h), Grid(w, h), Grid(w, h), Grid(w, h)
        pane.oval(cx, ay, rx, ry, 0, "f", fill="f")
        rim.oval(cx, ay, rx, ry, 1.5, "r")
        hr = 3.2 * pop
        for dx, dy in ((0, 0), (0, -1.75), (0, 1.75), (-1.1, -.87), (-1.1, .87), (1.1, -.87), (1.1, .87)):
            hx, hy = cx + dx * hr * .62, ay + dy * hr
            pts = [(hx + hr * .55 * math.cos(a * math.pi / 3), hy + hr * .9 * math.sin(a * math.pi / 3)) for a in range(6)]
            for a, b in zip(pts, pts[1:] + pts[:1]):
                hexes.line(*a, *b, 1, "h")
        contact = cx + rx * .4
        for g, d in ((r1, 0), (r2, .12)):
            if 0 < k - d < .5:
                g.ellipse(contact, ay, (k - d) / .5 * 20, 1.4, "p")
        frames.append({"pane": pane.lines(), "hex": hexes.lines(), "rim": rim.lines(), "ripple": r1.lines(), "ripple2": r2.lines()})
    return frames


def arrow(s):
    """Sixteen headings, frame j flying at j * 22.5 degrees (0 east, clockwise on screen); the anchor is the tip."""
    w, h = s["cell"]
    cx, cy = s["anchor"]
    frames = []
    for j in range(s["frames"]):
        a = j * 2 * math.pi / 16
        nx, ny = math.cos(a), math.sin(a)
        trail, body = Grid(w, h), Grid(w, h)
        tx, ty = cx - nx * 9, cy - ny * 9
        for i, c in zip(range(1, 5), "1234"):  # faint dashes behind
            d = i * 5
            trail.line(tx - nx * d, ty - ny * d, tx - nx * (d - 4), ty - ny * (d - 4), 1, c)
        body.line(tx, ty, cx, cy, 1, "s")
        body.line(tx, ty, tx - nx * 3 - ny * 2, ty - ny * 3 + nx * 2, 1, "f")
        body.line(cx - nx, cy - ny, cx + nx * 2, cy + ny * 2, 1, "h")
        frames.append({"trail": trail.lines(), "body": body.lines()})
    return frames


def spark(s):
    return [{"spark": ["ww", "ww"]}]


def hit_slash(s):
    """The cut across a hit unit: a white stroke and a gold one just behind it, each drawn on over 0.24 s."""
    w, h = s["cell"]
    ax, ay = s["anchor"]
    n = s["frames"]
    frames = []
    for i in range(n):
        t = (i + .5) / n * .27
        g = Grid(w, h)
        for dx, thick, c, delay in ((-3, 1, "g", .03), (0, 3, "w", 0)):
            k = (t - delay) / .24
            if not 0 < k < 1:
                continue
            head, tail = min(1, k / .6), max(0, (k - .6) / .4)

            def seg(u):
                px = (1 - u) ** 2 * -15 + u * u * 16
                py = (1 - u) ** 2 * -17 + 2 * (1 - u) * u * -5 + u * u * 14
                return ax + px + dx, ay + py
            u = tail
            while u < head:
                g.line(*seg(u), *seg(min(head, u + .05)), thick, c)
                u += .05
        frames.append({"cut": g.lines()})
    return frames


def ink(s):
    """Ink drops by size, radius 1 to 3 in half pixels."""
    w, h = s["cell"]
    cx, cy = s["anchor"]
    frames = []
    for r in (1, 1.5, 2, 2.5, 3):
        g = Grid(w, h)
        g.disc(cx + .5, cy + .5, r, "k")
        frames.append({"drop": g.lines()})
    return frames


def crit_ring(s):
    """The gold ring a critical hit throws out, frame j at j/n of its eased spread."""
    w, h = s["cell"]
    cx, cy = s["anchor"]
    frames = []
    for k in ks(s["frames"]):
        g = Grid(w, h)
        r = 20 * (.2 + 2 * k)
        g.oval(cx + .5, cy + .5, r, r, 2, "y")
        frames.append({"ring": g.lines()})
    return frames


def click_ring(s):
    """The gold ring a click on the town's ground throws out round the tile the hero walks to: a flat iso ring,
    frame j at j/n of its eased spread, a bright rim over a deeper gold edge so it reads on sand and grass."""
    w, h = s["cell"]
    cx, cy = s["anchor"]
    frames = []
    for k in ks(s["frames"]):
        g = Grid(w, h)
        rx = 5 + 16 * (1 - (1 - k) ** 2)
        g.ellipse(cx, cy, rx + .9, 1.2, "o")
        g.ellipse(cx, cy, rx, 1.4, "y")
        frames.append({"ring": g.lines()})
    return frames


def heal_streak(s):
    """Rising heal streaks by length, 4 to 10 pixels, the top pixel brightest."""
    w, h = s["cell"]
    frames = []
    for n in (4, 6, 8, 10):
        frames.append({"streak": ["w" if y == 0 else "g" if y < n else "." for y in range(h)]})
    return frames


def petal(s):
    return [{"petal": [("w" * n).ljust(4, "."), ("w" * n).ljust(4, ".")]} for n in (2, 3, 4)]


def _clash(s, reach):
    """Twelve rays out from the clash and a ring closing in, frame j at impact 1 - (j + .5) / n."""
    w, h = s["cell"]
    cx, cy = s["anchor"]
    frames = []
    for k in ks(s["frames"]):
        impact = 1 - k
        g = Grid(w, h)
        for i in range(12):
            a = i * math.pi / 6
            g.line(cx + .5 + math.cos(a) * 8 / 3, cy + .5 + math.sin(a) * 8 / 3,
                   cx + .5 + math.cos(a) * reach / 3, cy + .5 + math.sin(a) * reach / 3, 2 if impact > .5 else 1, "y", square=True)
        r = (reach * (1 - impact) + 8) / 3
        g.oval(cx + .5, cy + .5, r, r, 1, "y")
        frames.append({"burst": g.lines()})
    return frames


def clash(s):
    return _clash(s, 36)


def clash_crit(s):
    return _clash(s, 100)


def speck(s):
    return [{"speck": ["ww"]}]


if __name__ == "__main__":
    settings = json.loads((HERE / "settings.json").read_text())
    path = HERE / "pixels.json"
    pixels = json.loads(path.read_text()) if path.exists() else {}
    for name in sys.argv[1:]:
        pixels[name] = globals()[name](settings["sprites"][name])
    path.write_text(json.dumps(pixels, indent=1) + "\n")
