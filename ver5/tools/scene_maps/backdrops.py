#!/usr/bin/env python3
"""Draw the far pictures that lie behind outdoor scene maps (a scene's "Backdrop"): sky and ridges, a pass with its
gate, a city wall, Luoyang burning. Small pixel pictures (the game scales them up without smoothing): a banded sky
with clouds, three ridges paler with distance and lit on one side, mist between them, a tree line on the nearest,
and for the man-made ones walls, towers with curved eaves and banners. No outlines: they are far away. Writes
assets/graphics/backdrops/<id>.png and the "backdrops" table of assets/graphics/index.json.

  python3 ver5/tools/scene_maps/backdrops.py
"""
import json
import math
import random
from pathlib import Path

from PIL import Image

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
W, H = 320, 120

DAY_SKY = [(132, 164, 192), (150, 180, 202), (170, 194, 208), (190, 206, 210), (206, 214, 208)]
CLOUD = [(232, 236, 236), (212, 222, 226), (188, 202, 212)]
RIDGES = [  # far to near: body, lit side, dark side
    ((134, 152, 170), (152, 168, 182), (118, 136, 156)),
    ((104, 126, 140), (124, 146, 152), (88, 108, 126)),
    ((78, 102, 92), (96, 122, 100), (60, 82, 76)),
]
MIST = (176, 192, 198)
TREE = (52, 74, 60)
WALL, WALL_LIT, WALL_DARK = (118, 108, 98), (140, 128, 114), (82, 74, 70)
ROOF, ROOF_LIT = (58, 52, 60), (86, 78, 84)
RED = (170, 40, 44)


class Pic:
    def __init__(self):
        self.im = Image.new('RGBA', (W, H))

    def put(self, x, y, c):
        if 0 <= x < W and 0 <= y < H:
            self.im.putpixel((int(x), int(y)), (*c, 255))

    def get(self, x, y):
        return self.im.getpixel((x, y))[:3]

    def rect(self, x0, y0, x1, y1, c):
        for y in range(int(y0), int(y1)):
            for x in range(int(x0), int(x1)):
                self.put(x, y, c)


def sky(p, bands):
    for y in range(H):
        k = y / H * (len(bands) - 1)
        i = int(k)
        for x in range(W):
            c = bands[i]
            if i + 1 < len(bands) and k - i > .5 and (x + y) % 2 == 0:  # dithered seam between bands
                c = bands[i + 1]
            p.put(x, y, c)


