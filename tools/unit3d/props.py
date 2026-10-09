#!/usr/bin/env python3
"""ver4 scene props and the extra map tiles scenario scenes need: interior floors and walls,
pillars, the raised seat, tables, a wooden bridge, the city gate and its notice board, army
tents and banners, a tavern, an altar and a peach tree.

Same camera and cel renderer as structures.py (one model unit = one map cell, 30 degree
elevation). Unlike the battle-map structures, scene props are built to the characters' scale:
a standing officer is about 1.5 model units (30 px) tall. Props larger than a cell get their
own cell and pivot (the pivot is the footprint centre on the ground).

  python3 tools/unit3d/props.py --candidates   -> output/ver4-prop-candidates.png
  python3 tools/unit3d/props.py                -> tools/spritetool/assets/ver4-props/ (chosen props)
"""
import json
import argparse
import math
from pathlib import Path

import numpy as np
from PIL import Image, ImageDraw, ImageFont

import base_model as m
import heads as Hd
import render as R
import structures as S

ROOT = Path(__file__).resolve().parents[2]
V = S.V
FONT = S.FONT

# colors: hex -> four palette steps (shadow, base, light, highlight)
LACQUER, LACQUER_DARK, PLANK, PLANK_DARK = '#a3203a', '#7a1a2e', '#a06a3c', '#6e4126'
CANVAS, CANVAS_DARK, GOLD, ROPE = '#e9e0c4', '#bfb08a', '#d9a23a', '#8f7646'
BANNER, BANNER_DARK, CARPET = '#c4283c', '#8a1830', '#9a1f34'
BLOSSOM, BLOSSOM_DARK, BARK, LEAF = '#f2a7b8', '#d77790', '#5a3a28', '#4f8a3c'
INK, JAR, PAPER = '#2a1d18', '#8a5a3a', '#efe6cc'
R.add_ramps({LACQUER: 'bccd', LACQUER_DARK: 'bbcc', PLANK: 'ghhi', PLANK_DARK: 'fggh', CANVAS: 'TS56',
             CANVAS_DARK: 'UTSS', GOLD: 'klmm', ROPE: 'UTkk', BANNER: 'bcde', BANNER_DARK: 'bbcd',
             CARPET: 'bbcc', BLOSSOM: 'BCDD', BLOSSOM_DARK: 'ABCC', BARK: 'fggh', LEAF: 'GHIJ',
             INK: '0fff', JAR: 'fghh', PAPER: 'S566'})


TILE = 2  # model units per map cell: props are drawn at the characters' pixel scale, where a cell is 64 x 32


def stretch():
    """Pieces built in cell units (a cell spans [-.5, .5]) widened to TILE; heights stay absolute."""
    m.meshes = [(np.asarray(v)*[TILE, TILE, 1], c) for v, c in m.meshes]


def draw(cell, pivot):
    im = S.draw(m.meshes, cell=cell, pivot=pivot)
    im.pivot = (pivot[0] + 1, pivot[1] + 1)  # pixel of the footprint centre (see structures.json)
    return im


# ---------------------------------------------------------------- ground tiles (pixel)

def ground(fn, cell=(66, 34), pivot=(32, 16)):
    """A flat cell tile (a 64 x 32 diamond): fn(a, b) -> palette key for map coordinates a, b in
    [-.5, .5] (None: empty)."""
    cx, cy = pivot[0] + 1, pivot[1] + 1
    g = [['.'] * cell[0] for _ in range(cell[1])]
    for y in range(cell[1]):
        for x in range(cell[0]):
            dx, dy = x + .5 - cx, y + .5 - cy
            a, b = (dx/32 + dy/16)/2, (dy/16 - dx/32)/2
            if abs(a) <= .5 and abs(b) <= .5:
                g[y][x] = fn(a, b) or '.'
    im = Hd.to_img(g)
    im.pivot = (cx, cy)
    return im


