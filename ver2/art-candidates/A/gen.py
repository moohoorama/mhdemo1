#!/usr/bin/env python3
"""Candidate A — 영걸전 클래식: 24px top-down tiles, SD units, bottom info bar.

Run from ver2/: python3 art-candidates/A/gen.py
All sprites are authored at native resolution as palette-indexed pixel sets.
Every part is outlined with a 1px outer outline and composited in draw order,
so later parts cut clean separation lines into earlier ones.
"""
import json
import os
from collections import deque

from PIL import Image, ImageDraw, ImageFont

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "out")
DOC = os.path.join(HERE, "..", "..", "srpg_full_design.md")
FONT = "/Users/rosemily/orca/mhdemo1/ver1/assets/fonts/NotoSansKR.ttf"

PAL = {
    # unit palette
    "K": (20, 24, 34),  # outline
    "S": (238, 192, 142), "s": (186, 124, 86),  # skin
    "B": (52, 98, 164), "b": (30, 56, 104), "L": (118, 166, 222),  # 청람 armor
    "R": (190, 60, 50), "r": (112, 32, 34),  # enemy red
    "Y": (230, 184, 66), "y": (154, 108, 38),  # 황토 yellow
    "G": (220, 172, 78), "g": (140, 96, 44),  # 동금 gold
    "W": (244, 234, 206), "w": (198, 184, 144),  # 상아 ivory
    "M": (216, 222, 228), "m": (126, 137, 148),  # steel
    "H": (160, 96, 54), "h": (98, 56, 34),  # horse
    "J": (80, 156, 108), "j": (44, 96, 66), "o": (136, 198, 142),  # 옥록 jade
    "T": (140, 98, 58),  # wood
    "N": (40, 34, 42), "n": (88, 76, 92),  # hair/black
    "F": (196, 82, 60), "f": (140, 48, 38),  # 관우 face
    # terrain-only additions
    "P": (146, 186, 92), "p": (110, 156, 70), "q": (80, 124, 56),
    "D": (204, 168, 108), "d": (158, 124, 74),
    "A": (66, 130, 186), "a": (46, 94, 150), "i": (146, 200, 234),
    "Q": (30, 62, 44),
}
UNIT_KEYS = "KSsBbLRrYyGgWwMmHhJjoTNnFf"

CW, CH = 24, 28
AX, AY = 12, 25
TILE = 24
SCALE = 2


# ---------------------------------------------------------------- primitives
def put(p, rows, x0, y0, recolor=None):
    for j, row in enumerate(rows):
        for i, c in enumerate(row):
            if c != ".":
                p[(x0 + i, y0 + j)] = recolor.get(c, c) if recolor else c


def line(p, x0, y0, x1, y1, c):
    dx, dy = abs(x1 - x0), -abs(y1 - y0)
    sx, sy = (1 if x0 < x1 else -1), (1 if y0 < y1 else -1)
    err = dx + dy
    while True:
        p[(x0, y0)] = c
        if x0 == x1 and y0 == y1:
            return
        e2 = 2 * err
        if e2 >= dy:
            err += dy
            x0 += sx
        if e2 <= dx:
            err += dx
            y0 += sy


def outline(p):
    o = set()
    for (x, y) in p:
        for nx, ny in ((x + 1, y), (x - 1, y), (x, y + 1), (x, y - 1)):
            if (nx, ny) not in p:
                o.add((nx, ny))
    return o


def compose(parts, edge="K"):
    f = {}
    for p in parts:
        for q in outline(p):
            f[q] = edge
        f.update(p)
    return f


def shifted(p, dx, dy):
    return {(x + dx, y + dy): c for (x, y), c in p.items()}


# ---------------------------------------------------------------- unit parts
HEAD_HELM = [
    "...GG...",
    "..LLBB..",
    ".LLBBBB.",
    "LLBBBBBB",
    "bGGGGGGG",
    "bbbSSSNS",
    "bbbSSSNS",
    ".bbsSSSs",
    "..bssSs.",
]
HEAD_RIDER = HEAD_HELM[1:]
HEAD_BANDIT = [
    "........",
    "..NNNN..",
    ".NnNNNNN",
    "NNNNNNNN",
    "YYYYYYYY",
    "YNNSSSNS",
    "YNNSSSNS",
    ".NNsSSSs",
    "..NssSs.",
]
TORSO_ALLY = [
    ".BBBBB.",
    "BBLLBBb",
    "BLLBBBb",
    "BBBBBBb",
    "GGGGGGG",
    "bBBbBBb",
]
TORSO_BANDIT = [
    ".RRRRR.",
    "RRYRRRr",
    "RRRYRRr",
    "RRRRRRr",
    "yTyyyyy",
    "rRRrRRr",
]
TORSO_RIDER = [
    ".BBBB.",
    "BLLBBb",
    "BLBBBb",
    "GGGGGG",
    "bBBBBb",
]
HORSE = [
    "................NN..",
    "...............NHHh.",
    "..............NHHHHh",
    "..............NHHNHH",
    ".............NHHHHHH",
    ".............NHHH.hh",
    ".N..HHHHHHHHNHHHh...",
    "NN.HHHRRRRRRHHHHh...",
    "N..HHHRGGGGRHHHHh...",
    "...HHHHHHHHHHHHh....",
    "....hhhhhhhhhhh.....",
]


