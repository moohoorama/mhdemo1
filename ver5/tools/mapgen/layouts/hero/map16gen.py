"""Layout of map16 (하비 전투). Screen north is cell (0,0): +x runs screen south-east, +y screen south-west.

하비성 stands in the middle of the field, its walls a square from L to R on both axes. 조조's dykes have flooded it:
the moat round the walls has filled into a broad ford, puddles lie on the fields outside and in the streets inside.
Three gates, shut until the stage's opening event opens them — the south-west (middle of the y = R wall), the north-west (x = L) and the south-east (x = R);
the north-east wall has none; the wall has no breaches, so the gates are the only way in. 유비 enters from the field beyond the south-west moat, 조조's army from beyond the
north-west and south-east ones. Inside, the town is laid out on a grid: paved main streets from each gate meet at
여포's hall in the middle, a paved lane runs inside the walls, and the blocks between are plain ground with houses,
gardens and a pond.

    python3 tools/mapgen/layouts/hero/map16gen.py tools/mapgen/layouts/hero/map16.txt
    python3 tools/mapgen/mapgen.py tools/mapgen/layouts/hero/map16.txt assets/scenario/hero/maps/map16.json
"""
import random
import sys

N = 28
L, R = 5, 22                # the walls
MOAT = 1                    # the moat's width outside the walls
MID = 13                    # the gates and the main streets
rnd = random.Random(16)
g = [['g--'] * N for _ in range(N)]

# the field outside: grass with patches of plain and a few trees
for y in range(N):
    for x in range(N):
        if rnd.random() < .35:
            g[y][x] = 'l--'
for x, y in ((1, 1), (2, 6), (0, 20), (3, 25), (8, 1), (20, 0), (26, 3), (25, 8), (27, 19), (21, 26), (8, 26), (1, 9),
             (26, 25), (4, 2), (17, 1)):
    g[y][x] = 'gt-'
# the roads to the three gates across the field
for k in range(0, L - MOAT):
    g[MID][k] = g[MID][N - 1 - k] = 'l--'
for k in range(R + MOAT + 1, N):
    g[k][MID] = 'l--'

# the moat, flooded into a ford, round the walls
for y in range(L - MOAT, R + MOAT + 1):
    for x in range(L - MOAT, R + MOAT + 1):
        if not (L <= x <= R and L <= y <= R):
            g[y][x] = 's--'

# the walls and the three gates, shut until the opening event opens them
for k in range(L, R + 1):
    g[L][k] = g[R][k] = g[k][L] = g[k][R] = 'c--'
g[R][MID] = g[MID][L] = g[MID][R] = 'm--'

# inside: plain ground blocks, the lane along the walls and the main streets paved, the hall at the crossing
for y in range(L + 1, R):
    for x in range(L + 1, R):
        lane = x in (L + 1, R - 1) or y in (L + 1, R - 1)
        g[y][x] = 'i--' if lane or x == MID or y == MID else 'l--'
g[MID][MID] = 'io-'
# houses on the blocks, gardens, a pond in the north block
houses = [(8, 8), (10, 8), (8, 10), (16, 8), (18, 9), (16, 10), (8, 16), (9, 18), (11, 16), (16, 16), (18, 16),
          (16, 18), (19, 19), (10, 19)]
for x, y in houses:
    g[y][x] = 'lv-'
for x, y in ((10, 10), (11, 9), (17, 17), (18, 18), (9, 15), (19, 11)):
    g[y][x] = 'g--'
g[9][15] = g[9][16] = g[10][15] = 's--'

# puddles: about one cell in four on the open ground inside and out, more toward the low south-west and north-west;
# none on the cells the armies start on
start = {(x, y) for x in range(11, 17) for y in range(25, 28)} | {(x, y) for x in (0, 1, 2) for y in range(12, 16)} \
    | {(x, y) for x in (25, 26, 27) for y in range(12, 16)}
for y in range(N):
    for x in range(N):
        if g[y][x] not in ('g--', 'l--', 'i--') or (x, y) in start:
            continue
        if rnd.random() < .14 + .12 * (y / N) + .08 * (1 - x / N):
            g[y][x] = g[y][x][:2] + 'w'

out = open(sys.argv[1], 'w') if len(sys.argv) > 1 else sys.stdout
out.write('# map16 하비 전투 (made by map16gen.py). Screen north = (0,0).\n')
for r in g:
    out.write(' '.join(r) + '\n')
