#!/usr/bin/env python3
"""ver5 story-scene props modelled in 3D: weapon rack, armour stand, candle stand, antler barrier, watchtower,
shield wall, straw mat, bloody cloth, medicine pot, broken spear, fallen flag, cauldron, grain sack, wine jar, grain
cart and the big command tent.

Same camera and cel renderer as structures.py (one model unit = one 32 x 16 scene-map cell, 30 degree elevation,
seen from SW: +x runs to the lower right along map u, +y to the upper right). Heights are at the scene people's
scale: an officer on foot is about 1.9 units (38 px). After the render, RETOUCH puts back by hand the few pixels a
render this small loses (a blade tip, a candle flame, a stain). Each sprite is trimmed to its pixels, its pivot is
the footprint centre on the ground.

  python3 tools/unit3d/scene_props3d.py            -> assets_candidate/tiles/<name>.png and their tiles.json entries
  python3 tools/unit3d/scene_props3d.py --review   -> output/scene-props3d.png (4x, with an officer for scale)
  python3 tools/unit3d/scene_props3d.py campfire bench --review --out=/tmp/x.png   -> only those props
then python3 ver5/tools/build_extra.py packs them into the game's extra sheet.

The scene trees (stree_*, peach_bloom_*) are no longer these renders on the map: codex painted each from its render
and ver5/tools/pixelize_tree.py turned the paintings into dots (assets_candidate/tiles/sources/trees.json). A run
leaves their sprites alone; --3d writes their renders to output/scene-props3d/ as references for a repaint, and
--review still shows the render under the painted sprite.
"""
import json
import math
import sys
from pathlib import Path

import numpy as np
from PIL import Image

import base_model as m
import props as P
import render as R
import structures as S

ROOT = Path(__file__).resolve().parents[2]
CAND = ROOT / 'assets_candidate'
V = S.V

STEEL, DARK_STEEL, BLADE, SHAFT = '#636971', '#30343a', '#bad9f1', '#966131'
LEATHER, RED, GOLD = '#754323', '#b91828', '#d9a23a'
IRON, IRON_RIM, BRONZE = '#3a3d44', '#5a5e66', '#b07a34'
STRAW, STRAW_DARK, HEMP, CLAY, CLAY_DARK = '#d6bf84', '#a68c58', '#c9b48a', '#b0603a', '#7e3e26'
LINEN, BLOOD, WAX, FLAME, ASH, THATCH_SHADE = '#e8e2d2', '#8a1a22', '#f2ecd8', '#ffd24a', '#4a4440', '#8a7448'
CANVAS_LIGHT = '#e9e0c4'
R.add_ramps({IRON: '0112', IRON_RIM: '2345', BRONZE: 'gkll', STRAW: 'TSSS', STRAW_DARK: 'UTTS', HEMP: 'UTSS',
             CLAY: 'KLLM', CLAY_DARK: 'KKLL', LINEN: '4566', BLOOD: 'bbbc', WAX: '5666', FLAME: 'lmmm', ASH: '0112', THATCH_SHADE: 'UUTT'})


def draw(cell=(96, 120), pivot=(48, 96)):
    """Render m.meshes, trimmed to its pixels; pivot: the footprint centre in the trimmed image."""
    im = S.draw(m.meshes, cell=cell, pivot=pivot)
    x0, y0, x1, y1 = im.getbbox()
    return im.crop((x0, y0, x1, y1)), [pivot[0] + 1 - x0, pivot[1] + 1 - y0]


def disc(a, b, r, color, n=14, ends=True):
    """A flat-ended cylinder from a to b (m.rod rounds its ends into balls, too big for trays and rims); ends=False:
    an open band (a rim)."""
    a, b = V(*a), V(*b)
    ring = [V(r*math.cos(i*math.tau/n), r*math.sin(i*math.tau/n), 0) for i in range(n)]
    for i in range(n):
        m.poly([a + ring[i], a + ring[(i + 1) % n], b + ring[(i + 1) % n], b + ring[i]], color)
    if ends:
        m.poly([b + q for q in ring], color)
        m.poly([a + q for q in reversed(ring)], color)


def bowl(c, r, depth, color, n=14):
    """The lower half of an ellipsoid, open at the top (a cauldron, a pot): the rim at c, the bottom depth below."""
    c = V(*c)
    for j in range(4, 8):
        for i in range(n):
            pts = []
            for a, b in ((j, i), (j, i + 1), (j + 1, i + 1), (j + 1, i)):
                u, t = math.pi*a/8, math.tau*b/n
                pts.append(c + [r*math.sin(u)*math.cos(t), r*math.sin(u)*math.sin(t), depth*math.cos(u)])
            m.poly(pts, color)


def spear(a, b, r=.035, head='spear'):
    """A pole arm from foot a to top b with its head: 'spear' a leaf blade, 'halberd' a blade and a hook."""
    a, b = V(*a), V(*b)
    d = (b - a) / np.linalg.norm(b - a)
    m.rod(a, b, r, SHAFT)
    m.rod(b, b + d*.24, r*1.1, BLADE)
    if head == 'halberd':
        side = np.cross(d, [1, -1, 0])
        side /= np.linalg.norm(side)
        m.box(b - d*.02 + side*.1, [.05, .05, .14] if abs(d[2]) > .9 else [.14, .05, .05], BLADE)


def weapon_rack():
    """A timber rack along u, two posts and two rails, spears and halberds standing in it."""
    m.meshes = []
    for x in (-.42, .42):
        m.box(V(x, 0, .8), V(.09, .09, 1.6), P.PLANK)
        m.box(V(x, 0, .04), V(.16, .36, .08), P.PLANK_DARK)  # feet
    for z in (.32, 1.25):
        m.box(V(0, 0, z), V(.96, .08, .07), P.PLANK)
    for i, x in enumerate((-.3, -.1, .1, .3)):
        spear((x, -.04, .1), (x + .02, .01, 1.62 + .1*(i % 2)), head='halberd' if i % 2 else 'spear')
    return draw()


def armor_stand():
    """A cross-shaped stand wearing a lamellar coat with red edging, a helmet on the post."""
    m.meshes = []
    m.box(V(0, 0, .04), V(.5, .5, .08), P.PLANK_DARK)
    m.box(V(0, 0, .8), V(.07, .07, 1.6), P.PLANK_DARK)
    m.box(V(0, 0, 1.28), V(.72, .07, .07), P.PLANK_DARK)  # shoulders
    m.box(V(0, 0, .96), V(.5, .3, .66), STEEL)  # coat
    for z in (.8, .98, 1.14):  # rows of plates
        m.box(V(0, -.005, z), V(.52, .31, .03), DARK_STEEL)
    m.box(V(0, 0, .62), V(.54, .32, .05), RED)  # hem
    m.box(V(0, -.16, 1.0), V(.08, .02, .58), RED)  # front opening
    for x in (-.34, .34):  # shoulder guards
        m.box(V(x, 0, 1.2), V(.18, .3, .16), STEEL)
        m.box(V(x, 0, 1.11), V(.19, .31, .03), RED)
    m.ell(V(0, 0, 1.5), [.17, .17, .16], STEEL)  # helmet
    m.box(V(0, 0, 1.38), V(.38, .38, .04), DARK_STEEL)
    m.rod(V(0, 0, 1.64), V(0, 0, 1.76), .02, RED)  # plume
    m.ell(V(0, 0, 1.8), [.05, .05, .06], RED)
    return draw()