def head(kind, x, y):
    p = {}
    put(p, {"ally": HEAD_HELM, "rider": HEAD_RIDER}.get(kind, HEAD_BANDIT), x, y)
    return p


def leg(hip_x, hip_y, foot_x, lift, pants, boot):
    p = {}
    ground = 24 - lift
    knee_y = ground - 1
    for y in range(hip_y, knee_y + 1):
        t = (y - hip_y) / max(1, knee_y - hip_y)
        x = round(hip_x + (foot_x - hip_x) * t)
        p[(x, y)] = pants
        p[(x + 1, y)] = pants
    p[(foot_x, ground - 1)] = boot
    p[(foot_x + 1, ground - 1)] = boot
    for x in range(foot_x, foot_x + 3):
        p[(x, ground)] = boot
    return p


def arm(sx, sy, hx, hy, sleeve):
    p = {}
    line(p, sx, sy, hx, hy, sleeve)
    p[(hx, hy)] = "S"
    return p


def sword(hx, hy, blade):
    p = {(hx + 1, hy): "g", (hx, hy - 1): "G", (hx + 2, hy + 1): "G"}
    line(p, hx + 1, hy - 1, hx + 4, hy - 4, blade)
    p[(hx + 4, hy - 4)] = "W"
    return p


def foot_soldier(kind, dy, near, far, lift_n, lift_f, swing):
    ally = kind == "ally"
    pants_n, pants_f = ("W", "w") if ally else ("w", "g")
    torso = TORSO_ALLY if ally else TORSO_BANDIT
    sleeve = "B" if ally else "R"
    far_arm = arm(10, 15 + dy, 9 - swing, 18 + dy, "b" if ally else "r")
    hand_x, hand_y = 16 + swing, 17 + dy
    t = {}
    put(t, torso, 9, 14 + dy)
    parts = [
        far_arm,
        leg(10, 20, 10 + far, lift_f, pants_f, "N"),
        leg(12, 20, 12 + near, lift_n, pants_n, "N"),
        t,
        head(kind, 8, 5 + dy),
        sword(hand_x, hand_y, "M" if ally else "m"),
        arm(13, 15 + dy, hand_x, hand_y, sleeve),
    ]
    return compose(parts)


IDLE_DY = [0, 0, 0, 1, 1, 1, 0, 0]
WALK_NEAR = [3, 2, 0, -2, -3, -1, 1, 2]
WALK_LIFT_NEAR = [0, 0, 0, 0, 0, 1, 1, 0]
WALK_DY = [0, 1, 0, -1, 0, 1, 0, -1]
WALK_SWING = [-1, -1, 0, 1, 1, 1, 0, -1]


def infantry_frames(kind, walk=True):
    frames = [foot_soldier(kind, IDLE_DY[i], 1, -2, 0, 0, 0) for i in range(8)]
    if walk:
        for i in range(8):
            j = (i + 4) % 8
            frames.append(foot_soldier(kind, WALK_DY[i], WALK_NEAR[i], WALK_NEAR[j],
                                       WALK_LIFT_NEAR[i], WALK_LIFT_NEAR[j], WALK_SWING[i]))
    return frames


def horse_parts(dy, head_dy, rear_n, rear_f, front_n, front_f, lifts):
    body, hd = {}, {}
    for (x, y), c in _horse_map().items():
        (hd if (y <= 5 and x >= 13) else body)[(x + 2, y + 9 + dy)] = c
    hd = shifted(hd, 0, head_dy)
    ln, lf, fn, ff = lifts
    far = {}
    far.update(leg_h(6, 20 + dy, 6 + rear_f, lf, "h"))
    far.update(leg_h(15, 20 + dy, 15 + front_f, ff, "h"))
    near_r = leg_h(8, 20 + dy, 8 + rear_n, ln, "H")
    near_f = leg_h(17, 20 + dy, 17 + front_n, fn, "H")
    return far, body, hd, near_r, near_f


_HM = None


def _horse_map():
    global _HM
    if _HM is None:
        _HM = {}
        put(_HM, HORSE, 0, 0)
    return _HM


def leg_h(hx, hy, fx, lift, c):
    p = {}
    ground = 24 - lift
    for y in range(hy, ground):
        t = (y - hy) / max(1, ground - 1 - hy)
        p[(round(hx + (fx - hx) * t), y)] = c
    p[(fx, ground)] = "N"
    p[(fx + 1, ground)] = "N"
    return p


def cavalry(dy, rider_dy, head_dy, legs, lifts):
    far, body, hd, near_r, near_f = horse_parts(dy, head_dy, *legs, lifts)
    ry = dy + rider_dy
    t = {}
    put(t, TORSO_RIDER, 8, 10 + ry)
    rleg = {}
    line(rleg, 11, 15 + ry, 12, 18 + dy, "b")
    rleg[(12, 19 + dy)] = "N"
    rleg[(13, 19 + dy)] = "N"
    spear = {}
    line(spear, 9, 18 + ry, 19, 5 + ry, "T")
    spear[(20, 4 + ry)] = "M"
    spear[(20, 3 + ry)] = "M"
    spear[(21, 2 + ry)] = "W"
    spear[(18, 7 + ry)] = "R"
    spear[(19, 7 + ry)] = "R"
    rarm = arm(11, 11 + ry, 13, 13 + ry, "B")
    return compose([far, body, hd, near_r, near_f, t, head("rider", 7, 2 + ry), rleg, spear, rarm])


