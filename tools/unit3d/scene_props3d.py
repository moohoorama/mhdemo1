#!/usr/bin/env python3
"""ver5 story-scene props modelled in 3D: weapon rack, armour stand, candle stand, antler barrier, watchtower,
shield wall, straw mat, bloody cloth, medicine pot, broken spear, fallen flag, cauldron, grain sack and wine jar.

Same camera and cel renderer as structures.py (one model unit = one 32 x 16 scene-map cell, 30 degree elevation,
seen from SW: +x runs to the lower right along map u, +y to the upper right). Heights are at the scene people's
scale: an officer on foot is about 1.9 units (38 px). After the render, RETOUCH puts back by hand the few pixels a
render this small loses (a blade tip, a candle flame, a stain). Each sprite is trimmed to its pixels, its pivot is
the footprint centre on the ground.

  python3 tools/unit3d/scene_props3d.py            -> assets_candidate/tiles/<name>.png and their tiles.json entries
  python3 tools/unit3d/scene_props3d.py --review   -> output/scene-props3d.png (4x, with an officer for scale)
then python3 ver5/tools/build_extra.py packs them into the game's extra sheet.
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


PROPS = {'weapon_rack': weapon_rack, 'armor_stand': armor_stand, 'candle_stand': candle_stand,
         'antler_barrier': antler_barrier, 'watchtower': watchtower, 'shield_wall': shield_wall,
         'straw_mat': straw_mat, 'bloody_cloth': bloody_cloth, 'medicine_pot': medicine_pot,
         'broken_spear': broken_spear, 'fallen_flag': fallen_flag, 'cauldron': cauldron,
         'grain_sack': grain_sack, 'wine_jar': wine_jar}
GROUND = {'bloody_cloth', 'broken_spear', 'fallen_flag', 'straw_mat'}  # drawn under the units

# Hand retouches after the render, in trimmed-image pixels: (x, y, palette key). Negative x, y count from the
# right / bottom edge.
RETOUCH = {}


def retouched(name, im):
    for x, y, key in RETOUCH.get(name, ()):
        im.putpixel((x % im.width, y % im.height), (*R.COLOR[key], 255) if key != '.' else (0, 0, 0, 0))
    return im


def made():
    return {name: (retouched(name, im), pivot) for name, (im, pivot) in ((n, f()) for n, f in PROPS.items())}


def review(props):
    """Old picture above, the 3D one below, each prop in a column, on grass at 4x."""
    tiles = json.load(open(CAND / 'tiles/tiles.json'))['sprites']
    cols = [(Image.open(CAND / tiles[n]['file']).convert('RGBA'), im) for n, (im, _) in props.items()]
    w = sum(max(a.width, b.width) + 8 for a, b in cols) + 8
    top = max(a.height for a, _ in cols) + 8
    h = top + max(b.height for _, b in cols) + 8
    sheet = Image.new('RGBA', (w, h), (70, 90, 60, 255))
    x = 8
    for old, new in cols:
        sheet.alpha_composite(old, (x, top - 4 - old.height))
        sheet.alpha_composite(new, (x, h - 4 - new.height))
        x += max(old.width, new.width) + 8
    out = ROOT / 'output/scene-props3d.png'
    out.parent.mkdir(exist_ok=True)
    sheet.resize((w*4, h*4), Image.NEAREST).save(out)
    print(out)


def main():
    props = made()
    if '--review' in sys.argv:
        review(props)
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
