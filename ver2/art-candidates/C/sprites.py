"""Candidate C sprite parts: hand-authored pixel grids composed with integer offsets."""
from PIL import Image

CELL_W, CELL_H = 40, 40
ANCHOR = (20, 36)

PAL = {
    "k": (22, 18, 26),      # outline
    "s": (236, 186, 140),   # skin
    "S": (196, 134, 98),    # skin shade
    "e": (40, 28, 30),      # eye
    "h": (40, 32, 34),      # hair
    "b": (84, 128, 204),    # blue light
    "B": (50, 82, 154),     # blue mid
    "n": (30, 42, 92),      # navy
    "g": (226, 184, 82),    # gold
    "G": (160, 112, 44),    # gold shade
    "m": (214, 222, 230),   # steel light
    "M": (122, 132, 152),   # steel dark
    "r": (200, 54, 52),     # red
    "R": (124, 32, 40),     # dark red
    "y": (240, 206, 72),    # yellow
    "Y": (178, 136, 40),    # ochre
    "l": (146, 96, 56),     # leather
    "L": (86, 54, 36),      # dark leather
    "w": (232, 222, 196),   # ivory cloth
    "W": (176, 164, 140),   # ivory shade
    "c": (178, 116, 70),    # horse light
    "C": (128, 78, 48),     # horse mid
    "D": (78, 46, 32),      # horse dark / mane
    "t": (98, 70, 46),      # bandit cloth brown
    "T": (64, 46, 34),      # bandit cloth dark
}


def grid(rows):
    return [r for r in rows]


# ---- infantry (facing SE) -------------------------------------------------
HELMET = grid([
    "....rr...",
    "...rrR...",
    "...MmmM..",
    "..MmmmmM.",
    "..BbgbbB.",
    "..hssssS.",
    "..hsesse.",
    "..hssssS.",
    "...sSSS..",
])

BANDIT_HEAD = grid([
    ".........",
    "...hhh...",
    "..hhhhhh.",
    "..yyYyyy.",
    "..yhsssy.",
    "..hssssS.",
    "..hsesse.",
    "..hhsssS.",
    "...hSSh..",
])

TORSO = grid([
    "..nBbbBn..",
    ".nBbbbbbB.",
    ".BbBbBbbB.",
    ".BbbBbBbB.",
    ".nBbbbbBn.",
    ".ggGgygGg.",
    ".nBBnBBn..",
    ".nBn.nBn..",
])

BANDIT_TORSO = grid([
    "..TttwT...",
    ".TtttwtT..",
    ".ttttwttT.",
    ".tttwtttT.",
    ".Tttwttt..",
    ".rrRrrRr..",
    ".TttTttT..",
    ".Tt..Tt...",
])

SHIELD = grid([
    "..ll..",
    ".lYyl.",
    "lYgyYl",
    "lygGyl",
    "lYyyYl",
    ".lYyl.",
    "..ll..",
])

SWORD_ARM = grid([
    "Bb.",
    "Bb.",
    "BB.",
    "sS.",
    "Gg.",
    "m..",
    "mM.",
    "mM.",
    ".M.",
])

CLUB_ARM = grid([
    "tT.",
    "tT.",
    "tt.",
    "sS.",
    "lL.",
    "lL.",
    "lLL",
    "LLL",
    ".L.",
])

BACK_HAND = grid([
    "Bn",
    "sS",
])

BANDIT_BACK_HAND = grid([
    "tT",
    "sS",
])

LEG = grid([
    "nn",
    "nn",
    "nn",
    "nw",
    "LL",
    "LLL",
])
BANDIT_LEG = grid([
    "TT",
    "TT",
    "Tw",
    "ww",
    "LL",
    "LLL",
])


def blit(px, g, ox, oy, recolor=None):
    for y, row in enumerate(g):
        for x, ch in enumerate(row):
            if ch == ".":
                continue
            if recolor and ch in recolor:
                ch = recolor[ch]
            px[(ox + x, oy + y)] = ch


