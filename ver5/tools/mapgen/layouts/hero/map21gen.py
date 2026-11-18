#!/usr/bin/env python3
"""map21 (여남 전투) layout: 여남성 in the north, then a long narrow valley walled by mountains that runs south past
the 여수 to the small castle on the road to 형주. Run from ver5/; writes map21.txt and maps/map21.json."""
import subprocess
from pathlib import Path

ROOT = Path('.')
W, H = 20, 52
g = [['n--'] * W for _ in range(H)]


def put(x, y, c):
    g[y][x] = c


def open_rows(y0, y1, x0, x1, c='g--'):
    for y in range(y0, y1 + 1):
        for x in range(x0, x1 + 1):
            put(x, y, c)


# 여남성 (rows 0..15), as before
open_rows(0, 0, 0, W - 1)
for y in range(1, 16):
    put(0, y, 'g--')
    put(1, y, 'g--')
    put(18, y, 'g--')
    put(19, y, 'g--')
    for x in range(2, 18):
        edge = x in (2, 17) or y in (1, 15)
        put(x, y, 'c--' if edge else 'l--')
for x in range(3, 17):
    put(x, 8, 'i--')
for y in range(2, 15):
    put(10, y, 'i--')
for x, y in ((10, 1), (2, 8), (17, 8), (10, 15)):
    put(x, y, 'm-O')
for x, y in ((5, 4), (7, 5), (13, 4), (15, 6), (5, 11), (7, 12), (13, 11), (15, 12), (4, 6), (14, 13)):
    put(x, y, 'lv-')

# the field before the south gate, closing into the valley
open_rows(16, 19, 0, 19)
open_rows(20, 21, 2, 19)        # east: a side road where 장비 and 하후연 come in
open_rows(22, 23, 3, 15)
open_rows(24, 26, 0, 14)        # west: a side valley where 장합 and 관우 come in
open_rows(27, 27, 4, 14)
# the 여수
open_rows(28, 29, 4, 15, 'w--')
put(10, 28, 'wb-')
put(10, 29, 'wb-')
put(14, 28, 's--')
put(14, 29, 's--')
# the valley south of the river bends west, then back
open_rows(30, 35, 4, 13)
open_rows(36, 37, 3, 12)
open_rows(38, 43, 2, 11)
open_rows(44, 45, 4, 12)
open_rows(46, 51, 5, 13)

road = [(10, y) for y in range(16, 28)] + [(10, y) for y in range(30, 36)] + [(9, 36), (8, 37)] + \
       [(7, y) for y in range(38, 44)] + [(8, 44), (9, 45)] + [(9, y) for y in range(46, 49)]
for x, y in road:
    put(x, y, 'd--')
put(9, 49, 'go-')

trees = ((3, 17), (6, 19), (14, 17), (15, 18), (16, 19), (13, 21), (17, 21), (4, 22), (5, 23), (12, 22), (14, 23),
         (1, 25), (6, 25), (13, 26), (5, 31), (12, 31), (12, 32), (4, 34), (5, 33), (11, 35), (3, 37), (11, 37),
         (2, 40), (3, 41), (10, 39), (10, 41), (11, 42), (5, 44), (11, 45), (6, 47), (12, 48), (7, 50), (13, 50))
for x, y in trees:
    put(x, y, 'gt-')
for x, y in ((16, 1), (19, 4), (0, 11), (19, 13), (0, 0), (5, 0)):
    put(x, y, 'gt-')

layout = ROOT / 'tools/mapgen/layouts/hero/map21.txt'
layout.write_text(
    '# map21 여남 전투 (20x52, night). Grid compass: north = row 0. 여남성 fills the top (walls x 2..17, y 1..15,\n'
    '# gates north (10,1), west (2,8), east (17,8), south (10,15), all open). South of it the field, then a long\n'
    '# valley walled by mountains: a side road east (rows 20-21), a side valley west (rows 24-26), the 여수 across\n'
    '# rows 28-29 (bridge 10, ford 14), a bend west and back, and the small castle on the road to 형주 at (9,49).\n'
    + '\n'.join(' '.join(r) for r in g) + '\n', encoding='utf-8')
subprocess.run(['python3', 'tools/mapgen/mapgen.py', str(layout), str(ROOT / 'assets/scenario/hero/maps/map21.json')],
               check=True)