def candle_stand():
    """A bronze candle stand: three feet, a slim column, a drip tray and a lit candle."""
    m.meshes = []
    for k in range(3):
        a = k*math.tau/3 + .4
        m.rod(V(0, 0, .14), V(.2*math.cos(a), .2*math.sin(a), .02), .025, BRONZE)
    m.rod(V(0, 0, .1), V(0, 0, 1.3), .035, BRONZE)
    m.ell(V(0, 0, .6), [.06, .06, .04], BRONZE)
    disc((0, 0, 1.3), (0, 0, 1.33), .15, BRONZE)  # tray
    m.rod(V(0, 0, 1.33), V(0, 0, 1.5), .045, WAX)
    m.ell(V(0, 0, 1.6), [.03, .03, .08], FLAME)
    return draw()


def antler_barrier():
    """녹각: a log along u with sharpened stakes crossed through it, points out both ways."""
    m.meshes = []
    m.rod(V(-.9, 0, .22), V(.9, 0, .22), .09, P.PLANK_DARK)
    for i, x in enumerate(np.linspace(-.75, .75, 6)):
        for s in (-1, 1):
            lean = .05*((i*3 + s) % 3 - 1)
            m.rod(V(x - s*.05, s*-.36, .02), V(x + s*.08 + lean, s*.36, .66), .035, P.PLANK)
            m.rod(V(x + s*.08 + lean, s*.36, .66), V(x + s*.1 + lean, s*.42, .74), .02, STRAW)  # bared point
    for x in (-.5, .25):  # rope lashings
        m.rod(V(x, 0, .12), V(x, 0, .33), .1, P.ROPE)
    return draw()


def watchtower():
    """A timber watchtower: four splayed posts with cross braces, a planked platform with a breastwork and a
    shallow hip roof over it, a ladder on the front, a red pennant."""
    m.meshes = []
    top, h = .42, 1.9
    corners = [(sx, sy) for sx in (-1, 1) for sy in (-1, 1)]
    for sx, sy in corners:
        m.rod(V(sx*.55, sy*.55, 0), V(sx*top, sy*top, h), .05, P.PLANK_DARK)
    for (ax, ay), (bx, by) in (((-1, -1), (1, -1)), ((1, -1), (1, 1)), ((-1, 1), (1, 1)), ((-1, -1), (-1, 1))):
        for z0, z1 in ((.12, .9), (.9, .12), (1.0, 1.8), (1.8, 1.0)):
            k0, k1 = .55 - (.55 - top)*z0/h, .55 - (.55 - top)*z1/h
            m.rod(V(ax*k0, ay*k0, z0), V(bx*k1, by*k1, z1), .025, P.PLANK)
    m.box(V(0, 0, h), V(1.1, 1.1, .1), P.PLANK)  # platform
    for sx, sy, w, d in ((0, -1, 1.1, .06), (1, 0, .06, 1.1), (0, 1, 1.1, .06), (-1, 0, .06, 1.1)):
        m.box(V(sx*.53, sy*.53, h + .22), V(w, d, .36), P.PLANK_DARK if sx or sy > 0 else P.PLANK)
    for x in (-.3, 0, .3):  # breastwork slats
        m.box(V(x, -.56, h + .22), V(.04, .02, .36), P.PLANK_DARK)
    for sx, sy in corners:
        m.rod(V(sx*.5, sy*.5, h), V(sx*.5, sy*.5, h + .75), .035, P.PLANK_DARK)
    eave, peak = h + .7, h + 1.75
    c = [V(sx*.72, sy*.72, eave) for sx, sy in ((-1, -1), (1, -1), (1, 1), (-1, 1))]
    for a, b in zip(c, c[1:] + c[:1]):
        m.poly([a, b, V(0, 0, peak)], STRAW if a[0] + b[0] < 0 else THATCH_SHADE)  # thatch, sunlit on the left
    for z in np.arange(.25, h, .25):  # ladder up the front
        m.box(V(.2, -.58 - (h - z)*.06, z), V(.3, .04, .04), P.PLANK)
    for x in (.06, .34):
        m.rod(V(x, -.7, 0), V(x, -.58, h + .15), .025, P.PLANK)
    m.rod(V(0, 0, peak - .1), V(0, 0, peak + .55), .02, P.PLANK_DARK)  # pennant on the peak
    m.box(V(.14, 0, peak + .45), V(.26, .02, .14), RED)
    return draw()


def shield_wall():
    """Four tall plank shields along u, each braced from behind and leaning a little, red bands and a boss."""
    m.meshes = []
    for i, x in enumerate((-.84, -.28, .28, .84)):
        y = -.3*(i % 2)
        m.box(V(x, y, .62), V(.44, .07, 1.24), LEATHER)
        for z in (.25, 1.0):
            m.box(V(x, y - .04, z), V(.44, .02, .07), RED)
        m.ell(V(x, y - .05, .66), [.08, .04, .08], STEEL)
        m.rod(V(x, y + .04, 1.0), V(x, y + .45, 0), .025, P.PLANK_DARK)  # brace
    return draw()


def straw_mat():
    """A woven straw mat on the ground, a cell long along u, the weave in alternating strips and a bound edge."""
    m.meshes = []
    m.box(V(0, 0, .015), V(1.2, .7, .03), STRAW_DARK)
    for i, x in enumerate(np.linspace(-.54, .54, 10)):
        m.box(V(x, 0, .035), V(.08, .62, .01), STRAW if i % 2 else HEMP)
    return draw()


def bloody_cloth():
    """A bandage cloth dropped in a heap, a dark stain soaking through."""
    m.meshes = []
    m.box(V(0, 0, .02), V(.46, .3, .04), LINEN)
    m.box(V(.08, .06, .06), V(.24, .2, .05), LINEN)
    m.box(V(-.12, -.06, .045), V(.16, .1, .01), BLOOD)
    m.box(V(.1, .04, .09), V(.1, .08, .01), BLOOD)
    m.rod(V(.2, -.12, .02), V(.42, -.2, .02), .03, LINEN)  # a loose end
    return draw()


def medicine_pot():
    """A round clay herb pot with a spout and a side handle, on a wide charcoal brazier glowing underneath."""
    m.meshes = []
    disc((0, 0, 0), (0, 0, .14), .22, IRON)
    disc((0, 0, .14), (0, 0, .16), .24, IRON_RIM)
    m.ell(V(0, 0, .17), [.2, .2, .02], FLAME)  # embers
    m.ell(V(0, 0, .32), [.2, .2, .14], CLAY)
    m.ell(V(0, 0, .44), [.1, .1, .04], CLAY_DARK)  # lid
    m.ell(V(0, 0, .49), [.03, .03, .03], CLAY_DARK)
    m.rod(V(.16, -.1, .32), V(.3, -.2, .42), .025, CLAY_DARK)  # spout
    m.rod(V(-.18, .1, .36), V(-.38, .2, .4), .025, P.PLANK_DARK)  # handle
    return draw()


def broken_spear():
    """A spear snapped in two on the ground, the splintered ends apart."""
    m.meshes = []
    m.rod(V(-.7, .1, .04), V(-.08, -.02, .04), .03, SHAFT)
    m.rod(V(-.08, -.02, .04), V(0, -.03, .06), .015, STRAW)  # splinter
    m.rod(V(.08, .05, .04), V(.5, -.1, .04), .03, SHAFT)
    m.rod(V(.5, -.1, .04), V(.76, -.19, .04), .045, BLADE)
    m.ell(V(.5, -.1, .04), [.04, .04, .04], RED)  # tassel
    return draw()


def fallen_flag():
    """A flag pole fallen along u, the red cloth crumpled on the ground beside it."""
    m.meshes = []
    m.rod(V(-.8, .05, .04), V(.7, -.05, .04), .03, P.PLANK_DARK)
    m.ell(V(.72, -.05, .05), [.04, .04, .04], GOLD)
    for i, (x, y, w) in enumerate(((-.5, -.2, .32), (-.22, -.28, .3), (.04, -.22, .26))):
        m.box(V(x, y, .03 + .025*(i % 2)), V(w, .3, .05), P.BANNER if i % 2 else P.BANNER_DARK)
    m.box(V(-.22, -.28, .085), V(.12, .1, .01), GOLD)  # the emblem showing in a fold
    return draw()


