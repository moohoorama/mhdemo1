#!/usr/bin/env python3
"""ver5 battle-map town houses (도시 가옥): the 마을 site drawn inside a castle, one cell each, in the same
camera, scale and paving as sites.py (a cell is 64 x 32, units stand on it, so everything stays low).
Three variants share the castle's grey tile roofs, plastered walls and paving:

  town_a  기와집: a courtyard wall round the cell with a gateway on the front-left edge, a long hall along the back
  town_b  두 채: an L of two halls along both back edges, a short wall on the front-right edge
  town_c  가게: a shop along the back-right edge with an awning over its counter, a wall on the back-left, jars

The game draws them for a 마을 (v) that stands on castle floor (i); see plan/draft_maps/README.md.

  uv run --with pillow --with numpy python tools/unit3d/town_houses.py [out_dir]
    -> <out_dir>/sites.png and sites.json: the sites.py sheet with the town houses appended
       (the first five cells are the sites.py ones, pixel for pixel), and <out_dir>/town_review.png
"""
import json
import sys
from pathlib import Path

from PIL import Image

import base_model as m
import props as P
import sites as Si
import structures as S

V, BX, BY = Si.V, Si.BX, Si.BY
PLASTER, TILE, TILE_DARK = S.PLASTER, S.TILE, S.TILE_DARK
WOOD, LACQUER, DARK = S.WOOD, P.LACQUER, '#2a1d18'
WALL_H, COPING = .12, .03  # courtyard wall and its tile coping
HALL_H, ROOF_H = .3, .19
ALONG_Y = ('nw', 'se')  # these edges run along model y
SX, SY = (1 if BX > 0 else -1), (1 if BY > 0 else -1)  # toward the back edges 'nw' (x) and 'ne' (y)


def court_wall(side, spans=((-.5, .5),), inset=.035, h=WALL_H):
    """A plastered courtyard wall along `side` with a grey tile coping."""
    for t0, t1 in spans:
        mid, ln = (t0 + t1)/2, t1 - t0
        size = (lambda a, c, z: V(c, a, z)) if side in ALONG_Y else (lambda a, c, z: V(a, c, z))
        m.box(Si.edge(side, mid, h/2, inset), size(ln, .05, h), PLASTER)
        m.box(Si.edge(side, mid, h + COPING/2, inset), size(ln + .02, .08, COPING), TILE)
        m.box(Si.edge(side, mid, h + COPING + .006, inset), size(ln + .02, .02, .012), TILE_DARK)


def gate_posts(side, s=0.0, half=.11, inset=.035):
    """Two lacquered posts and a little tile roof over a gateway at s on `side`."""
    size = (lambda a, c, z: V(c, a, z)) if side in ALONG_Y else (lambda a, c, z: V(a, c, z))
    for k in (-1, 1):
        m.box(Si.edge(side, s + k*half, (WALL_H + .05)/2, inset), size(.03, .05, WALL_H + .05), LACQUER)
    c = Si.edge(side, s, 0, inset)
    w, d = (.1, half*2 + .08) if side in ALONG_Y else (half*2 + .08, .1)
    Si.roof(c, w, d, WALL_H + .05, .07, TILE, TILE_DARK, .03)


def hall(side, length, depth=.22, inset=.17, s=0.0, h=HALL_H, roof_h=ROOF_H, doors=2):
    """A plastered hall along `side` (its long side parallel to the edge), lacquered posts, a hipped
    tile roof, dark doors and windows on the face toward the cell centre (the one the viewer sees)."""
    c = Si.edge(side, s, 0, inset)
    w, d = (depth, length) if side in ALONG_Y else (length, depth)
    m.box(V(c[0], c[1], .02), V(w + .04, d + .04, .04), S.STONE)
    m.box(V(c[0], c[1], h/2 + .03), V(w, d, h), PLASTER)
    for sx in (-1, 1):
        for sy in (-1, 1):
            m.box(V(c[0] + sx*w/2, c[1] + sy*d/2, h/2 + .03), V(.025, .025, h), LACQUER)
    Si.roof(V(c[0], c[1], 0), w, d, h + .03, roof_h, TILE, TILE_DARK, .035)
    # dark doors and windows on the face toward the cell centre, the one the viewer sees
    for i in range(doors):
        t = (i + .5)/doors - .5
        if side in ALONG_Y:
            p, door_size, win_size, off = V(c[0] - SX*(w/2 + .004), c[1] + t*d*.8, 0), V(.008, .07, h*.62), V(.008, .05, .045), V(0, .09, 0)
        else:
            p, door_size, win_size, off = V(c[0] + t*w*.8, c[1] - SY*(d/2 + .004), 0), V(.07, .008, h*.62), V(.05, .008, .045), V(.09, 0, 0)
        if i % 2 == 0:
            m.box(p + V(0, 0, .03 + h*.31), door_size, DARK)
        m.box(p + off*(.5 if i % 2 else 0) + V(0, 0, .03 + h*.6), win_size, DARK)


