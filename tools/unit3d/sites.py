#!/usr/bin/env python3
"""ver5 battle-map strongpoints, one cell each, drawn with the props camera at the characters' scale (a cell
is 64 x 32, a standing officer about 1.5 model units): 주둔지 camp, 마을 village, 성 castle and the 관문 gate
in a castle wall line (gate_u: the wall runs along map u, gate_v: along v). Units stand on all of them, so
everything stays low, inside a fence or wall on the four edges of the diamond. The gate is a ground-level
courtyard whose two walls meet the 10 px castle wall blocks either side.

  uv run --with pillow --with numpy python tools/unit3d/sites.py  -> tools/spritetool/assets/ver5-sites/
"""
import json
from pathlib import Path

import numpy as np
from PIL import Image

import base_model as m
import props as P
import structures as S

ROOT = Path(__file__).resolve().parents[2]
OUT = ROOT / 'tools/spritetool/assets/ver5-sites'
V = S.V
BX, BY = S.BACK  # the back corner is (BX/2, BY/2) in cell units
CELL, PIVOT = (100, 110), (49, 76)
STONE, STONE_DARK, WOOD, TILE, TILE_DARK = S.STONE, S.STONE_DARK, S.WOOD, S.TILE, S.TILE_DARK
EARTH, EARTH_DARK = '#c49a5c', '#9c7444'
S.R.add_ramps({EARTH: 'σρρρ', EARTH_DARK: 'γτττ'})  # the map's wasteland ground (structures.CLIFF_TOP)


def edge(side, s, z=0.0, inset=0.0):
    """Point s in [-.5, .5] along an edge of the cell diamond, moved inset toward the centre.
    'nw' and 'ne' are the back edges (x = BX/2, y = BY/2), 'se' and 'sw' the front ones."""
    k = .5 - inset
    return {'nw': V(BX*k, s, z), 'se': V(-BX*k, s, z), 'ne': V(s, BY*k, z), 'sw': V(s, -BY*k, z)}[side]


def corner(kind, inset=0.0):
    """'back', 'left'/'right' (the side corners), 'front' corner, moved inset toward the centre."""
    c = {'back': V(BX, BY), 'front': V(-BX, -BY), 'left': V(BX, -BY), 'right': V(-BX, BY)}[kind]*(.5 - inset)
    return V(c[0], c[1], 0)


def roof(c, w, d, z0, h, color, dark, ov=.06):
    """Hipped tile roof over a w x d (cell units) rectangle centred on c, ridge along the longer side."""
    x0, x1, y0, y1 = c[0] - w/2 - ov, c[0] + w/2 + ov, c[1] - d/2 - ov, c[1] + d/2 + ov
    z0 += c[2]  # z0 is above c
    z1 = z0 + h
    if w >= d:
        r0, r1 = V(c[0] - (w - d)/2, c[1], z1), V(c[0] + (w - d)/2, c[1], z1)
        m.poly([V(x0, y0, z0), V(x1, y0, z0), r1, r0], color)
        m.poly([V(x0, y1, z0), r0, r1, V(x1, y1, z0)], color)
        m.poly([V(x0, y0, z0), r0, V(x0, y1, z0)], dark)
        m.poly([V(x1, y0, z0), V(x1, y1, z0), r1], dark)
    else:
        r0, r1 = V(c[0], c[1] - (d - w)/2, z1), V(c[0], c[1] + (d - w)/2, z1)
        m.poly([V(x0, y0, z0), r0, r1, V(x0, y1, z0)], color)
        m.poly([V(x1, y0, z0), V(x1, y1, z0), r1, r0], color)
        m.poly([V(x0, y0, z0), V(x1, y0, z0), r0], dark)
        m.poly([V(x0, y1, z0), r1, V(x1, y1, z0)], dark)


def stake(p, h, r=.022):
    m.rod(p, p + V(0, 0, h), r, WOOD)
    m.poly([p + V(-r, 0, h), p + V(r, 0, h), p + V(0, 0, h + .12)], P.PLANK_DARK)


def flag(p, h=2.2, color=P.BANNER):
    m.rod(p, p + V(0, 0, h), .02, WOOD)
    m.box(p + V(.09, 0, h - .3), V(.16, .01, .5), color)


def battlements(side, z, h, color=STONE, merlon=.07, ends=(-.5, .5), inset=.04):
    for s in np.arange(ends[0] + merlon, ends[1] - merlon/2, merlon*2):
        p = edge(side, s, z + h/2, inset)
        m.box(p, (V(.06, merlon, h) if side == 'nw' else V(merlon, .06, h)), color)