def planks(a, b, along='a', boards=4):
    """Wooden floorboards running along one map axis, joints staggered."""
    t, s = (b, a) if along == 'a' else (a, b)
    k = (t + .5)*boards
    row = int(k)
    if k % 1 < .1:
        return 'f'
    end = ((s + .5) + .37*row) % 1
    if end < .05:
        return 'f'
    return 'h' if k % 1 < .22 else ('g' if row % 2 else 'h')


def carpet(a, b):
    """Red carpet cell with a gold border on all four edges; cells in a row read as a runner."""
    e = .5 - max(abs(a), abs(b))
    if e < .05:
        return 'b'
    if e < .11:
        return 'l'
    if e < .14:
        return 'b'
    return 'c' if (int((a + .5)*6) + int((b + .5)*6)) % 2 else 'b'


def dark_tiles(a, b):
    """Square dark fired-clay floor tiles (palace)."""
    fa, fb = (a + .5)*2 % 1, (b + .5)*2 % 1
    if fa < .08 or fb < .08:
        return '1'
    return '3' if fa < .2 or fb < .2 else '2'


def tent_floor(a, b):
    """Packed earth with a woven mat in the middle (camp tent)."""
    if max(abs(a), abs(b)) < .36:
        return 'T' if (int((a + .5)*10) % 2) else 'S'
    return 'U'


# ---------------------------------------------------------------- interior walls

WALL_H = 1.6  # model units: a little taller than a standing officer


def room_wall(sides, style='plaster'):
    """Wall pieces on the back edges of a room cell (sides from 'nw', 'ne'); the front stays open
    so the room is seen from outside, like a stage set."""
    m.meshes = []
    t = .05
    face, post, beam = {'plaster': (S.PLASTER, S.WOOD, S.WOOD), 'lacquer': (LACQUER, LACQUER_DARK, GOLD),
                        'canvas': (CANVAS, ROPE, ROPE)}[style]
    bx, by = S.BACK  # model x = map u, model y = -map v: nw is the x = bx/2 edge, ne the y = by/2 edge
    for side in sides:
        sx = side == 'nw'
        if sx:
            c, size = V(bx*(.5 - t/2), 0, WALL_H/2), V(t, 1, WALL_H)
        else:
            c, size = V(0, by*(.5 - t/2), WALL_H/2), V(1, t, WALL_H)
        m.box(c, size, face)
        m.box(c + V(0, 0, WALL_H/2 - .06), size*V(1.02, 1.02, 0) + V(0, 0, .12), beam)
        m.box(c - V(0, 0, WALL_H/2 - .05), size*V(1.02, 1.02, 0) + V(0, 0, .1), beam)
        if style == 'plaster':
            for k in (-.5, 0):  # timber posts at the ends and the middle
                p = V(bx*(.5 - t/2), k + .25, WALL_H/2) if sx else V(k + .25, by*(.5 - t/2), WALL_H/2)
                m.box(p, V(t*1.2, .03, WALL_H) if sx else V(.03, t*1.2, WALL_H), post)
        if style == 'canvas':
            for k in (-.3, .3):
                p = V(bx*(.5 - t/2 - .02), k, WALL_H/2) if sx else V(k, by*(.5 - t/2 - .02), WALL_H/2)
                m.box(p, V(.01, .02, WALL_H) if sx else V(.02, .01, WALL_H), ROPE)
    stretch()
    return draw((72, 88), (35, 63))


# ---------------------------------------------------------------- furniture

def pillar(style='lacquer'):
    m.meshes = []
    color = {'lacquer': LACQUER, 'wood': PLANK}[style]
    m.box(V(0, 0, .06), V(.42, .42, .12), S.STONE)
    m.rod(V(0, 0, .1), V(0, 0, 2.1), .14, color)
    m.box(V(0, 0, 2.12), V(.36, .36, .1), GOLD if style == 'lacquer' else S.WOOD)
    return draw((40, 80), (20, 70))