def clouds(p, seed, n=5, top=6, colors=CLOUD):
    rnd = random.Random(seed)
    for _ in range(n):
        cx, cy, w = rnd.randrange(W), rnd.randrange(top, top + 26), rnd.randrange(26, 60)
        for x in range(cx - w // 2, cx + w // 2):
            k = 1 - abs(x - cx) / (w / 2)
            h = int(2 + 5 * k + 2 * math.sin(x * .4 + cx))
            for y in range(cy - h, cy + 2):
                c = colors[0] if y < cy - h + 2 else colors[1] if y < cy else colors[2]
                p.put(x % W, y, c)


def ridge_line(seed, base, amp, rough=1.0):
    rnd = random.Random(seed)
    phases = [rnd.random() * 6 for _ in range(4)]
    return [base - amp * (.5 * math.sin(x * .018 + phases[0]) + .28 * math.sin(x * .047 + phases[1])
                          + .14 * math.sin(x * .11 + phases[2]) * rough + .08 * math.sin(x * .29 + phases[3]) * rough)
            for x in range(W)]


def mountain(p, line, colors, trees=False, seed=0):
    body, lit, dark = colors
    rnd = random.Random(seed)
    for x in range(W):
        top = int(line[x])
        slope = line[min(W - 1, x + 1)] - line[max(0, x - 1)]  # rising to the right: the left face is sunlit
        for y in range(max(0, top), H):
            c = body
            if y - top < 6:
                c = lit if slope > .3 else dark if slope < -.3 else body
            p.put(x, y, c)
        if trees and rnd.random() < .55:  # a conifer on the skyline
            h = rnd.randrange(3, 7)
            for dy in range(h):
                for dx in range(-(dy // 2), dy // 2 + 1):
                    p.put(x + dx, top - h + dy, TREE)


def mist(p, y0, h):
    for y in range(y0, y0 + h):
        for x in range(W):
            if (x * 7 + y * 13) % (2 + abs(y - y0 - h // 2)) == 0:
                p.put(x, y, MIST)


def ridges(p, seed=1, near_trees=True):
    mountain(p, ridge_line(seed, 62, 26), RIDGES[0], seed=seed)
    mist(p, 70, 5)
    mountain(p, ridge_line(seed + 3, 84, 18), RIDGES[1], seed=seed + 1)
    mist(p, 90, 4)
    mountain(p, ridge_line(seed + 7, 104, 9, rough=1.6), RIDGES[2], trees=near_trees, seed=seed + 2)


def tower(p, cx, base, w, h, tiers=2):
    """A gate tower: stone base with the gate, timber storeys under curved eaves, a banner on the ridge."""
    p.rect(cx - w // 2, base - h, cx + w // 2, base, WALL)
    p.rect(cx - w // 2, base - h, cx - w // 2 + 2, base, WALL_LIT)
    p.rect(cx + w // 2 - 2, base - h, cx + w // 2, base, WALL_DARK)
    p.rect(cx - 3, base - 9, cx + 3, base, (40, 34, 34))  # the gate
    for x in range(cx - 3, cx + 3):
        p.put(x, base - 10 + abs(x - cx + .5) // 2, (40, 34, 34))
    top = base - h
    for t in range(tiers):
        half = w // 2 + 4 - t * 4
        p.rect(cx - half + 4, top - 7, cx + half - 4, top, (96, 60, 46))  # timber storey
        for x in range(cx - half + 6, cx + half - 6, 4):
            p.put(x, top - 4, (210, 170, 90))  # window lights
        for x in range(cx - half, cx + half):  # eaves: flat in the middle, swept up at the tips
            e = max(0, abs(x - cx) - (half - 5))
            for y in range(top - 10 - e // 2, top - 6):
                p.put(x, y, ROOF_LIT if y == top - 10 - e // 2 else ROOF)
        top -= 11
    p.rect(cx, top - 6, cx + 1, top + 1, (60, 44, 34))  # banner pole and flag
    p.rect(cx + 1, top - 6, cx + 6, top - 3, RED)


def wall(p, y, h, crenel=True):
    for x in range(W):
        for yy in range(y, y + h):
            p.put(x, yy, WALL_LIT if yy == y else WALL if yy < y + h - 2 else WALL_DARK)
        if crenel and x % 6 < 3:
            for yy in range(y - 3, y):
                p.put(x, yy, WALL)


def mountains():
    p = Pic()
    sky(p, DAY_SKY)
    clouds(p, 4)
    ridges(p, 1)
    return p.im


def pass_gate():
    p = Pic()
    sky(p, DAY_SKY)
    clouds(p, 9, n=4)
    far = ridge_line(2, 56, 30)
    far = [v - (0 if abs(x - 160) > 54 else 26 * math.cos((x - 160) / 54 * math.pi / 2)) * -1 for x, v in enumerate(far)]
    mountain(p, [min(v + (0 if abs(x - 160) > 54 else 30 * math.cos((x - 160) / 54 * math.pi / 2)), 100)
                 for x, v in enumerate(ridge_line(2, 50, 24))], RIDGES[0], seed=2)
    mist(p, 78, 5)
    wall(p, 80, 14)
    tower(p, 160, 94, 30, 16)
    for bx in (120, 136, 184, 200):  # banners along the wall
        p.rect(bx, 70, bx + 1, 80, (60, 44, 34))
        p.rect(bx + 1, 70, bx + 5, 75, RED)
    mountain(p, [min(v, 98 + abs(x - 160) // 12) if abs(x - 160) < 70 else v for x, v in enumerate(ridge_line(8, 104, 12, 1.6))],
             RIDGES[2], trees=True, seed=5)
    return p.im


def city_wall():
    p = Pic()
    sky(p, DAY_SKY)
    clouds(p, 13, n=4)
    mountain(p, ridge_line(3, 58, 18), RIDGES[0], seed=3)
    mist(p, 66, 4)
    for x0, h in ((40, 8), (210, 10), (260, 7)):  # roofs of the city behind the wall
        for x in range(x0, x0 + 30):
            for y in range(84 - h + abs(x - x0 - 15) // 4, 86):
                p.put(x, y, ROOF if y > 84 - h + 2 else ROOF_LIT)
    wall(p, 86, 22)
    for cx in (70, 250):
        tower(p, cx, 90, 16, 10, tiers=1)
    tower(p, 160, 108, 40, 24)
    mountain(p, [112 + 2 * math.sin(x * .2) for x in range(W)], RIDGES[2], trees=False, seed=9)
    return p.im


def burning_city():
    p = Pic()
    sky(p, [(46, 30, 44), (82, 36, 44), (132, 50, 42), (184, 82, 46), (220, 120, 60)])
    rnd = random.Random(11)
    mountain(p, ridge_line(5, 70, 12), ((92, 50, 50), (110, 62, 56), (74, 40, 44)), seed=5)
    base = 96
    for layer, (y0, shade) in enumerate(((86, (54, 34, 40)), (92, (38, 26, 32)))):  # two rows of roofs
        x = -10 if layer else 0
        while x < W:
            w, h = rnd.randrange(14, 30), rnd.randrange(6, 14)
            for xx in range(x, x + w):
                k = abs(xx - x - w / 2) / (w / 2)
                for y in range(int(y0 - h + 4 * k * k), H):
                    p.put(xx, y, shade)
            x += w - rnd.randrange(2, 6)
    wall(p, base, 24, crenel=True)
    for x in range(W):  # the wall darkened, lit red from the fires behind
        for y in range(base, H):
            r, g, b = p.get(x, y)
            p.put(x, y, (r * 2 // 3 + 20, g // 2, b // 2))
    fires, x = [], 8
    while x < W:  # fires at uneven spacing, big and small
        fires.append(x)
        x += rnd.randrange(14, 44)
    for i, fx in enumerate(fires):
        h = rnd.randrange(5, 18)
        smoky = h > 9 or rnd.random() < .3
        for y in range(base - 6 - h, base - 4):  # tongues of flame over the roofline
            k = (base - 4 - y) / (h + 2)
            half = max(1, int((1 - k) * (3 + h / 2.5) + math.sin(y * .9 + i) * 1.5))
            col = (255, 230, 140) if k < .3 else (250, 170, 70) if k < .65 else (210, 90, 40)
            for x in range(fx - half, fx + half + 1):
                p.put(x, y, col)
        for j in range(26 if smoky else 0):  # a smoke column leaning with the wind
            cy = base - 12 - h - j * 3
            cx = fx + int(j * 1.4 + 4 * math.sin(j * .5 + i))
            r = 2 + j // 4 + h // 8
            for y in range(cy - r, cy + r):
                for x in range(cx - r, cx + r):
                    if (x - cx) ** 2 + (y - cy) ** 2 <= r * r and 0 <= y < H:
                        g = 52 + (j * 2) + (8 if (x + y) % 3 == 0 else 0)
                        p.put(x, y, (g + 6, g - 6, g - 4))
    for _ in range(80):  # embers
        p.put(rnd.randrange(W), rnd.randrange(20, base), (255, 196, 90))
    return p.im


BACKDROPS = {'mountains': mountains, 'pass': pass_gate, 'city_wall': city_wall, 'burning_city': burning_city}


def main():
    out = GFX / 'backdrops'
    out.mkdir(exist_ok=True)
    index_path = GFX / 'index.json'
    index = json.load(open(index_path))
    index['backdrops'] = {}
    for name, draw in BACKDROPS.items():
        draw().save(out / f'{name}.png')
        index['backdrops'][name] = f'backdrops/{name}.png'
        print(name)
    json.dump(index, open(index_path, 'w'), ensure_ascii=False, indent=1)
    open(index_path, 'a').write('\n')


if __name__ == '__main__':
    main()