HORSE_WALK = [2, 1, 0, -1, -2, -1, 0, 1]
HORSE_LIFT = [0, 0, 0, 0, 0, 1, 1, 0]


def cavalry_frames():
    frames = []
    for i in range(8):
        frames.append(cavalry(0, IDLE_DY[i], 1 if i in (4, 5) else 0, (1, -1, 0, 1), (0, 0, 0, 0)))
    for i in range(8):
        a, b = i, (i + 4) % 8
        bob = 1 if i in (1, 5) else 0
        legs = (HORSE_WALK[a], HORSE_WALK[b], HORSE_WALK[b], HORSE_WALK[a])
        lifts = (HORSE_LIFT[a], HORSE_LIFT[b], HORSE_LIFT[b], HORSE_LIFT[a])
        frames.append(cavalry(bob, 0, bob, legs, lifts))
    return frames


def to_image(f, w=CW, h=CH):
    im = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    px = im.load()
    for (x, y), c in f.items():
        if 0 <= x < w and 0 <= y < h:
            px[x, y] = PAL[c] + (255,)
    return im


# ---------------------------------------------------------------- portrait
PORTRAIT = [
    "................................",
    "...........jjjjjjjjj............",
    ".........jjJJJJJJJJJjj..........",
    "........jJJJJJooJJJJJJj.........",
    ".......jJJJJJooooJJJJJJj........",
    ".......jJJJJJJooJJJJJJJj........",
    "......jJJJJJJJJJJJJJJJJJj.......",
    "......jJJJJJJJJJJJJJJJJJj.......",
    "......jjjjjjjjjjjjjjjjjjjj......",
    "......GGGGGGGGGGGGGGGGGGGGJj....",
    ".......fFFFFFFFFFFFFFFFFf.JJj...",
    ".......fFFFFFFFFFFFFFFFFf..JJ...",
    ".......fNNNNFFFFFFFNNNNFf...Jj..",
    "......ffFFNNNFFFFFNNNFFFFf...J..",
    "......fFFKWWKFFFFFKWWKFFFf......",
    "......fFFFKKFFFFFFFKKFFFFf......",
    "......ffFFFFFFfFFFFFFFFFFf......",
    ".......fFFFFFFfFFFFFFFFFf.......",
    "........fFFFFfffFFFFFFFff.......",
    ".......fFNNNNNNNNNNNNNNFf.......",
    "........NNNNNNrrrNNNNNNN........",
    "........NNNNNNNNNNNNNNNN........",
    "........NNNnNNNNnNNNnNNN........",
    ".........NNnNNNNnNNNnNN.........",
    ".........NNNnNNNnNNnNNN.........",
    "...jjJ...NNNnNNnNNNnNN...JJJ....",
    ".jjjJJJG..NNNnNNnNnNN..GJJJJJ...",
    "jjjjJJJJG.NNNnNNnNNNN.GJJJJJJJ..",
    "jjjjJJJJJGNNNNnNnNNNNGJJJJJJJJJ.",
    "jjjjJJJJJJGNNNNnNNNNGJJJJJJJJJJ.",
    "jjjjJJJJJJJGNNNNNNNGJJJJJJJJJJJJ",
    "jjjjJJJJJJJJGNNNNNGJJJJJJJJJJJJJ",
    "jjjjJJJJJJJJJGNNNGJJJJJJJJJJJJJJ",
    "jjjjJJJJJJJJJJGNGJJJJJJJJJJJJJJJ",
    "jjjjJJJJJJJJJJJGJJJJJJJJJJJJJJJJ",
    "jjjjJJJJJJJJJJJJJJJJJJJJJJJJJJJJ",
]


def portrait():
    W, H = 32, 36
    p = {}
    put(p, [r.ljust(W, ".")[:W] for r in PORTRAIT], 0, 0)
    f = compose([p])
    f = {q: c for q, c in f.items() if 0 <= q[0] < W and 0 <= q[1] < H}
    return to_image(f, W, H)


# ---------------------------------------------------------------- terrain
def hsh(*v):
    h = 2166136261
    for x in v:
        h = ((h ^ (x & 0xFFFFFFFF)) * 16777619) & 0xFFFFFFFF
    h ^= h >> 13
    h = (h * 0x5BD1E995) & 0xFFFFFFFF
    return h ^ (h >> 15)


def tile_img(grid):
    im = Image.new("RGB", (TILE, TILE))
    px = im.load()
    for y in range(TILE):
        for x in range(TILE):
            px[x, y] = PAL[grid.get((x, y), "P")]
    return im


def grass_base(tx, ty, base="P"):
    g = {(x, y): base for x in range(TILE) for y in range(TILE)}
    for k in range(7):
        v = hsh(tx, ty, k)
        x, y = 2 + v % 19, 2 + (v >> 8) % 19
        g[(x, y)] = "p"
        g[(x + 2, y)] = "p"
        g[(x + 1, y + 1)] = "p"
        if k % 3 == 0:
            g[(x + 1, y - 1)] = "o"
    for k in range(4):
        v = hsh(tx, ty, k, 9)
        g[(1 + v % 22, 1 + (v >> 8) % 22)] = "q"
    if hsh(tx, ty, 77) % 5 == 0:
        v = hsh(tx, ty, 78)
        x, y = 3 + v % 17, 3 + (v >> 8) % 17
        g[(x, y)] = "Y"
        g[(x + 3, y + 2)] = "W"
    return g