def dais(screen=True):
    """Raised seat: a two-step platform, a chair, and a folding screen behind it."""
    m.meshes = []
    bx, by = S.BACK*.25
    m.box(V(0, 0, .06), V(1.6, 1.6, .12), PLANK_DARK)
    m.box(V(bx, by, .18), V(1.1, 1.1, .12), LACQUER)
    m.box(V(bx*1.4, by*1.4, .42), V(.5, .5, .36), PLANK_DARK)  # chair
    m.box(V(bx*1.4 + bx*.4, by*1.4 + by*.4, .75), V(.5 if bx == 0 else .1, .1 if bx == 0 else .5, .7), PLANK_DARK)
    if screen:  # three panels along each back edge
        for k in (-1, 0, 1):
            m.box(V(S.BACK[0]*.62, k*.48, .8), V(.06, .46, 1.3), GOLD)
            m.box(V(k*.48, S.BACK[1]*.62, .8), V(.46, .06, 1.3), GOLD)
    return draw((80, 80), (40, 58))


def table(kind='wine'):
    """Low table; 'wine': a jar and cups, 'altar': incense burner and offerings, 'desk': brush and scroll."""
    m.meshes = []
    m.box(V(0, 0, .34), V(.86, .56, .07), PLANK if kind != 'altar' else LACQUER)
    for sx in (-1, 1):
        for sy in (-1, 1):
            m.box(V(sx*.36, sy*.22, .16), V(.06, .06, .32), PLANK_DARK if kind != 'altar' else LACQUER_DARK)
    if kind == 'wine':
        m.ell(V(-.2, 0, .5), [.13, .13, .15], JAR)
        m.rod(V(-.2, 0, .6), V(-.2, 0, .68), .06, JAR)
        for x in (.12, .28):
            m.rod(V(x, .05, .38), V(x, .05, .44), .05, S.STONE)
    elif kind == 'altar':
        m.rod(V(0, 0, .38), V(0, 0, .5), .11, GOLD)  # burner
        m.ell(V(0, 0, .52), [.12, .12, .04], GOLD)
        for x in (-.3, .3):
            m.ell(V(x, 0, .44), [.08, .08, .07], BLOSSOM)  # peaches
        m.rod(V(-.02, 0, .52), V(-.02, 0, .75), .012, BANNER)  # incense sticks
        m.rod(V(.03, 0, .52), V(.04, 0, .73), .012, BANNER)
    elif kind == 'desk':
        m.box(V(.1, 0, .39), V(.4, .3, .02), PAPER)
        m.rod(V(-.25, -.05, .4), V(-.25, .12, .4), .02, INK)
    return draw((40, 48), (20, 38))


# ---------------------------------------------------------------- bridge

def bridge(along='u', rails=True):
    """Wooden bridge cell over water; along 'u' or 'v' (the map axis it spans). Deck at water level
    plus a little, so units walk on it at ground height."""
    m.meshes = []
    ax = V(1, 0, 0) if along == 'u' else V(0, 1, 0)
    side = V(0, 1, 0) if along == 'u' else V(1, 0, 0)
    for k in range(5):  # planks across the span
        off = -.4 + k*.2
        m.box(ax*off + V(0, 0, .04), ax*.18 + side*.86 + V(0, 0, .05), PLANK if k % 2 else PLANK_DARK)
    for s in (-1, 1):  # stringers and posts
        m.box(side*s*.4 + V(0, 0, -.02), ax*1.0 + side*.08 + V(0, 0, .08), PLANK_DARK)
        if rails:
            for off in (-.4, .4):
                m.box(side*s*.42 + ax*off + V(0, 0, .2), V(.07, .07, .34), PLANK_DARK)
            m.box(side*s*.42 + V(0, 0, .34), ax*1.0 + side*.05 + V(0, 0, .05), PLANK)
    stretch()
    return draw((72, 46), (35, 19))  # stringers hang a little below the ground


# ---------------------------------------------------------------- gate and notice board

