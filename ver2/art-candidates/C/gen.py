"""Graphic candidate C — isometric. Run from ver2/: python3 art-candidates/C/gen.py"""
import json
import os
import sys
from collections import deque

from PIL import Image, ImageDraw, ImageFont

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import sprites as S  # noqa: E402

OUT = os.path.join(HERE, "out")
ROOT = os.path.abspath(os.path.join(HERE, "..", "..", ".."))  # mhdemo1
FONT = os.path.join(ROOT, "ver1", "assets", "fonts", "NotoSansKR.ttf")
OBJECTS = os.path.join(ROOT, "assets", "objects.png")

TW, TH = 32, 16          # big tile diamond
EDGE = 7                 # map slab thickness
SCALE = 2                # map display scale
SPRITE_SCALE = 3         # sheet/preview display scale

B01 = [
    "..................",
    "..................",
    ".........f........",
    ".........f........",
    ".........f........",
    ".vvv..............",
    ".vvv..............",
    ".vvv..............",
    ".vvv..............",
    "..................",
    "..................",
    ".........~........",
    ".........~........",
    ".........~........",
]
MW, MH = 18, 14


def h32(*v):
    x = 0x9E3779B9
    for n in v:
        x = (x ^ (n & 0xFFFFFFFF)) * 0x85EBCA6B & 0xFFFFFFFF
        x ^= x >> 13
        x = x * 0xC2B2AE35 & 0xFFFFFFFF
        x ^= x >> 16
    return x


def rgb(h):
    return tuple(int(h[i:i + 2], 16) for i in (1, 3, 5)) + (255,)


T = {
    "grass": [rgb("#6f9a48"), rgb("#7fab52"), rgb("#5e8740"), rgb("#93bb5e")],
    "dirt": [rgb("#b99863"), rgb("#c8a873"), rgb("#a3845a"), rgb("#d7bb85")],
    "water": [rgb("#2f6f86"), rgb("#3b8299"), rgb("#28607a"), rgb("#8cc6cf")],
    "edge": [rgb("#6b4a33"), rgb("#523726"), rgb("#3c281d")],
    "line": (20, 30, 24, 46),
}


