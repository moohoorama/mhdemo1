#!/usr/bin/env python3
"""ver4 map structures: village houses, castle wall walkways and castle paving.

Rendered with the unit cel renderer (render.py), but from a fixed camera that matches
the map: azimuth on a diagonal and 30 degree elevation, so one model unit is one map
cell (a 32x16 diamond). A cell spans x, y in [-.5, .5] around the model origin, which
lands on the cell centre (the sprite pivot).

  python3 tools/unit3d/structures.py --candidates   -> output/ver4-structure-candidates.png
  python3 tools/unit3d/structures.py                -> tools/spritetool/assets/ver4-structures/
  python3 -c "import structures; structures.scene()"  -> output/ver4-structure-scene.png (review)
"""
import argparse
import json
import math
from pathlib import Path

import numpy as np
from PIL import Image, ImageDraw, ImageFont

import base_model as m
import heads as Hd
import render as R

ROOT = Path(__file__).resolve().parents[2]
OUT = ROOT / 'tools/spritetool/assets/ver4-structures'
FONT = '/System/Library/Fonts/AppleSDGothicNeo.ttc'
VIEW = 'SW'
SCALE = 16*math.sqrt(2)          # one model unit = one cell edge
CELL, PIVOT = (40, 48), (20, 38)  # sprite cell and the cell centre in it
STONE, STONE_DARK, PLASTER, WOOD = '#aeb6bc', '#7d858c', '#e6dcbc', '#6e4126'
THATCH, THATCH_DARK, TILE, TILE_DARK = '#d9a23a', '#8f7646', '#4a4d5a', '#30323c'
R.add_ramps({'#2a1d18': '0fff', STONE: '3455', STONE_DARK: '2344', PLASTER: 'TSS6', WOOD: 'fggh',
             THATCH: 'klmm', THATCH_DARK: 'UTkk', TILE: '1234', TILE_DARK: '0123'})


def camera(fn):
    """Render at the map's 2:1 projection (30 degree elevation)."""
    def wrapped(*a, **k):
        old, R.ELEVATION = R.ELEVATION, math.radians(30)
        try:
            return fn(*a, **k)
        finally:
            R.ELEVATION = old
    return wrapped


@camera
def draw(meshes, cell=CELL, pivot=PIVOT):
    return Hd.to_img(R.render(meshes, VIEW, SCALE, cell=cell, pivot=pivot))


@camera
def back_corner():
    """Model-space direction of the diamond's top (farthest) corner."""
    corners = [np.array([x, y]) for x in (-.5, .5) for y in (-.5, .5)]
    return min(corners, key=lambda c: R.project([c[0], c[1], 0], VIEW, SCALE, PIVOT)[1])*2


BACK = back_corner()  # (+-1, +-1)


def V(*a):
    return np.array(a, float)


def gable(cx, cy, w, d, wall_h, roof_h, wall, roof, roof_side, overhang=.07):
    """Box house with a gable roof; the ridge runs along x."""
    m.box(V(cx, cy, wall_h/2), V(w, d, wall_h), wall)
    x0, x1 = cx - w/2 - overhang, cx + w/2 + overhang
    y0, y1 = cy - d/2 - overhang, cy + d/2 + overhang
    z0, z1 = wall_h - .02, wall_h + roof_h
    m.poly([V(x0, y0, z0), V(x1, y0, z0), V(x1, cy, z1), V(x0, cy, z1)], roof)
    m.poly([V(x0, y1, z0), V(x0, cy, z1), V(x1, cy, z1), V(x1, y1, z0)], roof)
    for x in (cx - w/2, cx + w/2):
        m.poly([V(x, cy - d/2, wall_h), V(x, cy, z1 - .01), V(x, cy + d/2, wall_h)], roof_side)


def hip(cx, cy, r, wall_h, roof_h, wall, roof):
    """Round hut: short cylinder wall and a cone roof."""
    m.rod(V(cx, cy, 0), V(cx, cy, wall_h), r, wall)
    n = 10
    rim = [V(cx + (r + .06)*math.cos(a), cy + (r + .06)*math.sin(a), wall_h - .02) for a in np.linspace(0, 2*math.pi, n + 1)]
    top = V(cx, cy, wall_h + roof_h)
    for a, b in zip(rim, rim[1:]):
        m.poly([a, b, top], roof)