def gate(along='u', tower=True):
    """City gate for a castle wall line running along 'u' or 'v': two stone blocks with a passage
    tall enough for a standing officer, a lintel, and optionally a gatehouse with a tiled roof.
    Built in cell units and stretched to a full cell, like the walls it joins."""
    m.meshes = []
    ax = V(1, 0, 0) if along == 'u' else V(0, 1, 0)
    side = V(0, 1, 0) if along == 'u' else V(1, 0, 0)
    h, gap = 2.0, 1.6  # block height and passage height (a standing officer is about 1.5)
    for s in (-1, 1):
        m.box(ax*s*.36 + V(0, 0, h/2), ax*.28 + side*.9 + V(0, 0, h), S.STONE)
    m.box(V(0, 0, (gap + h)/2), ax*.46 + side*.9 + V(0, 0, h - gap), S.STONE)  # lintel
    m.box(V(0, 0, gap/2), ax*.44 + side*.04 + V(0, 0, gap), INK)  # dark passage beyond the opening
    if tower:
        m.box(V(0, 0, h + .35), ax*.7 + side*.6 + V(0, 0, .7), PLANK)
        for s in (-1, 1):
            for t in (-1, 1):
                m.box(ax*s*.33 + side*t*.28 + V(0, 0, h + .35), V(.03, .03, .7), LACQUER)
        z0, z1 = h + .7, h + 1.25
        a0, a1 = ax*.55, side*.5
        m.poly([V(*(-a0 - a1)) + V(0, 0, z0), V(*(a0 - a1)) + V(0, 0, z0), V(*(a0*.8)) + V(0, 0, z1),
                V(*(-a0*.8)) + V(0, 0, z1)], S.TILE)
        m.poly([V(*(-a0 + a1)) + V(0, 0, z0), V(*(-a0*.8)) + V(0, 0, z1), V(*(a0*.8)) + V(0, 0, z1),
                V(*(a0 + a1)) + V(0, 0, z0)], S.TILE)
        for s in (-1, 1):
            m.poly([V(*(ax*s*.55 - side*.5)) + V(0, 0, z0), V(*(ax*s*.44)) + V(0, 0, z1),
                    V(*(ax*s*.55 + side*.5)) + V(0, 0, z0)], S.TILE_DARK)
    stretch()
    return draw((84, 120), (41, 96))


def notice_board():
    m.meshes = []
    for s in (-1, 1):
        m.rod(V(s*.3, 0, 0), V(s*.3, 0, 1.2), .04, S.WOOD)
    m.box(V(0, 0, .95), V(.7, .05, .45), PLANK)
    m.box(V(0, -.03, .95), V(.5, .02, .32), PAPER)
    m.box(V(0, 0, 1.22), V(.86, .14, .06), S.TILE)
    return draw((40, 56), (20, 46))


# ---------------------------------------------------------------- camp

def tent(kind='ridge'):
    """Army tents. 'ridge': an A-frame tent along x; 'round': the commander's round tent with
    a pointed roof and a trim band."""
    m.meshes = []
    if kind == 'ridge':
        w, d, h = 1.5, 1.1, 1.3
        m.poly([V(-w/2, -d/2, 0), V(w/2, -d/2, 0), V(w/2, 0, h), V(-w/2, 0, h)], CANVAS)
        m.poly([V(-w/2, d/2, 0), V(-w/2, 0, h), V(w/2, 0, h), V(w/2, d/2, 0)], CANVAS)
        for x in (-w/2, w/2):
            m.poly([V(x, -d/2, 0), V(x, 0, h), V(x, d/2, 0)], CANVAS_DARK)
        front = -S.BACK[0]*w/2
        m.poly([V(front*1.002, -d/4, 0), V(front*1.002, 0, h*.5), V(front*1.002, d/4, 0)], INK)  # doorway
        m.rod(V(-w/2 - .05, 0, h + .02), V(w/2 + .05, 0, h + .02), .03, ROPE)
        return draw((72, 72), (36, 50))
    r, wall, roof = .9, .8, .95
    n = 16
    ring = [(math.cos(a), math.sin(a)) for a in np.linspace(0, 2*math.pi, n + 1)]
    for (c0, s0), (c1, s1) in zip(ring, ring[1:]):  # straight canvas wall (a rod is round-ended)
        m.poly([V(r*c0, r*s0, 0), V(r*c1, r*s1, 0), V(r*c1, r*s1, wall), V(r*c0, r*s0, wall)], CANVAS)
    for (c0, s0), (c1, s1) in zip(ring, ring[1:]):  # trim band: side faces only (a rod's cap would cover the roof)
        q = r + .01
        m.poly([V(q*c0, q*s0, wall - .16), V(q*c1, q*s1, wall - .16), V(q*c1, q*s1, wall - .02), V(q*c0, q*s0, wall - .02)], BANNER)
    rim = [V((r + .1)*math.cos(a), (r + .1)*math.sin(a), wall - .02) for a in np.linspace(0, 2*math.pi, n + 1)]
    top = V(0, 0, wall + roof)
    for a, b in zip(rim, rim[1:]):
        m.poly([b, a, top], CANVAS)
    m.rod(top, top + V(0, 0, .5), .025, S.WOOD)
    m.box(top + V(.14, 0, .4), V(.26, .02, .16), BANNER)
    f = -S.BACK*.62
    m.box(V(f[0], f[1], .3), V(.3 if S.BACK[0] == 0 else .05, .05 if S.BACK[0] == 0 else .3, .6), INK)
    return draw((80, 88), (40, 62))


