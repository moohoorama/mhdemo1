#!/usr/bin/env python3
"""Candidate D: ink-wash (수묵) UI + hi-bit pixel art.

Run from ver2/:  python3 art-candidates/D/gen.py
Deterministic; uses only Pillow.
"""
import json
import os

from PIL import Image, ImageDraw, ImageFont

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "out")
FONT = "/Users/rosemily/orca/mhdemo1/ver1/assets/fonts/NotoSansKR.ttf"
DESIGN = os.path.join(HERE, "..", "..", "srpg_full_design.md")

CELL = 64
ANCHOR = (32, 57)

# ---------------------------------------------------------------- palette
INK = (34, 30, 28)
PAPER = [(214, 201, 170), (226, 214, 186), (236, 227, 204), (244, 238, 220)]
SEAL = [(110, 30, 24), (160, 44, 34), (190, 64, 46)]
INDIGO_SEAL = [(30, 40, 70), (46, 62, 104), (70, 92, 140)]

# ramps: [line, dark, mid, light, hi]
BLUE = [(26, 32, 54), (44, 60, 98), (62, 88, 134), (92, 124, 168), (146, 172, 200)]
STEEL = [(44, 46, 52), (96, 102, 110), (150, 156, 164), (200, 204, 208), (238, 240, 240)]
BRONZE = [(62, 44, 22), (120, 86, 40), (166, 124, 58), (206, 166, 88), (236, 212, 146)]
SKIN = [(70, 42, 30), (150, 98, 70), (192, 136, 98), (222, 172, 132), (238, 200, 162)]
IVORY = [(84, 74, 60), (160, 146, 120), (204, 192, 166), (230, 222, 200), (246, 242, 228)]
LEATHER = [(36, 24, 18), (64, 44, 32), (94, 64, 44), (128, 90, 62), (160, 120, 84)]
TASSEL = [(76, 24, 20), (132, 38, 30), (176, 58, 44), (204, 92, 66), (226, 132, 96)]
HAIR = [(18, 16, 16), (30, 28, 28), (46, 42, 42), (66, 60, 58), (92, 86, 82)]
TUNIC = [(60, 26, 20), (104, 44, 32), (142, 66, 46), (176, 96, 66), (204, 130, 94)]
YELLOW = [(84, 62, 20), (156, 120, 40), (204, 166, 62), (228, 198, 96), (244, 226, 150)]
HORSE = [(40, 24, 16), (78, 48, 30), (114, 72, 44), (150, 102, 62), (186, 140, 94)]
MANE = [(18, 14, 12), (34, 26, 22), (52, 40, 32), (76, 60, 48), (100, 82, 66)]
WOOD = [(40, 28, 18), (76, 54, 34), (110, 80, 50), (142, 108, 70), (172, 138, 96)]
CLOTH_RED = [(70, 22, 20), (120, 36, 30), (156, 52, 40), (186, 80, 58), (214, 120, 88)]
FACE_RED = [(70, 24, 20), (118, 46, 34), (152, 64, 46), (182, 92, 66), (206, 124, 92)]
ROBE_GREEN = [(18, 36, 28), (34, 64, 48), (52, 90, 64), (78, 118, 84), (116, 150, 110)]
SCARF_GREEN = [(20, 44, 34), (40, 80, 58), (60, 108, 76), (90, 140, 98), (132, 172, 128)]

GRASS = [(84, 102, 62), (98, 116, 70), (110, 128, 78), (124, 140, 88), (146, 156, 104)]
FLOOR = [(70, 88, 56), (82, 100, 62), (94, 112, 68), (108, 124, 76)]
LEAF = [(22, 36, 28), (38, 62, 44), (56, 86, 56), (80, 112, 66), (114, 140, 84)]
EARTH = [(108, 90, 64), (134, 114, 82), (158, 138, 102), (182, 164, 128)]
WATER = [(40, 64, 78), (54, 86, 100), (72, 108, 118), (104, 140, 144), (164, 190, 184)]
ROOF = [(36, 38, 44), (60, 62, 70), (84, 88, 96), (114, 118, 124), (150, 154, 158)]
WALL = [(120, 108, 88), (176, 162, 134), (206, 194, 166), (228, 220, 196), (242, 236, 220)]


def h(*args):
    n = 2166136261
    for a in args:
        n = ((n ^ (a & 0xFFFFFFFF)) * 16777619) & 0xFFFFFFFF
    n ^= n >> 13
    n = (n * 1274126177) & 0xFFFFFFFF
    n ^= n >> 16
    return n / 4294967296.0


def vnoise(x, y, cell, period, seed):
    """Value noise periodic in `period` pixels."""
    n = period // cell
    gx, gy = x // cell, y // cell
    fx, fy = (x % cell) / cell, (y % cell) / cell
    fx = fx * fx * (3 - 2 * fx)
    fy = fy * fy * (3 - 2 * fy)

    def g(i, j):
        return h(i % n, j % n, seed)

    a = g(gx, gy) * (1 - fx) + g(gx + 1, gy) * fx
    b = g(gx, gy + 1) * (1 - fx) + g(gx + 1, gy + 1) * fx
    return a * (1 - fy) + b * fy


# ---------------------------------------------------------------- part renderer
class Part:
    def __init__(self, ramp, draw, deco=None, hi=True, band=False):
        self.ramp, self.draw, self.deco, self.hi, self.band = ramp, draw, deco or [], hi, band


def mask_pts(draw_fn, dx, dy):
    m = Image.new("L", (CELL, CELL), 0)
    draw_fn(ImageDraw.Draw(m), dx, dy)
    px = m.load()
    return {(x, y) for y in range(CELL) for x in range(CELL) if px[x, y]}


def P(pts, dx, dy):
    return [(x + dx, y + dy) for x, y in pts]


def B(box, dx, dy):
    return [box[0] + dx, box[1] + dy, box[2] + dx, box[3] + dy]


def compose(layers):
    """layers: list of (Part, dx, dy). Returns RGBA image with alpha 0/255."""
    canvas = {}
    for part, dx, dy in layers:
        pts = mask_pts(part.draw, dx, dy)
        if not pts:
            continue
        r = part.ramp
        xs = [p[0] for p in pts]
        minx, maxx = min(xs), max(xs)
        w = maxx - minx + 1
        for x, y in pts:
            for n in ((x + 1, y), (x - 1, y), (x, y + 1), (x, y - 1)):
                if n not in pts:
                    canvas[n] = r[0]
        for x, y in pts:
            ro = (x + 1, y) not in pts
            do = (x, y + 1) not in pts
            lo = (x - 1, y) not in pts
            uo = (x, y - 1) not in pts
            if ro or do:
                c = r[1]
            elif part.band and w >= 7 and ((x + 2, y) not in pts or (x, y + 2) not in pts):
                c = r[1]
            elif lo or uo:
                c = r[4] if (part.hi and uo and x < minx + w * 0.45) else r[3]
            else:
                c = r[2]
            canvas[(x, y)] = c
        for x, y, c in part.deco:
            canvas[(x + dx, y + dy)] = c
    img = Image.new("RGBA", (CELL, CELL), (0, 0, 0, 0))
    px = img.load()
    for (x, y), c in canvas.items():
        if 0 <= x < CELL and 0 <= y < CELL:
            px[x, y] = c + (255,)
    # outer silhouette in ink
    edge = []
    for (x, y) in canvas:
        if not (0 <= x < CELL and 0 <= y < CELL):
            continue
        for n in ((x + 1, y), (x - 1, y), (x, y + 1), (x, y - 1)):
            if n not in canvas:
                edge.append((x, y))
                break
    for x, y in edge:
        px[x, y] = INK + (255,)
    return img


# ---------------------------------------------------------------- infantry / bandit
IDLE_BREATH = [0, 0, 0, 1, 1, 1, 0, 0]
# walk: (back leg dx, back lift, front leg dx, front lift, body dy, arm swing)
WALK = [
    (-3, 0, 3, 0, 0, 1),    # contact
    (-2, 0, 2, 0, 1, 1),    # down
    (-1, 0, 0, 2, 0, 0),    # passing (front foot lifted)
    (1, 0, -2, 1, -1, -1),  # up
    (3, 0, -3, 0, 0, -1),   # opposite contact
    (2, 0, -2, 0, 1, -1),   # down
    (0, 2, -1, 0, 0, 0),    # passing (back foot lifted)
    (-2, 1, 1, 0, -1, 1),   # up
]