def door(cx, cy, w, d, wall_h):
    """Door on the wall facing the viewer (the side opposite the back corner)."""
    fx = cx - BACK[0]*(w/2 + .005)
    m.box(V(fx, cy, wall_h*.36), V(.01, .1, wall_h*.66), WOOD)


def timber_house(cx, cy, w, d, wall_h, roof_h, along_x=True):
    """Thatched gable house after the mhero reference: timber corner posts and sill, plaster walls,
    door and windows, a ridge log with crossed ends."""
    if not along_x:
        w, d = d, w
    m.box(V(cx, cy, wall_h/2), V(w, d, wall_h), PLASTER)
    for sx in (-1, 1):
        for sy in (-1, 1):
            m.box(V(cx + sx*w/2, cy + sy*d/2, wall_h/2), V(.05, .05, wall_h + .02), WOOD)
    m.box(V(cx, cy, .025), V(w + .03, d + .03, .05), WOOD)
    ov = .08
    z0, z1 = wall_h - .02, wall_h + roof_h
    if along_x:
        x0, x1, y0, y1 = cx - w/2 - ov, cx + w/2 + ov, cy - d/2 - ov, cy + d/2 + ov
        m.poly([V(x0, y0, z0), V(x1, y0, z0), V(x1, cy, z1), V(x0, cy, z1)], THATCH)
        m.poly([V(x0, y1, z0), V(x0, cy, z1), V(x1, cy, z1), V(x1, y1, z0)], THATCH)
        for x in (cx - w/2, cx + w/2):
            m.poly([V(x, cy - d/2, wall_h), V(x, cy, z1 - .01), V(x, cy + d/2, wall_h)], PLASTER)
        m.rod(V(x0 - .02, cy, z1 + .02), V(x1 + .02, cy, z1 + .02), .035, WOOD)
        for x in (x0 + .03, x1 - .03):
            m.rod(V(x, cy - .07, z1 - .03), V(x, cy + .07, z1 + .09), .02, WOOD)
            m.rod(V(x, cy + .07, z1 - .03), V(x, cy - .07, z1 + .09), .02, WOOD)
    else:
        x0, x1, y0, y1 = cx - w/2 - ov, cx + w/2 + ov, cy - d/2 - ov, cy + d/2 + ov
        m.poly([V(x0, y0, z0), V(cx, y0, z1), V(cx, y1, z1), V(x0, y1, z0)], THATCH)
        m.poly([V(x1, y0, z0), V(x1, y1, z0), V(cx, y1, z1), V(cx, y0, z1)], THATCH)
        for y in (cy - d/2, cy + d/2):
            m.poly([V(cx - w/2, y, wall_h), V(cx, y, z1 - .01), V(cx + w/2, y, wall_h)], PLASTER)
        m.rod(V(cx, y0 - .02, z1 + .02), V(cx, y1 + .02, z1 + .02), .035, WOOD)
        for y in (y0 + .03, y1 - .03):
            m.rod(V(cx - .07, y, z1 - .03), V(cx + .07, y, z1 + .09), .02, WOOD)
            m.rod(V(cx + .07, y, z1 - .03), V(cx - .07, y, z1 + .09), .02, WOOD)
    # door and windows on the two walls that face the viewer (away from the back corner)
    fx, fy = cx - BACK[0]*(w/2 + .006), cy - BACK[1]*(d/2 + .006)
    m.box(V(fx, cy + BACK[1]*d*.12, wall_h*.38), V(.012, .11, wall_h*.7), WOOD)
    m.box(V(fx, cy - BACK[1]*d*.25, wall_h*.6), V(.012, .07, .07), '#2a1d18')
    for k in (-.22, .18):
        m.box(V(cx + k*w, fy, wall_h*.6), V(.08, .012, .07), '#2a1d18')


def house(kind):
    m.meshes = []
    bx, by = BACK*.2
    if kind in ('timber', 'timber_y'):
        timber_house(bx, by, .56, .42, .27, .24, along_x=kind == 'timber')
        return draw(m.meshes)
    if kind == 'thatch':
        gable(bx, by, .5, .42, .28, .22, PLASTER, THATCH, THATCH_DARK)
        door(bx, by, .5, .42, .28)
    elif kind == 'tile':
        gable(bx, by, .5, .42, .3, .2, PLASTER, TILE, TILE_DARK)
        door(bx, by, .5, .42, .3)
    elif kind == 'hut':
        hip(bx, by, .24, .26, .26, PLASTER, THATCH)
    elif kind == 'pair':
        gable(bx + BACK[0]*.05, by - BACK[1]*.12, .34, .3, .24, .16, PLASTER, THATCH, THATCH_DARK)
        gable(bx - BACK[0]*.14, by + BACK[1]*.12, .3, .28, .22, .15, PLASTER, TILE, TILE_DARK)
    return draw(m.meshes)