def tree(g, cx, cy, r):
    blob = {}
    for y in range(cy - r, cy + r + 1):
        for x in range(cx - r, cx + r + 1):
            d = ((x - cx) ** 2 + (y - cy) ** 2) ** 0.5
            if d <= r + 0.3:
                if (x - cx) + (y - cy) < -r * 0.6:
                    blob[(x, y)] = "o"
                elif (x - cx) + (y - cy) > r * 0.5:
                    blob[(x, y)] = "j"
                else:
                    blob[(x, y)] = "J"
    trunk = {(cx, cy + r + 1): "T", (cx - 1, cy + r + 1): "T", (cx, cy + r + 2): "T", (cx - 1, cy + r + 2): "h"}
    shadow = {(x, cy + r + 3): "q" for x in range(cx - r + 1, cx + r)}
    for q in outline(trunk) | outline(blob):
        if 0 <= q[0] < TILE and 0 <= q[1] < TILE and q not in blob and q not in trunk:
            g[q] = "Q"
    for d in (shadow, trunk, blob):
        for q, c in d.items():
            if 0 <= q[0] < TILE and 0 <= q[1] < TILE:
                g[q] = c


def forest_tile(tx, ty):
    g = grass_base(tx, ty, "p")
    tree(g, 7, 6, 4)
    tree(g, 17, 9, 4)
    tree(g, 9, 15, 4)
    return g


def dirt_base(tx, ty):
    g = {(x, y): "D" for x in range(TILE) for y in range(TILE)}
    for k in range(9):
        v = hsh(tx, ty, k, 3)
        g[(1 + v % 22, 1 + (v >> 8) % 22)] = "d"
    return g


def house(g):
    roof = {}
    for y in range(4, 11):
        k = y - 4
        x0, x1 = 6 - k // 2, 17 + k // 2
        for x in range(x0, x1 + 1):
            roof[(x, y)] = "n" if y == 4 else ("m" if (x + y) % 4 else "n")
    roof[(2, 10)] = "m"
    roof[(21, 10)] = "m"
    roof[(1, 9)] = "m"
    roof[(22, 9)] = "m"
    walls = {}
    for y in range(11, 19):
        for x in range(5, 19):
            walls[(x, y)] = "w" if x < 7 else "W"
    for y in range(14, 19):
        walls[(11, y)] = "h"
        walls[(12, y)] = "h"
    for x, y in ((7, 13), (8, 13), (15, 13), (16, 13), (7, 14), (8, 14), (15, 14), (16, 14)):
        walls[(x, y)] = "N"
    for x in range(5, 19):
        walls[(x, 11)] = "r"
    for d in (walls, roof):
        for q in outline(d):
            if 0 <= q[0] < TILE and 0 <= q[1] < TILE and q not in roof:
                g[q] = "K"
        g.update({q: c for q, c in d.items() if 0 <= q[0] < TILE and 0 <= q[1] < TILE})
    for x in range(4, 21):
        if (x, 20) not in g or g[(x, 20)] in "Dd":
            g[(x, 20)] = "d"


def village_tile(tx, ty):
    g = dirt_base(tx, ty)
    if hsh(tx, ty, 5) % 4 == 0:
        well = {}
        for y in range(8, 15):
            for x in range(8, 16):
                if ((x - 11.5) / 3.8) ** 2 + ((y - 11) / 3.2) ** 2 <= 1:
                    well[(x, y)] = "m"
        for x in range(10, 14):
            well[(x, 11)] = "a"
            well[(x, 10)] = "a"
        for q in outline(well):
            g[q] = "K"
        g.update(well)
        for y in range(2, 22, 3):
            g[(2, y)] = "T"
            g[(21, y)] = "T"
    else:
        house(g)
    return g


def water_tile(tx, ty, nb):
    g = {(x, y): "A" for x in range(TILE) for y in range(TILE)}
    for k in range(5):
        v = hsh(tx, ty, k, 7)
        x, y = 2 + v % 17, 2 + (v >> 8) % 20
        for i in range(3):
            g[(x + i, y)] = "i"
        g[(x + 1, y + 1)] = "a"
    # shores: band on non-water neighbors
    for (dx, dy), land in nb.items():
        if not land:
            continue
        for t in range(TILE):
            for depth, c in ((0, "D"), (1, "d"), (2, "i")):
                if dx == -1:
                    g[(depth, t)] = c
                elif dx == 1:
                    g[(TILE - 1 - depth, t)] = c
                elif dy == -1:
                    g[(t, depth)] = c
                elif dy == 1:
                    g[(t, TILE - 1 - depth)] = c
    return g


