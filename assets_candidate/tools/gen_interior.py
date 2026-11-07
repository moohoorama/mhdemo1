"""Interior props for tent and hall scenes: a brazier (flame goes on top). The war-map table is a 3D model in
tools/unit3d/scene_props3d.py."""
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


m = gen_all.manifest["sprites"]
im, pv, anchor = brazier()
gen_all.save("tiles", "brazier", im, pv, "object", flame_anchor=list(anchor), note="draw one flame (k=1) at the anchor")
json.dump(gen_all.manifest, open(out("tiles", "tiles.json"), "w"), indent=1)
print(len(gen_all.manifest["sprites"]), "sprites")