def banner(kind='tall'):
    """Army banner on a pole. 'tall': a long vertical flag; 'square': a square command flag."""
    m.meshes = []
    h = 2.6
    m.rod(V(0, 0, 0), V(0, 0, h), .035, S.WOOD)
    m.ell(V(0, 0, h + .03), [.06, .06, .05], GOLD)
    if kind == 'tall':
        m.box(V(.2, 0, h - .75), V(.36, .02, 1.3), BANNER)
        m.box(V(.2, 0, h - .1), V(.4, .03, .06), BANNER_DARK)
        for z in (h - .55, h - .95):
            m.box(V(.2, -.012, z), V(.16, .01, .14), GOLD)  # emblem marks
    else:
        m.box(V(.33, 0, h - .35), V(.6, .02, .6), BANNER)
        for k in range(3):  # flame-tooth edge
            m.box(V(.66, 0, h - .14 - k*.2), V(.08, .02, .1), GOLD)
        m.box(V(.33, -.012, h - .35), V(.24, .01, .24), GOLD)
    return draw((40, 88), (14, 78))


# ---------------------------------------------------------------- village pieces

def tavern(flag=True):
    """Tavern at character scale (about 2 x 2 cells): stone footing, plastered timber walls, thatch
    with darker eaves and a ridge log, a curtained door with a red lantern, a bench and wine jars
    out front, and a wine flag on a pole. The door wall faces the viewer (away from BACK)."""
    m.meshes = []
    w, d, h, roof = 1.15, .85, .82, .46
    fy = -S.BACK[1]*d/2  # the long wall facing the viewer
    fx = -S.BACK[0]*w/2  # the gable end facing the viewer
    m.box(V(0, 0, .06), V(w + .08, d + .08, .12), S.STONE_DARK)
    m.box(V(0, 0, .12 + h/2), V(w, d, h), S.PLASTER)
    for x in (-w/2, 0, w/2):  # timber posts on the long walls
        for y in (-d/2, d/2):
            m.box(V(x, y, .12 + h/2), V(.06, .06, h + .02), S.WOOD)
    for z in (.15, .12 + h - .03):  # sill and top beam
        m.box(V(0, 0, z), V(w + .02, d + .02, .05), S.WOOD)
    z0, z1, ov = .12 + h - .02, .12 + h + roof, .07
    x0, x1, y0, y1 = -w/2 - ov, w/2 + ov, -d/2 - ov, d/2 + ov
    m.poly([V(x0, y0, z0), V(x1, y0, z0), V(x1, 0, z1), V(x0, 0, z1)], S.THATCH)
    m.poly([V(x0, y1, z0), V(x0, 0, z1), V(x1, 0, z1), V(x1, y1, z0)], S.THATCH)
    for y in (y0, y1):  # darker straw along the eaves
        m.rod(V(x0, y*1.02, z0 + .01), V(x1, y*1.02, z0 + .01), .035, S.THATCH_DARK)
    for k in range(7):  # straw strokes down both slopes
        x = x0 + .1 + k*(x1 - x0 - .2)/6
        for y in (y0, y1):
            m.rod(V(x, y*.97, z0 + .03), V(x + .03, y*.25, z1 - .1), .022, S.THATCH_DARK)
    m.box(V(0, 0, z1 - .02), V(x1 - x0 + .02, .16, .06), S.THATCH_DARK)  # ridge cap
    for x in (-w/2, w/2):
        m.poly([V(x, -d/2, z0), V(x, 0, z1 - .01), V(x, d/2, z0)], S.PLASTER)
        m.rod(V(x, -d/2, z0), V(x, 0, z1 - .02), .02, S.WOOD)
        m.rod(V(x, 0, z1 - .02), V(x, d/2, z0), .02, S.WOOD)
    m.rod(V(x0 - .03, 0, z1 + .03), V(x1 + .03, 0, z1 + .03), .045, S.WOOD)
    for x in (x0 + .04, x1 - .04):
        m.rod(V(x, -.09, z1 - .03), V(x, .09, z1 + .12), .025, S.WOOD)
        m.rod(V(x, .09, z1 - .03), V(x, -.09, z1 + .12), .025, S.WOOD)
    # curtained door on the front long wall, lantern beside it, a window on the gable end
    m.box(V(-.22, fy*1.01, .12 + .3), V(.3, .02, .58), INK)
    m.box(V(-.22, fy*1.03, .12 + .52), V(.34, .02, .16), BANNER)  # noren curtain
    m.box(V(-.22, fy*1.04, .12 + .52), V(.02, .02, .16), BANNER_DARK)
    m.rod(V(.12, fy - .08, .62), V(.12, fy - .08, .7), .02, S.WOOD)
    m.ell(V(.12, fy - .08, .58), [.06, .06, .08], BANNER)
    m.box(V(.3, fy*1.01, .12 + .42), V(.18, .02, .14), INK)  # window
    m.box(V(.3, fy*1.02, .12 + .42), V(.02, .02, .14), S.WOOD)
    m.box(V(fx*1.01, .1, .12 + .42), V(.02, .22, .14), INK)
    # bench and wine jars in front
    by = fy - S.BACK[1]*.32
    m.box(V(.25, by, .22), V(.5, .14, .04), PLANK)
    for x in (.05, .45):
        m.box(V(x, by, .1), V(.04, .1, .2), PLANK_DARK)
    for x, r in ((-.55, .13), (-.38, .1)):
        m.ell(V(x, by, r), [r, r, r*1.1], JAR)
        m.ell(V(x, by, r*2), [r*.5, r*.5, .03], PLANK_DARK)
    if flag:
        f = V(fx + S.BACK[0]*-.25, S.BACK[1]*.15, 0)  # beside the gable end, clear of the door
        m.rod(f, f + V(0, 0, 1.9), .03, S.WOOD)
        side = V(-S.BACK[0], 0, 0)
        c = f + V(0, 0, 1.45) + side*.2
        m.box(c, V(.3, .03, .66), PAPER)
        m.box(c + V(0, 0, .3), V(.32, .035, .06), BANNER)
        m.box(c + V(0, -.016, -.05), V(.12, .01, .3), INK)  # the character for wine, as a dark mark
    m.meshes = [(np.asarray(v)*1.6, c) for v, c in m.meshes]  # a building, well above a standing officer
    return draw((150, 150), (75, 108))