def mountain_tile(tx, ty):
    g = grass_base(tx, ty, "p")
    m = {}
    for y in range(3, 21):
        half = (y - 3) * 10 // 17
        for x in range(12 - half, 12 + half + 1):
            m[(x, y)] = "w" if x < 12 - half // 3 else ("d" if x > 12 + half // 3 else "D")
    for y in range(3, 7):
        for x in range(10, 15):
            if (x, y) in m:
                m[(x, y)] = "W"
    for q in outline(m):
        if 0 <= q[0] < TILE and 0 <= q[1] < TILE:
            g[q] = "K"
    g.update({q: c for q, c in m.items() if 0 <= q[0] < TILE and 0 <= q[1] < TILE})
    return g


def render_map(tiles):
    h, w = len(tiles), len(tiles[0])
    im = Image.new("RGB", (w * TILE, h * TILE))

    def is_water(x, y):
        return 0 <= x < w and 0 <= y < h and tiles[y][x] == "~"

    for ty in range(h):
        for tx in range(w):
            c = tiles[ty][tx]
            if c == "f":
                g = forest_tile(tx, ty)
            elif c == "v":
                g = village_tile(tx, ty)
            elif c == "~":
                nb = {}
                for d in ((-1, 0), (1, 0), (0, -1), (0, 1)):
                    nx, ny = tx + d[0], ty + d[1]
                    nb[d] = 0 <= nx < w and 0 <= ny < h and not is_water(nx, ny)
                g = water_tile(tx, ty, nb)
            else:
                g = grass_base(tx, ty)
            im.paste(tile_img(g), (tx * TILE, ty * TILE))
    return im


# ---------------------------------------------------------------- content
def load_stage():
    t = open(DOC, encoding="utf-8").read()
    i = t.index("# 부록 B")
    j = t.index("```json", i) + 7
    k = t.index("```", j)
    return json.loads(t[j:k])["Stages"][0]


def move_range(tiles, sx, sy, mv, blocked):
    cost = {".": 1, "d": 2, "f": 3, "v": 1, "~": None}
    h, w = len(tiles), len(tiles[0])
    best = {(sx, sy): 0}
    dq = deque([(sx, sy)])
    while dq:
        x, y = dq.popleft()
        for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            nx, ny = x + dx, y + dy
            if not (0 <= nx < w and 0 <= ny < h) or (nx, ny) in blocked:
                continue
            c = cost[tiles[ny][nx]]
            if c is None:
                continue
            nc = best[(x, y)] + c
            if nc <= mv and nc < best.get((nx, ny), 99):
                best[(nx, ny)] = nc
                dq.append((nx, ny))
    return set(best)


def gray(im):
    out = im.copy()
    px = out.load()
    for y in range(out.height):
        for x in range(out.width):
            r, g, b, a = px[x, y]
            if a:
                v = int(0.3 * r + 0.59 * g + 0.11 * b)
                v = 48 + v * 150 // 255
                px[x, y] = (v, v, v + 6, 255)
    return out


def font(size, weight="Medium"):
    f = ImageFont.truetype(FONT, size)
    f.set_variation_by_name(weight)
    return f


UI_BG = (16, 22, 34)
UI_PANEL = (24, 34, 52)
UI_PANEL2 = (32, 46, 70)
UI_GOLD = (214, 170, 82)
UI_GOLD_D = (122, 88, 40)
UI_TEXT = (240, 232, 210)
UI_DIM = (160, 166, 180)


def panel(d, box, fill=UI_PANEL):
    x0, y0, x1, y1 = box
    d.rectangle(box, fill=fill)
    d.rectangle(box, outline=UI_GOLD_D, width=4)
    d.rectangle((x0 + 2, y0 + 2, x1 - 2, y1 - 2), outline=UI_GOLD, width=2)
    for cx, cy in ((x0, y0), (x1 - 7, y0), (x0, y1 - 7), (x1 - 7, y1 - 7)):
        d.rectangle((cx, cy, cx + 7, cy + 7), fill=UI_GOLD)


def bar(d, x, y, w, label, cur, mx, col, f_lab, f_num):
    d.text((x, y - 2), label, font=f_lab, fill=UI_DIM)
    bx = x + 44
    d.rectangle((bx, y + 4, bx + w, y + 16), fill=(10, 14, 22), outline=(70, 80, 100))
    fw = int(w * cur / mx) if mx else 0
    if fw:
        d.rectangle((bx + 1, y + 5, bx + fw - 1, y + 15), fill=col)
        d.rectangle((bx + 1, y + 5, bx + fw - 1, y + 7), fill=tuple(min(255, c + 50) for c in col))
    d.text((bx + w + 10, y - 3), f"{cur} / {mx}", font=f_num, fill=UI_TEXT)


