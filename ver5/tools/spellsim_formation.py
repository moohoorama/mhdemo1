#!/usr/bin/env python3
"""팔진도 (eightfold, a 3x3 square, mixed) drafts on top of spellsim: a formation drawn into the ground over the
nine cells, and per cell an ally piece (gold, rising) or an enemy piece (dark purple, sinking), because the game
decides cell by cell who stands where.

    uv run --with numpy --with scipy --with pillow python tools/spellsim_formation.py [palzin_a ...]

Each draft writes to reports/spellsim/: <name>.gif (all together: ground plus allies on three cells and enemies on
three), <name>_ground.gif, <name>_ally.gif and <name>_enemy.gif (the separable pieces), and .png contact sheets.
The ground lines are dots on the iso ground plane (world lines step one pixel down per two across on screen).
"""
import argparse
import math
from pathlib import Path

import numpy as np
from PIL import Image, ImageDraw

import spellsim as S
from spellsim import FPS, ORIGIN, ROOT, SIZE

SQUARE = [(32 * i, 32 * j) for i in (-1, 0, 1) for j in (-1, 0, 1)]
S.LAYOUTS["square"] = (SQUARE, (-96, 0))
ALLY_CELLS = [(-32, 32), (-32, 0), (0, 32)]     # the screen-left three
ENEMY_CELLS = [(32, -32), (32, 0), (0, -32)]    # the screen-right three
SECS = 1.6

GOLD = [(255, 248, 214), (252, 214, 110), (220, 156, 52), (138, 86, 26)]
PURPLE = [(232, 196, 255), (168, 98, 222), (104, 46, 150), (50, 18, 78)]
LINE_DARK = (70, 42, 14)


def to_screen(x, y, h=0.0, origin=ORIGIN):
    return x - y + origin[0], (x + y) / 2 - h + origin[1]


def put(img, x, y, col, a=255):
    h, w = img.shape[:2]
    x, y = int(round(x)), int(round(y))
    if 0 <= x < w and 0 <= y < h:
        if a >= 255 or img[y, x, 3] == 0:
            img[y, x] = (*col, a)
        else:
            k = a / 255
            img[y, x, :3] = (img[y, x, :3] * (1 - k) + np.array(col) * k).astype(np.uint8)
            img[y, x, 3] = max(img[y, x, 3], a)


# --- the ground formation: polylines in world (x, y), each point with a reveal value; points revealed by time t
#     are drawn, the newest ones white-hot

def poly(pts, step=.5):
    """Points every `step` world units along a polyline."""
    out = []
    for (ax, ay), (bx, by) in zip(pts, pts[1:]):
        n = max(1, int(math.hypot(bx - ax, by - ay) / step))
        out += [(ax + (bx - ax) * i / n, ay + (by - ay) * i / n) for i in range(n)]
    out.append(pts[-1])
    return out


def ngon(r, n=8, rot=0.0):
    return [(r * math.cos(rot + 2 * math.pi * k / n), r * math.sin(rot + 2 * math.pi * k / n)) for k in range(n + 1)]


def trigram(k, r_in, r_out, ang, half=6.0, rot=0.0):
    """Three bars across the radius at angle ang: bit set = a solid bar, clear = a broken one."""
    out = []
    a = ang + rot
    ux, uy = math.cos(a), math.sin(a)
    tx, ty = -uy, ux
    for b in range(3):
        r = r_in + (r_out - r_in) * (b + .5) / 3
        cx, cy = ux * r, uy * r
        if k >> b & 1:
            out.append([(cx - tx * half, cy - ty * half), (cx + tx * half, cy + ty * half)])
        else:
            out.append([(cx - tx * half, cy - ty * half), (cx - tx * half * .25, cy - ty * half * .25)])
            out.append([(cx + tx * half * .25, cy + ty * half * .25), (cx + tx * half, cy + ty * half)])
    return out


def draw_ground(img, strokes, reveal_at, fade, head=4.0, glow=True):
    """strokes: list of point lists in world; reveal_at: the reveal value reached now (points with value <= it
    show; value = the point's own reveal). fade 0..1 multiplies the whole layer's opacity."""
    lit, hot = {}, set()
    for pts, vals in strokes:
        for (x, y), v in zip(pts, vals):
            if v <= reveal_at:
                sx, sy = to_screen(x, y)
                key = (int(round(sx)), int(round(sy)))
                lit[key] = True
                if reveal_at - v < head:
                    hot.add(key)
    if not lit:
        return
    a = int(255 * fade)
    if glow:  # a faint warm halo round the lines and a dark seam under them, so they read on grass
        for (x, y) in lit:
            for dx, dy in ((0, -1), (-1, 0), (2, 0), (0, 1), (1, 1)):
                if (x + dx, y + dy) not in lit:
                    put(img, x + dx, y + dy, GOLD[2], int(70 * fade))
        for (x, y) in lit:
            if (x, y + 1) not in lit:
                put(img, x, y + 1, LINE_DARK, int(150 * fade))
    for (x, y) in lit:
        col = GOLD[0] if (x, y) in hot else GOLD[1]
        put(img, x, y, col, a)
        put(img, x + 1, y, col if (x, y) in hot else GOLD[2], a)


