#!/usr/bin/env python3
"""Marks a critical spell leaves on the battle map's ground for a few rounds (docs/logic-combat.md):

  charred_<v>  under 불씨 (fire): charred spots blurred into a soft dark patch, a dull red middle, a few live coals
  wet_<v>      물웅덩이 (water): two to four uneven damp patches, as after rain, with a streak or two of sky
  rubble_<v>   돌무더기 (earth): seven to ten small stones on darker soil that fades into whatever lies under them

Five of each, one per cell, picked by the cell. They are drawn at map pixels (a cell is a 32 x 16 diamond) with the
tileset's colours and doubled, so their dots are the tileset's size on the canvas. Every blur is twice as wide as it
is tall, like the map's ground. The pivot is the cell's centre.

  python3 assets_candidate/tools/gen_ground_marks.py   -> assets_candidate/tiles/{charred,wet,rubble}_*.png and tiles.json entries
then python3 ver5/tools/build_extra.py packs them into the game's extra sheet.
"""
import json
import math
import random
from pathlib import Path

from PIL import Image, ImageFilter

ROOT = Path(__file__).resolve().parents[2]
CAND = ROOT / 'assets_candidate'
VARIANTS = 5
W, H = 32, 16
OUTLINE = (20, 16, 18, 255)
ROCK = [(62, 56, 52), (102, 93, 84), (141, 130, 120), (176, 166, 152)]  # the map rocks' shades
SOIL = [(150, 124, 84), (138, 112, 76)]  # a little darker than the wasteland
WET = (40, 62, 74)
SKY = (170, 205, 210)
SOOT = (40, 30, 26)
COALS = [(255, 150, 40, 255), (255, 200, 80, 255), (230, 90, 30, 255)]


def inside(x, y, w=W, h=H, k=1.0):
    return abs(x + .5 - w / 2) / (w / 2 * k) + abs(y + .5 - h / 2) / (h / 2 * k) <= 1


def blur21(mask, radius):
    """A Gaussian blur twice as wide as it is tall: stretch the mask tall, blur, squash it back."""
    w, h = mask.size
    return mask.resize((w, h * 2), Image.NEAREST).filter(ImageFilter.GaussianBlur(radius)).resize((w, h), Image.BILINEAR)


def charred(seed):
    rnd = random.Random(seed)
    spots = Image.new('L', (W, H))
    sp = spots.load()
    hot = []
    for _ in range(rnd.randint(4, 6)):
        x, y = rnd.randrange(6, 26), rnd.randrange(4, 12)
        if inside(x, y, k=.8):
            hot.append((x, y))
            for dx in range(-rnd.randint(1, 2), rnd.randint(1, 3)):
                if 0 <= x + dx < W:
                    sp[x + dx, y] = 255
    char, glow = blur21(spots, 2.8).load(), blur21(spots, 1.2).load()
    im = Image.new('RGBA', (W, H))
    px = im.load()
    for y in range(H):
        for x in range(W):
            if not inside(x, y):
                continue
            a = min(210, int(char[x, y] * 3.4))
            if a > 10:
                px[x, y] = SOOT + (a,)
            g = glow[x, y] / 255
            if g > .25:  # the hot middle: soot warming to a dull red
                px[x, y] = (int(90 + 90 * g), int(40 + 30 * g), 24, max(px[x, y][3], int(120 + 100 * g)))
    for x, y in rnd.sample(hot, min(len(hot), 3)):
        px[x, y] = rnd.choice(COALS)
    return im