def jar(p, r=.035, h=.07):
    m.rod(p, p + V(0, 0, h), r, '#7a5a3c')
    m.rod(p + V(0, 0, h), p + V(0, 0, h + .012), r*.7, '#5a3f2a')


def town_a():
    """기와집: courtyard wall on all four edges, a gateway in the front-left one, a long hall at the back."""
    Si.brick_floor()
    for side in ('nw', 'ne', 'se'):
        court_wall(side)
    court_wall('sw', spans=((-.5, -.11), (.11, .5)))
    gate_posts('sw')
    hall('nw', .66, depth=.32, inset=.22, doors=3)
    jar(Si.corner('right', .16))


def town_b():
    """두 채: two halls in an L along both back edges, a short wall on the front-right edge."""
    Si.brick_floor()
    hall('nw', .5, depth=.28, inset=.19, s=.1, doors=2)
    hall('ne', .4, depth=.26, inset=.18, s=-.16, h=.26, roof_h=.17, doors=2)
    court_wall('se', spans=((-.5, -.05), (.2, .5)))
    court_wall('sw', spans=((.15, .5),))
    jar(Si.corner('left', .12))


def town_c():
    """가게: a shop along the back-right edge with an awning over its counter, a wall on the back-left, jars."""
    Si.brick_floor()
    court_wall('nw')
    hall('ne', .6, depth=.28, inset=.21, doors=2, h=.28)
    c = Si.edge('ne', 0, 0, .2)
    # awning on posts in front of the shop (toward the centre: -y for the ne edge in model space)
    ay = c[1] - (.28/2 + .08)*SY
    for k in (-.2, .2):
        m.rod(V(c[0] + k, ay, 0), V(c[0] + k, ay, .17), .012, WOOD)
    m.poly([V(c[0] - .25, ay - .03*SY, .16), V(c[0] + .25, ay - .03*SY, .16),
            V(c[0] + .25, c[1] - .14*SY, .22), V(c[0] - .25, c[1] - .14*SY, .22)], TILE)
    m.box(V(c[0], ay + .03*SY, .05), V(.36, .05, .1), WOOD)  # counter
    for k in (-.12, 0, .12):
        jar(V(c[0] + k, ay + .03*SY, .1), .022, .04)
    jar(Si.corner('front', .2) + V(.05, 0, 0))
    jar(Si.corner('front', .2) + V(-.03, .06, 0), .03, .06)


TOWN = {'town_a': town_a, 'town_b': town_b, 'town_c': town_c}


def render(make):
    m.meshes = []
    make()
    P.stretch()
    return Si.draw_bricked()


def main():
    out = Path(sys.argv[1]) if len(sys.argv) > 1 else Si.OUT.parent / 'ver5-town'
    out.mkdir(parents=True, exist_ok=True)
    allsites = {**Si.SITES, **TOWN}
    cw, ch = Si.CELL
    sheet = Image.new('RGBA', (cw*len(allsites), ch))
    for i, make in enumerate(allsites.values()):
        sheet.alpha_composite(render(make), (i*cw, 0))
    sheet.save(out / 'sites.png')
    meta = dict(version=1, image='sites.png', cell=list(Si.CELL), pivot=[Si.PIVOT[0] + 1, Si.PIVOT[1] + 1],
                pivot_note='cell centre of the 64x32 map diamond', sprites={k: i for i, k in enumerate(allsites)})
    (out / 'sites.json').write_text(json.dumps(meta, indent=1) + '\n')
    review = Image.new('RGBA', (cw*(2 + len(TOWN)), ch))  # village, castle, then the town houses
    for j, i in enumerate([1, 2] + list(range(len(Si.SITES), len(allsites)))):
        review.alpha_composite(sheet.crop((i*cw, 0, (i + 1)*cw, ch)), (j*cw, 0))
    review.resize((review.width*4, review.height*4), Image.NEAREST).save(out / 'town_review.png')
    print(out / 'sites.png')


if __name__ == '__main__':
    main()