def outline(px):
    out = dict(px)
    for (x, y) in px:
        for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            q = (x + dx, y + dy)
            if q not in px:
                out[q] = "k"
    return out


def to_image(px, pal=PAL):
    im = Image.new("RGBA", (CELL_W, CELL_H), (0, 0, 0, 0))
    for (x, y), ch in px.items():
        if 0 <= x < CELL_W and 0 <= y < CELL_H:
            r, g, b = pal[ch]
            im.putpixel((x, y), (r, g, b, 255))
    return im


# Feet bottom row (before outline) sits on ANCHOR[1]-1 so the outline lands on ANCHOR[1].
FOOT_Y = ANCHOR[1] - 1


def footman(frame, kind="ally"):
    """frame: ('idle', i) or ('walk', i)."""
    anim, i = frame
    head = HELMET if kind == "ally" else BANDIT_HEAD
    torso = TORSO if kind == "ally" else BANDIT_TORSO
    arm = SWORD_ARM if kind == "ally" else CLUB_ARM
    hand = BACK_HAND if kind == "ally" else BANDIT_BACK_HAND
    leg = LEG if kind == "ally" else BANDIT_LEG

    # body origin: torso top-left. Legs are 5 rows; torso 8 rows overlaps 2 rows of legs.
    lx_a, lx_b, lift_a, lift_b, bob, arm_dx, breath = 0, 0, 0, 0, 0, 0, 0
    if anim == "idle":
        breath = [0, 0, 0, 1, 1, 1, 0, 0][i]
    else:
        # A = screen-left leg (near), B = screen-right leg (far)
        lx_a, lx_b, lift_a, lift_b, bob = [
            (-2, 2, 0, 0, 0),   # contact A back, B forward
            (-1, 1, 0, 0, 1),   # down
            (0, 0, 0, 2, 0),    # passing, B lifted
            (1, -1, 0, 1, -1),  # up
            (2, -2, 0, 0, 0),   # contact A forward
            (1, -1, 0, 0, 1),   # down
            (0, 0, 2, 0, 0),    # passing, A lifted
            (-1, 1, 1, 0, -1),  # up
        ][i]
        arm_dx = -(lx_a > 0) + (lx_a < 0)  # opposite arm swing, 1px

    px = {}
    base_x = 15
    leg_top = FOOT_Y - 5
    # far leg first
    blit(px, leg, base_x + 5 + lx_b, leg_top - lift_b + max(bob, 0) * 0)
    blit(px, leg, base_x + 1 + lx_a, leg_top - lift_a)
    torso_y = leg_top - 5 + bob
    # lower torso (belt + skirt) fixed relative to bob, upper torso breathes
    blit(px, torso[5:], base_x, torso_y + 5)
    blit(px, torso[:5], base_x, torso_y + breath)
    blit(px, hand, base_x + 9 - arm_dx, torso_y + 4 + breath)
    if kind == "ally":
        blit(px, SHIELD, base_x + 8 - arm_dx, torso_y + 1 + breath)
    blit(px, head, base_x, torso_y - 9 + breath)
    blit(px, arm, base_x - 2 + arm_dx, torso_y + 1 + breath)
    return outline(px)


# ---- cavalry (facing SE / right) -----------------------------------------
HORSE_BODY = grid([
    "...................D......",
    "..................DcD.....",
    ".................DDcccC...",
    ".................DcceccCC.",
    "................DDccccccCC",
    "...............DDccC.CcCDC",
    "..............DDccC...CCC.",
    "..............DcccC.......",
    ".DD...ccccccccccccC.......",
    "DDD.cccccccccccccC........",
    "DD.ccccccccccccccC........",
    "D..ccccccccccccccC........",
    "D..CcccccccccccCC.........",
    "...CCCCCCCCCCCCC..........",
])
SADDLE = grid([
    ".rrrrr.",
    "rrRRRrr",
    "rRggGRr",
    "RRRRRRR",
])
# one horse leg: 6 rows
HLEG = grid([
    "Cc",
    "Cc",
    "Cc",
    "CC",
    "C.",
    "C.",
    "DD",
])
HLEG_BENT = grid([
    "Cc",
    "Cc",
    "CCc",
    ".CC",
    "..C",
    "..D",
])

