#!/usr/bin/env python3
"""Walls and crossings of the battle map's ground (docs/plan-tile-floors.md). Floors lie lowered by their depth (land
0, ford FORD, water DEPTH); a wall stands on a small tile's NW or NE edge where the tile behind lies higher, from the
back tile's depth down to this tile's.

Pieces drawn in a 128 x 128 frame (tiles.json "frame": 128) have the small tile's centre at (64, 64);
ver5/tools/build_extra.py trims them and works out their pivots. The small tile is a 32 x 16 diamond.

  wall_{kind}_{side}_{top}{bottom}_{d2}   the wall under the tile's `side` (nw, ne) edge, rows top..bottom below it:
                                          kind grass / soil (earth under land with that lip), stone (a bridge's
                                          abutment), weir (under a ford: water slipping down between stones). d2 is the
                                          depth of the side neighbour in front (the SW one for nw, the SE one for ne):
                                          rows deeper than it are cut away over its lowered diamond.
  ford_bed_{0..7}                         pebbles, stones, now and then flat stepping stones and glints over a ford
  bridge_edge_{se,sw}                     a bridge small tile's front edge over water: side beam, a pier, a rail
  bridge_rail_{nw,ne}                     its back edge over water: the rail

  python3 assets_candidate/tools/gen_banks.py            -> assets_candidate/tiles/*.png and tiles.json entries
  python3 assets_candidate/tools/gen_banks.py --review   -> output/walls-review.png (8x)
then python3 ver5/tools/build_extra.py packs them into the game's extra sheet.
"""
import json
import random
import sys
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
CAND = ROOT / 'assets_candidate'
DEPTH = 9
FORD = 5
FORD_BEDS = 8
F, C = 128, 64  # frame size, the small tile's centre in it
EARTH = [(150, 112, 70), (128, 92, 58), (104, 74, 48), (82, 58, 40)]
LIP = {'grass': [(96, 132, 58), (70, 100, 44)], 'soil': [(176, 140, 92), (138, 104, 66)]}
STONE = [(150, 146, 134), (128, 124, 114), (104, 100, 92), (82, 78, 72)]
WOOD = [(150, 92, 52), (112, 64, 34), (78, 44, 24)]
OUTLINE = (26, 18, 22)
FOAM = (200, 225, 225, 170)
FALL = [(150, 200, 210), (112, 168, 182)]
WET_STONE = (58, 92, 100)
SPANS = {'grass': ((0, FORD), (0, DEPTH)), 'soil': ((0, FORD), (0, DEPTH)), 'stone': ((0, DEPTH),), 'weir': ((FORD, DEPTH),)}


def mul(c, k):
    return tuple(min(255, max(0, round(v*k))) for v in c[:3])


def edge_row(side, x):
    """Local row of the small tile's edge on that side at column x of a 16 px wide piece (pivot row 8 is the centre):
    se runs S..E over [cx, cx+16), sw W..S over [cx-16, cx), nw W..N over [cx-16, cx), ne N..E over [cx, cx+16)."""
    return {'se': 16 - (x + .5)/2, 'sw': 8 + (x + .5)/2, 'nw': 8 - (x + .5)/2, 'ne': (x + .5)/2}[side]