def cauldron():
    """An iron camp cauldron, wide and shallow, on three legs over burning logs: the rim and the gruel inside show."""
    m.meshes = []
    for k in range(3):
        a = k*math.tau/3 + .3
        m.rod(V(.24*math.cos(a), .24*math.sin(a), .25), V(.32*math.cos(a), .32*math.sin(a), 0), .03, IRON)
    m.rod(V(-.26, -.12, .05), V(.24, .14, .05), .045, P.PLANK_DARK)  # firewood
    m.rod(V(-.22, .16, .05), V(.26, -.1, .05), .045, P.PLANK)
    m.ell(V(0, -.02, .1), [.1, .1, .05], FLAME)
    bowl((0, 0, .58), .36, .36, IRON)
    disc((0, 0, .56), (0, 0, .6), .37, IRON_RIM, ends=False)
    disc((0, 0, .5), (0, 0, .52), .33, HEMP)  # gruel
    for s in (-1, 1):  # lugs
        m.rod(V(s*.36, 0, .56), V(s*.44, 0, .62), .03, IRON_RIM)
    return draw()


def grain_sack():
    """A full hemp sack, tied at the neck, slumped a little."""
    m.meshes = []
    m.ell(V(0, 0, .22), [.24, .2, .22], HEMP)
    m.ell(V(0, 0, .42), [.12, .11, .07], HEMP)
    disc((0, 0, .45), (0, 0, .49), .075, P.ROPE)
    m.ell(V(0, 0, .56), [.07, .06, .07], STRAW_DARK)  # the tied top
    return draw()


def wine_jar():
    """A fired clay wine jar, wide shouldered, its mouth bound with cloth and cord."""
    m.meshes = []
    m.ell(V(0, 0, .3), [.22, .22, .3], CLAY)
    disc((0, 0, 0), (0, 0, .08), .13, CLAY_DARK)
    disc((0, 0, .55), (0, 0, .64), .1, CLAY_DARK)
    m.ell(V(0, 0, .68), [.13, .13, .06], HEMP)
    disc((0, 0, .63), (0, 0, .655), .115, P.ROPE)
    return draw()


WATER, STONE_GREY, PEACH_LEAF, CHAR, SOUP = '#49adf0', '#93a3af', '#79b866', '#3d211d', '#e2a061'
CLOTH_BLUE, CLOTH_DARK, CABBAGE, RADISH, PERSIMMON = '#3567a6', '#264d80', '#98b242', '#f1f6f5', '#e69d23'
R.add_ramps({WATER: 'opqq', STONE_GREY: '2344', PEACH_LEAF: 'IJJJ', CHAR: '0ffg', SOUP: 'ijjj',
             CLOTH_BLUE: 'XYZZ', CLOTH_DARK: 'WXYY', CABBAGE: 'PQRR', RADISH: '4566', PERSIMMON: 'kllm'})


def wheel(x, y, z, r, side):
    """A cart wheel standing in the x-z plane at (x, y, z): an iron-shod rim, a hub and six spokes (disc() only
    turns round a vertical axis, so the rim is built here)."""
    n, t = 20, .05
    for i in range(n):
        a0, a1 = i*math.tau/n, (i + 1)*math.tau/n
        for rr, c in ((r, IRON), (r - .07, P.PLANK_DARK)):
            p0, p1 = V(x + rr*math.cos(a0), 0, z + rr*math.sin(a0)), V(x + rr*math.cos(a1), 0, z + rr*math.sin(a1))
            m.poly([p0 + [0, y - t, 0], p1 + [0, y - t, 0], p1 + [0, y + t, 0], p0 + [0, y + t, 0]], c)
        q = [V(x + rr*math.cos(a), y - t*side, z + rr*math.sin(a)) for rr, a in ((r, a0), (r, a1), (r - .09, a1), (r - .09, a0))]
        m.poly(q, P.PLANK_DARK)  # outer face of the felloe
    m.rod(V(x, y - .1*side, z), V(x, y + .1*side, z), .09, IRON)
    for k in range(6):
        a = k*math.pi/3
        m.rod(V(x + .08*math.cos(a), y - .02*side, z + .08*math.sin(a)), V(x + (r - .08)*math.cos(a), y - .02*side, z + (r - .08)*math.sin(a)), .03, SHAFT)


def prison_cart():
    """함거: a cage of square bars on a two-wheeled cart bed, a plank roof, the shafts resting on the ground. Tall
    enough for an officer standing in it: put it down with a beat's Prop step on the prisoner's cell, so it draws
    over him and he shows between the bars above the bed; he stands best a third of a cell behind its centre."""
    m.meshes = []
    bed, top, L, W = .3, 2.3, 1.6, 1.15
    m.box(V(0, 0, bed - .06), V(L + .1, W + .1, .12), P.PLANK_DARK)  # bed
    for s in (-1, 1):  # shafts running forward to the ground
        m.box(V(L/2 + .6, s*.34, .2), V(1.3, .07, .07), P.PLANK_DARK)
    m.box(V(L/2 + 1.2, 0, .12), V(.07, .76, .07), P.PLANK_DARK)  # yoke bar
    for sx in (-1, 1):
        for sy in (-1, 1):
            m.box(V(sx*(L/2 - .04), sy*(W/2 - .04), (bed + top)/2), V(.1, .1, top - bed), P.PLANK)
    # bars only on the two sides facing the viewer: at this size bars on all four sides plus their outlines
    # close up into a solid box and hide the prisoner standing inside
    for x in np.linspace(-L/2, L/2, 4)[1:-1]:
        m.box(V(x, -(W/2 - .04), (bed + top)/2), V(.045, .045, top - bed), SHAFT)
    m.box(V(L/2 - .04, 0, (bed + top)/2), V(.045, .045, top - bed), SHAFT)
    for z in (bed + .06, top - .08):  # rails
        m.box(V(0, -(W/2 - .04), z), V(L, .07, .07), P.PLANK_DARK)
        m.box(V(0, W/2 - .04, z), V(L, .07, .07), P.PLANK_DARK)
        m.box(V(L/2 - .04, 0, z), V(.07, W, .07), P.PLANK_DARK)
        m.box(V(-(L/2 - .04), 0, z), V(.07, W, .07), P.PLANK_DARK)
    m.box(V(0, 0, top + .04), V(L + .2, W + .2, .08), P.PLANK)  # roof
    for sy in (-1, 1):
        wheel(-.1, sy*(W/2 + .16), .45, .45, sy)
    m.rod(V(-.1, -W/2 - .16, .45), V(-.1, W/2 + .16, .45), .035, IRON)  # axle
    return draw(cell=(160, 180), pivot=(70, 140))


