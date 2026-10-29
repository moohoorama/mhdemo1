"""Battle-map scenery: soft ground decoration, pine variants with a wind sway, and tree shadows.

  python3 gen_scenery.py            # writes ../scenery/scenery.png, scenery.json and preview_*.png
  python3 gen_scenery.py --install  # also copies the sheet and json into ver5/assets/graphics/map

Two rules decide how a piece is drawn. Anything that changes a battle (trees: cover and movement cost) keeps the
game's look: ink outline and the three pine greens of ver5's forest sprites. Anything that is only decoration is soft:
no outline, tones close to the ground it lies on, partly transparent. The ground tiles stay as they are; decoration
is stamped on top, so no tile seam can show.

The pines come from one model: a stack of cones lit from the left, rasterised row by row (the class-sprite-3d route
of modelling first, then converting to the dot palette, kept simple because a cone needs no renderer). Every pine has
4 sway frames: the crown rows shift sideways by up to a pixel or two, more toward the tip, while the trunk stays.
"""
import json
import math
import os
import random

from PIL import Image, ImageDraw

import common
import gen_pixel_kit

OUT = os.path.join(common.ROOT, "scenery")

INK, DARK, MID, LIGHT, TRUNK = (11, 16, 27), (34, 48, 26), (64, 88, 38), (96, 124, 52), (61, 33, 29)
GRASS, SOIL, FLOOR, FLOOR_JOINT = (112, 143, 77), (187, 158, 106), (147, 163, 175), (99, 114, 128)
SHADOW = (18, 26, 14)
WATER = (44, 108, 119)


def mix(a, b, t):
    return tuple(round(a[i] + (b[i] - a[i]) * t) for i in range(3))


def shade(c, f):
    return tuple(max(0, min(255, round(v * f))) for v in c)


