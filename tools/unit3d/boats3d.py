#!/usr/bin/env python3
"""The boat soldiers cross water in on a battle map with boats (docs/plan-naval-water.md): 들린 배, an open wooden
boat whose rim sweeps up to both ends, a post at bow and stern.

Same camera and cel renderer as props.py (built in cell units, a cell spans [-.5, .5], widened by props.stretch to
the characters' scale where a cell is 64 x 32; heights absolute, an officer about 1.8 units). Two sizes: "boat" for
soldiers on foot and "boat_l", long enough for a horse. Each along map u and v, and each in two layers: the whole
boat and its front rim only (the near long side and the near end), so the game draws whole boat -> soldier -> rim
and the soldier stands inside the hull. Sprites are trimmed; the pivot is the centre of the boat's floor.

  python3 tools/unit3d/boats3d.py            -> assets_candidate/tiles/boat*.png and their tiles.json entries
  python3 tools/unit3d/boats3d.py --review   -> output/boats3d.png (4x, with soldiers aboard)
then python3 ver5/tools/build_extra.py packs them into the game's extra sheet.
"""
import json
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
HULL, HULL_DARK, PALE = '#8a5a34', '#5a3720', '#b98a55'
R.add_ramps({HULL: 'fggh', HULL_DARK: 'ffgg', PALE: 'ghhi'})
CELL, PIVOT = (140, 100), (69, 62)
SIZES = {'boat': (.7, .54), 'boat_l': (.98, .6)}  # length, width in cell units


def quad(a, b, c, d, color):
    m.poly([V(*a), V(*b), V(*c), V(*d)], color)


def boat(L, W, h=.34, rake=.1, lift=.22):
    """(vertices, colour, front) of an open hull along x: floor, cross boards, walls that flare out and a rim that
    rises to both ends, posts at bow and stern. Front pieces lie between a soldier at the centre and the camera
    (seen from SW: the -y long side and the +x end)."""
    items = []

    def add(front, fn, *a):
        m.meshes = []
        fn(*a)
        items.extend((v, c, front) for v, c in m.meshes)

    x0, x1, y0, y1 = -L/2, L/2, -W/2, W/2
    X0, X1, Y0, Y1 = x0 - rake, x1 + rake, y0 - .05, y1 + .05
    z0 = .02
    rim = lambda x: h + lift*(abs(x)/X1)**3  # noqa: E731
    add(False, quad, (x0, y0, z0), (x1, y0, z0), (x1, y1, z0), (x0, y1, z0), PALE)
    for k in range(1, 4):
        add(False, m.box, V(x0 + k*L/4, 0, z0 + .03), V(.035, W - .02, .04), P.PLANK_DARK)
    n = 6
    for front, yb, yt in ((True, y0, Y0), (False, y1, Y1)):  # long sides in six pieces so the rim can curve
        for i in range(n):
            bx0, bx1 = x0 + L*i/n, x0 + L*(i + 1)/n
            tx0, tx1 = X0 + (X1 - X0)*i/n, X0 + (X1 - X0)*(i + 1)/n
            add(front, quad, (bx0, yb, z0), (bx1, yb, z0), (tx1, yt, rim(tx1)), (tx0, yt, rim(tx0)), HULL)
            add(front, m.box, V((tx0 + tx1)/2, yt, (rim(tx0) + rim(tx1))/2 + .02), V(abs(tx1 - tx0) + .01, .04, .05),
                P.PLANK_DARK)
    for front, xb, xt in ((True, x1, X1), (False, x0, X0)):  # ends
        add(front, quad, (xb, y0, z0), (xb, y1, z0), (xt, Y1, rim(xt)), (xt, Y0, rim(xt)), HULL)
        add(front, m.box, V(xt, 0, rim(xt) + .02), V(.04, Y1 - Y0 + .04, .05), P.PLANK_DARK)
    add(True, m.box, V(X1 + .01, 0, rim(X1) + .1), V(.05, .05, .2), HULL_DARK)  # bow post
    add(False, m.box, V(X0 - .01, 0, rim(X0) + .08), V(.05, .05, .16), HULL_DARK)  # stern post
    return items


def turned(items, along):
    """Along v: mirrored across the diagonal, so the near sides stay near."""
    if along == 'u':
        return items
    return [(np.asarray(v)[:, [1, 0, 2]]*[-1, -1, 1], c, f) for v, c, f in items]


def render(items):
    m.meshes = [(v, c) for v, c, _ in items]
    P.stretch()
    return P.draw(CELL, PIVOT)


