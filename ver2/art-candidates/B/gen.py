#!/usr/bin/env python3
"""Candidate B (조조전 정밀): 48px oblique tiles, 56px unit cells, native ×1 battlefield."""
import json
import math
import os

from PIL import Image, ImageDraw, ImageFont

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "out")
ROOT = os.path.dirname(os.path.dirname(HERE))
DOC = os.path.join(ROOT, "srpg_full_design.md")
FONT = os.path.join(ROOT, "..", "ver1", "assets", "fonts", "NotoSansKR.ttf")

CW, CH = 56, 56
AX, AY = 28, 50  # ground anchor: lowest opaque row (outline) at AY
G = AY - 1       # lowest fill row
TILE = 48


def hx(s):
    return tuple(int(s[i:i + 2], 16) for i in (1, 3, 5))


def ramp(*cs):
    return [hx(c) for c in cs]


OUTLINE = hx("#1a1622")
BLUE = ramp("#17213f", "#233a74", "#30549a", "#4a7abe", "#8ab4e4")
GOLD = ramp("#5a3c14", "#8c6224", "#c49438", "#e6c060", "#fbe7a0")
SKIN = ramp("#6e3c28", "#a8684a", "#d89a70", "#f0bf92", "#fbdcb8")
FACE_R = ramp("#4a1a16", "#7a2a22", "#a8402e", "#c8604a", "#e08a6c")
IVORY = ramp("#5c5040", "#9a8a6a", "#cfc09a", "#e8dcbc", "#fbf3dc")
STEEL = ramp("#2c3240", "#5a6478", "#8e9aae", "#c4ceda", "#f0f4f8")
HORSE = ramp("#2e1a10", "#5c3420", "#8c5430", "#b47a48", "#d8a470")
DARK = ramp("#1e1a1e", "#342c2c", "#4c4038", "#6a5a4a", "#8a7860")
HAIR = ramp("#121016", "#1e1a22", "#2e2832", "#46404a", "#605a66")
RED = ramp("#3c1010", "#6e1c1a", "#a42c24", "#d0483a", "#ec7a62")
YEL = ramp("#6a4a0a", "#a87c14", "#dcb02c", "#f4d454", "#fff0a0")
GREEN = ramp("#10281c", "#1e4a30", "#2e6e46", "#4c9660", "#82c08a")
WOOD = ramp("#2a1a0e", "#4a301a", "#6e4a28", "#94683c", "#b88c58")
TAN = ramp("#3e2a14", "#6a4a24", "#94703a", "#b89458", "#d8bc84")

GRASS = ramp("#22401f", "#33582b", "#436e36", "#578443", "#76a057")
DIRT = ramp("#4a3420", "#6e4e30", "#8e6a42", "#a8845a", "#c4a276")
WATER = ramp("#12294a", "#1b3e68", "#285a8c", "#4580b0", "#88bede")
SLATE = ramp("#1e2232", "#323a52", "#48526e", "#66728e", "#93a0ba")
LEAF = ramp("#0f2716", "#1a3d22", "#285a2e", "#3c7a3a", "#62a050")

SPRITE_RAMPS = [BLUE, GOLD, SKIN, IVORY, STEEL, HORSE, DARK, HAIR, RED, YEL, WOOD, TAN]

LX, LY = -0.6, -0.8


def level(v, rp):
    if v > 0.42:
        i = 4
    elif v > 0.1:
        i = 3
    elif v > -0.32:
        i = 2
    elif v > -0.72:
        i = 1
    else:
        i = 0
    return rp[i]


def lit(nx, ny, bias):
    nz = math.sqrt(max(0.0, 1 - nx * nx - ny * ny))
    return 0.75 * (nx * LX + ny * LY) + 0.45 * nz - 0.3 + bias


class Cv:
    def __init__(self, w, h):
        self.w, self.h = w, h
        self.p = [[None] * w for _ in range(h)]

    def put(self, x, y, c):
        x, y = int(x), int(y)
        if c is not None and 0 <= x < self.w and 0 <= y < self.h:
            self.p[y][x] = c

    def get(self, x, y):
        if 0 <= x < self.w and 0 <= y < self.h:
            return self.p[y][x]
        return None

    def ellipse(self, cx, cy, rx, ry, rp, bias=0.0, flat=None):
        out = []
        for y in range(int(cy - ry) - 1, int(cy + ry) + 2):
            for x in range(int(cx - rx) - 1, int(cx + rx) + 2):
                nx = (x + 0.5 - cx) / rx
                ny = (y + 0.5 - cy) / ry
                if nx * nx + ny * ny <= 1.0:
                    self.put(x, y, flat or level(lit(nx, ny, bias), rp))
                    out.append((x, y))
        return out

    def capsule(self, a, b, r, rp, bias=0.0, flat=None):
        (x0, y0), (x1, y1) = a, b
        dx, dy = x1 - x0, y1 - y0
        ll = dx * dx + dy * dy or 1e-9
        out = []
        for y in range(int(min(y0, y1) - r) - 1, int(max(y0, y1) + r) + 2):
            for x in range(int(min(x0, x1) - r) - 1, int(max(x0, x1) + r) + 2):
                px, py = x + 0.5, y + 0.5
                t = max(0.0, min(1.0, ((px - x0) * dx + (py - y0) * dy) / ll))
                qx, qy = x0 + t * dx, y0 + t * dy
                nx, ny = (px - qx) / r, (py - qy) / r
                if nx * nx + ny * ny <= 1.0:
                    self.put(x, y, flat or level(lit(nx, ny, bias), rp))
                    out.append((x, y))
        return out

    def poly(self, pts, color_fn):
        m = Image.new("1", (self.w, self.h), 0)
        ImageDraw.Draw(m).polygon(pts, fill=1)
        out = []
        for y in range(self.h):
            for x in range(self.w):
                if m.getpixel((x, y)):
                    self.put(x, y, color_fn(x, y))
                    out.append((x, y))
        return out

    def line(self, a, b, c):
        (x0, y0), (x1, y1) = a, b
        n = int(max(abs(x1 - x0), abs(y1 - y0))) + 1
        for i in range(n):
            t = i / max(1, n - 1)
            self.put(round(x0 + (x1 - x0) * t), round(y0 + (y1 - y0) * t), c)

    def outline(self, c=OUTLINE):
        add = []
        for y in range(self.h):
            for x in range(self.w):
                if self.p[y][x] is None:
                    for ddx, ddy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                        if self.get(x + ddx, y + ddy) is not None and self.get(x + ddx, y + ddy) != c:
                            add.append((x, y))
                            break
        for x, y in add:
            self.p[y][x] = c

    def image(self):
        im = Image.new("RGBA", (self.w, self.h), (0, 0, 0, 0))
        px = im.load()
        for y in range(self.h):
            for x in range(self.w):
                c = self.p[y][x]
                if c is not None:
                    px[x, y] = c + (255,)
        return im


