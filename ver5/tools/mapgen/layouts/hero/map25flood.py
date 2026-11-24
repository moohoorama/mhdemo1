#!/usr/bin/env python3
"""Chapter 25 신야 전투 map: the ground the flood of assets/graphics/effects/flood/suigong was baked on
(internal/gui/uidemo_suigong.go suigongGround: the 6-wide river down the v axis, the gorge's cliffs above the
dam at row 8, the ford from row 30) with the burning city of Xinye on the east bank. Run from ver5/:

    python3 tools/mapgen/layouts/hero/map25flood.py
"""
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[4]
W, H = 48, 44
RIVER, WIDTH, DAM, FORD, CLIFFS = 12, 6, 8, 30, 3


def ground(u, v):
    du = u - RIVER
    if 0 <= du < WIDTH and v >= FORD:
        return 's'
    if 0 <= du < WIDTH:
        return 'w'
    if v <= DAM + 1 and (-CLIFFS <= du < 0 or WIDTH <= du < WIDTH + 5):
        return 'h'
    return 'g'


rows = [[ground(u, v) + '--' for u in range(W)] for v in range(H)]

# Xinye on the east bank: its gate faces the ford the army marched out of; the walls and the town burn
city = {}
for y in range(32, 40):
    for x in range(39, 47):
        city[x, y] = 'g--'
for x in range(40, 47):
    for y in (33, 38):
        city[x, y] = 'c--'
for y in range(33, 39):
    for x in (40, 46):
        city[x, y] = 'c--'
for y in range(34, 38):
    for x in range(41, 46):
        city[x, y] = 'i--'
city[40, 37] = 'ip-'
city[42, 38] = 'ip-'
for (x, y), cell in {(43, 35): 'ivF', (45, 35): 'io-', (44, 37): 'ivF', (41, 36): 'i-e', (42, 34): 'ivF',
                     (46, 36): 'c-F', (39, 36): 'dvF', (40, 40): 'd-e', (42, 41): 'dvF', (38, 33): 'dvF',
                     (36, 40): 'd-e'}.items():
    city[x, y] = cell
for (x, y), cell in city.items():
    rows[y][x] = cell

trees = set()
for v in range(H):
    for u in range(W):
        if rows[v][u] != 'g--':
            continue
        west = u < RIVER - 1 and v > 11
        east = 19 <= u <= 37 and v > 12 and v < FORD + 9
        if (west or east) and (u * 11 + v * 7) % 17 < 2 and (v < 21 or v > 37 or u > 5) and not (2 <= u <= 6 and 31 <= v <= 36) and not (u < 12 and 10 <= v <= 20):
            trees.add((u, v))
for u, v in sorted(trees):
    rows[v][u] = 'gt-'
for u, v in [(1, 24), (1, 25), (2, 25), (2, 24), (1, 27), (2, 27), (10, 36), (10, 37), (9, 37), (5, 28), (6, 28)]:
    rows[v][u] = 'u--'

layout = ROOT / 'tools/mapgen/layouts/hero/map25.txt'
layout.write_text('# B25 신야 화공 · 수공으로 끊긴 조인군: baked flood ground 48x44 (tools/mapgen/layouts/hero/map25flood.py)\n' +
                  '\n'.join(' '.join(r) for r in rows) + '\n')
subprocess.run([sys.executable, str(ROOT / 'tools/mapgen/mapgen.py'), str(layout),
                str(ROOT / 'assets/scenario/hero/maps/map25.json')], check=True)
