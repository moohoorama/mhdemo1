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
    """One sheet: village, interior floor, the 16 wall variants by open-edge mask, and the deformed
    mountain cell battle maps draw instead of the big rocks (user pick, 2026-10-05). The battle-map
    forest is drawn at the characters' pixel scale, so it lives in props.py (forest_0-3)."""
    OUT.mkdir(parents=True, exist_ok=True)
    sprites = [('village', village(4)), ('floor', floor('light'))]
    sprites += [(f'wall_{mask:02d}', castle({s: bool(mask >> i & 1) for i, s in enumerate(SIDES)}))
                for mask in range(16)]
    sprites += [('mountain', mountain('stone'))]
    sheet = Image.new('RGBA', (CELL[0]*len(sprites), CELL[1]))
    for i, (_, im) in enumerate(sprites):
        sheet.alpha_composite(im, (i*CELL[0], 0))
    sheet.save(OUT / 'structures.png')
    meta = dict(version=1, image='structures.png', cell=list(CELL), pivot=[PIVOT[0] + 1, PIVOT[1] + 1],
                pivot_note='cell centre of the map diamond (32x16)', rise=RISE,
                wall_mask_sides=SIDES, layer=dict(village='object', floor='ground', wall='object', mountain='object'),
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


def mini_tree(kind='round', size=1.0):
    """Deformed little tree for battle-map forest cells (about a third of a character), drawn as
    pixels like MINI_HOUSE: 'round' broadleaf or 'pine'. Light from the upper left. Returns
    (image, pivot at the foot of the trunk)."""
    w, h = 13, 15
    g = [['.'] * w for _ in range(h)]
    cx = 6
    if kind == 'round':
        r = 4.2*size
        cy = 5.0
        for y in range(h):
            for x in range(w):
                dx, dy = x + .5 - (cx + .5), y + .5 - cy
                if dx*dx + (dy*1.1)**2 <= r*r:
                    d = dx + dy*1.2
                    g[y][x] = 'J' if d < -3.2 else 'I' if d < .2 else 'H' if d < 3.2 else 'G'
        for y in range(int(cy + r*.8), int(cy + r*.8) + 3):
            g[y][cx] = 'g'
            g[y][cx - 1] = 'h' if g[y][cx - 1] == '.' else g[y][cx - 1]
        foot = int(cy + r*.8) + 3
    else:
        top, base = 1, 11
        for y in range(top, base):
            t = (y - top)/(base - top)
            half = 1 + int(4.6*size*t) - (1 if (y - top) % 3 == 0 and y > top + 2 else 0)  # tiered outline
            for x in range(cx - half, cx + half + 1):
                d = (x - cx) + (y - top)*.2 - 1
                g[y][x] = 'J' if d < -2 else 'I' if d < 0 else 'H' if d < 2 else 'G'
        for y in range(base, base + 2):
            g[y][cx] = 'g'
        foot = base + 2
    edge = R.outline(g, (0, 0, w, h))
    return Hd.to_img(edge), (cx, foot)


FOREST_SPOTS = {  # tree offsets from the cell centre (pixels), kind per spot
    'round': [((-6, -3), 'round'), ((6, -3), 'round'), ((0, -1), 'round'), ((-7, 3), 'round'), ((7, 3), 'round')],
    'pine': [((-6, -3), 'pine'), ((6, -3), 'pine'), ((0, -1), 'pine'), ((-7, 3), 'pine'), ((7, 3), 'pine')],
    'mixed': [((-6, -3), 'pine'), ((6, -3), 'round'), ((0, -1), 'pine'), ((-7, 3), 'round'), ((7, 3), 'pine')],
    'dense': [((-8, -3), 'round'), ((0, -4), 'pine'), ((8, -3), 'round'), ((-4, 0), 'round'), ((4, 0), 'pine'),
              ((-8, 4), 'round'), ((0, 4), 'round'), ((8, 4), 'pine')],
}


def forest(kind='round'):
    """A battle-map forest cell: little trees in a diamond cluster, drawn back to front, like village()."""
    im = Image.new('RGBA', CELL)
    cx, cy = PIVOT[0] + 1, PIVOT[1] + 1
    size = .8 if kind == 'dense' else 1.0
    for (dx, dy), tree in sorted(FOREST_SPOTS[kind], key=lambda s: s[0][1]):
        t, (tx, ty) = mini_tree(tree, size)
        im.alpha_composite(t, (cx + dx - tx, cy + dy - ty))
    return im


MOUNTAIN_KEYS = {  # lit face, base, shadow face, dark edge
    'stone': ('5', '4', '3', '2'), 'earth': ('j', 'i', 'h', 'g'), 'green': ('J', 'I', 'H', 'G')}


def mini_peak(w, h, kind='stone', snow=False):
    """Deformed little mountain: a triangle with a lit left face and a shaded right face split at the
    ridge, a darker band at the foot. Returns (image, pivot at the middle of the base)."""
    lit, base, shade, dark = MOUNTAIN_KEYS[kind]
    W, H = w + 2, h + 2
    g = [['.'] * W for _ in range(H)]
    cx = W//2
    for y in range(1, h + 1):
        t = y/h
        half = max(0, int(round(t*w/2)))
        ridge = cx + int(round((t - .5)*1.5))  # the ridge leans a little to the right going down
        for x in range(cx - half, cx + half + 1):
            if not 0 <= x < W:
                continue
            c = lit if x < ridge else shade
            if y > h - 2:
                c = base if x < ridge else dark
            if snow and y <= max(2, h//4):
                c = '6' if x < ridge else '5'
            g[y][x] = c
    edge = R.outline(g, (0, 0, W, H))
    return Hd.to_img(edge), (cx, h + 1)


def boulder(r=3, kind='stone'):
    lit, base, shade, dark = MOUNTAIN_KEYS[kind]
    W, H = 2*r + 3, 2*r + 2
    g = [['.'] * W for _ in range(H)]
    cx, cy = W/2, H/2
    for y in range(H):
        for x in range(W):
            dx, dy = x + .5 - cx, (y + .5 - cy)*1.3
            if dx*dx + dy*dy <= r*r:
                d = dx + dy
                g[y][x] = lit if d < -1.5 else base if d < 1 else shade
    edge = R.outline(g, (0, 0, W, H))
    return Hd.to_img(edge), (W//2, H - 1)


MOUNTAIN_SPOTS = {  # (offset from the cell centre, maker)
    'stone': [((0, -2), lambda: mini_peak(16, 15, 'stone', True)), ((-8, 3), lambda: mini_peak(12, 10, 'stone')),
              ((8, 3), lambda: mini_peak(11, 9, 'stone'))],
    'earth': [((-3, -2), lambda: mini_peak(16, 13, 'earth')), ((6, 0), lambda: mini_peak(13, 11, 'earth')),
              ((-5, 4), lambda: mini_peak(10, 7, 'earth'))],
    'green': [((0, -2), lambda: mini_peak(18, 12, 'green')), ((-8, 3), lambda: mini_peak(11, 7, 'green')),
              ((7, 3), lambda: boulder(3, 'stone'))],
    'rocks': [((-6, -3), lambda: boulder(4)), ((5, -3), lambda: boulder(3)), ((0, 0), lambda: boulder(4)),
              ((-8, 3), lambda: boulder(3)), ((7, 3), lambda: boulder(4))],
}


def mountain(kind='stone'):
    """A battle-map mountain cell: little peaks (or boulders) in a cluster, drawn back to front."""
    im = Image.new('RGBA', CELL)
    cx, cy = PIVOT[0] + 1, PIVOT[1] + 1
    for (dx, dy), make in sorted(MOUNTAIN_SPOTS[kind], key=lambda s: s[0][1]):
        t, (tx, ty) = make()
        im.alpha_composite(t, (cx + dx - tx, cy + dy - ty))
    return im


# ---------------------------------------------------------------- forest, second round (2026-10-05)
# The first forest read cold and scattered. These use the olive ramp N-R (matching the map's grass and
# trees), leaf clumps shaded one by one, and a dark olive outline instead of black.

def canopy(clumps, w, h, ramp='NOPQR', trunks=(), outline='N'):
    """Pixel canopy: the union of leaf clumps (cx, cy, r); each clump is lit from the upper left on
    its own, so the mass reads as bunched leaves. trunks: (x, y0, y1) columns drawn under the leaves."""
    dark, shade, base, lit, hi = ramp
    g = [['.'] * w for _ in range(h)]
    for x, y0, y1 in trunks:
        for y in range(y0, y1):
            g[y][x], g[y][x + 1] = 'g', 'f'
    owner = {}
    for i, (cx, cy, r) in enumerate(clumps):  # later clumps sit in front
        for y in range(h):
            for x in range(w):
                dx, dy = x + .5 - cx, (y + .5 - cy)*1.15
                if dx*dx + dy*dy <= r*r:
                    owner[x, y] = i
    for (x, y), i in owner.items():
        cx, cy, r = clumps[i]
        dx, dy = (x + .5 - cx)/r, (y + .5 - cy)/r
        d = dx*.8 + dy  # toward the lower right is darker
        k = hi if d < -.95 else lit if d < -.35 else base if d < .3 else shade
        if y > max(c[1] for c in clumps) + 1:  # underside of the whole mass
            k = shade if k in (hi, lit) else dark if k == shade else k
        g[y][x] = k
    lined = R.outline(g, (0, 0, w, h))
    return [[outline if c == '0' else c for c in row] for row in lined]


def olive_tree(r=4, kind='broad'):
    """One tree of the second round: 'broad' (three clumps), 'tall' (stacked clumps), 'pine' (tiers)."""
    if kind == 'broad':
        w, h = 2*r + 6, 2*r + 7
        c = w/2
        clumps = [(c - r*.55, r + 2.2, r*.75), (c + r*.6, r + 2.4, r*.72), (c, r*.9 + 1, r*.8), (c, r + 3, r*.7)]
        g = canopy(clumps, w, h, trunks=[(int(c) - 1, int(r*1.8) + 2, h - 1)])
    elif kind == 'tall':
        w, h = 2*r + 4, 3*r + 6
        c = w/2
        clumps = [(c, r*.9, r*.65), (c - r*.3, r*1.7, r*.75), (c + r*.3, r*2.3, r*.75), (c, r*2.6, r*.7)]
        g = canopy(clumps, w, h, trunks=[(int(c) - 1, int(r*3), h - 1)])
    else:
        w, h = 2*r + 4, 3*r + 5
        c = w/2
        clumps = [(c, r*.7, r*.45), (c, r*1.5, r*.65), (c, r*2.3, r*.85)]
        g = canopy(clumps, w, h, ramp='NNOPQ', trunks=[(int(c) - 1, int(r*2.9), h - 1)])
    foot = max(y for y, row in enumerate(g) for ch in row if ch != '.') + 1
    return Hd.to_img(g), (int(w/2), foot)


def forest_mass(bumps=7):
    """One merged canopy over the cell: a bumpy-topped block of leaves with trunks at its foot."""
    w, h = CELL
    cx, cy = PIVOT[0] + 1, PIVOT[1] + 1
    clumps = []
    for i in range(bumps):  # back row of bumps along the diamond's upper edges, front row lower
        t = i/(bumps - 1)
        clumps.append((cx - 13 + 26*t, cy - 9 + abs(t - .5)*6, 5.2))
    for i in range(4):
        t = i/3
        clumps.append((cx - 10 + 20*t, cy - 3 + abs(t - .5)*4, 5.6))
    trunks = [(int(cx - 9), int(cy) - 2, int(cy) + 3), (int(cx + 2), int(cy), int(cy) + 5), (int(cx + 10), int(cy) - 3, int(cy) + 2)]
    return Hd.to_img(canopy(clumps, w, h, trunks=trunks))


def forest2(kind):
    """Second-round forest cells."""
    if kind == 'mass':
        return forest_mass()
    im = Image.new('RGBA', CELL)
    cx, cy = PIVOT[0] + 1, PIVOT[1] + 1
    spots = {
        'broad3': [((-7, -2), 'broad', 5), ((7, -2), 'broad', 4), ((0, 3), 'broad', 5)],
        'mixed5': [((-8, -3), 'pine', 3), ((5, -4), 'broad', 4), ((-2, 0), 'tall', 3), ((9, 2), 'pine', 3),
                   ((-6, 4), 'broad', 4)],
        'grove': [((-9, -2), 'tall', 3), ((-2, -4), 'broad', 4), ((7, -3), 'tall', 3), ((-5, 3), 'broad', 4),
                  ((4, 2), 'broad', 4), ((10, 4), 'tall', 2)],
    }[kind]
    for (dx, dy), tree, r in sorted(spots, key=lambda s: s[0][1]):
        t, (tx, ty) = olive_tree(r, tree)
        im.alpha_composite(t, (cx + dx - tx, cy + dy - ty))
    return im


FOREST2 = {'broad3': '올리브 활엽수 3그루 (큰 수관)', 'mixed5': '올리브 활엽·키 큰 나무·침엽 5그루',
           'grove': '숲 덩어리 (6그루 촘촘히)', 'mass': '한 덩어리 수관 (이어지는 숲)'}


def forest2_candidates():
    """Like forest_candidates: a 3 x 3 forest block, an L of forest, a village and an infantry unit."""
    kinds = list(FOREST2)
    sc, bw, bh = 4, 170, 110
    sheet = Image.new('RGBA', (2*bw*sc, 2*(bh*sc + 40)), (236, 232, 220, 255))
    d = ImageDraw.Draw(sheet)
    font = ImageFont.truetype(FONT, 22)
    unit = Image.open(ROOT / 'ver4/assets/graphics/units/infantry.png').crop((0, 3*46, 56, 4*46))
    for i, kind in enumerate(kinds):
        box = Image.new('RGBA', (bw, bh), (104, 150, 72, 255))
        bd = ImageDraw.Draw(box)
        ox, oy = 85, 22
        items = []
        for v in range(4):
            for u in range(4):
                x, y = ox + (u - v)*16, oy + (u + v + 1)*8
                bd.polygon([(x, y - 8), (x + 16, y), (x, y + 8), (x - 16, y)], outline=(92, 136, 62))
                if u < 3 and v < 3 or (u, v) == (3, 0):
                    items.append((y, forest2(kind), x, y))
                elif (u, v) == (3, 2):
                    items.append((y, village(4), x, y))
        for y, im, x, yy in sorted(items, key=lambda t: t[0]):
            box.alpha_composite(im, (x - PIVOT[0] - 1, yy - PIVOT[1] - 1))
        big = box.resize((bw*2, bh*2), Image.NEAREST)
        x, y = ox + (3 - 3)*16, oy + (3 + 3 + 1)*8
        big.alpha_composite(unit, (x*2 - 28, y*2 - 38))
        X, Y = (i % 2)*bw*sc, (i//2)*(bh*sc + 40)
        sheet.alpha_composite(big.resize((bw*sc, bh*sc), Image.NEAREST), (X, Y + 40))
        d.text((X + 10, Y + 8), f'숲 {i + 1} · {FOREST2[kind]}', fill=(40, 36, 30), font=font)
    out = ROOT / 'output/ver4-forest-candidates-2.png'
    sheet.convert('RGB').save(out)
    print(out)


# ---------------------------------------------------------------- forest, third round (2026-10-05)
# Rendered with the cel renderer like the units and props (leaf clumps as ellipsoids, cone tiers for
# pines), small trees jittered in size and place, and a dark forest floor that ties a cell together.
OLIVE, OLIVE_LIGHT, OLIVE_DARK, PINE, FLOOR = '#688630', '#98b242', '#405826', '#3e5a2c', '#33461f'
R.add_ramps({OLIVE: 'NOPQ', OLIVE_LIGHT: 'OPQR', OLIVE_DARK: 'NOOP', PINE: 'NNOP', FLOOR: 'NNOO'})


def leafy(x, y, r, color=OLIVE, seed=0):
    """A broadleaf tree at (x, y): trunk and a crown of four to five leaf clumps."""
    rng = np.random.default_rng(seed)
    h = r*1.5
    m.rod(V(x, y, 0), V(x, y, h), r*.13, WOOD)
    for k in range(5):
        a = k*2*math.pi/5 + rng.uniform(-.4, .4)
        d = r*(.42 if k else 0)
        m.ell(V(x + d*math.cos(a), y + d*math.sin(a), h + r*(.35 + rng.uniform(-.1, .2))),
              [r*.62]*2 + [r*.55], color if k % 2 else OLIVE_LIGHT if color == OLIVE else color)
    m.ell(V(x, y, h + r*.75), [r*.5, r*.5, r*.42], OLIVE_LIGHT if color == OLIVE else color)


def conifer(x, y, r, seed=0):
    """A pine at (x, y): a short trunk and three stacked cone tiers."""
    m.rod(V(x, y, 0), V(x, y, r*.8), r*.12, WOOD)
    n = 10
    for i, (z, rr, hh) in enumerate([(r*.55, r*.75, r*1.1), (r*1.15, r*.58, r*.95), (r*1.7, r*.4, r*.85)]):
        ring = [V(x + rr*math.cos(t), y + rr*math.sin(t), z) for t in np.linspace(0, 2*math.pi, n + 1)]
        top = V(x, y, z + hh)
        for p0, p1 in zip(ring, ring[1:]):
            m.poly([p1, p0, top], PINE)


PINE_LIGHT = '#4f7030'
R.add_ramps({PINE_LIGHT: 'NOPQ'})


def spruce(x, y, r, tiers=4, slim=1.0, color=PINE, seed=0):
    """Pine of the fifth round: tiers of cones that narrow toward the top, each a little offset."""
    rng = np.random.default_rng(seed)
    m.rod(V(x, y, 0), V(x, y, r*.7), r*.11, WOOD)
    n = 12
    z = r*.45
    for i in range(tiers):
        t = i/max(1, tiers - 1)
        rr = r*(.78 - .45*t)*slim
        hh = r*(1.05 - .25*t)
        ox, oy = rng.uniform(-.02, .02), rng.uniform(-.02, .02)
        ring = [V(x + ox + rr*math.cos(a), y + oy + rr*math.sin(a), z) for a in np.linspace(0, 2*math.pi, n + 1)]
        top = V(x + ox, y + oy, z + hh)
        for p0, p1 in zip(ring, ring[1:]):
            m.poly([p1, p0, top], color)
        z += hh*.5


def bush(x, y, r):
    m.ell(V(x, y, r*.45), [r, r*.9, r*.6], OLIVE_DARK)


def forest3(kind, seed=1, floor=True, fine=False):
    """Third-round forest cells (map scale: one model unit is one cell). fine: rendered at the
    characters' pixel scale (twice the map's), returned with its pivot for a 1x blit."""
    m.meshes = []
    rng = np.random.default_rng(seed)
    if floor:
        floor_r = .44
        pts = [V(floor_r*math.cos(t)*1.0, floor_r*math.sin(t), .002) for t in np.linspace(0, 2*math.pi, 17)]
        m.poly(pts[:-1], FLOOR)  # dark forest floor under the trees
    j = lambda: rng.uniform(-.04, .04)  # noqa: E731
    if kind == 'broad':
        for (x, y), r in zip([(-.22, .2), (.2, .22), (0, -.02), (-.25, -.22), (.22, -.24)], [.15, .14, .17, .14, .15]):
            leafy(x + j(), y + j(), r*rng.uniform(.9, 1.1), seed=int(rng.integers(99)))
    elif kind == 'mixed':
        for (x, y), r, t in zip([(-.22, .22), (.2, .2), (0, 0), (-.24, -.2), (.22, -.24), (.02, -.3)],
                                [.13, .15, .16, .14, .13, .1], 'plbpbl'):
            x, y = x + j(), y + j()
            if t == 'p':
                conifer(x, y, r*1.2)
            elif t == 'b':
                bush(x, y, r*.9)
            else:
                leafy(x, y, r, seed=int(rng.integers(99)))
    elif kind == 'dense':
        for (x, y) in [(-.28, .28), (0, .3), (.28, .26), (-.3, 0), (-.05, .05), (.25, .02), (-.22, -.27), (.08, -.24), (.32, -.28)]:
            leafy(x + j(), y + j(), rng.uniform(.11, .14), seed=int(rng.integers(99)))
    elif kind == 'pines':
        for (x, y) in [(-.22, .22), (.18, .24), (-.02, .02), (-.26, -.22), (.22, -.2), (.0, -.32)]:
            conifer(x + j(), y + j(), rng.uniform(.13, .17))
    elif kind == 'big':  # fewer, larger crowns
        for (x, y), r in zip([(-.2, .18), (.2, .16), (-.02, -.18)], [.2, .18, .21]):
            leafy(x + j(), y + j(), r*rng.uniform(.92, 1.08), seed=int(rng.integers(99)))
    elif kind == 'bigmix':
        for (x, y), r, t in zip([(-.22, .2), (.2, .18), (0, -.04), (-.2, -.26), (.24, -.22)], [.15, .16, .18, .14, .13], 'lplpl'):
            (conifer if t == 'p' else leafy)(x + j(), y + j(), r*(1.15 if t == 'p' else 1), seed=int(rng.integers(99)))
    elif kind in ('pine_bold', 'pine_bold8'):  # the round-4 pine (three wide tiers), all pines
        spots = [(-.24, .22), (.2, .24), (-.02, .02), (-.26, -.22), (.22, -.2), (.02, -.3)] if kind == 'pine_bold' else \
            [(-.28, .28), (0, .3), (.28, .24), (-.28, -.02), (.02, .02), (.28, -.04), (-.18, -.3), (.14, -.28)]
        for x, y in spots:
            conifer(x + j(), y + j(), rng.uniform(.17, .21) if kind == 'pine_bold' else rng.uniform(.14, .17))
    elif kind == 'pine_fill':  # the chosen pine spread over the whole cell so neighbouring cells join
        g = [-.34, 0, .34]
        for x in g:
            for y in g:
                conifer(x + rng.uniform(-.06, .06), y + rng.uniform(-.06, .06), rng.uniform(.14, .17))
    elif kind in ('pine6', 'pine_tall', 'pine_light', 'pine_broad'):
        spots = {'pine6': [(-.24, .22, .15), (.2, .24, .13), (-.02, .02, .17), (-.26, -.22, .13), (.22, -.2, .15), (.02, -.3, .11)],
                 'pine_tall': [(-.22, .2, .16), (.2, .2, .13), (0, 0, .19), (-.24, -.24, .12), (.24, -.22, .15)],
                 'pine_light': [(-.24, .22, .15), (.2, .24, .13), (-.02, .02, .17), (-.26, -.22, .13), (.22, -.2, .15), (.02, -.3, .11)],
                 'pine_broad': [(-.22, .2, .15), (.22, .22, .15), (0, -.02, .17), (-.24, -.24, .15), (.22, -.22, .13)]}[kind]
        for i, (x, y, r) in enumerate(spots):
            x, y, r = x + j(), y + j(), r*rng.uniform(.9, 1.1)
            if kind == 'pine_broad' and i in (1, 3):
                leafy(x, y, r*.95, seed=int(rng.integers(99)))
            elif kind == 'pine_tall':
                spruce(x, y, r, tiers=5, slim=.8, seed=int(rng.integers(99)))
            else:
                spruce(x, y, r, color=PINE_LIGHT if kind == 'pine_light' else PINE, seed=int(rng.integers(99)))
    if fine:
        m.meshes = [(np.asarray(v)*2, c) for v, c in m.meshes]
        im = draw(m.meshes, cell=(80, 96), pivot=(39, 71))
        im.pivot = (40, 72)
        return im
    return draw(m.meshes)


FOREST3 = {'broad': '활엽수 5그루 + 숲 바닥', 'mixed': '활엽·침엽·덤불 섞음', 'dense': '작은 활엽수 9그루 빽빽이',
           'pines': '침엽수 6그루'}


FOREST4 = [('dense', False, '3번 그대로 · 숲 바닥 없음 (맵 픽셀)'), ('dense', True, '3번 · 캐릭터 픽셀(1배, 2배 촘촘)'),
           ('big', True, '큰 활엽수 3그루 · 캐릭터 픽셀'), ('bigmix', True, '활엽·침엽 5그루 · 캐릭터 픽셀')]


FOREST6 = [('pine_bold8', True, '5차 2번 (가운데에 모임)'), ('pine_fill', True, '칸 전체에 고르게 9그루')]
FOREST5 = [('pine_bold', True, '굵은 침엽수 6그루 (4차 4번의 나무)'), ('pine_bold8', True, '굵은 침엽수 8그루 촘촘히'),
           ('pine6', True, '침엽수 6그루 · 4층'), ('pine_broad', True, '침엽수 3 + 활엽수 2')]


def forest4_candidates(rounds=FOREST4, out_name='ver4-forest-candidates-4.png'):
    """Fourth round: no forest floor; map-scale vs character-scale pixels, at game scale."""
    sc, bw, bh = 2, 340, 220  # canvas pixels (the game's scale), then x2 for viewing
    sheet = Image.new('RGBA', (2*bw*sc, 2*(bh*sc + 40)), (236, 232, 220, 255))
    d = ImageDraw.Draw(sheet)
    font = ImageFont.truetype(FONT, 22)
    unit = Image.open(ROOT / 'ver4/assets/graphics/units/infantry.png').crop((0, 3*46, 56, 4*46))
    vil = village(4).resize((CELL[0]*2, CELL[1]*2), Image.NEAREST)
    for i, (kind, fine, title) in enumerate(rounds):
        box = Image.new('RGBA', (bw, bh), (104, 150, 72, 255))
        bd = ImageDraw.Draw(box)
        ox, oy = 170, 44
        items = []
        for v in range(4):
            for u in range(4):
                x, y = ox + (u - v)*32, oy + (u + v + 1)*16
                bd.polygon([(x, y - 16), (x + 32, y), (x, y + 16), (x - 32, y)], outline=(92, 136, 62))
                if u < 3 and v < 3 or (u, v) == (3, 0):
                    im = forest3(kind, seed=u*7 + v*3 + 1, floor=False, fine=fine)
                    if not fine:
                        im = im.resize((CELL[0]*2, CELL[1]*2), Image.NEAREST)
                        im.pivot = ((PIVOT[0] + 1)*2, (PIVOT[1] + 1)*2)
                    items.append((y, im, x - im.pivot[0], y - im.pivot[1]))
                elif (u, v) == (3, 2):
                    items.append((y, vil, x - (PIVOT[0] + 1)*2, y - (PIVOT[1] + 1)*2))
        x, y = ox, oy + 7*16
        items.append((y + .5, unit, x - 28, y - 38))
        for _, im, x, y in sorted(items, key=lambda t: t[0]):
            box.alpha_composite(im, (int(x), int(y)))
        X, Y = (i % 2)*bw*sc, (i//2)*(bh*sc + 40)
        sheet.alpha_composite(box.resize((bw*sc, bh*sc), Image.NEAREST), (X, Y + 40))
        d.text((X + 10, Y + 8), f'숲 {i + 1} · {title}', fill=(40, 36, 30), font=font)
    out = ROOT / 'output' / out_name
    sheet.convert('RGB').save(out)
    print(out)


def forest3_candidates():
    kinds = list(FOREST3)
    sc, bw, bh = 4, 170, 110
    sheet = Image.new('RGBA', (2*bw*sc, 2*(bh*sc + 40)), (236, 232, 220, 255))
    d = ImageDraw.Draw(sheet)
    font = ImageFont.truetype(FONT, 22)
    unit = Image.open(ROOT / 'ver4/assets/graphics/units/infantry.png').crop((0, 3*46, 56, 4*46))
    for i, kind in enumerate(kinds):
        box = Image.new('RGBA', (bw, bh), (104, 150, 72, 255))
        bd = ImageDraw.Draw(box)
        ox, oy = 85, 22
        items = []
        for v in range(4):
            for u in range(4):
                x, y = ox + (u - v)*16, oy + (u + v + 1)*8
                bd.polygon([(x, y - 8), (x + 16, y), (x, y + 8), (x - 16, y)], outline=(92, 136, 62))
                if u < 3 and v < 3 or (u, v) == (3, 0):
                    items.append((y, forest3(kind, seed=u*7 + v*3 + 1), x, y))
                elif (u, v) == (3, 2):
                    items.append((y, village(4), x, y))
        for y, im, x, yy in sorted(items, key=lambda t: t[0]):
            box.alpha_composite(im, (x - PIVOT[0] - 1, yy - PIVOT[1] - 1))
        big = box.resize((bw*2, bh*2), Image.NEAREST)
        x, y = ox + (3 - 3)*16, oy + (3 + 3 + 1)*8
        big.alpha_composite(unit, (x*2 - 28, y*2 - 38))
        X, Y = (i % 2)*bw*sc, (i//2)*(bh*sc + 40)
        sheet.alpha_composite(big.resize((bw*sc, bh*sc), Image.NEAREST), (X, Y + 40))
        d.text((X + 10, Y + 8), f'숲 {i + 1} · {FOREST3[kind]}', fill=(40, 36, 30), font=font)
    out = ROOT / 'output/ver4-forest-candidates-3.png'
    sheet.convert('RGB').save(out)
    print(out)


def mountain_candidates():
    """3 x 2 mountain blocks per candidate with the chosen forest, a village and an infantry unit."""
    kinds = list(MOUNTAIN_SPOTS)
    titles = {'stone': '회색 바위산 (눈 덮인 봉우리)', 'earth': '갈색 흙산', 'green': '초록 언덕 + 바위', 'rocks': '바위 무더기'}
    sc, bw, bh = 4, 170, 110
    sheet = Image.new('RGBA', (len(kinds)*bw*sc, bh*sc + 40), (236, 232, 220, 255))
    d = ImageDraw.Draw(sheet)
    font = ImageFont.truetype(FONT, 22)
    unit = Image.open(ROOT / 'ver4/assets/graphics/units/infantry.png').crop((0, 3*46, 56, 4*46))
    for i, kind in enumerate(kinds):
        box = Image.new('RGBA', (bw, bh), (104, 150, 72, 255))
        bd = ImageDraw.Draw(box)
        ox, oy = 85, 22
        items = []
        for v in range(4):
            for u in range(4):
                x, y = ox + (u - v)*16, oy + (u + v + 1)*8
                fill = (164, 142, 96, 255) if u < 3 and v < 2 else None
                bd.polygon([(x, y - 8), (x + 16, y), (x, y + 8), (x - 16, y)], outline=(92, 136, 62), fill=fill)
                if u < 3 and v < 2:
                    items.append((y, mountain(kind), x, y))
                elif v == 2 and u < 2:
                    items.append((y, forest('mixed'), x, y))
                elif (u, v) == (3, 2):
                    items.append((y, village(4), x, y))
        for y, im, x, yy in sorted(items, key=lambda t: t[0]):
            box.alpha_composite(im, (x - PIVOT[0] - 1, yy - PIVOT[1] - 1))
        big = box.resize((bw*2, bh*2), Image.NEAREST)
        x, y = ox + (3 - 3)*16, oy + (3 + 3 + 1)*8
        big.alpha_composite(unit, (x*2 - 28, y*2 - 38))
        sheet.alpha_composite(big.resize((bw*sc, bh*sc), Image.NEAREST), (i*bw*sc, 40))
        d.text((i*bw*sc + 10, 8), f'산 {i + 1} · {titles[kind]}', fill=(40, 36, 30), font=font)
    out = ROOT / 'output/ver4-mountain-candidates.png'
    sheet.convert('RGB').save(out)
    print(out)


def forest_candidates():
    """3 x 3 forest blocks per candidate beside a village cell and an infantry unit, at game scale (x2 map)."""
    kinds = list(FOREST_SPOTS)
    titles = {'round': '활엽수', 'pine': '침엽수', 'mixed': '활엽+침엽', 'dense': '빽빽한 숲'}
    sc = 4
    bw, bh = 170, 110
    sheet = Image.new('RGBA', (len(kinds)*bw*sc, bh*sc + 40), (236, 232, 220, 255))
    d = ImageDraw.Draw(sheet)
    font = ImageFont.truetype(FONT, 22)
    unit = Image.open(ROOT / 'ver4/assets/graphics/units/infantry.png').crop((0, 3*46, 56, 4*46))  # SE idle
    for i, kind in enumerate(kinds):
        box = Image.new('RGBA', (bw, bh), (104, 150, 72, 255))
        bd = ImageDraw.Draw(box)
        ox, oy = 85, 22
        cells = [(u, v) for v in range(4) for u in range(4)]
        items = []
        for u, v in cells:
            x, y = ox + (u - v)*16, oy + (u + v + 1)*8
            bd.polygon([(x, y - 8), (x + 16, y), (x, y + 8), (x - 16, y)], outline=(92, 136, 62))
            if u < 3 and v < 3:
                items.append((y, forest(kind), x, y))
            elif (u, v) == (3, 1):
                items.append((y, village(4), x, y))
        for y, im, x, yy in sorted(items, key=lambda t: t[0]):
            box.alpha_composite(im, (x - PIVOT[0] - 1, yy - PIVOT[1] - 1))
        big = box.resize((bw*2, bh*2), Image.NEAREST)  # the game draws map art twice as large
        x, y = ox + (3 - 3)*16, oy + (3 + 3 + 1)*8
        big.alpha_composite(unit, (x*2 - 28, y*2 - 38))
        sheet.alpha_composite(big.resize((bw*sc, bh*sc), Image.NEAREST).crop((0, 0, bw*sc, bh*sc)), (i*bw*sc, 40))
        d.text((i*bw*sc + 10, 8), f'숲 {i + 1} · {titles[kind]}', fill=(40, 36, 30), font=font)
    out = ROOT / 'output/ver4-forest-candidates.png'
    sheet.convert('RGB').save(out)
    print(out)


if __name__ == '__main__':
    main()