FLOOR_KEYS = {'light': ('4', '5', '3'), 'dark': ('3', '4', '2')}  # stone, lit edge, joint


def paving(tone, rows=2):
    """Flat paving drawn as pixels: staggered stones on the cell diamond, joints along both axes.
    Returns a CELL-sized key grid; (a, b) are map-space coordinates in [-.5, .5]."""
    stone, lit, joint = FLOOR_KEYS[tone]
    cx, cy = PIVOT[0] + 1, PIVOT[1] + 1  # cell centre in pixels (see SPRITE_PIVOT)
    g = [['.'] * CELL[0] for _ in range(CELL[1])]
    for y in range(CELL[1]):
        for x in range(CELL[0]):
            dx, dy = x + .5 - cx, y + .5 - cy
            a, b = (dx/16 + dy/8)/2, (dy/8 - dx/16)/2
            if abs(a) > .5 or abs(b) > .5:
                continue
            ra, rb = (a + .5)*rows, (b + .5)*rows
            rb += .5*(int(ra) % 2)  # stagger every other row of stones
            fa, fb = ra % 1, rb % 1
            if fa < .09 or fb < .07:
                g[y][x] = joint
            elif fa < .2 or fb < .15:
                g[y][x] = lit
            else:
                g[y][x] = stone
    return g


EDGES = [(1, 0), (0, 1), (-1, 0), (0, -1)]  # bit i: parapet on the +x, +y, -x, -y edge


def parapet(mask, height, color, merlons=2):
    """Crenellated parapet on the masked edges: a low wall plus merlon blocks."""
    t = .12
    for i, (ex, ey) in enumerate(EDGES):
        if not mask & (1 << i):
            continue
        cx, cy = ex*(.5 - t/2), ey*(.5 - t/2)
        long = V(t, 1, 0) if ex else V(1, t, 0)
        m.box(V(cx, cy, height/2), long + V(0, 0, height), color)
        for k in range(merlons):
            off = -.5 + (k + .5)/merlons
            px, py = (cx, off) if ex else (off, cy)
            m.box(V(px, py, height + .08), V(t, .24, .16) if ex else V(.24, t, .16), color)


def wall(mask, height=.22, floor='dark'):
    """Walkway: paving with a parapet on the masked edges (the outside of the wall)."""
    m.meshes = []
    parapet(mask, height, STONE)
    top = draw(m.meshes) if m.meshes else Image.new('RGBA', CELL)
    base = Hd.to_img(paving(floor))
    base.alpha_composite(top)
    return base


RISE = 10   # castle wall cells stand this many pixels above the ground
PARAPET, MERLON = 3, 3  # parapet height and the extra height of each merlon (px)
FACE = {'sw': ('4', '5', '3'), 'se': ('3', '4', '2')}  # stone, lit top course, joint


def edge_columns(side, cx, cy, lift):
    """(x, y) of the cell's diamond edge, one per pixel column, lifted by `lift` px."""
    for x in (range(cx - 16, cx) if side in ('sw', 'nw') else range(cx, cx + 16)):
        t = (x + .5 - (cx - 16))/16 if side in ('sw', 'nw') else (cx + 16 - x - .5)/16
        y = cy + t*8 if side in ('sw', 'se') else cy - t*8
        yield x, int(y) - lift