def wall_colour(kind, side, col, r, top, bottom, rnd):
    """Wall row r (0 at its top) of the bottom - top rows, at frame column col."""
    h = bottom - top
    if kind in LIP:
        if r < 2:
            c = LIP[kind][r]
        else:
            c = EARTH[min(3, ((r + (col // 4) % 2) // 3) % 3 + (1 if side == 'nw' else 0))]
            if (col*13 + r*7) % 23 == 0:
                c = mul(c, 1.18)  # a stone
            if r >= h - 2:
                c = mul(c, .72)  # wet foot
    elif kind == 'stone':
        if r == 0:
            c = LIP['soil'][0]
        else:
            q = r - 1
            c = STONE[3] if q % 3 == 2 or (col + (q // 3) * 3) % 6 == 0 else STONE[1 + (q // 3) % 2]
            if r >= h - 2:
                c = mul(c, .75)
    else:  # weir: low stones along the ford's lip, then water slipping down over dark wet stone
        if r < 2:
            c = STONE[3] if (col*7 + r) % 5 == 0 else STONE[(col + r) % 2]
        else:
            c = FALL[(col + r) % 2] if rnd.random() < .7 else WET_STONE
    if side == 'nw':
        c = mul(c, .88)
    return c + (255,)


def wall(kind, side, top, bottom, d2):
    """The wall under the `side` edge in the 128 frame, rows top..bottom below the edge, a foam row at its foot over
    water; rows deeper than d2 cut away over the side neighbour's diamond lowered by d2."""
    im = Image.new('RGBA', (F, F))
    px = im.load()
    rnd = random.Random(f'{kind}{side}{top}{bottom}')
    x0 = C - 16 if side == 'nw' else C
    ncx, ncy = (C - 16, C + 8 + d2) if side == 'nw' else (C + 16, C + 8 + d2)
    for i in range(16):
        col = x0 + i
        e = C - 8 + edge_row(side, i)
        for z in range(top, bottom + (1 if bottom == DEPTH else 0)):
            y = int(e + z + .5)
            if z > d2 and abs(col + .5 - ncx)/16 + abs(y + .5 - ncy)/8 <= 1:
                continue  # under the floor of the side neighbour in front
            if z == bottom:
                if rnd.random() < .7:
                    px[col, y] = FOAM
            else:
                px[col, y] = wall_colour(kind, side, i, z - top, top, bottom, rnd)
    return im, (C, C)


def ford_bed(seed):
    """Clear shallow water over a stony bed: pebbles, a few round stones, now and then flat stepping stones, glints."""
    rnd = random.Random(seed)
    im = Image.new('RGBA', (32, 16))
    px = im.load()
    inside = lambda x, y: abs(x + .5 - 16)/16 + abs(y + .5 - 8)/8 <= .95  # noqa: E731
    pebble = [STONE[0], STONE[1], STONE[2], (140, 128, 104), (118, 132, 128)]
    for _ in range(16):
        x, y = rnd.randrange(32), rnd.randrange(16)
        if inside(x, y):
            px[x, y] = rnd.choice(pebble) + (200,)
            if rnd.random() < .4 and inside(x + 1, y):
                px[x + 1, y] = rnd.choice(pebble) + (200,)
    stones = [(rnd.uniform(4, 28), rnd.uniform(3, 13), rnd.uniform(1.5, 2.5), rnd.uniform(1, 1.4)) for _ in range(3)]
    stones += [(rnd.uniform(10, 22), rnd.uniform(5, 10), 3.5, 1.8) for _ in range(seed % 3)]
    for cx, cy, rx, ry in stones:
        for y in range(16):
            for x in range(32):
                if not inside(x, y):
                    continue
                d = ((x + .5 - cx)/rx)**2 + ((y + .5 - cy)/ry)**2
                if d <= 1:
                    px[x, y] = (STONE[0] if y + .5 < cy else STONE[1]) + (255,)
                elif d <= 1.7 and y + .5 > cy:
                    px[x, y] = STONE[3] + (160,)
    for _ in range(2):
        x, y = rnd.randrange(6, 24), rnd.randrange(4, 12)
        for i in range(rnd.randrange(2, 4)):
            if inside(x + i, y) and not px[x + i, y][3]:
                px[x + i, y] = (176, 214, 220, 150)
    frame = Image.new('RGBA', (F, F))
    frame.alpha_composite(im, (C - 16, C - 8))
    return frame, (C, C)


def bridge_edge(side, front):
    """A bridge small tile's edge over water. Every piece holds a rail (a post at its x = 0 end, a bar RAIL px up);
    a front (se, sw) edge also shows the deck's side beam and a pier in the middle down to the water."""
    rail = 5
    im = Image.new('RGBA', (16, 16 + rail + DEPTH + 2))
    px = im.load()
    for x in range(16):
        e = int(edge_row(side, x)) + rail
        if front:
            px[x, e] = WOOD[1] + (255,)
            px[x, e + 1] = WOOD[2] + (255,)
            px[x, e + 2] = OUTLINE + (255,)
            if 7 <= x <= 8:
                for y in range(e + 3, e + DEPTH):
                    px[x, y] = (WOOD[2] if x == 7 else OUTLINE) + (255,)
            if 5 <= x <= 10:
                px[x, e + DEPTH] = FOAM
        px[x, e - rail] = WOOD[0] + (255,)
        px[x, e - rail + 1] = OUTLINE + (255,)
        if x == 0:
            for y in range(e - rail, e + 1):
                px[x, y] = WOOD[2] + (255,)
    return im, (0 if side in ('se', 'ne') else 16, 8 + rail)


def sprites():
    """name -> (image, pivot, in a 128 frame)"""
    out = {}
    for kind, spans in SPANS.items():
        for top, bottom in spans:
            for side in ('nw', 'ne'):
                for d2 in (0, FORD, DEPTH):
                    out[f'wall_{kind}_{side}_{top}{bottom}_{d2}'] = wall(kind, side, top, bottom, d2) + (True,)
    for i in range(FORD_BEDS):
        out[f'ford_bed_{i}'] = ford_bed(i * 31 + 5) + (True,)
    for side in ('nw', 'ne'):
        out[f'bridge_rail_{side}'] = bridge_edge(side, False) + (False,)
    for side in ('se', 'sw'):
        out[f'bridge_edge_{side}'] = bridge_edge(side, True) + (False,)
    return out


def review(path):
    """Every wall piece in its frame, 4x, the small tile outlined."""
    sp = [(n, im) for n, (im, _, framed) in sprites().items() if n.startswith('wall_')]
    cols = 6
    sheet = Image.new('RGBA', (cols * 64, (len(sp) + cols - 1) // cols * 48), (40, 44, 40, 255))
    for i, (n, im) in enumerate(sp):
        sheet.alpha_composite(im.crop((C - 32, C - 16, C + 32, C + 32)), ((i % cols) * 64, (i // cols) * 48))
    sheet.resize((sheet.width * 4, sheet.height * 4), Image.NEAREST).save(path)
    print(path)


OBSOLETE = ('bank_', 'ford_nw_', 'ford_ne_', 'ford_back_')  # the old banks (makeBanks before docs/plan-tile-floors.md)


def main():
    if '--review' in sys.argv:
        review(ROOT / 'output/walls-review.png')
        return
    path = CAND / 'tiles/tiles.json'
    doc = json.loads(path.read_text())
    for name in [n for n in doc['sprites'] if n.startswith(OBSOLETE)]:
        (CAND / doc['sprites'].pop(name)['file']).unlink(missing_ok=True)
    for name, (im, pivot, framed) in sprites().items():
        im.save(CAND / f'tiles/{name}.png')
        doc['sprites'][name] = {'file': f'tiles/{name}.png', 'size': list(im.size), 'pivot': list(pivot),
                                'layer': 'ground', 'note': 'battle-map ground walls and crossings (gen_banks.py)'}
        if framed:
            doc['sprites'][name]['frame'] = F
    path.write_text(json.dumps(doc, indent=1, ensure_ascii=False) + '\n')


if __name__ == '__main__':
    main()
