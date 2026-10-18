"""Interior props for tent and hall scenes: a brazier (flame goes on top) and a war-map table."""
import json
from common import *
from PIL import Image, ImageDraw
import gen_all  # noqa: F401  (also registers the manifest it writes)

O = (26, 18, 22, 255)


def brazier():
    im = Image.new("RGBA", (28, 28))
    d = ImageDraw.Draw(im)
    for x0, y0, x1, y1 in ((8, 19, 5, 26), (20, 19, 23, 26), (14, 21, 14, 26)):
        d.line([x0, y0, x1, y1], fill=O, width=2)
        d.line([x0, y0, x1, y1], fill=(70, 62, 66, 255), width=1)
    d.polygon([(4, 12), (24, 12), (21, 21), (7, 21)], fill=(92, 96, 108, 255), outline=O)
    d.polygon([(5, 13), (9, 13), (10, 20), (8, 20)], fill=(132, 138, 150, 255))
    d.line([(7, 17), (21, 17)], fill=(60, 62, 72, 255))
    d.ellipse([3, 8, 24, 16], fill=(150, 66, 28, 255), outline=O)
    d.ellipse([6, 10, 21, 14], fill=(236, 124, 40, 255))
    d.ellipse([9, 11, 18, 13], fill=(255, 206, 96, 255))
    return im, (14, 24), (14, 11)


def map_table():
    im = Image.new("RGBA", (48, 36))
    d = ImageDraw.Draw(im)
    cx, cy = 24, 16
    top = [(cx, cy - 10), (cx + 22, cy), (cx, cy + 10), (cx - 22, cy)]
    for dx in (-17, 17):
        d.line([(cx + dx, cy + 4), (cx + dx, cy + 15)], fill=O, width=3)
        d.line([(cx + dx, cy + 4), (cx + dx, cy + 15)], fill=(96, 62, 36, 255), width=1)
    d.polygon([(cx - 22, cy), (cx, cy + 10), (cx, cy + 15), (cx - 22, cy + 5)], fill=(104, 66, 38, 255), outline=O)
    d.polygon([(cx + 22, cy), (cx, cy + 10), (cx, cy + 15), (cx + 22, cy + 5)], fill=(80, 50, 30, 255), outline=O)
    d.polygon(top, fill=(142, 92, 52, 255), outline=O)
    inner = [(cx, cy - 7), (cx + 16, cy), (cx, cy + 7), (cx - 16, cy)]
    d.polygon(inner, fill=(226, 208, 164, 255), outline=(150, 118, 70, 255))
    d.line([(cx - 10, cy - 1), (cx - 3, cy + 1), (cx + 4, cy - 2), (cx + 11, cy + 1)], fill=(70, 120, 150, 255))
    for x, y, c in ((cx - 7, cy - 2, (170, 40, 34)), (cx + 5, cy + 2, (170, 40, 34)), (cx + 1, cy - 3, (30, 30, 34)), (cx - 2, cy + 3, (30, 30, 34))):
        d.rectangle([x, y, x + 1, y + 1], fill=c + (255,))
    return im, (24, 20)


m = gen_all.manifest["sprites"]
im, pv, anchor = brazier()
gen_all.save("tiles", "brazier", im, pv, "object", flame_anchor=list(anchor), note="draw one flame (k=1) at the anchor")
im, pv = map_table()
gen_all.save("tiles", "map_table", im, pv, "object")
json.dump(gen_all.manifest, open(out("tiles", "tiles.json"), "w"), indent=1)
print(len(gen_all.manifest["sprites"]), "sprites")