def layers(size, along):
    """Whole boat and front rim, trimmed alike, and the pivot (floor centre) in them."""
    items = turned(boat(*SIZES[size]), along)
    whole, front_only = np.array(render(items)), np.array(render([it for it in items if it[2]]))
    mask = front_only[..., 3] > 0
    ink = front_only[..., :3].sum(-1) < 60
    mask &= ~ink | (whole[..., :3].sum(-1) < 60)  # the rim's outline only where the whole boat has one
    rim = whole.copy()
    rim[~mask] = 0
    whole, rim = Image.fromarray(whole), Image.fromarray(rim)
    x0, y0, x1, y1 = whole.getbbox()
    pivot = [PIVOT[0] + 1 - x0, PIVOT[1] + 1 - y0]
    return whole.crop((x0, y0, x1, y1)), rim.crop((x0, y0, x1, y1)), pivot


def sprites():
    out = {}
    for size in SIZES:
        for along in 'uv':
            whole, rim, pivot = layers(size, along)
            out[f'{size}_{along}'] = (whole, pivot, {'layer': 'object'})
            out[f'{size}_{along}_rim'] = (rim, pivot, {'layer': 'object', 'shadow': False})
    return out


def review(path):
    sys.path.insert(0, str(ROOT / 'ver5/tools/spritebuild'))
    import shadows
    sp = sprites()
    pal = (36, 78, 104, 255)
    tiles = []
    for size, who in (('boat', 'light_infantry'), ('boat_l', 'light_cavalry')):
        for d in ('SE', 'SW', 'NW', 'NE'):
            along = 'u' if d in ('SE', 'NW') else 'v'
            whole = sp[f'{size}_{along}'][0]
            rim = sp[f'{size}_{along}_rim'][0]
            t = Image.new('RGBA', (140, 100), pal)
            fig = Image.new('RGBA', (140, 100))
            pivot = [70, 63]
            fig.alpha_composite(whole, (70 - sp[f'{size}_{along}'][1][0], 63 - sp[f'{size}_{along}'][1][1]))
            unit, upivot = unit_frame(who, d)
            fig.alpha_composite(unit, (pivot[0] - upivot[0], pivot[1] - 3 - upivot[1]))
            fig.alpha_composite(rim, (70 - sp[f'{size}_{along}'][1][0], 63 - sp[f'{size}_{along}'][1][1]))
            s, spv = shadows.shadow(fig, pivot)
            t.alpha_composite(s, (70 - spv[0], 63 - spv[1]))
            t.alpha_composite(fig, (70 - pivot[0], 63 - pivot[1]))
            tiles.append(t)
    sheet = Image.new('RGBA', (140*4, 100*2))
    for i, t in enumerate(tiles):
        sheet.paste(t, ((i % 4)*140, (i // 4)*100))
    sheet.resize((140*16, 100*8), Image.NEAREST).save(path)
    print(path)


def unit_frame(name, direction):
    """The idle frame of a built .spr sheet with default colours (docs/presentation-sprite.md)."""
    import struct
    import zlib
    b = zlib.decompress((ROOT / f'ver5/assets/sprites/{name}.spr').read_bytes()[6:])
    o = 0

    def u(f):
        nonlocal o
        v = struct.unpack_from(f, b, o)
        o += struct.calcsize(f)
        return v

    def s():
        n = u('<H')[0]
        nonlocal o
        o += n
        return b[o - n:o].decode()

    s(), s()
    w, h, px, py = u('<HHHH')
    rows = [s() for _ in range(u('<B')[0])]
    anims = {}
    for _ in range(u('<B')[0]):
        name_ = s()
        first, frames, loop, k = u('<HHBB')
        u(f'<{k}I')
        anims[name_] = first
    tags = {}
    for _ in range(u('<B')[0]):
        idx = u('<B')[0]
        s()
        u('<B')
        tags[idx] = u('<4B')
    pal = [u('<4B') for _ in range(u('<H')[0])]
    cols = u('<H')[0]
    off = o + (rows.index(direction)*cols + anims['idle'])*2*w*h
    pix, tag = b[off:off + w*h], b[off + w*h:off + 2*w*h]
    im = Image.new('RGBA', (w, h))
    im.putdata([(0, 0, 0, 0) if p == 0 else (tags[t][:3] + (pal[p][3],) if t else pal[p]) for p, t in zip(pix, tag)])
    return im, (px, py)


def main():
    if '--review' in sys.argv:
        review(ROOT / 'output/boats3d.png')
        return
    path = CAND / 'tiles/tiles.json'
    doc = json.loads(path.read_text())
    for name, (im, pivot, extra) in sprites().items():
        im.save(CAND / f'tiles/{name}.png')
        doc['sprites'][name] = {'file': f'tiles/{name}.png', 'size': list(im.size), 'pivot': pivot, **extra,
                                'note': 'battle-map boat (tools/unit3d/boats3d.py)'}
    path.write_text(json.dumps(doc, indent=1, ensure_ascii=False) + '\n')


if __name__ == '__main__':
    main()