SIDES4 = ('nw', 'ne', 'se', 'sw')
GAP = .13  # half the entrance gap on the sw edge


def spans(side):
    return [(-.5, -GAP), (GAP, .5)] if side == 'sw' else [(-.5, .5)]


def stake_fence(h=.42):
    for side in SIDES4:
        for t0, t1 in spans(side):
            n = max(1, int((t1 - t0)/.075))
            for i in range(n):
                stake(edge(side, t0 + (i + .5)*(t1 - t0)/n, inset=.03), h, .02)
            m.rod(edge(side, t0, h*.6, .03), edge(side, t1, h*.6, .03), .012, P.ROPE)


def rail_fence(h=.3, shabby=False):
    for side in SIDES4:
        for t0, t1 in spans(side):
            n = max(1, int((t1 - t0)/.16))
            for i in range(n + 1):
                p = edge(side, t0 + i*(t1 - t0)/n, inset=.03)
                hh = h*(.75 + .5*S.noise(i, len(side), 3)) if shabby else h
                m.rod(p, p + V(0, 0, hh), .014, WOOD)
            for z in (h*.45, h*.85):
                tilt = (S.noise(int(t0*10), int(z*100), 5) - .5)*.06 if shabby else 0
                m.rod(edge(side, t0, z + tilt, .03), edge(side, t1, z - tilt, .03), .009, P.PLANK)


def barracks(roof_color=TILE, roof_dark=TILE_DARK, wall=P.PLANK):
    """병영: a long low hall across the middle of the cell, door facing the viewer."""
    w, d, h = .46, .24, .26
    m.box(V(0, 0, h/2), V(w, d, h), wall)
    for sx in (-1, 1):
        for sy in (-1, 1):
            m.box(V(sx*w/2, sy*d/2, h/2), V(.03, .03, h + .01), P.PLANK_DARK)
    roof(V(0, 0, 0), w, d, h, .2, roof_color, roof_dark, .04)
    m.box(V(0, -BY*(d/2 + .004), h*.4), V(.08, .006, h*.75), '#2a1d18')
    for k in (-.15, .15):
        m.box(V(k, -BY*(d/2 + .004), h*.62), V(.05, .006, .05), '#2a1d18')


def hut(c, w, d, h, along_x, seed):
    """허름한 집: a small thatched hut, a little crooked, with a dark door."""
    S.gable(c[0], c[1], w if along_x else d, d if along_x else w, h, h*.75, S.PLASTER if seed % 2 else EARTH,
            S.THATCH_DARK if seed % 3 else S.THATCH, S.THATCH_DARK, .05) if along_x else \
        S.gable(c[0], c[1], d, w, h, h*.75, S.PLASTER if seed % 2 else EARTH, S.THATCH_DARK if seed % 3 else S.THATCH,
                S.THATCH_DARK, .05)
    S.door(c[0], c[1], w if along_x else d, d if along_x else w, h)


def office(w=.38, d=.28, h=.28):
    """관사: a plastered hall with lacquered pillars and a hipped tile roof."""
    m.box(V(0, 0, .03), V(w + .06, d + .06, .06), STONE)
    m.box(V(0, 0, h/2 + .06), V(w*.86, d*.8, h), S.PLASTER)
    for sx in (-1, 0, 1):
        for sy in (-1, 1):
            if sx == 0 and sy == 1:
                continue
            m.box(V(sx*w*.45, sy*d*.42, h/2 + .06), V(.025, .025, h), P.LACQUER)
    roof(V(0, 0, 0), w, d, h + .06, .22, TILE, TILE_DARK, .06)
    m.box(V(0, -BY*(d*.4 + .004), .2), V(.09, .006, .2), P.LACQUER)


def camp():
    """주둔지: sharpened stake fence round the diamond, a plank barracks with a dark tile roof."""
    earth_floor()
    stake_fence()
    barracks()
    flag(corner('back', .1), 1.25)


def village():
    """마을: a crooked rail fence and four shabby thatched huts, one in each quarter."""
    earth_floor()
    rail_fence(shabby=True)
    for i, k in enumerate(('back', 'left', 'right', 'front')):
        c = corner(k, .24) + V((S.noise(i, 1) - .5)*.04, (S.noise(i, 2) - .5)*.04, 0)
        hut(c, .2 - .02*(i % 2), .15, .2 + .03*(i % 3 == 0), i % 2 == 0, i)


