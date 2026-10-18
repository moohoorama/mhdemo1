"""Closed gate: the game's gate house with its passage filled by two studded wooden doors."""
from common import *
from PIL import Image, ImageDraw

OUT = (26, 18, 22, 255)
WOOD = [(138, 88, 50, 255), (112, 70, 40, 255), (82, 50, 30, 255)]
IRON = (176, 182, 194, 255)
# the passage seen from the front: top-left, top-right, bottom-right, bottom-left (gate_u; gate_v mirrors it)
POLY_U = [(13, 45), (25, 51), (25, 70), (14, 75)]


def closed(name):
    im, pivot = prop(name)
    w, h = im.size
    poly = POLY_U if name == "gate_u" else [(w - 1 - x, y) for x, y in POLY_U]
    mask = Image.new("L", im.size, 0)
    ImageDraw.Draw(mask).polygon(poly, fill=255)
    mpx, px = mask.load(), im.load()
    xs = [p[0] for p in poly]
    x0, x1 = min(xs), max(xs)
    left_high = name == "gate_u"  # in gate_u the top edge falls to the right
    for y in range(h):
        for x in range(w):
            if not mpx[x, y]:
                continue
            u = (x - x0) if left_high else (x1 - x)
            slope = u * 6 // 12
            c = WOOD[0] if u % 4 in (0, 1) else WOOD[1]
            if u in (5, 6, 7):  # the seam between the two leaves
                c = OUT if u == 6 else WOOD[2]
            if (y - slope) % 13 in (0, 1):
                c = IRON if u % 3 else WOOD[2]
            if u == 0 or u == x1 - x0:
                c = WOOD[2]
            px[x, y] = c
    return im, pivot


if __name__ == "__main__":
    for n in ("gate_u", "gate_v"):
        im, pivot = closed(n)
        im.save(out("tiles", "gate_closed_%s.png" % n[-1]))