def wet(seed):
    rnd = random.Random(seed)
    ovals = []
    for _ in range(rnd.randint(2, 4)):
        x, y = rnd.uniform(7, 25), rnd.uniform(4, 12)
        size = rnd.uniform(.6, 1.2)
        for _ in range(rnd.randint(2, 5)):  # a patch wanders a little as it spreads
            ovals.append((x, y, rnd.uniform(2.5, 5) * size, rnd.uniform(1.2, 2.4) * size))
            x, y = x + rnd.uniform(-3.5, 3.5), y + rnd.uniform(-1.2, 1.2)
    im = Image.new('RGBA', (W, H))
    px = im.load()
    for y in range(H):
        for x in range(W):
            if not inside(x, y):
                continue
            k = max(1 - math.hypot((x + .5 - ox) / rx, (y + .5 - oy) / ry) for ox, oy, rx, ry in ovals)
            k += rnd.uniform(-.08, .08)
            if k > 0:
                px[x, y] = WET + (int(35 + 70 * min(1, k * 1.6)),)
    for _ in range(rnd.randint(1, 3)):
        for _ in range(40):
            x, y = rnd.randrange(3, 27), rnd.randrange(2, 14)
            if px[x, y][3] > 75:
                n = rnd.randint(2, 3)
                for i in range(n):
                    if x + i < W and px[x + i, y][3] > 50:
                        px[x + i, y] = SKY + (110 if i in (0, n - 1) else 150,)
                break
    return im


LIFT = 4  # rows above the diamond for the stones' tops


def stone(px, size, cx, by, r, rnd):
    """A lumpy rock sitting on (cx, by), r pixels wide each side, lit from the upper left, outlined."""
    hgt = r * rnd.uniform(.8, 1.1)
    pts = {}
    for y in range(int(by - hgt * 1.4) - 1, by + 1):
        for x in range(cx - r - 1, cx + r + 2):
            dx, dy = (x + .5 - cx) / (r + .3), (by + .3 - (y + .5)) / (hgt + .3)
            if dy < 0 or dx * dx + dy * dy > 1:
                continue
            light = -dx * .6 + dy * .8 + rnd.uniform(-.2, .2)
            pts[x, y] = ROCK[3] if light > .75 else ROCK[2] if light > .3 else ROCK[1] if light > -.2 else ROCK[0]
    for (x, y), c in pts.items():
        px[x, y] = c + (255,)
    for x, y in pts:
        for ox, oy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            q = (x + ox, y + oy)
            if q not in pts and 0 <= q[0] < size[0] and 0 <= q[1] < size[1] and px[q][3] < 255:
                px[q] = OUTLINE


def rubble(seed):
    rnd = random.Random(seed)
    size = (W, H + LIFT)
    im = Image.new('RGBA', size)
    px = im.load()
    spots = []
    want = rnd.randint(7, 10)
    for _ in range(400):
        if len(spots) == want:
            break
        x, y = rnd.uniform(5, 27), rnd.uniform(3, 13)
        if inside(x, y, k=.85) and all(math.hypot((x - a) / 2, y - b) > 1.8 for a, b, _ in spots):
            spots.append((x, y, rnd.choice([1, 1, 2, 2, 2])))
    for x, y, r in sorted(spots, key=lambda s: s[1]):
        stone(px, size, int(x), int(y) + LIFT, r, rnd)
    # soil under the stones: their mask, blurred, is its alpha
    mask = blur21(im.getchannel('A').point(lambda a: 255 if a else 0), 2.6).load()
    soil = Image.new('RGBA', size)
    sp = soil.load()
    for y in range(size[1]):
        for x in range(size[0]):
            a = min(255, int(mask[x, y] * 2.4))
            if a > 12:
                sp[x, y] = rnd.choice(SOIL) + (a,)
    soil.alpha_composite(im)
    return soil


def sprites():
    """name -> (image doubled, pivot)"""
    out = {}
    for v in range(VARIANTS):
        out[f'charred_{v}'] = (charred(30 + v), (W, H))
        out[f'wet_{v}'] = (wet(10 + v), (W, H))
        out[f'rubble_{v}'] = (rubble(20 + v), (W, H + 2 * LIFT))
    return {n: (im.resize((im.width * 2, im.height * 2), Image.NEAREST), p) for n, (im, p) in out.items()}


def main():
    path = CAND / 'tiles/tiles.json'
    doc = json.loads(path.read_text())
    for name, (im, pivot) in sprites().items():
        im.save(CAND / f'tiles/{name}.png')
        doc['sprites'][name] = {'file': f'tiles/{name}.png', 'size': list(im.size), 'pivot': list(pivot),
                                'layer': 'ground', 'note': 'battle-map ground mark of a critical spell (gen_ground_marks.py)'}
    path.write_text(json.dumps(doc, indent=1, ensure_ascii=False) + '\n')


if __name__ == '__main__':
    main()