WALL_TOP = .52  # model units: the castle wall blocks stand 10 px high
DOOR = .14      # half the width of a gateway along its wall


def gateway(side, h, thick, roof_h=.16):
    """A gateway in the middle of the wall on `side`: lacquered leaves swung open, a lintel and a little
    tiled roof spanning the opening (the wall pieces either side are built by thin_wall)."""
    along_nw = side in ('nw', 'se')  # these walls run along model y
    size = lambda a, c, z: V(c, a, z) if along_nw else V(a, c, z)
    for s_ in (-1, 1):
        p = edge(side, s_*(DOOR - .02), h*.42, thick/2 + .03)
        m.box(p, size(.025, .09, h*.8), P.LACQUER)
    m.box(edge(side, 0, h*.88, thick/2), size(DOOR*2 + .04, thick + .02, h*.2), STONE_DARK)
    c = edge(side, 0, 0, thick/2)
    w, d = (thick + .1, DOOR*2 + .1) if along_nw else (DOOR*2 + .1, thick + .1)
    roof(c, w, d, h + .02, roof_h, TILE, TILE_DARK, .03)


def thin_wall(side, h, thick):
    """A wall along the whole edge with a gateway gap in the middle and battlements on top."""
    along_nw = side in ('nw', 'se')
    for t0, t1 in ((-.5, -DOOR), (DOOR, .5)):
        mid, ln = (t0 + t1)/2, t1 - t0
        m.box(edge(side, mid, h/2, thick/2), V(thick, ln, h) if along_nw else V(ln, thick, h), STONE)
        battlements(side, h, .1, ends=(t0, t1), inset=thick/2, merlon=.055)
    gateway(side, h, thick)


def brick_floor(n=6):
    """Paved courtyard at ground level: stone bricks with dark joints."""
    m.box(V(0, 0, .004), V(1, 1, .008), STONE_DARK)
    for i in range(n):
        for j in range(n):
            off = .5/n*(i % 2)
            x = -.5 + (j + .5)/n + off
            if x > .5:
                continue
            y = -.5 + (i + .5)/n
            m.box(V(x, y, .012), V(1/n - .02, 1/n - .02, .008), STONE)


def earth_floor(n=8):
    """Bare ochre ground inside a fence: packed earth in two tones."""
    m.box(V(0, 0, .004), V(.96, .96, .008), EARTH)
    for i in range(n):
        for j in range(n):
            if S.noise(i, j, 21) > .72:
                m.box(V(-.5 + (j + .5)/n, -.5 + (i + .5)/n, .011), V(.05, .04, .006), EARTH_DARK)


def castle():
    """성: battlemented walls on all four edges, a gateway in the middle of each, corner towers, the 관사."""
    brick_floor()
    for side in SIDES4:
        thin_wall(side, .32, .08)
    for k in ('back', 'left', 'right', 'front'):
        c = corner(k, .07)
        m.box(c + V(0, 0, .25), V(.14, .14, .5), STONE)
        roof(c + V(0, 0, .5), .14, .14, 0, .14, TILE, TILE_DARK, .03)
    office(.3, .22, .22)


def gate(along):
    """관문 in a wall line: ground-level brick courtyard; the two edges facing across the line carry thin
    walls (as tall as the wall blocks, which they meet) with a gateway each; the 관사 in the middle.
    along: 'u' or 'v', the direction the wall line runs."""
    brick_floor()
    for side in (('ne', 'sw') if along == 'u' else ('nw', 'se')):
        thin_wall(side, WALL_TOP, .1)
    office(.3, .22, .22) if along == 'v' else office(.22, .3, .22)


SITES = {'camp': camp, 'village': village, 'castle': castle, 'gate_u': lambda: gate('u'), 'gate_v': lambda: gate('v')}


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    sheet = Image.new('RGBA', (CELL[0]*len(SITES), CELL[1]))
    for i, make in enumerate(SITES.values()):
        m.meshes = []
        make()
        P.stretch()
        sheet.alpha_composite(P.draw(CELL, PIVOT), (i*CELL[0], 0))
    sheet.save(OUT / 'sites.png')
    meta = dict(version=1, image='sites.png', cell=list(CELL), pivot=[PIVOT[0] + 1, PIVOT[1] + 1],
                pivot_note='cell centre of the 64x32 map diamond', sprites={k: i for i, k in enumerate(SITES)})
    (OUT / 'sites.json').write_text(json.dumps(meta, indent=1) + '\n')
    print(OUT / 'sites.png')


if __name__ == '__main__':
    main()