def grain_cart(draw_it=True):
    """군량 수레: a two-wheeled cart, its bed piled with tied hemp grain sacks under a cross rope, the shafts run
    forward (+x) down to the ground on a prop block. Long along u."""
    m.meshes = []
    bed, L, W, x0 = .55, 1.3, .8, -.35
    m.box(V(x0, 0, bed - .05), V(L, W, .1), P.PLANK_DARK)  # bed
    for s in (-1, 1):  # side boards
        m.box(V(x0, s*(W/2 - .03), bed + .1), V(L, .05, .22), P.PLANK)
    m.box(V(x0 - L/2 + .03, 0, bed + .1), V(.05, W, .22), P.PLANK)
    for s in (-1, 1):  # shafts from under the bed forward to the ground
        m.rod(V(x0 + L/2 - .2, s*.3, bed - .1), V(x0 + L/2 + .95, s*.24, .1), .04, P.PLANK_DARK)
    m.box(V(x0 + L/2 + .95, 0, .05), V(.12, .6, .1), P.PLANK_DARK)  # block under the shaft ends
    for x, y, z in ((-.7, -.18, .82), (-.35, -.2, .82), (0, -.18, .82), (-.7, .2, .82), (-.35, .2, .82),
                    (0, .2, .82), (-.52, 0, 1.08), (-.17, 0, 1.08)):
        m.ell(V(x, y, z), [.19, .17, .17], HEMP)
        m.ell(V(x + .14, y, z + .04), [.05, .05, .05], STRAW_DARK)  # the tied neck
    for x in (-.6, -.1):  # ropes over the load
        m.rod(V(x, -W/2, bed + .2), V(x, 0, 1.27), .02, P.ROPE)
        m.rod(V(x, 0, 1.27), V(x, W/2, bed + .2), .02, P.ROPE)
    for sy in (-1, 1):
        wheel(x0, sy*(W/2 + .1), .42, .42, sy)
    m.rod(V(x0, -W/2 - .1, .42), V(x0, W/2 + .1, .42), .035, IRON)  # axle
    return draw(cell=(140, 140), pivot=(66, 104)) if draw_it else None


def grain_cart_v():
    """The grain cart long along v (its shafts toward the viewer's lower left)."""
    grain_cart(draw_it=False)
    m.meshes = [(np.array([[p[1], -p[0], p[2]] for p in pts]), c) for pts, c in m.meshes]
    return draw(cell=(140, 140), pivot=(74, 104))


def command_tent():
    """큰 막사: the commander's pavilion tent, about two game cells square: straight canvas walls on posts, a hip
    roof along u with a ridge pole, a red scalloped valance under the eaves, the door on the long wall facing the
    viewer (-y, the lower left) tied open under a small porch awning on two poles, a pennant at each ridge end."""
    m.meshes = []
    w, d, wh, peak, ov = 3.4, 2.8, 1.45, 2.9, .18
    for y, c in ((-d/2, P.CANVAS), (d/2, P.CANVAS)):
        m.poly([V(-w/2, y, 0), V(w/2, y, 0), V(w/2, y, wh), V(-w/2, y, wh)], c)
    for x in (-w/2, w/2):
        m.poly([V(x, -d/2, 0), V(x, d/2, 0), V(x, d/2, wh), V(x, -d/2, wh)], P.CANVAS_DARK if x > 0 else P.CANVAS)
    for sx in (-1, 1):  # corner posts and a middle post on the long walls
        for sy in (-1, 1):
            m.box(V(sx*w/2, sy*d/2, wh/2), V(.08, .08, wh), S.WOOD)
    for x in (-w/4, w/4):
        m.box(V(x, -d/2 - .01, wh/2), V(.06, .02, wh), P.CANVAS_DARK)  # seams
    door = .9  # the open doorway and its flaps tied back
    m.box(V(0, -d/2 - .01, .5), V(door, .02, 1.0), P.INK)
    for s in (-1, 1):
        m.poly([V(s*door/2, -d/2 - .03, 1.1), V(s*(door/2 + .3), -d/2 - .03, 1.1), V(s*(door/2 + .05), -d/2 - .03, .25)],
               P.CANVAS_DARK)
    z0, x0, x1, y0, y1 = wh - .02, -w/2 - ov, w/2 + ov, -d/2 - ov, d/2 + ov
    r0, r1 = -w/2 + d/2 - .2, w/2 - d/2 + .2  # the ridge ends
    m.poly([V(x0, y0, z0), V(x1, y0, z0), V(r1, 0, peak), V(r0, 0, peak)], P.CANVAS)
    m.poly([V(x0, y1, z0), V(r0, 0, peak), V(r1, 0, peak), V(x1, y1, z0)], P.CANVAS_DARK)
    m.poly([V(x0, y0, z0), V(r0, 0, peak), V(x0, y1, z0)], P.CANVAS)
    m.poly([V(x1, y0, z0), V(x1, y1, z0), V(r1, 0, peak)], P.CANVAS_DARK)
    for t in np.linspace(.12, .88, 6):  # seams down the front and back slopes
        for y in (y0, y1):
            a, b = V(x0 + (x1 - x0)*t, y*1.003, z0 + .005), V(r0 + (r1 - r0)*t, 0, peak)
            m.poly([a - [.025, 0, 0], a + [.025, 0, 0], b + [.02, 0, 0], b - [.02, 0, 0]], P.CANVAS_DARK)
    m.rod(V(r0 - .05, 0, peak + .02), V(r1 + .05, 0, peak + .02), .05, S.WOOD)  # ridge pole
    for a, b, side in ((V(x0, y0, z0), V(x1, y0, z0), 0), (V(x1, y0, z0), V(x1, y1, z0), 1),
                       (V(x0, y1, z0), V(x0, y0, z0), 1), (V(x1, y1, z0), V(x0, y1, z0), 0)):
        n = max(3, int(np.linalg.norm(b - a)/.42))
        for i in range(n):  # the valance: a band and a scallop under each stretch
            p, q = a + (b - a)*i/n, a + (b - a)*(i + 1)/n
            m.poly([p, q, q - [0, 0, .16], p - [0, 0, .16]], P.BANNER)
            m.poly([p - [0, 0, .16], q - [0, 0, .16], (p + q)/2 - [0, 0, .28]], P.BANNER_DARK if side else P.BANNER)
    for x in (-door/2 - .15, door/2 + .15):  # porch: two poles and an awning over the door
        m.rod(V(x, -d/2 - .75, 0), V(x, -d/2 - .75, 1.0), .035, S.WOOD)
    m.poly([V(-door/2 - .25, -d/2 - .02, 1.2), V(door/2 + .25, -d/2 - .02, 1.2), V(door/2 + .25, -d/2 - .8, 1.0),
            V(-door/2 - .25, -d/2 - .8, 1.0)], P.CANVAS_DARK)
    m.poly([V(-door/2 - .25, -d/2 - .8, 1.0), V(door/2 + .25, -d/2 - .8, 1.0), V(door/2 + .25, -d/2 - .81, .88),
            V(-door/2 - .25, -d/2 - .81, .88)], P.BANNER)
    for x in (r0, r1):  # pennants
        m.rod(V(x, 0, peak), V(x, 0, peak + .7), .025, S.WOOD)
        m.box(V(x + .2, 0, peak + .55), V(.36, .02, .22), P.BANNER)
    return draw(cell=(260, 240), pivot=(130, 180))


def water_bucket():
    """A wooden water bucket: staves with two iron hoops, water at the brim, a rope handle."""
    m.meshes = []
    disc((0, 0, 0), (0, 0, .4), .19, P.PLANK)
    disc((0, 0, .4), (0, 0, .42), .2, P.PLANK_DARK, ends=False)
    for z in (.08, .3):
        disc((0, 0, z), (0, 0, z + .04), .2, IRON, ends=False)
    disc((0, 0, .34), (0, 0, .37), .17, WATER)
    for t0, t1 in ((0, .5), (.5, 1)):  # handle over the top
        a0, a1 = math.pi*t0, math.pi*t1
        m.rod(V(.19*math.cos(a0), 0, .42 + .22*math.sin(a0)), V(.19*math.cos(a1), 0, .42 + .22*math.sin(a1)), .015, P.ROPE)
    return draw()


def water_jar():
    """A big clay water jar (물동이), a wide mouth showing the water, a dark lip."""
    m.meshes = []
    m.ell(V(0, 0, .32), [.28, .28, .3], CLAY)
    disc((0, 0, 0), (0, 0, .08), .17, CLAY_DARK)
    disc((0, 0, .55), (0, 0, .64), .2, CLAY_DARK)
    disc((0, 0, .6), (0, 0, .62), .16, WATER)
    disc((0, 0, .64), (0, 0, .67), .22, CLAY, ends=False)
    return draw()


