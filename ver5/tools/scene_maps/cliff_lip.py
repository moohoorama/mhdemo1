#!/usr/bin/env python3
"""The lip of a scene cliff's back edges (blip_ne_<k>, blip_nw_<k>): python3 ver5/tools/scene_maps/cliff_lip.py

A raised scene ground shows its rock faces only toward the camera (+u, +v); where its top drops away behind it (toward
-v, the upper right, or -u, the upper left) nothing marked the brink, and the top met the low ground beyond as if
they were level. This draws that brink, one small tile of edge per picture, as ground that rolls over and falls
away: nothing stands up past the edge, the corner is worn round, and the top shades off smoothly toward the brink as it
turns from the light, in bands along the edge blended with a checker dither, the band swelling and narrowing in long
waves so the brink does not run ruler-straight. The palette is the battle cliff block's
(lcliff_12_0).

Each picture is an ASCII grid first (printed with --show), one character a pixel:
  .  nothing      E  deep shade    D  shade      M  half lit    s  soil shade (half see-through)
Four variants per side tile with each other: every one meets the tile's corners with the same width. The upper-left
pictures are the upper-right ones mirrored. Writes assets_candidate/tiles/blip_<side>_<k>.png and their tiles.json
entries; ver5/tools/build_extra.py packs them, scene.json "heights" names the family ("back": "blip").
"""
import json
import random
import sys
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[3]
CAND = ROOT / 'assets_candidate'
PALETTE = {'K': (58, 43, 28, 255), 'E': (88, 67, 42, 255), 'D': (120, 93, 57, 255), 'M': (152, 121, 75, 255),
           'L': (187, 158, 106, 255), 'H': (216, 189, 134, 255), 's': (120, 93, 57, 110)}
X0, Y0, W, H = 14, -8, 20, 22   # the picture's box in the small tile's frame (its diamond's box is 0..32 x 0..16)
VARIANTS = 4


def edge_y(x):
    """The upper right edge of the small tile's diamond, from its top corner (16, 0) to its right corner (32, 8)."""
    return (x - 16) / 2


def lip_ne(seed):
    """The ASCII grid of an upper right lip: the ground's edge worn round, shading off smoothly toward the brink."""
    import math
    rnd = random.Random(seed)
    grid = [['.'] * W for _ in range(H)]

    def put(x, y, ch):
        gx, gy = x - X0, y - Y0
        if 0 <= gx < W and 0 <= gy < H:
            grid[gy][gx] = ch

    # the worn band's width swells and narrows in one or two long waves, 4 px at the tile's corners for every variant
    a, b = rnd.uniform(.6, 1.4), rnd.uniform(0, 2 * math.pi)
    width = {x: 4 + a * math.sin(math.pi * (x - 16) / 16) * math.sin(b + (x - 16) / 5) for x in range(15, 34)}
    shades = 'EDDMs'  # from the brink inward: deep shade, shade, half lit, soil
    for x in range(15, 34):
        line = round(edge_y(x))
        n = max(3, round(width[x]))
        for k in range(n + 1):
            t = k / n * (len(shades) - 1)              # where in the shades this row falls, between two of them
            lo = int(t)
            ch = shades[min(lo + 1, len(shades) - 1)] if (t - lo) > (.25 if (x + k) % 2 else .75) else shades[lo]
            put(x, line + k, ch)
    return [''.join(r) for r in grid]


def mirror(rows):
    return [r[::-1] for r in rows]


def image(rows):
    im = Image.new('RGBA', (W, H), (0, 0, 0, 0))
    for y, r in enumerate(rows):
        for x, ch in enumerate(r):
            if ch != '.':
                im.putpixel((x, y), PALETTE[ch])
    return im


def main():
    tiles_path = CAND / 'tiles/tiles.json'
    tiles = json.load(open(tiles_path))
    for k in range(VARIANTS):
        ne = lip_ne(1000 + k)
        for side, rows, pivot in (('ne', ne, [16 - X0, 8 - Y0]), ('nw', mirror(ne), [16 - (32 - X0 - W), 8 - Y0])):
            name = f'blip_{side}_{k}'
            image(rows).save(CAND / f'tiles/{name}.png')
            tiles['sprites'][name] = {'file': f'tiles/{name}.png', 'size': [W, H], 'pivot': pivot, 'layer': 'ground',
                                      'scale': 'scene', 'made': 'cliff back edge lip (cliff_lip.py)', 'shadow': False}
            if '--show' in sys.argv:
                print(name, *rows, sep='\n')
    json.dump(tiles, open(tiles_path, 'w'), ensure_ascii=False, indent=1)
    open(tiles_path, 'a').write('\n')


if __name__ == '__main__':
    main()
