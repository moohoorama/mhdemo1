#!/usr/bin/env python3
"""Banks of the battle map's water (docs/plan-naval-water.md): the water surface lies DEPTH canvas px below the
ground, so a water small tile shows the earth face of a land neighbour on its NW and NE edges, and a dark line on
its SE and SW edges where the land in front hides the lower water.

Small tile geometry (canvas px, native battle map): a 32 x 16 diamond around its centre. Each sprite's pivot is
that centre, so the game blits it at the water tile's centre.

  bank_{nw,ne}_{grass,soil}        the face under a land neighbour on that edge, 9 px tall, grass or soil lip
  bank_{nw,ne}_{grass,soil}_cut    the same, cut along the tile's own SW (nw face) or SE (ne face) edge, for when
                                   the neighbour there is land too (otherwise the face runs on into that water)
  bank_line_{se,sw}                1 px dark line inside the tile's SE / SW edge

  python3 assets_candidate/tools/gen_banks.py            -> assets_candidate/tiles/bank_*.png and tiles.json entries
  python3 assets_candidate/tools/gen_banks.py --review   -> output/banks-review.png (8x)
then python3 ver5/tools/build_extra.py packs them into the game's extra sheet.
"""
import json
import sys
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
CAND = ROOT / 'assets_candidate'
DEPTH = 9
EARTH = [(150, 112, 70), (128, 92, 58), (104, 74, 48), (82, 58, 40)]
LIP = {'grass': [(96, 132, 58), (70, 100, 44)], 'soil': [(176, 140, 92), (138, 104, 66)]}
FOAM = (200, 225, 225, 140)
WATER_SHADE = (10, 30, 50, 90)
LINE = (8, 22, 30, 140)


def mul(c, k):
    return tuple(min(255, max(0, round(v*k))) for v in c[:3])


def face(side, lip, cut):
    """side 'nw': x spans [cx-16, cx), the top edge rises from the left corner to the top corner.
    side 'ne': x spans [cx, cx+16), the top edge falls from the top corner to the right corner."""
    w, h = 16, 8 + DEPTH + 2
    im = Image.new('RGBA', (w, h))
    px = im.load()
    for x in range(w):
        # local y of the diamond's top edge at this column (pivot row 8 is the tile centre)
        top = 8 - (x + .5)/2 if side == 'nw' else (x + .5)/2
        # the tile's lower edge on the same side: SW edge (nw face) or SE edge (ne face)
        bottom = 8 + (x + .5)/2 if side == 'nw' else 16 - (x + .5)/2
        for y in range(h):
            t = y + .5 - top
            if t < 0 or (cut and y + .5 > bottom):
                continue
            row = int(t)
            if row >= DEPTH:
                if row <= DEPTH + 1:
                    px[x, y] = FOAM if row == DEPTH else WATER_SHADE
                continue
            band = (row + (x // 4) % 2) // 3
            c = EARTH[min(3, band % 3 + (1 if side == 'nw' else 0))]
            if (x*13 + row*7) % 23 == 0:
                c = mul(c, 1.18)  # a stone
            if row >= DEPTH - 2:
                c = mul(c, .72)  # wet foot
            if row < 2:
                c = LIP[lip][row]
                if side == 'nw':
                    c = mul(c, .85)
            elif side == 'ne' and row == 2:
                c = mul(c, 1.08)
            px[x, y] = c + (255,) if len(c) == 3 else c
    return im, (16 if side == 'nw' else 0, 8)


def line(side):
    """side 'se': x spans [cx, cx+16); 'sw': x spans [cx-16, cx). The line lies on the last row inside the edge;
    the tile centre is row 0."""
    im = Image.new('RGBA', (16, 9))
    px = im.load()
    for x in range(16):
        edge = 8 - (x + .5)/2 if side == 'se' else (x + .5)/2
        px[x, int(edge - .5)] = LINE
    return im, (0 if side == 'se' else 16, 0)


def sprites():
    out = {}
    for side in ('nw', 'ne'):
        for lip in LIP:
            for cut in (False, True):
                out[f'bank_{side}_{lip}' + ('_cut' if cut else '')] = face(side, lip, cut)
    for side in ('se', 'sw'):
        out[f'bank_line_{side}'] = line(side)
    return out


def review(path):
    """The sprites laid on a small patch of water and grass, 8x: a bay with banks on every side."""
    cells = ['..........', '...~~~~...', '..~~~~~~..', '..~~~~~~..', '...~~~~...', '....~~....', '..........']
    w, h = 360, 220
    im = Image.new('RGBA', (w, h), (110, 150, 70, 255))
    sp = sprites()
    water = lambda x, y: 0 <= y < len(cells) and 0 <= x < len(cells[0]) and cells[y][x] == '~'  # noqa: E731
    cen = lambda x, y: (w//2 + (x - y)*16, 30 + (x + y)*8)  # noqa: E731
    from PIL import ImageDraw
    d = ImageDraw.Draw(im)
    for y in range(len(cells)):
        for x in range(len(cells[0])):
            if water(x, y):
                cx, cy = cen(x, y)
                d.polygon([(cx - 16, cy), (cx, cy - 8), (cx + 16, cy), (cx, cy + 8)], fill=(40, 96, 118, 255))
    for y in range(len(cells)):
        for x in range(len(cells[0])):
            if not water(x, y):
                continue
            cx, cy = cen(x, y)
            for side, (dx, dy), (ex, ey) in (('nw', (-1, 0), (0, 1)), ('ne', (0, -1), (1, 0))):
                if not water(x + dx, y + dy):
                    s, p = sp[f'bank_{side}_grass' + ('' if water(x + ex, y + ey) else '_cut')]
                    im.alpha_composite(s, (cx - p[0], cy - p[1]))
            for side, (dx, dy) in (('se', (1, 0)), ('sw', (0, 1))):
                if not water(x + dx, y + dy):
                    s, p = sp[f'bank_line_{side}']
                    im.alpha_composite(s, (cx - p[0], cy - p[1]))
    im.resize((w*4, h*4), Image.NEAREST).save(path)
    print(path)


def main():
    if '--review' in sys.argv:
        review(ROOT / 'output/banks-review.png')
        return
    path = CAND / 'tiles/tiles.json'
    doc = json.loads(path.read_text())
    for name, (im, pivot) in sprites().items():
        im.save(CAND / f'tiles/{name}.png')
        doc['sprites'][name] = {'file': f'tiles/{name}.png', 'size': list(im.size), 'pivot': list(pivot),
                                'layer': 'ground', 'note': 'battle-map water banks (gen_banks.py)'}
    path.write_text(json.dumps(doc, indent=1, ensure_ascii=False) + '\n')


if __name__ == '__main__':
    main()
