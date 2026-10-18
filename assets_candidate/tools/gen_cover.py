"""Ground cover drawn over a unit's feet when it stands in a forest: low leafy mounds in the forest sprites' own palette."""
import random
from PIL import Image, ImageDraw
import gen_all

INK, DARK, MID, LIGHT = (11, 16, 27, 255), (34, 48, 26, 255), (64, 88, 38, 255), (96, 124, 52, 255)


def mound(w, h, seed, lobes):
    """a bush w x h: overlapping lobes, flat at the bottom; dark body, lit top-left of every lobe, ink outline, a few leaf tips"""
    rnd = random.Random(seed)
    body = Image.new("L", (w, h), 0)
    lit = Image.new("L", (w, h), 0)
    bd, ld = ImageDraw.Draw(body), ImageDraw.Draw(lit)
    base = h - 3
    r0 = h * .38
    span = w - 2 * r0 - 2
    for i in range(lobes):
        r = r0 * rnd.uniform(.85, 1.15) * (1.15 if 0 < i < lobes - 1 else .9)
        cx = r0 + 1 + (span * i / (lobes - 1) if lobes > 1 else span / 2)
        cy = base - r * rnd.uniform(.55, .85)
        bd.ellipse([cx - r, cy - r, cx + r, cy + r], fill=255)
        ld.ellipse([cx - r * .8, cy - r * .85, cx + r * .15, cy - r * .05], fill=255)
    bd.rectangle([r0 * .6, base - r0 * .8, w - r0 * .6, base], fill=255)
    bd.rectangle([0, base + 1, w, h], fill=0)                      # a flat base
    out = Image.new("RGBA", (w, h))
    px, bp, lp = out.load(), body.load(), lit.load()
    for y in range(h):
        for x in range(w):
            if not bp[x, y]:
                continue
            c = LIGHT if lp[x, y] else (MID if y < base - 4 else DARK)
            px[x, y] = c
    for x in range(2, w - 2, 3):                                    # a few leaf tips on the top edge
        for y in range(1, h - 1):
            if bp[x, y] and not bp[x, y - 1]:
                if rnd.random() < .6:
                    px[x, y - 1] = MID
                    bp[x, y - 1] = 255
                break
    edge = Image.new("RGBA", (w, h))
    ep = edge.load()
    for y in range(h):
        for x in range(w):
            if bp[x, y]:
                ep[x, y] = px[x, y]
                continue
            if any(0 <= x + a < w and 0 <= y + b < h and bp[x + a, y + b] for a, b in ((1, 0), (-1, 0), (0, 1), (0, -1))):
                ep[x, y] = INK
    return edge, (w // 2, base)


def run():
    for name, (w, h, seed, lobes) in {"cover_bush_0": (30, 17, 3, 3), "cover_bush_1": (36, 20, 8, 4), "cover_bush_2": (24, 14, 5, 2)}.items():
        im, pv = mound(w + 2, h + 2, seed, lobes)
        gen_all.save("tiles", name, im, (pv[0], pv[1]), "object", scale="battle", note="drawn over a unit's feet on a forest tile")


if __name__ == "__main__":
    import sys
    run()
