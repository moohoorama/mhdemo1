"""Props for the scenario scenes (real scale, not the deformed battle map): folding screen, bed, lantern, curtains, stairs,
garden rock, plum tree, pavilion, milestone, incense burner. Drawn procedurally in the game's pixel style (dark outline, three tones).
Pivots: footprint centre on the ground (wall-like pieces: the top vertex of their edge, like wall_canvas_*)."""
import random
from common import *
from PIL import Image, ImageDraw

OL = (26, 18, 22, 255)
WOOD = [(150, 98, 56, 255), (120, 76, 44, 255), (88, 54, 32, 255)]
STONE = [(190, 196, 200, 255), (150, 158, 166, 255), (108, 116, 128, 255), (72, 80, 92, 255)]
RED = [(190, 40, 48, 255), (150, 28, 40, 255), (104, 22, 34, 255)]
GOLD = [(244, 200, 80, 255), (200, 150, 40, 255)]
SILK = [(236, 226, 196, 255), (214, 200, 164, 255), (186, 170, 132, 255)]
SLATE = [(99, 114, 128, 255), (63, 74, 86, 255)]
EAVE = (102, 55, 32, 255)


def box(d, x, y, w, depth, h, top, left, right, axis="u"):
    """an iso box standing with its bottom-front vertex at (x, y): w along the x-axis slope, depth along the other"""
    # left-bottom (x-w.., ) simplified: the box footprint is a parallelogram with vertices
    # F=(x,y) front, R=(x+w,y-w/2) right, B=(x+w-depth, y-w/2-depth/2) back, L=(x-depth, y-depth/2) left
    F = (x, y)
    R = (x + w, y - w // 2)
    Bk = (x + w - depth, y - w // 2 - depth // 2)
    L = (x - depth, y - depth // 2)
    up = lambda p: (p[0], p[1] - h)
    d.polygon([L, F, up(F), up(L)], fill=left, outline=OL)
    d.polygon([F, R, up(R), up(F)], fill=right, outline=OL)
    d.polygon([up(L), up(F), up(R), up(Bk)], fill=top, outline=OL)
    return F, R, Bk, L, up


def screen(orient):
    im = Image.new("RGBA", (50, 56))
    d = ImageDraw.Draw(im)
    n, pw, ph = 3, 11, 28
    sl = 0.5 if orient == "u" else -0.5
    x0, y0 = (6, 36) if orient == "u" else (6 + n * pw, 36 + n * pw // 2 - 0)
    for k in range(n):
        xa = (6 + k * pw) if orient == "u" else (6 + k * pw)
        ya = (30 + (k * pw) * 0.5) if orient == "u" else (30 + (n - k) * pw * 0.5)
        xb = xa + pw
        yb = ya + (pw * 0.5 if orient == "u" else -pw * 0.5)
        panel = [(xa, ya), (xb, yb), (xb, yb - ph), (xa, ya - ph)]
        d.polygon([(x, y) for x, y in panel], fill=SILK[k % 2], outline=OL)
        # a painted ridge of hills and a pale sun
        for i in range(pw - 2):
            x = xa + 1 + i
            yy = ya + (x - xa) * sl
            top = ph - 7 - int(5 * abs(((i + k * 5) % 10) - 5) / 5)
            d.line([(x, yy - 2), (x, yy - top)], fill=(120, 150, 130, 255) if (i + k) % 2 else (96, 128, 112, 255))
        d.ellipse([xa + 3, ya - ph + 3 + (pw // 2) * (sl > 0), xa + 6, ya - ph + 6 + (pw // 2) * (sl > 0)], fill=(230, 120, 90, 255))
        d.line([(xa, ya - ph), (xb, yb - ph)], fill=WOOD[1])
        d.line([(xa, ya - 1), (xb, yb - 1)], fill=WOOD[1])
    return im, (25, 44)


def bed():
    im = Image.new("RGBA", (52, 40))
    d = ImageDraw.Draw(im)
    box(d, 26, 34, 24, 14, 6, WOOD[0], WOOD[1], WOOD[2])                    # the frame
    # the quilt lies on the frame, the pillow at the far end
    d.polygon([(14, 24), (26, 28), (40, 21), (28, 17)], fill=RED[0], outline=OL)
    d.polygon([(26, 28), (40, 21), (40, 23), (26, 30)], fill=RED[1], outline=OL)
    d.polygon([(14, 24), (22, 20), (28, 23), (20, 27)], fill=(236, 232, 224, 255), outline=OL)
    d.line([(26, 27), (33, 23)], fill=GOLD[0])
    return im, (26, 34)


def lantern():
    im = Image.new("RGBA", (16, 36))
    d = ImageDraw.Draw(im)
    d.rectangle([7, 14, 8, 33], fill=WOOD[1], outline=OL)
    d.rectangle([4, 33, 11, 35], fill=STONE[2], outline=OL)
    d.rectangle([3, 3, 12, 14], fill=(250, 214, 110, 255), outline=OL)
    d.rectangle([5, 5, 10, 12], fill=(255, 236, 160, 255))
    d.line([(3, 8), (12, 8)], fill=(200, 150, 60, 255))
    d.polygon([(1, 3), (14, 3), (11, 0), (4, 0)], fill=RED[1], outline=OL)
    d.rectangle([3, 14, 12, 15], fill=RED[1], outline=OL)
    return im, (8, 35)


def curtain(side):
    """a hung drape on a wall edge, same footprint as wall_lacquer_* / wall_canvas_* (nw: the bottom rises to the right, ne: it falls)"""
    im = Image.new("RGBA", (37, 50))
    d = ImageDraw.Draw(im)
    nw = side == "nw"
    xs = range(2, 35) if nw else range(3, 36)
    H = 32
    for x in xs:
        yb = 49 - (x - 2) * 0.5 if nw else 33 + (x - 3) * 0.45
        yb = int(round(yb))
        ytop = yb - H
        col = RED[0] if (x // 3) % 2 == 0 else RED[1]
        if x % 6 == 0:
            col = RED[2]
        d.line([(x, ytop), (x, yb)], fill=col)
        d.point((x, ytop), fill=GOLD[0])
        d.point((x, ytop + 1), fill=GOLD[1])
        if x % 2 == 0:
            d.point((x, yb + 1), fill=GOLD[1])
    px = im.load()
    for x in (xs[0], xs[-1]):
        ys = [y for y in range(im.height) if px[x, y][3]]
        for y in range(min(ys), max(ys) + 1):
            px[x, y] = OL
    for x in xs:   # a dark line under the rod and along the bottom so the drape reads against the wall
        ys = [y for y in range(im.height) if px[x, y][3]]
        if ys:
            top = min(ys)
            px[x, max(0, top - 1)] = OL
    return im, ((34, 48) if nw else (4, 48))


def stairs(orient):
    im = Image.new("RGBA", (48, 36))
    d = ImageDraw.Draw(im)
    for k in range(3):
        w = 28 - k * 4
        x = 24 + (0 if orient == "u" else 0) - 0
        y = 32 - k * 5
        top = STONE[0] if k == 2 else STONE[1]
        if orient == "u":
            box(d, 14 + k * 2, y, w, 12 - k * 2, 5, top, STONE[2], STONE[3])
        else:
            box(d, 12 + k * 3, y - k * 0, w, 12 - k * 2, 5, top, STONE[3], STONE[2])
    return im, (24, 32)


def garden_rock():
    im = Image.new("RGBA", (28, 44))
    d = ImageDraw.Draw(im)
    pts = [(4, 42), (2, 30), (6, 20), (4, 10), (10, 3), (15, 8), (20, 2), (24, 12), (22, 22), (26, 32), (22, 42), (14, 44)]
    d.polygon(pts, fill=STONE[1], outline=OL)
    d.polygon([(10, 3), (15, 8), (14, 22), (8, 30), (6, 20), (4, 10)], fill=STONE[0], outline=None)
    d.polygon([(20, 2), (24, 12), (22, 22), (26, 32), (22, 42), (16, 40), (18, 24)], fill=STONE[2], outline=None)
    d.polygon(pts, outline=OL)
    for cx, cy, r in ((10, 17, 2), (17, 28, 2), (13, 36, 1), (16, 12, 1)):
        d.ellipse([cx - r, cy - r, cx + r, cy + r], fill=OL)
    return im, (14, 42)


def plum_tree():
    pm, pv = prop("peach_tree")
    im = pm.copy()
    px = im.load()
    rnd = random.Random(4)
    for y in range(im.height):
        for x in range(im.width):
            r, g, b, a = px[x, y]
            if a and r > 170 and b > 110 and g < 200:  # the pink crown -> pale plum blossom with red buds
                k = (r + g + b) / 3
                c = (236, 226, 218) if k > 200 else (214, 196, 196)
                if rnd.random() < .12:
                    c = (196, 56, 74)
                px[x, y] = c + (a,)
    return im, pv


def pavilion():
    """a four-pillared garden pavilion on a 2x2-cell wood floor; the roof is drawn in the same sprite"""
    im = Image.new("RGBA", (92, 90))
    d = ImageDraw.Draw(im)
    cx, cy = 46, 70
    d.polygon([(cx - 32, cy), (cx, cy - 16), (cx + 32, cy), (cx, cy + 16)], fill=WOOD[1], outline=OL)
    for i in range(-3, 4):
        d.line([(cx + i * 8 - 12, cy - 6 + abs(i)), (cx + i * 8 + 12, cy + 6 - abs(i))], fill=WOOD[0])
    d.polygon([(cx - 32, cy), (cx, cy + 16), (cx, cy + 20), (cx - 32, cy + 4)], fill=WOOD[2], outline=OL)
    d.polygon([(cx + 32, cy), (cx, cy + 16), (cx, cy + 20), (cx + 32, cy + 4)], fill=(70, 44, 26, 255), outline=OL)
    ph = 34
    for (px_, py_) in ((cx - 28, cy), (cx, cy - 14), (cx + 28, cy), (cx, cy + 14)):
        if py_ < cy:   # far pillars first
            d.rectangle([px_ - 1, py_ - ph, px_ + 1, py_], fill=RED[0], outline=OL)
    rc = cy - ph
    d.polygon([(cx - 38, rc), (cx, rc + 19), (cx + 38, rc), (cx + 38, rc - 3), (cx, rc + 16), (cx - 38, rc - 3)], fill=EAVE, outline=OL)
    peak = (cx, rc - 20)
    d.polygon([(cx - 38, rc - 3), (cx, rc + 16), peak], fill=SLATE[0], outline=OL)
    d.polygon([(cx, rc + 16), (cx + 38, rc - 3), peak], fill=SLATE[1], outline=OL)
    d.line([(cx, rc + 16), peak], fill=OL)
    d.polygon([(cx - 3, rc - 22), (cx + 3, rc - 22), (cx, rc - 28)], fill=GOLD[0], outline=OL)
    for (px_, py_) in ((cx - 28, cy), (cx + 28, cy), (cx, cy + 14)):
        d.rectangle([px_ - 1, py_ - ph + 3, px_ + 1, py_], fill=RED[0], outline=OL)
        d.line([(px_ + 1, py_ - ph + 4), (px_ + 1, py_ - 1)], fill=RED[1])
    return im, (cx, cy)


def milestone():
    im = Image.new("RGBA", (14, 24))
    d = ImageDraw.Draw(im)
    d.polygon([(3, 22), (3, 6), (7, 3), (11, 6), (11, 22), (7, 24)], fill=STONE[1], outline=OL)
    d.polygon([(7, 3), (11, 6), (11, 22), (7, 24)], fill=STONE[2], outline=OL)
    d.line([(5, 9), (5, 13)], fill=RED[0])
    d.line([(4, 11), (6, 11)], fill=RED[0])
    d.line([(5, 15), (5, 19)], fill=OL)
    return im, (7, 22)


def incense():
    im = Image.new("RGBA", (24, 28))
    d = ImageDraw.Draw(im)
    d.line([(7, 20), (4, 27)], fill=OL, width=2)
    d.line([(16, 20), (19, 27)], fill=OL, width=2)
    d.ellipse([4, 12, 19, 22], fill=(176, 130, 50, 255), outline=OL)
    d.ellipse([6, 12, 17, 16], fill=(90, 70, 40, 255), outline=OL)
    d.rectangle([2, 14, 4, 17], fill=(176, 130, 50, 255), outline=OL)
    d.rectangle([19, 14, 21, 17], fill=(176, 130, 50, 255), outline=OL)
    d.line([(11, 10), (11, 3)], fill=(200, 200, 205, 200))
    return im, (12, 26)


def ibox(d, ox, oy, Lu, Lv, h, top, sw, se):
    """an iso box from its back-bottom vertex (ox, oy): Lu px along u (down-right), Lv px along v (down-left); the SW and SE faces show"""
    C = lambda a, b, z=0: (ox + a - b, oy + (a + b) // 2 - z)
    d.polygon([C(0, Lv), C(Lu, Lv), C(Lu, Lv, h), C(0, Lv, h)], fill=sw, outline=OL)
    d.polygon([C(Lu, 0), C(Lu, Lv), C(Lu, Lv, h), C(Lu, 0, h)], fill=se, outline=OL)
    d.polygon([C(0, 0, h), C(Lu, 0, h), C(Lu, Lv, h), C(0, Lv, h)], fill=top, outline=OL)
    return C


def throne(orient):
    """a seat of state on a low platform standing against a wall. u: the wall runs along u and the seat faces SW; v: along v, facing SE.
    The platform is 3 x 2 cells, the back panel sits at the wall side, the pivot is the platform's centre on the ground."""
    big = 48, 32   # along the wall, away from the wall
    Lu, Lv = (big[0], big[1]) if orient == "u" else (big[1], big[0])
    ox, oy = 4 + Lv, 8 + 32                      # back vertex, leaving room for the back panel above
    im = Image.new("RGBA", (Lu + Lv + 8, 112))
    d = ImageDraw.Draw(im)
    ox, oy = 4 + Lv, 64
    ibox(d, ox, oy, Lu, Lv, 7, STONE[1], STONE[2], STONE[3])        # platform
    C = lambda a, b, z=0: (ox + a - b, oy + (a + b) // 2 - z)
    if orient == "u":
        # back panel along u at the wall side (v small), seat in front of it
        cx0, cx1 = Lu // 2 - 11, Lu // 2 + 11
        for a in range(cx0, cx1):
            x0, y0 = C(a, 3)
            d.line([(x0, y0 - 7), (x0, y0 - 7 - 30)], fill=RED[0] if (a // 3) % 2 == 0 else RED[1])
        d.polygon([C(cx0, 3, 7), C(cx1, 3, 7), C(cx1, 3, 38), C(cx0, 3, 38)], outline=OL)
        d.line([C(cx0, 3, 38), C(cx1, 3, 38)], fill=GOLD[0])
        ibox(d, *C(Lu // 2 - 6, 5, 7), 12, 12, 6, WOOD[0], WOOD[1], WOOD[2])
        ibox(d, *C(Lu // 2 - 6, 5, 13), 12, 3, 12, RED[0], RED[1], RED[2])
    else:
        for b in range(Lv // 2 - 11, Lv // 2 + 11):
            x0, y0 = C(3, b)
            d.line([(x0, y0 - 7), (x0, y0 - 7 - 30)], fill=RED[1] if (b // 3) % 2 == 0 else RED[2])
        cb0, cb1 = Lv // 2 - 11, Lv // 2 + 11
        d.polygon([C(3, cb0, 7), C(3, cb1, 7), C(3, cb1, 38), C(3, cb0, 38)], outline=OL)
        d.line([C(3, cb0, 38), C(3, cb1, 38)], fill=GOLD[0])
        ibox(d, *C(5, Lv // 2 - 6, 7), 12, 12, 6, WOOD[0], WOOD[1], WOOD[2])
        ibox(d, *C(5, Lv // 2 - 6, 13), 3, 12, 12, RED[0], RED[1], RED[2])
    # the pivot: platform centre on the ground
    pv = C(Lu // 2, Lv // 2)
    return im, pv


def avenue(axis):
    """a paved avenue cell (a 32x16 diamond): warm sandstone slabs running along `axis`, darker joints and a thin curb on both sides"""
    im = Image.new("RGBA", (32, 16))
    px = im.load()
    base = [(206, 194, 170, 255), (192, 178, 152, 255)]
    joint, curb = (150, 134, 108, 255), (120, 104, 82, 255)
    for y in range(16):
        for x in range(32):
            dx, dy = (x + .5 - 16) / 16, (y + .5 - 8) / 8
            if abs(dx) + abs(dy) > 1:
                continue
            a, b = (dx + dy) / 2 + .5, (dy - dx) / 2 + .5      # a along u (down-right), b along v (down-left)
            along, across = (a, b) if axis == "u" else (b, a)  # position along the avenue, across it
            c = base[int(along * 3) % 2]
            if across < .09 or across > .91:
                c = curb
            elif int(along * 3 + .02) != int(along * 3 - .02) or (along * 3) % 1 < .05:
                c = joint
            elif .47 < across < .53:
                c = joint
            px[x, y] = c
    return im, (16, 8)


def run():
    import gen_all
    reg = lambda n, f, layer="object", **kw: (lambda r: gen_all.save("tiles", n, r[0], r[1], layer, scale="scene", **kw))(f)
    for o in ("u", "v"):
        reg("screen_" + o, screen(o))
        reg("stairs_" + o, stairs(o))
    reg("bed", bed())
    reg("lantern", lantern(), glow=[8, 8])
    for sd in ("nw", "ne"):
        reg("curtain_" + sd, curtain(sd))
    reg("milestone", milestone())
    reg("incense", incense())
    for o in ("u", "v"):
        reg("avenue_" + o, avenue(o), layer="ground")
    for o in ("u", "v"):
        reg("throne_" + o, throne(o), cells="3x2")