def leg_part(x0, ramp_pants, ramp_boot, dx, lift):
    def pants(d, ox, oy):
        d.rectangle(B((x0, 44, x0 + 4, 51), ox + dx, oy - lift), fill=255)

    def boot(d, ox, oy):
        d.rectangle(B((x0, 52, x0 + 5, 56), ox + dx, oy - lift), fill=255)
        d.point((x0 + 5 + ox + dx, 52 + oy - lift), fill=0)

    return [(Part(ramp_pants, pants, hi=False), 0, 0), (Part(ramp_boot, boot, hi=False), 0, 0)]


def infantry_frame(breath=0, walk=None, bandit=False):
    if walk:
        bdx, blift, fdx, flift, body, swing = walk
    else:
        bdx = blift = fdx = flift = swing = 0
        body = breath
    layers = []
    boots = LEATHER if not bandit else WOOD
    layers += leg_part(25, IVORY, boots, bdx, blift)
    layers += leg_part(33, IVORY, boots, fdx, flift)
    oy = body

    if not bandit:
        def coat(d, ox, y):
            d.polygon(P([(24, 31), (39, 31), (41, 46), (22, 46)], ox, y), fill=255)
        lam = []
        for yy in range(34, 46, 3):
            for xx in range(25 + (yy % 2), 40, 2):
                lam.append((xx, yy, BLUE[1]))
        for xx in range(23, 41):
            lam.append((xx, 39, BRONZE[2]))
            lam.append((xx, 40, BRONZE[1]))
        lam.append((31, 39, BRONZE[4]))
        layers.append((Part(BLUE, coat, lam, band=True), 0, oy))

        def pauld(d, ox, y):
            d.ellipse(B((21, 29, 28, 35), ox, y), fill=255)
            d.ellipse(B((35, 29, 42, 35), ox, y), fill=255)
        layers.append((Part(BLUE, pauld), 0, oy))
    else:
        def tunic(d, ox, y):
            d.polygon(P([(25, 31), (38, 31), (40, 47), (23, 47)], ox, y), fill=255)
        deco = [(xx, 40, YELLOW[1]) for xx in range(24, 40)] + [(xx, 41, LEATHER[2]) for xx in range(24, 40)]
        deco += [(31, yy, TUNIC[1]) for yy in range(32, 39)]
        layers.append((Part(TUNIC, tunic, deco, band=True), 0, oy))

    # head
    def face(d, ox, y):
        d.ellipse(B((27, 19, 37, 30), ox, y), fill=255)
    eyes = [(31, 25, INK), (31, 26, INK), (35, 25, INK), (35, 26, INK),
            (30, 24, SKIN[1]), (31, 24, SKIN[1]), (35, 24, SKIN[1]), (36, 24, SKIN[1]),
            (34, 28, SKIN[1]), (33, 28, SKIN[1])]
    layers.append((Part(SKIN, face, eyes), 0, oy))

    if not bandit:
        def helm(d, ox, y):
            d.chord(B((25, 14, 39, 30), ox, y), 180, 360, fill=255)
            d.rectangle(B((26, 21, 38, 22), ox, y), fill=255)
        rim = [(xx, 22, BRONZE[2]) for xx in range(26, 39)] + [(xx, 21, BRONZE[3]) for xx in range(27, 38)]
        rim += [(32, yy, BRONZE[2]) for yy in range(15, 21)]
        layers.append((Part(BLUE, helm, rim), 0, oy))

        def plume(d, ox, y):
            d.polygon(P([(31, 15), (33, 15), (34, 10), (32, 7), (30, 10)], ox, y), fill=255)
        layers.append((Part(TASSEL, plume), 0, oy))
    else:
        def hair(d, ox, y):
            d.chord(B((26, 15, 38, 29), ox, y), 180, 360, fill=255)
            d.ellipse(B((30, 11, 34, 16), ox, y), fill=255)
        layers.append((Part(HAIR, hair), 0, oy))

        def band(d, ox, y):
            d.rectangle(B((26, 20, 38, 22), ox, y), fill=255)
            d.polygon(P([(26, 21), (22, 24), (21, 27), (24, 25), (26, 23)], ox, y), fill=255)
        layers.append((Part(YELLOW, band), 0, oy))

    if not bandit:
        # sword arm (front, screen right)
        def arm(d, ox, y):
            d.rectangle(B((39, 32, 42, 40), ox + swing, y), fill=255)
        layers.append((Part(BLUE, arm), 0, oy))

        def blade(d, ox, y):
            s = swing
            d.polygon(P([(42, 43), (44, 42), (49, 48), (52, 53), (50, 54), (46, 49)], ox + s, y), fill=255)
        layers.append((Part(STEEL, blade), 0, oy))

        def hand(d, ox, y):
            d.rectangle(B((40, 40, 43, 43), ox + swing, y), fill=255)
            d.rectangle(B((39, 43, 41, 44), ox + swing, y), fill=255)
        layers.append((Part(SKIN, hand), 0, oy))

        # shield (back arm, screen left)
        def shield(d, ox, y):
            d.ellipse(B((16, 31, 28, 44), ox - swing, y), fill=255)
        boss = [(21, 37, BRONZE[0]), (22, 37, BRONZE[4]), (21, 38, BRONZE[1]), (22, 38, BRONZE[2])]
        boss += [(19, 34, BRONZE[4]), (20, 33, BRONZE[4])]
        layers.append((Part(BRONZE, shield, [(x - swing, y, c) for x, y, c in boss], band=True), 0, oy))
    else:
        def arm(d, ox, y):
            d.rectangle(B((38, 32, 41, 39), ox + swing, y), fill=255)
        layers.append((Part(TUNIC, arm), 0, oy))

        def shaft(d, ox, y):
            d.rectangle(B((43, 14, 44, 55), ox + swing, y), fill=255)
        layers.append((Part(WOOD, shaft, hi=False), 0, oy))

        def tip(d, ox, y):
            d.polygon(P([(43, 13), (44, 13), (46, 9), (43.5, 4), (41, 9)], ox + swing, y), fill=255)
        layers.append((Part(STEEL, tip), 0, oy))

        def ribbon(d, ox, y):
            d.polygon(P([(42, 14), (45, 14), (46, 18), (44, 17), (42, 19)], ox + swing, y), fill=255)
        layers.append((Part(YELLOW, ribbon), 0, oy))

        def hand(d, ox, y):
            d.rectangle(B((41, 38, 45, 41), ox + swing, y), fill=255)
        layers.append((Part(SKIN, hand), 0, oy))

        def backarm(d, ox, y):
            d.rectangle(B((21, 32, 24, 41), ox - swing, y), fill=255)
        layers.append((Part(TUNIC, backarm), 0, oy))

        def fist(d, ox, y):
            d.rectangle(B((21, 41, 24, 43), ox - swing, y), fill=255)
        layers.append((Part(SKIN, fist), 0, oy))
    return compose(layers)


# ---------------------------------------------------------------- cavalry
LEG_PHASE = {"LH": 0, "LF": 2, "RH": 4, "RF": 6}
STANCE = [3, 2, 0, -1, -3]
SWING = [(-1, 2), (1, 3), (2, 2)]


def hoof_offset(p):
    if p < 5:
        return STANCE[p], 0
    return SWING[p - 5]