def campfire():
    """모닥불: a ring of field stones round a crossed stack of logs on charred ground. The flames are the scene's
    fire effect; the embers glow low between the logs."""
    m.meshes = []
    m.box(V(0, 0, .005), V(.8, .8, .01), CHAR)
    for k in range(9):
        a = k*math.tau/9 + .2
        r = .48 + .03*(k % 2)
        m.ell(V(r*math.cos(a), r*math.sin(a), .07), [.1 + .02*(k % 3), .09, .08 + .015*(k % 2)], STONE_GREY)
    m.ell(V(0, 0, .06), [.22, .22, .05], FLAME)
    for k in range(5):  # logs leaning together over the embers
        a = k*math.tau/5 + .5
        m.rod(V(.36*math.cos(a), .36*math.sin(a), .04), V(.05*math.cos(a), .05*math.sin(a), .4), .045, P.PLANK_DARK if k % 2 else SHAFT)
    m.rod(V(-.3, -.12, .05), V(.3, .1, .05), .05, P.PLANK_DARK)
    return draw()


def notice_post():
    """방문: a board on two posts under a small gable roof, two papers pasted on its front, one with a face."""
    m.meshes = []
    for x in (-.5, .5):
        m.box(V(x, 0, .85), V(.08, .08, 1.7), P.PLANK_DARK)
    m.box(V(0, 0, 1.15), V(1.0, .06, .8), P.PLANK)  # board
    m.box(V(0, -.035, 1.55), V(1.06, .04, .05), P.PLANK_DARK)
    m.box(V(0, -.035, .75), V(1.06, .04, .05), P.PLANK_DARK)
    m.box(V(-.24, -.04, 1.15), V(.36, .01, .6), P.PAPER)  # the wanted notice with a face
    m.box(V(.22, -.04, 1.13), V(.34, .01, .52), P.PAPER)  # a written notice
    for s in (-1, 1):  # roof
        m.poly([V(-.65, 0, 1.92), V(.65, 0, 1.92), V(.65, s*.28, 1.74), V(-.65, s*.28, 1.74)], P.PLANK_DARK if s > 0 else P.PLANK)
    return draw()


BIG = dict(cell=(300, 340), pivot=(150, 290))  # render cell for the trees, a few people tall
PINE_LEAF, PINE_LIGHT, BROAD, BROAD_LIGHT, BROAD_DARK = S.PINE, '#4f7030', S.OLIVE, S.OLIVE_LIGHT, S.OLIVE_DARK
PINE_BARK, OLD_BARK, PLUM_WHITE = '#8e3326', '#75604a', '#f1f6f5'
R.add_ramps({PINE_LIGHT: 'NOPQ', PINE_BARK: 'fKKL', OLD_BARK: 'UUTS', PLUM_WHITE: 'D566'})


def crown(rnd, centres, n, spread, r0, r1, colors, flat=.8):
    """Leaf clumps scattered round each centre, drawn lowest first."""
    clumps = []
    for c in centres:
        for _ in range(n):
            clumps.append((c + V(*rnd.uniform(-spread, spread, 2), rnd.uniform(-spread*.4, spread*.5)), rnd.uniform(r0, r1)))
    for c, r in sorted(clumps, key=lambda q: q[0][2]):
        k = rnd.random()
        m.ell(c, [r, r, r*flat], colors[0] if k < .5 else colors[1] if k < .8 else colors[2])


def limbs(rnd, base, n, length, rise, r, bark, jitter=.4):
    """n limbs from base spreading round it; returns their tips."""
    tips = []
    for k in range(n):
        a = k*math.tau/n + rnd.uniform(-jitter, jitter)
        tip = base + V(length*math.cos(a), length*math.sin(a), rnd.uniform(*rise))
        m.rod(base, tip, r, bark)
        tips.append(tip)
    return tips


def blossom_tree(seed=0, colors=(P.BLOSSOM, P.BLOSSOM, P.BLOSSOM_DARK), leaf=PEACH_LEAF):
    """A fruit tree in full blossom (peach, plum), about twice a person's height: a crooked trunk forking low into
    three or four limbs, a wide crown of blossom clumps, a few new leaves."""
    rnd = np.random.default_rng(seed + 11)
    m.meshes = []
    lean = rnd.uniform(-.25, .25, 2)
    fork = V(lean[0], lean[1], 1.5)
    m.rod(V(0, 0, 0), fork, .17, P.BARK)
    tips = limbs(rnd, fork, 3 + seed % 2, 1.1, (1.0, 1.5), .1, P.BARK)
    crown(rnd, tips + [fork + V(0, 0, 1.9)], 10, .7, .26, .4, colors)
    for _ in range(7):
        m.ell(tips[rnd.integers(len(tips))] + V(*rnd.uniform(-.6, .6, 2), rnd.uniform(-.4, 0)), [.13, .13, .09], leaf)
    return draw(**BIG)


def peach_tree(seed=0):
    return blossom_tree(seed)


def plum_tree(seed=0):
    """매화: the garden's plum, white blossom with a blush."""
    return blossom_tree(seed + 40, (PLUM_WHITE, PLUM_WHITE, P.BLOSSOM), leaf=P.BLOSSOM_DARK)