RIDER_TORSO = grid([
    "..nBbbBn..",
    ".nBbbbbbB.",
    ".BbBbBbbB.",
    ".BbbBbBbB.",
    ".ggGgygGg.",
    ".nBBbBB...",
    "..nBBBn...",
])


def cavalry(frame, kind="ally"):
    anim, i = frame
    px = {}
    hx, hy = 9, FOOT_Y - 6 - 13  # horse body origin; legs 7 rows below body row 13
    bob, breath, head_nod = 0, 0, 0
    if anim == "idle":
        breath = [0, 0, 0, 1, 1, 1, 0, 0][i]
        head_nod = [0, 0, 0, 0, 1, 1, 1, 0][i]
        legs = [(0, 0, False)] * 4
    else:
        bob = [0, 1, 0, 0, 0, 1, 0, 0][i]
        head_nod = [0, 1, 1, 0, 0, 1, 1, 0][i]
        # four-beat walk: (dx, lift, bent)
        cyc = [(-1, 0, False), (-1, 0, False), (0, 0, False), (1, 0, False),
               (1, 0, False), (1, 1, True), (0, 2, True), (-1, 1, True)]
        # far-hind, far-front, near-hind, near-front
        legs = [cyc[(i + 2) % 8], cyc[(i + 6) % 8], cyc[(i + 4) % 8], cyc[i % 8]]
    leg_top = FOOT_Y - 6
    leg_x = [hx + 5, hx + 14, hx + 3, hx + 12]
    far_recolor = {"c": "C", "C": "D"}
    for n, (dx, lift, bent) in enumerate(legs):
        g = HLEG_BENT if bent else HLEG
        top = leg_top - lift + (1 if bent else 0)
        blit(px, g, leg_x[n] + dx, top, far_recolor if n < 2 else None)
    body_y = hy + bob
    blit(px, HORSE_BODY[8:], hx, body_y + 8)
    blit(px, HORSE_BODY[:8], hx, body_y + head_nod)
    blit(px, SADDLE, hx + 6, body_y + 8)
    rx, ry = hx + 5, body_y + 2 + breath
    torso = RIDER_TORSO
    head = HELMET if kind == "ally" else BANDIT_HEAD
    # spear behind rider's near arm: shaft from lower-back to upper-front
    sx, sy = rx + 2, ry + 9
    shaft = [(sx + t, sy - (t * 2) // 3) for t in range(0, 17)]
    for (x, y) in shaft:
        px[(x, y)] = "l" if (x + y) % 3 else "L"
    tx, ty = shaft[-1]
    for (x, y, c) in [(tx + 1, ty, "m"), (tx + 1, ty - 1, "m"), (tx + 2, ty - 1, "m"),
                      (tx + 2, ty - 2, "m"), (tx + 3, ty - 2, "M"), (tx, ty, "M"),
                      (tx - 1, ty + 1, "r"), (tx - 1, ty + 2, "r"), (tx - 2, ty + 1, "R")]:
        px[(x, y)] = c
    blit(px, torso, rx, ry)
    blit(px, head, rx, ry - 9)
    blit(px, grid(["nB", "nB", "nb", "LL", "LLL"]), rx + 5, body_y + 9)
    # hands on shaft
    for (x, y) in shaft:
        if x == rx + 7 or x == rx + 10:
            px[(x, y)] = "s"
    return outline(px)


GRAY = {}
for k_, (r_, g_, b_) in PAL.items():
    v = int(0.3 * r_ + 0.55 * g_ + 0.15 * b_)
    GRAY[k_] = (v * 3 // 4 + 30, v * 3 // 4 + 30, v * 3 // 4 + 36) if k_ != "k" else PAL["k"]