class Stamp:
    """A small RGBA picture with a pivot; put() blends one pixel over what is there."""

    def __init__(self, w, h, pivot=None):
        self.im = Image.new("RGBA", (w, h))
        self.px = self.im.load()
        self.w, self.h = w, h
        self.pivot = pivot or (w // 2, h - 1)

    def put(self, x, y, c, a=1.0):
        if not (0 <= x < self.w and 0 <= y < self.h) or a <= 0:
            return
        r, g, b, da = self.px[x, y]
        da /= 255
        oa = a + da * (1 - a)
        if oa <= 0:
            return
        self.px[x, y] = tuple(round((c[i] * a + (r, g, b)[i] * da * (1 - a)) / oa) for i in range(3)) + (round(oa * 255),)


# ---------------------------------------------------------------- soft ground decoration


def tuft(seed, base, h=5, w=7, dry=0.0):
    """grass blades: dark roots, mid body, a lit tip; a few thin vertical strokes leaning a little"""
    r = random.Random(seed)
    s = Stamp(w, h + 1, (w // 2, h))
    root = mix(shade(base, .74), (150, 128, 70), dry)
    body = mix(shade(base, .90), (176, 150, 80), dry)
    tip = mix(shade(base, 1.14), (206, 184, 104), dry)
    for i in range(r.randint(3, 5)):
        x = 1 + round((w - 3) * (i + r.random() * .6) / 4.5)
        top = r.randint(max(2, h - 3), h)
        lean = r.choice((-1, 0, 0, 1))
        for k in range(top):
            xx = x + (lean if k >= top // 2 else 0)
            y = h - 1 - k
            s.put(xx, y, root if k == 0 else (tip if k == top - 1 else body), .9 if k < top - 1 else .8)
    return s


def flowers(seed, base, hue, n=None):
    """a few one-pixel heads in a patch, with a pale highlight pixel; low saturation so the field stays calm"""
    r = random.Random(seed)
    s = Stamp(11, 6, (5, 4))
    for _ in range(n or r.randint(3, 6)):
        x, y = r.randint(1, 9), r.randint(0, 4)
        c = mix(hue, base, .2)
        s.put(x, y + 1, shade(base, .78), .6)
        s.put(x, y, c, .95)
        if r.random() < .4:
            s.put(x + 1, y, mix(c, (255, 255, 240), .45), .8)
    return s


def pebbles(seed, tone, n=None):
    """round stones lit top-left, a darker lower rim, no outline"""
    r = random.Random(seed)
    s = Stamp(11, 6, (5, 4))
    for _ in range(n or r.randint(1, 3)):
        w, h = r.choice(((3, 2), (4, 3), (5, 3), (3, 2)))
        x, y = r.randint(0, 11 - w), r.randint(0, 6 - h)
        body = shade(tone, r.uniform(.92, 1.06))
        for j in range(h):
            for i in range(w):
                if (i in (0, w - 1)) and (j in (0, h - 1)):
                    continue
                c = body
                if j == 0 and i < w - 1:
                    c = shade(body, 1.16)
                if j == h - 1 or (i == w - 1 and j > 0):
                    c = shade(body, .8)
                s.put(x + i, y + j, c, .88)
        s.put(x + 1, y + h, shade(SOIL if tone[0] > 140 else GRASS, .7), .35)
    return s


def blob(w, h, color, alpha, seed, rough=.0, core=1.0):
    """a soft iso-ellipse: alpha falls off in three dithered steps toward the rim, so it stays pixel art"""
    r = random.Random(seed)
    s = Stamp(w, h, (w // 2, h // 2))
    for y in range(h):
        for x in range(w):
            d = math.hypot((x + .5 - w / 2) / (w / 2), (y + .5 - h / 2) / (h / 2))
            d += (r.random() - .5) * rough
            if d >= 1:
                continue
            level = 1.0 if d < .5 * core else (.62 if d < .75 else .3)
            if level < 1 and (x + y) % 2 == 0 and d > .8:
                continue
            s.put(x, y, color, alpha * level)
    return s


def crack(seed, tone, length, a=.7):
    """a thin meandering crack with one branch, a faint lit lip below"""
    r = random.Random(seed)
    w = length + 4
    s = Stamp(w, 9, (w // 2, 4))
    x, y = 1, 4
    pts = []
    for i in range(length):
        pts.append((x, y))
        x += 1
        if r.random() < .38:
            y += r.choice((-1, 1))
            y = max(1, min(7, y))
    for k, (px_, py_) in enumerate(pts):
        s.put(px_, py_ + 1, mix(tone, (255, 255, 255), .25), a * .35)
        s.put(px_, py_, shade(tone, .55), a)
    if len(pts) > 8:
        bx, by = pts[len(pts) // 2]
        d = r.choice((-1, 1))
        for k in range(1, 4):
            s.put(bx + k, by + d * k // 2 + d, shade(tone, .6), a * .85)
    return s


def moss(seed):
    r = random.Random(seed)
    s = Stamp(10, 5, (5, 3))
    for _ in range(r.randint(7, 13)):
        x, y = r.randint(0, 9), r.randint(0, 4)
        s.put(x, y, mix((84, 118, 70), FLOOR, r.uniform(.05, .3)), r.uniform(.45, .75))
    return s


def litter(seed, n=None):
    """fallen needles and leaves: small flecks in a few earth tones"""
    r = random.Random(seed)
    s = Stamp(13, 6, (6, 4))
    tones = [(110, 98, 50), (146, 106, 52), (84, 60, 36), (126, 118, 62)]
    for _ in range(n or r.randint(4, 8)):
        x, y = r.randint(0, 11), r.randint(0, 5)
        c = r.choice(tones)
        s.put(x, y, c, .8)
        if r.random() < .5:
            s.put(x + 1, y, shade(c, .85), .7)
    return s


def twig(seed):
    r = random.Random(seed)
    s = Stamp(8, 4, (4, 2))
    x, y = 0, r.randint(1, 2)
    for _ in range(r.randint(4, 7)):
        s.put(x, y, mix(TRUNK, (130, 100, 60), .35), .75)
        x += 1
        if r.random() < .3:
            y = max(0, min(3, y + r.choice((-1, 1))))
    return s


def mushroom(seed):
    s = Stamp(4, 4, (2, 3))
    s.put(1, 2, (214, 206, 188), .9)
    s.put(1, 3, (214, 206, 188), .9)
    s.put(0, 1, (184, 80, 70), .9)
    s.put(1, 1, (198, 92, 78), .9)
    s.put(2, 1, (184, 80, 70), .9)
    s.put(1, 0, (198, 92, 78), .9)
    return s


def bush(seed, w, h):
    """a low leafy clump for the forest floor: overlapping round lobes, lit top-left, dark underside; soft (no outline)"""
    r = random.Random(seed)
    s = Stamp(w, h + 2, (w // 2, h))
    base = h - 1
    mask = [[0] * w for _ in range(h)]
    lobes = max(2, round(w / 4.2))
    for i in range(lobes):
        rad = h * r.uniform(.36, .5)
        cx = rad + (w - 2 * rad) * (i / (lobes - 1) if lobes > 1 else .5) + r.uniform(-.6, .6)
        cy = base - rad * r.uniform(.75, 1.0)
        for y in range(h):
            for x in range(w):
                if ((x + .5 - cx) / (rad * 1.15)) ** 2 + ((y + .5 - cy) / rad) ** 2 <= 1:
                    mask[y][x] = 1
    body, lit, deep = (44, 66, 32), (76, 104, 48), (30, 46, 24)
    for x in range(w):
        for y in range(h):
            if not mask[y][x]:
                continue
            top = y == 0 or not mask[y - 1][x]
            left = x == 0 or not mask[y][x - 1]
            under = y == h - 1 or not mask[y + 1][x]
            c = deep if under else (lit if top or (left and y < base - 1) else body)
            s.put(x, y, c, .93)
    for x in range(w):
        s.put(x + 1, h, SHADOW, .2)
    return s


def blades_dark(seed):
    """a darker fern-like tuft for under the trees"""
    return tuft(seed, (44, 66, 32), h=6, w=9)


def shrub(seed, w, h):
    """a low dry shrub for cliff tops: olive clump, lit top-left, no outline"""
    r = random.Random(seed)
    s = Stamp(w, h + 1, (w // 2, h - 1))
    for y in range(h):
        for x in range(w):
            d = ((x + .5 - w / 2) / (w / 2)) ** 2 + ((y + .5 - h * .6) / (h * .6)) ** 2
            if d <= 1 + (r.random() - .5) * .35:
                c = (122, 134, 66) if (y < h * .4 and x < w * .6) else ((84, 100, 52) if y > h * .65 else (100, 116, 58))
                s.put(x, y, c, .9)
    for x in range(1, w):
        s.put(x, h, (90, 70, 42), .22)
    return s


def reed(seed, h):
    """a clump of reeds standing in the shallows: thin blades leaning a little, some with a cattail head; soft"""
    r = random.Random(seed)
    w = 11
    s = Stamp(w, h + 2, (w // 2, h))
    for x in range(2, w - 2):
        s.put(x, h + 1, (20, 62, 72), .28)
    for i in range(r.randint(3, 6)):
        x0 = 2 + round((w - 5) * (i + r.random() * .5) / 5)
        top = r.randint(h - 3, h)
        lean = r.choice((-1, 0, 0, 1))
        for k in range(top):
            x = x0 + (lean if k > top // 2 else 0)
            y = h - 1 - k
            c = (56, 92, 50) if k < 2 else ((98, 128, 62) if k < top - 1 else (142, 150, 82))
            s.put(x, y, c, .88)
        if top >= h - 1 and r.random() < .45:
            hx = x0 + lean
            s.put(hx, h - top, (112, 72, 42), .95)
            s.put(hx, h - top + 1, (112, 72, 42), .95)
    return s


MARSH_INK = {"a": (150, 168, 96), "b": (110, 140, 78), "c": (74, 100, 64), "r": (122, 84, 52), "w": (164, 186, 194)}
MARSH = [  # reed clump, cattails, sedge, thin reeds; each stands in a pale ring of water (w)
    """
....a....
..a.a..a.
..a.b..a.
.ab.b.ab.
.ab.bbab.
.bbcbbbb.
.bccbcbb.
.cccccc..
.wwwwwww.""",
    """
..r....
..r..r.
..a..r.
.ra..a.
.ra.ba.
.ba.b..
.bb.b.a
.bbcbba
.cbcbc.
..ccc..
.wwwww.""",
    """
...a...a...
.a.ab.ab.a.
.abbbbbbba.
bbcbbcbbcbb
.ccccccccc.
..wwwwwww..""",
    """
.a...
.a..a
.a..a
.ab.a
.bb.b
.bb.b
abbbb
bbcbb
.ccc.
wwwww""",
]


def marsh(art):
    """a marsh plant of the ford drawn as ASCII dots, its water ring half seen through"""
    rows = art.strip("\n").split("\n")
    s = Stamp(len(rows[0]), len(rows))
    for y, row in enumerate(rows):
        for x, ch in enumerate(row):
            if ch in MARSH_INK:
                s.put(x, y, MARSH_INK[ch], .6 if ch == "w" else 1)
    return s


def lily(seed, big):
    """a lily pad (a notched ellipse) lit top-left; now and then a flower pixel"""
    r = random.Random(seed)
    w, h = (7, 4) if big else (5, 3)
    s = Stamp(w + 3, h + 2, (w // 2 + 1, h // 2 + 1))
    ox, oy = 1, 1
    for y in range(h):
        for x in range(w):
            d = ((x + .5 - w / 2) / (w / 2)) ** 2 + ((y + .5 - h / 2) / (h / 2)) ** 2
            if d <= 1 and not (x == w - 1 and y == h // 2):
                c = (92, 156, 96) if (y == 0 or x == 0) else (62, 124, 76)
                s.put(ox + x, oy + y, c, .92)
    if r.random() < .5:
        s.put(ox + w // 2, oy + h // 2 - (1 if h > 3 else 0), r.choice(((240, 226, 230), (226, 150, 170), (240, 230, 160))), .95)
    return s


def foam(seed, length, up):
    """a broken pale streak along the shore (up: rising to the right, else falling); soft"""
    r = random.Random(seed)
    s = Stamp(length + 2, length // 2 + 4, (length // 2 + 1, (length // 2 + 4) // 2))
    for i in range(length):
        if r.random() < .22:
            continue
        y = (length // 2 - i // 2) if up else (i // 2)
        s.put(1 + i, 1 + y, (214, 238, 234), .6)
        if r.random() < .5:
            s.put(1 + i, 2 + y, (170, 214, 214), .28)
    return s


def glint(frame):
    """a twinkle on the water: nothing, a dot, a small cross, a dot"""
    s = Stamp(5, 3, (2, 1))
    if frame in (1, 3):
        s.put(2, 1, (206, 236, 240), .8)
    elif frame == 2:
        s.put(2, 1, (230, 248, 250), .95)
        s.put(1, 1, (190, 226, 232), .45)
        s.put(3, 1, (190, 226, 232), .45)
        s.put(2, 0, (190, 226, 232), .3)
    return s


def ripple(frame):
    """an expanding ring (iso ellipse, 2:1) that fades; frame 3 is empty"""
    s = Stamp(15, 9, (7, 4))
    if frame < 3:
        rx = 3 + 2 * frame
        ry = rx / 2
        for a in range(0, 360, 12):
            x = 7 + round(rx * math.cos(math.radians(a)))
            y = 4 + round(ry * math.sin(math.radians(a)))
            s.put(x, y, (176, 218, 222), (.55, .4, .24)[frame])
    return s


# ---------------------------------------------------------------- structures with life


def flutter(rows, frame, amp=1.6, step=.5):
    """a hanging cloth that waves: rows is a list of pixel lists, first pixel against the pole, last the free edge;
    a row grows or shrinks by a pixel or two as the wave passes, so the free edge swings"""
    out = []
    n = len(rows)
    for r, px in enumerate(rows):
        dx = round(amp * (r / max(1, n - 1)) * math.sin(math.pi / 2 * frame + r * step))
        px = list(px)
        if dx > 0:
            px = [px[0]] + [px[1]] * dx + px[1:]
        elif dx < 0:
            px = [px[0]] + px[1 - dx:] if len(px) > 2 - dx else px
        out.append(px)
    return out


def camp_parts():
    """the camp flag cloth's four waving frames, cut from the sites.png camp's pixels, and where the cloth hangs from
    the pivot (on the pole of gen_pixel_kit.py's camp)"""
    im, pivot = common.site("camp")
    ip = im.load()
    x0, y0, x1, y1 = 50, 40, 56, 52  # cloth, free edge included, inclusive
    rows = []
    for y in range(y0, y1 + 1):
        row = [ip[x, y] for x in range(x0, x1 + 1)]
        while row and row[-1][3] == 0:
            row.pop()
        rows.append(row)
    frames = []
    for f in range(4):
        st = Stamp(10, y1 - y0 + 1, (0, 0))
        for y, px in enumerate(flutter(rows, f, amp=1.2, step=.55)):
            for x, c in enumerate(px):
                if c[3]:
                    st.px[x, y] = c
        frames.append(st)
    return frames, (x0 - pivot[0], y0 - pivot[1])


def castle_flag(frame):
    """a small red pennant on a pole for the castle's back turret, same red and ink as the camp flag"""
    INKC, POLE, RED, HI = (11, 16, 27), (102, 55, 32), (171, 22, 51), (237, 35, 66)
    tri = [[RED, RED, RED, RED, RED, INKC], [HI, RED, RED, RED, INKC], [RED, RED, RED, INKC], [RED, INKC]]
    st = Stamp(12, 13, (1, 12))
    for y in range(3, 13):
        st.put(0, y, INKC)
        st.put(1, y, POLE)
    st.put(1, 2, INKC)
    for y, px in enumerate(flutter([[POLE] + r for r in tri], frame, amp=1.4, step=.9)):
        for x, c in enumerate(px[1:]):
            st.put(2 + x, 3 + y, c)
    return st


def smoke(frame):
    """one stage of a chimney plume: two puffs half a loop apart, each a lumpy cluster rising, widening and fading
    while it drifts right; soft, no outline, the lower right a little darker"""
    st = Stamp(20, 34, (5, 33))
    for j in range(2):
        age = (frame + 4 * j) % 8
        cy = 32 - age * 3.8
        cx = 5 + age * .7
        rad = 1.8 + age * .6
        alpha = .8 * (1 - age / 8) ** .8 * min(1, (age + 1) / 2)
        lobes = [(-.55, .1, .8), (.5, -.35, .75), (.05, .55, .7), (0, -.2, .85)]
        for yy in range(34):
            for xx in range(20):
                inside = [(math.hypot((xx + .5 - cx - lx * rad) / (rad * lr), (yy + .5 - cy - ly * rad) / (rad * lr * .85)), lr) for lx, ly, lr in lobes]
                d = min(v for v, _ in inside)
                if d >= 1:
                    continue
                edge = d > .72 and (xx + yy) % 2 == 0
                if edge:
                    continue
                shade_dark = (xx - cx) + (yy - cy) > rad * .5
                st.put(xx, yy, (186, 186, 182) if shade_dark else (218, 216, 208), alpha * (.7 if d > .55 else 1))
    return st


# ---------------------------------------------------------------- pines


def pine(spec, frame):
    """One pine, frame 0..3 of its sway. Returns a Stamp (ink outline, lit from the left) with its pivot at the trunk foot."""
    H, W = spec["H"], spec["W"]
    kind, tiers = spec.get("kind", "cone"), spec.get("tiers", 3)
    lean, amp = spec.get("lean", 0), spec.get("amp", 1.6)
    body, lit = (MID, LIGHT) if spec.get("tone") == "young" else (DARK, MID)
    trunk = spec.get("trunk", 2)
    pad = 4
    cw, ch = W + 2 * pad, H + trunk + 2 * pad
    cx = cw / 2
    wave = (0, 1, 0, -1)[frame]
    rows = []  # (left, right, row) of the crown, top to bottom
    if kind == "snag":
        rows = []
    for y in range(H):
        t = (y + .5) / H
        if kind == "cone":
            hw = W / 2 * (t ** .85)
        elif kind == "tier":
            s = (t * tiers) % 1
            hw = W / 2 * (t ** .8) * (.74 + .26 * s)
        elif kind == "round":
            hw = W / 2 * min(1.0, (t * 1.6) ** .75) * (1 - .12 * max(0, t - .7) / .3)
        else:
            hw = 0
        if kind != "snag":
            hw = max(.5, hw)
        off = lean * (1 - t) ** 1.2 + amp * wave * (1 - t) ** 1.5
        lo, hi = round(cx - hw + off), round(cx + hw + off)
        rows.append((lo, max(lo + 1, hi), y))
    st = Stamp(cw, ch, (int(cx), pad + H + trunk - 1))
    sil = set()
    if kind == "snag":
        tx = int(cx)
        for k in range(H):
            y = pad + H + trunk - 1 - k
            for dx in (0, 1):
                sil.add((tx + dx + round(.8 * wave * (k / H) ** 2), y))
        for k, (dy, d, ln) in enumerate(((H * .45, -1, 3), (H * .62, 1, 3), (H * .8, -1, 2))):
            y = pad + H + trunk - 1 - int(dy)
            for i in range(1, ln + 1):
                sil.add((tx + (1 if d > 0 else 0) + d * i, y - i // 2))
        for p in sil:
            st.put(p[0], p[1], TRUNK)
    else:
        for lo, hi, y in rows:
            wpx = hi - lo
            litw = max(1, round(wpx * .28))
            if frame in (1, 3) and (y + frame) % 3 == 0 and wpx > 5:
                litw += 1 if frame == 1 else -1
            for x in range(lo, hi):
                sil.add((x, pad + y))
                st.put(x, pad + y, lit if x - lo < litw and wpx > 2 else body)
        tx = int(cx)
        for k in range(trunk):
            for dx in (0, 1):
                sil.add((tx + dx, pad + H + k))
                st.put(tx + dx, pad + H + k, TRUNK)
    edge = {(x + a, y + b) for (x, y) in sil for a, b in ((1, 0), (-1, 0), (0, 1), (0, -1))} - sil
    for x, y in edge:
        st.put(x, y, INK)
    return st


PINES = [
    # name, spec, weight (interior), weight (edge)
    ("pine_a", dict(H=15, W=12, kind="cone"), 5, 3),
    ("pine_b", dict(H=19, W=11, kind="cone", trunk=2), 3, 1),
    ("pine_c", dict(H=20, W=14, kind="tier", tiers=3), 3, 1),
    ("pine_d", dict(H=14, W=14, kind="round"), 2, 2),
    ("pine_e", dict(H=13, W=9, kind="cone", tone="young"), 1, 3),
    ("pine_f", dict(H=16, W=12, kind="tier", tiers=2, tone="young"), 1.2, 2),
    ("pine_g", dict(H=23, W=10, kind="cone", lean=-1, trunk=3, amp=1.8), 1.5, 0.5),
    ("pine_h", dict(H=15, W=12, kind="cone", lean=1), 1.5, 1.5),
    ("pine_i", dict(H=9, W=7, kind="cone", tone="young", trunk=1, amp=1.2), 1, 4),
    ("snag", dict(H=15, W=6, kind="snag", amp=1.2), .4, .8),
]


def shadow_stamp(w):
    """the unit-shadow look: one flat dark-green shape, hard edge, thrown to the right and down"""
    h = max(3, round(w * .36))
    s = Stamp(w + 4, h + 2, (2, h // 2 + 1))
    for y in range(h):
        for x in range(w):
            if ((x + .5 - w / 2) / (w / 2)) ** 2 + ((y + .5 - h / 2) / (h / 2)) ** 2 <= 1:
                s.px[x + 3, y + 1] = SHADOW + (130,)
    return s


# ---------------------------------------------------------------- assembly


def build():
    sprites, groups = {}, {}

    def add(group, name, st):
        sprites[name] = st
        groups.setdefault(group, []).append(name)

    for i in range(7):
        add("grass.tuft", f"grass_tuft_{i}", tuft(100 + i, GRASS, dry=0.0 if i < 5 else .35))
    for i, hue in enumerate([(238, 236, 222), (232, 208, 96), (232, 214, 110), (214, 150, 172), (160, 178, 226), (238, 236, 222), (196, 150, 214)]):
        add("grass.flower", f"flower_{i}", flowers(200 + i, GRASS, hue))
    for i in range(5):
        add("grass.pebble", f"grass_pebble_{i}", pebbles(300 + i, (150, 148, 130)))
    for i, (w, h) in enumerate([(16, 7), (22, 9), (30, 12), (20, 8), (26, 10)]):
        add("grass.patch", f"grass_patch_{i}", blob(w, h, mix(SOIL, GRASS, .25), .5, 400 + i, rough=.25))
    for i, (w, h) in enumerate([(44, 18), (56, 22), (38, 15)]):
        add("grass.tone", f"grass_dark_{i}", blob(w, h, shade(GRASS, .8), .20, 450 + i, rough=.2))
    for i, (w, h) in enumerate([(44, 18), (52, 20), (36, 14)]):
        add("grass.tone", f"grass_light_{i}", blob(w, h, mix(GRASS, (200, 210, 110), .5), .17, 470 + i, rough=.2))

    for i in range(6):
        add("waste.tuft", f"waste_tuft_{i}", tuft(500 + i, SOIL, h=4, w=6, dry=.9))
    for i in range(6):
        add("waste.crack", f"waste_crack_{i}", crack(520 + i, SOIL, 8 + 3 * i))
    for i in range(6):
        add("waste.pebble", f"waste_pebble_{i}", pebbles(540 + i, (170, 140, 104)))
    for i in range(3):
        add("waste.rock", f"waste_rock_{i}", pebbles(560 + i, (156, 134, 108), n=1))
    for i, (w, h) in enumerate([(20, 8), (28, 11), (36, 14), (24, 9)]):
        add("waste.patch", f"waste_dark_{i}", blob(w, h, shade(SOIL, .84), .34, 580 + i, rough=.25))
    for i, (w, h) in enumerate([(26, 10), (34, 13)]):
        add("waste.patch", f"waste_light_{i}", blob(w, h, mix(SOIL, (240, 220, 170), .5), .26, 600 + i, rough=.25))

    for i in range(6):
        add("floor.crack", f"floor_crack_{i}", crack(700 + i, FLOOR_JOINT, 7 + 3 * i, a=.62))
    for i in range(4):
        add("floor.moss", f"floor_moss_{i}", moss(720 + i))
    for i, (w, h) in enumerate([(18, 7), (24, 9), (14, 6)]):
        add("floor.stain", f"floor_stain_{i}", blob(w, h, shade(FLOOR_JOINT, .85), .26, 740 + i, rough=.3))
    for i in range(5):
        add("floor.pebble", f"floor_pebble_{i}", pebbles(760 + i, (176, 184, 192)))
    for i in range(4):
        add("floor.litter", f"floor_litter_{i}", litter(780 + i, n=3))

    for i in range(6):
        add("forest.litter", f"litter_{i}", litter(800 + i))
    for i, (w, h) in enumerate([(8, 5), (11, 7), (14, 9), (9, 6), (12, 7)]):
        add("forest.bush", f"bush_{i}", bush(820 + i, w, h))
    for i in range(4):
        add("forest.fern", f"fern_{i}", blades_dark(840 + i))
    for i in range(4):
        add("forest.twig", f"twig_{i}", twig(860 + i))
    for i in range(2):
        add("forest.mushroom", f"mushroom_{i}", mushroom(880 + i))
    for i, (w, h) in enumerate([(52, 22), (60, 26), (46, 20)]):
        add("forest.shade", f"forest_shade_{i}", blob(w, h, (20, 32, 16), .38, 900 + i, rough=.2))

    for i, (w, h) in enumerate([(18, 7), (22, 8), (16, 6)]):
        add("cliff.patch", f"cliff_dark_{i}", blob(w, h, shade(SOIL, .84), .32, 940 + i, rough=.25))
    for i, (w, h) in enumerate([(18, 7), (22, 8)]):
        add("cliff.patch", f"cliff_light_{i}", blob(w, h, mix(SOIL, (240, 220, 170), .5), .24, 950 + i, rough=.25))
    for i, (w, h) in enumerate([(7, 5), (9, 6), (6, 4)]):
        add("cliff.shrub", f"shrub_{i}", shrub(960 + i, w, h))
    for g in ("tuft", "crack", "pebble", "rock"):
        groups[f"cliff.{g}"] = list(groups[f"waste.{g}"])

    for i, h in enumerate((6, 8, 9, 7, 8)):
        add("water.reed", f"reed_{i}", reed(1000 + i, h))
    for i in range(4):
        add("water.lily", f"lily_{i}", lily(1020 + i, i % 2 == 0))
    for i, (l, up) in enumerate([(10, True), (13, True), (9, True), (10, False), (13, False), (9, False)]):
        add("water.foam_up" if up else "water.foam_down", f"foam_{'u' if up else 'd'}{i % 3}", foam(1040 + i, l, up))
    for i, (w, h) in enumerate([(48, 18), (60, 22), (40, 15)]):
        add("water.tone", f"water_dark_{i}", blob(w, h, (24, 80, 94), .24, 1060 + i, rough=.2))
    for i, (w, h) in enumerate([(48, 18), (56, 20)]):
        add("water.tone", f"water_light_{i}", blob(w, h, (110, 176, 182), .15, 1070 + i, rough=.2))
    # the ford: marsh plants of its own, the shore's reeds and the grass's tufts; the plain: the grass's tufts and flowers
    # with the waste's pebbles, cracks and patches
    for i, art in enumerate(MARSH):
        add("ford.marsh", f"marsh_{i}", marsh(art))
    for g, src in (("reed", "water.reed"), ("tuft", "grass.tuft"), ("tone", "grass.tone")):
        groups[f"ford.{g}"] = list(groups[src])
    for g, src in (("tuft", "grass.tuft"), ("flower", "grass.flower"), ("pebble", "waste.pebble"), ("crack", "waste.crack"), ("patch", "waste.patch")):
        groups[f"plain.{g}"] = list(groups[src])
    # looping strips: frame sprite names and how long each frame shows (the game's FrameStrip)
    anim = {"glint": {"frames": [], "ms": 260}, "ripple": {"frames": [], "ms": 380}}
    for f in range(4):
        sprites[f"glint_{f}"] = glint(f)
        anim["glint"]["frames"].append(f"glint_{f}")
        sprites[f"ripple_{f}"] = ripple(f)
        anim["ripple"]["frames"].append(f"ripple_{f}")

    camp_flag, camp_at = camp_parts()
    for f in range(4):
        sprites[f"camp_flag_{f}"] = camp_flag[f]
        sprites[f"castle_flag_{f}"] = castle_flag(f)
    for f in range(8):
        sprites[f"smoke_{f}"] = smoke(f)
    anim["camp_flag"] = {"frames": [f"camp_flag_{f}" for f in range(4)], "ms": 170}
    anim["castle_flag"] = {"frames": [f"castle_flag_{f}" for f in range(4)], "ms": 150}
    anim["smoke"] = {"frames": [f"smoke_{f}" for f in range(8)], "ms": 220}
    sites = {
        "camp": {"overlays": [{"anim": "camp_flag", "at": list(camp_at)}]},
        "castle": {"overlays": [{"anim": "castle_flag", "at": [-1, -27]}]},
    }
    for n in range(len(gen_pixel_kit.VILLAGES)):   # each village picture smokes from its own chimneys
        sites["village_%d" % n] = {"overlays": [{"anim": "smoke", "at": list(at), "phase": 3 * i, "chance": (.4, .25)[i]}
                                                for i, at in enumerate(gen_pixel_kit.smoke_points(n))]}

    pines = []
    for name, spec, w_in, w_edge in PINES:
        frames = []
        for f in range(4):
            fn = f"{name}_{f}"
            sprites[fn] = pine(spec, f)
            frames.append(fn)
        size = "s" if spec["H"] < 12 else ("l" if spec["H"] >= 19 else "m")
        anim[name] = {"frames": frames, "ms": 300}
        pines.append({"anim": name, "shadow": f"shadow_{size}", "weight": [w_in, w_edge], "height": spec["H"]})
    for size, w in (("s", 8), ("m", 11), ("l", 14)):
        sprites[f"shadow_{size}"] = shadow_stamp(w)

    return sprites, groups, pines, anim, sites


SCATTER = {
    # expected stamps per small tile (a 32x16 diamond; a cell has four)
    "grass": {"tuft": 1.1, "flower": .16, "pebble": .11, "patch": .05, "tone": .07},
    "waste": {"tuft": .45, "crack": .20, "pebble": .35, "rock": .06, "patch": .09},
    "floor": {"crack": .14, "moss": .08, "stain": .07, "pebble": .18, "litter": .05},
    "forest": {"litter": 1.1, "bush": .55, "fern": .4, "twig": .25, "mushroom": .04, "shade": .55},
    "cliff": {"tuft": .30, "crack": .12, "pebble": .36, "rock": .09, "patch": .14, "shrub": .05},
    # reeds only where water meets land, lilies only away from it; foam is a chance per shore edge (a second
    # streak with foam2); glint and ripple are animated, per water tile
    "water": {"reed": .9, "lily": .10, "tone": .10, "foam": .85, "foam2": .35, "glint": .22, "ripple": .04},
    # marsh plants gather in clumps (internal/scenemap/scenery.go clumped)
    "ford": {"marsh": .35, "reed": .10, "tuft": .20, "tone": .06},
    "plain": {"tuft": .70, "pebble": .22, "crack": .08, "patch": .06, "flower": .04},
}


def pack(sprites, width=640):
    x = y = row = 0
    place = {}
    for name, st in sprites.items():
        if x + st.w + 1 > width:
            x, y, row = 0, y + row + 1, 0
        place[name] = (x, y)
        x += st.w + 1
        row = max(row, st.h)
    sheet = Image.new("RGBA", (width, y + row + 1))
    index = {}
    for name, st in sprites.items():
        px, py = place[name]
        sheet.paste(st.im, (px, py))
        index[name] = {"rect": [px, py, st.w, st.h], "pivot": list(st.pivot)}
    return sheet, index


def zoom(im, k):
    return im.resize((im.width * k, im.height * k), Image.NEAREST)


def preview(sprites, groups, pines, anim):
    """contact sheets: decoration on the ground colour it is meant for, pines with their 4 sway frames"""
    bgs = {"grass": GRASS, "waste": SOIL, "floor": FLOOR, "forest": mix(GRASS, DARK, .35), "cliff": SOIL, "water": WATER,
           "ford": (98, 120, 128), "plain": mix(GRASS, SOIL, .5)}
    k = 5
    for area, bg in bgs.items():
        names = [n for g, ns in groups.items() if g.startswith(area + ".") for n in ns]
        sheet = Image.new("RGBA", (1500, 900), bg + (255,))
        x = y = row = 8
        for n in names:
            st = sprites[n]
            im = zoom(st.im, k)
            if x + im.width + 10 > sheet.width:
                x, y, row = 8, y + row + 10, 0
            sheet.alpha_composite(im, (x, y))
            x += im.width + 10
            row = max(row, im.height)
        sheet.crop((0, 0, 1500, min(900, y + row + 10))).convert("RGB").save(os.path.join(OUT, f"preview_{area}.png"))
    sheet = Image.new("RGBA", (1500, 520), mix(GRASS, DARK, .2) + (255,))
    for i, p in enumerate(pines):
        x0, y0 = 10 + (i % 5) * 295, 14 + (i // 5) * 250
        for f, fn in enumerate(anim[p["anim"]]["frames"]):
            sheet.alpha_composite(zoom(sprites[fn].im, 3), (x0 + f * 72, y0))
    sheet.convert("RGB").save(os.path.join(OUT, "preview_pines.png"))


def install():
    """copy the sheet and its description next to the game's other map art (ver5 reads graphics/map/scenery.json)"""
    import shutil
    for name in ("scenery.png", "scenery.json"):
        shutil.copy(os.path.join(OUT, name), os.path.join(common.GAME, "map", name))
    print("installed into", os.path.join(common.GAME, "map"))


def main():
    import sys
    os.makedirs(OUT, exist_ok=True)
    sprites, groups, pines, anim, sites = build()
    sheet, index = pack(sprites)
    sheet.save(os.path.join(OUT, "scenery.png"))
    meta = {"image": "map/scenery.png", "sprites": index, "groups": groups, "scatter": SCATTER, "pines": pines, "anim": anim, "sites": sites,
            "note": "ground decoration is soft (no outline); pines keep the game's ink outline; see assets_candidate/tools/gen_scenery.py"}
    json.dump(meta, open(os.path.join(OUT, "scenery.json"), "w"), indent=1)
    preview(sprites, groups, pines, anim)
    print(f"{len(sprites)} sprites, sheet {sheet.size}")
    if "--install" in sys.argv:
        install()


if __name__ == "__main__":
    main()