def peach_tree(full=True):
    m.meshes = []
    m.rod(V(0, 0, 0), V(.05, 0, .7), .09, BARK)
    m.rod(V(.05, 0, .6), V(.3, .1, 1.0), .05, BARK)
    m.rod(V(.03, 0, .6), V(-.25, -.08, 1.0), .05, BARK)
    blobs = [(0, 0, 1.25, .5), (.32, .12, 1.12, .34), (-.3, -.1, 1.1, .34), (.05, .3, 1.05, .3), (-.05, -.32, 1.05, .3)]
    if not full:
        blobs = blobs[:3]
    for x, y, z, r in blobs:
        m.ell(V(x, y, z), [r, r, r*.8], BLOSSOM)
    for x, y, z in ((.2, -.25, 1.35), (-.2, .2, 1.4), (.35, .2, 1.25)):
        m.ell(V(x, y, z), [.1, .1, .08], BLOSSOM_DARK if full else LEAF)
    m.meshes = [(np.asarray(v)*1.8, c) for v, c in m.meshes]  # as tall as the map's trees (drawn K times larger)
    return draw((88, 110), (44, 92))


# ---------------------------------------------------------------- candidates

def scale_figure():
    """The packed scene sprite of Liu Bei (SW idle), for scale next to every candidate."""
    sheet = Image.open(ROOT / 'ver4/assets/graphics/units/civ_liubei.png')
    w, h = 22, 32
    return sheet.crop((0, h, w, 2*h)), (11, 29)