def pine(seed=0):
    """큰 소나무, three and a half people tall: a red-barked trunk leaning and bending, bare low down, flat pads of
    dark needles on side branches up the top half."""
    rnd = np.random.default_rng(seed + 21)
    m.meshes = []
    pts = [V(0, 0, 0)]
    for z in (1.6, 3.2, 4.8, 6.2):
        q = pts[-1]
        pts.append(V(q[0] + rnd.uniform(-.35, .35), q[1] + rnd.uniform(-.35, .35), z))
    for i, (a, b) in enumerate(zip(pts, pts[1:])):
        m.rod(a, b, .2 - .035*i, PINE_BARK)
    tiers = (2.8, 3.5, 4.2, 4.9, 5.6, 6.3)
    for i, z in enumerate(tiers):
        q = pts[min(4, 2 + i//2)]
        for side in range(2 if i < 4 else 1):
            a = i*2.3 + side*math.pi + rnd.uniform(-.6, .6)
            L = max(0, 1.5 - .22*i) * rnd.uniform(.7, 1)
            tip = V(q[0] + L*math.cos(a), q[1] + L*math.sin(a), z + rnd.uniform(-.15, .15))
            if L:
                m.rod(V(q[0], q[1], z - .35), tip, .07, PINE_BARK)
            for _ in range(5):
                c = tip + V(*rnd.uniform(-.55, .55, 2), rnd.uniform(-.05, .15))
                r = rnd.uniform(.4, .6) * (1 - .07*i)
                m.ell(c, [r, r, r*.32], PINE_LEAF if rnd.random() < .7 else S.FLOOR)
    return draw(**BIG)


def broadleaf(seed=0):
    """활엽수, two and a half people tall: a straight trunk, a few limbs, a full round crown."""
    rnd = np.random.default_rng(seed + 31)
    m.meshes = []
    fork = V(*rnd.uniform(-.15, .15, 2), 2.0)
    m.rod(V(0, 0, 0), fork, .2, P.BARK)
    tips = limbs(rnd, fork, 4, .8, (.9, 1.4), .1, P.BARK)
    crown(rnd, tips + [fork + V(0, 0, 2.2)], 9, .75, .4, .6, (BROAD, BROAD_LIGHT, BROAD_DARK), flat=.85)
    return draw(**BIG)


def willow(seed=0):
    """버드나무: a short leaning trunk, a small crown, long pale-green strands hanging nearly to the ground."""
    rnd = np.random.default_rng(seed + 51)
    m.meshes = []
    top = V(.3, -.1, 2.6)
    m.rod(V(0, 0, 0), top, .2, P.BARK)
    tips = limbs(rnd, top, 5, .9, (.6, 1.0), .07, P.BARK)
    crown(rnd, tips, 4, .4, .3, .45, (BROAD_LIGHT, BROAD, BROAD_LIGHT))
    for _ in range(46):  # hanging strands
        t = tips[rnd.integers(len(tips))] + V(*rnd.uniform(-.6, .6, 2), 0)
        lo = rnd.uniform(.5, 1.6)
        m.box(V(t[0], t[1], (t[2] + lo)/2), V(.07, .07, t[2] - lo), BROAD_LIGHT if rnd.random() < .6 else BROAD)
    return draw(**BIG)


def old_tree(seed=0):
    """늙은 고목 (a village's old zelkova): a thick gnarled trunk with buttress roots, wide crooked limbs, a broad
    but broken crown."""
    rnd = np.random.default_rng(seed + 61)
    m.meshes = []
    m.ell(V(0, 0, .5), [.55, .5, .7], OLD_BARK)
    for k in range(5):  # roots
        a = k*math.tau/5 + .3
        m.rod(V(0, 0, .3), V(.75*math.cos(a), .75*math.sin(a), 0), .14, OLD_BARK)
    fork = V(.1, .05, 1.8)
    m.rod(V(0, 0, .5), fork, .4, OLD_BARK)
    tips = limbs(rnd, fork, 5, 1.6, (.7, 1.3), .14, OLD_BARK, jitter=.5)
    keep = [t for i, t in enumerate(tips) if i != 2]  # one limb dead and bare
    crown(rnd, keep, 8, .6, .38, .55, (BROAD_DARK, BROAD, BROAD_DARK), flat=.75)
    m.rod(tips[2], tips[2] + V(.4, .2, .5), .06, OLD_BARK)
    return draw(**BIG)


def dead_tree(seed=0):
    """A tree killed by fire: a blackened trunk and bare broken limbs."""
    rnd = np.random.default_rng(seed + 71)
    m.meshes = []
    fork = V(.1, 0, 2.2)
    m.rod(V(0, 0, 0), fork, .2, CHAR)
    for t in limbs(rnd, fork, 4, 1.0, (.6, 1.4), .07, CHAR):
        m.rod(t, t + V(*rnd.uniform(-.4, .4, 2), .4), .04, CHAR)
    m.rod(fork, fork + V(0, 0, 1.2), .1, CHAR)
    return draw(**BIG)


def bush(seed=0):
    """덤불: a low clump of shrubs, knee to waist high."""
    rnd = np.random.default_rng(seed + 81)
    m.meshes = []
    crown(rnd, [V(0, 0, .3)], 9, .45, .2, .32, (BROAD, BROAD_DARK, BROAD_LIGHT), flat=.85)
    return draw()


def fallen_petals(seed=0):
    """Pink petals fallen on the ground in loose drifts (drawn under the units)."""
    rnd = np.random.default_rng(seed + 5)
    m.meshes = []
    for _ in range(40):
        x, y = rnd.uniform(-.8, .8, 2)
        if x*x + y*y > .64:
            continue
        m.box(V(x, y, .01), V(.1, .09, .02), P.BLOSSOM if rnd.random() < .7 else P.BLOSSOM_DARK)
    return draw()


def awning(L, W, h_back, h_front, cloth, stripe):
    """A cloth awning over a stall: four poles, the cloth sloping down to the front (-y), striped."""
    for sx in (-1, 1):
        m.box(V(sx*L/2, W/2, h_back/2), V(.06, .06, h_back), P.PLANK_DARK)
        m.box(V(sx*L/2, -W/2, h_front/2), V(.06, .06, h_front), P.PLANK_DARK)
    n = 6
    for i in range(n):
        x0, x1 = -L/2 - .1 + (L + .2)*i/n, -L/2 - .1 + (L + .2)*(i + 1)/n
        m.poly([V(x0, W/2 + .1, h_back + .02), V(x1, W/2 + .1, h_back + .02), V(x1, -W/2 + .05, h_front - .04),
                V(x0, -W/2 + .05, h_front - .04)], cloth if i % 2 else stripe)
    for i in range(n):  # the scalloped front edge
        x0, x1 = -L/2 - .1 + (L + .2)*i/n, -L/2 - .1 + (L + .2)*(i + 1)/n
        m.poly([V(x0, -W/2 + .05, h_front - .04), V(x1, -W/2 + .05, h_front - .04), V(x1, -W/2 + .04, h_front - .18),
                V(x0, -W/2 + .04, h_front - .18)], cloth if i % 2 else stripe)


def basket(x, y, z, r, goods, rnd):
    """A shallow wicker basket heaped with round goods."""
    bowl((x, y, z + .12), r, .12, STRAW_DARK)
    disc((x, y, z + .11), (x, y, z + .14), r, STRAW, ends=False)
    for _ in range(5):
        dx, dy = rnd.uniform(-r*.5, r*.5, 2)
        m.ell(V(x + dx, y + dy, z + .14), [.07, .07, .06], goods[rnd.integers(len(goods))])


def market_stall(kind='veg'):
    """장터 노점: a plank counter on trestles under a striped awning, baskets of greens and fruit (veg), bolts of
    cloth (cloth) or jars (jar) on it, a jar or two at its foot."""
    rnd = np.random.default_rng({'veg': 1, 'cloth': 2, 'jar': 3}[kind])
    m.meshes = []
    L, W = 1.5, .7
    m.box(V(0, 0, .62), V(L, W, .07), P.PLANK)  # counter
    for sx in (-1, 1):
        m.box(V(sx*(L/2 - .12), 0, .3), V(.07, W - .1, .58), P.PLANK_DARK)  # trestles
    m.box(V(0, -W/2 + .02, .45), V(L - .1, .03, .3), P.PLANK_DARK)  # front skirt
    cloth, stripe = {'veg': (CANVAS_LIGHT, CLOTH_BLUE), 'cloth': (CANVAS_LIGHT, RED), 'jar': (CANVAS_LIGHT, CLOTH_DARK)}[kind]
    awning(L + .1, W + .1, 2.25, 1.95, cloth, stripe)
    if kind == 'veg':
        for x, goods in ((-.45, (CABBAGE, PEACH_LEAF)), (0, (PERSIMMON, RED)), (.45, (RADISH, CABBAGE))):
            basket(x, .02, .65, .2, goods, rnd)
    elif kind == 'cloth':
        for i, (x, c) in enumerate(((-.5, CLOTH_BLUE), (-.2, RED), (.1, LINEN), (.4, CLOTH_DARK))):
            m.rod(V(x, -.22, .72), V(x, .22, .72), .07, c)
        m.rod(V(-.35, -.05, .86), V(-.35, .25, .86), .065, P.BANNER_DARK)
    else:
        for x in (-.5, -.15, .2, .5):
            m.ell(V(x, .05, .78), [.1, .1, .13], CLAY if x < 0 else CLAY_DARK)
            disc((x, .05, .89), (x, .05, .94), .05, CLAY_DARK)
    m.ell(V(.55, -.5, .2), [.15, .15, .2], CLAY)  # a jar at its foot
    m.ell(V(-.6, -.45, .12), [.16, .13, .12], HEMP)  # a sack
    return draw(cell=(120, 140), pivot=(60, 110))


def soup_stove():
    """국밥 솥: a clay stove with its fire mouth to the front, a big iron pot of broth set in it, the wooden lid
    pushed half off, a ladle on the rim. The steam is the scene's steam effect."""
    m.meshes = []
    m.box(V(0, 0, .3), V(.8, .8, .6), CLAY)
    m.box(V(0, 0, .61), V(.84, .84, .04), CLAY_DARK)
    m.box(V(0, -.405, .2), V(.3, .02, .24), CHAR)  # fire mouth
    m.box(V(0, -.41, .14), V(.22, .02, .1), FLAME)
    m.box(V(.25, .25, .9), V(.14, .14, .6), CLAY_DARK)  # flue at the back corner
    bowl((0, 0, .78), .34, .3, IRON)
    disc((0, 0, .76), (0, 0, .8), .35, IRON_RIM, ends=False)
    disc((0, 0, .72), (0, 0, .74), .31, SOUP)
    disc((.12, -.05, .8), (.12, -.05, .83), .26, P.PLANK)  # lid, half off
    m.box(V(.12, -.05, .86), V(.16, .04, .04), P.PLANK_DARK)
    m.rod(V(-.2, .05, .78), V(-.45, .2, 1.05), .02, P.PLANK_DARK)  # ladle handle
    m.ell(V(-.2, .05, .76), [.07, .07, .04], IRON_RIM)
    m.ell(V(.55, -.2, .12), [.13, .13, .12], P.PLANK)  # a stack of bowls beside
    disc((-.55, -.25, 0), (-.55, -.25, .35), .14, P.PLANK)  # firewood stump
    return draw()


def bench():
    """A long plank bench on four legs, along u."""
    m.meshes = []
    m.box(V(0, 0, .42), V(1.3, .3, .06), P.PLANK)
    for sx in (-1, 1):
        for sy in (-1, 1):
            m.box(V(sx*.55, sy*.1, .2), V(.06, .06, .4), P.PLANK_DARK)
    return draw()


def counter():
    """A tavern counter along u: a plank top on a boarded front, wine jars and cups on it."""
    m.meshes = []
    m.box(V(0, 0, .45), V(1.8, .5, .9), P.PLANK_DARK)
    m.box(V(0, 0, .92), V(1.9, .6, .06), P.PLANK)
    for x in np.linspace(-.8, .8, 6):
        m.box(V(x, -.252, .45), V(.03, .01, .85), P.PLANK)
    for x in (-.6, .1):
        m.ell(V(x, .05, 1.15), [.15, .15, .2], CLAY)
        disc((x, .05, 1.32), (x, .05, 1.38), .07, CLAY_DARK)
        m.ell(V(x, .05, 1.4), [.09, .09, .04], HEMP)
    for x in (-.25, .4, .55):
        disc((x, -.1, .95), (x, -.1, 1.03), .05, LINEN)
    return draw(cell=(120, 140), pivot=(60, 110))


PLASTER, TILE_ROOF, TILE_ROOF_DARK, SCORCH = S.PLASTER, S.TILE, S.TILE_DARK, '#6c583c'
R.add_ramps({SCORCH: 'fUUT'})


def turned(f):
    """The same model turned a quarter round: what faced -y (the lower left) faces +x (the lower right)."""
    def g():
        f(draw_it=False)
        m.meshes = [(np.array([[-p[1], p[0], p[2]] for p in pts]), c) for pts, c in m.meshes]
        return draw(cell=(160, 170), pivot=(80, 130))
    return g


def house(kind='tile', draw_it=True):
    """A town house at the people's scale: plaster walls on a timber frame and sill, a door and two lattice
    windows on the long wall facing the viewer (-y), a window on the gable end (+x), a tiled (tile) or thatched
    (thatch) gable roof along x. burnt: the same house after a fire, the walls blackened, the roof fallen in to a
    few charred rafters."""
    m.meshes = []
    w, d, wh, rh, ov = 2.2, 1.5, 1.45, .85, .14
    burnt = kind == 'burnt'
    wall, frame = (SCORCH, CHAR) if burnt else (PLASTER, P.PLANK_DARK)
    if burnt:  # four broken wall slabs round an open, ash-filled shell
        for i, x in enumerate(np.linspace(-w/2 + .2, w/2 - .2, 6)):
            for y, t in ((-d/2, (.9, 1.3, .6, 1.1, .8, 1.25)[i]), (d/2, (1.2, .7, 1.3, 1.0, 1.35, .9)[i])):
                m.box(V(x, y, t/2), V(w/6 + .01, .06, t), wall)
        for i, y in enumerate(np.linspace(-d/2 + .2, d/2 - .2, 4)):
            for x, t in ((-w/2, (1.1, 1.4, 1.2, .9)[i]), (w/2, (.7, 1.2, .8, 1.3)[i])):
                m.box(V(x, y, t/2), V(.06, d/4 + .01, t), wall)
        m.box(V(0, 0, .02), V(w, d, .04), ASH)
    else:
        m.box(V(0, 0, wh/2), V(w, d, wh), wall)
    m.box(V(0, 0, .05), V(w + .06, d + .06, .1), frame)  # sill
    for sx in (-1, 1):
        for sy in (-1, 1):
            m.box(V(sx*w/2, sy*d/2, wh/2), V(.09, .09, wh + .04), frame)
    if not burnt:
        m.box(V(0, -d/2, wh - .05), V(w, .06, .08), frame)  # lintel beam
    m.box(V(-.35, -d/2 - .01, .55), V(.42, .02, 1.0), CHAR if burnt else P.PLANK)  # door
    m.box(V(-.35, -d/2 - .02, .55), V(.03, .02, 1.0), CHAR)
    for x in (.35, .8):  # lattice windows
        m.box(V(x, -d/2 - .01, .95), V(.3, .02, .26), CHAR if burnt else P.INK)
        if not burnt:
            m.box(V(x, -d/2 - .02, .95), V(.3, .01, .03), P.PLANK)
            m.box(V(x, -d/2 - .02, .95), V(.03, .01, .26), P.PLANK)
    m.box(V(w/2 + .01, 0, .95), V(.02, .3, .26), P.INK)
    z0, z1 = wh - .03, wh + rh
    x0, x1, y0, y1 = -w/2 - ov, w/2 + ov, -d/2 - ov, d/2 + ov
    if burnt:
        for x in np.linspace(x0 + .2, x1 - .2, 5):  # charred rafters, a few fallen askew
            k = int(x*7) % 3
            m.rod(V(x, y0 + .1, z0 - (.3 if k == 0 else 0)), V(x + .05*k, 0, z1 - (.5 if k == 1 else .05)), .04, CHAR)
        m.rod(V(x0 + .3, 0, z1 - .1), V(x1 - .6, 0, z1 - .35), .05, CHAR)
        m.ell(V(.3, 0, .1), [.6, .45, .12], ASH)  # ash heap inside
        return draw(cell=(160, 170), pivot=(80, 130)) if draw_it else None
    roof, side = (TILE_ROOF, TILE_ROOF_DARK) if kind == 'tile' else (S.THATCH, S.THATCH_DARK)
    m.poly([V(x0, y0, z0), V(x1, y0, z0), V(x1, 0, z1), V(x0, 0, z1)], roof)
    m.poly([V(x0, y1, z0), V(x0, 0, z1), V(x1, 0, z1), V(x1, y1, z0)], side)
    for x in (-w/2, w/2):
        m.poly([V(x, -d/2, wh), V(x, 0, z1 - .02), V(x, d/2, wh)], wall)
    if kind == 'tile':
        for x in np.linspace(x0 + .12, x1 - .12, 9):  # tile rows down the front slope
            m.poly([V(x - .02, y0 - .005, z0 + .005), V(x + .02, y0 - .005, z0 + .005), V(x + .02, -.02, z1 - .01),
                    V(x - .02, -.02, z1 - .01)], TILE_ROOF_DARK)
        m.box(V(0, 0, z1 + .03), V(w + 2*ov + .1, .14, .1), TILE_ROOF_DARK)  # ridge
        for sx in (-1, 1):
            m.box(V(sx*(w/2 + ov + .05), 0, z1 + .1), V(.1, .12, .12), TILE_ROOF_DARK)
    else:
        for x in np.linspace(x0 + .15, x1 - .15, 8):  # bundles of straw down the front slope
            m.poly([V(x - .025, y0 - .005, z0 + .005), V(x + .025, y0 - .005, z0 + .005), V(x + .025, -.02, z1 - .01),
                    V(x - .025, -.02, z1 - .01)], S.THATCH_DARK)
        m.rod(V(x0, 0, z1 + .03), V(x1, 0, z1 + .03), .06, P.PLANK_DARK)
    return draw(cell=(160, 170), pivot=(80, 130)) if draw_it else None


def flagstones(seed=0):
    """A patch of worn flat paving slabs set in the earth, gaps between them (drawn under the units): where a
    paved way frays into a dirt lane, a yard's stepping stones."""
    rnd = np.random.default_rng(seed + 91)
    m.meshes = []
    for _ in range(11):
        x, y = rnd.uniform(-.8, .8, 2)
        if x*x/.7 + y*y/.7 > 1:
            continue
        w, d = rnd.uniform(.22, .38, 2)
        m.box(V(x, y, .015), V(w, d, .03), STONE_GREY if rnd.random() < .6 else S.STONE)
    return draw()


def well():
    """우물: a round stone well curb, a wooden frame with a pulley beam and a rope bucket."""
    m.meshes = []
    disc((0, 0, 0), (0, 0, .5), .42, STONE_GREY, n=16)
    disc((0, 0, .5), (0, 0, .56), .46, S.STONE, n=16, ends=False)
    disc((0, 0, .44), (0, 0, .46), .34, P.INK, n=16)
    for x in (-.4, .4):
        m.box(V(x, 0, .9), V(.08, .08, 1.0), P.PLANK_DARK)
    m.rod(V(-.45, 0, 1.35), V(.45, 0, 1.35), .05, P.PLANK)
    m.rod(V(0, 0, 1.33), V(0, 0, .75), .012, P.ROPE)
    disc((0, 0, .62), (0, 0, .78), .1, P.PLANK)
    return draw()


PROPS = {'weapon_rack': weapon_rack, 'armor_stand': armor_stand, 'candle_stand': candle_stand,
         'antler_barrier': antler_barrier, 'watchtower': watchtower, 'shield_wall': shield_wall,
         'straw_mat': straw_mat, 'bloody_cloth': bloody_cloth, 'medicine_pot': medicine_pot,
         'broken_spear': broken_spear, 'fallen_flag': fallen_flag, 'cauldron': cauldron,
         'grain_sack': grain_sack, 'wine_jar': wine_jar,
         'prison_cart': prison_cart, 'grain_cart': grain_cart, 'grain_cart_v': grain_cart_v,
         'command_tent': command_tent, 'water_bucket': water_bucket, 'water_jar': water_jar, 'campfire': campfire,
         'notice_post': notice_post, 'peach_bloom_0': lambda: peach_tree(0), 'peach_bloom_1': lambda: peach_tree(1),
         'peach_bloom_2': lambda: peach_tree(2),
         'stree_pine_0': pine, 'stree_pine_1': lambda: pine(1), 'stree_broad_0': broadleaf, 'stree_broad_1': lambda: broadleaf(1),
         'stree_willow': willow, 'stree_old': old_tree, 'stree_dead': dead_tree, 'stree_bush_0': bush, 'stree_bush_1': lambda: bush(1),
         'stree_plum_0': plum_tree, 'stree_plum_1': lambda: plum_tree(1),
         'flagstones_0': flagstones, 'flagstones_1': lambda: flagstones(1), 'well': well, 'fallen_petals': fallen_petals,
         'market_stall': market_stall, 'market_stall_cloth': lambda: market_stall('cloth'),
         'market_stall_jar': lambda: market_stall('jar'), 'soup_stove': soup_stove, 'bench': bench, 'counter': counter,
         'house_tile': house, 'house_thatch': lambda: house('thatch'), 'house_burnt': lambda: house('burnt'),
         'house_tile_x': turned(lambda draw_it: house('tile', draw_it)), 'house_thatch_x': turned(lambda draw_it: house('thatch', draw_it))}
GROUND = {'bloody_cloth', 'broken_spear', 'fallen_flag', 'straw_mat', 'fallen_petals', 'flagstones_0', 'flagstones_1'}  # drawn under the units

# Hand retouches after the render, in trimmed-image pixels: (x, y, palette key). Negative x, y count from the
# right / bottom edge.
RETOUCH = {
    # the wanted face on the left notice (eyes, beard) and lines of writing on the right one
    'notice_post': [(9, 20, '2'), (12, 21, '2'), (10, 23, 'f'), (11, 24, 'f'), (10, 24, 'f'),
                    (17, 22, '3'), (17, 24, '3'), (19, 23, '3'), (19, 25, '3'), (17, 26, '3')],
}


def retouched(name, im):
    for x, y, key in RETOUCH.get(name, ()):
        im.putpixel((x % im.width, y % im.height), (*R.COLOR[key], 255) if key != '.' else (0, 0, 0, 0))
    return im


def made():
    return {name: (retouched(name, im), pivot) for name, (im, pivot) in ((n, f()) for n, f in PROPS.items())}


def review(props, out=None):
    """Old picture above, the 3D one below, each prop in a column, on grass at 4x, a standing officer (the
    scene people's scale, 38 px) at the left. A new prop has no old picture and leaves its top empty."""
    tiles = json.load(open(CAND / 'tiles/tiles.json'))['sprites']
    blank = Image.new('RGBA', (1, 1))
    cols = [(Image.open(CAND / tiles[n]['file']).convert('RGBA') if n in tiles else blank, im) for n, (im, _) in props.items()]
    man = Image.new('RGBA', (12, 38), (40, 60, 110, 255))
    cols.insert(0, (blank, man))
    w = sum(max(a.width, b.width) + 8 for a, b in cols) + 8
    top = max(a.height for a, _ in cols) + 8
    h = top + max(b.height for _, b in cols) + 8
    sheet = Image.new('RGBA', (w, h), (70, 90, 60, 255))
    x = 8
    for old, new in cols:
        sheet.alpha_composite(old, (x, top - 4 - old.height))
        sheet.alpha_composite(new, (x, h - 4 - new.height))
        x += max(old.width, new.width) + 8
    out = Path(out) if out else ROOT / 'output/scene-props3d.png'
    out.parent.mkdir(parents=True, exist_ok=True)
    sheet.resize((w*4, h*4), Image.NEAREST).save(out)
    print(out)


def main():
    names = [a for a in sys.argv[1:] if not a.startswith('--')]
    props = made()
    if names:
        props = {n: props[n] for n in names}
    if '--review' in sys.argv:
        out = next((a.split('=', 1)[1] for a in sys.argv if a.startswith('--out=')), None)
        review(props, out)
        return
    painted = json.load(open(CAND / 'tiles/sources/trees.json'))['trees']
    refs = {n: props.pop(n) for n in list(props) if n in painted}
    if '--3d' in sys.argv:  # a painted tree's render, as a new reference to paint it from, never over its sprite
        out = ROOT / 'output/scene-props3d'
        out.mkdir(parents=True, exist_ok=True)
        for name, (im, _) in refs.items():
            im.save(out / f'{name}.png')
            print(out / f'{name}.png', im.size)
    elif refs:
        print('painted, kept (--3d renders them to output/scene-props3d/):', ' '.join(refs))
    if not props:
        return
    path = CAND / 'tiles/tiles.json'
    tiles = json.load(open(path))
    for name, (im, pivot) in props.items():
        im.save(CAND / 'tiles' / f'{name}.png')
        layer = 'ground' if name in GROUND else 'object'
        tiles['sprites'][name] = {'file': f'tiles/{name}.png', 'size': list(im.size), 'pivot': pivot, 'layer': layer}
        print(name, im.size)
    json.dump(tiles, open(path, 'w'), ensure_ascii=False, indent=1)
    open(path, 'a').write('\n')


if __name__ == '__main__':
    main()
