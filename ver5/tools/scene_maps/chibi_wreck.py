#!/usr/bin/env python3
"""Make S_chibi_wreck: S_chibi_fleet after the fire — the long chained deck in the middle broken into separate burnt
pieces with water between them, broken spears and fallen flags on them, the boats round about as they were.

  python3 ver5/tools/scene_maps/chibi_wreck.py          write the map, register it, format it
  python3 ver5/tools/scene_maps/chibi_wreck.py --show   also print the map one game cell per character

The fleet map (southern_routes.py chibi) is kept as it is; this one has the same far shore (cliffs, v < -4), the same
boats, and the water running 30 cells past the playing area on every side so no edge shows on a zoom-1 screen.
The deck pieces are whole game cells (a wooden floor is drawn per game cell), ragged at their corners, each with a
broken railing (sfence) along its outer small-tile rows: on a piece of rows v0..v1 the railing stands on the first
small tile of v0, so people stand on rows v0+1..v1.
"""
import json
import math
import random
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from southern_routes import SceneMap  # noqa: E402

GFX = Path(__file__).resolve().parents[2] / 'assets/graphics'
ID = 'S_chibi_wreck'
W, H = 24, 17
U0, U1, V0, V1 = -30, W + 30, -30, H + 30      # game cells the picture covers

# deck pieces: game cells u0..u1 × v0..v1 (inclusive), minus the cells listed (ragged corners)
PIECES = [
    ((1, 4, 8, 10), [(1, 8), (4, 10)]),           # the bow
    ((7, 11, 7, 10), [(7, 7), (11, 7), (11, 10)]),  # the middle of the ship, the largest
    ((13, 15, 8, 11), [(15, 8), (13, 11)]),       # a piece drifted a cell south-west
    ((18, 20, 7, 9), [(20, 9)]),                  # the stern
    ((8, 10, 13, 13), []),                        # a loose strip of planks
]
BOATS = [('boat_l_u', -1, 12), ('boat_u', 4, 27), ('boat_l_u', 29, 28), ('boat_l_u', 36, 15), ('boat_l_u', 44, 17),
         ('boat_l_u', 0, 2), ('boat_l_u', 7, 1), ('boat_l_v', 14, 4), ('boat_l_u', 20, 0), ('boat_l_u', 28, 3),
         ('boat_l_u', 36, 3), ('boat_l_u', 5, -4), ('boat_l_u', 17, -4), ('boat_l_u', 27, -3), ('boat_l_u', 38, -3),
         ('boat_v', 24, 9), ('boat_u', 33, 8)]      # S_chibi_fleet's boats, small tiles


def deck_cells():
    cells = set()
    for (u0, u1, v0, v1), cut in PIECES:
        cells |= {(u, v) for u in range(u0, u1 + 1) for v in range(v0, v1 + 1)} - set(cut)
    return cells


def build():
    rnd = random.Random(208)
    scene = SceneMap(ID, '적벽 · 부서진 선단', [W, H], [2 * U0, 2 * V0], [2 * (U1 - U0), 2 * (V1 - V0)],
                     {'hue': 31, 'sat': 0.64, 'val': 0.67})
    scene.terrain(lambda u, v: '~')
    for U in range(2 * U0, 2 * U1):
        edge = -9 + round(3 * math.sin(U / 8)) + (2 if U < 5 else 0)
        scene.patch(U, 2 * V0, U, edge, 'd', '2')
        scene.patch(U, edge + 1, U, edge + 1, 'd')

    deck = deck_cells()
    for u, v in deck:
        scene.patch(2 * u, 2 * v, 2 * u + 1, 2 * v + 1, 'W')

    # broken railings on the outer small-tile rows of the pieces, with gaps where they burnt through
    rail = set()
    for u, v in deck:
        if (u, v - 1) not in deck and (u, v + 1) not in deck:
            continue                      # a strip one cell deep floats bare
        for U in (2 * u, 2 * u + 1):
            if (u, v - 1) not in deck:
                rail.add((U, 2 * v))
            if (u, v + 1) not in deck:
                rail.add((U, 2 * v + 1))
    for U, V in sorted(rail):
        if rnd.random() < .22:
            rail.discard((U, V))
            rail.discard((U + 1, V))
    for U, V in sorted(rail):
        mask = (1 if (U - 1, V) in rail else 0) | (4 if (U + 1, V) in rail else 0)
        scene.props((f'sfence_{mask:02d}', U, V))

    scene.props(
        # what is left on the pieces (small tiles)
        ('broken_spear', 5, 18), ('fallen_flag', 6, 20),
        ('fallen_flag', 17, 16), ('broken_spear', 19, 19), ('grain_sack', 15, 18), ('lantern', 21, 17),
        ('broken_spear', 27, 18), ('water_jar', 29, 20),
        ('banner', 37, 16), ('broken_spear', 39, 17),
        # flotsam between them
        ('broken_spear', 12, 22), ('fallen_flag', 24, 24), ('broken_spear', 33, 22),
        *BOATS,
        ('stree_pine_0', 5, -20), ('stree_pine_1', 15, -22), ('rock_pile_0', 27, -20), ('stree_pine_0', 41, -20),
        ('stree_broad_0', -5, -17), ('stree_pine_1', 52, -21), ('stree_pine_0', -30, -24), ('stree_pine_1', -18, -30),
        ('stree_pine_0', 64, -26), ('stree_broad_1', 80, -22), ('stree_pine_1', 98, -28), ('rock_pile_1', -40, -20),
    )
    return scene


def register(relative):
    """Add the map to index.json's maps by text, so other sessions' edits of the file stay as they are."""
    path = GFX / 'index.json'
    text = path.read_text()
    if f'"{relative}"' in text:
        return
    end = text.index('\n ],', text.index(' "maps": ['))
    path.write_text(text[:end] + f',\n  "{relative}"' + text[end:])


def show(scene):
    blocked = {(o[1] // 2, o[2] // 2) for o in scene.objects}
    for v in range(-8, 26):
        line = []
        for u in range(-8, 40):
            g = scene.ground[2 * (v - V0)][2 * (u - U0)]
            line.append('#' if (u, v) in blocked else g)
        print(f'{v:4d} ' + ''.join(line))


if __name__ == '__main__':
    scene = build()
    target = scene.write()
    register(target)
    subprocess.run(['go', 'run', './cmd/_walkmap', '-fmt', ID], cwd=GFX.parents[1], check=True)
    print(ID, scene.meta['size'], scene.meta['origin'], len(scene.objects), 'objects')
    if '--show' in sys.argv:
        show(scene)
