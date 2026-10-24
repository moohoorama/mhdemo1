#!/usr/bin/env python3
"""Corner shadows and cliff-foot shadows, dark colour and alpha only: python3 ver5/tools/scene_maps/join_overlay.py

The sun stands in the west-north-west. Where two faces meet in an inner corner that opens toward the camera (a wall
against the tall cliff behind it, a city wall turning a 'ㄱ' corner at the back), the face on the left (it runs along
v, lower left) is out of the sun and casts a shadow on the face on the right (it runs along u, lower right): a
triangle with straight, crisp edges, full height at the corner, its foot running SPREAD x the height along the right
face's foot. A 1px dark line marks the corner itself. The triangle is cut into upright slices half a cell (16 px)
wide, jshade_<height>_<j>, height in steps of 8 px; slice j stands on the right face's foot 16 j px right of the
corner, so it sorts just after the piece of the right face it lies on and before the people in front.

A cliff block's sunlit faces still meet the ground with no shadow: cfoot_sw / cfoot_se is a soft band on the ground
along the foot of a small tile's lower left / lower right face (on the SE face, away from the sun, it is wider: that
face throws its shadow on the ground in front). They lie on the ground layer, anchored on the tile's centre.

compose.corner_shadow and compose.cliff_feet place them. The shapes are drawn pixel by pixel, the alphas set here.
Writes assets_candidate/tiles/jshade_*.png, cfoot_*.png and their tiles.json entries (only these).
"""
import json
import math
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[3]
CAND = ROOT / 'assets_candidate'
INK = (11, 16, 27)
SPREAD = .8          # the shadow's foot along the right face, per px of height (the sun's height and bearing)
SHADE, LINE = 96, 205
HEIGHTS = range(8, 161, 8)


def triangle(H):
    """{(x, y): alpha} of the corner shadow of height H, x right of the corner, y from the corner's foot (down +)"""
    L = SPREAD * H
    px = {}
    for x in range(0, int(math.ceil(L)) + 1):
        foot = .5 * x                       # the right face's foot drops half a px per px
        top = -H + (H / L) * x + .5 * x     # the hypotenuse from the corner's top to the shadow's far foot
        for y in range(int(math.ceil(top)), int(math.floor(foot)) + 1):
            px[(x, y)] = LINE if x == 0 else int(SHADE * (1 - .25 * x / L))
    return px


def slices(px):
    out = {}
    for (x, y), a in px.items():
        out.setdefault(x // 16, {})[(x % 16, y - 8 * (x // 16))] = a  # each slice relative to its own foot
    return out


def picture(px, pivot_at=(0, 0)):
    xs = [x for x, _ in px] + [pivot_at[0]]
    ys = [y for _, y in px] + [pivot_at[1]]
    x0, y0 = min(xs), min(ys)
    im = Image.new('RGBA', (max(xs) - x0 + 1, max(ys) - y0 + 1))
    for (x, y), a in px.items():
        im.putpixel((x - x0, y - y0), (*INK, a))
    return im, [pivot_at[0] - x0, pivot_at[1] - y0]


def foot_band(face):
    """the ground band along a small tile's lower face, relative to the tile's centre: sw runs from its left corner
    (-16, 0) to its bottom corner (0, 8), se from the bottom corner to its right corner (16, 0)"""
    px = {}
    depth = (3, 2) if face == 'sw' else (6, 3)  # px of the darker band, px of the fading edge
    for x in range(-16, 1) if face == 'sw' else range(0, 17):
        foot = 8 + .5 * x if face == 'sw' else 8 - .5 * x
        f = int(math.floor(foot)) + 1
        for d in range(depth[0]):
            px[(x, f + d)] = 72
        for d in range(depth[1]):
            px[(x, f + depth[0] + d)] = 40 - 12 * d
    return px


def main():
    made = {}

    def save(name, im, pivot, layer='object'):
        im.save(CAND / 'tiles' / f'{name}.png')
        made[name] = dict(file=f'tiles/{name}.png', size=list(im.size), pivot=pivot, layer=layer, scale='scene',
                          made='alpha-only shadow (join_overlay.py)', shadow=False)

    for H in HEIGHTS:
        for j, px in slices(triangle(H)).items():
            im, pv = picture(px)
            save(f'jshade_{H}_{j}', im, pv)
    for face in ('sw', 'se'):
        im, pv = picture(foot_band(face))
        save(f'cfoot_{face}', im, pv, 'ground')
    path = CAND / 'tiles/tiles.json'
    man = json.load(open(path))  # read just before writing: other tools write this file too
    for n in [n for n in man['sprites'] if n.startswith('join_')]:  # the first, dotted join overlays
        f = CAND / man['sprites'][n]['file']
        if f.exists():
            f.unlink()
        del man['sprites'][n]
    man['sprites'].update(made)
    json.dump(man, open(path, 'w'), ensure_ascii=False, indent=1)
    open(path, 'a').write('\n')
    print(len(made), 'shadow pieces')


if __name__ == '__main__':
    main()
