"""The game's wall cells with the seams removed: where two cells join, the sprites carry a short dark vertical line (the outline of the
joint), which shows as a tick along the wall. Dark vertical runs with face colour on both sides are painted over, except the
corner edge at the bottom vertex when both front sides are open."""
from common import *
from PIL import Image

OL = (11, 16, 27, 255)


def clean(im, mask):
    im = im.copy()
    px = im.load()
    for x in range(1, im.width - 1):
        y = 0
        while y < im.height:
            if px[x, y] == OL:
                e = y
                while e < im.height and px[x, e] == OL:
                    e += 1
                ok = e - y >= 3 and all(px[x - 1, yy][3] and px[x + 1, yy][3] and px[x - 1, yy] != OL and px[x + 1, yy] != OL for yy in range(y, e))
                if ok and not (x == 21 and mask & 12 == 12):
                    for yy in range(y, e):
                        px[x, yy] = px[x - 1, yy]
                y = e
            else:
                y += 1
    # the end outlines of a cell that continues to the NW / SE neighbour: the neighbour overlaps them, leaving a tick along the wall
    for side, xs, dx in ((0, range(2, 10), 1), (2, range(17, 26), -1)):
        if mask >> side & 1:
            continue
        for x in xs:
            y = 0
            while y < im.height:
                if px[x, y] == OL and px[x + dx, y][3] and px[x + dx, y] != OL:
                    e = y
                    while e < im.height and px[x, e] == OL and px[x + dx, e][3] and px[x + dx, e] != OL:
                        e += 1
                    outer = [px[x - dx, yy][3] == 0 or px[x - dx, yy] != OL for yy in range(y, e)]
                    if e - y >= 3 and all(outer):
                        for yy in range(y, e):
                            px[x, yy] = px[x + dx, yy]
                    y = max(e, y + 1)
                else:
                    y += 1
    return im


def run():
    import gen_all
    for m in range(16):
        im, pv = structure("wall_%02d" % m)
        gen_all.save("tiles", "wall_clean_%02d" % m, clean(im, m), pv, "object", scale="battle", mask=m, note="the game's wall cell without the joint seams")