def strokes_bagua(rot=0.0):
    """Draft A's figure: an outer and inner octagon, eight spokes, the eight trigrams in the band between, a ring
    at the centre. Reveal = distance from the centre plus a little sweep round, so it spreads outward."""
    out = []

    def add(pts):
        pts = poly(pts)
        vals = [math.hypot(x, y) + (math.atan2(y, x) % (2 * math.pi)) * 1.2 for x, y in pts]
        out.append((pts, vals))
    add(ngon(9, 16))
    add(ngon(25, 8, math.pi / 8 + rot))
    add(ngon(54, 8, math.pi / 8 + rot))
    for k in range(8):
        a = k * math.pi / 4 + rot
        add([(math.cos(a) * 9, math.sin(a) * 9), (math.cos(a) * 54, math.sin(a) * 54)])
        for bar in trigram(k, 30, 49, a + math.pi / 8, 6.5):
            add(bar)
    return out


def strokes_grid():
    """Draft B's figure: the nine cells' edges and two diagonals through the square, with a small diamond in each
    cell. Reveal = the city-block distance from the centre, so it spreads as a diamond wave along the lines."""
    out = []

    def add(pts):
        pts = poly(pts)
        out.append((pts, [abs(x) + abs(y) for x, y in pts]))
    for c in (-48, -16, 16, 48):
        add([(c, -48), (c, 0), (c, 48)])
        add([(-48, c), (0, c), (48, c)])
    add([(-48, -48), (0, 0), (48, 48)])
    add([(-48, 48), (0, 0), (48, -48)])
    for cx, cy in SQUARE:
        add([(cx - 5, cy), (cx + 5, cy)])
        add([(cx, cy - 5), (cx, cy + 5)])
    return out


def strokes_rings(t):
    """Draft C's figure at time t: two dashed rings turning opposite ways with the trigrams riding the outer one,
    an eight-point star turning at the centre; it opens out from nothing."""
    grow = min(1.0, t / .35) ** .6
    out = []

    def add(pts):
        pts = poly(pts)
        out.append((pts, [0.0] * len(pts)))
    for r, spin, dashes in ((46, .9, 24), (30, -1.4, 16)):
        r *= grow
        for d in range(dashes):
            a0 = spin * t + 2 * math.pi * d / dashes
            a1 = a0 + 2 * math.pi / dashes * .62
            add([(r * math.cos(a0 + (a1 - a0) * i / 6), r * math.sin(a0 + (a1 - a0) * i / 6)) for i in range(7)])
    for k in range(8):
        a = .9 * t + k * math.pi / 4 + math.pi / 8
        for bar in trigram(k, 36 * grow, 44 * grow, a, 4.5 * grow):
            add(bar)
    star = 14 * grow
    for rot in (-1.6 * t, -1.6 * t + math.pi / 4):
        sq = [(star * math.cos(rot + k * math.pi / 2), star * math.sin(rot + k * math.pi / 2)) for k in range(5)]
        add(sq)
    return out


def ground_frames(style):
    n = int(SECS * FPS)
    frames = []
    for f in range(n):
        t = f / FPS
        img = np.zeros((SIZE[1], SIZE[0], 4), np.uint8)
        fade = 1.0 if t < 1.05 else max(0.0, 1 - (t - 1.05) / .5)
        if style == "a":
            draw_ground(img, strokes_bagua(), 85 * min(1, t / .45) ** .8, fade, 5)
        elif style == "b":
            reach = 100 * min(1, t / .5) ** .8
            draw_ground(img, strokes_grid(), reach, fade, 6)
            if t > .5:  # pulses run back out along the finished lines
                w = (t - .5) * 160 % 110
                draw_ground(img, [(p, [0] * len(p)) for p, v in strokes_grid() for p in
                                  [[q for q, vv in zip(p, v) if abs(vv - w) < 4]] if p], 1, fade * .9, 99, glow=False)
        else:
            draw_ground(img, strokes_rings(t), 1, fade, 99 if t < .3 else 0)
        frames.append(img)
    return frames


# --- per-cell pieces: motes with analytic paths, drawn back to front, plus rings and columns on the ground