CANDIDATES = [
    ('실내 바닥', [('나무 마루', lambda: ground(planks)), ('붉은 카펫', lambda: ground(carpet)),
                ('검은 전돌 (궁)', lambda: ground(dark_tiles)), ('흙바닥 + 돗자리 (막사)', lambda: ground(tent_floor))]),
    ('실내 벽', [('회벽 + 목재', lambda: room_wall(('nw', 'ne'), 'plaster')), ('붉은 칠 (궁)', lambda: room_wall(('nw', 'ne'), 'lacquer')),
               ('천막 벽 (막사)', lambda: room_wall(('nw', 'ne'), 'canvas')), ('회벽 · 한쪽 변', lambda: room_wall(('nw',), 'plaster'))]),
    ('기둥 · 상석 · 탁자', [('붉은 기둥', lambda: pillar('lacquer')), ('나무 기둥', lambda: pillar('wood')),
                      ('상석 + 병풍', lambda: dais(True)), ('술상', lambda: table('wine'))]),
    ('제단 · 탁자 · 복숭아나무', [('제단', lambda: table('altar')), ('서안 (붓·두루마리)', lambda: table('desk')),
                         ('복숭아나무 (만개)', lambda: peach_tree(True)), ('복숭아나무 (듬성)', lambda: peach_tree(False))]),
    ('나무 다리', [('u 방향 · 난간', lambda: bridge('u', True)), ('v 방향 · 난간', lambda: bridge('v', True)),
                ('u 방향 · 난간 없음', lambda: bridge('u', False)), ('v 방향 · 난간 없음', lambda: bridge('v', False))]),
    ('성문 · 방문', [('성문 u · 문루', lambda: gate('u', True)), ('성문 v · 문루', lambda: gate('v', True)),
                 ('성문 u · 문루 없음', lambda: gate('u', False)), ('방문(벽보)', notice_board)]),
    ('군막 · 깃발', [('A자 군막', lambda: tent('ridge')), ('원형 장수 막사', lambda: tent('round')),
                 ('긴 깃발', lambda: banner('tall')), ('사각 사령기', lambda: banner('square'))]),
    ('주막', [('주막 + 술 깃발', lambda: tavern(True)), ('주막 (깃발 없음)', lambda: tavern(False))]),
]


def candidates():
    sc, pad, label = 4, 12, 28
    bw, bh = 130*sc, 100*sc
    width = pad + 4*(bw + pad)
    height = pad + len(CANDIDATES)*(bh + label + pad)
    sheet = Image.new('RGBA', (width, height), (236, 232, 220, 255))
    d = ImageDraw.Draw(sheet)
    font = ImageFont.truetype(FONT, 20)
    fig, (fx, fy) = scale_figure()
    for r, (group, items) in enumerate(CANDIDATES):
        for c, (title, make) in enumerate(items):
            im = make()
            x0, y0 = pad + c*(bw + pad), pad + r*(bh + label + pad)
            d.text((x0, y0), f'{group} {c + 1} · {title}', fill=(40, 36, 30), font=font)
            box = Image.new('RGBA', (130, 100), (148, 179, 110, 255))
            bd = ImageDraw.Draw(box)
            gx, gy = 58, 72  # ground point of the prop
            for du, dv in ((0, 0), (1, 0), (0, 1), (-1, 0), (0, -1)):
                cx, cy = gx + (du - dv)*16, gy + (du + dv)*8
                bd.polygon([(cx, cy - 8), (cx + 16, cy), (cx, cy + 8), (cx - 16, cy)], outline=(128, 160, 98))
            px, py = im.pivot
            box.alpha_composite(im, (gx - px, gy - py))
            box.alpha_composite(fig, (gx + 34 - fx, gy + 10 - fy))
            sheet.alpha_composite(box.resize((bw, bh), Image.NEAREST), (x0, y0 + label))
    out = ROOT / 'output/ver4-prop-candidates.png'
    sheet.convert('RGB').save(out)
    print(out)