def castle(open_, tone='dark'):
    """Raised wall cell: paving lifted RISE px; stone faces on open front edges (sw, se) and a
    crenellated parapet on every open edge. open_[side]: the neighbour on that side is not wall.
    Bricks and merlons repeat every 8 px from the cell centre, so neighbouring cells line up."""
    cx, cy = PIVOT[0] + 1, PIVOT[1] + 1
    g = [['.'] * CELL[0] for _ in range(CELL[1])]
    top = paving(tone)
    for y in range(CELL[1]):
        for x in range(CELL[0]):
            if top[y][x] != '.' and y - RISE >= 0:
                g[y - RISE][x] = top[y][x]

    def put(x, y, c):
        if 0 <= x < CELL[0] and 0 <= y < CELL[1]:
            g[y][x] = c

    def parapet(side):
        stone, lit, joint = FACE['sw' if side in ('sw', 'ne') else 'se']
        for x, y in edge_columns(side, cx, cy, RISE):
            merlon = (x - cx) % 8 < 4
            h = PARAPET + (MERLON if merlon else 0)
            for k in range(1, h + 1):
                put(x, y - k, lit if k == h else (joint if k == PARAPET and merlon else stone))
            put(x, y - h - 1, '0')

    for side in ('nw', 'ne'):  # back parapets first: the paving and front walls overlap them
        if open_[side]:
            parapet(side)
    for side in ('sw', 'se'):
        if not open_[side]:
            continue
        stone, lit, joint = FACE[side]
        for x, edge in edge_columns(side, cx, cy, RISE):
            for k in range(RISE + 1):
                brick = ((x - cx) // 8 + k // 4) % 2  # stagger every course
                c = joint if k % 4 == 0 and k else (joint if (x - cx) % 8 == 0 and brick else stone)
                put(x, edge + k, c)
            put(x, edge + RISE + 1, '0')
        parapet(side)
    lined = R.outline(g, (0, 0, CELL[0], CELL[1]))
    for y in range(CELL[1]):
        for x in range(CELL[0]):
            if lined[y][x] == '0' and g[y][x] == '.':
                dx, dy = x + .5 - cx, y + .5 - (cy - RISE)
                side = ('n' if dy < 0 else 's') + ('e' if dx > 0 else 'w')
                if not open_[side] and abs(dy) > 1:
                    lined[y][x] = '.'
    return Hd.to_img(lined)


def floor(tone='light'):
    return Hd.to_img(paving(tone))


CANDIDATES = {
    'house2': [('목조 초가 (용마루 X)', lambda: house('timber')), ('목조 초가 · 90도 돌림', lambda: house('timber_y'))],
    'house': [('초가집', lambda: house('thatch')), ('기와집', lambda: house('tile')),
              ('둥근 초가', lambda: house('hut')), ('초가+기와 두 채', lambda: house('pair'))],
    'wall': [('낮은 성가퀴', lambda: wall(0b1111, .22)), ('높은 성가퀴', lambda: wall(0b1111, .38)),
             ('낮은 성가퀴 · 밝은 바닥', lambda: wall(0b1111, .22, 'light')), ('바깥쪽 두 변만 (예: 성벽 줄)', lambda: wall(0b0101, .3))],
    'floor': [('밝은 돌바닥', lambda: floor('light')), ('어두운 돌바닥', lambda: floor('dark'))],
}
TITLES = {'house2': '마을 집 v2', 'house': '마을 집', 'wall': '성벽', 'floor': '성내 바닥'}


def candidates():
    sc, pad, label = 6, 12, 30
    bw, bh = CELL[0]*sc*2, CELL[1]*sc
    width = pad + 4*(bw + pad)
    height = pad + len(CANDIDATES)*(bh + label + pad)
    sheet = Image.new('RGBA', (width, height), (236, 232, 220, 255))
    d = ImageDraw.Draw(sheet)
    font = ImageFont.truetype(FONT, 20)
    for r, (key, items) in enumerate(CANDIDATES.items()):
        for c, (title, make) in enumerate(items):
            im = make()
            x0, y0 = pad + c*(bw + pad), pad + r*(bh + label + pad)
            d.text((x0, y0), f'{TITLES[key]} {c + 1} · {title}', fill=(40, 36, 30), font=font)
            d.rectangle((x0, y0 + label, x0 + bw - 1, y0 + label + bh - 1), fill=(148, 179, 110, 255))
            # the cell's diamond and its neighbours for scale: 2 x 1 cells side by side
            for dx in (0, 1):
                ox = x0 + (dx*32 + 4)*sc
                cxp, cyp = ox + PIVOT[0]*sc, y0 + label + PIVOT[1]*sc
                d.polygon([(cxp, cyp - 8*sc), (cxp + 16*sc, cyp), (cxp, cyp + 8*sc), (cxp - 16*sc, cyp)],
                          outline=(90, 110, 70, 255))
                sheet.alpha_composite(im.resize((CELL[0]*sc, CELL[1]*sc), Image.NEAREST), (ox, y0 + label))
    out = ROOT / 'output/ver4-structure-candidates.png'
    sheet.convert('RGB').save(out)
    print(out)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--candidates', action='store_true')
    args = ap.parse_args()
    if args.candidates:
        candidates()
        return
    build()


SIDES = ['nw', 'ne', 'se', 'sw']  # wall mask bit i: the neighbour on SIDES[i] is not wall


def build():
    """One sheet: village, interior floor, and the 16 wall variants by open-edge mask."""
    OUT.mkdir(parents=True, exist_ok=True)
    sprites = [('village', village(4)), ('floor', floor('light'))]
    sprites += [(f'wall_{mask:02d}', castle({s: bool(mask >> i & 1) for i, s in enumerate(SIDES)}))
                for mask in range(16)]
    sheet = Image.new('RGBA', (CELL[0]*len(sprites), CELL[1]))
    for i, (_, im) in enumerate(sprites):
        sheet.alpha_composite(im, (i*CELL[0], 0))
    sheet.save(OUT / 'structures.png')
    meta = dict(version=1, image='structures.png', cell=list(CELL), pivot=[PIVOT[0] + 1, PIVOT[1] + 1],
                pivot_note='cell centre of the map diamond (32x16)', rise=RISE,
                wall_mask_sides=SIDES, layer=dict(village='object', floor='ground', wall='object'),
                sprites={name: i for i, (name, _) in enumerate(sprites)})
    (OUT / 'structures.json').write_text(json.dumps(meta, indent=1) + '\n')
    print(OUT / 'structures.png')



def scene():
    """Small map: village block, castle block and a wall line, with units for scale."""
    import looks  # noqa: F401  (registers kits)
    rows = ['........ccccc',
            '..vv....ciiic',
            '..vv....ciiic',
            '........ciiic',
            '........ccccc',
            '.............']
    W, H = len(rows[0]), len(rows)
    raised = lambda u, v: 0 <= v < H and 0 <= u < W and rows[v][u] == 'c'  # noqa: E731
    ox, oy = 16*H + 24, 24
    img = Image.new('RGBA', (16*(W + H) + 48, 8*(W + H) + 64), (148, 179, 110, 255))
    d = ImageDraw.Draw(img)
    items = []
    for v in range(H):
        for u in range(W):
            x, y = ox + (u - v)*16, oy + (u + v + 1)*8
            d.polygon([(x, y - 8), (x + 16, y), (x, y + 8), (x - 16, y)], outline=(128, 160, 98))
            t = rows[v][u]
            if t == 'c':
                items.append((y, castle(dict(sw=not raised(u, v + 1), se=not raised(u + 1, v),
                                             nw=not raised(u - 1, v), ne=not raised(u, v - 1))), (x, y)))
            elif t == 'i':
                items.append((y - 100, floor('light'), (x, y)))
            elif t == 'v':
                items.append((y - .1, village(4), (x, y)))
    unit = Image.open(ROOT / 'ver3/asset/units/infantry.png')
    frame = unit.crop((0, 3*46, 56, 4*46))  # SE idle
    for u, v in ((3, 3), (10, 2), (6, 4), (12, 3), (2, 2)):
        x, y = ox + (u - v)*16, oy + (u + v + 1)*8
        lift = RISE if raised(u, v) else 0
        items.append((y + .5, frame, (x - 28 + 21 - 20, y - lift - 38 + 39 - 39)))
    for y, im, (x, yy) in sorted(items, key=lambda i: i[0]):
        if im is frame:
            img.alpha_composite(im, (x, yy))
        else:
            img.alpha_composite(im, (x - PIVOT[0] - 1, yy - PIVOT[1] - 1))
    img = img.resize((img.width*4, img.height*4), Image.NEAREST)
    out = ROOT / 'output/ver4-structure-scene.png'
    img.convert('RGB').save(out)
    print(out)


def pixel_house(mirror=False):
    """Hand-placed pixel house after the mhero reference, drawn flat at 1x (no 3D):
    long plastered wall with two windows on the left, gable end with the door on the right,
    golden thatch with darker strokes, a ridge log with crossed ends, timber posts and sill.
    The footprint sits in the back half of the cell; returns a CELL image with the cell pivot."""
    C = {k: R.COLOR[k] + (255,) for k in R.COLOR}
    im = Image.new('RGBA', CELL)
    d = ImageDraw.Draw(im)
    px, py = PIVOT[0] + 1, PIVOT[1] + 1  # cell centre
    ox, oy = px - 13, py - 30            # house origin: bottom corner at (13, 27) lands just behind the centre
    P = lambda pts: [(ox + x, oy + y) for x, y in pts]  # noqa: E731
    # walls: left long wall (lit), right gable end (shade); 10px tall
    d.polygon(P([(1, 20), (13, 26), (13, 16), (1, 10)]), fill=C['S'])
    d.polygon(P([(13, 26), (23, 21), (23, 11), (13, 16)]), fill=C['T'])
    d.polygon(P([(13, 16), (23, 11), (18, 4)]), fill=C['T'])  # gable triangle
    # timber: posts, sill, gable frame
    for x0, y0, x1, y1 in [(1, 10, 1, 20), (13, 16, 13, 26), (23, 11, 23, 21)]:
        d.line(P([(x0, y0), (x1, y1)]), fill=C['g'])
    d.line(P([(1, 20), (13, 26), (23, 21)]), fill=C['f'])
    # framed windows on the long wall, door on the gable end
    for wx in (3, 8):
        wy = 14 + wx//2
        d.polygon(P([(wx, wy), (wx + 2, wy + 1), (wx + 2, wy + 4), (wx, wy + 3)]), fill=C['g'])
        d.line(P([(wx + 1, wy + 1), (wx + 1, wy + 3)]), fill=C['0'])
    d.polygon(P([(16, 24), (19, 22), (19, 16), (16, 18)]), fill=C['g'])
    d.line(P([(16, 18), (16, 24)]), fill=C['f'])
    # thatch: front plane from the eave to the ridge, overhanging the walls
    d.polygon(P([(-1, 10), (13, 17), (19, 4), (6, -3)]), fill=C['l'])
    d.polygon(P([(19, 4), (25, 11), (24, 12), (18, 6)]), fill=C['k'])  # back plane edge at the gable
    for i in range(2, 13, 3):  # straw strokes down the slope
        d.line(P([(6 + i, -3 + i//2 + 1), (i, 10 + i//2)]), fill=C["k"])
    d.line(P([(-1, 10), (13, 17)]), fill=C['k'])
    d.line(P([(1, 8), (14, 15)]), fill=C['m'])
    # ridge log with crossed ends
    d.line(P([(5, -4), (20, 4)]), fill=C['g'])
    d.line(P([(5, -3), (20, 5)]), fill=C['h'])
    for x, y in ((5, -3), (20, 4)):
        d.line(P([(x - 2, y - 2), (x + 2, y + 2)]), fill=C['f'])
        d.line(P([(x - 2, y + 2), (x + 2, y - 2)]), fill=C['f'])
    if mirror:
        im = im.transpose(Image.FLIP_LEFT_RIGHT)
        im = im.crop((CELL[0] - 2*px, 0, CELL[0] - 2*px + CELL[0], CELL[1])) if CELL[0] != 2*px else im
    g = [['.' if im.getpixel((x, y))[3] == 0 else 'x' for x in range(CELL[0])] for y in range(CELL[1])]
    edge = R.outline(g, (0, 0, CELL[0], CELL[1]))
    for y in range(CELL[1]):
        for x in range(CELL[0]):
            if edge[y][x] == '0' and g[y][x] == '.':
                im.putpixel((x, y), C['0'])
    return im


MINI_HOUSE = [  # hand-placed 13x12; long wall faces SW (left), gable end with door faces SE (right)
    '....000......',
    '...0gg00.....',
    '..0lllgg00...',
    '.0mllllllg0..',
    '0mmllllllk00.',
    '0kmmmlllkTT0.',
    '0SkkmmlkTTTT0',
    '0SSSkkkTTffT0',
    '0S0SSSSTTffT0',
    '.0SSSSSTTffT0',
    '..00SSSTTTT0.',
    '....000000...',
]


def mini_house():
    """Deformed little village house (about a quarter of a character), always the same way round."""
    im = Hd.to_img(MINI_HOUSE)
    return im, (7, 11)  # pivot: bottom of the footprint


def village(count=4):
    """A village cell: small houses in a diamond cluster, drawn back to front."""
    house, (hx, hy) = mini_house()
    cx, cy = PIVOT[0] + 1, PIVOT[1] + 1
    spots = {1: [(0, 2)], 2: [(-5, 0), (5, 3)], 3: [(-6, -1), (6, -1), (0, 4)], 4: [(0, -4), (-9, 0), (9, 0), (0, 4)]}[count]
    im = Image.new('RGBA', CELL)
    for dx, dy in sorted(spots, key=lambda p: p[1]):
        im.alpha_composite(house, (cx + dx - hx, cy + dy - hy))
    return im


if __name__ == '__main__':
    main()