def diamond_rows():
    rows = []
    for y in range(TH):
        half = 2 * (y + 1) if y < TH // 2 else 2 * (TH - y)
        rows.append((TW // 2 - half, TW // 2 + half))
    return rows


DROWS = diamond_rows()


def tile_image(kind, u, v):
    im = Image.new("RGBA", (TW, TH), (0, 0, 0, 0))
    pal = T["water"] if kind == "~" else (T["dirt"] if kind == "v" else T["grass"])
    for y, (x0, x1) in enumerate(DROWS):
        for x in range(x0, x1):
            n = h32(u, v, x, y) % 100
            c = pal[0]
            if kind == "~":
                wave = (x + 2 * y + (u - v) * 6) % 11
                c = pal[3] if (wave == 0 and n < 70) else (pal[1] if wave in (1, 2) else pal[0])
                if n < 6:
                    c = pal[2]
            else:
                # clustered speckle: 2px clusters from coarse hash
                m = h32(u, v, x // 2, y) % 100
                if m < 14:
                    c = pal[1]
                elif m < 22:
                    c = pal[2]
                elif m < 25 and kind != "v":
                    c = pal[3]
            im.putpixel((x, y), c)
    return im


def load_objects():
    sheet = Image.open(OBJECTS).convert("RGBA")
    cell = lambda c, r: sheet.crop((c * 40, r * 48, c * 40 + 40, r * 48 + 48))
    trees = [cell(c, 1) for c in range(3)]
    tufts = [cell(c, 0) for c in range(4, 8)] + [cell(c, 1) for c in range(3, 8)]
    rocks = [cell(c, 0) for c in range(4)]
    return trees, tufts, rocks


def house():
    """Small isometric thatched house, 28x24, pivot at bottom center (14, 22)."""
    P = {
        "k": S.PAL["k"], "w": rgb("#e3d5b3")[:3], "W": rgb("#b8a27c")[:3],
        "r": rgb("#a8573c")[:3], "R": rgb("#7a3a2a")[:3], "d": rgb("#4a3020")[:3],
        "t": rgb("#d2a95a")[:3], "T": rgb("#9a7438")[:3],
    }
    g = [
        "............kk..............",
        "..........kkRRkk............",
        "........kkRRRRrrkk..........",
        "......kkRRRRRRrrrrkk........",
        "....kkRRRRRRRRrrrrrrkk......",
        "..kkRRRRRRRRRRrrrrrrrrkk....",
        "kkRRRRRRRRRRRRrrrrrrrrrrkk..",
        "kRRRRRRRRRRRRRrrrrrrrrrrrrk.",
        ".kRRRRRRRRRRRRrrrrrrrrrrrk..",
        "..kkRRRRRRRRRRrrrrrrrrrkk...",
        "...kWkkRRRRRRRrrrrrrkkwk....",
        "...kWWWkkRRRRRrrrrkkwwwk....",
        "...kWWWWWkkRRRrrkkwwwwwk....",
        "...kWWWWWWWkkRkkwwwwwwwk....",
        "...kWdWWWWWWWkwwwwwddwwk....",
        "...kWdWWWWWWWkwwwwwddwwk....",
        "...kWWWWWWWWWkwwddwwwwwk....",
        "...kWWWWWWWWWkwwddwwwwwk....",
        "...kkWWWWWWWWkwwddwwwkk.....",
        ".....kkWWWWWWkwwddwkk.......",
        ".......kkWWWWkwwwkk.........",
        ".........kkWWkwkk...........",
        "...........kkkk.............",
    ]
    im = Image.new("RGBA", (28, 24), (0, 0, 0, 0))
    for y, row in enumerate(g):
        for x, ch in enumerate(row):
            if ch != ".":
                im.putpixel((x, y), P[ch] + (255,))
    return im, (14, 22)


# ---- portrait --------------------------------------------------------------
def portrait():
    W = H = 48
    P = {
        "k": (22, 18, 26), "f": (196, 74, 56), "F": (150, 48, 40), "x": (226, 112, 84),
        "h": (30, 26, 30), "H": (58, 50, 56), "g": (56, 128, 84), "G": (32, 84, 58),
        "j": (110, 172, 116), "a": (60, 66, 86), "A": (98, 106, 128), "o": (226, 184, 82),
        "e": (250, 240, 220), "r": (150, 40, 44),
    }
    px = {}

    def ell(cx, cy, rx, ry, ch):
        for y in range(cy - ry, cy + ry + 1):
            for x in range(cx - rx, cx + rx + 1):
                if ((x - cx) / (rx + 0.5)) ** 2 + ((y - cy) / (ry + 0.5)) ** 2 <= 1:
                    px[(x, y)] = ch

    def rect(x0, y0, x1, y1, ch):
        for y in range(y0, y1 + 1):
            for x in range(x0, x1 + 1):
                px[(x, y)] = ch

    # shoulders: green robe over dark armor
    ell(24, 47, 21, 11, "G")
    ell(24, 46, 18, 9, "g")
    for x in range(10, 40, 3):
        px[(x, 44)] = "j"
    rect(17, 38, 31, 47, "a")
    for y in range(39, 47, 2):
        for x in range(18, 31, 2):
            px[(x, y)] = "A"
    rect(22, 36, 27, 39, "o")
    # neck + face (3/4 facing right)
    rect(20, 31, 28, 36, "F")
    ell(26, 24, 9, 11, "f")
    # shadow side (left/back of face)
    for y in range(16, 34):
        for x in range(16, 21):
            if px.get((x, y)) == "f":
                px[(x, y)] = "F"
    # cheek/forehead highlights
    for (x, y) in [(30, 22), (31, 22), (31, 23), (32, 23), (29, 17), (30, 17), (31, 18), (33, 21)]:
        px[(x, y)] = "x"
    # green headscarf (wrapped turban)
    ell(25, 11, 11, 6, "g")
    rect(14, 12, 35, 15, "g")
    for x in range(14, 36):
        px[(x, 16)] = "G"
    for x in range(16, 34, 2):
        px[(x, 14)] = "G"
    for (x, y) in [(19, 8), (23, 7), (27, 7), (31, 9), (21, 10), (26, 10), (33, 12), (18, 12)]:
        px[(x, y)] = "j"
    # knot and tails at back
    ell(13, 14, 3, 3, "g")
    px[(13, 13)] = "j"
    rect(9, 17, 12, 30, "g")
    rect(7, 20, 8, 28, "G")
    for y in range(18, 31, 3):
        px[(11, y)] = "j"
        px[(10, y + 1)] = "G"
    # hair at temple
    rect(17, 17, 19, 23, "h")
    # ear
    rect(20, 21, 21, 25, "F")
    px[(21, 23)] = "f"
    # heavy brows (rising toward temple), phoenix eye
    for (x, y) in [(24, 19), (25, 19), (26, 18), (27, 18), (28, 18), (29, 18), (31, 19), (32, 19), (33, 19)]:
        px[(x, y)] = "h"
    for (x, y, c) in [(25, 21, "h"), (26, 21, "h"), (27, 21, "e"), (28, 21, "h"), (29, 20, "h"),
                      (32, 21, "h"), (33, 21, "e"), (34, 21, "h")]:
        px[(x, y)] = c
    # nose
    for (x, y) in [(32, 23), (33, 24), (34, 25), (33, 26), (32, 26)]:
        px[(x, y)] = "F"
    # moustache
    for (x, y) in [(27, 28), (28, 28), (29, 28), (30, 28), (31, 28), (32, 28), (33, 28), (26, 29), (34, 29), (35, 30)]:
        px[(x, y)] = "h"
    px[(30, 29)] = "r"
    px[(31, 29)] = "r"
    # sideburns into beard
    for y in range(24, 31):
        px[(22, y)] = "h"
        px[(23, y + 1)] = "h"
    # long beard flowing down past the chest, tapering
    for y in range(30, 47):
        if y < 36:
            w = 10
        elif y < 41:
            w = 9
        else:
            w = 9 - (y - 40)
        cx = 29 + (y - 30) // 5
        for x in range(cx - w // 2, cx + (w + 1) // 2):
            if y == 30 and 29 <= x <= 32:
                continue
            px[(x, y)] = "h"
    for y in range(31, 46):
        cx = 29 + (y - 30) // 5
        px[(cx - 2 + (y % 3), y)] = "H"
        if y % 2:
            px[(cx + 2, y)] = "H"
    # outline
    full = dict(px)
    for (x, y) in px:
        for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            q = (x + dx, y + dy)
            if q not in px:
                full[q] = "k"
    im = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    for (x, y), ch in full.items():
        if 0 <= x < W and 0 <= y < H:
            im.putpixel((x, y), P[ch] + (255,))
    return im


# ---- sprite sheet ----------------------------------------------------------
ENEMY_RECOLOR = {"b": "r", "B": "R", "n": "D", "r": "y", "R": "Y"}


def recolored(px, mapping):
    return {k: mapping.get(v, v) for k, v in px.items()}


def build_sheet():
    rows = [
        ("infantry-ally", [S.footman(("idle", i)) for i in range(8)] + [S.footman(("walk", i)) for i in range(8)],
         [("idle", 0, 8, 150), ("walk", 8, 8, 100)]),
        ("cavalry-ally", [S.cavalry(("idle", i)) for i in range(8)] + [S.cavalry(("walk", i)) for i in range(8)],
         [("idle", 0, 8, 150), ("walk", 8, 8, 110)]),
        ("bandit-enemy", [S.footman(("idle", i), "enemy") for i in range(8)], [("idle", 0, 8, 150)]),
        ("officer-enemy", [recolored(S.footman(("idle", i)), ENEMY_RECOLOR) for i in range(8)], [("idle", 0, 8, 150)]),
        ("cavalry-enemy", [recolored(S.cavalry(("idle", i)), ENEMY_RECOLOR) for i in range(8)], [("idle", 0, 8, 150)]),
    ]
    sheet = Image.new("RGBA", (16 * S.CELL_W, len(rows) * S.CELL_H), (0, 0, 0, 0))
    meta = {"cell": [S.CELL_W, S.CELL_H], "anchor": list(S.ANCHOR), "displayScale": SPRITE_SCALE,
            "mapDisplayScale": SCALE, "sprites": []}
    frames = {}
    for r, (sid, fr, anims) in enumerate(rows):
        frames[sid] = [S.to_image(px) for px in fr]
        for c, im in enumerate(frames[sid]):
            sheet.alpha_composite(im, (c * S.CELL_W, r * S.CELL_H))
        meta["sprites"].append({"id": sid, "row": r, "anims": [
            {"name": n, "start": st, "frames": f, "ms": ms} for n, st, f, ms in anims]})
    return sheet, meta, frames


def check_sheet(sheet, meta):
    cw, ch = meta["cell"]
    ax, ay = meta["anchor"]
    problems = []
    a = sheet.split()[3]
    vals = set(a.getdata())
    if not vals <= {0, 255}:
        problems.append(f"alpha values {sorted(vals)[:6]}")
    for s in meta["sprites"]:
        n = sum(an["frames"] for an in s["anims"])
        for c in range(n):
            cell = a.crop((c * cw, s["row"] * ch, (c + 1) * cw, (s["row"] + 1) * ch))
            bb = cell.getbbox()
            if bb is None:
                problems.append(f"{s['id']}#{c} empty")
                continue
            x0, y0, x1, y1 = bb
            if x0 == 0 or y0 == 0 or x1 == cw or y1 == ch:
                problems.append(f"{s['id']}#{c} touches edge {bb}")
            if y1 - 1 != ay:
                problems.append(f"{s['id']}#{c} lowest row {y1 - 1} != anchor {ay}")
    return problems


# ---- battle screen ---------------------------------------------------------
def iso(u, v, ox, oy):
    return ox + (u - v) * (TW // 2), oy + (u + v) * (TH // 2)


def move_range(start, move, blocked, allies):
    cost = {".": 1, "f": 3, "v": 1}
    best = {start: 0}
    q = deque([start])
    while q:
        u, v = q.popleft()
        for du, dv in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            nu, nv = u + du, v + dv
            if not (0 <= nu < MW and 0 <= nv < MH):
                continue
            t = B01[nv][nu]
            if t not in cost or (nu, nv) in blocked:
                continue
            c = best[(u, v)] + cost[t]
            if c <= move and c < best.get((nu, nv), 99):
                best[(nu, nv)] = c
                q.append((nu, nv))
    return {p for p in best if p != start and p not in allies}


def diamond_mask(color, inset=0):
    im = Image.new("RGBA", (TW, TH), (0, 0, 0, 0))
    for y, (x0, x1) in enumerate(DROWS):
        for x in range(x0, x1):
            im.putpixel((x, y), color)
    return im


def diamond_outline(color):
    im = Image.new("RGBA", (TW, TH), (0, 0, 0, 0))
    for y, (x0, x1) in enumerate(DROWS):
        im.putpixel((x0, y), color)
        im.putpixel((x1 - 1, y), color)
        if y in (0, TH - 1):
            for x in range(x0, x1):
                im.putpixel((x, y), color)
    return im


def gray(im):
    out = im.copy()
    px = out.load()
    for y in range(out.height):
        for x in range(out.width):
            r, g, b, a = px[x, y]
            if a:
                v = int(0.3 * r + 0.55 * g + 0.15 * b)
                px[x, y] = (v * 3 // 4 + 34, v * 3 // 4 + 34, v * 3 // 4 + 40, 255)
    return out


def pennant(team):
    col = (60, 110, 220, 255) if team == "ally" else (210, 52, 48, 255)
    dark = (30, 54, 128, 255) if team == "ally" else (120, 28, 30, 255)
    im = Image.new("RGBA", (8, 14), (0, 0, 0, 0))
    k = S.PAL["k"] + (255,)
    for y in range(1, 14):
        im.putpixel((1, y), (90, 60, 40, 255))
        im.putpixel((0, y), k)
    for y in range(1, 6):
        for x in range(2, 7 - (y == 5) - (y == 1)):
            im.putpixel((x, y), col if y < 4 else dark)
    for x in range(2, 7):
        im.putpixel((x, 0), k)
    return im


HOUSES = {(1, 5), (1, 6), (1, 7), (1, 8), (3, 5), (2, 8), (3, 8)}


def render_screen(frames, portrait_img):
    trees, tufts, rocks = load_objects()
    house_img, house_pivot = house()

    # native map canvas
    CW, CH = 528, 312
    OX, OY = 248, 44  # screen position of tile (0,0) top-left of diamond bbox
    ox, oy = OX - TW // 2, OY
    canvas = Image.new("RGBA", (CW, CH), (0, 0, 0, 0))

    # slab edges (south-west and south-east faces)
    sw = Image.new("RGBA", (CW, CH), (0, 0, 0, 0))
    d = ImageDraw.Draw(sw)
    L = iso(0, MH, ox + TW // 2, oy)        # left-bottom vertex
    Bt = iso(MW, MH, ox + TW // 2, oy)      # bottom vertex
    R = iso(MW, 0, ox + TW // 2, oy)
    d.polygon([L, Bt, (Bt[0], Bt[1] + EDGE), (L[0], L[1] + EDGE)], fill=T["edge"][0])
    d.polygon([Bt, R, (R[0], R[1] + EDGE), (Bt[0], Bt[1] + EDGE)], fill=T["edge"][1])
    for i in range(0, Bt[0] - L[0], 5):
        x = L[0] + i
        y = L[1] + i // 2 + 2 + (h32(i) % 3)
        d.point((x, y), fill=T["edge"][2])
    for i in range(0, R[0] - Bt[0], 6):
        x = Bt[0] + i
        y = Bt[1] - i // 2 + 3 + (h32(i, 7) % 3)
        d.point((x, y), fill=T["edge"][2])
    d.line([(L[0], L[1] + EDGE), (Bt[0], Bt[1] + EDGE), (R[0], R[1] + EDGE)], fill=S.PAL["k"] + (255,))
    canvas.alpha_composite(sw)

    for v in range(MH):
        for u in range(MW):
            x, y = iso(u, v, ox, oy)
            canvas.alpha_composite(tile_image(B01[v][u], u, v), (x, y))
    grid_line = diamond_outline(T["line"])
    for v in range(MH):
        for u in range(MW):
            x, y = iso(u, v, ox, oy)
            canvas.alpha_composite(grid_line, (x, y))

    # units
    allies = {(2, 5): ("유비", "infantry-ally", False), (2, 6): ("관우", "cavalry-ally", False),
              (7, 6): ("장비", "cavalry-ally", True)}
    enemies = {(8, 5): ("황건적", "bandit-enemy"), (8, 6): ("황건적", "bandit-enemy"),
               (10, 10): ("황건적", "bandit-enemy"), (13, 3): ("황건적", "bandit-enemy"),
               (12, 8): ("등무", "officer-enemy"), (14, 6): ("정원지", "cavalry-enemy")}
    occupied_enemy = set(enemies)
    mr = move_range((2, 6), 5, occupied_enemy, set(allies))
    blue = diamond_mask((70, 130, 240, 92))
    blue_edge = diamond_outline((150, 200, 255, 170))
    for (u, v) in mr:
        x, y = iso(u, v, ox, oy)
        canvas.alpha_composite(blue, (x, y))
        canvas.alpha_composite(blue_edge, (x, y))
    # attackable from range
    attack = set()
    for (u, v) in mr | {(2, 6)}:
        for du, dv in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            if (u + du, v + dv) in occupied_enemy:
                attack.add((u + du, v + dv))
    red = diamond_mask((230, 60, 50, 110))
    red_edge = diamond_outline((255, 150, 120, 220))
    for (u, v) in attack:
        x, y = iso(u, v, ox, oy)
        canvas.alpha_composite(red, (x, y))
        canvas.alpha_composite(red_edge, (x, y))
    cursor = diamond_outline((255, 224, 96, 255))
    sel = iso(2, 6, ox, oy)
    canvas.alpha_composite(cursor, sel)
    canvas.alpha_composite(diamond_outline((255, 120, 90, 255)), iso(8, 5, ox, oy))

    # depth-sorted objects
    drawables = []
    for v in range(MH):
        for u in range(MW):
            cx, cy = iso(u, v, ox + TW // 2, oy + TH // 2)
            t = B01[v][u]
            if t == "f":
                tr = trees[h32(u, v, 3) % 3]
                drawables.append((u + v, 0, tr, (cx - 20, cy + 2 - 44)))
            elif t == "v" and (u, v) in HOUSES:
                drawables.append((u + v, 0, house_img, (cx - house_pivot[0], cy + 3 - house_pivot[1])))
            elif t == "." and h32(u, v, 9) % 100 < 22:
                tf = tufts[h32(u, v, 11) % len(tufts)]
                jx = h32(u, v, 12) % 9 - 4
                jy = h32(u, v, 13) % 5 - 2
                drawables.append((u + v - 0.5, 0, tf, (cx - 20 + jx, cy - 44 + jy)))
            elif t == "." and h32(u, v, 19) % 100 < 4:
                drawables.append((u + v - 0.5, 0, rocks[h32(u, v, 2) % 4], (cx - 20, cy - 44 + 1)))
    unit_tags = []
    for pos, (name, sid, done) in allies.items():
        im = frames[sid][0 if "cavalry" not in sid else 2]
        if done:
            im = gray(im)
        cx, cy = iso(*pos, ox + TW // 2, oy + TH // 2)
        drawables.append((sum(pos) + 0.1, 1, im, (cx - S.ANCHOR[0], cy + 2 - S.ANCHOR[1])))
        drawables.append((sum(pos) + 0.05, 1, pennant("ally") if not done else gray(pennant("ally")),
                          (cx - 9, cy - 26 if "cavalry" not in sid else cy - 30)))
        unit_tags.append((name, cx, cy, "ally", done))
    for pos, (name, sid) in enemies.items():
        im = frames[sid][3]
        im = im.transpose(Image.FLIP_LEFT_RIGHT)  # enemies face SW toward allies
        cx, cy = iso(*pos, ox + TW // 2, oy + TH // 2)
        drawables.append((sum(pos) + 0.1, 1, im, (cx - (S.CELL_W - S.ANCHOR[0]), cy + 2 - S.ANCHOR[1])))
        drawables.append((sum(pos) + 0.05, 1, pennant("enemy"),
                          (cx + 3, cy - 26 if "cavalry" not in sid else cy - 30)))
        unit_tags.append((name, cx, cy, "enemy", False))
    # team base shadows (drawn before sprites, after terrain)
    for name, cx, cy, team, done in unit_tags:
        sh = Image.new("RGBA", (20, 8), (0, 0, 0, 0))
        ImageDraw.Draw(sh).ellipse((0, 0, 19, 7), fill=(10, 14, 10, 90))
        canvas.alpha_composite(sh, (cx - 10, cy - 2))
    drawables.sort(key=lambda d_: (d_[0], d_[1]))
    for _, _, im, pos in drawables:
        canvas.alpha_composite(im, pos)

    map_big = canvas.resize((CW * SCALE, CH * SCALE), Image.NEAREST)

    # ---- full-resolution screen ----
    W, H = 1280, 850
    screen = Image.new("RGBA", (W, H), (0, 0, 0, 255))
    bg = ImageDraw.Draw(screen)
    for y in range(H):
        t = y / H
        bg.line([(0, y), (W, y)], fill=(int(16 + 10 * t), int(24 + 12 * t), int(34 + 8 * t), 255))
    # faint lattice pattern in background
    for y in range(0, H, 24):
        for x in range((y // 24) % 2 * 24, W, 48):
            bg.point((x, y), fill=(44, 58, 70, 255))
    MX, MY = 124, 64
    screen.alpha_composite(map_big, (MX, MY))

    def S2(x, y):
        return MX + x * SCALE, MY + y * SCALE

    f = lambda sz: ImageFont.truetype(FONT, sz)
    dr = ImageDraw.Draw(screen)
    GOLD = (226, 188, 98, 255)
    IVORY = (238, 230, 210, 255)

    def panel(box, alpha=200):
        lay = Image.new("RGBA", (W, H), (0, 0, 0, 0))
        ld = ImageDraw.Draw(lay)
        x0, y0, x1, y1 = box
        ld.rounded_rectangle(box, radius=6, fill=(14, 20, 30, alpha), outline=(176, 138, 64, 255), width=2)
        ld.line([(x0 + 6, y0 + 4), (x1 - 6, y0 + 4)], fill=(240, 210, 130, 90))
        screen.alpha_composite(lay)

    def txt(xy, s, size, fill=IVORY, anchor="la", stroke=0):
        dr.text(xy, s, font=f(size), fill=fill, anchor=anchor,
                stroke_width=stroke, stroke_fill=(10, 10, 14, 255))

    # name tags + hp bars over units
    hp = {"유비": 1.0, "관우": 1.0, "장비": 0.86, "등무": 1.0, "정원지": 1.0}
    hp_enemy = {(8, 6): 0.38}
    for name, cx, cy, team, done in unit_tags:
        sx, sy = S2(cx, cy)
        top = sy - (78 if team == "ally" and name in ("관우", "장비") else 62)
        if name == "정원지":
            top = sy - 78
        if name == "유비":
            top -= 16
        w = 34
        frac = hp.get(name, 0.8)
        for pos, fr in hp_enemy.items():
            if iso(*pos, ox + TW // 2, oy + TH // 2) == (cx, cy):
                frac = fr
        dr.rectangle((sx - w // 2 - 1, top - 1, sx + w // 2 + 1, top + 4), fill=(10, 10, 14, 230))
        col = (96, 200, 110, 255) if frac > 0.5 else (230, 180, 60, 255)
        if done:
            col = (130, 136, 140, 255)
        dr.rectangle((sx - w // 2, top, sx - w // 2 + int(w * frac), top + 3), fill=col)
        if name != "황건적":
            tc = (150, 196, 255, 255) if team == "ally" else (255, 150, 140, 255)
            if done:
                tc = (160, 160, 166, 255)
            txt((sx, top - 4), name, 13, tc, "md", stroke=2)

    # floating damage number on hit bandit (8,6)
    hx_, hy_ = S2(*iso(8, 6, ox + TW // 2, oy + TH // 2))
    txt((hx_ + 34, hy_ - 66), "-142", 26, (255, 236, 120, 255), "md", stroke=3)
    # target tooltip for (8,5)
    tx_, ty_ = S2(*iso(8, 5, ox + TW // 2, oy + TH // 2))
    panel((tx_ + 70, ty_ - 128, tx_ + 230, ty_ - 66), 215)
    txt((tx_ + 80, ty_ - 122), "공격 예상 · 황건적", 13, GOLD)
    txt((tx_ + 80, ty_ - 102), "피해 118 · 명중 92%", 14)
    txt((tx_ + 80, ty_ - 84), "반격 없음", 12, (180, 186, 196, 255))

    # top-left: stage title
    txt((28, 22), "황건적의 난", 30, GOLD, stroke=0)
    txt((30, 62), "탁현 들판 · B01", 14, (170, 180, 190, 255))

    # top-right: round + objective
    panel((928, 20, 1258, 132))
    txt((946, 30), "1턴 · 아군 진영", 22, (150, 196, 255, 255))
    dr.line([(946, 66), (1240, 66)], fill=(176, 138, 64, 120))
    txt((946, 76), "승리  장각 격퇴", 15, IVORY)
    txt((946, 100), "패배  유비 전투불능", 15, (200, 170, 160, 255))

    # action menu near 관우
    gx, gy = S2(*iso(2, 6, ox + TW // 2, oy + TH // 2))
    mx0, my0 = 206, 96
    panel((mx0, my0, mx0 + 128, my0 + 200), 225)
    items = ["이동", "공격", "병법", "아이템", "대기", "일기토"]
    for i, it in enumerate(items):
        y = my0 + 10 + i * 31
        if i == 0:
            lay = Image.new("RGBA", (W, H), (0, 0, 0, 0))
            ImageDraw.Draw(lay).rounded_rectangle((mx0 + 8, y - 2, mx0 + 120, y + 28), 4,
                                                   fill=(176, 138, 64, 110), outline=GOLD)
            screen.alpha_composite(lay)
        col = IVORY if it != "일기토" else (120, 126, 134, 255)
        txt((mx0 + 22, y + 2), it, 18, col)
        if i == 0:
            txt((mx0 + 112, y + 2), "▶", 14, GOLD, "ra")
    # connector line from menu to unit
    dr.line([(mx0 + 128, my0 + 24), (gx - 14, gy - 84)], fill=(226, 188, 98, 160), width=2)

    # bottom-left: unit card
    cx0, cy0 = 20, 636
    panel((cx0, cy0, cx0 + 430, cy0 + 196), 225)
    pbox = (cx0 + 14, cy0 + 22, cx0 + 14 + 152, cy0 + 22 + 152)
    dr.rectangle(pbox, fill=(30, 52, 60, 255), outline=(176, 138, 64, 255), width=2)
    # portrait background stripes (UI, not baked)
    for i in range(0, 148, 8):
        dr.line([(pbox[0] + 2, pbox[1] + 2 + i), (pbox[2] - 2, pbox[1] + 2 + i)], fill=(36, 60, 68, 255))
    pimg = portrait_img.resize((48 * 3, 48 * 3), Image.NEAREST)
    screen.alpha_composite(pimg, (pbox[0] + 4, pbox[1] + 4))
    x = cx0 + 184
    txt((x, cy0 + 16), "관우", 26, IVORY)
    txt((x + 66, cy0 + 26), "운장", 13, (170, 180, 190, 255))
    txt((x, cy0 + 54), "경기병 · Lv 1", 15, GOLD)

    def bar(y, label, cur, mx, col):
        txt((x, y), label, 13, (180, 190, 200, 255))
        bx0, bx1 = x + 34, x + 230
        dr.rectangle((bx0, y + 4, bx1, y + 14), fill=(8, 10, 14, 255), outline=(70, 80, 90, 255))
        dr.rectangle((bx0 + 1, y + 5, bx0 + 1 + int((bx1 - bx0 - 2) * cur / mx), y + 13), fill=col)
        txt((bx1, y - 16), f"{cur} / {mx}" if label != "XP" else f"{cur}", 12, IVORY, "ra")
    bar(cy0 + 92, "병력", 1140, 1140, (96, 200, 110, 255))
    bar(cy0 + 120, "병법", 34, 69, (90, 150, 240, 255))
    bar(cy0 + 148, "XP", 53, 100, (226, 188, 98, 255))
    txt((x, cy0 + 170), "특성  호걸 · 반격술", 12, (190, 196, 206, 255))

    # bottom-right: log + minimap hint
    panel((928, 690, 1258, 832))
    txt((946, 700), "전투 기록", 15, GOLD)
    log = ["장비 → 황건적  -142", "황건적 반격 없음", "관우 선택 · 이동 5"]
    for i, l in enumerate(log):
        txt((946, 728 + i * 24), l, 14, IVORY if i else (255, 236, 160, 255))
    # key hints
    txt((640, 836), "클릭: 선택 · 우클릭: 취소 · Tab: 부대 전환 · Esc: 메뉴", 12, (130, 140, 150, 255), "md")
    return screen


def main():
    os.makedirs(OUT, exist_ok=True)
    sheet, meta, frames = build_sheet()
    sheet.save(os.path.join(OUT, "sheet.png"))
    with open(os.path.join(OUT, "sheet.json"), "w") as fp:
        json.dump(meta, fp, ensure_ascii=False, indent=1)
    probs = check_sheet(sheet, meta)
    bgc = Image.new("RGBA", sheet.size, (92, 138, 80, 255))
    bgc.alpha_composite(sheet)
    bgc.resize((sheet.width * 4, sheet.height * 4), Image.NEAREST).save(os.path.join(OUT, "sheet-x4.png"))

    por = portrait()
    por.save(os.path.join(OUT, "portrait.png"))
    with open(os.path.join(OUT, "portrait.json"), "w") as fp:
        json.dump({"size": list(por.size), "displayScale": 3}, fp)
    screen = render_screen(frames, por)
    screen.convert("RGB").save(os.path.join(OUT, "screen.png"))
    assert screen.size == (1280, 850)
    print("sheet checks:", "OK" if not probs else probs)


if __name__ == "__main__":
    main()