def motes(rng, n, t0, rise, spiral=0.0, r0=(3, 12), life=(.5, .9), h0=(0, 6), big=.3):
    a = rng.uniform(0, 2 * math.pi, n)
    return {"t0": t0 + rng.uniform(0, .35, n), "a": a, "r": rng.uniform(*r0, n), "h": rng.uniform(*h0, n),
            "vz": rng.uniform(.7, 1.3, n) * rise, "w": spiral * rng.uniform(.7, 1.3, n),
            "life": rng.uniform(*life, n), "big": rng.random(n) < big}


def mote_pos(m, t):
    age = t - m["t0"]
    ang = m["a"] + m["w"] * np.maximum(age, 0)
    r = m["r"] * (1 - .35 * np.clip(age / m["life"], 0, 1) * (m["w"] != 0))
    return np.column_stack([np.cos(ang) * r, np.sin(ang) * r, m["h"] + m["vz"] * np.maximum(age, 0)]), age


def draw_motes(img, m, t, pal, origin):
    p, age = mote_pos(m, t)
    on = (age >= 0) & (age < m["life"])
    if not on.any():
        return
    p0, _ = mote_pos(m, t - .05)
    order = np.argsort((p[:, 0] + p[:, 1] + p[:, 2])[on])
    idx = np.flatnonzero(on)[order]
    for i in idx:
        k = age[i] / m["life"][i]
        col = pal[0] if k < .25 else pal[1] if k < .6 else pal[2]
        a = 255 if k < .75 else int(255 * (1 - k) / .25)
        x, y = to_screen(*p[i], origin)
        bx, by = to_screen(*p0[i], origin)
        n = int(min(4, math.hypot(bx - x, by - y)))
        for j in range(1, n + 1):  # the trail
            put(img, x + (bx - x) * j / max(n, 1), y + (by - y) * j / max(n, 1), pal[2], a // 2)
        put(img, x, y, col, a)
        if m["big"][i]:
            for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                put(img, x + dx, y + dy, pal[1], a)


def ground_ring(img, r, pal, a, origin, fill=None):
    pts = poly(ngon(r, 32))
    if fill is not None:
        cx, cy = to_screen(0, 0, 0, origin)
        for yy in range(int(-r / 2) - 1, int(r / 2) + 2):
            for xx in range(int(-r) - 1, int(r) + 2):
                if (xx / r) ** 2 + (yy / (r / 2)) ** 2 <= 1:
                    put(img, cx + xx, cy + yy, fill[0], fill[1])
    for x, y in pts:
        sx, sy = to_screen(x, y, 0, origin)
        put(img, sx, sy, pal, a)


def column(img, height, half, pal, a, origin):
    cx, cy = to_screen(0, 0, 0, origin)
    for yy in range(int(height)):
        for xx in range(-int(half), int(half) + 1):
            edge = abs(xx) >= half - 1
            fall = 1 - yy / height
            put(img, cx + xx, cy - yy, pal[0] if edge else pal[1], int(a * fall * (1 if edge else .45)))


def piece_frames(style, side, origin=ORIGIN, seed=0):
    """An ally (gold, rising) or enemy (purple, sinking) piece on one cell, from .45 s."""
    rng = np.random.default_rng(seed + (1 if side == "ally" else 2) + ord(style))
    pal = GOLD if side == "ally" else PURPLE
    up = side == "ally"
    t0 = .45
    more, big = (1.0, .3) if up else (1.5, .55)  # purple reads weaker on grass: more of it, bigger
    if style == "a":
        m = motes(rng, int(40 * more), t0, 70 if up else -40, 0, (2, 13), (.5, .9), (0, 6) if up else (20, 34), big)
    elif style == "b":
        m = motes(rng, int(28 * more), t0 + .1, 55 if up else -45, 0, (2, 10), (.4, .8), (0, 10) if up else (18, 30), big)
    else:
        m = motes(rng, int(46 * more), t0, 50 if up else -40, 7 if up else -7, (8, 14), (.6, 1.0), (0, 4) if up else (22, 34), big)
    n = int(SECS * FPS)
    frames = []
    for f in range(n):
        t = f / FPS
        img = np.zeros((SIZE[1], SIZE[0], 4), np.uint8)
        k = (t - t0) / .9
        if 0 <= k <= 1:
            a = int(255 * (1 - k) ** .7)
            if style == "a":
                if up:
                    ground_ring(img, 6 + 10 * k ** .5, pal[1], a, origin)
                else:
                    ground_ring(img, 16 - 8 * k ** .7, pal[1], a, origin, (pal[3], int(110 * (1 - k))))
            elif style == "b":
                if up:
                    column(img, 46 * min(1, k * 4), 8 * (1 - .5 * k), pal, int(220 * (1 - k)), origin)
                else:
                    for j in range(2):  # rings pressing down from above the head to the ground
                        kk = k * 1.6 - j * .45
                        if 0 <= kk <= 1:
                            hh = 50 * (1 - kk ** .6)
                            for x, y in poly(ngon(12 - 4 * kk, 24)):
                                sx, sy = to_screen(x, y, hh, origin)
                                put(img, sx, sy, pal[1], int(255 * (1 - kk)))
                    ground_ring(img, 12, pal[2], int(160 * (1 - k)), origin, (pal[3], int(90 * (1 - k))))
            else:
                ground_ring(img, 14, pal[1] if up else pal[2], a, origin)
        draw_motes(img, m, t, pal, origin)
        frames.append(img)
    return frames


def cell_origin(c):
    return (ORIGIN[0] + c[0] - c[1], ORIGIN[1] + (c[0] + c[1]) / 2)


def stack(layers):
    out = layers[0].copy()
    for lay in layers[1:]:
        m = lay[:, :, 3] > 0
        a = lay[:, :, 3:4].astype(float) / 255
        out[m, :3] = (out[m, :3] * (1 - a[m]) + lay[m, :3] * a[m]).astype(np.uint8)
        out[m, 3] = np.maximum(out[m, 3], lay[m, 3])
    return out


def compose_all(name, frames, out, box):
    """The combined preview: the ground figure under units on the ally and enemy cells, the pieces over them.
    frames: (ground, pieces) pairs."""
    unit_frame = S.sheet_unit()
    ally, afoot = unit_frame(4)
    foe, ffoot = unit_frame(1)
    x0, y0, x1, y1 = box
    ox, oy = ORIGIN
    gif = []
    for fr in frames:
        bg = Image.new("RGBA", SIZE, (86, 118, 60, 255))
        d = ImageDraw.Draw(bg)
        for cx, cy in SQUARE:
            px, py = ox + cx - cy, oy + (cx + cy) / 2
            d.polygon([(px, py - 16), (px + 32, py), (px, py + 16), (px - 32, py)], outline=(120, 150, 92, 255))
        bg.alpha_composite(Image.fromarray(fr[0], "RGBA"))  # the ground figure lies under the people
        people = sorted([(c, ally, afoot) for c in ALLY_CELLS] + [(c, foe, ffoot) for c in ENEMY_CELLS],
                        key=lambda q: q[0][0] + q[0][1])
        for c, im, foot in people:
            px, py = cell_origin(c)
            bg.alpha_composite(im, (int(px - im.width / 2), int(py - foot)))
        bg.alpha_composite(Image.fromarray(fr[1], "RGBA"))
        bg = bg.crop((x0, y0, x1, y1))
        gif.append(bg.resize((bg.width * 3, bg.height * 3), Image.NEAREST).convert("RGB"))
    gif[0].save(out / f"{name}.gif", save_all=True, append_images=gif[1:], duration=1000 // FPS, loop=0)
    cols = 6
    fw, fh = (x1 - x0) * 2, (y1 - y0) * 2
    sheet = Image.new("RGB", (fw * cols, fh * ((len(gif) + cols - 1) // cols)), (40, 40, 40))
    for i, g in enumerate(gif):
        sheet.paste(g.resize((fw, fh), Image.NEAREST), ((i % cols) * fw, (i // cols) * fh))
    sheet.save(out / f"{name}.png")


def draft(style):
    def run(out):
        name = f"palzin_{style}"
        ground = ground_frames(style)
        S.compose(name + "_ground", ground, "square", out)
        S.compose(name + "_ally", piece_frames(style, "ally"), "single", out)
        S.compose(name + "_enemy", piece_frames(style, "enemy"), "single", out)
        pieces = [piece_frames(style, "ally", cell_origin(c), i) for i, c in enumerate(ALLY_CELLS)]
        pieces += [piece_frames(style, "enemy", cell_origin(c), i) for i, c in enumerate(ENEMY_CELLS)]
        over = [stack([p[f] for p in pieces]) for f in range(len(ground))]
        frames = list(zip(ground, over))
        compose_all(name, frames, out, S.crop_box([stack([g, o]) for g, o in frames]))
        print(out / f"{name}.gif")
    return run


SPELLS = {"palzin_a": draft("a"), "palzin_b": draft("b"), "palzin_c": draft("c")}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("names", nargs="*")
    ap.add_argument("--out", default=str(ROOT / "reports/spellsim"))
    args = ap.parse_args()
    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    for name in args.names or SPELLS:
        SPELLS[name](out)


if __name__ == "__main__":
    main()