def step(c, rp, d):
    if c in rp:
        return rp[max(0, min(len(rp) - 1, rp.index(c) + d))]
    return c


def lamellar(cv, pts, rp, phase=0):
    s = set(pts)
    for x, y in pts:
        c = cv.get(x, y)
        if c not in rp or rp.index(c) < 2:
            continue
        if (y + phase) % 3 == 0:
            cv.put(x, y, step(c, rp, -1))
        elif (x + ((y + phase) // 3) * 2) % 4 == 0:
            cv.put(x, y, step(c, rp, -1))
    return s


# ---------------------------------------------------------------- characters

def leg(cv, hip, sx, lift, pants, boot, bias=0.0, wrap=None):
    fx, fy = hip[0] + sx, G - 2 - lift
    kx = (hip[0] + fx) / 2 + (0.8 if lift else 0)
    ky = (hip[1] + fy) / 2 - lift * 0.4
    cv.capsule(hip, (kx, ky), 2.1, pants, bias)
    cv.capsule((kx, ky), (fx, fy), 1.8, wrap or pants, bias - 0.05)
    cv.ellipse(fx + 0.8, G - 0.9 - lift, 2.7, 1.6, boot, bias)


def helmet_head(cv, cx, cy, plume=True):
    """Head centered on face at (cx, cy); helmet + face, facing SE."""
    cv.ellipse(cx - 0.5, cy + 1.2, 4.3, 4.0, SKIN, 0.05)
    cv.ellipse(cx - 0.8, cy - 3.2, 5.3, 4.6, BLUE, 0.05)
    for x in range(int(cx - 5), int(cx + 5)):
        cv.put(x, int(cy - 0.5), GOLD[3] if x < cx else GOLD[2])
    cv.put(int(cx - 6), int(cy + 0.5), BLUE[1])
    cv.put(int(cx - 6), int(cy + 1.5), BLUE[1])
    cv.put(int(cx - 5), int(cy + 1.5), BLUE[2])
    if plume:
        cv.capsule((cx - 1, cy - 9.5), (cx - 2.5, cy - 11.5), 1.4, RED, 0.1)
        cv.put(int(cx - 1), int(cy - 8), GOLD[3])
        cv.put(int(cx - 1), int(cy - 7), GOLD[2])
    cv.put(int(cx - 2), int(cy - 5), BLUE[4])
    cv.put(int(cx - 3), int(cy - 4), BLUE[4])
    face_details(cv, cx, cy)


def face_details(cv, cx, cy):
    ex = int(cx)
    ey = int(cy + 0.5)
    for x in (ex - 1, ex + 2):
        cv.put(x, ey, HAIR[0])
        cv.put(x, ey + 1, HAIR[0])
    cv.put(ex + 1, ey + 3, SKIN[1])
    cv.put(ex + 2, ey + 3, SKIN[1])


def bandit_head(cv, cx, cy):
    cv.ellipse(cx - 0.5, cy + 1.0, 4.3, 4.2, SKIN, 0.05)
    cv.ellipse(cx - 1.0, cy - 2.6, 4.8, 3.6, HAIR, 0.1)
    cv.ellipse(cx - 3.0, cy - 5.6, 2.2, 1.9, HAIR, 0.1)
    band = int(cy - 1)
    for x in range(int(cx - 5), int(cx + 4)):
        cv.put(x, band, YEL[3] if x < cx else YEL[2])
        cv.put(x, band - 1, YEL[2] if x < cx else YEL[1])
    cv.capsule((cx - 5, band), (cx - 9, band + 4), 0.9, YEL, 0.0)
    cv.capsule((cx - 5, band), (cx - 8, band + 6), 0.7, YEL, -0.2)
    face_details(cv, cx, cy)
    cv.put(int(cx) + 1, int(cy) + 3, HAIR[2])


def infantry_frame(up=0, bob=0, legs=((0, 0), (0, 0)), arms=(0, 0), enemy=False):
    cv = Cv(CW, CH)
    ub = up + bob
    hip_y = 36 + bob
    body = TAN if enemy else BLUE
    # far arm and shield / club hand
    far_sh = (34.5, 27 + ub)
    far_hand = (36 + arms[1], 34 + ub)
    cv.capsule(far_sh, far_hand, 1.6, SKIN if enemy else BLUE, -0.15)
    if not enemy:
        cv.ellipse(37.5 + arms[1], 32.5 + ub, 4.6, 5.0, WOOD, 0.05)
        ring = []
        for y in range(CH):
            for x in range(CW):
                nx = (x + 0.5 - (37.5 + arms[1])) / 4.6
                ny = (y + 0.5 - (32.5 + ub)) / 5.0
                d = nx * nx + ny * ny
                if 0.55 <= d <= 1.0:
                    ring.append((x, y))
        for x, y in ring:
            cv.put(x, y, GOLD[3] if (x + y) % 5 and y < 32.5 + ub else GOLD[1])
        cv.put(int(37 + arms[1]), int(32 + ub), GOLD[4])
        cv.put(int(38 + arms[1]), int(32 + ub), GOLD[2])
    # legs: far (right) first then near (left)
    pants = DARK if enemy else BLUE
    wrap = IVORY if enemy else None
    boot = WOOD if enemy else DARK
    leg(cv, (31, hip_y), legs[1][0], legs[1][1], pants, boot, -0.2, wrap)
    leg(cv, (25.5, hip_y), legs[0][0], legs[0][1], pants, boot, 0.0, wrap)
    # skirt / tassets
    sk = []
    skirt_rp = RED if enemy else IVORY
    sk = cv.poly([(22, 33 + ub), (35, 33 + ub), (36, 39 + bob), (21, 39 + bob)],
                 lambda x, y: skirt_rp[3] if x < 26 else (skirt_rp[2] if x < 32 else skirt_rp[1]))
    for x, y in sk:
        if (x % 3 == 0) and y > 34 + ub:
            cv.put(x, y, step(cv.get(x, y), skirt_rp, -1))
    # torso
    tp = cv.ellipse(28.5, 29.5 + ub, 6.4, 6.6, body, 0.05 if not enemy else -0.12)
    if not enemy:
        lamellar(cv, tp, BLUE, ub)
        for i in range(5):
            cv.put(26 + i, 24 + ub + (i if i < 3 else 4 - i), GOLD[3])
        for x in range(23, 35):
            cv.put(x, 34 + ub, GOLD[2] if x != 29 else GOLD[4])
            cv.put(x, 35 + ub, GOLD[1])
        cv.ellipse(22.5, 26.5 + ub, 3.2, 2.6, BLUE, 0.15)
        cv.ellipse(34.5, 26.5 + ub, 3.0, 2.4, BLUE, -0.1)
        cv.put(21, 25 + ub, GOLD[3])
        cv.put(22, 25 + ub, GOLD[3])
    else:
        for x in range(23, 35):
            cv.put(x, 33 + ub, RED[3] if x < 29 else RED[2])
            cv.put(x, 34 + ub, RED[1])
        cv.line((25, 24 + ub), (31, 33 + ub), TAN[1])
    # head
    if enemy:
        bandit_head(cv, 29.5, 18.5 + ub)
    else:
        helmet_head(cv, 29.5, 18.5 + ub)
    # weapon + near arm
    hand = (21.5 + arms[0], 34.5 + ub)
    if enemy:
        cv.capsule((hand[0] + 1, hand[1] - 1), (hand[0] - 6, hand[1] - 13), 1.5, WOOD, 0.1)
        cv.ellipse(hand[0] - 6, hand[1] - 13.5, 2.4, 2.4, WOOD, 0.1)
        cv.put(int(hand[0] - 7), int(hand[1] - 14), WOOD[4])
    else:
        tip = (hand[0] - 6, hand[1] + 11)
        cv.capsule((hand[0] - 0.5, hand[1] + 2), tip, 1.0, STEEL, 0.35)
        cv.line((hand[0] - 1, hand[1] + 2), (tip[0] - 0.5, tip[1] - 1), STEEL[4])
        cv.line((hand[0] - 2, hand[1] + 1), (hand[0] + 2, hand[1] + 2), GOLD[2])
    cv.capsule((22, 27 + ub), hand, 1.7, SKIN if enemy else BLUE, 0.1)
    if not enemy:
        cv.capsule((22, 27 + ub), (21.5 + arms[0] * 0.5, 30 + ub), 1.9, BLUE, 0.1)
    cv.ellipse(hand[0], hand[1] + 0.3, 1.5, 1.4, SKIN, 0.1)
    cv.outline()
    return cv.image()


IDLE_UP = [0, 0, 0, 1, 1, 1, 1, 0]
WALK_S = [3, 2, 0, -2, -3, -2, 0, 2]
WALK_LIFT = [0, 0, 0, 0, 0, 1, 2, 1]
WALK_BOB = [0, 1, 0, -1, 0, 1, 0, -1]


def walk_params(i):
    a, b = i, (i + 4) % 8
    return dict(bob=WALK_BOB[i], legs=((WALK_S[a], WALK_LIFT[a]), (WALK_S[b], WALK_LIFT[b])),
                arms=(-round(WALK_S[a] * 0.7), -round(WALK_S[b] * 0.6)))


def infantry_frames(enemy=False):
    fr = [infantry_frame(up=IDLE_UP[i], enemy=enemy) for i in range(8)]
    if not enemy:
        fr += [infantry_frame(**walk_params(i)) for i in range(8)]
    return fr


def horse_leg(cv, top, sx, lift, rp, bias):
    fx, fy = top[0] + sx, G - 1.5 - lift
    kx = (top[0] + fx) / 2 + (1.3 if lift else 0)
    ky = top[1] + (fy - top[1]) * 0.5 - lift * 0.5
    cv.capsule(top, (kx, ky), 1.9, rp, bias)
    cv.capsule((kx, ky), (fx, fy), 1.3, rp, bias - 0.1)
    cv.ellipse(fx + 0.4, G - 0.6 - lift, 1.7, 1.2, HAIR, 0.1)


HORSE_PHASE = (0, 4, 2, 6)  # near-front, far-front, far-hind, near-hind
H_S = [2, 1, 0, -1, -2, -1, 0, 1]
H_LIFT = [0, 0, 0, 0, 0, 1, 2, 1]


def cavalry_frame(i, walk):
    cv = Cv(CW, CH)
    if walk:
        bob = [0, 0, -1, -1, 0, 0, -1, -1][i]
        hs = [(H_S[(i + p) % 8], H_LIFT[(i + p) % 8]) for p in HORSE_PHASE]
        head = [0, 1, 1, 0, 0, 1, 1, 0][i]
        rider = bob
        tail = [0, 1, 1, 0, 0, -1, -1, 0][i]
    else:
        bob = 0
        hs = [(1, 0), (0, 0), (-1, 0), (-2, 0)]
        head = [0, 0, 0, 1, 1, 1, 0, 0][i]
        rider = IDLE_UP[i]
        tail = [0, 0, 1, 1, 0, 0, -1, -1][i]
    by = 31 + bob
    # tail
    cv.capsule((15, by - 1), (11 + tail, by + 9), 1.9, HAIR, 0.1)
    cv.capsule((12 + tail, by + 6), (10 + tail * 2, by + 13), 1.3, HAIR, 0.0)
    # far legs
    horse_leg(cv, (34, by + 4), hs[1][0], hs[1][1], HORSE, -0.35)
    horse_leg(cv, (19, by + 4), hs[2][0], hs[2][1], HORSE, -0.35)
    # body
    cv.ellipse(26, by + 1, 11.5, 5.8, HORSE, 0.05)
    cv.ellipse(18, by + 0.5, 5.6, 5.6, HORSE, 0.1)
    cv.ellipse(35, by + 0.5, 5.0, 5.4, HORSE, 0.0)
    # neck + head
    hy = head
    cv.capsule((35, by), (40, by - 8 + hy), 4.2, HORSE, 0.1)
    cv.capsule((40.5, by - 10 + hy), (46.5, by - 4 + hy), 2.6, HORSE, 0.1)
    cv.ellipse(42, by - 9.5 + hy, 3.0, 2.6, HORSE, 0.15)
    cv.ellipse(46.8, by - 3.8 + hy, 1.9, 1.7, DARK, 0.1)
    cv.capsule((40, by - 12 + hy), (40.5, by - 15 + hy), 1.0, HORSE, 0.2)
    cv.capsule((34, by - 4), (39, by - 12 + hy), 1.6, HAIR, 0.15)
    cv.put(43, int(by - 10 + hy), HAIR[0])
    cv.put(44, int(by - 10 + hy), HORSE[4])
    cv.put(47, int(by - 4 + hy), HAIR[0])
    # bridle
    cv.line((41, by - 8 + hy), (46, by - 5 + hy), RED[3])
    cv.line((40, by - 11 + hy), (41, by - 8 + hy), RED[2])
    # near legs
    horse_leg(cv, (36, by + 4), hs[0][0], hs[0][1], HORSE, 0.0)
    horse_leg(cv, (18, by + 4), hs[3][0], hs[3][1], HORSE, 0.05)
    # saddle cloth
    cv.poly([(20, by - 4), (32, by - 4), (31, by + 4), (21, by + 4)],
            lambda x, y: RED[3] if y < by - 1 else RED[2])
    for x in range(21, 32):
        cv.put(x, by + 4, GOLD[3] if x % 2 else GOLD[2])
    cv.put(26, by + 1, GOLD[4])
    cv.put(27, by + 1, GOLD[3])
    # rider
    ry = by - 1 + (rider if not walk else 0)
    cv.capsule((27, ry - 2), (30, ry + 5), 2.1, BLUE, 0.05)
    cv.ellipse(30.8, ry + 6, 2.0, 1.4, DARK, 0.0)
    tp = cv.ellipse(26.5, ry - 7.5, 5.0, 5.2, BLUE, 0.05)
    lamellar(cv, tp, BLUE, ry)
    for x in range(22, 31):
        cv.put(x, int(ry - 3), GOLD[2] if x != 26 else GOLD[4])
    cv.ellipse(22.5, ry - 10.5, 2.7, 2.2, BLUE, 0.2)
    helmet_head(cv, 27.5, ry - 14.5)
    # spear held across
    h1 = (31.5, ry - 8)
    h2 = (24.5, ry - 6)
    a, b = (13, ry + 1), (47, ry - 25)
    cv.line(a, b, WOOD[3])
    cv.line((a[0] + 1, a[1]), (b[0] + 1, b[1]), WOOD[1])
    cv.capsule((b[0] - 1, b[1] + 1.5), (b[0] + 2, b[1] - 1.5), 0.9, STEEL, 0.2)
    cv.put(int(b[0] + 2), int(b[1] - 2), STEEL[4])
    cv.ellipse(b[0] - 2.5, b[1] + 3.5, 1.4, 1.4, RED, 0.1)
    cv.capsule((24, ry - 11), h2, 1.6, BLUE, 0.1)
    cv.capsule((29, ry - 11), h1, 1.6, BLUE, 0.15)
    cv.ellipse(h1[0], h1[1], 1.4, 1.3, SKIN, 0.1)
    cv.ellipse(h2[0], h2[1], 1.4, 1.3, SKIN, 0.1)
    cv.outline()
    return cv.image()


def cavalry_frames():
    return [cavalry_frame(i, False) for i in range(8)] + [cavalry_frame(i, True) for i in range(8)]


# ---------------------------------------------------------------- portrait

PW, PH = 64, 72


def portrait():
    cv = Cv(PW, PH)
    # robe
    cv.poly([(3, 72), (8, 57), (20, 50), (44, 50), (56, 57), (61, 72)],
            lambda x, y: GREEN[3] if x < 20 else (GREEN[2] if x < 44 else GREEN[1]))
    for y in range(52, 72):
        for x in range(4, 60):
            c = cv.get(x, y)
            if c in GREEN and (x * 3 + y) % 11 == 0:
                cv.put(x, y, step(c, GREEN, -1))
    cv.poly([(22, 50), (32, 66), (42, 50), (39, 50), (32, 61), (25, 50)],
            lambda x, y: GOLD[3] if x < 32 else GOLD[2])
    cv.poly([(25, 50), (32, 61), (39, 50)], lambda x, y: IVORY[3] if x < 32 else IVORY[2])
    cv.capsule((10, 58), (20, 51), 2.0, GOLD, 0.1)
    cv.capsule((44, 51), (54, 58), 2.0, GOLD, -0.2)
    # neck
    cv.capsule((32, 42), (32, 51), 5.0, FACE_R, -0.25)
    # tails of headscarf
    cv.capsule((42, 14), (50, 30), 2.6, GREEN, -0.05)
    cv.capsule((44, 16), (53, 26), 2.0, GREEN, -0.15)
    # ear
    cv.ellipse(21.5, 33, 2.0, 3.0, FACE_R, 0.1)
    # face
    cv.ellipse(32.5, 32.5, 10.5, 12.5, FACE_R, 0.05)
    # headscarf
    cv.ellipse(31.5, 18.5, 12.6, 8.8, GREEN, 0.05)
    for x in range(20, 44):
        y = 24 if 22 < x < 42 else 23
        cv.put(x, y, GREEN[1] if x > 32 else GREEN[2])
    for x, y in [(26, 13), (27, 13), (28, 14), (24, 15)]:
        cv.put(x, y, GREEN[4])
    cv.ellipse(42.5, 15.5, 2.6, 2.4, GREEN, 0.0)
    # brows
    cv.line((24, 27), (30, 28), HAIR[0])
    cv.line((24, 26), (29, 27), HAIR[1])
    cv.line((35, 28), (41, 26), HAIR[0])
    cv.line((36, 27), (40, 25), HAIR[1])
    # phoenix eyes
    cv.line((25, 31), (30, 31), HAIR[0])
    cv.put(30, 30, HAIR[0])
    cv.line((36, 31), (40, 30), HAIR[0])
    cv.put(41, 29, HAIR[0])
    cv.put(27, 32, FACE_R[4])
    cv.put(28, 32, FACE_R[4])
    cv.put(38, 32, FACE_R[3])
    cv.put(28, 31, IVORY[4])
    cv.put(37, 31, IVORY[4])
    cv.put(29, 30, HAIR[0])
    # nose
    cv.line((33, 31), (34, 37), FACE_R[1])
    cv.put(32, 38, FACE_R[1])
    cv.put(34, 38, FACE_R[0])
    cv.put(31, 33, FACE_R[4])
    # beard
    beard = cv.poly([(24, 38), (29, 41), (36, 41), (41, 37), (42, 46), (39, 58), (34, 70),
                     (30, 64), (26, 50)],
                    lambda x, y: HAIR[2] if x < 33 else HAIR[1])
    for x, y in beard:
        if (x + (y // 4)) % 4 == 0:
            cv.put(x, y, HAIR[3] if x < 33 else HAIR[2])
    cv.poly([(26, 40), (32, 38), (38, 39), (40, 42), (36, 41), (32, 40), (28, 42)],
            lambda x, y: HAIR[1])
    cv.line((30, 42), (35, 42), FACE_R[0])
    cv.outline()
    return cv.image()


# ---------------------------------------------------------------- terrain

def h2(x, y, s=0):
    n = (x * 374761393 + y * 668265263 + s * 2147483647) & 0xFFFFFFFF
    n = (n ^ (n >> 13)) * 1274126177 & 0xFFFFFFFF
    return (n ^ (n >> 16)) & 0xFFFF


def vnoise(x, y, cell, s):
    gx, gy = x / cell, y / cell
    x0, y0 = int(gx), int(gy)
    fx, fy = gx - x0, gy - y0
    fx, fy = fx * fx * (3 - 2 * fx), fy * fy * (3 - 2 * fy)
    a = h2(x0, y0, s) / 65535
    b = h2(x0 + 1, y0, s) / 65535
    c = h2(x0, y0 + 1, s) / 65535
    d = h2(x0 + 1, y0 + 1, s) / 65535
    return (a * (1 - fx) + b * fx) * (1 - fy) + (c * (1 - fx) + d * fx) * fy


def load_stage():
    t = open(DOC, encoding="utf-8").read()
    i = t.index("# 부록 B")
    j = t.index("```json", i) + 7
    k = t.index("```", j)
    return json.loads(t[j:k])


def tree(cv, cx, by, size):
    cv.capsule((cx, by - 3), (cx, by - 7), 1.4, WOOD, 0.0)
    r = size
    cv.ellipse(cx + 0.5, by - 7 - r * 0.7, r + 1.2, r * 0.9 + 0.5, LEAF, -0.25)
    cv.ellipse(cx - 1, by - 9 - r * 0.9, r * 0.85, r * 0.75, LEAF, 0.05)
    cv.ellipse(cx + 2.5, by - 8 - r * 0.6, r * 0.6, r * 0.55, LEAF, 0.0)
    cv.ellipse(cx - 2, by - 11 - r, r * 0.5, r * 0.45, LEAF, 0.3)


def house(cv, ox, oy, roof):
    # ox, oy: tile origin
    wall = cv.poly([(ox + 8, oy + 24), (ox + 40, oy + 24), (ox + 40, oy + 42), (ox + 8, oy + 42)],
                   lambda x, y: IVORY[3] if x < ox + 30 else IVORY[2])
    for x in range(ox + 8, ox + 41):
        cv.put(x, oy + 41, WOOD[2])
        cv.put(x, oy + 42, WOOD[1])
    for y in range(oy + 24, oy + 43):
        cv.put(ox + 8, y, WOOD[2])
        cv.put(ox + 40, y, WOOD[1])
        cv.put(ox + 24, y, WOOD[2])
    for y in range(oy + 31, oy + 42):
        for x in range(ox + 14, ox + 19):
            cv.put(x, y, WOOD[1] if x != ox + 16 else WOOD[0])
    for y in range(oy + 29, oy + 34):
        for x in range(ox + 29, ox + 36):
            cv.put(x, y, DARK[1] if (x + y) % 2 else WOOD[2])
    rp = roof
    cv.poly([(ox + 3, oy + 26), (ox + 45, oy + 26), (ox + 40, oy + 10), (ox + 8, oy + 10)],
            lambda x, y: rp[3] if (y - oy) % 4 == 0 else (rp[2] if x < ox + 30 else rp[1]))
    for y in range(oy + 11, oy + 26):
        for x in range(ox + 4, ox + 45):
            c = cv.get(x, y)
            if c in rp and (x - ox) % 5 == 0 and (y - oy) % 4:
                cv.put(x, y, step(c, rp, -1))
    for x in range(ox + 7, ox + 42):
        cv.put(x, oy + 9, rp[4] if x < ox + 30 else rp[3])
        cv.put(x, oy + 10, rp[0])
        cv.put(x, oy + 26, rp[0])
    cv.put(ox + 6, oy + 8, rp[4])
    cv.put(ox + 42, oy + 8, rp[3])


def render_map(stage):
    W, H = stage["Width"], stage["Height"]
    tiles = stage["Tiles"]
    cv = Cv(W * TILE, H * TILE)

    def tt(tx, ty):
        if 0 <= tx < W and 0 <= ty < H:
            return tiles[ty][tx]
        return "."

    for y in range(H * TILE):
        for x in range(W * TILE):
            tx, ty = x // TILE, y // TILE
            t = tt(tx, ty)
            n = h2(x, y)
            if t == "~":
                lx, ly = x % TILE, y % TILE
                edge = min(lx if tt(tx - 1, ty) != "~" else 99,
                           TILE - 1 - lx if tt(tx + 1, ty) != "~" else 99,
                           ly if tt(tx, ty - 1) != "~" else 99,
                           TILE - 1 - ly if tt(tx, ty + 1) != "~" else 99)
                wob = (h2(x // 3, y // 3, 7) % 3)
                if edge < 2 + wob:
                    c = DIRT[3] if edge < 1 + wob else DIRT[1]
                elif edge < 4 + wob:
                    c = WATER[1]
                else:
                    c = WATER[2]
                    if (y % 6 == 0) and ((x // 4 + y // 6) % 5 == 0):
                        c = WATER[3]
                    elif (y % 6 == 1) and ((x // 4 + y // 6) % 5 == 0) and n % 2:
                        c = WATER[4]
                    elif n % 37 == 0:
                        c = WATER[1]
            elif t == "v":
                c = DIRT[2]
                if n % 9 == 0:
                    c = DIRT[1]
                elif n % 13 == 0:
                    c = DIRT[3]
            else:
                v = vnoise(x, y, 14, 3) * 0.7 + vnoise(x, y, 5, 4) * 0.3
                c = GRASS[1] if v < 0.2 else (GRASS[3] if v > 0.8 else GRASS[2])
                if n % 97 == 0:
                    c = GRASS[1]
                if t == "f":
                    c = step(c, GRASS, -1)
            cv.p[y][x] = c
    # plain tufts / flowers
    for ty in range(H):
        for tx in range(W):
            t = tt(tx, ty)
            ox, oy = tx * TILE, ty * TILE
            if t == ".":
                for k in range(5):
                    r = h2(tx, ty, 10 + k)
                    if r % 4 == 0:
                        continue
                    x, y = ox + 4 + r % 40, oy + 6 + (r >> 6) % 38
                    for dx, dy, c in ((0, 0, GRASS[4]), (-1, 1, GRASS[3]), (1, 1, GRASS[3]),
                                      (0, 1, GRASS[1]), (-2, 2, GRASS[3]), (2, 2, GRASS[3])):
                        cv.put(x + dx, y + dy, c)
                r = h2(tx, ty, 20)
                if r % 5 == 0:
                    x, y = ox + 6 + r % 34, oy + 8 + (r >> 5) % 32
                    for dx, dy in ((0, 0), (2, 1), (1, 3)):
                        cv.put(x + dx, y + dy, IVORY[4] if (r >> 3) % 2 else YEL[3])
                if r % 11 == 1:
                    x, y = ox + 10 + r % 26, oy + 12 + (r >> 4) % 24
                    cv.ellipse(x, y, 2.6, 1.8, STEEL, 0.0)
                    cv.put(int(x) - 2, int(y) + 2, GRASS[0])
            elif t == "v":
                pass
    # village path along column x=2 and fences
    for ty in range(H):
        for tx in range(W):
            if tt(tx, ty) != "v":
                continue
            ox, oy = tx * TILE, ty * TILE
            if tt(tx - 1, ty) != "v" or tt(tx + 1, ty) != "v":
                side = 0 if tt(tx - 1, ty) != "v" else TILE - 2
                for y in range(oy + 2, oy + TILE - 2, 3):
                    cv.put(ox + side, y, WOOD[2])
                    cv.put(ox + side, y + 1, WOOD[1])
                    cv.put(ox + side + 1, y, WOOD[3])
    objs = []
    for ty in range(H):
        for tx in range(W):
            t = tt(tx, ty)
            ox, oy = tx * TILE, ty * TILE
            if t == "f":
                spots = [(14, 22, 7), (34, 18, 6), (24, 40, 8), (40, 42, 6), (8, 42, 5)]
                for i, (sx, sy, s) in enumerate(spots):
                    r = h2(tx, ty, 30 + i)
                    objs.append((oy + sy, "tree", ox + sx + r % 3 - 1, oy + sy, s + (r >> 4) % 2))
            elif t == "v" and tx != 2:
                if (ty - 5) % 2 == 0:
                    objs.append((oy + 42, "house", ox, oy, SLATE if (tx + ty) % 4 else RED))
                else:
                    objs.append((oy + 30, "stack", ox, oy))
    objs.sort(key=lambda o: o[0])
    for o in objs:
        if o[1] == "tree":
            tree(cv, o[2], o[3], o[4])
        elif o[1] == "house":
            house(cv, o[2], o[3], o[4])
        else:
            ox, oy = o[2], o[3]
            cv.ellipse(ox + 16, oy + 28, 7, 5, YEL, -0.1)
            cv.ellipse(ox + 16, oy + 24, 5, 4, YEL, 0.15)
            cv.ellipse(ox + 33, oy + 30, 4.5, 4, WOOD, 0.0)
            cv.ellipse(ox + 33, oy + 30, 2.5, 2, WATER, 0.1)
    return cv.image()


# ---------------------------------------------------------------- sheet

def build_sheet():
    rows = [
        ("infantry-ally", infantry_frames(False), [("idle", 0, 140), ("walk", 8, 100)]),
        ("cavalry-ally", cavalry_frames(), [("idle", 0, 150), ("walk", 8, 110)]),
        ("bandit-enemy", infantry_frames(True), [("idle", 0, 140)]),
    ]
    cols = max(len(r[1]) for r in rows)
    sheet = Image.new("RGBA", (cols * CW, len(rows) * CH), (0, 0, 0, 0))
    meta = {"cell": [CW, CH], "anchor": [AX, AY], "displayScale": 1, "facing": "right (SE); mirror around anchor x for left",
            "sprites": []}
    for ri, (sid, frames, anims) in enumerate(rows):
        for ci, f in enumerate(frames):
            sheet.paste(f, (ci * CW, ri * CH))
        meta["sprites"].append({"id": sid, "row": ri,
                                "anims": [{"name": n, "start": s, "frames": 8, "ms": ms} for n, s, ms in anims]})
    return sheet, meta, rows


def check_sheet(sheet, meta):
    px = sheet.load()
    problems = []
    nframes = 0
    for sp in meta["sprites"]:
        r = sp["row"]
        for an in sp["anims"]:
            for c in range(an["start"], an["start"] + an["frames"]):
                nframes += 1
                ox, oy = c * CW, r * CH
                low, edge = -1, False
                for y in range(CH):
                    for x in range(CW):
                        a = px[ox + x, oy + y][3]
                        if a not in (0, 255):
                            problems.append(f"{sp['id']}#{c} alpha {a}")
                        if a:
                            low = max(low, y)
                            if x in (0, CW - 1) or y in (0, CH - 1):
                                edge = True
                if low != AY:
                    problems.append(f"{sp['id']}#{c} lowest row {low} != {AY}")
                if edge:
                    problems.append(f"{sp['id']}#{c} touches cell edge")
    return nframes, problems


# ---------------------------------------------------------------- screen

NAVY = (16, 22, 38)
PANEL = (22, 30, 52)
PANEL2 = (30, 40, 68)
GOLDL = (214, 176, 92)
GOLDD = (120, 92, 44)
TXT = (236, 228, 204)
TXT2 = (170, 168, 160)


def font(sz):
    return ImageFont.truetype(FONT, sz)


def frame_box(d, box, fill=PANEL, title=None):
    x0, y0, x1, y1 = box
    d.rectangle(box, fill=fill)
    d.rectangle(box, outline=GOLDD)
    d.rectangle((x0 + 2, y0 + 2, x1 - 2, y1 - 2), outline=GOLDL)
    for cx, cy in ((x0, y0), (x1, y0), (x0, y1), (x1, y1)):
        d.rectangle((cx - 3, cy - 3, cx + 3, cy + 3), fill=GOLDL, outline=GOLDD)
    if title:
        d.text((x0 + 12, y0 + 7), title, font=font(15), fill=GOLDL)


def mirror(im):
    out = Image.new("RGBA", im.size, (0, 0, 0, 0))
    src, dst = im.load(), out.load()
    for y in range(im.height):
        for x in range(im.width):
            nx = 2 * AX - x
            if 0 <= nx < im.width:
                dst[nx, y] = src[x, y]
    return out


def gray(im):
    out = im.copy()
    p = out.load()
    for y in range(out.height):
        for x in range(out.width):
            r, g, b, a = p[x, y]
            if a:
                l = int(0.3 * r + 0.59 * g + 0.11 * b)
                l = 40 + (l * 150) // 255
                p[x, y] = (l, l, l + 8, 255)
    return out


def flag(team):
    cv = Cv(14, 24)
    cv.line((2, 2), (2, 22), WOOD[2])
    cv.line((3, 2), (3, 22), WOOD[1])
    rp = BLUE if team == "ally" else RED
    cv.poly([(4, 2), (12, 3), (11, 7), (12, 11), (4, 11)], lambda x, y: rp[3] if y < 6 else rp[2])
    cv.put(7, 6, GOLD[4] if team == "ally" else YEL[4])
    cv.put(8, 6, GOLD[3] if team == "ally" else YEL[3])
    cv.put(7, 7, GOLD[3] if team == "ally" else YEL[3])
    cv.put(2, 1, GOLD[3])
    cv.outline()
    return cv.image()


def bar(d, x, y, w, label, cur, mx, col, fnt):
    d.text((x, y - 2), label, font=fnt, fill=TXT2)
    bx = x + 52
    d.rectangle((bx, y + 4, bx + w, y + 14), fill=(10, 12, 20), outline=GOLDD)
    fw = int((w - 2) * cur / mx) if mx else 0
    if fw:
        d.rectangle((bx + 1, y + 5, bx + fw, y + 13), fill=col)
        d.line((bx + 1, y + 5, bx + fw, y + 5), fill=tuple(min(255, c + 60) for c in col))
    d.text((bx + w + 8, y - 2), f"{cur} / {mx}", font=fnt, fill=TXT)


def build_screen(data, rows, port):
    stage = data["Stages"][0]
    W, H = stage["Width"], stage["Height"]
    tiles = stage["Tiles"]
    scr = Image.new("RGBA", (1280, 850), NAVY + (255,))
    d = ImageDraw.Draw(scr)
    MX, MY = 16, 60
    mapim = render_map(stage)
    # map frame
    frame_box(d, (MX - 8, MY - 8, MX + W * TILE + 7, MY + H * TILE + 7), fill=NAVY)
    scr.alpha_composite(mapim, (MX, MY))
    ov = Image.new("RGBA", scr.size, (0, 0, 0, 0))
    od = ImageDraw.Draw(ov)
    for i in range(W + 1):
        od.line((MX + i * TILE, MY, MX + i * TILE, MY + H * TILE), fill=(0, 0, 0, 34))
    for j in range(H + 1):
        od.line((MX, MY + j * TILE, MX + W * TILE, MY + j * TILE), fill=(0, 0, 0, 34))

    allies = [("유비", 2, 5, "inf", True), ("관우", 2, 6, "cav", False), ("장비", 2, 7, "cav", False)]
    enemies = [(e["Officer"], e["X"], e["Y"]) for e in stage["Enemies"]] + [("B01_scout", 7, 6)]
    occ_enemy = {(x, y) for _, x, y in enemies}
    occ_ally = {(x, y) for _, x, y, _, _ in allies}
    # move range for 관우 (경기병, move 5)
    cost = data["Terrain"]["기병"]
    sx, sy = 2, 6
    best = {(sx, sy): 0}
    frontier = [(sx, sy)]
    while frontier:
        nf = []
        for x, y in frontier:
            for ddx, ddy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                nx, ny = x + ddx, y + ddy
                if not (0 <= nx < W and 0 <= ny < H) or (nx, ny) in occ_enemy:
                    continue
                c = cost[tiles[ny][nx]]["Cost"]
                if c <= 0:
                    continue
                v = best[(x, y)] + c
                if v <= 5 and v < best.get((nx, ny), 99):
                    best[(nx, ny)] = v
                    nf.append((nx, ny))
        frontier = nf
    for (x, y) in best:
        if (x, y) in occ_ally and (x, y) != (sx, sy):
            continue
        X, Y = MX + x * TILE, MY + y * TILE
        od.rectangle((X + 1, Y + 1, X + TILE - 2, Y + TILE - 2), fill=(70, 130, 230, 82), outline=(150, 200, 255, 150))
    tgt = (7, 6)
    X, Y = MX + tgt[0] * TILE, MY + tgt[1] * TILE
    od.rectangle((X + 1, Y + 1, X + TILE - 2, Y + TILE - 2), fill=(230, 60, 50, 96), outline=(255, 140, 120, 220))
    od.rectangle((X + 3, Y + 3, X + TILE - 4, Y + TILE - 4), outline=(255, 140, 120, 120))
    scr.alpha_composite(ov)

    inf, cav, ban = rows[0][1], rows[1][1], rows[2][1]
    fa, fe = flag("ally"), flag("enemy")
    draws = []
    for name, x, y, kind, done in allies:
        im = (inf[0] if kind == "inf" else cav[(x + y) % 3])
        draws.append((y, x, gray(im) if done else im, fa, False))
    for name, x, y in enemies:
        im = mirror(ban[(x * 3 + y) % 8])
        draws.append((y, x, im, fe, name in ("정원지", "등무", "장각")))
    draws.sort(key=lambda t: (t[0], t[1]))
    for y, x, im, fl, boss in draws:
        bx, by = MX + x * TILE + TILE // 2, MY + y * TILE + TILE - 6
        shadow = Image.new("RGBA", scr.size, (0, 0, 0, 0))
        ImageDraw.Draw(shadow).ellipse((bx - 12, by - 4, bx + 12, by + 3), fill=(0, 0, 0, 70))
        scr.alpha_composite(shadow)
        fx = bx - 22 if fl is fa else bx + 10
        scr.alpha_composite(fl, (fx, by - 40))
        scr.alpha_composite(im, (bx - AX, by - AY))
        if boss:
            d.polygon([(fx + 7, by - 46), (fx + 10, by - 43), (fx + 7, by - 40), (fx + 4, by - 43)],
                      fill=GOLDL, outline=(60, 40, 10))
    # cursor on 관우
    cx0, cy0 = MX + sx * TILE, MY + sy * TILE
    for (ax, ay, bx, by) in ((0, 0, 1, 0), (0, 0, 0, 1)):
        pass
    L = 12
    for (px0, py0, dx, dy) in ((cx0, cy0, 1, 1), (cx0 + TILE - 1, cy0, -1, 1), (cx0, cy0 + TILE - 1, 1, -1),
                               (cx0 + TILE - 1, cy0 + TILE - 1, -1, -1)):
        for w in range(3):
            d.line((px0 + dx * w, py0 + dy * w, px0 + dx * L, py0 + dy * w), fill=GOLDL if w < 2 else GOLDD)
            d.line((px0 + dx * w, py0 + dy * w, px0 + dx * w, py0 + dy * L), fill=GOLDL if w < 2 else GOLDD)
    # damage number on target
    f22 = font(24)
    tx, ty = MX + tgt[0] * TILE + 8, MY + tgt[1] * TILE - 22
    d.text((tx, ty), "-128", font=f22, fill=(255, 236, 160), stroke_width=3, stroke_fill=(70, 20, 10))
    # action menu next to 관우
    mx0, my0 = cx0 + TILE + 14, cy0 - 40
    items = ["이동", "공격", "병법", "아이템", "대기", "일기토"]
    frame_box(d, (mx0, my0, mx0 + 112, my0 + 22 + len(items) * 28), fill=(18, 24, 44))
    f16 = font(16)
    for i, it in enumerate(items):
        iy = my0 + 12 + i * 28
        if it == "공격":
            d.rectangle((mx0 + 7, iy - 2, mx0 + 105, iy + 24), fill=(120, 40, 36), outline=GOLDL)
            d.polygon([(mx0 + 12, iy + 6), (mx0 + 18, iy + 11), (mx0 + 12, iy + 16)], fill=GOLDL)
        col = TXT if it not in ("일기토",) else TXT2
        d.text((mx0 + 26, iy), it, font=f16, fill=col)
    # top bar
    f20 = font(20)
    d.text((MX, 16), "황건적의 난", font=f20, fill=GOLDL)
    d.text((MX + 128, 20), "1턴 · 아군 진영", font=f16, fill=(150, 196, 255))
    d.text((MX + 270, 20), "목표: 장각 격퇴", font=f16, fill=TXT)
    d.text((MX + 420, 20), "패배: 유비 전투불능", font=f16, fill=TXT2)
    # right column
    RX = 900
    frame_box(d, (RX, 16, 1264, 144))
    mm_s = 6
    mmx, mmy = RX + 16, 44
    colors = {".": (78, 124, 58), "f": (36, 80, 40), "v": (150, 112, 70), "~": (44, 96, 150)}
    for y in range(H):
        for x in range(W):
            d.rectangle((mmx + x * mm_s, mmy + y * mm_s, mmx + x * mm_s + mm_s - 1, mmy + y * mm_s + mm_s - 1),
                        fill=colors[tiles[y][x]])
    for _, x, y, _, _ in allies:
        d.rectangle((mmx + x * mm_s + 1, mmy + y * mm_s + 1, mmx + x * mm_s + 4, mmy + y * mm_s + 4), fill=(110, 170, 255))
    for _, x, y in enemies:
        d.rectangle((mmx + x * mm_s + 1, mmy + y * mm_s + 1, mmx + x * mm_s + 4, mmy + y * mm_s + 4), fill=(240, 80, 60))
    d.rectangle((mmx - 1, mmy - 1, mmx + W * mm_s, mmy + H * mm_s), outline=GOLDL)
    f14 = font(14)
    info = [("아군", "3부대"), ("적군", f"{len(enemies)}부대"), ("라운드", "1 / 20"), ("목표", "장각 격퇴")]
    for i, (k, v) in enumerate(info):
        d.text((RX + 150, 34 + i * 26), k, font=f14, fill=TXT2)
        d.text((RX + 214, 34 + i * 26), v, font=f14, fill=TXT)

    frame_box(d, (RX, 156, 1264, 618))
    pscale = 3
    pim = port.resize((port.width * pscale, port.height * pscale), Image.NEAREST)
    pbx, pby = RX + 14, 172
    d.rectangle((pbx, pby, pbx + pim.width + 4, pby + pim.height + 4), fill=(34, 46, 76), outline=GOLDL)
    for i in range(pim.height):
        d.line((pbx + 2, pby + 2 + i, pbx + pim.width + 2, pby + 2 + i), fill=(26 + i // 12, 36 + i // 10, 62 + i // 8))
    scr.alpha_composite(pim, (pbx + 2, pby + 2))
    tx0 = pbx + pim.width + 16
    d.text((tx0, 178), "관우", font=font(28), fill=TXT)
    d.text((tx0, 216), "운장", font=f14, fill=TXT2)
    d.text((tx0, 242), "경기병", font=f16, fill=GOLDL)
    d.text((tx0, 266), "Lv 1", font=f20, fill=TXT)
    d.text((tx0, 300), "아군", font=f14, fill=(150, 196, 255))
    d.text((tx0, 330), "지형: 마을", font=f14, fill=TXT2)
    d.text((tx0, 350), "효과 100%", font=f14, fill=TXT2)
    by0 = 412
    bar(d, RX + 18, by0, 170, "병력", 318, 318, (90, 190, 110), f14)
    bar(d, RX + 18, by0 + 24, 170, "병법치", 9, 18, (90, 140, 230), f14)
    bar(d, RX + 18, by0 + 48, 170, "경험치", 0, 100, (224, 180, 70), f14)
    stats = data["Officers"]["관우"]["Stats"]
    names = ["통솔", "무력", "지력", "민첩", "운", "매력"]
    for i, (n, v) in enumerate(zip(names, stats)):
        cx = RX + 18 + (i % 3) * 116
        cy = by0 + 84 + (i // 3) * 26
        d.text((cx, cy), n, font=f14, fill=TXT2)
        d.text((cx + 40, cy), str(v), font=f14, fill=TXT)
    d.text((RX + 18, by0 + 146), "특성", font=f14, fill=GOLDL)
    d.text((RX + 64, by0 + 146), "호걸 · 반격술", font=f14, fill=TXT)
    eq = data["Equipment"]
    eqn = " · ".join(eq[e]["Name"] for e in data["Officers"]["관우"]["Equipment"] if e in eq)
    d.text((RX + 18, by0 + 170), "장비", font=f14, fill=GOLDL)
    d.text((RX + 64, by0 + 170), eqn, font=f14, fill=TXT)

    frame_box(d, (RX, 630, 1264, 834), title="전투 예측")
    d.text((RX + 18, 658), "관우  →  황건 적병", font=f16, fill=TXT)
    rows_p = [("예상 피해", "128"), ("명중", "92%"), ("반격", "없음"), ("지형", "평지 100%")]
    for i, (k, v) in enumerate(rows_p):
        d.text((RX + 18, 686 + i * 22), k, font=f14, fill=TXT2)
        d.text((RX + 120, 686 + i * 22), v, font=f14, fill=TXT)
    d.text((RX + 18, 800), "Space 확정 · Esc 취소", font=f14, fill=TXT2)

    frame_box(d, (MX - 8, 750, 880, 834))
    d.text((MX + 8, 762), "장비", font=f16, fill=GOLDL)
    d.text((MX + 60, 762), "형님, 저 황건 놈들은 제가 먼저 쓸어버리겠소!", font=f16, fill=TXT)
    d.text((MX + 8, 792), "유비 행동 완료 · 관우 선택 중", font=f14, fill=TXT2)
    return scr.convert("RGB")


def x4(im, k=4):
    return im.resize((im.width * k, im.height * k), Image.NEAREST)


def main():
    os.makedirs(OUT, exist_ok=True)
    data = load_stage()
    sheet, meta, rows = build_sheet()
    sheet.save(os.path.join(OUT, "sheet.png"))
    with open(os.path.join(OUT, "sheet.json"), "w", encoding="utf-8") as f:
        json.dump(meta, f, ensure_ascii=False, indent=2)
    bg = Image.new("RGBA", sheet.size, (40, 46, 60, 255))
    gpx = bg.load()
    for y in range(bg.height):
        for x in range(bg.width):
            if (x // CW + y // CH) % 2:
                gpx[x, y] = (52, 58, 74, 255)
            if y % CH == AY:
                gpx[x, y] = (120, 60, 60, 255)
            if x % CW == AX:
                gpx[x, y] = (60, 90, 120, 255)
    bg.alpha_composite(sheet)
    x4(bg).save(os.path.join(OUT, "sheet-x4.png"))
    port = portrait()
    port.save(os.path.join(OUT, "portrait.png"))
    with open(os.path.join(OUT, "portrait.json"), "w", encoding="utf-8") as f:
        json.dump({"size": [PW, PH], "displayScale": 3}, f)
    x4(port).save(os.path.join(OUT, "portrait-x4.png"))
    build_screen(data, rows, port).save(os.path.join(OUT, "screen.png"))
    n, probs = check_sheet(sheet, meta)
    alpha_p = {a for a in port.getdata(3)} if False else {p[3] for p in port.getdata()}
    print(f"frames checked: {n}; problems: {len(probs)}")
    for p in probs[:20]:
        print("  ", p)
    print("portrait alpha values:", sorted(alpha_p))
    colors = set()
    for p in sheet.getdata():
        if p[3]:
            colors.add(p[:3])
    print("sheet colors:", len(colors))


if __name__ == "__main__":
    main()