def build_screen(stage, sprites, port):
    tiles = stage["Tiles"]
    W, H = 1280, 850
    scr = Image.new("RGB", (W, H), UI_BG)
    d = ImageDraw.Draw(scr)
    mapw, maph = len(tiles[0]) * TILE, len(tiles) * TILE
    mx0, my0 = (W - mapw * SCALE) // 2, 46

    base = render_map(tiles).convert("RGBA")
    over = Image.new("RGBA", base.size, (0, 0, 0, 0))
    od = ImageDraw.Draw(over)

    allies = [("유비", "infantry-ally", 2, 5, True), ("관우", "cavalry-ally", 2, 6, False),
              ("장비", "cavalry-ally", 2, 7, False)]
    enemies = []
    for e in stage["Enemies"]:
        name = e["Officer"]
        x, y = e["X"], e["Y"]
        if name == "B01_troop_3":
            x = 8  # mockup: advanced one tile so the attack preview is reachable
        enemies.append(("황건적" if name.startswith("B01") else name, x, y))
    occupied = {(a[2], a[3]) for a in allies} | {(e[1], e[2]) for e in enemies}
    reach = move_range(tiles, 2, 6, 5, {(e[1], e[2]) for e in enemies})
    for (x, y) in reach:
        if (x, y) in occupied and (x, y) != (2, 6):
            continue
        od.rectangle((x * TILE, y * TILE, x * TILE + TILE - 1, y * TILE + TILE - 1), fill=(70, 130, 255, 92))
        od.rectangle((x * TILE, y * TILE, x * TILE + TILE - 1, y * TILE + TILE - 1), outline=(150, 200, 255, 150))
    tgt = (8, 6)
    od.rectangle((tgt[0] * TILE, tgt[1] * TILE, tgt[0] * TILE + TILE - 1, tgt[1] * TILE + TILE - 1),
                 fill=(255, 60, 50, 110), outline=(255, 120, 100, 255))
    # chosen destination path ghost
    for x in range(3, 8):
        od.rectangle((x * TILE + 10, 6 * TILE + 10, x * TILE + 13, 6 * TILE + 13), fill=(255, 240, 180, 220))
    # grid
    for x in range(0, mapw, TILE):
        od.line((x, 0, x, maph), fill=(0, 0, 0, 34))
    for y in range(0, maph, TILE):
        od.line((0, y, mapw, y), fill=(0, 0, 0, 34))
    base = Image.alpha_composite(base, over)

    def place(img, tx, ty, flip=False):
        if flip:
            img = img.transpose(Image.FLIP_LEFT_RIGHT)
        base.alpha_composite(img, (tx * TILE + (TILE // 2 - AX), ty * TILE + (TILE - 2) - AY))

    def flag(tx, ty, col):
        fx, fy = tx * TILE + 1, ty * TILE + 1
        fd = ImageDraw.Draw(base)
        fd.line((fx, fy, fx, fy + 8), fill=PAL["K"])
        fd.polygon([(fx + 1, fy), (fx + 6, fy + 2), (fx + 1, fy + 4)], fill=col)

    for ty in range(len(tiles)):
        for name, key, x, y, done in allies:
            if y == ty:
                img = sprites[key][0]
                place(gray(img) if done else img, x, y)
                flag(x, y, (90, 150, 240))
        for name, x, y in enemies:
            if y == ty:
                place(sprites["bandit-enemy"][2], x, y, flip=True)
                flag(x, y, (220, 60, 50))
    # selection cursor around 관우
    cd = ImageDraw.Draw(base)
    sx, sy = 2 * TILE, 6 * TILE
    for (ax, ay, bx, by) in ((0, 0, 5, 0), (0, 0, 0, 5), (23, 0, 18, 0), (23, 0, 23, 5),
                             (0, 23, 5, 23), (0, 23, 0, 18), (23, 23, 18, 23), (23, 23, 23, 18)):
        cd.line((sx + ax, sy + ay, sx + bx, sy + by), fill=(255, 228, 120))

    big = base.resize((mapw * SCALE, maph * SCALE), Image.NEAREST)
    scr.paste(big, (mx0, my0))
    d.rectangle((mx0 - 6, my0 - 6, mx0 + mapw * SCALE + 5, my0 + maph * SCALE + 5), outline=UI_GOLD_D, width=4)
    d.rectangle((mx0 - 3, my0 - 3, mx0 + mapw * SCALE + 2, my0 + maph * SCALE + 2), outline=UI_GOLD, width=1)

    f_s, f_m, f_l, f_xl = font(14), font(16), font(20, "Bold"), font(28, "Bold")
    # HP mini bars and names under units
    def unit_hud(tx, ty, ratio, col):
        x = mx0 + tx * TILE * SCALE + 6
        y = my0 + (ty + 1) * TILE * SCALE - 6
        d.rectangle((x, y, x + 36, y + 4), fill=(10, 12, 18))
        d.rectangle((x + 1, y + 1, x + 1 + int(34 * ratio), y + 3), fill=col)
    for name, key, x, y, done in allies:
        unit_hud(x, y, 1.0 if name != "유비" else 0.82, (110, 220, 120))
    for i, (name, x, y) in enumerate(enemies):
        unit_hud(x, y, [1.0, 0.9, 1.0, 0.7, 1.0, 1.0, 1.0][i % 7], (240, 110, 90))
    for name, x, y in enemies:
        if not name.startswith("황건"):
            tx = mx0 + x * TILE * SCALE + TILE
            ty = my0 + y * TILE * SCALE - 14
            w = d.textlength(name, font=f_s)
            d.rectangle((tx - w / 2 - 5, ty - 1, tx + w / 2 + 5, ty + 17), fill=(90, 24, 24), outline=(220, 120, 90))
            d.text((tx - w / 2, ty - 2), name, font=f_s, fill=UI_TEXT)
    # floating damage number over 황건적 at (10,4) (예: 이전 공격 결과)
    dx_, dy_ = mx0 + 8 * TILE * SCALE + 4, my0 + 6 * TILE * SCALE - 30
    f_dmg = font(26, "Black")
    d.text((dx_, dy_), "-86", font=f_dmg, fill=(255, 236, 140), stroke_width=3, stroke_fill=(70, 20, 10))

    # top bar
    panel(d, (12, 4, W - 12, 38), UI_PANEL)
    d.text((32, 7), "제1전  황건적의 난", font=f_l, fill=UI_GOLD)
    msg = "1턴 · 아군 진영"
    tw = d.textlength(msg, font=f_l)
    d.rectangle((W / 2 - tw / 2 - 22, 8, W / 2 + tw / 2 + 22, 34), fill=(40, 80, 150), outline=(140, 190, 255))
    d.text((W / 2 - tw / 2, 7), msg, font=f_l, fill=UI_TEXT)
    obj = "목표 : 장각 격퇴"
    d.text((W - 32 - d.textlength(obj, font=f_l), 7), obj, font=f_l, fill=UI_TEXT)

    # left sidebar: terrain card + swatches
    lx0, lx1 = 12, mx0 - 18
    panel(d, (lx0, 46, lx1, 330))
    d.text((lx0 + 16, 64), "지형", font=f_l, fill=UI_GOLD)
    sw = render_map(["."]).resize((TILE * 3, TILE * 3), Image.NEAREST)
    scr.paste(sw, (lx0 + 16, 98))
    d.text((lx0 + 100, 100), "초원", font=f_l, fill=UI_TEXT)
    d.text((lx0 + 100, 128), "이동 1", font=f_m, fill=UI_DIM)
    d.text((lx0 + 16, 176), "기병 공격 110%", font=f_m, fill=(150, 220, 150))
    d.text((lx0 + 16, 200), "보병·궁병 100%", font=f_m, fill=UI_DIM)
    d.text((lx0 + 16, 236), "지형 견본", font=f_s, fill=UI_DIM)
    swatches = [grass_base(0, 0), forest_tile(3, 1), village_tile(1, 1),
                water_tile(2, 2, {(0, -1): True}), mountain_tile(5, 5)]
    for i, g in enumerate(swatches):
        scr.paste(tile_img(g), (lx0 + 16 + i * 30, 262))
    labels = "초 숲 마 물 산".split()
    for i, t in enumerate(labels):
        d.text((lx0 + 22 + i * 30, 290), t, font=f_s, fill=UI_DIM)

    # right sidebar: forces + log
    rx0, rx1 = mx0 + mapw * SCALE + 18, W - 12
    panel(d, (rx0, 46, rx1, 420))
    d.text((rx0 + 16, 64), "부대", font=f_l, fill=UI_GOLD)
    rows = [("유비", "경보병", "행동 완료", UI_DIM), ("관우", "경기병", "선택 중", (255, 228, 120)),
            ("장비", "경기병", "대기", UI_TEXT)]
    for i, (n, c, st, col) in enumerate(rows):
        y = 96 + i * 40
        d.rectangle((rx0 + 14, y + 4, rx0 + 22, y + 20), fill=(90, 150, 240))
        d.text((rx0 + 30, y), n, font=f_m, fill=col)
        d.text((rx0 + 70, y + 2), c, font=f_s, fill=UI_DIM)
        d.text((rx0 + 30, y + 20), st, font=f_s, fill=col)
    d.line((rx0 + 14, 222, rx1 - 14, 222), fill=UI_GOLD_D)
    d.text((rx0 + 16, 228), "적군 7부대", font=f_m, fill=(240, 130, 110))
    d.text((rx0 + 16, 252), "장각 · 정원지 · 등무", font=f_s, fill=UI_DIM)
    d.text((rx0 + 16, 272), "황건적 ×4", font=f_s, fill=UI_DIM)
    d.line((rx0 + 14, 296, rx1 - 14, 296), fill=UI_GOLD_D)
    d.text((rx0 + 16, 304), "전투 기록", font=f_m, fill=UI_GOLD)
    for i, t in enumerate(["유비 → 이동 (2,5)", "유비 · 대기", "관우 선택"]):
        d.text((rx0 + 16, 332 + i * 24), t, font=f_s, fill=UI_DIM)

    # floating action menu (top-left of map)
    ax0, ay0 = mx0 + 14, my0 + 14
    items = ["이동", "공격", "병법", "아이템", "대기", "일기토"]
    panel(d, (ax0, ay0, ax0 + 128, ay0 + 22 + len(items) * 30), (20, 28, 46))
    for i, it in enumerate(items):
        y = ay0 + 12 + i * 30
        dis = it == "일기토"
        if it == "공격":
            d.rectangle((ax0 + 10, y, ax0 + 118, y + 27), fill=(150, 46, 40))
            d.polygon([(ax0 + 16, y + 7), (ax0 + 24, y + 13), (ax0 + 16, y + 19)], fill=UI_GOLD)
        d.text((ax0 + 32, y + 1), it, font=f_m, fill=(110, 116, 130) if dis else UI_TEXT)

    # bottom info bar
    by0, by1 = 728, H - 6
    panel(d, (12, by0, W - 12, by1))
    pw, ph = port.size
    ps = 3
    pimg = port.resize((pw * ps, ph * ps), Image.NEAREST)
    crop_h = min(ph * ps, (by1 - 8) - (by0 + 8))
    d.rectangle((25, by0 + 7, 28 + pw * ps + 2, by0 + 8 + crop_h), fill=(14, 24, 40), outline=UI_GOLD)
    pc = pimg.crop((0, 0, pw * ps, crop_h))
    scr.paste(pc, (27, by0 + 8), pc)
    nx = 28 + pw * ps + 22
    d.text((nx, by0 + 10), "관우", font=f_xl, fill=UI_TEXT)
    d.text((nx + 80, by0 + 22), "운장", font=f_s, fill=UI_DIM)
    d.text((nx, by0 + 52), "경기병 · Lv 1", font=f_m, fill=UI_GOLD)
    d.text((nx, by0 + 78), "특성  호걸 · 반격술", font=f_s, fill=UI_DIM)
    bx = nx + 170
    bar(d, bx, by0 + 16, 190, "병력", 420, 420, (80, 200, 110), f_s, f_m)
    bar(d, bx, by0 + 46, 190, "병법", 18, 36, (80, 140, 240), f_s, f_m)
    bar(d, bx, by0 + 76, 190, "경험", 0, 100, (230, 180, 70), f_s, f_m)
    sx0 = bx + 340
    d.line((sx0 - 14, by0 + 14, sx0 - 14, by1 - 14), fill=UI_GOLD_D)
    stats = [("무력", 98), ("통솔", 95), ("지력", 79), ("순발", 83), ("이동", 5), ("사거리", 1)]
    for i, (k, v) in enumerate(stats):
        cx, cy = sx0 + (i % 2) * 96, by0 + 14 + (i // 2) * 30
        d.text((cx, cy), k, font=f_s, fill=UI_DIM)
        d.text((cx + 50, cy - 2), str(v), font=f_m, fill=UI_TEXT)
    px0 = sx0 + 210
    d.line((px0 - 14, by0 + 14, px0 - 14, by1 - 14), fill=UI_GOLD_D)
    d.text((px0, by0 + 12), "공격 예상 → 황건적", font=f_m, fill=(255, 160, 140))
    d.text((px0, by0 + 42), "피해 86   명중 92%   크리 8%", font=f_m, fill=UI_TEXT)
    d.text((px0, by0 + 72), "반격 없음 · 대상 병력 120 → 34", font=f_s, fill=UI_DIM)
    return scr


# ---------------------------------------------------------------- checks
def check(sheet, rows_frames):
    px = sheet.load()
    problems = []
    alphas = set()
    for row, (sid, n) in enumerate(rows_frames):
        for fi in range(n):
            x0, y0 = fi * CW, row * CH
            lowest, edge = -1, False
            for y in range(CH):
                for x in range(CW):
                    a = px[x0 + x, y0 + y][3]
                    alphas.add(a)
                    if a:
                        lowest = max(lowest, y)
                        if x in (0, CW - 1) or y in (0, CH - 1):
                            edge = True
            if lowest != AY:
                problems.append(f"{sid}[{fi}] lowest row {lowest} != anchor {AY}")
            if edge:
                problems.append(f"{sid}[{fi}] touches cell edge")
    if not alphas <= {0, 255}:
        problems.append(f"alpha values {sorted(alphas)}")
    return problems


def main():
    os.makedirs(OUT, exist_ok=True)
    rows = [
        ("infantry-ally", infantry_frames("ally"), [("idle", 0, 140), ("walk", 8, 100)]),
        ("cavalry-ally", cavalry_frames(), [("idle", 0, 150), ("walk", 8, 100)]),
        ("bandit-enemy", infantry_frames("bandit", walk=False), [("idle", 0, 140)]),
    ]
    cols = max(len(r[1]) for r in rows)
    sheet = Image.new("RGBA", (cols * CW, len(rows) * CH), (0, 0, 0, 0))
    meta = {"cell": [CW, CH], "anchor": [AX, AY], "displayScale": SCALE, "sprites": []}
    sprites = {}
    for r, (sid, frames, anims) in enumerate(rows):
        imgs = [to_image(f) for f in frames]
        sprites[sid] = imgs
        for i, im in enumerate(imgs):
            sheet.paste(im, (i * CW, r * CH))
        meta["sprites"].append({"id": sid, "row": r, "anims": [
            {"name": n, "start": s, "frames": 8, "ms": ms} for n, s, ms in anims]})
    sheet.save(os.path.join(OUT, "sheet.png"))
    sheet.resize((sheet.width * 4, sheet.height * 4), Image.NEAREST).save(os.path.join(OUT, "sheet-x4.png"))
    with open(os.path.join(OUT, "sheet.json"), "w") as f:
        json.dump(meta, f, indent=1)

    port = portrait()
    port.save(os.path.join(OUT, "portrait.png"))
    with open(os.path.join(OUT, "portrait.json"), "w") as f:
        json.dump({"size": list(port.size), "displayScale": 3}, f)

    scr = build_screen(load_stage(), sprites, port)
    assert scr.size == (1280, 850)
    scr.save(os.path.join(OUT, "screen.png"))

    probs = check(sheet, [(sid, len(fr)) for sid, fr, _ in rows])
    used = {c for _, fr, _ in rows for f in fr for c in f.values()}
    print("unit colors used:", len(used), "| total palette:", len(PAL))
    print("checks:", "OK" if not probs else "\n  " + "\n  ".join(probs))


if __name__ == "__main__":
    main()