OUT = ROOT / 'tools/spritetool/assets/ver4-props'
# Chosen by the user (2026-10-05). Walls are single pieces per back edge, each rendered alone so no
# piece carries the other's shadow; bridges have no railings so a bridge can be any number of cells wide.
CHOSEN = [
    ('floor_wood', 'ground', lambda: ground(planks)),
    ('floor_carpet', 'ground', lambda: ground(carpet)),
    ('floor_palace', 'ground', lambda: ground(dark_tiles)),
    ('floor_tent', 'ground', lambda: ground(tent_floor)),
    ('bridge_u', 'ground', lambda: bridge('u', False)),
    ('bridge_v', 'ground', lambda: bridge('v', False)),
] + [(f'wall_{style}_{side}', 'object', lambda style=style, side=side: room_wall((side,), style))
     for style in ('plaster', 'lacquer', 'canvas') for side in ('nw', 'ne')] + [
    ('pillar_red', 'object', lambda: pillar('lacquer')),
    ('pillar_wood', 'object', lambda: pillar('wood')),
    ('dais', 'object', lambda: dais(True)),
    ('table_wine', 'object', lambda: table('wine')),
    ('altar', 'object', lambda: table('altar')),
    ('desk', 'object', lambda: table('desk')),
    ('peach_tree', 'object', lambda: peach_tree(True)),
    ('gate_u', 'object', lambda: gate('u', True)),
    ('gate_v', 'object', lambda: gate('v', True)),
    ('notice_board', 'object', notice_board),
    ('tent', 'object', lambda: tent('ridge')),
    ('tent_round', 'object', lambda: tent('round')),
    ('banner', 'object', lambda: banner('tall')),
    ('command_flag', 'object', lambda: banner('square')),
    ('tavern', 'object', lambda: tavern(True)),
] + [  # battle-map forest (user pick, 2026-10-05: round-5 pines spread over the whole cell), four variants
    (f'forest_{k}', 'object', lambda k=k: S.forest3('pine_fill', seed=k + 1, floor=False, fine=True)) for k in range(4)]


def build():
    """One sheet of the chosen props, each trimmed to its pixels; props.json gives every sprite's
    rectangle, pivot (footprint centre) and layer (ground: drawn with the floor; object: depth sorted)."""
    OUT.mkdir(parents=True, exist_ok=True)
    items = []
    for name, layer, make in CHOSEN:
        im = make()
        x0, y0, x1, y1 = im.getbbox()
        items.append((name, layer, im.crop((x0, y0, x1, y1)), (im.pivot[0] - x0, im.pivot[1] - y0)))
    width, height = sum(im.width + 1 for _, _, im, _ in items), max(im.height for _, _, im, _ in items)
    sheet = Image.new('RGBA', (width, height))
    sprites, x = {}, 0
    for name, layer, im, (px, py) in items:
        sheet.alpha_composite(im, (x, 0))
        sprites[name] = dict(rect=[x, 0, im.width, im.height], pivot=[px, py], layer=layer)
        x += im.width + 1
    sheet.save(OUT / 'props.png')
    meta = dict(version=1, image='props.png', pivot_note='footprint centre on the ground (cell centre for one-cell props)',
                axes_note='map u runs to the lower right, v to the lower left; _u/_v props span that axis', sprites=sprites)
    (OUT / 'props.json').write_text(json.dumps(meta, indent=1) + '\n')
    print(OUT / 'props.png', sheet.size)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--candidates', action='store_true')
    args = ap.parse_args()
    if args.candidates:
        candidates()
        return
    build()


if __name__ == '__main__':
    main()