def horse_leg(hipx, ramp, dx, lift, body, hind):
    hip = (hipx, 39 + body)
    hoof = (hipx + dx, 54 - lift)
    bend = (-2 if hind else 1) + (2 if lift else 0)
    knee = ((hip[0] + hoof[0]) // 2 + bend, (hip[1] + hoof[1]) // 2 + 1 - (1 if lift else 0))

    def thigh(d, ox, oy):
        d.polygon([(hip[0] - 3, hip[1] - 2), (hip[0] + 3, hip[1] - 2), (knee[0] + 1, knee[1]), (knee[0] - 1, knee[1])], fill=255)

    def leg(d, ox, oy):
        d.line([knee, hoof], fill=255, width=2)

    def hoofp(d, ox, oy):
        d.rectangle([hoof[0] - 1, hoof[1] + 1, hoof[0] + 2, hoof[1] + 2], fill=255)

    return [(Part(ramp, thigh, hi=False), 0, 0), (Part(ramp, leg, hi=False), 0, 0),
            (Part(MANE, hoofp, hi=False), 0, 0)]


def cavalry_frame(f, walking):
    layers = []
    if walking:
        body = [0, 0, -1, 0, 0, 0, -1, 0][f]
        offs = {k: hoof_offset((f + v) % 8) for k, v in LEG_PHASE.items()}
        neck, tail, breath = body, 0, 0
    else:
        body = 0
        offs = {k: (0, 0) for k in LEG_PHASE}
        neck = [0, 0, 0, 1, 1, 1, 0, 0][f]
        tail = [0, 0, 1, 1, 1, 0, 0, 0][f]
        breath = [0, 0, 0, 0, 1, 1, 1, 0][f]
    far = [HORSE[0], HORSE[0], HORSE[1], HORSE[2], HORSE[3]]
    layers += horse_leg(19, far, *offs["RH"], body, True)
    layers += horse_leg(41, far, *offs["RF"], body, False)

    layers += horse_leg(16, HORSE, *offs["LH"], body, True)
    layers += horse_leg(44, HORSE, *offs["LF"], body, False)

    def tailp(d, ox, oy):
        d.polygon(P([(15, 33), (10, 35), (8 + tail, 46), (11 + tail, 48), (13, 41), (16, 37)], ox, oy), fill=255)
    layers.append((Part(MANE, tailp), 0, body))

    def bodyp(d, ox, oy):
        d.ellipse(B((11, 29, 48, 44), ox, oy), fill=255)
        d.ellipse(B((10, 29, 24, 43), ox, oy), fill=255)
    layers.append((Part(HORSE, bodyp, band=True), 0, body))


    def neckp(d, ox, oy):
        d.polygon(P([(38, 32), (45, 19), (50, 14), (54, 18), (51, 28), (48, 40)], ox, oy), fill=255)
    layers.append((Part(HORSE, neckp), 0, neck))

    def headp(d, ox, oy):
        d.polygon(P([(48, 15), (53, 12), (56, 15), (61, 23), (61, 26), (58, 27), (54, 24), (50, 21)], ox, oy), fill=255)
        d.polygon(P([(50, 13), (51, 9), (53, 12)], ox, oy), fill=255)
    head_deco = [(54, 16, INK), (55, 16, INK), (54, 15, HORSE[4]), (60, 25, HORSE[0]), (59, 25, HORSE[0]),
                 (52, 14, BRONZE[2]), (53, 17, BRONZE[2]), (54, 19, BRONZE[2]), (56, 21, BRONZE[2]),
                 (58, 22, BRONZE[2]), (57, 23, BRONZE[3])]
    layers.append((Part(HORSE, headp, head_deco), 0, neck))

    def manep(d, ox, oy):
        d.polygon(P([(38, 31), (42, 23), (47, 16), (50, 13), (48, 17), (44, 25), (40, 32)], ox, oy), fill=255)
    layers.append((Part(MANE, manep), 0, neck))

    def saddle(d, ox, oy):
        d.polygon(P([(23, 29), (37, 29), (38, 39), (22, 39)], ox, oy), fill=255)
    trim = [(x, 39, BRONZE[2]) for x in range(23, 38)] + [(x, 38, BRONZE[3]) for x in range(23, 38, 2)]
    layers.append((Part(CLOTH_RED, saddle, trim), 0, body))

    ry = body + breath
    def rider_leg(d, ox, oy):
        d.polygon(P([(28, 30), (34, 30), (35, 37), (33, 42), (30, 41), (31, 36)], ox, oy), fill=255)
    layers.append((Part(IVORY, rider_leg), 0, body))

    def rider_boot(d, ox, oy):
        d.rectangle(B((30, 40, 35, 43), ox, oy), fill=255)
    layers.append((Part(LEATHER, rider_boot, hi=False), 0, body))

    def torso(d, ox, oy):
        d.polygon(P([(26, 18), (36, 18), (37, 31), (25, 31)], ox, oy), fill=255)
    lam = [(xx, yy, BLUE[1]) for yy in range(21, 30, 3) for xx in range(27 + yy % 2, 36, 2)]
    lam += [(xx, 28, BRONZE[2]) for xx in range(26, 37)]
    layers.append((Part(BLUE, torso, lam, band=True), 0, ry))

    def face(d, ox, oy):
        d.ellipse(B((28, 8, 37, 18), ox, oy), fill=255)
    eyes = [(33, 13, INK), (33, 14, INK), (36, 13, INK), (36, 14, INK), (35, 16, SKIN[1])]
    layers.append((Part(SKIN, face, eyes), 0, ry))

    def helm(d, ox, oy):
        d.chord(B((26, 4, 38, 18), ox, oy), 180, 360, fill=255)
        d.rectangle(B((27, 10, 38, 11), ox, oy), fill=255)
    rim = [(xx, 11, BRONZE[2]) for xx in range(27, 39)] + [(xx, 10, BRONZE[3]) for xx in range(28, 38)]
    layers.append((Part(BLUE, helm, rim), 0, ry))

    def plume(d, ox, oy):
        d.polygon(P([(30, 5), (32, 5), (31, 3), (27, 4)], ox, oy), fill=255)
    layers.append((Part(TASSEL, plume), 0, ry))

    # spear: diagonal shaft, tip upper right
    def shaft(d, ox, oy):
        d.line(P([(19, 47), (50, 9)], ox, oy), fill=255, width=2)
    layers.append((Part(WOOD, shaft, hi=False), 0, ry))

    def tip(d, ox, oy):
        d.polygon(P([(49, 10), (52, 9), (57, 3), (51, 6)], ox, oy), fill=255)
    layers.append((Part(STEEL, tip), 0, ry))

    def tas(d, ox, oy):
        d.polygon(P([(47, 11), (50, 11), (49, 16), (46, 15)], ox, oy), fill=255)
    layers.append((Part(TASSEL, tas), 0, ry))

    def arm(d, ox, oy):
        d.polygon(P([(33, 19), (37, 20), (38, 26), (35, 27)], ox, oy), fill=255)
    layers.append((Part(BLUE, arm), 0, ry))

    def hand(d, ox, oy):
        d.rectangle(B((35, 25, 38, 28), ox, oy), fill=255)
    layers.append((Part(SKIN, hand), 0, ry))
    return compose(layers)


# ---------------------------------------------------------------- sheet
def build_sheet():
    rows = []
    rows.append(("infantry-ally",
                 [infantry_frame(breath=b) for b in IDLE_BREATH] + [infantry_frame(walk=w) for w in WALK],
                 [("idle", 0, 8, 150), ("walk", 8, 8, 100)]))
    rows.append(("cavalry-ally",
                 [cavalry_frame(f, False) for f in range(8)] + [cavalry_frame(f, True) for f in range(8)],
                 [("idle", 0, 8, 150), ("walk", 8, 8, 110)]))
    rows.append(("bandit-enemy",
                 [infantry_frame(breath=b, bandit=True) for b in IDLE_BREATH],
                 [("idle", 0, 8, 150)]))
    cols = max(len(r[1]) for r in rows)
    sheet = Image.new("RGBA", (cols * CELL, len(rows) * CELL), (0, 0, 0, 0))
    meta = {"cell": [CELL, CELL], "anchor": list(ANCHOR), "displayScale": 1, "sprites": []}
    for ri, (sid, frames, anims) in enumerate(rows):
        for ci, fr in enumerate(frames):
            sheet.paste(fr, (ci * CELL, ri * CELL))
        meta["sprites"].append({"id": sid, "row": ri, "anims": [
            {"name": n, "start": s, "frames": c, "ms": ms} for n, s, c, ms in anims]})
    return sheet, meta, rows


def check_sheet(rows):
    problems = []
    for sid, frames, _ in rows:
        for i, fr in enumerate(frames):
            a = fr.getchannel("A")
            vals = set(a.getdata())
            if not vals <= {0, 255}:
                problems.append(f"{sid}[{i}] alpha {vals}")
            px = a.load()
            low = max(y for y in range(CELL) for x in range(CELL) if px[x, y])
            if low != ANCHOR[1]:
                problems.append(f"{sid}[{i}] lowest row {low} != {ANCHOR[1]}")
            for k in range(CELL):
                if px[k, 0] or px[k, CELL - 1] or px[0, k] or px[CELL - 1, k]:
                    problems.append(f"{sid}[{i}] touches edge")
                    break
    return problems


# ---------------------------------------------------------------- portrait (관우)
PW = 72


def portrait():
    S = PW
    canvas = {}

    def layer(ramp, fn, deco=(), band=True, hi=True):
        m = Image.new("L", (S, S), 0)
        fn(ImageDraw.Draw(m))
        mp = m.load()
        pts = {(x, y) for y in range(S) for x in range(S) if mp[x, y]}
        if not pts:
            return
        xs = [p[0] for p in pts]
        minx, w = min(xs), max(xs) - min(xs) + 1
        for x, y in pts:
            for n in ((x + 1, y), (x - 1, y), (x, y + 1), (x, y - 1)):
                if n not in pts:
                    canvas[n] = ramp[0]
        for x, y in pts:
            ro, do = (x + 1, y) not in pts, (x, y + 1) not in pts
            lo, uo = (x - 1, y) not in pts, (x, y - 1) not in pts
            if ro or do:
                c = ramp[1]
            elif band and ((x + 2, y) not in pts or (x + 3, y) not in pts):
                c = ramp[1]
            elif lo or uo:
                c = ramp[4] if (hi and uo and x < minx + w * 0.45) else ramp[3]
            elif band and ((x - 2, y) not in pts or (x, y - 2) not in pts):
                c = ramp[3]
            else:
                c = ramp[2]
            canvas[(x, y)] = c
        for x, y, c in deco:
            canvas[(x, y)] = c

    # robe & shoulders
    layer(ROBE_GREEN, lambda d: d.polygon([(4, 71), (8, 58), (22, 52), (50, 52), (64, 58), (68, 71)], fill=255),
          deco=[(x, y, ROBE_GREEN[3]) for x, y in [(14, 60), (15, 61), (16, 62), (17, 63), (18, 64)]]
          + [(x, 56 + (x - 40) // 3, BRONZE[2]) for x in range(40, 62)])
    # inner collar
    layer(IVORY, lambda d: d.polygon([(26, 52), (36, 66), (46, 52)], fill=255), band=False)
    # scarf tails hanging behind head
    layer(SCARF_GREEN, lambda d: d.polygon([(16, 18), (21, 16), (20, 30), (15, 46), (11, 44), (14, 30)], fill=255),
          deco=[(16, y, SCARF_GREEN[1]) for y in range(24, 42)])
    # neck
    layer(FACE_RED, lambda d: d.rectangle([29, 42, 43, 54], fill=255))
    # ear
    layer(FACE_RED, lambda d: d.ellipse([20, 28, 25, 37], fill=255), band=False,
          deco=[(22, 31, FACE_RED[0]), (22, 32, FACE_RED[0]), (23, 33, FACE_RED[1])])
    # face (3/4, narrower jaw)
    layer(FACE_RED, lambda d: (d.ellipse([23, 15, 49, 46], fill=255),
                               d.polygon([(25, 34), (47, 34), (44, 46), (36, 50), (28, 46)], fill=255)))
    # headscarf
    layer(SCARF_GREEN, lambda d: (d.chord([19, 3, 53, 37], 180, 360, fill=255),
                                  d.polygon([(19, 20), (53, 20), (52, 24), (20, 24)], fill=255)),
          deco=[(x, 22, SCARF_GREEN[1]) for x in range(21, 52)] + [(x, 11, SCARF_GREEN[3]) for x in range(27, 40)]
          + [(x, 8, SCARF_GREEN[4]) for x in range(30, 36)] + [(x, 15, SCARF_GREEN[1]) for x in range(24, 31)])
    # long beard flowing to a point
    beard = [(27, 42), (31, 46), (37, 47), (44, 44), (48, 40), (49, 47), (46, 58), (41, 68), (38, 71), (34, 67), (29, 57), (25, 47)]
    strands = []
    for k, x0 in enumerate(range(29, 47, 3)):
        for y in range(48, 70):
            x = x0 + (y - 48) // 8 - (k > 3) * ((y - 48) // 6)
            if (y + k) % 5 < 3:
                strands.append((x, y, HAIR[3] if k % 2 else HAIR[4]))
    layer(HAIR, lambda d: d.polygon(beard, fill=255), deco=strands)
    # mustache, side whiskers
    layer(HAIR, lambda d: (d.polygon([(30, 40), (36, 38), (37, 41), (31, 43), (27, 47)], fill=255),
                           d.polygon([(38, 38), (45, 39), (49, 45), (44, 42), (38, 41)], fill=255)), band=False)
    deco_face = []
    # phoenix eyes: dark upper lid, iris 2px tall, slant upward outward
    for x, y in [(29, 28), (30, 28), (31, 28), (32, 27), (33, 27), (34, 27),
                 (40, 27), (41, 27), (42, 27), (43, 27), (44, 28), (45, 28)]:
        deco_face.append((x, y, INK))
    for x, y in [(30, 29), (31, 29), (33, 29), (34, 28), (40, 28), (41, 28), (43, 28), (44, 29)]:
        deco_face.append((x, y, (240, 232, 212)))
    for x, y in [(32, 28), (32, 29), (33, 28), (42, 28), (42, 29), (41, 29)]:
        deco_face.append((x, y, INK))
    for x, y in [(29, 30), (30, 30), (31, 30), (32, 30), (41, 30), (42, 30), (43, 30), (44, 30)]:
        deco_face.append((x, y, FACE_RED[1]))
    # brows thick, slanting up outward
    for x, y in [(28, 25), (29, 24), (30, 24), (31, 23), (32, 23), (33, 23), (29, 25), (30, 25), (31, 24),
                 (40, 23), (41, 23), (42, 23), (43, 23), (44, 24), (45, 23), (46, 22), (41, 24), (42, 24), (43, 24)]:
        deco_face.append((x, y, HAIR[0]))
    # nose
    deco_face += [(38, 29, FACE_RED[1]), (38, 30, FACE_RED[1]), (38, 31, FACE_RED[1]), (39, 32, FACE_RED[1]),
                  (39, 33, FACE_RED[1]), (36, 35, FACE_RED[0]), (37, 35, FACE_RED[0]), (39, 35, FACE_RED[0]),
                  (37, 30, FACE_RED[3]), (37, 31, FACE_RED[3]), (37, 32, FACE_RED[4]), (37, 33, FACE_RED[3])]
    # cheekbone light & under-eye shadow
    deco_face += [(29, 32, FACE_RED[3]), (30, 32, FACE_RED[4]), (30, 33, FACE_RED[3]), (31, 33, FACE_RED[3]),
                  (43, 31, FACE_RED[1]), (44, 31, FACE_RED[1]), (45, 32, FACE_RED[1])]
    for x, y, c in deco_face:
        canvas[(x, y)] = c

    img = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    px = img.load()
    for (x, y), c in canvas.items():
        if 0 <= x < S and 0 <= y < S:
            px[x, y] = c + (255,)
    edge = [(x, y) for (x, y) in canvas if 0 <= x < S and 0 <= y < S and any(
        n not in canvas for n in ((x + 1, y), (x - 1, y), (x, y + 1), (x, y - 1)))]
    for x, y in edge:
        if y < S - 1:
            px[x, y] = INK + (255,)
    return img


# ---------------------------------------------------------------- terrain tiles
def wash_level(wx, wy):
    """Map-scale ink-wash tone: -1/0/+1, continuous in world coords (seamless)."""
    w = vnoise(wx + 4096, wy + 4096, 96, 1 << 16, 401) + (h(wx // 4, wy // 4, 402) - 0.5) * 0.08
    return -1 if w < 0.36 else (1 if w > 0.64 else 0)


def grass_px(x, y, seed=1, ramp=GRASS, wx=None, wy=None):
    if wx is not None:
        base = grass_px(x, y, seed, ramp)
        i = ramp.index(base) + wash_level(wx, wy)
        return ramp[max(0, min(len(ramp) - 1, i))]
    v = 0.75 * vnoise(x, y, 32, 64, seed) + 0.25 * vnoise(x, y, 8, 64, seed + 7)
    v += (h(x % 64, y % 64, seed + 3) - 0.5) * 0.08
    if v < 0.4:
        return ramp[1]
    if v < 0.66:
        return ramp[2]
    return ramp[3]


def tile_plain(tx, ty):
    t = Image.new("RGB", (64, 64))
    px = t.load()
    for y in range(64):
        for x in range(64):
            px[x, y] = grass_px(x, y, wx=tx * 64 + x, wy=ty * 64 + y)
    # tufts
    for k in range(5):
        cx = 6 + int(h(tx, ty, k, 11) * 52)
        cy = 8 + int(h(tx, ty, k, 12) * 50)
        for dx, dy, c in [(0, 0, GRASS[0]), (-1, -1, GRASS[1]), (-2, -2, GRASS[3]), (1, -1, GRASS[1]),
                          (2, -2, GRASS[3]), (0, -1, GRASS[1]), (0, -2, GRASS[4])]:
            px[cx + dx, cy + dy] = c
    if h(tx, ty, 99) < 0.4:
        cx = 8 + int(h(tx, ty, 98) * 48)
        cy = 8 + int(h(tx, ty, 97) * 48)
        for dx, dy in [(0, 0), (2, 1), (-1, 2)]:
            px[cx + dx, cy + dy] = (232, 222, 190)
            px[cx + dx, cy + dy + 1] = GRASS[0]
    if h(tx, ty, 55) < 0.3:
        cx = 8 + int(h(tx, ty, 56) * 48)
        cy = 8 + int(h(tx, ty, 57) * 48)
        for dx, dy, c in [(0, 0, (150, 144, 128)), (1, 0, (180, 174, 156)), (2, 0, (150, 144, 128)),
                          (0, 1, (110, 104, 92)), (1, 1, (110, 104, 92)), (2, 1, INK)]:
            px[cx + dx, cy + dy] = c
    return t


def stamp(img, sprite, x, y):
    img.paste(sprite, (x, y), sprite)


def tree_sprite(r, seed):
    size = r * 2 + 8
    m = Image.new("L", (size, size), 0)
    d = ImageDraw.Draw(m)
    c = size // 2
    # lumpy canopy: union of circles
    lobes = []
    for k in range(7):
        a = k / 7 * 6.283
        import math
        lx = c + int(math.cos(a) * r * 0.55)
        ly = c - 2 + int(math.sin(a) * r * 0.45)
        rr = int(r * (0.5 + 0.15 * h(seed, k)))
        lobes.append((lx, ly, rr))
        d.ellipse([lx - rr, ly - rr, lx + rr, ly + rr], fill=255)
    d.ellipse([c - r * 0.7, c - r * 0.7 - 2, c + r * 0.7, c + r * 0.7 - 2], fill=255)
    mp = m.load()
    pts = {(x, y) for y in range(size) for x in range(size) if mp[x, y]}
    img = Image.new("RGBA", (size, size + 8), (0, 0, 0, 0))
    px = img.load()
    # trunk
    for y in range(c + r // 2, c + r // 2 + 9):
        for x in range(c - 2, c + 2):
            if y < size + 8:
                px[x, y] = (WOOD[1] if x > c - 1 else WOOD[3]) + (255,)
    for x, y in pts:
        # per-lobe light: pixel near top-left of a lobe = light
        lit = 0
        for lx, ly, rr in lobes:
            ddx, ddy = x - lx, y - ly
            if ddx * ddx + ddy * ddy <= rr * rr:
                v = (-(ddx) - ddy * 1.3) / rr
                lit = max(lit, v)
        n = h(x, y, seed) * 0.35
        v = lit + n
        if v > 1.15:
            col = LEAF[4]
        elif v > 0.7:
            col = LEAF[3]
        elif v > 0.15:
            col = LEAF[2]
        else:
            col = LEAF[1]
        px[x, y] = col + (255,)
    for x, y in pts:
        if any(n not in pts for n in ((x + 1, y), (x - 1, y), (x, y + 1), (x, y - 1))):
            px[x, y] = INK + (255,) if (y > c - r * 0.3 or h(x, y, 5) < 0.6) else LEAF[1] + (255,)
    return img


def tile_forest(tx, ty):
    t = Image.new("RGB", (64, 64))
    px = t.load()
    for y in range(64):
        for x in range(64):
            px[x, y] = grass_px(x, y, 3, FLOOR, tx * 64 + x, ty * 64 + y)
    trees = [(17, 17, 13, 1), (47, 21, 12, 2), (31, 43, 14, 3)]
    for tx0, ty0, r, s in sorted(trees, key=lambda q: q[1]):
        spr = tree_sprite(r, s + tx * 7 + ty * 13)
        # ground shadow
        for y in range(ty0 + r - 3, ty0 + r + 3):
            for x in range(tx0 - r + 2, tx0 + r - 1):
                if 0 <= x < 64 and 0 <= y < 64 and ((x - tx0) / (r - 1)) ** 2 + ((y - ty0 - r) / 3.2) ** 2 <= 1:
                    px[x, y] = FLOOR[0]
        stamp(t, spr, tx0 - spr.width // 2, ty0 - spr.width // 2)
    return t


def earth_px(x, y, seed=21):
    v = 0.6 * vnoise(x, y, 16, 64, seed) + 0.4 * vnoise(x, y, 8, 64, seed + 1)
    if v < 0.42:
        return EARTH[1]
    if v < 0.7:
        return EARTH[2]
    return EARTH[3]


def house_sprite(w, hgt):
    """w: wall width, hgt: wall height. Returns RGBA."""
    W = w + 12
    H = hgt + 22
    img = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    px = img.load()
    roof_h = 14
    wx0 = 6
    # walls
    for y in range(roof_h - 2, roof_h - 2 + hgt):
        for x in range(wx0, wx0 + w):
            c = WALL[2]
            if x == wx0 + w - 1 or y == roof_h - 3 + hgt:
                c = WALL[1]
            elif x == wx0:
                c = WALL[3]
            px[x, y] = c + (255,)
    # timber posts
    for x in (wx0 + 2, wx0 + w - 3):
        for y in range(roof_h - 2, roof_h - 2 + hgt):
            px[x, y] = WOOD[1] + (255,)
    # door
    dx0 = wx0 + w // 2 - 3
    for y in range(roof_h - 2 + hgt - 9, roof_h - 2 + hgt):
        for x in range(dx0, dx0 + 6):
            px[x, y] = (WOOD[2] if x < dx0 + 3 else WOOD[1]) + (255,)
    # lattice window
    for y in range(roof_h + 1, roof_h + 6):
        for x in range(wx0 + 4, wx0 + 10):
            px[x, y] = (WOOD[0] if (x + y) % 2 == 0 else WALL[4]) + (255,)
    # roof: trapezoid with upturned eaves
    for y in range(roof_h):
        inset = max(0, (roof_h - y) // 2 - 1)
        lift = 0
        x0, x1 = 1 + inset, W - 2 - inset
        for x in range(x0, x1 + 1):
            c = ROOF[2] if (x // 2) % 2 == 0 else ROOF[1]
            if y < 2:
                c = ROOF[0]
            elif y == 2:
                c = ROOF[4]
            elif y >= roof_h - 2:
                c = ROOF[3] if y == roof_h - 2 else ROOF[0]
            elif x < x0 + 2:
                c = ROOF[3]
            px[x, y] = c + (255,)
    # upturned corners
    for (x, y) in [(0, roof_h - 3), (1, roof_h - 2), (W - 1, roof_h - 3), (W - 2, roof_h - 2), (0, roof_h - 4), (W - 1, roof_h - 4)]:
        px[x, y] = ROOF[0] + (255,)
    # ink outline
    pts = {(x, y) for y in range(H) for x in range(W) if px[x, y][3]}
    out = [(x, y) for x, y in pts if any((n not in pts) for n in ((x + 1, y), (x - 1, y), (x, y + 1), (x, y - 1)))]
    for x, y in out:
        px[x, y] = INK + (255,)
    return img


def tile_village(tx, ty):
    t = Image.new("RGB", (64, 64))
    px = t.load()
    for y in range(64):
        for x in range(64):
            px[x, y] = earth_px(x, y)
    variant = (tx + ty * 2) % 3
    if variant == 0:
        hs = house_sprite(30, 18)
        # shadow
        for y in range(44, 48):
            for x in range(14, 50):
                px[x, y] = EARTH[0]
        stamp(t, hs, 32 - hs.width // 2, 12)
    elif variant == 1:
        a = house_sprite(18, 14)
        b = house_sprite(16, 12)
        stamp(t, a, 4, 4)
        stamp(t, b, 30, 30)
        # fence
        for x in range(4, 30, 3):
            for y in range(52, 58):
                px[x, y] = WOOD[1] if y > 52 else WOOD[3]
        for x in range(4, 30):
            px[x, 54] = WOOD[2]
    else:
        hs = house_sprite(24, 16)
        stamp(t, hs, 22, 4)
        # well
        cx, cy = 14, 46
        for y in range(cy - 5, cy + 6):
            for x in range(cx - 6, cx + 7):
                dd = (x - cx) ** 2 / 36 + (y - cy) ** 2 / 25
                if dd <= 1:
                    px[x, y] = INK if dd > 0.7 else ((150, 144, 128) if dd > 0.4 else WATER[0])
    return t


def bank_noise(v, seed):
    return int(vnoise(v, 0, 8, 1 << 16, seed) * 4)


def tile_water(tx, ty, grid):
    t = Image.new("RGB", (64, 64))
    px = t.load()

    def is_water(x, y):
        if y >= len(grid):
            return True
        if x < 0 or x >= len(grid[0]) or y < 0:
            return False
        return grid[y][x] == "~"

    for y in range(64):
        for x in range(64):
            v = 0.7 * vnoise(x, y, 16, 64, 31) + 0.3 * vnoise(x, y, 4, 64, 32)
            px[x, y] = WATER[1] if v < 0.42 else (WATER[2] if v < 0.7 else WATER[3])
    for k in range(7):
        cx = 4 + int(h(tx, ty, k, 41) * 50)
        cy = 4 + int(h(tx, ty, k, 42) * 56)
        ln = 3 + int(h(tx, ty, k, 43) * 5)
        for i in range(ln):
            px[cx + i, cy] = WATER[4] if i in (1, 2) else WATER[3]
        px[cx + 1, cy + 1] = WATER[0]
    wy = ty * 64
    wx = tx * 64
    for y in range(64):
        for x in range(64):
            gl = None
            if not is_water(tx - 1, ty):
                b = 5 + bank_noise(wy + y, 51)
                if x < b:
                    gl = "g" if x < b - 3 else ("ink" if x == b - 3 else "mud")
            if not is_water(tx + 1, ty):
                b = 5 + bank_noise(wy + y, 52)
                if x > 63 - b:
                    gl = "g" if x > 66 - b else ("ink" if x == 66 - b else "mud")
            if not is_water(tx, ty - 1):
                b = 5 + bank_noise(wx + x, 53)
                if y < b:
                    gl = "g" if y < b - 3 else ("ink" if y == b - 3 else "mud")
            if gl == "g":
                px[x, y] = grass_px(x, y)
            elif gl == "ink":
                px[x, y] = INK
            elif gl == "mud":
                px[x, y] = EARTH[1] if px[x, y] != WATER[0] else EARTH[0]
    return t


# ---------------------------------------------------------------- stage data
def load_stage():
    t = open(DESIGN, encoding="utf-8").read()
    i = t.index("# 부록 B")
    j = t.index("```json", i) + 7
    k = t.index("```", j)
    d = json.loads(t[j:k])
    return d["Stages"][0], d


def render_map(grid):
    H, W = len(grid), len(grid[0])
    img = Image.new("RGB", (W * 64, H * 64))
    for ty in range(H):
        for tx in range(W):
            c = grid[ty][tx]
            if c == "f":
                tile = tile_forest(tx, ty)
            elif c == "v":
                tile = tile_village(tx, ty)
            elif c == "~":
                tile = tile_water(tx, ty, grid)
            else:
                tile = tile_plain(tx, ty)
            img.paste(tile, (tx * 64, ty * 64))
    return img


# ---------------------------------------------------------------- UI helpers
def font(sz, weight="Regular"):
    f = ImageFont.truetype(FONT, sz)
    f.set_variation_by_name(weight)
    return f


def blend(a, b, t):
    return tuple(int(a[i] * (1 - t) + b[i] * t) for i in range(3))


def paper_fill(img, box, seed=0):
    px = img.load()
    x0, y0, x1, y1 = box
    for y in range(y0, y1):
        for x in range(x0, x1):
            v = vnoise(x, y, 32, 1 << 16, seed) * 0.6 + h(x, y, seed) * 0.4
            c = PAPER[2]
            if v > 0.78:
                c = PAPER[3]
            elif v < 0.22:
                c = PAPER[1]
            if h(x // 2, y, seed + 9) < 0.012:
                c = PAPER[0]
            px[x, y] = c


def brush_border(img, box, seed=0, thick=5, ink=INK):
    """Pixel brush stroke frame with variable width and dry-brush gaps."""
    px = img.load()
    x0, y0, x1, y1 = box
    W, H = img.size

    def put(x, y, c):
        if 0 <= x < W and 0 <= y < H:
            px[x, y] = c

    def stroke(length, at, horiz, s):
        for i in range(length):
            w = 2 + int(vnoise(i, 0, 24, 1 << 16, s) * thick)
            for k in range(w):
                dry = k == w - 1 and h(i, k, s) < 0.45
                if dry:
                    continue
                col = ink if k < w - 1 else blend(ink, PAPER[1], 0.35)
                if horiz:
                    put(at[0] + i, at[1] + k * at[2], col)
                else:
                    put(at[0] + k * at[2], at[1] + i, col)

    stroke(x1 - x0, (x0, y0, 1), True, seed + 1)
    stroke(x1 - x0, (x0, y1 - 1, -1), True, seed + 2)
    stroke(y1 - y0, (x0, y0, 1), False, seed + 3)
    stroke(y1 - y0, (x1 - 1, y0, -1), False, seed + 4)
    # corner blots
    for cx, cy in [(x0 + 3, y0 + 3), (x1 - 4, y1 - 4)]:
        for dy in range(-4, 5):
            for dx in range(-4, 5):
                if dx * dx + dy * dy <= 12 + int(h(dx, dy, seed) * 6):
                    put(cx + dx, cy + dy, ink)


def seal(img, box, glyph, colors=SEAL, fsize=None):
    d = ImageDraw.Draw(img)
    x0, y0, x1, y1 = box
    px = img.load()
    for y in range(y0, y1):
        for x in range(x0, x1):
            edge = x in (x0, x1 - 1) or y in (y0, y1 - 1)
            corner = (x in (x0, x1 - 1)) and (y in (y0, y1 - 1))
            if corner or (edge and h(x, y, 77) < 0.25):
                continue
            px[x, y] = colors[1] if h(x // 2, y // 2, 78) > 0.15 else colors[2]
    if glyph:
        f = font(fsize or int((y1 - y0) * 0.7), "Bold")
        bb = d.textbbox((0, 0), glyph, font=f)
        tx = x0 + (x1 - x0 - (bb[2] - bb[0])) // 2 - bb[0]
        ty = y0 + (y1 - y0 - (bb[3] - bb[1])) // 2 - bb[1]
        d.text((tx, ty), glyph, font=f, fill=PAPER[3])


def overlay(img, box, color, alpha, border=None):
    px = img.load()
    x0, y0, x1, y1 = box
    W, H = img.size
    for y in range(max(0, y0), min(H, y1)):
        for x in range(max(0, x0), min(W, x1)):
            on_b = x in (x0, x1 - 1) or y in (y0, y1 - 1)
            if border and on_b:
                px[x, y] = blend(px[x, y], border, 0.7)
            else:
                px[x, y] = blend(px[x, y], color, alpha)


def gray_sprite(spr):
    g = spr.copy()
    px = g.load()
    for y in range(g.height):
        for x in range(g.width):
            r, gg, b, a = px[x, y]
            if a:
                l = int(0.3 * r + 0.59 * gg + 0.11 * b)
                l = int(l * 0.85 + 20)
                px[x, y] = (l, l, int(l * 0.95), 255)
    return g


def scale_nn(img, k):
    return img.resize((img.width * k, img.height * k), Image.NEAREST)


def ink_text(d, xy, text, f, fill=INK, stroke=None, sw=0):
    if stroke:
        d.text(xy, text, font=f, fill=fill, stroke_width=sw, stroke_fill=stroke)
    else:
        d.text(xy, text, font=f, fill=fill)


def brush_bar(img, x, y, w, frac, color, seed):
    px = img.load()
    for i in range(w):
        top = 1 + int(vnoise(i, 0, 16, 1 << 16, seed) * 2)
        bot = 9 - int(vnoise(i, 3, 16, 1 << 16, seed + 1) * 2)
        for yy in range(top, bot):
            c = color if i < w * frac else blend(PAPER[1], INK, 0.15)
            if i < w * frac and yy == top:
                c = blend(color, (255, 255, 255), 0.25)
            px[x + i, y + yy] = c
    # end blot
    e = int(w * frac)
    for yy in range(1, 10):
        if h(e, yy, seed) < 0.6:
            px[x + e, y + yy] = INK


# ---------------------------------------------------------------- screen
def move_range(grid, terrain, origin, mv, blocked, allies):
    import heapq
    H, W = len(grid), len(grid[0])
    dist = {origin: 0}
    q = [(0, origin)]
    while q:
        c, (x, y) = heapq.heappop(q)
        if c > dist[(x, y)]:
            continue
        for nx, ny in ((x + 1, y), (x - 1, y), (x, y + 1), (x, y - 1)):
            if not (0 <= nx < W and 0 <= ny < H) or (nx, ny) in blocked:
                continue
            cost = terrain[grid[ny][nx]]["Cost"]
            if cost <= 0:
                continue
            nc = c + cost
            if nc <= mv and nc < dist.get((nx, ny), 99):
                dist[(nx, ny)] = nc
                heapq.heappush(q, (nc, (nx, ny)))
    return {p for p in dist if p not in allies or p == origin}


def build_screen(sheet_rows, por):
    stage, data = load_stage()
    grid = stage["Tiles"]
    mapimg = render_map(grid)
    SW, SH = 1280, 850
    VX0, VY0 = 1 * 64, 46  # viewport origin in map px (cols 1.., crop top 46)
    VW = 896
    screen = Image.new("RGB", (SW, SH), (29, 26, 23))
    screen.paste(mapimg.crop((VX0, VY0, VX0 + VW, VY0 + SH)), (0, 0))
    d = ImageDraw.Draw(screen)

    def tile_xy(tx, ty):
        return tx * 64 - VX0, ty * 64 - VY0

    # faint grid
    px = screen.load()
    for tx in range(1, 16):
        x = tx * 64 - VX0
        if 0 <= x < VW:
            for y in range(SH):
                px[x, y] = blend(px[x, y], INK, 0.22)
    for ty in range(0, 15):
        y = ty * 64 - VY0
        if 0 <= y < SH:
            for x in range(VW):
                px[x, y] = blend(px[x, y], INK, 0.22)

    inf = sheet_rows[0][1]
    cav = sheet_rows[1][1]
    ban = sheet_rows[2][1]
    allies = {(2, 5): ("유비", inf[2], False), (2, 6): ("관우", cav[1], False), (5, 8): ("장비", cav[5], True)}
    enemies = {(7, 6): ("황건적", ban[0]), (6, 8): ("황건적", ban[3]), (10, 7): ("황건적", ban[5]),
               (8, 10): ("황건적", ban[2]), (14, 6): ("정원지", ban[1]), (12, 8): ("등무", ban[4])}
    terrain = data["Terrain"]["기병"]
    rng = move_range(grid, terrain, (2, 6), 5, set(enemies), set(allies))
    for (tx, ty) in rng:
        x, y = tile_xy(tx, ty)
        overlay(screen, (x + 1, y + 1, x + 64, y + 64), (70, 110, 190), 0.24, border=(40, 64, 120))
    targets = [e for e in enemies if any(
        (e[0] + dx, e[1] + dy) in rng for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)))]
    for (tx, ty) in targets:
        x, y = tile_xy(tx, ty)
        overlay(screen, (x + 1, y + 1, x + 64, y + 64), (190, 50, 36), 0.38, border=(130, 30, 24))

    # units
    def draw_unit(pos, spr, ally, done, name=None):
        x, y = tile_xy(*pos)
        # ground shadow
        for yy in range(-2, 3):
            for xx in range(-14, 15):
                if (xx / 14) ** 2 + (yy / 2.6) ** 2 <= 1:
                    sx, sy = x + 32 + xx, y + 57 + yy
                    if 0 <= sx < VW and 0 <= sy < SH:
                        px[sx, sy] = blend(px[sx, sy], INK, 0.35)
        s = gray_sprite(spr) if done else spr
        screen.paste(s, (x + 32 - ANCHOR[0], y + 60 - ANCHOR[1]), s)
        # faction seal (UI)
        cols = INDIGO_SEAL if ally else SEAL
        seal(screen, (x + 3, y + 3, x + 13, y + 13), None, colors=cols if not done else
             [(90, 88, 84), (120, 118, 112), (140, 138, 130)])
        # hp brush bar
        bx, by = x + 14, y + 61
        for i in range(36):
            for yy in range(2):
                px[bx + i, by + yy] = (cols[2] if not done else (130, 128, 120)) if i < 36 else INK
        if name:
            f = font(12, "Bold")
            bb = d.textbbox((0, 0), name, font=f)
            tw = bb[2] - bb[0]
            tx0 = x + 32 - tw // 2 - 4
            paper_fill(screen, (tx0, y - 6, tx0 + tw + 8, y + 10), 5)
            for xx in range(tx0, tx0 + tw + 8):
                px[xx, y + 10] = INK
            d.text((tx0 + 4, y - 7), name, font=f, fill=SEAL[0] if not ally else INDIGO_SEAL[0])

    order = sorted(list(allies.items()) + list(enemies.items()), key=lambda kv: kv[0][1])
    for pos, val in order:
        if pos in allies:
            n, spr, done = val
            draw_unit(pos, spr, True, done, n)
        else:
            n, spr = val
            draw_unit(pos, spr, False, False, n if n != "황건적" else None)

    # selection cursor on 관우: brush corners
    x, y = tile_xy(2, 6)
    for (cx, cy, sx, sy) in [(x, y, 1, 1), (x + 63, y, -1, 1), (x, y + 63, 1, -1), (x + 63, y + 63, -1, -1)]:
        for i in range(14):
            for k in range(3):
                px[cx + sx * i, cy + sy * k] = PAPER[3] if k == 1 else INK
                px[cx + sx * k, cy + sy * i] = PAPER[3] if k == 1 else INK
    # target cursor on (7,6)
    x, y = tile_xy(7, 6)
    for i in range(64):
        for k in (0, 1):
            if (i // 4) % 2 == 0:
                for (a, b) in [(x + i, y + k), (x + i, y + 63 - k), (x + k, y + i), (x + 63 - k, y + i)]:
                    px[a, b] = SEAL[2]

    # floating damage number over (6,8)
    x, y = tile_xy(6, 8)
    fdmg = font(30, "Black")
    ink_text(d, (x + 8, y - 22), "-142", fdmg, fill=SEAL[2], stroke=PAPER[3], sw=3)

    # turn / objective tags (minimal chrome)
    paper_fill(screen, (14, 14, 286, 60), 11)
    brush_border(screen, (14, 14, 286, 60), 11, thick=3)
    seal(screen, (24, 21, 56, 53), "一", colors=INDIGO_SEAL)
    d.text((66, 20), "1턴 · 아군 진영", font=font(22, "Bold"), fill=INK)
    paper_fill(screen, (14, 68, 230, 102), 12)
    brush_border(screen, (14, 68, 230, 102), 12, thick=2)
    d.text((26, 72), "목표", font=font(17, "Bold"), fill=SEAL[1])
    d.text((76, 72), "장각 격퇴", font=font(17, "Medium"), fill=INK)

    # ---------- right panel
    PX0 = VW
    paper_fill(screen, (PX0, 0, SW, SH), 21)
    # ink wash band separator
    for y in range(SH):
        for k in range(6):
            if k < 4 or h(y, k, 3) < 0.5:
                px[PX0 + k, y] = INK
    # portrait mounting (족자 style)
    pk = 3
    pw = PW * pk
    fx0 = PX0 + (SW - PX0 - pw) // 2
    fy0 = 26
    # silk mounting
    for y in range(fy0 - 12, fy0 + pw + 12):
        for x in range(fx0 - 12, fx0 + pw + 12):
            px[x, y] = (96, 74, 54) if h(x // 3, y // 3, 8) > 0.1 else (84, 64, 46)
    for y in range(fy0 - 4, fy0 + pw + 4):
        for x in range(fx0 - 4, fx0 + pw + 4):
            px[x, y] = BRONZE[2] if (x in (fx0 - 4, fx0 + pw + 3) or y in (fy0 - 4, fy0 + pw + 3)) else PAPER[1]
    # ink-wash mountains backdrop inside portrait window (native res, scaled)
    bd = Image.new("RGB", (PW, PW))
    bpx = bd.load()
    for yy in range(PW):
        for xx in range(PW):
            c = PAPER[2] if h(xx, yy, 61) > 0.03 else PAPER[1]
            m1 = 30 + int(10 * vnoise(xx, 0, 12, 1 << 16, 62)) + abs(xx - 52) // 3
            m2 = 44 + int(8 * vnoise(xx, 0, 9, 1 << 16, 63)) + abs(xx - 14) // 4
            if yy > m2:
                c = (174, 170, 156) if yy > m2 + 1 else (120, 116, 106)
            elif yy > m1:
                c = (204, 198, 180) if yy > m1 + 1 else (156, 150, 136)
            bpx[xx, yy] = c
    stamp(bd, por, 0, 0)
    screen.paste(scale_nn(bd, pk), (fx0, fy0))
    # name seal on painting corner
    seal(screen, (fx0 + pw - 40, fy0 + 8, fx0 + pw - 8, fy0 + 40), "關")

    y0 = fy0 + pw + 22
    d.text((PX0 + 28, y0), "관우", font=font(34, "Black"), fill=INK)
    d.text((PX0 + 120, y0 + 12), "운장 · 경기병", font=font(17, "Medium"), fill=(80, 70, 60))
    seal(screen, (SW - 66, y0 + 4, SW - 30, y0 + 40), "劉", colors=INDIGO_SEAL)
    d.text((SW - 70, y0 + 44), "Lv 1", font=font(15, "Bold"), fill=INK)
    yb = y0 + 66
    for i, (lab, val, frac, col) in enumerate([
            ("병력", "1,020 / 1,020", 1.0, SEAL[1]),
            ("병법치", "18 / 36", 0.5, INDIGO_SEAL[2]),
            ("경험", "0 / 100", 0.02, BRONZE[2])]):
        yy = yb + i * 34
        d.text((PX0 + 28, yy - 4), lab, font=font(15, "Bold"), fill=INK)
        brush_bar(screen, PX0 + 96, yy, 160, frac, col, 70 + i)
        d.text((PX0 + 266, yy - 4), val, font=font(14, "Medium"), fill=INK)
    yt = yb + 104
    d.text((PX0 + 28, yt), "공격 158   방어 121   정신 96", font=font(15, "Medium"), fill=INK)
    d.text((PX0 + 28, yt + 24), "특성  호걸 · 반격술", font=font(15, "Medium"), fill=(80, 70, 60))

    # hanging scroll action menu
    sx0, sx1 = PX0 + 44, SW - 44
    sy0 = yt + 64
    items = ["이동", "공격", "병법", "아이템", "대기", "일기토"]
    sy1 = sy0 + 18 + len(items) * 38 + 10
    # rollers
    for ry in (sy0 - 8, sy1):
        for y in range(ry, ry + 9):
            for x in range(sx0 - 14, sx1 + 14):
                c = WOOD[2]
                if y in (ry, ry + 8):
                    c = INK
                elif y == ry + 2:
                    c = WOOD[4]
                elif y >= ry + 6:
                    c = WOOD[1]
                if x < sx0 - 6 or x > sx1 + 6:
                    c = BRONZE[2] if INK != c else INK
                px[x, y] = c
    for y in range(sy0 + 1, sy1):
        for x in range(sx0, sx1):
            px[x, y] = PAPER[3] if h(x, y, 90) > 0.02 else PAPER[1]
        px[sx0, y] = INK
        px[sx1 - 1, y] = INK
    for i, it in enumerate(items):
        iy = sy0 + 14 + i * 38
        sel = it == "공격"
        dim = it == "일기토"
        if sel:
            # red ink swash behind
            for y in range(iy + 2, iy + 30):
                for x in range(sx0 + 10, sx1 - 10):
                    t = (x - sx0 - 10) / (sx1 - sx0 - 20)
                    if h(x, y // 2, 91) < 0.92 - t * 0.5 and abs(y - iy - 16) < 13 - int(t * 4):
                        px[x, y] = blend(PAPER[3], SEAL[2], 0.32)
        # brush dot bullet
        bx, by = sx0 + 24, iy + 16
        for dy in range(-3, 4):
            for dx in range(-3, 4):
                if dx * dx + dy * dy <= 8:
                    px[bx + dx, by + dy] = SEAL[1] if sel else INK
        d.text((sx0 + 40, iy + 1), it, font=font(20, "Bold" if sel else "Medium"),
               fill=(150, 140, 126) if dim else (SEAL[0] if sel else INK))
        if it == "공격":
            d.text((sx1 - 92, iy + 7), "적 2 대상", font=font(13, "Medium"), fill=SEAL[0])
    # log
    ly = sy1 + 22
    d.text((PX0 + 28, ly), "장비 → 황건적  142 피해", font=font(14, "Medium"), fill=(70, 62, 54))
    d.text((PX0 + 28, ly + 22), "Space 추천 · Tab 부대 · Esc 메뉴", font=font(12, "Regular"), fill=(110, 100, 88))
    brush_border(screen, (PX0 + 4, 0, SW, SH), 31, thick=4)
    return screen


def main():
    os.makedirs(OUT, exist_ok=True)
    sheet, meta, rows = build_sheet()
    probs = check_sheet(rows)
    sheet.save(os.path.join(OUT, "sheet.png"))
    scale_nn(sheet, 4).save(os.path.join(OUT, "sheet-x4.png"))
    with open(os.path.join(OUT, "sheet.json"), "w") as f:
        json.dump(meta, f, indent=1)
    por = portrait()
    por.save(os.path.join(OUT, "portrait.png"))
    scale_nn(por, 4).save(os.path.join(OUT, "portrait-x4.png"))
    with open(os.path.join(OUT, "portrait.json"), "w") as f:
        json.dump({"size": [PW, PW], "displayScale": 3}, f)
    # terrain atlas sample: plain, forest, village x3, water(top bank)
    stage, _ = load_stage()
    grid = stage["Tiles"]
    tiles = [tile_plain(0, 0), tile_forest(9, 2), tile_village(1, 5), tile_village(2, 5), tile_village(3, 5),
             tile_water(9, 11, grid), tile_water(9, 12, grid)]
    atlas = Image.new("RGB", (64 * len(tiles), 64))
    for i, t in enumerate(tiles):
        atlas.paste(t, (i * 64, 0))
    atlas.save(os.path.join(OUT, "terrain.png"))
    scale_nn(atlas, 3).save(os.path.join(OUT, "terrain-x3.png"))
    screen = build_screen(rows, por)
    assert screen.size == (1280, 850)
    screen.save(os.path.join(OUT, "screen.png"))
    print("sheet checks:", "OK" if not probs else probs)


if __name__ == "__main__":
    main()
