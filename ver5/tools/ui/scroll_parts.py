#!/usr/bin/env python3
"""Draw the dialogue scroll parts (docs/ui-mockup, idea 5) into assets/graphics/ui/scroll.

One pixel here is one art pixel; the game scales them up with nearest filtering.
  rod-top.png, rod.png, rod-bottom.png  the roller: knob and collar, a body row repeated
                                        down, collar and knob
  band.png                              brocade edge, repeated across the top and bottom
  paper.png                             paper, repeated in both directions

    python3 tools/ui/scroll_parts.py
"""
import pathlib
import random

from PIL import Image

OUT = pathlib.Path(__file__).resolve().parents[2] / "assets" / "graphics" / "ui" / "scroll"

ROD = 12
GOLD, GOLD_L, GOLD_D, GOLD_X = (214, 170, 85), (242, 213, 140), (134, 101, 44), (74, 52, 18)
RODC = [(26, 12, 4), (52, 27, 10), (90, 50, 22), (138, 84, 40), (182, 122, 66), (217, 160, 98),
        (232, 184, 120), (192, 134, 72), (142, 90, 42), (94, 54, 24), (56, 32, 12), (30, 16, 6)]
BAND, BAND2, BAND3 = (31, 52, 54), (43, 69, 71), (56, 88, 90)
PAPER, FIBER_L, FIBER_D = (243, 234, 211), (251, 245, 228), (226, 213, 182)


def knob(img, y):
    for i, (o, w) in enumerate([(3, 10), (1, 14), (0, 16), (0, 16), (1, 14), (3, 10)]):
        for x in range(w):
            t = x / (w - 1)
            c = GOLD_L if t < .2 else GOLD if t < .55 else GOLD_D if t < .85 else GOLD_X
            img.putpixel((o + x, y + i), c + (255,))
    img.putpixel((4, y + 2), (255, 244, 200, 255))
    img.putpixel((5, y + 2), (255, 244, 200, 255))


def collar(img, y):
    for x in range(ROD):
        img.putpixel((x + 2, y), GOLD_D + (255,))
    for x in range(1, ROD - 2):
        img.putpixel((x + 2, y), GOLD + (255,))
    for x in range(ROD):
        img.putpixel((x + 2, y + 1), GOLD_D + (255,))


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    w = ROD + 4
    top = Image.new("RGBA", (w, 8))
    knob(top, 0)
    collar(top, 6)
    top.save(OUT / "rod-top.png")
    body = Image.new("RGBA", (w, 1))
    for x in range(ROD):
        body.putpixel((x + 2, 0), RODC[x] + (255,))
    body.save(OUT / "rod.png")
    bottom = Image.new("RGBA", (w, 8))
    collar(bottom, 0)
    knob(bottom, 2)
    bottom.save(OUT / "rod-bottom.png")

    band = Image.new("RGBA", (8, 10), BAND + (255,))
    for x in range(8):
        band.putpixel((x, 0), GOLD_D + (255,))
        band.putpixel((x, 1), GOLD + (255,))
        band.putpixel((x, 8), GOLD + (255,))
        band.putpixel((x, 9), GOLD_D + (255,))
    for x, y in [(3, 3), (2, 4), (4, 4), (3, 5), (1, 5), (5, 5), (2, 6), (4, 6), (3, 7)]:
        band.putpixel((x + 1, y - 1), (GOLD_L if (x, y) == (3, 5) else GOLD) + (255,))
    for x, y, c in [(0, 4, BAND3), (7, 4, BAND3), (0, 5, BAND2), (7, 5, BAND2)]:
        band.putpixel((x, y), c + (255,))
    band.save(OUT / "band.png")

    r = random.Random(3)
    paper = Image.new("RGBA", (24, 24), PAPER + (255,))
    for _ in range(14):
        x, y, n = r.randrange(24), r.randrange(24), 2 + r.randrange(3)
        c = FIBER_L if r.random() < .5 else FIBER_D
        for k in range(n):
            paper.putpixel(((x + k) % 24, y), c + (255,))
    for _ in range(10):
        paper.putpixel((r.randrange(24), r.randrange(24)), FIBER_D + (255,))
    paper.save(OUT / "paper.png")
    print("wrote", OUT)


if __name__ == "__main__":
    main()
